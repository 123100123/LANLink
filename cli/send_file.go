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

func sendFile(address string, filePath string) {
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

	startTime := time.Now()

	sendFileStart(conn, transferID, filepath.Base(filePath), info.Size())
	sendFileChunks(conn, transferID, file, info.Size(), startTime)
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
	startTime time.Time,
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

		printProgress(
			sent,
			totalSize,
			startTime,
		)

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

func printProgress(
	sent int64,
	total int64,
	startTime time.Time,
) {
	elapsed := time.Since(startTime).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}

	speedMBps := (float64(sent) / 1024 / 1024) / elapsed

	eta := "unknown"

	if total > 0 && speedMBps > 0 {
		remainingBytes := total - sent
		etaSeconds := float64(remainingBytes) / (speedMBps * 1024 * 1024)
		eta = formatDuration(etaSeconds)
	}

	if total <= 0 {
		fmt.Printf(
			"\r%s sent | %.2f MB/s | ETA %s",
			formatBytes(sent),
			speedMBps,
			eta,
		)
		return
	}

	percent := float64(sent) / float64(total) * 100

	fmt.Printf(
		"\r%.2f%% | %s / %s | %.2f MB/s | ETA %s",
		percent,
		formatBytes(sent),
		formatBytes(total),
		speedMBps,
		eta,
	)
}

func formatBytes(bytes int64) string {
	const unit = 1024

	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div := int64(unit)
	exp := 0

	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{
		"KB",
		"MB",
		"GB",
		"TB",
	}

	return fmt.Sprintf(
		"%.2f %s",
		float64(bytes)/float64(div),
		units[exp],
	)
}

func formatDuration(seconds float64) string {
	if seconds < 1 {
		return "<1s"
	}

	total := int(seconds)

	minutes := total / 60
	remainingSeconds := total % 60

	if minutes == 0 {
		return fmt.Sprintf("%ds", remainingSeconds)
	}

	return fmt.Sprintf("%dm%ds", minutes, remainingSeconds)
}