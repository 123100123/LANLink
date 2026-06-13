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
	"github.com/123100123/lanlink/internal/config"
	"github.com/123100123/lanlink/internal/wsutil"
	"github.com/123100123/lanlink/protocol"
)

type chunkJob struct {
	Index  int
	Offset int64
	Data   []byte
	MsgID  string
}

func sendFile(address string, filePath string) {
	cfg := config.Load()

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

	rawConn := cliws.ConnectAuthenticated(address)
	defer rawConn.Close()

	conn := wsutil.NewSafeConn(rawConn)

	startTime := time.Now()

	sendFileStart(
		conn,
		transferID,
		filepath.Base(filePath),
		info.Size(),
	)

	sendFileChunksPipelined(
		conn,
		transferID,
		file,
		info.Size(),
		startTime,
		cfg.TransferChunkSize,
		cfg.TransferMaxInFlightChunks,
	)

	result := sendFileEnd(conn, transferID)

	fmt.Println("Chunked file upload complete")
	fmt.Println("Saved as:", result.Path)
}

func sendFileStart(
	conn *wsutil.SafeConn,
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

	if err := conn.WriteJSON(msg); err != nil {
		log.Fatal("failed to send file.start:", err)
	}

	expectChunkResponse(conn, msg.ID)
}

func sendFileChunksPipelined(
	conn *wsutil.SafeConn,
	transferID string,
	file *os.File,
	totalSize int64,
	startTime time.Time,
	chunkSize int,
	maxInFlightChunks int,
) {
	inFlight := 0
	nextIndex := 0
	nextOffset := int64(0)
	eof := false
	ackedBytes := int64(0)

	for !eof || inFlight > 0 {
		for !eof && inFlight < maxInFlightChunks {
			job, ok := readNextChunk(
				file,
				nextIndex,
				nextOffset,
				chunkSize,
			)
			if !ok {
				eof = true
				break
			}

			if err := sendFileChunk(conn, transferID, job); err != nil {
				log.Fatal("failed to send file.chunk:", err)
			}

			inFlight++
			nextIndex++
			nextOffset += int64(len(job.Data))
		}

		response := expectAnyChunkResponse(conn)

		if response.Status != "chunk.received" &&
			response.Status != "chunk.duplicate" {
			log.Fatal("unexpected chunk status:", response.Status)
		}

		inFlight--

		if response.Received > ackedBytes {
			ackedBytes = response.Received
		}

		printProgress(ackedBytes, totalSize, startTime)
	}

	fmt.Println()
}

func readNextChunk(
	file *os.File,
	index int,
	offset int64,
	chunkSize int,
) (chunkJob, bool) {
	buffer := make([]byte, chunkSize)

	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}

	if n == 0 {
		return chunkJob{}, false
	}

	return chunkJob{
		Index:  index,
		Offset: offset,
		Data:   buffer[:n],
		MsgID:  fmt.Sprintf("file_chunk_%d", index),
	}, true
}

func sendFileChunk(
	conn *wsutil.SafeConn,
	transferID string,
	job chunkJob,
) error {
	content := base64.StdEncoding.EncodeToString(job.Data)

	payload, err := protocol.EncodePayload(
		protocol.FileChunkPayload{
			TransferID: transferID,
			Index:      job.Index,
			Offset:     job.Offset,
			Length:     len(job.Data),
			Content:    content,
		},
	)
	if err != nil {
		return err
	}

	msg := protocol.Message{
		Type:      "file.chunk",
		ID:        job.MsgID,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}

	return conn.WriteJSON(msg)
}

func sendFileEnd(
	conn *wsutil.SafeConn,
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

	if err := conn.WriteJSON(msg); err != nil {
		log.Fatal("failed to send file.end:", err)
	}

	return expectChunkResponse(conn, msg.ID)
}

func expectChunkResponse(
	conn *wsutil.SafeConn,
	expectedID string,
) protocol.FileChunkResponse {
	for {
		var response protocol.Message

		if err := conn.ReadJSON(&response); err != nil {
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

		if err := protocol.DecodePayload(response.Payload, &result); err != nil {
			log.Fatal("failed to decode file transfer response:", err)
		}

		return result
	}
}

func expectAnyChunkResponse(
	conn *wsutil.SafeConn,
) protocol.FileChunkResponse {
	var response protocol.Message

	if err := conn.ReadJSON(&response); err != nil {
		log.Fatal("failed to read file transfer response:", err)
	}

	if response.Type == "error" {
		log.Fatal("file transfer failed:", string(response.Payload))
	}

	if response.Type != "file.chunk.response" {
		log.Fatal("unexpected response:", response.Type)
	}

	var result protocol.FileChunkResponse

	if err := protocol.DecodePayload(response.Payload, &result); err != nil {
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
