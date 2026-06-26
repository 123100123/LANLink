// Command cli is the original LANLink terminal client, preserved for
// compatibility. It is now a thin wrapper over the shared internal/client and
// internal/agentserver packages; the unified cmd/lanlink binary is preferred.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/123100123/lanlink/internal/client"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "health":
		requireArgs(3)
		fail(client.Health(os.Args[2]))

	case "pair":
		requireArgs(4)
		fail(client.Pair(os.Args[2], os.Args[3], "lanlink-cli"))

	case "devices":
		requireArgs(3)
		fail(client.Devices(os.Args[2]))

	case "ws":
		requireArgs(3)
		fail(client.WSHello(os.Args[2]))

	case "ping":
		requireArgs(3)
		fail(client.Ping(os.Args[2]))

	case "message":
		requireArgs(4)
		fail(client.Message(os.Args[2], os.Args[3:]))

	case "send-file":
		requireArgs(4)
		fail(client.SendFile(os.Args[2], os.Args[3]))

	default:
		fmt.Println("Unknown command:", os.Args[1])
		printUsage()
	}
}

func requireArgs(n int) {
	if len(os.Args) < n {
		printUsage()
		os.Exit(1)
	}
}

func fail(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run ./cli health <host:port>")
	fmt.Println("  go run ./cli pair <host:port> <token>")
	fmt.Println("  go run ./cli devices <host:port>")
	fmt.Println("  go run ./cli ws <host:port>")
	fmt.Println("  go run ./cli ping <host:port>")
	fmt.Println("  go run ./cli message <host:port> <text>")
	fmt.Println("  go run ./cli send-file <host:port> <file>")
}
