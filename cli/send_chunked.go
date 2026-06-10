package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	cliws "github.com/123100123/lanlink/cli/ws"
	"github.com/123100123/lanlink/internal/auth"
	"github.com/123100123/lanlink/protocol"
)

const chunkSize = 64 * 1024

func sendFileChunked(address string, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	transferRawID, err := auth.GenerateToken(8)
	if err != nil {
		log.Fatal(err)
	}

	transferID := "transfer_" + transferRawID

	conn := cliws.ConnectAuthenticated(address)
	defer conn.Close()

	sendFileStart(conn, transferID, filepath.Base(filePath), info.Size())
	sendFileChunks(conn, transferID, file, info.Size())
	result := sendFileEnd(conn, transferID)

	fmt.Println("Chunked file upload complete")
	fmt.Println("Saved as:", result.Path)
}

func sendFileStart(
	conn interface {
		WriteJSON(v any) error
		ReadJSON(v any) error
	},
	transferID string,
	filename string,
	size int64,
) {
	payload, err := protocol.EncodePayload(
		protocol.FileStartPayload{
			TransferID: transferID,
			Filename:   filename,
			Size:       size,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	msg := protocol.Message{
		Type:      "file.start",
		ID:        "file_start_1",
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}

	err = conn.WriteJSON(msg)
	if err != nil {
		log.Fatal("failed to send file.start:", err)
	}

	expectChunkResponse(conn, "file_start_1")
}

func sendFileChunks(
	conn interface {
		WriteJSON(v any) error
		ReadJSON(v any) error
	},
	transferID string,
	file *os.File,
	totalSize int64,
) {
	buffer := make([]byte, chunkSize)
	index := 0
	var sent int64

	for {
		n, err := file.Read(buffer)

		if err != nil && err != io.EOF {
			log.Fatal(err)
		}

		if n == 0 {
			break
		}

		content := base64.StdEncoding.EncodeToString(buffer[:n])

		payload, err := protocol.EncodePayload(
			protocol.FileChunkPayload{
				TransferID: transferID,
				Index:      index,
				Content:    content,
			},
		)
		if err != nil {
			log.Fatal(err)
		}

		msg := protocol.Message{
			Type:      "file.chunk",
			ID:        fmt.Sprintf("file_chunk_%d", index),
			Timestamp: time.Now().UnixMilli(),
			Payload:   payload,
		}

		err = conn.WriteJSON(msg)
		if err != nil {
			log.Fatal("failed to send file.chunk:", err)
		}

		expectChunkResponse(conn, msg.ID)

		sent += int64(n)
		printProgress(sent, totalSize)

		index++
	}

	fmt.Println()
}

func sendFileEnd(
	conn interface {
		WriteJSON(v any) error
		ReadJSON(v any) error
	},
	transferID string,
) protocol.FileChunkResponse {
	payload, err := protocol.EncodePayload(
		protocol.FileEndPayload{
			TransferID: transferID,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	msg := protocol.Message{
		Type:      "file.end",
		ID:        "file_end_1",
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}

	err = conn.WriteJSON(msg)
	if err != nil {
		log.Fatal("failed to send file.end:", err)
	}

	return expectChunkResponse(conn, msg.ID)
}

func expectChunkResponse(
	conn interface {
		ReadJSON(v any) error
	},
	expectedID string,
) protocol.FileChunkResponse {
	var response protocol.Message

	err := conn.ReadJSON(&response)
	if err != nil {
		log.Fatal("failed to read file transfer response:", err)
	}

	if response.Type == "error" {
		log.Fatal("file transfer failed:", string(response.Payload))
	}

	if response.Type != "file.chunk.response" {
		log.Fatal("unexpected response:", response.Type)
	}

	if response.ID != expectedID {
		log.Fatal("unexpected response id:", response.ID)
	}

	var result protocol.FileChunkResponse

	err = protocol.DecodePayload(response.Payload, &result)
	if err != nil {
		log.Fatal("failed to decode file transfer response:", err)
	}

	return result
}

func printProgress(sent int64, total int64) {
	if total <= 0 {
		fmt.Printf("\rSent %d bytes", sent)
		return
	}

	percent := float64(sent) / float64(total) * 100

	fmt.Printf(
		"\rUploading: %.2f%% (%d/%d bytes)",
		percent,
		sent,
		total,
	)
}