// Package discovery implements LANLink's zero-config LAN discovery: receivers
// broadcast a small UDP beacon, and clients scan for it. It is pure Go (stdlib
// only) and cross-platform (Linux/Windows/macOS).
package discovery

import (
	"context"
	"encoding/json"
	"net"
	"syscall"
	"time"
)

const (
	// DefaultPort is the UDP port used for discovery beacons.
	DefaultPort = 8788
	// ServiceName tags beacons so unrelated UDP traffic is ignored.
	ServiceName = "lanlink"

	announceInterval = 2 * time.Second
)

// Announcement is the JSON payload a receiver broadcasts on the discovery port.
type Announcement struct {
	Service string `json:"service"`
	Name    string `json:"name"`
	Addr    string `json:"addr"` // receiver host:port (HTTP/WS data+control plane)
	Port    string `json:"port"`
	Version string `json:"v"`
	Open    bool   `json:"open"` // accepts tokenless /pair/auto connections
}

// Announce starts broadcasting info on the discovery UDP port every ~2s until
// ctx is cancelled. It broadcasts to the global broadcast address and to each
// up broadcast-capable interface's directed broadcast address.
func Announce(ctx context.Context, info Announcement, discoveryPort int) error {
	info.Service = ServiceName
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var sockErr error
			if err := c.Control(func(fd uintptr) {
				sockErr = controlBroadcast(fd)
			}); err != nil {
				return err
			}
			return sockErr
		},
	}

	pc, err := lc.ListenPacket(ctx, "udp4", ":0")
	if err != nil {
		return err
	}
	conn := pc.(*net.UDPConn)

	go func() {
		defer conn.Close()
		ticker := time.NewTicker(announceInterval)
		defer ticker.Stop()

		send := func() {
			for _, target := range broadcastTargets(discoveryPort) {
				conn.WriteToUDP(data, target)
			}
		}

		send()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				send()
			}
		}
	}()

	return nil
}

// broadcastTargets returns the global broadcast plus each interface's directed
// broadcast address, so the beacon reaches every attached LAN segment.
func broadcastTargets(port int) []*net.UDPAddr {
	targets := []*net.UDPAddr{{IP: net.IPv4bcast, Port: port}}

	ifaces, err := net.Interfaces()
	if err != nil {
		return targets
	}
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagBroadcast == 0 {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil || len(ipnet.Mask) != net.IPv4len {
				continue
			}
			bcast := make(net.IP, net.IPv4len)
			for i := 0; i < net.IPv4len; i++ {
				bcast[i] = ip4[i] | ^ipnet.Mask[i]
			}
			targets = append(targets, &net.UDPAddr{IP: bcast, Port: port})
		}
	}
	return targets
}
