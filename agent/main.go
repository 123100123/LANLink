package main

import (
	"log"
	"net/http"

	ws "github.com/123100123/lanlink/agent/ws"
	"github.com/123100123/lanlink/internal/config"

	"github.com/123100123/lanlink/internal/network"
)

func main() {
	cfg := config.Load()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/pair", pairHandler)
	http.HandleFunc("/devices", devicesHandler)
	http.HandleFunc("/ws", ws.Handler)

	address := ":" + cfg.Port

	log.Println("LANLink agent listening on", address)
	
	ips, err := network.GetLocalIPs()
	if err == nil {
	
		log.Println("")
		log.Println("Available addresses:")
	
		log.Println("127.0.0.1:" + cfg.Port)
	
		for _, ip := range ips {
			log.Println(ip + ":" + cfg.Port)
		}
	
		log.Println("")
	}
	
	err = http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal(err)
	}
}
