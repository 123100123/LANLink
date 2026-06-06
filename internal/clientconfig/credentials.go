package clientconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Credentials struct {
	AgentAddress string `json:"agent_address"`
	DeviceID     string `json:"device_id"`
	AuthToken    string `json:"auth_token"`
}

func Path() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(homeDir, ".lanlink")

	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "credentials.json"), nil
}

func Save(creds Credentials) error {
	path, err := Path()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func Load() (*Credentials, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var creds Credentials

	err = json.Unmarshal(data, &creds)
	if err != nil {
		return nil, err
	}

	return &creds, nil
}