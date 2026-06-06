package main

import (
	"fmt"
	"os"
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
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run ./cli health localhost:8787")
	fmt.Println("  go run ./cli pair localhost:8787 123456")
	fmt.Println("  go run ./cli devices localhost:8787")
}