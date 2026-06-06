package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/123100123/lanlink/protocol"
)

func listDevices(address string) {
	url := "http://" + address + "/devices"

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

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