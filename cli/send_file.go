package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/123100123/lanlink/internal/auth"
	"github.com/123100123/lanlink/internal/clientconfig"
	"github.com/123100123/lanlink/internal/config"
	"github.com/123100123/lanlink/protocol"
)

type chunkJob struct {
	Index  int
	Offset int64
	Data   []byte
}

type chunkResult struct {
	Bytes int
}

func sendFile(address string, filePath string) {
	cfg := config.Load()

	creds, err := clientconfig.Load()
	if err != nil {
		log.Fatal("not paired yet, run pair command first")
	}

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
	baseURL := "http://" + address
	client := newTransferHTTPClient(cfg.TransferMaxInFlightChunks)

	startTime := time.Now()

	startTransferHTTP(
		client,
		baseURL,
		creds.AuthToken,
		transferID,
		filepath.Base(filePath),
		info.Size(),
	)

	sendFileChunksHTTP(
		client,
		baseURL,
		creds.AuthToken,
		transferID,
		file,
		info.Size(),
		startTime,
		cfg.TransferChunkSize,
		cfg.TransferMaxInFlightChunks,
	)

	result := finishTransferHTTP(
		client,
		baseURL,
		creds.AuthToken,
		transferID,
	)

	fmt.Println("HTTP file upload complete")
	fmt.Println("Saved as:", result.Path)
}

func newTransferHTTPClient(maxWorkers int) *http.Client {
	if maxWorkers <= 0 {
		maxWorkers = 1
	}

	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        maxWorkers * 2,
			MaxIdleConnsPerHost: maxWorkers * 2,
			MaxConnsPerHost:     maxWorkers,
		},
	}
}

func startTransferHTTP(
	client *http.Client,
	baseURL string,
	authToken string,
	transferID string,
	filename string,
	size int64,
) {
	body, err := json.Marshal(protocol.TransferStartRequest{
		TransferID: transferID,
		Filename:   filename,
		Size:       size,
	})
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		baseURL+"/transfers/start",
		bytes.NewReader(body),
	)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("failed to start transfer:", err)
	}
	defer resp.Body.Close()

	var result protocol.TransferStartResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatal("failed to decode transfer start response:", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatal("failed to start transfer:", result.Error)
	}
}

func sendFileChunksHTTP(
	client *http.Client,
	baseURL string,
	authToken string,
	transferID string,
	file *os.File,
	totalSize int64,
	startTime time.Time,
	chunkSize int,
	maxWorkers int,
) {
	if chunkSize <= 0 {
		log.Fatal("TRANSFER_CHUNK_SIZE must be greater than zero")
	}

	if maxWorkers <= 0 {
		log.Fatal("TRANSFER_MAX_IN_FLIGHT_CHUNKS must be greater than zero")
	}

	totalChunks := int((totalSize + int64(chunkSize) - 1) / int64(chunkSize))
	if totalChunks == 0 {
		fmt.Println()
		return
	}

	jobs := make(chan chunkJob, maxWorkers)
	results := make(chan chunkResult, maxWorkers)
	errs := make(chan error, 1)

	var workerWG sync.WaitGroup

	for workerID := 0; workerID < maxWorkers; workerID++ {
		workerWG.Add(1)

		go func() {
			defer workerWG.Done()

			for job := range jobs {
				err := uploadChunkHTTP(
					client,
					baseURL,
					authToken,
					transferID,
					job,
				)
				if err != nil {
					reportTransferError(errs, err)
					return
				}

				results <- chunkResult{
					Bytes: len(job.Data),
				}
			}
		}()
	}

	go func() {
		defer close(jobs)

		index := 0
		offset := int64(0)

		for {
			buffer := make([]byte, chunkSize)

			n, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				reportTransferError(errs, err)
				return
			}

			if n == 0 {
				return
			}

			jobs <- chunkJob{
				Index:  index,
				Offset: offset,
				Data:   buffer[:n],
			}

			index++
			offset += int64(n)
		}
	}()

	go func() {
		workerWG.Wait()
		close(results)
	}()

	acknowledgedChunks := 0
	acknowledgedBytes := int64(0)

	for acknowledgedChunks < totalChunks {
		select {
		case err := <-errs:
			log.Fatal("file upload failed:", err)

		case result, ok := <-results:
			if !ok {
				log.Fatal("file upload stopped before all chunks were acknowledged")
			}

			acknowledgedChunks++
			acknowledgedBytes += int64(result.Bytes)

			printProgress(
				acknowledgedBytes,
				totalSize,
				startTime,
			)
		}
	}

	fmt.Println()
}

func uploadChunkHTTP(
	client *http.Client,
	baseURL string,
	authToken string,
	transferID string,
	job chunkJob,
) error {
	chunkURL := fmt.Sprintf(
		"%s/transfers/%s/chunks/%d?offset=%d",
		baseURL,
		url.PathEscape(transferID),
		job.Index,
		job.Offset,
	)

	req, err := http.NewRequest(
		http.MethodPut,
		chunkURL,
		bytes.NewReader(job.Data),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result protocol.TransferChunkResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode chunk response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"chunk %d failed: %s",
			job.Index,
			result.Error,
		)
	}

	if result.Status != "chunk.received" &&
		result.Status != "chunk.duplicate" {
		return fmt.Errorf(
			"chunk %d returned unexpected status: %s",
			job.Index,
			result.Status,
		)
	}

	return nil
}

func finishTransferHTTP(
	client *http.Client,
	baseURL string,
	authToken string,
	transferID string,
) protocol.TransferFinishResponse {
	finishURL := fmt.Sprintf(
		"%s/transfers/%s/finish",
		baseURL,
		url.PathEscape(transferID),
	)

	req, err := http.NewRequest(
		http.MethodPost,
		finishURL,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("failed to finish transfer:", err)
	}
	defer resp.Body.Close()

	var result protocol.TransferFinishResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatal("failed to decode transfer finish response:", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatal("failed to finish transfer:", result.Error)
	}

	return result
}

func reportTransferError(errs chan<- error, err error) {
	select {
	case errs <- err:
	default:
	}
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
