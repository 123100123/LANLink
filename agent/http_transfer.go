package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/123100123/lanlink/agent/dashboard"
	transferpkg "github.com/123100123/lanlink/internal/transfer"
	"github.com/123100123/lanlink/protocol"
)

var httpTransferManager = transferpkg.NewManager(dashboard.GetOutputDir)

func transferStartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeTransferJSON(
			w,
			http.StatusMethodNotAllowed,
			protocol.TransferStartResponse{
				Status: "error",
				Error:  "method not allowed",
			},
		)
		return
	}

	_, ok := authenticateRequest(r)
	if !ok {
		writeTransferJSON(
			w,
			http.StatusUnauthorized,
			protocol.TransferStartResponse{
				Status: "error",
				Error:  "unauthorized",
			},
		)
		return
	}

	var req protocol.TransferStartRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeTransferJSON(
			w,
			http.StatusBadRequest,
			protocol.TransferStartResponse{
				Status: "error",
				Error:  "invalid request body",
			},
		)
		return
	}

	if req.TransferID == "" {
		writeTransferJSON(
			w,
			http.StatusBadRequest,
			protocol.TransferStartResponse{
				Status: "error",
				Error:  "missing transfer id",
			},
		)
		return
	}

	if req.Filename == "" {
		writeTransferJSON(
			w,
			http.StatusBadRequest,
			protocol.TransferStartResponse{
				Status: "error",
				Error:  "missing filename",
			},
		)
		return
	}

	if req.Size < 0 {
		writeTransferJSON(
			w,
			http.StatusBadRequest,
			protocol.TransferStartResponse{
				Status: "error",
				Error:  "invalid file size",
			},
		)
		return
	}

	_, err := httpTransferManager.Start(
		req.TransferID,
		req.Filename,
		req.Size,
	)
	if err != nil {
		status := http.StatusInternalServerError
		message := "failed to start transfer"

		if errors.Is(err, transferpkg.ErrDuplicateTransfer) {
			status = http.StatusConflict
			message = "duplicate transfer id"
		}

		writeTransferJSON(
			w,
			status,
			protocol.TransferStartResponse{
				Status: "error",
				Error:  message,
			},
		)
		return
	}

	dashboard.AddTransfer(req.TransferID, req.Filename, req.Size)

	writeTransferJSON(
		w,
		http.StatusOK,
		protocol.TransferStartResponse{
			Status:     "started",
			TransferID: req.TransferID,
			Total:      req.Size,
		},
	)
}

func transferSubresourceHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := authenticateRequest(r)
	if !ok {
		writeTransferJSON(
			w,
			http.StatusUnauthorized,
			protocol.TransferChunkResponse{
				Status: "error",
				Error:  "unauthorized",
			},
		)
		return
	}

	transferID, action, index, err := parseTransferPath(r.URL.Path)
	if err != nil {
		writeTransferJSON(
			w,
			http.StatusBadRequest,
			protocol.TransferChunkResponse{
				Status: "error",
				Error:  err.Error(),
			},
		)
		return
	}

	switch action {
	case "chunk":
		handleHTTPTransferChunk(w, r, transferID, index)

	case "finish":
		handleHTTPTransferFinish(w, r, transferID)

	default:
		writeTransferJSON(
			w,
			http.StatusNotFound,
			protocol.TransferChunkResponse{
				Status: "error",
				Error:  "unknown transfer action",
			},
		)
	}
}

