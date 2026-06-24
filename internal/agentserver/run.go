package agentserver

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	ws "github.com/123100123/lanlink/agent/ws"
	"github.com/123100123/lanlink/internal/config"
	"github.com/123100123/lanlink/internal/discovery"
	"github.com/123100123/lanlink/internal/network"
	"github.com/123100123/lanlink/internal/pairing"
)

const pairingTokenLength = 6

// Version is reported in discovery beacons.
const Version = "0.5.0-dev"

// Options lets a caller layer optional behaviour on top of the pure-Go receiver
// core without the core depending on any UI package (e.g. the dashboard binary
// injects its /ui routes and browser-open behaviour here). The terminal binary
// passes a zero Options value and runs fully headless.
type Options struct {
	// RegisterRoutes, if set, is called before the core HTTP routes are
	// registered, letting a UI layer attach additional handlers on the mux.
	RegisterRoutes func()
	// OnListening, if set, is called once the server is listening, with the port.
	OnListening func(port string)
	// DisableDiscovery turns off the LAN discovery beacon (and thus open
	// auto-connect advertising). The receiver still works; it just isn't
	// announced for `lanlink scan`.
	DisableDiscovery bool
}

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

// Run starts the LANLink receiver: HTTP data plane + WebSocket control plane,
// pairing, terminal QR and terminal progress. It blocks until the server stops.
func Run(opts Options) error {
	cfg := config.Load()

	var err error
	pairingManager, err = pairing.NewManager(pairingTokenLength)
	if err != nil {
		return err
	}

	token := pairingManager.Token()
	address := selectedPairingAddress(cfg.Port)

	InitSettings("received")
	SetAddress(address)
	SetToken(token)
	SetTransferCancelFunc(func(id string) error {
		httpTransferManager.Cancel(id)
		return nil
	})

	go startTerminalProgress()

	if !opts.DisableDiscovery {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "lanlink-receiver"
		}
		if err := discovery.Announce(context.Background(), discovery.Announcement{
			Name:    hostname,
			Addr:    address,
			Port:    cfg.Port,
			Version: Version,
			Open:    true,
		}, cfg.DiscoveryPort); err != nil {
			log.Println("discovery: failed to start beacon:", err)
		}
	}

	if opts.RegisterRoutes != nil {
		opts.RegisterRoutes()
	}

	http.HandleFunc("/health", corsMiddleware(healthHandler))
	http.HandleFunc("/pair", corsMiddleware(pairHandler))
	http.HandleFunc("/pair/auto", corsMiddleware(pairAutoHandler))
	http.HandleFunc("/devices", corsMiddleware(devicesHandler))
	http.HandleFunc("/ws", corsMiddleware(ws.Handler))

	http.HandleFunc("/transfers/start", corsMiddleware(transferStartHandler))
	http.HandleFunc("/transfers/upload", corsMiddleware(transferUploadHandler))
	http.HandleFunc("/transfers/resumable/start", corsMiddleware(resumableStartHandler))
	http.HandleFunc("/transfers/resumable/", corsMiddleware(resumableSubresourceHandler))
	http.HandleFunc("/transfers/", corsMiddleware(transferSubresourceHandler))

	listenAddr := ":" + cfg.Port

	if ips, err := network.GetLocalIPs(); err == nil {
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

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	log.Println("LANLink agent listening on", listenAddr)

	if opts.OnListening != nil {
		opts.OnListening(cfg.Port)
	}

	return http.Serve(listener, nil)
}
