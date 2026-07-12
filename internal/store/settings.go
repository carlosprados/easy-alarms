package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings holds user preferences, persisted separately from the alarms so a
// schema change here never risks the alarm data.
type Settings struct {
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
	ShowSeconds  bool    `json:"show_seconds"`
	LastSoundDir string  `json:"last_sound_dir,omitempty"` // where the sound picker opens
}

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "easy-alarms", "settings.json"), nil
}

// LoadSettings reads settings.json, falling back to def for a missing,
// unreadable or corrupt file (and for any field the file omits).
func LoadSettings(def Settings) Settings {
	s := def
	path, err := settingsPath()
	if err != nil {
		return s
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

func SaveSettings(s Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
