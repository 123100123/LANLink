package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/123100123/lanlink/internal/clientconfig"
	"github.com/123100123/lanlink/protocol"
)

func pair(address string, token string) {
	url := "http://" + address + "/pair"

	requestBody := protocol.PairRequest{
		DeviceName: "termux-cli",
		Token:      token,
	}

	data, err := json.Marshal(requestBody)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(
		url,
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var pairResponse protocol.PairResponse

	err = json.NewDecoder(resp.Body).Decode(&pairResponse)
	if err != nil {
		log.Fatal(err)
	}

	if pairResponse.Status != "paired" {
		fmt.Println("Pairing failed:", pairResponse.Error)
		return
	}

	creds := clientconfig.Credentials{
		AgentAddress: address,
		DeviceID:     pairResponse.DeviceID,
		AuthToken:    pairResponse.AuthToken,
	}

	err = clientconfig.Save(creds)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Paired successfully")
	fmt.Println("Device ID:", pairResponse.DeviceID)
	fmt.Println("Credentials saved locally")
}
