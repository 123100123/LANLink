package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/123100123/lanlink/protocol"
)

func checkHealth(address string) {
	url := "http://" + address + "/health"

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var health protocol.HealthResponse

	err = json.NewDecoder(resp.Body).Decode(&health)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Agent reachable")
	fmt.Println("Status:", health.Status)
	fmt.Println("Service:", health.Service)
}