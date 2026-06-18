package main

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/123100123/lanlink/agent/dashboard"
	ws "github.com/123100123/lanlink/agent/ws"
	"github.com/123100123/lanlink/internal/config"
	"github.com/123100123/lanlink/internal/network"
	"github.com/123100123/lanlink/internal/pairing"
)

const pairingTokenLength = 6

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Filename, X-Transfer-Id")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	cfg := config.Load()

	var err error

	pairingManager, err = pairing.NewManager(pairingTokenLength)
	if err != nil {
		log.Fatal("failed to generate pairing token:", err)
	}

	token := pairingManager.Token()
	address := selectedPairingAddress(cfg.Port)

	dashboard.InitSettings("received")
	dashboard.SetAddress(address)
	dashboard.SetToken(token)

	go startTerminalProgress()

	dashboard.RegisterRoutes()

	http.HandleFunc("/health", corsMiddleware(healthHandler))
	http.HandleFunc("/pair", corsMiddleware(pairHandler))
	http.HandleFunc("/devices", corsMiddleware(devicesHandler))
	http.HandleFunc("/ws", corsMiddleware(ws.Handler))

	http.HandleFunc("/transfers/start", corsMiddleware(transferStartHandler))
	http.HandleFunc("/transfers/upload", corsMiddleware(transferUploadHandler))
	http.HandleFunc("/transfers/resumable/start", corsMiddleware(resumableStartHandler))
	http.HandleFunc("/transfers/resumable/", corsMiddleware(resumableSubresourceHandler))
	http.HandleFunc("/transfers/", corsMiddleware(transferSubresourceHandler))

	listenAddr := ":" + cfg.Port

	ips, err := network.GetLocalIPs()
	if err == nil {
		log.Println("\nAvailable addresses:")
		log.Println("127.0.0.1:" + cfg.Port)
		for _, ip := range ips {
			log.Println(ip + ":" + cfg.Port)
		}
		log.Println("")
	}

	log.Println("Pairing token:", token)
	log.Println("Use this token to pair a new device.")
	log.Println("A new token will be generated after each successful pairing.")
	log.Println("")

	printPairingQR(token, cfg.Port)

	log.Println("")
	log.Println("Dashboard: http://127.0.0.1:" + cfg.Port + "/ui")
	log.Println("")

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("LANLink agent listening on", listenAddr)

	if dashboard.ShouldOpenDashboard() {
		go func() {
			time.Sleep(300 * time.Millisecond)
			dashboard.OpenDashboard(cfg.Port)
		}()
	}

	err = http.Serve(listener, nil)
	if err != nil {
		log.Fatal(err)
	}
}
