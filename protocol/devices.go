package protocol

type DeviceInfo struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
}

type DevicesResponse struct {
	Devices []DeviceInfo `json:"devices"`
}