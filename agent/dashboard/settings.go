package dashboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Settings struct {
	ReceivedDir string `json:"received_dir"`
}

var (
	currentDir string
	defaultDir string
	settingsMu sync.RWMutex
)

func InitSettings(defaultOutputDir string) {
	defaultDir = defaultOutputDir
	currentDir = defaultOutputDir

	if envDir := os.Getenv("LANLINK_RECEIVED_DIR"); envDir != "" {
		if info, err := os.Stat(envDir); err == nil && info.IsDir() {
			currentDir = envDir
		}
	}

	if saved := loadSettingsFile(); saved != "" {
		if info, err := os.Stat(saved); err == nil && info.IsDir() {
			currentDir = saved
		}
	}
}

func GetOutputDir() string {
	settingsMu.RLock()
	defer settingsMu.RUnlock()
	return currentDir
}

func SetOutputDir(dir string) error {
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	settingsMu.Lock()
	currentDir = dir
	settingsMu.Unlock()

	saveSettingsFile(Settings{ReceivedDir: dir})
	return nil
}

func ResetOutputDir() {
	settingsMu.Lock()
	currentDir = defaultDir
	settingsMu.Unlock()

	os.Remove(settingsPath())
}

func settingsPath() string {
	return filepath.Join("data", "settings.json")
}

func loadSettingsFile() string {
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return ""
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return ""
	}
	return s.ReceivedDir
}

func saveSettingsFile(s Settings) {
	os.MkdirAll("data", 0755)
	data, _ := json.MarshalIndent(s, "", "  ")
	os.WriteFile(settingsPath(), data, 0644)
}

func CurrentSettings() Settings {
	settingsMu.RLock()
	defer settingsMu.RUnlock()
	return Settings{ReceivedDir: currentDir}
}
