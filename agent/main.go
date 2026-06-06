package main

import (
	"log"
	"net/http"

	"github.com/123100123/lanlink/internal/config"
)

func main() {
	cfg := config.Load()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/pair", pairHandler)
	http.HandleFunc("/devices", devicesHandler)

	address := ":" + cfg.Port

	log.Println("LANLink agent listening on", address)

	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal(err)
	}
}