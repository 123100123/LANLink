package discovery

import (
	"encoding/json"
	"net"
	"time"
)

// Scan listens for receiver beacons for the given timeout and returns the
// unique receivers found, deduplicated by address. It returns an empty (non-nil)
// slice when nothing is discovered, so callers can range over it safely.
func Scan(timeout time.Duration, discoveryPort int) ([]Announcement, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: discoveryPort})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	found := []Announcement{}
	buf := make([]byte, 2048)

	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			break // deadline reached or socket closed
		}

		var a Announcement
		if err := json.Unmarshal(buf[:n], &a); err != nil {
			continue
		}
		if a.Service != ServiceName || a.Addr == "" || seen[a.Addr] {
			continue
		}
		seen[a.Addr] = true
		found = append(found, a)
	}

	return found, nil
}
