package ws

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type ActiveTransfer struct {
	TransferID string
	Filename   string
	TempPath   string
	FinalPath  string
	File       *os.File
	Size       int64
	Received   int64
	mu         sync.Mutex
}

type TransferManager struct {
	mu        sync.Mutex
	transfers map[string]*ActiveTransfer
}

func NewTransferManager() *TransferManager {
	return &TransferManager{
		transfers: make(map[string]*ActiveTransfer),
	}
}

func (m *TransferManager) Start(
	transferID string,
	filename string,
	size int64,
) (*ActiveTransfer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	safeName := filepath.Base(filename)

	err := os.MkdirAll("received/tmp", 0755)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll("received", 0755)
	if err != nil {
		return nil, err
	}

	finalPath := uniquePath("received", safeName)
	tempPath := filepath.Join("received", "tmp", transferID+"_"+safeName)

	file, err := os.Create(tempPath)
	if err != nil {
		return nil, err
	}

	transfer := &ActiveTransfer{
		TransferID: transferID,
		Filename:   safeName,
		TempPath:   tempPath,
		FinalPath:  finalPath,
		File:       file,
		Size:       size,
		Received:   0,
	}

	m.transfers[transferID] = transfer

	return transfer, nil
}

func (m *TransferManager) Get(transferID string) (*ActiveTransfer, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	transfer, ok := m.transfers[transferID]
	return transfer, ok
}

func (m *TransferManager) Finish(transferID string) (*ActiveTransfer, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	transfer, ok := m.transfers[transferID]
	if !ok {
		return nil, false
	}

	delete(m.transfers, transferID)

	return transfer, true
}

func (m *TransferManager) Cancel(transferID string) {
	m.mu.Lock()
	transfer, ok := m.transfers[transferID]
	if ok {
		delete(m.transfers, transferID)
	}
	m.mu.Unlock()

	if !ok {
		return
	}

	transfer.mu.Lock()
	defer transfer.mu.Unlock()

	transfer.File.Close()
	os.Remove(transfer.TempPath)
}

func uniquePath(dir string, filename string) string {
	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]

	path := filepath.Join(dir, filename)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	for i := 1; ; i++ {
		candidate := filepath.Join(
			dir,
			base+"_"+strconv.Itoa(i)+ext,
		)

		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}