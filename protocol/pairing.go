package protocol

type PairRequest struct {
	DeviceName string `json:"device_name"`
	Token      string `json:"token"`
}

type PairResponse struct {
	Status    string `json:"status"`
	DeviceID  string `json:"device_id,omitempty"`
	AuthToken string `json:"auth_token,omitempty"`
	Error     string `json:"error,omitempty"`
}
