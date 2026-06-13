package protocol

type TransferStartRequest struct {
	TransferID string `json:"transfer_id"`
	Filename   string `json:"filename"`
	Size       int64  `json:"size"`
}

type TransferStartResponse struct {
	Status     string `json:"status"`
	TransferID string `json:"transfer_id,omitempty"`
	Total      int64  `json:"total,omitempty"`
	Error      string `json:"error,omitempty"`
}

type TransferChunkResponse struct {
	Status     string `json:"status"`
	TransferID string `json:"transfer_id,omitempty"`
	Index      int    `json:"index,omitempty"`
	Offset     int64  `json:"offset,omitempty"`
	Received   int64  `json:"received,omitempty"`
	Total      int64  `json:"total,omitempty"`
	Error      string `json:"error,omitempty"`
}

type TransferFinishResponse struct {
	Status     string `json:"status"`
	TransferID string `json:"transfer_id,omitempty"`
	Path       string `json:"path,omitempty"`
	Received   int64  `json:"received,omitempty"`
	Total      int64  `json:"total,omitempty"`
	Error      string `json:"error,omitempty"`
}
