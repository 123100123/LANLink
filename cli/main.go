package main

import (
	"fmt"
	"os"

	cliws "github.com/123100123/lanlink/cli/ws"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "health":
		if len(os.Args) < 3 {
			printUsage()
			return
		}

		address := os.Args[2]
		checkHealth(address)

	case "pair":
		if len(os.Args) < 4 {
			printUsage()
			return
		}

		address := os.Args[2]
		token := os.Args[3]

		pair(address, token)

	case "devices":
		if len(os.Args) < 3 {
			printUsage()
			return
		}

		address := os.Args[2]
		listDevices(address)

	case "ws":
		if len(os.Args) < 3 {
			printUsage()
			return
		}

		address := os.Args[2]
		cliws.Connect(address)

	case "ping":
		if len(os.Args) < 3 {
			printUsage()
			return
		}

		address := os.Args[2]
		Run(address)

	case "message":
		if len(os.Args) < 4 {
			printUsage()
			return
		}

		address := os.Args[2]
		messageParts := os.Args[3:]

		cliws.SendDirectMessage(address, messageParts)

	case "send-file":
		if len(os.Args) < 4 {
			printUsage()
			return
		}

		address := os.Args[2]
		filePath := os.Args[3]

		sendFile(address, filePath)
		
	default:
		fmt.Println("Unknown command:", command)
		printUsage()
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
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run ./cli health localhost:8787")
	fmt.Println("  go run ./cli pair localhost:8787 123456")
	fmt.Println("  go run ./cli devices localhost:8787")
	fmt.Println("  go run ./cli ws localhost:8787")
	fmt.Println("  go run ./cli ping localhost:8787")
	fmt.Println(`  go run ./cli message localhost:8787 "hello from termux"`)
	fmt.Println("  go run ./cli send-file localhost:8787 test.txt")
}
