package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type resumableTransfer struct {
	TransferID string
	Filename   string
	SafeName   string
	TempPath   string
	FinalPath  string
	File       *os.File
	Size       int64
	Received   int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
	mu         sync.Mutex
}

type resumableManager struct {
	mu        sync.Mutex
	transfers map[string]*resumableTransfer
}

var resumableStore = &resumableManager{
	transfers: make(map[string]*resumableTransfer),
}

func (m *resumableManager) Start(transferID, filename string, size int64) (*resumableTransfer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.transfers[transferID]; ok {
		existing.mu.Lock()
		defer existing.mu.Unlock()
		return existing, nil
	}

	safeName := filepath.Base(filename)

	if err := os.MkdirAll("received/tmp", 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll("received", 0755); err != nil {
		return nil, err
	}

	tempPath := filepath.Join("received/tmp", "resumable_"+transferID+"_"+safeName)
	finalPath := uniqueResumablePath("received", safeName)

	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	t := &resumableTransfer{
		TransferID: transferID,
		Filename:   filename,
		SafeName:   safeName,
		TempPath:   tempPath,
		FinalPath:  finalPath,
		File:       file,
		Size:       size,
		Received:   0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	m.transfers[transferID] = t
	return t, nil
}

func (m *resumableManager) Get(transferID string) (*resumableTransfer, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.transfers[transferID]
	return t, ok
}

func (m *resumableManager) Remove(transferID string) {
	m.mu.Lock()
	t, ok := m.transfers[transferID]
	if ok {
		delete(m.transfers, transferID)
	}
	m.mu.Unlock()

	if !ok {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.File != nil {
		t.File.Close()
	}
	os.Remove(t.TempPath)
}

func (t *resumableTransfer) WriteChunkAt(offset int64, data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if offset != t.Received {
		return errOffsetMismatch
	}

	n, err := t.File.WriteAt(data, offset)
	if err != nil {
		return err
	}

	t.Received += int64(n)
	t.UpdatedAt = time.Now()
	return nil
}

func (t *resumableTransfer) Finalize() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Size != t.Received {
		return errSizeMismatch
	}

	if err := t.File.Sync(); err != nil {
		return err
	}
	if err := t.File.Close(); err != nil {
		return err
	}
	t.File = nil

	if err := os.Rename(t.TempPath, t.FinalPath); err != nil {
		os.Remove(t.TempPath)
		return err
	}

	return nil
}

var errOffsetMismatch = errors.New("offset mismatch")
var errSizeMismatch = errors.New("file size mismatch")

func uniqueResumablePath(dir string, filename string) string {
	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]

	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	for i := 1; ; i++ {
		candidate := filepath.Join(dir, base+"_"+strconv.Itoa(i)+ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func resumableStartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeResumableJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"status": "error", "error": "method not allowed",
		})
		return
	}

	_, ok := authenticateRequest(r)
	if !ok {
		writeResumableJSON(w, http.StatusUnauthorized, map[string]any{
			"status": "error", "error": "unauthorized",
		})
		return
	}

	var req struct {
		TransferID string `json:"transfer_id"`
		Filename   string `json:"filename"`
		Size       int64  `json:"size"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeResumableJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error", "error": "invalid request body",
		})
		return
	}

	if req.TransferID == "" {
		writeResumableJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error", "error": "missing transfer_id",
		})
		return
	}
	if req.Filename == "" {
		writeResumableJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error", "error": "missing filename",
		})
		return
	}

	t, err := resumableStore.Start(req.TransferID, req.Filename, req.Size)
	if err != nil {
		log.Println("resumable start failed:", err)
		writeResumableJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error", "error": "failed to start transfer",
		})
		return
	}

	t.mu.Lock()
	received := t.Received
	t.mu.Unlock()

	writeResumableJSON(w, http.StatusOK, map[string]any{
		"status":      "started",
		"transfer_id": t.TransferID,
		"filename":    t.Filename,
		"received":    received,
		"total":       t.Size,
	})
}

func resumableStatusHandler(w http.ResponseWriter, r *http.Request, transferID string) {
	if r.Method != http.MethodGet {
		writeResumableJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"status": "error", "error": "method not allowed",
		})
		return
	}

	_, ok := authenticateRequest(r)
	if !ok {
		writeResumableJSON(w, http.StatusUnauthorized, map[string]any{
			"status": "error", "error": "unauthorized",
		})
		return
	}

	t, ok := resumableStore.Get(transferID)
	if !ok {
		writeResumableJSON(w, http.StatusNotFound, map[string]any{
			"status":      "error",
			"transfer_id": transferID,
			"error":       "unknown transfer id",
		})
		return
	}

	t.mu.Lock()
	received := t.Received
	size := t.Size
	filename := t.Filename
	t.mu.Unlock()

	writeResumableJSON(w, http.StatusOK, map[string]any{
		"status":      "active",
		"transfer_id": transferID,
		"filename":    filename,
		"received":    received,
		"total":       size,
	})
}

func resumableChunkHandler(w http.ResponseWriter, r *http.Request, transferID string) {
	if r.Method != http.MethodPut {
		writeResumableJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"status": "error", "error": "method not allowed",
		})
		return
	}

	_, ok := authenticateRequest(r)
	if !ok {
		writeResumableJSON(w, http.StatusUnauthorized, map[string]any{
			"status": "error", "error": "unauthorized",
		})
		return
	}

	offsetRaw := r.URL.Query().Get("offset")
	if offsetRaw == "" {
		writeResumableJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error", "error": "missing offset",
		})
		return
	}

	offset, err := strconv.ParseInt(offsetRaw, 10, 64)
	if err != nil || offset < 0 {
		writeResumableJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error", "error": "invalid offset",
		})
		return
	}

	t, ok := resumableStore.Get(transferID)
	if !ok {
		writeResumableJSON(w, http.StatusNotFound, map[string]any{
			"status":      "error",
			"transfer_id": transferID,
			"error":       "unknown transfer id",
		})
		return
	}

	t.mu.Lock()
	expectedOffset := t.Received
	size := t.Size
	t.mu.Unlock()

	if offset != expectedOffset {
		writeResumableJSON(w, http.StatusConflict, map[string]any{
			"status":          "error",
			"error":           "offset mismatch",
			"expected_offset": expectedOffset,
			"received":        expectedOffset,
			"total":           size,
		})
		return
	}

	buf := make([]byte, 256*1024)
	written := int64(0)

	for {
		n, readErr := r.Body.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])

			if writeErr := t.WriteChunkAt(offset+written, chunk); writeErr != nil {
				if errors.Is(writeErr, errOffsetMismatch) {
					writeResumableJSON(w, http.StatusConflict, map[string]any{
						"status":          "error",
						"error":           "offset mismatch during write",
						"expected_offset": t.Received,
						"received":        t.Received,
						"total":           size,
					})
					return
				}
				log.Println("resumable chunk write failed:", writeErr)
				writeResumableJSON(w, http.StatusInternalServerError, map[string]any{
					"status": "error", "error": "failed to write chunk",
				})
				return
			}
			written += int64(n)
		}
		if readErr != nil {
			break
		}
	}

	t.mu.Lock()
	received := t.Received
	t.mu.Unlock()

	writeResumableJSON(w, http.StatusOK, map[string]any{
		"status":      "chunk.received",
		"transfer_id": transferID,
		"offset":      offset,
		"received":    received,
		"total":       size,
	})
}

func resumableFinishHandler(w http.ResponseWriter, r *http.Request, transferID string) {
	if r.Method != http.MethodPost {
		writeResumableJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"status": "error", "error": "method not allowed",
		})
		return
	}

	_, ok := authenticateRequest(r)
	if !ok {
		writeResumableJSON(w, http.StatusUnauthorized, map[string]any{
			"status": "error", "error": "unauthorized",
		})
		return
	}

	t, ok := resumableStore.Get(transferID)
	if !ok {
		writeResumableJSON(w, http.StatusNotFound, map[string]any{
			"status":      "error",
			"transfer_id": transferID,
			"error":       "unknown transfer id",
		})
		return
	}

	if err := t.Finalize(); err != nil {
		status := http.StatusInternalServerError
		message := "failed to finalize"

		if errors.Is(err, errSizeMismatch) {
			status = http.StatusBadRequest
			message = "file size mismatch"
		}

		t.mu.Lock()
		received := t.Received
		size := t.Size
		t.mu.Unlock()

		writeResumableJSON(w, status, map[string]any{
			"status":      "error",
			"transfer_id": transferID,
			"received":    received,
			"total":       size,
			"error":       message,
		})
		return
	}

	t.mu.Lock()
	received := t.Received
	size := t.Size
	finalPath := t.FinalPath
	safeName := t.SafeName
	t.mu.Unlock()

	resumableStore.Remove(transferID)

	writeResumableJSON(w, http.StatusOK, map[string]any{
		"status":      "saved",
		"transfer_id": transferID,
		"filename":    safeName,
		"path":        finalPath,
		"received":    received,
		"total":       size,
	})
}

func resumableDeleteHandler(w http.ResponseWriter, r *http.Request, transferID string) {
	if r.Method != http.MethodDelete {
		writeResumableJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"status": "error", "error": "method not allowed",
		})
		return
	}

	_, ok := authenticateRequest(r)
	if !ok {
		writeResumableJSON(w, http.StatusUnauthorized, map[string]any{
			"status": "error", "error": "unauthorized",
		})
		return
	}

	t, found := resumableStore.Get(transferID)
	if !found {
		writeResumableJSON(w, http.StatusNotFound, map[string]any{
			"status":      "error",
			"transfer_id": transferID,
			"error":       "unknown transfer id",
		})
		return
	}

	t.mu.Lock()
	transferID = t.TransferID
	t.mu.Unlock()

	resumableStore.Remove(transferID)

	writeResumableJSON(w, http.StatusOK, map[string]any{
		"status":      "cancelled",
		"transfer_id": transferID,
	})
}

func resumableSubresourceHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/transfers/resumable/")
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")

	transferID, err := url.PathUnescape(parts[0])
	if err != nil {
		writeResumableJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error", "error": "invalid transfer id",
		})
		return
	}

	if len(parts) == 1 {
		if r.Method == http.MethodDelete {
			resumableDeleteHandler(w, r, transferID)
			return
		}
		writeResumableJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error", "error": "missing action",
		})
		return
	}

	action := parts[1]

	switch action {
	case "status":
		resumableStatusHandler(w, r, transferID)
	case "chunk":
		resumableChunkHandler(w, r, transferID)
	case "finish":
		resumableFinishHandler(w, r, transferID)
	case "delete":
		resumableDeleteHandler(w, r, transferID)
	default:
		writeResumableJSON(w, http.StatusNotFound, map[string]any{
			"status": "error", "error": "unknown action: " + action,
		})
	}
}

func writeResumableJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
