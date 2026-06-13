package transfer

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

var ErrDuplicateTransfer = errors.New("duplicate transfer id")
var ErrInvalidChunk = errors.New("invalid chunk")
var ErrDuplicateChunk = errors.New("duplicate chunk")
var ErrSizeMismatch = errors.New("file size mismatch")

type ActiveTransfer struct {
	TransferID string
	Filename   string
	TempPath   string
	FinalPath  string
	File       *os.File
	Size       int64

	ReceivedBytes int64
	Chunks        map[int]struct{}

	mu sync.Mutex
}

type Manager struct {
	mu        sync.Mutex
	transfers map[string]*ActiveTransfer
}

func NewManager() *Manager {
	return &Manager{
		transfers: make(map[string]*ActiveTransfer),
	}
}

func (m *Manager) Start(
	transferID string,
	filename string,
	size int64,
) (*ActiveTransfer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.transfers[transferID]; exists {
		return nil, ErrDuplicateTransfer
	}

	safeName := filepath.Base(filename)

	if err := os.MkdirAll("received/tmp", 0755); err != nil {
		return nil, err
	}

	if err := os.MkdirAll("received", 0755); err != nil {
		return nil, err
	}

	finalPath := uniquePath("received", safeName)
	tempPath := filepath.Join("received", "tmp", transferID+"_"+safeName)

	file, err := os.OpenFile(
		tempPath,
		os.O_CREATE|os.O_RDWR|os.O_TRUNC,
		0644,
	)
	if err != nil {
		return nil, err
	}

	if size > 0 {
		if err := file.Truncate(size); err != nil {
			file.Close()
			os.Remove(tempPath)
			return nil, err
		}
	}

	active := &ActiveTransfer{
		TransferID:    transferID,
		Filename:      safeName,
		TempPath:      tempPath,
		FinalPath:     finalPath,
		File:          file,
		Size:          size,
		ReceivedBytes: 0,
		Chunks:        make(map[int]struct{}),
	}

	m.transfers[transferID] = active

	return active, nil
}

func (m *Manager) Get(transferID string) (*ActiveTransfer, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	active, ok := m.transfers[transferID]
	return active, ok
}

func (m *Manager) Finish(transferID string) (*ActiveTransfer, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	active, ok := m.transfers[transferID]
	if !ok {
		return nil, false
	}

	delete(m.transfers, transferID)

	return active, true
}

func (m *Manager) Cancel(transferID string) {
	m.mu.Lock()
	active, ok := m.transfers[transferID]
	if ok {
		delete(m.transfers, transferID)
	}
	m.mu.Unlock()

	if !ok {
		return
	}

	active.mu.Lock()
	defer active.mu.Unlock()

	active.File.Close()
	os.Remove(active.TempPath)
}

func (t *ActiveTransfer) WriteChunk(
	index int,
	offset int64,
	data []byte,
) (int64, error) {
	if index < 0 || offset < 0 || len(data) == 0 {
		return 0, ErrInvalidChunk
	}

	if t.Size >= 0 && offset+int64(len(data)) > t.Size {
		return 0, ErrInvalidChunk
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.Chunks[index]; exists {
		return t.ReceivedBytes, ErrDuplicateChunk
	}

	n, err := t.File.WriteAt(data, offset)
	if err != nil {
		return t.ReceivedBytes, err
	}

	if n != len(data) {
		return t.ReceivedBytes, ErrInvalidChunk
	}

	t.Chunks[index] = struct{}{}
	t.ReceivedBytes += int64(n)

	return t.ReceivedBytes, nil
}

func (t *ActiveTransfer) Finalize() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.Size != t.ReceivedBytes {
		t.File.Close()
		os.Remove(t.TempPath)
		return ErrSizeMismatch
	}

	if err := t.File.Sync(); err != nil {
		t.File.Close()
		os.Remove(t.TempPath)
		return err
	}

	if err := t.File.Close(); err != nil {
		os.Remove(t.TempPath)
		return err
	}

	if err := os.Rename(t.TempPath, t.FinalPath); err != nil {
		os.Remove(t.TempPath)
		return err
	}

	return nil
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
