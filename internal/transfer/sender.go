package transfer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/123100123/lanlink/internal/auth"
	"github.com/123100123/lanlink/protocol"
)

// ProgressFunc is invoked as chunks are acknowledged, with the running total of
// acknowledged bytes and the total transfer size.
type ProgressFunc func(sentBytes, totalBytes int64)

// SendOptions configures a file upload.
type SendOptions struct {
	ChunkSize         int
	MaxInFlightChunks int
	OnProgress        ProgressFunc
}

// SendResult describes a completed upload.
type SendResult struct {
	TransferID string
	Path       string
}

type chunkJob struct {
	Index  int
	Offset int64
	Data   []byte
}

// SendFile uploads filePath to a LANLink receiver at address (host:port) using
// the parallel chunked HTTP transfer protocol. It returns an error rather than
// terminating the process, so any binary can reuse it.
func SendFile(address, authToken, filePath string, opts SendOptions) (*SendResult, error) {
	if opts.ChunkSize <= 0 {
		return nil, fmt.Errorf("chunk size must be greater than zero")
	}
	if opts.MaxInFlightChunks <= 0 {
		opts.MaxInFlightChunks = 1
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	transferRawID, err := auth.GenerateToken(8)
	if err != nil {
		return nil, err
	}

	transferID := "transfer_" + transferRawID
	baseURL := "http://" + address
	client := newSenderHTTPClient(opts.MaxInFlightChunks)

	if err := startTransfer(client, baseURL, authToken, transferID, filepath.Base(filePath), info.Size()); err != nil {
		return nil, err
	}

	if err := sendChunks(client, baseURL, authToken, transferID, file, info.Size(), opts); err != nil {
		return nil, err
	}

	result, err := finishTransfer(client, baseURL, authToken, transferID)
	if err != nil {
		return nil, err
	}

	return &SendResult{TransferID: transferID, Path: result.Path}, nil
}

func newSenderHTTPClient(maxWorkers int) *http.Client {
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

func startTransfer(client *http.Client, baseURL, authToken, transferID, filename string, size int64) error {
	body, err := json.Marshal(protocol.TransferStartRequest{
		TransferID: transferID,
		Filename:   filename,
		Size:       size,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/transfers/start", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start transfer: %w", err)
	}
	defer resp.Body.Close()

	var result protocol.TransferStartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode transfer start response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to start transfer: %s", result.Error)
	}
	return nil
}

func sendChunks(client *http.Client, baseURL, authToken, transferID string, file *os.File, totalSize int64, opts SendOptions) error {
	chunkSize := opts.ChunkSize
	maxWorkers := opts.MaxInFlightChunks

	totalChunks := int((totalSize + int64(chunkSize) - 1) / int64(chunkSize))
	if totalChunks == 0 {
		if opts.OnProgress != nil {
			opts.OnProgress(0, 0)
		}
		return nil
	}

	jobs := make(chan chunkJob, maxWorkers)
	results := make(chan int, maxWorkers)
	errs := make(chan error, 1)

	var workerWG sync.WaitGroup
	for workerID := 0; workerID < maxWorkers; workerID++ {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			for job := range jobs {
				if err := uploadChunk(client, baseURL, authToken, transferID, job); err != nil {
					reportErr(errs, err)
					return
				}
				results <- len(job.Data)
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
				reportErr(errs, err)
				return
			}
			if n == 0 {
				return
			}
			jobs <- chunkJob{Index: index, Offset: offset, Data: buffer[:n]}
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
			return fmt.Errorf("file upload failed: %w", err)
		case n, ok := <-results:
			if !ok {
				return fmt.Errorf("file upload stopped before all chunks were acknowledged")
			}
			acknowledgedChunks++
			acknowledgedBytes += int64(n)
			if opts.OnProgress != nil {
				opts.OnProgress(acknowledgedBytes, totalSize)
			}
		}
	}
	return nil
}

func uploadChunk(client *http.Client, baseURL, authToken, transferID string, job chunkJob) error {
	chunkURL := fmt.Sprintf("%s/transfers/%s/chunks/%d?offset=%d", baseURL, url.PathEscape(transferID), job.Index, job.Offset)

	req, err := http.NewRequest(http.MethodPut, chunkURL, bytes.NewReader(job.Data))
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
		return fmt.Errorf("chunk %d failed: %s", job.Index, result.Error)
	}
	if result.Status != "chunk.received" && result.Status != "chunk.duplicate" {
		return fmt.Errorf("chunk %d returned unexpected status: %s", job.Index, result.Status)
	}
	return nil
}

func finishTransfer(client *http.Client, baseURL, authToken, transferID string) (*protocol.TransferFinishResponse, error) {
	finishURL := fmt.Sprintf("%s/transfers/%s/finish", baseURL, url.PathEscape(transferID))

	req, err := http.NewRequest(http.MethodPost, finishURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to finish transfer: %w", err)
	}
	defer resp.Body.Close()

	var result protocol.TransferFinishResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode transfer finish response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to finish transfer: %s", result.Error)
	}
	return &result, nil
}

func reportErr(errs chan<- error, err error) {
	select {
	case errs <- err:
	default:
	}
}

// TerminalProgress returns a ProgressFunc that renders an updating progress line
// to stdout (percent, transferred, speed, ETA). Shared by terminal clients.
func TerminalProgress() ProgressFunc {
	start := time.Now()
	return func(sent, total int64) {
		elapsed := time.Since(start).Seconds()
		if elapsed <= 0 {
			elapsed = 0.001
		}
		speedMBps := (float64(sent) / 1024 / 1024) / elapsed

		eta := "unknown"
		if total > 0 && speedMBps > 0 {
			remaining := total - sent
			eta = formatDuration(float64(remaining) / (speedMBps * 1024 * 1024))
		}

		if total <= 0 {
			fmt.Printf("\r%s sent | %.2f MB/s | ETA %s", formatBytes(sent), speedMBps, eta)
			return
		}
		percent := float64(sent) / float64(total) * 100
		fmt.Printf("\r%.2f%% | %s / %s | %.2f MB/s | ETA %s",
			percent, formatBytes(sent), formatBytes(total), speedMBps, eta)
	}
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
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.2f %s", float64(bytes)/float64(div), units[exp])
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
