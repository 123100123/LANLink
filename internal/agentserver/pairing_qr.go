package agentserver

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"strings"

	"github.com/123100123/lanlink/internal/network"
	qr "github.com/mdp/qrterminal/v3"
)

type pairPayload struct {
	Type    string `json:"t"`
	Address string `json:"a"`
	Token   string `json:"tk"`
}

// selectedPairingAddress picks the host:port that clients should use to reach
// this receiver. LANLINK_HOST forces a specific host; otherwise it uses the
// exact outbound-route LAN IP (network.PrimaryIP), which ignores Docker bridges
// and VPN tunnels. Falls back to the old preference heuristic, then loopback.
func selectedPairingAddress(port string) string {
	if host := strings.TrimSpace(os.Getenv("LANLINK_HOST")); host != "" {
		return host + ":" + port
	}

	if ip := network.PrimaryIP(); ip != "" {
		return ip + ":" + port
	}

	ips, err := network.GetLocalIPs()
	if err != nil || len(ips) == 0 {
		return "127.0.0.1:" + port
	}

	for _, ip := range ips {
		if isPreferredPairingIP(ip) {
			return ip + ":" + port
		}
	}

	for _, ip := range ips {
		if isUsablePairingIP(ip) {
			return ip + ":" + port
		}
	}

	return ips[0] + ":" + port
}

func isPreferredPairingIP(value string) bool {
	ip := net.ParseIP(value).To4()
	if ip == nil {
		return false
	}

	if !isUsablePairingIP(value) {
		return false
	}

	if ip[0] == 10 {
		return true
	}

	if ip[0] == 192 && ip[1] == 168 {
		return true
	}

	if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
		return true
	}

	return false
}

func isUsablePairingIP(value string) bool {
	ip := net.ParseIP(value).To4()
	if ip == nil {
		return false
	}

	if ip.IsLoopback() {
		return false
	}

	if ip[0] == 127 {
		return false
	}

	if ip[0] == 169 && ip[1] == 254 {
		return false
	}

	// Ignore Docker bridge IPs commonly seen in this project.
	if ip[0] == 172 && (ip[1] == 17 || ip[1] == 18) {
		return false
	}

	return true
}

func printPairingQR(token string, port string) {
	address := selectedPairingAddress(port)

	payload := pairPayload{
		Type:    "l",
		Address: address,
		Token:   token,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("failed to encode QR payload:", err)
		return
	}

	log.Println("Pairing QR address:", address)
	log.Println("QR payload:", string(data))
	log.Println("Scan this QR code from the mobile app to pair:")

	qr.GenerateWithConfig(string(data), qr.Config{
		Level:     qr.L,
		Writer:    log.Writer(),
		BlackChar: "\033[40m  \033[0m",
		WhiteChar: "\033[47m  \033[0m",
		QuietZone: 2,
	})
}
