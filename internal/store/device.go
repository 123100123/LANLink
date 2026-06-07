package store

type Device struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	AuthToken  string `json:"auth_token"`
}
