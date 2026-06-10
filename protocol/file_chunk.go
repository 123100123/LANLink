package protocol

type FileStartPayload struct {
	TransferID string `json:"transfer_id"`
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
}

type FileChunkPayload struct {
	TransferID string `json:"transfer_id"`
	Index      int    `json:"index"`
	Content    string `json:"content"`
}

type FileEndPayload struct {
	TransferID string `json:"transfer_id"`
}

type FileChunkResponse struct {
	Status     string `json:"status"`
	TransferID string `json:"transfer_id,omitempty"`
	Path       string `json:"path,omitempty"`
	Error      string `json:"error,omitempty"`
}