package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/123100123/lanlink/internal/clientconfig"
	"github.com/123100123/lanlink/internal/config"
	"github.com/123100123/lanlink/internal/discovery"
	"github.com/123100123/lanlink/protocol"
)

const scanTimeout = 2500 * time.Millisecond

// Scan discovers receivers on the LAN and tokenlessly auto-connects to one.
// With target empty it connects to the first "open" receiver found; otherwise
// it connects to the receiver whose name or address matches target.
func Scan(target, deviceName string) error {
	cfg := config.Load()

	fmt.Println("Scanning the local network…")
	hosts, err := discovery.Scan(scanTimeout, cfg.DiscoveryPort)
	if err != nil {
		return err
	}

	if len(hosts) == 0 {
		fmt.Println("No LANLink receivers found on the network.")
		return nil
	}

	fmt.Printf("\nDiscovered %d receiver(s):\n", len(hosts))
	for i, h := range hosts {
		openLabel := ""
		if h.Open {
			openLabel = "  [open]"
		}
		fmt.Printf("  %d. %-22s %s%s\n", i+1, h.Name, h.Addr, openLabel)
	}

	var chosen *discovery.Announcement
	if target != "" {
		for i := range hosts {
			if hosts[i].Name == target || hosts[i].Addr == target {
				chosen = &hosts[i]
				break
			}
		}
		if chosen == nil {
			return fmt.Errorf("no discovered receiver matches %q", target)
		}
	} else {
		for i := range hosts {
			if hosts[i].Open {
				chosen = &hosts[i]
				break
			}
		}
		if chosen == nil {
			fmt.Println("\nNo open receivers to auto-connect to. Pair manually with a token.")
			return nil
		}
	}

	fmt.Printf("\nAuto-connecting to %s (%s)…\n", chosen.Name, chosen.Addr)
	return PairAuto(chosen.Addr, deviceName)
}

// PairAuto obtains credentials from a receiver's open /pair/auto endpoint (no
// token) and saves them locally, so subsequent send/devices calls are authorized.
func PairAuto(address, deviceName string) error {
	if deviceName == "" {
		deviceName = "lanlink-desktop"
	}

	data, err := json.Marshal(protocol.PairRequest{DeviceName: deviceName})
	if err != nil {
		return err
	}

	resp, err := http.Post("http://"+address+"/pair/auto", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var pairResponse protocol.PairResponse
	if err := json.NewDecoder(resp.Body).Decode(&pairResponse); err != nil {
		return err
	}
	if pairResponse.Status != "paired" {
		return fmt.Errorf("auto-connect failed: %s", pairResponse.Error)
	}

	if err := clientconfig.Save(clientconfig.Credentials{
		AgentAddress: address,
		DeviceID:     pairResponse.DeviceID,
		AuthToken:    pairResponse.AuthToken,
	}); err != nil {
		return err
	}

	fmt.Println("Connected (tokenless)")
	fmt.Println("Device ID:", pairResponse.DeviceID)
	fmt.Println("Credentials saved locally")
	return nil
}
