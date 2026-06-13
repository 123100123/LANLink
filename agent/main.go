package main

import (
	"log"
	"net/http"

	ws "github.com/123100123/lanlink/agent/ws"
	"github.com/123100123/lanlink/internal/config"
	"github.com/123100123/lanlink/internal/network"
)

// corsMiddleware attaches the necessary headers and handles the OPTIONS preflight
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	cfg := config.Load()

	// Wrap all your handlers with the CORS middleware
	http.HandleFunc("/health", corsMiddleware(healthHandler))
	http.HandleFunc("/pair", corsMiddleware(pairHandler))
	http.HandleFunc("/devices", corsMiddleware(devicesHandler))
	http.HandleFunc("/ws", corsMiddleware(ws.Handler))

	address := ":" + cfg.Port

	log.Println("LANLink agent listening on", address)

	ips, err := network.GetLocalIPs()
	if err == nil {
		log.Println("\nAvailable addresses:")
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
