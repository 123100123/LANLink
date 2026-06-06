package store

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/123100123/lanlink/protocol"
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

func (s *DeviceStore) PublicDevices() []protocol.DeviceInfo {
	devices := make([]protocol.DeviceInfo, 0, len(s.Devices))

	for _, device := range s.Devices {
		devices = append(devices, protocol.DeviceInfo{
			DeviceID:   device.DeviceID,
			DeviceName: device.DeviceName,
		})
	}

	return devices
}

func (s *DeviceStore) FindDeviceByToken(authToken string) (*Device, bool) {
	for _, device := range s.Devices {
		if device.AuthToken == authToken {
			return &device, true
		}
	}

	return nil, false
}