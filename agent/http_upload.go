package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/123100123/lanlink/agent/dashboard"
)

func transferUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeUploadJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"status": "error",
			"error":  "method not allowed",
		})
		return
	}

	_, ok := authenticateRequest(r)
	if !ok {
		writeUploadJSON(w, http.StatusUnauthorized, map[string]any{
			"status": "error",
			"error":  "unauthorized",
		})
		return
	}

	filename := r.Header.Get("X-Filename")
	transferID := r.Header.Get("X-Transfer-Id")

	if filename == "" {
		writeUploadJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error",
			"error":  "missing X-Filename header",
		})
		return
	}

	safeName := filepath.Base(filename)
	outputDir := dashboard.GetOutputDir()

	tmpDir := filepath.Join(outputDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		log.Println("upload: mkdir tmp failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "server error",
		})
		return
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Println("upload: mkdir output failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "server error",
		})
		return
	}

	finalPath := uniqueUploadPath(outputDir, safeName)

	tmpName := safeName
	if transferID != "" {
		tmpName = transferID + "_" + safeName
	}
	tempPath := filepath.Join(tmpDir, tmpName)

	out, err := os.Create(tempPath)
	if err != nil {
		log.Println("upload: create temp file failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "server error",
		})
		return
	}

	dashID := transferID
	if dashID == "" {
		dashID = safeName
	}
	dashboard.AddTransfer(dashID, safeName, 0)
	startTime := time.Now()

	buf := make([]byte, 256*1024)
	written, err := io.CopyBuffer(out, r.Body, buf)

	if err != nil {
		out.Close()
		os.Remove(tempPath)
		dashboard.FailTransfer(dashID, "write failed")
		log.Println("upload: write failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "write failed",
		})
		return
	}

	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(tempPath)
		dashboard.FailTransfer(dashID, "sync failed")
		log.Println("upload: sync failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "sync failed",
		})
		return
	}

	if err := out.Close(); err != nil {
		os.Remove(tempPath)
		dashboard.FailTransfer(dashID, "close failed")
		log.Println("upload: close failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "close failed",
		})
		return
	}

	if err := os.Rename(tempPath, finalPath); err != nil {
		os.Remove(tempPath)
		dashboard.FailTransfer(dashID, "rename failed")
		log.Println("upload: rename failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "rename failed",
		})
		return
	}

	dashboard.CompleteTransfer(dashID, finalPath)

	elapsed := time.Since(startTime).Seconds()
	speed := int64(0)
	if elapsed > 0 {
		speed = int64(float64(written) / elapsed)
	}

	log.Printf("upload: saved %s (%d bytes) as %s [%.1f MB/s]", safeName, written, finalPath, float64(speed)/1024/1024)

	writeUploadJSON(w, http.StatusOK, map[string]any{
		"status":      "saved",
		"transfer_id": transferID,
		"filename":    safeName,
		"path":        finalPath,
		"received":    written,
	})
}

func writeUploadJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}

func uniqueUploadPath(dir string, filename string) string {
	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]

	path := filepath.Join(dir, filename)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	for i := 1; ; i++ {
		candidate := filepath.Join(
			dir,
			base+"_"+strconv.Itoa(i)+ext,
		)

		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
