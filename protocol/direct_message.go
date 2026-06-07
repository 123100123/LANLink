package protocol

type DirectMessagePayload struct {
	Text string `json:"text"`
}

type DirectMessageResponse struct {
	Status string `json:"status"`
}
