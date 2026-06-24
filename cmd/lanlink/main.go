// Command lanlink is the unified, pure-Go LANLink terminal binary. It has no
// dependency on agent-web or the mobile app: it can run a receiver, send files,
// and perform pairing/health/latency operations entirely from the terminal.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/123100123/lanlink/internal/agentserver"
	"github.com/123100123/lanlink/internal/client"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "receive", "agent":
		// Headless receiver: no dashboard, no agent-web.
		must(agentserver.Run(agentserver.Options{
			DisableDiscovery: hasFlag("--no-discovery"),
		}))

	case "scan":
		// Discover receivers and auto-connect (optional name/addr target).
		must(client.Scan(positional(2), defaultDeviceName()))

	case "send":
		need(4, "send <host:port> <file>")
		must(client.SendFile(os.Args[2], os.Args[3]))

	case "pair":
		need(4, "pair <host:port> <token>")
		must(client.Pair(os.Args[2], os.Args[3], defaultDeviceName()))

	case "health":
		need(3, "health <host:port>")
		must(client.Health(os.Args[2]))

	case "devices":
		need(3, "devices <host:port>")
		must(client.Devices(os.Args[2]))

	case "ping":
		need(3, "ping <host:port>")
		must(client.Ping(os.Args[2]))

	case "message":
		need(4, "message <host:port> <text>")
		must(client.Message(os.Args[2], os.Args[3:]))

	case "-h", "--help", "help":
		usage()

	default:
		fmt.Println("Unknown command:", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func need(n int, form string) {
	if len(os.Args) < n {
		fmt.Println("Usage: lanlink " + form)
		os.Exit(1)
	}
}

// positional returns os.Args[i] if present and not a flag, else "".
func positional(i int) string {
	if len(os.Args) > i && len(os.Args[i]) > 0 && os.Args[i][0] != '-' {
		return os.Args[i]
	}
	return ""
}

func hasFlag(name string) bool {
	for _, a := range os.Args[2:] {
		if a == name {
			return true
		}
	}
	return false
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func defaultDeviceName() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return "lanlink-" + h
	}
	return "lanlink-desktop"
}

func usage() {
	fmt.Println("LANLink — local network file transfer")
	fmt.Println()
	fmt.Println("Usage: lanlink <command> [args]")
	fmt.Println()
	fmt.Println("Receiver:")
	fmt.Println("  receive [--no-discovery]     run a receiver (terminal QR + progress)")
	fmt.Println()
	fmt.Println("Client:")
	fmt.Println("  scan [name|host:port]        find receivers and auto-connect (no token)")
	fmt.Println("  pair <host:port> <token>     pair with a receiver")
	fmt.Println("  send <host:port> <file>      send a file to a receiver")
	fmt.Println("  health <host:port>           check a receiver is reachable")
	fmt.Println("  devices <host:port>          list paired devices")
	fmt.Println("  ping <host:port>             measure latency")
	fmt.Println("  message <host:port> <text>   send a direct message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  lanlink receive")
	fmt.Println("  lanlink scan")
	fmt.Println("  lanlink send 192.168.1.5:8787 movie.mkv")
}
