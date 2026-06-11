package ws

import (
	"encoding/base64"
	"os"
	"time"

	"github.com/123100123/lanlink/protocol"
	"github.com/gorilla/websocket"
)

func handleFileStart(conn *websocket.Conn, msg protocol.Message) {
	var payload protocol.FileStartPayload

	err := protocol.DecodePayload(msg.Payload, &payload)
	if err != nil {
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

	_, err = transferManager.Start(
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
		},
	)
}

func handleFileChunk(conn *websocket.Conn, msg protocol.Message) {
	var payload protocol.FileChunkPayload

	err := protocol.DecodePayload(msg.Payload, &payload)
	if err != nil {
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

	transfer.mu.Lock()

	_, err = transfer.File.Write(data)
	if err != nil {
		transfer.mu.Unlock()
		transferManager.Cancel(payload.TransferID)
		writeError(conn, msg.ID, "failed to write chunk")
		return
	}

	transfer.Received += int64(len(data))

	transfer.mu.Unlock()

	writeFileChunkResponse(
		conn,
		msg.ID,
		protocol.FileChunkResponse{
			Status:     "chunk.received",
			TransferID: payload.TransferID,
		},
	)
}

func handleFileEnd(conn *websocket.Conn, msg protocol.Message) {
	var payload protocol.FileEndPayload

	err := protocol.DecodePayload(msg.Payload, &payload)
	if err != nil {
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
	
	err = transfer.File.Close()
	if err != nil {
		writeError(conn, msg.ID, "failed to close file")
		return
	}
	
	if transfer.Size != transfer.Received {
		os.Remove(transfer.TempPath)
		writeError(conn, msg.ID, "file size mismatch")
		return
	}
	
	err = os.Rename(transfer.TempPath, transfer.FinalPath)
	if err != nil {
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
		},
	)
}

func writeFileChunkResponse(
	conn *websocket.Conn,
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