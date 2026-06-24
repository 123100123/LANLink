package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/123100123/lanlink/internal/clientconfig"
	"github.com/123100123/lanlink/internal/config"
	"github.com/123100123/lanlink/internal/transfer"
	"github.com/123100123/lanlink/protocol"
)

// Health checks whether an agent is reachable at address (host:port).
func Health(address string) error {
	resp, err := http.Get("http://" + address + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var health protocol.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return err
	}

	fmt.Println("Agent reachable")
	fmt.Println("Status:", health.Status)
	fmt.Println("Service:", health.Service)
	return nil
}

// Pair exchanges a pairing token for credentials and saves them locally.
func Pair(address, token, deviceName string) error {
	if deviceName == "" {
		deviceName = "lanlink-cli"
	}

	data, err := json.Marshal(protocol.PairRequest{DeviceName: deviceName, Token: token})
	if err != nil {
		return err
	}

	resp, err := http.Post("http://"+address+"/pair", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var pairResponse protocol.PairResponse
	if err := json.NewDecoder(resp.Body).Decode(&pairResponse); err != nil {
		return err
	}
	if pairResponse.Status != "paired" {
		return fmt.Errorf("pairing failed: %s", pairResponse.Error)
	}

	if err := clientconfig.Save(clientconfig.Credentials{
		AgentAddress: address,
		DeviceID:     pairResponse.DeviceID,
		AuthToken:    pairResponse.AuthToken,
	}); err != nil {
		return err
	}

	fmt.Println("Paired successfully")
	fmt.Println("Device ID:", pairResponse.DeviceID)
	fmt.Println("Credentials saved locally")
	return nil
}

// Devices lists the agent's paired devices (requires prior pairing).
func Devices(address string) error {
	creds, err := clientconfig.Load()
	if err != nil {
		return fmt.Errorf("not paired yet, run pair command first")
	}

	req, err := http.NewRequest(http.MethodGet, "http://"+address+"/devices", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+creds.AuthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized, pair again")
	}

	var devicesResponse protocol.DevicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&devicesResponse); err != nil {
		return err
	}

	if len(devicesResponse.Devices) == 0 {
		fmt.Println("No paired devices found.")
		return nil
	}

	fmt.Println("Paired devices:")
	fmt.Println()
	for index, device := range devicesResponse.Devices {
		fmt.Printf("%d. %s\n", index+1, device.DeviceID)
		fmt.Println("   Name:", device.DeviceName)
	}
	return nil
}

// SendFile uploads filePath to the agent at address using saved credentials,
// rendering terminal progress.
func SendFile(address, filePath string) error {
	cfg := config.Load()

	creds, err := clientconfig.Load()
	if err != nil {
		return fmt.Errorf("not paired yet, run pair command first")
	}

	result, err := transfer.SendFile(address, creds.AuthToken, filePath, transfer.SendOptions{
		ChunkSize:         cfg.TransferChunkSize,
		MaxInFlightChunks: cfg.TransferMaxInFlightChunks,
		OnProgress:        transfer.TerminalProgress(),
	})
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("HTTP file upload complete")
	fmt.Println("Saved as:", result.Path)
	return nil
}
