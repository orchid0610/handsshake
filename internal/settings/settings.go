// Package settings stores user preferences. They hold no secrets, so they
// live in a plain JSON file next to the vault and are readable while it is locked.
package settings

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"slices"
	"strings"
	"sync"

	"handsshake/internal/fileutil"
)

type Settings struct {
	Theme           string  `json:"theme"`
	FontFamily      string  `json:"fontFamily"`  // "jetbrains-mono" | "system"
	FontSize        float64 `json:"fontSize"`    // px
	LineHeight      float64 `json:"lineHeight"`  // multiplier
	CursorStyle     string  `json:"cursorStyle"` // "block" | "bar" | "underline"
	CursorBlink     bool    `json:"cursorBlink"`
	Scrollback      int     `json:"scrollback"`      // lines
	AutoLockMinutes int     `json:"autoLockMinutes"` // 0 = never
}

func Default() Settings {
	return Settings{
		Theme:           "tokyo-night",
		FontFamily:      "jetbrains-mono",
		FontSize:        13.5,
		LineHeight:      1.2,
		CursorStyle:     "block",
		CursorBlink:     true,
		Scrollback:      5000,
		AutoLockMinutes: 0,
	}
}

var (
	fontFamilies = []string{"jetbrains-mono", "system"}
	cursorStyles = []string{"block", "bar", "underline"}
)

// Sanitize forces every field into a sane range. Settings arrive from the UI
// and from a file anyone can hand-edit, so we never trust them as they are.
func (s Settings) Sanitize() Settings {
	d := Default()
	if t := strings.TrimSpace(s.Theme); t == "" || len(t) > 40 {
		s.Theme = d.Theme
	} else {
		s.Theme = t
	}
	if !slices.Contains(fontFamilies, s.FontFamily) {
		s.FontFamily = d.FontFamily
	}
	if !slices.Contains(cursorStyles, s.CursorStyle) {
		s.CursorStyle = d.CursorStyle
	}
	s.FontSize = clamp(s.FontSize, 9, 28, d.FontSize)
	s.LineHeight = clamp(s.LineHeight, 1.0, 2.0, d.LineHeight)
	s.Scrollback = min(100000, max(500, s.Scrollback))
	s.AutoLockMinutes = min(1440, max(0, s.AutoLockMinutes))
	return s
}

func clamp(v, lo, hi, def float64) float64 {
	if math.IsNaN(v) {
		return def
	}
	return math.Min(hi, math.Max(lo, v))
}

// Store keeps the current settings in memory and on disk.
type Store struct {
	mu   sync.Mutex
	path string
	cur  Settings
}

// Open loads the settings file. A missing file means defaults; an unreadable
// one also falls back to defaults, because settings aren't worth refusing to
// start over (the next save replaces it).
func Open(path string) (*Store, error) {
	s := &Store{path: path, cur: Default()}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	loaded := Default() // fields missing from the file keep their defaults
	if err := json.Unmarshal(data, &loaded); err != nil {
		return s, nil
	}
	s.cur = loaded.Sanitize()
	return s, nil
}

func (s *Store) Get() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cur
}

// Set sanitizes, saves and returns the values actually kept.
func (s *Store) Set(next Settings) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clean := next.Sanitize()
	data, err := json.MarshalIndent(clean, "", "  ")
	if err != nil {
		return s.cur, err
	}
	if err := fileutil.WriteAtomic(s.path, append(data, '\n'), 0o600); err != nil {
		return s.cur, err
	}
	s.cur = clean
	return clean, nil
}