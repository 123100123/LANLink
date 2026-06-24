// Package client provides the reusable LANLink client-side operations (pair,
// health, devices, ping, message, send) shared by the cli and the unified
// lanlink terminal binary. Functions return errors instead of terminating, and
// print human-readable results to stdout.
package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/123100123/lanlink/internal/clientconfig"
	"github.com/123100123/lanlink/protocol"
	"github.com/gorilla/websocket"
)

func connectAuthenticated(address string) (*websocket.Conn, error) {
	creds, err := clientconfig.Load()
	if err != nil {
		return nil, fmt.Errorf("not paired yet, run pair command first")
	}

	conn, _, err := websocket.DefaultDialer.Dial("ws://"+address+"/ws", nil)
	if err != nil {
		return nil, fmt.Errorf("websocket connection failed: %w", err)
	}

	if err := authenticate(conn, creds.AuthToken); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func authenticate(conn *websocket.Conn, authToken string) error {
	authPayload, err := protocol.EncodePayload(protocol.AuthRequest{Token: authToken})
	if err != nil {
		return fmt.Errorf("failed to encode auth payload: %w", err)
	}

	authMessage := protocol.Message{
		Type:      "auth",
		ID:        "auth_1",
		Timestamp: time.Now().UnixMilli(),
		Payload:   authPayload,
	}
	if err := conn.WriteJSON(authMessage); err != nil {
		return fmt.Errorf("failed to send auth message: %w", err)
	}

	var authResponse protocol.Message
	if err := conn.ReadJSON(&authResponse); err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	if authResponse.Type == "auth.failed" {
		var failed protocol.AuthFailed
		if err := protocol.DecodePayload(authResponse.Payload, &failed); err != nil {
			return fmt.Errorf("failed to decode auth failure: %w", err)
		}
		return fmt.Errorf("websocket authentication failed: %s", failed.Error)
	}
	if authResponse.Type != "auth.success" {
		return fmt.Errorf("unexpected websocket response: %s", authResponse.Type)
	}

	var success protocol.AuthSuccess
	if err := protocol.DecodePayload(authResponse.Payload, &success); err != nil {
		return fmt.Errorf("failed to decode auth success: %w", err)
	}

	fmt.Println("WebSocket connected")
	fmt.Println("Authenticated")
	fmt.Println("Device ID:", success.DeviceID)
	fmt.Println("Device Name:", success.DeviceName)
	return nil
}

// WSHello connects, authenticates, and sends a hello message (debug helper).
func WSHello(address string) error {
	conn, err := connectAuthenticated(address)
	if err != nil {
		return err
	}
	defer conn.Close()

	helloPayload, err := protocol.EncodePayload("hello from authenticated cli")
	if err != nil {
		return fmt.Errorf("failed to encode hello payload: %w", err)
	}
	helloMessage := protocol.Message{
		Type:      "hello",
		ID:        "hello_1",
		Timestamp: time.Now().Unix(),
		Payload:   helloPayload,
	}
	if err := conn.WriteJSON(helloMessage); err != nil {
		return fmt.Errorf("failed to send hello message: %w", err)
	}

	var helloResponse protocol.Message
	if err := conn.ReadJSON(&helloResponse); err != nil {
		return fmt.Errorf("failed to read hello response: %w", err)
	}

	fmt.Println("Post-auth response:")
	fmt.Println("Type:", helloResponse.Type)
	fmt.Println("ID:", helloResponse.ID)
	fmt.Println("Payload:", string(helloResponse.Payload))
	return nil
}

// Message sends a direct text message to the agent over the websocket.
func Message(address string, parts []string) error {
	if len(parts) == 0 {
		return fmt.Errorf("message text is required")
	}
	text := strings.Join(parts, " ")

	conn, err := connectAuthenticated(address)
	if err != nil {
		return err
	}
	defer conn.Close()

	payload, err := protocol.EncodePayload(protocol.DirectMessagePayload{Text: text})
	if err != nil {
		return fmt.Errorf("failed to encode message payload: %w", err)
	}
	msg := protocol.Message{
		Type:      "direct_message",
		ID:        "msg_1",
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}
	if err := conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	var response protocol.Message
	if err := conn.ReadJSON(&response); err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "done"))

	if response.Type != "direct_message.response" {
		return fmt.Errorf("unexpected response: %s", response.Type)
	}
	var responsePayload protocol.DirectMessageResponse
	if err := protocol.DecodePayload(response.Payload, &responsePayload); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Println("Message sent")
	fmt.Println("Agent response:", responsePayload.Status)
	return nil
}

// Ping measures websocket round-trip latency to the agent.
func Ping(address string) error {
	conn, err := connectAuthenticated(address)
	if err != nil {
		return err
	}
	defer conn.Close()

	sentAt := time.Now().UnixMilli()
	payload, err := protocol.EncodePayload(protocol.PingPayload{SentAt: sentAt})
	if err != nil {
		return fmt.Errorf("failed to encode ping payload: %w", err)
	}
	message := protocol.Message{
		Type:      "ping",
		ID:        "ping_1",
		Timestamp: sentAt,
		Payload:   payload,
	}
	if err := conn.WriteJSON(message); err != nil {
		return fmt.Errorf("failed to send ping: %w", err)
	}

	var response protocol.Message
	if err := conn.ReadJSON(&response); err != nil {
		return fmt.Errorf("failed to read pong: %w", err)
	}
	if response.Type != "pong" {
		return fmt.Errorf("expected pong, got: %s", response.Type)
	}

	receivedAt := time.Now().UnixMilli()
	var pong protocol.PongPayload
	if err := protocol.DecodePayload(response.Payload, &pong); err != nil {
		return fmt.Errorf("failed to decode pong payload: %w", err)
	}

	fmt.Println("Pong received")
	fmt.Println("Latency:", receivedAt-pong.SentAt, "ms")
	return nil
}
