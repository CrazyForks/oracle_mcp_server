package reviewwhitelist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Entry stores one approved SQL header for a connection.
type Entry struct {
	Connection string `json:"connection"`
	HeaderLine string `json:"header_line"`
}

// Store manages whitelist.json persistence.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore creates a whitelist store for the given file path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// DefaultPath returns the whitelist.json path in the executable directory.
func DefaultPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(exePath), "whitelist.json"), nil
}

// Contains reports whether the exact connection + header_line pair already exists.
func (s *Store) Contains(connection, headerLine string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.loadLocked()
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.Connection == connection && entry.HeaderLine == headerLine {
			return true, nil
		}
	}
	return false, nil
}

// Add inserts the exact connection + header_line pair if it does not already exist.
func (s *Store) Add(connection, headerLine string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := s.loadLocked()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Connection == connection && entry.HeaderLine == headerLine {
			return nil
		}
	}
	entries = append(entries, Entry{
		Connection: connection,
		HeaderLine: headerLine,
	})
	return s.saveLocked(entries)
}

func (s *Store) loadLocked() ([]Entry, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("read whitelist file %s: %w", s.path, err)
	}
	if len(data) == 0 {
		return []Entry{}, nil
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse whitelist file %s: %w", s.path, err)
	}
	return entries, nil
}

func (s *Store) saveLocked(entries []Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal whitelist: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("write whitelist file %s: %w", s.path, err)
	}
	return nil
}
