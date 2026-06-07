package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/123100123/lanlink/internal/clientconfig"
	"github.com/123100123/lanlink/protocol"
)

func listDevices(address string) {
	url := "http://" + address + "/devices"

	creds, err := clientconfig.Load()
	if err != nil {
		log.Fatal("not paired yet, run pair command first")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+creds.AuthToken)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		log.Fatal("unauthorized, pair again")
	}

	var devicesResponse protocol.DevicesResponse

	err = json.NewDecoder(resp.Body).Decode(&devicesResponse)
	if err != nil {
		log.Fatal(err)
	}

	if len(devicesResponse.Devices) == 0 {
		fmt.Println("No paired devices found.")
		return
	}

	fmt.Println("Paired devices:")
	fmt.Println()

	for index, device := range devicesResponse.Devices {
		fmt.Printf("%d. %s\n", index+1, device.DeviceID)
		fmt.Println("   Name:", device.DeviceName)
	}
}
