package protocol

type PingPayload struct {
	SentAt int64 `json:"sent_at"`
}

type PongPayload struct {
	SentAt     int64 `json:"sent_at"`
	ReceivedAt int64 `json:"received_at"`
}