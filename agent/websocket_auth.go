package main

import (
	"time"

	"github.com/123100123/lanlink/internal/store"
	"github.com/123100123/lanlink/protocol"
	"github.com/gorilla/websocket"
)

func authenticateWebSocket(conn *websocket.Conn) (*store.Device, bool) {
	var msg protocol.Message

	err := conn.ReadJSON(&msg)
	if err != nil {
		writeAuthFailed(conn, "failed to read auth message")
		return nil, false
	}

	if msg.Type != "auth" {
		writeAuthFailed(conn, "first websocket message must be auth")
		return nil, false
	}

	var authRequest protocol.AuthRequest

	err = protocol.DecodePayload(msg.Payload, &authRequest)
	if err != nil {
		writeAuthFailed(conn, "invalid auth payload")
		return nil, false
	}

	deviceStore, err := store.Load(deviceStorePath)
	if err != nil {
		writeAuthFailed(conn, "failed to load device store")
		return nil, false
	}

	device, ok := deviceStore.FindDeviceByToken(authRequest.Token)
	if !ok {
		writeAuthFailed(conn, "invalid auth token")
		return nil, false
	}

	writeAuthSuccess(conn, device)

	return device, true
}

func writeAuthSuccess(conn *websocket.Conn, device *store.Device) {
	payload, err := protocol.EncodePayload(protocol.AuthSuccess{
		DeviceID:   device.DeviceID,
		DeviceName: device.DeviceName,
	})
	if err != nil {
		return
	}

	msg := protocol.Message{
		Type:      "auth.success",
		Timestamp: time.Now().Unix(),
		Payload:   payload,
	}

	conn.WriteJSON(msg)
}

func writeAuthFailed(conn *websocket.Conn, reason string) {
	payload, err := protocol.EncodePayload(protocol.AuthFailed{
		Error: reason,
	})
	if err != nil {
		return
	}

	msg := protocol.Message{
		Type:      "auth.failed",
		Timestamp: time.Now().Unix(),
		Payload:   payload,
	}

	conn.WriteJSON(msg)
}