func handleHTTPTransferChunk(
	w http.ResponseWriter,
	r *http.Request,
	transferID string,
	index int,
) {
	if r.Method != http.MethodPut {
		writeTransferJSON(
			w,
			http.StatusMethodNotAllowed,
			protocol.TransferChunkResponse{
				Status: "error",
				Error:  "method not allowed",
			},
		)
		return
	}

	offsetRaw := r.URL.Query().Get("offset")
	if offsetRaw == "" {
		writeTransferJSON(
			w,
			http.StatusBadRequest,
			protocol.TransferChunkResponse{
				Status: "error",
				Error:  "missing offset",
			},
		)
		return
	}

	offset, err := strconv.ParseInt(offsetRaw, 10, 64)
	if err != nil || offset < 0 {
		writeTransferJSON(
			w,
			http.StatusBadRequest,
			protocol.TransferChunkResponse{
				Status: "error",
				Error:  "invalid offset",
			},
		)
		return
	}

	active, ok := httpTransferManager.Get(transferID)
	if !ok {
		writeTransferJSON(
			w,
			http.StatusNotFound,
			protocol.TransferChunkResponse{
				Status:     "error",
				TransferID: transferID,
				Error:      "unknown transfer id",
			},
		)
		return
	}

	defer r.Body.Close()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		httpTransferManager.Cancel(transferID)
		writeTransferJSON(
			w,
			http.StatusInternalServerError,
			protocol.TransferChunkResponse{
				Status:     "error",
				TransferID: transferID,
				Error:      "failed to read chunk body",
			},
		)
		return
	}

	received, err := active.WriteChunk(index, offset, data)
	if err != nil {
		if errors.Is(err, transferpkg.ErrDuplicateChunk) {
			writeTransferJSON(
				w,
				http.StatusOK,
				protocol.TransferChunkResponse{
					Status:     "chunk.duplicate",
					TransferID: transferID,
					Index:      index,
					Offset:     offset,
					Received:   received,
					Total:      active.Size,
				},
			)
			return
		}

		httpTransferManager.Cancel(transferID)
		dashboard.FailTransfer(transferID, "write chunk failed")
		writeTransferJSON(
			w,
			http.StatusBadRequest,
			protocol.TransferChunkResponse{
				Status:     "error",
				TransferID: transferID,
				Index:      index,
				Offset:     offset,
				Error:      "failed to write chunk",
			},
		)
		return
	}

	dashboard.UpdateTransfer(transferID, received, 0)

	writeTransferJSON(
		w,
		http.StatusOK,
		protocol.TransferChunkResponse{
			Status:     "chunk.received",
			TransferID: transferID,
			Index:      index,
			Offset:     offset,
			Received:   received,
			Total:      active.Size,
		},
	)
}

func handleHTTPTransferFinish(
	w http.ResponseWriter,
	r *http.Request,
	transferID string,
) {
	if r.Method != http.MethodPost {
		writeTransferJSON(
			w,
			http.StatusMethodNotAllowed,
			protocol.TransferFinishResponse{
				Status: "error",
				Error:  "method not allowed",
			},
		)
		return
	}

	active, ok := httpTransferManager.Finish(transferID)
	if !ok {
		writeTransferJSON(
			w,
			http.StatusNotFound,
			protocol.TransferFinishResponse{
				Status:     "error",
				TransferID: transferID,
				Error:      "unknown transfer id",
			},
		)
		return
	}

	if err := active.Finalize(); err != nil {
		status := http.StatusInternalServerError
		message := "failed to finalize file"

		if errors.Is(err, transferpkg.ErrSizeMismatch) {
			status = http.StatusBadRequest
			message = "file size mismatch"
		}

		dashboard.FailTransfer(transferID, message)
		writeTransferJSON(
			w,
			status,
			protocol.TransferFinishResponse{
				Status:     "error",
				TransferID: transferID,
				Received:   active.ReceivedBytes,
				Total:      active.Size,
				Error:      message,
			},
		)
		return
	}

	dashboard.CompleteTransfer(transferID, active.FinalPath)

	writeTransferJSON(
		w,
		http.StatusOK,
		protocol.TransferFinishResponse{
			Status:     "saved",
			TransferID: transferID,
			Path:       active.FinalPath,
			Received:   active.ReceivedBytes,
			Total:      active.Size,
		},
	)
}

func parseTransferPath(path string) (string, string, int, error) {
	trimmed := strings.TrimPrefix(path, "/transfers/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")

	if len(parts) == 2 && parts[1] == "finish" {
		transferID, err := url.PathUnescape(parts[0])
		if err != nil {
			return "", "", 0, err
		}

		return transferID, "finish", 0, nil
	}

	if len(parts) == 3 && parts[1] == "chunks" {
		transferID, err := url.PathUnescape(parts[0])
		if err != nil {
			return "", "", 0, err
		}

		index, err := strconv.Atoi(parts[2])
		if err != nil || index < 0 {
			return "", "", 0, errors.New("invalid chunk index")
		}

		return transferID, "chunk", index, nil
	}

	return "", "", 0, errors.New("invalid transfer path")
}

func writeTransferJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
