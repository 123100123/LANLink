package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

	if err := os.MkdirAll("received/tmp", 0755); err != nil {
		log.Println("upload: mkdir tmp failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "server error",
		})
		return
	}

	if err := os.MkdirAll("received", 0755); err != nil {
		log.Println("upload: mkdir received failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "server error",
		})
		return
	}

	finalPath := uniqueUploadPath("received", safeName)

	tmpName := safeName
	if transferID != "" {
		tmpName = transferID + "_" + safeName
	}
	tempPath := filepath.Join("received", "tmp", tmpName)

	out, err := os.Create(tempPath)
	if err != nil {
		log.Println("upload: create temp file failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "server error",
		})
		return
	}

	buf := make([]byte, 256*1024)
	written, err := io.CopyBuffer(out, r.Body, buf)

	if err != nil {
		out.Close()
		os.Remove(tempPath)
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
		log.Println("upload: sync failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "sync failed",
		})
		return
	}

	if err := out.Close(); err != nil {
		os.Remove(tempPath)
		log.Println("upload: close failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "close failed",
		})
		return
	}

	if err := os.Rename(tempPath, finalPath); err != nil {
		os.Remove(tempPath)
		log.Println("upload: rename failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "rename failed",
		})
		return
	}

	log.Printf("upload: saved %s (%d bytes) as %s", safeName, written, finalPath)

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
