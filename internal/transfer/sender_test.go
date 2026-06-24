package transfer

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/123100123/lanlink/protocol"
)

// TestSendFile exercises the full start → chunks → finish flow against a fake
// receiver and asserts every byte arrives at the correct offset.
func TestSendFile(t *testing.T) {
	content := bytes.Repeat([]byte("lanlink-"), 4096) // 32 KiB, multiple chunks
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.bin")
	if err := os.WriteFile(srcPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	var (
		mu        sync.Mutex
		assembled = make([]byte, len(content))
		finished  bool
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/transfers/start", func(w http.ResponseWriter, r *http.Request) {
		var req protocol.TransferStartRequest
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(protocol.TransferStartResponse{
			Status: "started", TransferID: req.TransferID, Total: req.Size,
		})
	})
	mux.HandleFunc("/transfers/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, "/")
		switch {
		case strings.HasSuffix(path, "/finish"):
			mu.Lock()
			finished = true
			mu.Unlock()
			json.NewEncoder(w).Encode(protocol.TransferFinishResponse{Status: "saved", Path: "/tmp/out.bin"})
		case strings.Contains(path, "/chunks/"):
			offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			copy(assembled[offset:], body)
			mu.Unlock()
			json.NewEncoder(w).Encode(protocol.TransferChunkResponse{Status: "chunk.received"})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	address := strings.TrimPrefix(server.URL, "http://")

	result, err := SendFile(address, "tok", srcPath, SendOptions{
		ChunkSize:         4096,
		MaxInFlightChunks: 4,
	})
	if err != nil {
		t.Fatalf("SendFile: %v", err)
	}
	if result.Path != "/tmp/out.bin" {
		t.Fatalf("unexpected result path: %q", result.Path)
	}
	if !finished {
		t.Fatal("finish was never called")
	}
	if !bytes.Equal(assembled, content) {
		t.Fatal("assembled bytes do not match source")
	}
}

func TestSendFileRejectsZeroChunkSize(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	os.WriteFile(p, []byte("x"), 0o644)

	if _, err := SendFile("127.0.0.1:1", "tok", p, SendOptions{ChunkSize: 0}); err == nil {
		t.Fatal("expected error for zero chunk size")
	}
}
