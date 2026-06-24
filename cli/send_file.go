package main

import (
	"fmt"
	"log"

	"github.com/123100123/lanlink/internal/clientconfig"
	"github.com/123100123/lanlink/internal/config"
	"github.com/123100123/lanlink/internal/transfer"
)

func sendFile(address string, filePath string) {
	cfg := config.Load()

	creds, err := clientconfig.Load()
	if err != nil {
		log.Fatal("not paired yet, run pair command first")
	}

	result, err := transfer.SendFile(address, creds.AuthToken, filePath, transfer.SendOptions{
		ChunkSize:         cfg.TransferChunkSize,
		MaxInFlightChunks: cfg.TransferMaxInFlightChunks,
		OnProgress:        transfer.TerminalProgress(),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println()
	fmt.Println("HTTP file upload complete")
	fmt.Println("Saved as:", result.Path)
}
