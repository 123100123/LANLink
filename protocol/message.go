package protocol

type Message struct {
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	Module    string `json:"module,omitempty"`
	Action    string `json:"action,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}