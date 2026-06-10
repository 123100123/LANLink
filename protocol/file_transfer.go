package protocol

type FileSendPayload struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type FileSendResponse struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}