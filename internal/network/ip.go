package network

import (
	"net"
	"strings"
)

// GetLocalIPs returns the IPv4 addresses of real (physical) LAN interfaces,
// excluding loopback, virtual bridges (docker/br-/veth/…), VPN/tunnel
// interfaces, and link-local addresses.
func GetLocalIPs() ([]string, error) {
	return realInterfaceIPs()
}

func realInterfaceIPs() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		// Real LAN interfaces support broadcast; this drops VPN/PPP tunnels.
		if iface.Flags&net.FlagBroadcast == 0 {
			continue
		}
		if isVirtualName(iface.Name) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			ips = append(ips, ip.String())
		}
	}
	return ips, nil
}

// isVirtualName matches interface names that are not the user's real LAN/Wi-Fi
// (Docker bridges, veth pairs, VPN tunnels, etc.).
func isVirtualName(name string) bool {
	n := strings.ToLower(name)
	for _, prefix := range []string{
		"docker", "br-", "veth", "virbr", "vmnet", "vboxnet",
		"tun", "tap", "wg", "tailscale", "zt", "utun", "ham",
		"singbox", "ppp", "ipsec", "wt", "nordlynx",
	} {
		if strings.HasPrefix(n, prefix) {
			return true
		}
	}
	return false
}

// OutboundIP returns the source IPv4 the OS picks for outbound traffic (the
// default-route interface). It opens no real connection — UDP "dial" only does a
// route lookup — so it works offline as long as a default route exists. Returns
// "" if it can't be determined.
func OutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	if ua, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		if ip := ua.IP.To4(); ip != nil {
			return ip.String()
		}
	}
	return ""
}

// PrimaryIP returns the single best LAN IPv4 to advertise to peers: the
// outbound-route source IP when it belongs to a real LAN interface (the exact IP
// the machine uses to reach the network), otherwise the first real LAN
// interface IP. This correctly ignores Docker bridges and VPN tunnels even when
// they hold the default route. Returns "" if nothing suitable is found.
func PrimaryIP() string {
	candidates, _ := realInterfaceIPs()

	if out := OutboundIP(); out != "" {
		for _, ip := range candidates {
			if ip == out {
				return out
			}
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return OutboundIP()
}
