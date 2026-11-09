package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Host is one saved connection.
type Host struct {
	Name    string `json:"name"`
	Address string `json:"address"` // host:port
	User    string `json:"user"`
	KeyPath string `json:"keyPath"`
}

// Store holds all saved hosts and knows where they live on disk.
type Store struct {
	path  string
	Hosts []Host `json:"hosts"`
}

// OpenStore loads hosts.json, or starts empty if it doesn't exist yet.
func OpenStore() (*Store, error) {
	dir, err := os.UserConfigDir() // ~/.config on Linux
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "sshclient")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	s := &Store{path: filepath.Join(dir, "hosts.json")}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", s.path, err)
	}
	return s, nil
}

func (s *Store) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *Store) Find(name string) (*Host, bool) {
	for i := range s.Hosts {
		if s.Hosts[i].Name == name {
			return &s.Hosts[i], true
		}
	}
	return nil, false
}

func (s *Store) Add(h Host) error {
	if _, exists := s.Find(h.Name); exists {
		return fmt.Errorf("host %q already exists", h.Name)
	}
	s.Hosts = append(s.Hosts, h)
	return s.Save()
}

func (s *Store) Remove(name string) error {
	for i, h := range s.Hosts {
		if h.Name == name {
			s.Hosts = append(s.Hosts[:i], s.Hosts[i+1:]...)
			return s.Save()
		}
	}
	return fmt.Errorf("no host named %q", name)
}