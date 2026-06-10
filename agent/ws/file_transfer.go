package ws

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"time"

	"github.com/123100123/lanlink/protocol"
	"github.com/gorilla/websocket"
)

func handleFileSend(
	conn *websocket.Conn,
	msg protocol.Message,
) {

	var payload protocol.FileSendPayload

	err := protocol.DecodePayload(
		msg.Payload,
		&payload,
	)

	if err != nil {
		writeError(
			conn,
			msg.ID,
			"invalid file payload",
		)
		return
	}

	data, err := base64.StdEncoding.DecodeString(
		payload.Content,
	)

	if err != nil {
		writeError(
			conn,
			msg.ID,
			"invalid base64 content",
		)
		return
	}

	err = os.MkdirAll(
		"received",
		0755,
	)

	if err != nil {
		writeError(
			conn,
			msg.ID,
			"failed creating folder",
		)
		return
	}

	path := filepath.Join(
		"received",
		payload.Filename,
	)

	err = os.WriteFile(
		path,
		data,
		0644,
	)

	if err != nil {
		writeError(
			conn,
			msg.ID,
			"failed saving file",
		)
		return
	}

	responsePayload, err := protocol.EncodePayload(
		protocol.FileSendResponse{
			Status: "saved",
			Path: path,
		},
	)

	if err != nil {
		writeError(
			conn,
			msg.ID,
			"failed encoding response",
		)
		return
	}

	response := protocol.Message{
		Type: "file.send.response",
		ID: msg.ID,
		Timestamp: time.Now().UnixMilli(),
		Payload: responsePayload,
	}

	conn.WriteJSON(response)
}