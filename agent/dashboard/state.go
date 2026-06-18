package dashboard

import (
	"path/filepath"
	"sync"
	"time"
)

type Transfer struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	Status      string `json:"status"`
	Received    int64  `json:"received"`
	Total       int64  `json:"total"`
	Speed       int64  `json:"speed"`
	Path        string `json:"path,omitempty"`
	StartedAt   int64  `json:"started_at"`
	UpdatedAt   int64  `json:"updated_at"`
	CompletedAt int64  `json:"completed_at,omitempty"`
	Error       string `json:"error,omitempty"`
}

type State struct {
	Service       string     `json:"service"`
	Status        string     `json:"status"`
	Address       string     `json:"address"`
	Token         string     `json:"token"`
	OutputDir     string     `json:"output_dir"`
	UptimeSeconds int64      `json:"uptime_seconds"`
	StartedAt     int64      `json:"started_at"`
	ReceivedCount int        `json:"received_count"`
	ActiveCount   int        `json:"active_count"`
	Transfers     []Transfer `json:"transfers"`
}

var (
	state = &State{
		Service:   "lanlink-agent",
		Status:    "ok",
		StartedAt: time.Now().Unix(),
	}
	mu sync.Mutex
)

func SetAddress(addr string) {
	mu.Lock()
	state.Address = addr
	mu.Unlock()
}

func SetToken(token string) {
	mu.Lock()
	state.Token = token
	mu.Unlock()
}

func GetState() State {
	mu.Lock()
	defer mu.Unlock()

	s := *state
	s.UptimeSeconds = time.Now().Unix() - s.StartedAt
	s.OutputDir = GetOutputDir()

	active := 0
	for _, t := range s.Transfers {
		if t.Status == "receiving" {
			active++
		}
	}
	s.ActiveCount = active

	return s
}

func AddTransfer(id, filename string, total int64) {
	mu.Lock()
	defer mu.Unlock()

	for i := range state.Transfers {
		if state.Transfers[i].ID == id {
			if total > 0 {
				state.Transfers[i].Total = total
			}
			state.Transfers[i].Filename = filename
			return
		}
	}

	now := time.Now().Unix()
	state.Transfers = append(state.Transfers, Transfer{
		ID:        id,
		Filename:  filename,
		Status:    "receiving",
		Received:  0,
		Total:     total,
		StartedAt: now,
		UpdatedAt: now,
	})
}

func UpdateTransfer(id string, received int64, speed int64) {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now().Unix()
	for i := range state.Transfers {
		if state.Transfers[i].ID == id {
			state.Transfers[i].Received = received
			state.Transfers[i].UpdatedAt = now

			if speed > 0 {
				state.Transfers[i].Speed = speed
			} else if received > 0 && state.Transfers[i].StartedAt > 0 {
				elapsed := float64(now - state.Transfers[i].StartedAt)
				if elapsed > 0 {
					state.Transfers[i].Speed = int64(float64(received) / elapsed)
				}
			}
			return
		}
	}
}

func CompleteTransfer(id, path string) {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now().Unix()
	for i := range state.Transfers {
		if state.Transfers[i].ID == id {
			t := &state.Transfers[i]
			t.Status = "saved"
			t.Path = path

			if t.Received > 0 && t.Total <= 0 {
				t.Total = t.Received
			} else if t.Total > 0 && t.Received <= 0 {
				t.Received = t.Total
			}

			if t.StartedAt > 0 {
				elapsed := float64(now - t.StartedAt)
				if elapsed > 0 && t.Received > 0 {
					t.Speed = int64(float64(t.Received) / elapsed)
				}
			}

			t.CompletedAt = now
			t.UpdatedAt = now
			state.ReceivedCount++
			return
		}
	}

	state.Transfers = append(state.Transfers, Transfer{
		ID:          id,
		Filename:    filepath.Base(path),
		Status:      "saved",
		Path:        path,
		Received:    0,
		StartedAt:   now,
		UpdatedAt:   now,
		CompletedAt: now,
	})
	state.ReceivedCount++
}

func FailTransfer(id, errMsg string) {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now().Unix()
	for i := range state.Transfers {
		if state.Transfers[i].ID == id {
			state.Transfers[i].Status = "failed"
			state.Transfers[i].Error = errMsg
			state.Transfers[i].CompletedAt = now
			state.Transfers[i].UpdatedAt = now
			return
		}
	}

	state.Transfers = append(state.Transfers, Transfer{
		ID:          id,
		Status:      "failed",
		Error:       errMsg,
		StartedAt:   now,
		UpdatedAt:   now,
		CompletedAt: now,
	})
}

func RemoveTransfer(id string) {
	mu.Lock()
	defer mu.Unlock()

	for i := range state.Transfers {
		if state.Transfers[i].ID == id {
			state.Transfers = append(state.Transfers[:i], state.Transfers[i+1:]...)
			return
		}
	}
}

func GetActiveTransfers() []Transfer {
	mu.Lock()
	defer mu.Unlock()

	var active []Transfer
	for _, t := range state.Transfers {
		if t.Status == "receiving" {
			active = append(active, t)
		}
	}
	return active
}

func GetRecentlyCompleted() []Transfer {
	mu.Lock()
	defer mu.Unlock()

	var completed []Transfer
	for _, t := range state.Transfers {
		if t.Status == "saved" && t.CompletedAt > 0 {
			completed = append(completed, t)
		}
	}
	return completed
}
