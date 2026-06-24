package dashboard

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// browse.go implements the dashboard folder browser. These routes are reached
// only through subHandler, which already rejects non-loopback requests via
// IsLocalRequest, so filesystem browsing is never exposed to LAN clients.

type fsEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type fsListResponse struct {
	Path    string    `json:"path"`
	Parent  string    `json:"parent"`
	Entries []fsEntry `json:"entries"`
	Quick   []fsEntry `json:"quick"`
}

func handleFsList(w http.ResponseWriter, r *http.Request) {
	dir := strings.TrimSpace(r.URL.Query().Get("path"))
	if dir == "" {
		dir = defaultBrowseDir()
	}

	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	dir = filepath.Clean(dir)

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "not a directory",
		})
		return
	}

	read, err := os.ReadDir(dir)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"status": "error",
			"error":  "cannot read directory",
		})
		return
	}

	entries := make([]fsEntry, 0, len(read))
	for _, e := range read {
		if !e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		entries = append(entries, fsEntry{
			Name: e.Name(),
			Path: filepath.Join(dir, e.Name()),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	parent := filepath.Dir(dir)
	if parent == dir {
		parent = "" // already at the volume root
	}

	writeJSON(w, http.StatusOK, fsListResponse{
		Path:    dir,
		Parent:  parent,
		Entries: entries,
		Quick:   quickLocations(),
	})
}

func handleFsMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "invalid request"})
		return
	}

	req.Path = strings.TrimSpace(req.Path)
	req.Name = strings.TrimSpace(req.Name)
	if req.Path == "" || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "missing path or name"})
		return
	}
	// Reject anything that could escape the chosen directory.
	if strings.ContainsAny(req.Name, `/\`) || req.Name == "." || req.Name == ".." {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "invalid folder name"})
		return
	}

	target := filepath.Join(req.Path, req.Name)
	if err := os.MkdirAll(target, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "path": target})
}

// defaultBrowseDir picks a sensible starting directory: the current output
// folder if it resolves to an absolute path, otherwise the user's home.
func defaultBrowseDir() string {
	if cur := GetOutputDir(); cur != "" {
		if abs, err := filepath.Abs(cur); err == nil {
			if info, err := os.Stat(abs); err == nil && info.IsDir() {
				return abs
			}
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return string(filepath.Separator)
}

func quickLocations() []fsEntry {
	var quick []fsEntry
	seen := map[string]bool{}
	add := func(name, path string) {
		if path == "" || seen[path] {
			return
		}
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			return
		}
		seen[path] = true
		quick = append(quick, fsEntry{Name: name, Path: path})
	}

	if home, err := os.UserHomeDir(); err == nil {
		add("Home", home)
		add("Downloads", filepath.Join(home, "Downloads"))
		add("Documents", filepath.Join(home, "Documents"))
		add("Desktop", filepath.Join(home, "Desktop"))
	}
	if wd, err := os.Getwd(); err == nil {
		add("Working dir", wd)
	}
	return quick
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
