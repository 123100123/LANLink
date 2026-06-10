package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	cliws "github.com/123100123/lanlink/cli/ws"
	"github.com/123100123/lanlink/protocol"
)

func sendFile(
	address string,
	filePath string,
) {

	data, err := os.ReadFile(
		filePath,
	)

	if err != nil {
		log.Fatal(err)
	}

	content := base64.StdEncoding.EncodeToString(
		data,
	)

	payload, err := protocol.EncodePayload(
		protocol.FileSendPayload{
			Filename: filepath.Base(filePath),
			Content: content,
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	conn := cliws.ConnectAuthenticated(
		address,
	)

	defer conn.Close()

	msg := protocol.Message{
		Type: "file.send",
		ID: "file_1",
		Timestamp: time.Now().UnixMilli(),
		Payload: payload,
	}

	err = conn.WriteJSON(msg)

	if err != nil {
		log.Fatal(err)
	}

	var response protocol.Message

	err = conn.ReadJSON(&response)

	if err != nil {
		log.Fatal(err)
	}

	var result protocol.FileSendResponse

	err = protocol.DecodePayload(
		response.Payload,
		&result,
	)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("File uploaded")
	fmt.Println("Saved as:", result.Path)
}