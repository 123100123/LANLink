package agentserver

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
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
	preferredID := r.Header.Get("X-Transfer-Id")

	if filename == "" {
		writeUploadJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error",
			"error":  "missing X-Filename header",
		})
		return
	}

	safeName := filepath.Base(filename)
	outputDir := GetOutputDir()

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

	dashID := ReserveTransferID(preferredID, safeName)

	tmpName := dashID + "_" + safeName
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

	var totalSize int64
	if r.ContentLength > 0 {
		totalSize = r.ContentLength
	} else if hs := r.Header.Get("X-File-Size"); hs != "" {
		if v, err := strconv.ParseInt(hs, 10, 64); err == nil && v > 0 {
			totalSize = v
		}
	}

	AddTransfer(dashID, safeName, totalSize)
	startTime := time.Now()

	buf := make([]byte, 512*1024)
	var written int64
	lastDashUpdate := startTime

	for {
		if IsTransferCancelled(dashID) {
			out.Close()
			os.Remove(tempPath)
			log.Printf("upload: cancelled %s (%d bytes received)", safeName, written)
			writeUploadJSON(w, http.StatusConflict, map[string]any{
				"status":      "cancelled",
				"transfer_id": dashID,
				"error":       "transfer cancelled",
			})
			return
		}

		n, readErr := r.Body.Read(buf)

		if n > 0 {
			wn, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				out.Close()
				os.Remove(tempPath)
				FailTransfer(dashID, "write failed")
				log.Println("upload: write failed:", writeErr)
				writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
					"status": "error",
					"error":  "write failed",
				})
				return
			}

			written += int64(wn)

			if time.Since(lastDashUpdate) >= 500*time.Millisecond {
				speed := calcSpeed(written, startTime)
				UpdateTransfer(dashID, written, speed)
				lastDashUpdate = time.Now()
			}
		}

		if readErr == io.EOF {
			break
		}

		if readErr != nil {
			out.Close()
			os.Remove(tempPath)
			FailTransfer(dashID, "read failed")
			log.Println("upload: read failed:", readErr)
			writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
				"status": "error",
				"error":  "read failed",
			})
			return
		}
	}

	finalSpeed := calcSpeed(written, startTime)
	UpdateTransfer(dashID, written, finalSpeed)

	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(tempPath)
		FailTransfer(dashID, "sync failed")
		log.Println("upload: sync failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "sync failed",
		})
		return
	}

	if err := out.Close(); err != nil {
		os.Remove(tempPath)
		FailTransfer(dashID, "close failed")
		log.Println("upload: close failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "close failed",
		})
		return
	}

	if err := os.Rename(tempPath, finalPath); err != nil {
		os.Remove(tempPath)
		FailTransfer(dashID, "rename failed")
		log.Println("upload: rename failed:", err)
		writeUploadJSON(w, http.StatusInternalServerError, map[string]any{
			"status": "error",
			"error":  "rename failed",
		})
		return
	}

	CompleteTransfer(dashID, finalPath)

	log.Printf("upload: saved %s (%d bytes) as %s [%.1f MB/s]", safeName, written, finalPath, float64(finalSpeed)/1024/1024)

	writeUploadJSON(w, http.StatusOK, map[string]any{
		"status":      "saved",
		"transfer_id": dashID,
		"filename":    safeName,
		"path":        finalPath,
		"received":    written,
	})
}

func calcSpeed(bytes int64, startTime time.Time) int64 {
	elapsed := time.Since(startTime).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return int64(float64(bytes) / elapsed)
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
