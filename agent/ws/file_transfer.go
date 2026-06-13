package ws

import (
	"encoding/base64"
	"errors"
	"os"
	"time"

	"github.com/123100123/lanlink/internal/wsutil"
	"github.com/123100123/lanlink/protocol"
)

func handleFileStart(conn *wsutil.SafeConn, msg protocol.Message) {
	var payload protocol.FileStartPayload

	if err := protocol.DecodePayload(msg.Payload, &payload); err != nil {
		writeError(conn, msg.ID, "invalid file.start payload")
		return
	}

	if payload.TransferID == "" {
		writeError(conn, msg.ID, "missing transfer id")
		return
	}

	if payload.Filename == "" {
		writeError(conn, msg.ID, "missing filename")
		return
	}

	if payload.Size < 0 {
		writeError(conn, msg.ID, "invalid file size")
		return
	}

	_, err := transferManager.Start(
		payload.TransferID,
		payload.Filename,
		payload.Size,
	)
	if err != nil {
		writeError(conn, msg.ID, "failed to start transfer")
		return
	}

	writeFileChunkResponse(
		conn,
		msg.ID,
		protocol.FileChunkResponse{
			Status:     "started",
			TransferID: payload.TransferID,
			Total:      payload.Size,
		},
	)
}

func handleFileChunk(conn *wsutil.SafeConn, msg protocol.Message) {
	var payload protocol.FileChunkPayload

	if err := protocol.DecodePayload(msg.Payload, &payload); err != nil {
		writeError(conn, msg.ID, "invalid file.chunk payload")
		return
	}

	transfer, ok := transferManager.Get(payload.TransferID)
	if !ok {
		writeError(conn, msg.ID, "unknown transfer id")
		return
	}

	data, err := base64.StdEncoding.DecodeString(payload.Content)
	if err != nil {
		writeError(conn, msg.ID, "invalid chunk content")
		return
	}

	if payload.Length != len(data) {
		writeError(conn, msg.ID, "chunk length mismatch")
		return
	}

	received, err := transfer.WriteChunk(
		payload.Index,
		payload.Offset,
		data,
	)
	if err != nil {
		if errors.Is(err, ErrDuplicateChunk) {
			writeFileChunkResponse(
				conn,
				msg.ID,
				protocol.FileChunkResponse{
					Status:     "chunk.duplicate",
					TransferID: payload.TransferID,
					Index:      payload.Index,
					Offset:     payload.Offset,
					Received:   received,
					Total:      transfer.Size,
				},
			)
			return
		}

		transferManager.Cancel(payload.TransferID)
		writeError(conn, msg.ID, "failed to write chunk")
		return
	}

	writeFileChunkResponse(
		conn,
		msg.ID,
		protocol.FileChunkResponse{
			Status:     "chunk.received",
			TransferID: payload.TransferID,
			Index:      payload.Index,
			Offset:     payload.Offset,
			Received:   received,
			Total:      transfer.Size,
		},
	)
}

func handleFileEnd(conn *wsutil.SafeConn, msg protocol.Message) {
	var payload protocol.FileEndPayload

	if err := protocol.DecodePayload(msg.Payload, &payload); err != nil {
		writeError(conn, msg.ID, "invalid file.end payload")
		return
	}

	transfer, ok := transferManager.Finish(payload.TransferID)
	if !ok {
		writeError(conn, msg.ID, "unknown transfer id")
		return
	}

	transfer.mu.Lock()
	defer transfer.mu.Unlock()

	if transfer.Size != transfer.ReceivedBytes {
		transfer.File.Close()
		os.Remove(transfer.TempPath)
		writeError(conn, msg.ID, "file size mismatch")
		return
	}

	if err := transfer.File.Sync(); err != nil {
		transfer.File.Close()
		os.Remove(transfer.TempPath)
		writeError(conn, msg.ID, "failed to sync file")
		return
	}

	if err := transfer.File.Close(); err != nil {
		os.Remove(transfer.TempPath)
		writeError(conn, msg.ID, "failed to close file")
		return
	}

	if err := os.Rename(transfer.TempPath, transfer.FinalPath); err != nil {
		os.Remove(transfer.TempPath)
		writeError(conn, msg.ID, "failed to finalize file")
		return
	}

	writeFileChunkResponse(
		conn,
		msg.ID,
		protocol.FileChunkResponse{
			Status:     "saved",
			TransferID: payload.TransferID,
			Path:       transfer.FinalPath,
			Received:   transfer.ReceivedBytes,
			Total:      transfer.Size,
		},
	)
}

func writeFileChunkResponse(
	conn *wsutil.SafeConn,
	id string,
	response protocol.FileChunkResponse,
) {
	payload, err := protocol.EncodePayload(response)
	if err != nil {
		return
	}

	msg := protocol.Message{
		Type:      "file.chunk.response",
		ID:        id,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}

	conn.WriteJSON(msg)
}
