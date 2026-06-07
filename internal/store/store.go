package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"easy-alarms/internal/alarm"
)

// Store keeps alarms in memory and persists them as JSON under
// ~/.config/easy-alarms/alarms.json.
type Store struct {
	mu     sync.Mutex
	path   string
	alarms []*alarm.Alarm
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "easy-alarms", "alarms.json"), nil
}

func Load() (*Store, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	s := &Store{path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	// An empty file is a fresh, never-saved store — not an error.
	if len(bytes.TrimSpace(data)) == 0 {
		return s, nil
	}
	// A corrupt file must not stop an alarm clock from starting: back it up
	// out of the way and start fresh rather than crash on launch.
	if err := json.Unmarshal(data, &s.alarms); err != nil {
		_ = os.Rename(path, path+".corrupt")
		s.alarms = nil
		return s, nil
	}
	return s, nil
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(s.alarms, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *Store) Add(a *alarm.Alarm) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alarms = append(s.alarms, a)
}

func (s *Store) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.alarms {
		if a.ID == id {
			s.alarms = append(s.alarms[:i], s.alarms[i+1:]...)
			return
		}
	}
}

// List returns a snapshot of the alarm slice. The pointed-to alarms are
// shared; this app mutates them only from the Fyne main thread.
func (s *Store) List() []*alarm.Alarm {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*alarm.Alarm, len(s.alarms))
	copy(out, s.alarms)
	return out
}
