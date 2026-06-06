package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type DeviceStore struct {
	Devices []Device `json:"devices"`
}

func Load(path string) (*DeviceStore, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		if os.IsNotExist(err) {
			return &DeviceStore{
				Devices: []Device{},
			}, nil
		}

		return nil, err
	}

	var store DeviceStore

	err = json.Unmarshal(data, &store)
	if err != nil {
		return nil, err
	}

	return &store, nil
}

func (s *DeviceStore) Save(path string) error {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (s *DeviceStore) AddDevice(device Device) {
	s.Devices = append(s.Devices, device)
}