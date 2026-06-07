package protocol

type AuthRequest struct {
	Token string `json:"token"`
}

type AuthSuccess struct {
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
}

type AuthFailed struct {
	Error string `json:"error"`
}
