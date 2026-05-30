package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		printUsage()
		return
	}

	command := os.Args[1]
	address := os.Args[2]

	switch command {
	case "health":
		checkHealth(address)
	default:
		fmt.Println("Unknown command:", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run ./cli health <host:port>")
}