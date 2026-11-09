// Package app is the thin layer the UI talks to. Every exported method on
// App becomes a JavaScript function in the frontend.
package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"sync"

	"sshgui/internal/sshconn"
	"sshgui/internal/vault"
)

type App struct {
	ctx   context.Context
	vault *vault.Vault
	known *sshconn.KnownHosts

	mu       sync.Mutex // protects the two maps below
	sessions map[string]*session
	prompts  map[string]chan bool // pending "trust this host?" questions
}

func New() (*App, error) {
	vp, err := vault.DefaultPath()
	if err != nil {
		return nil, err
	}
	return &App{
		vault:    vault.New(vp),
		known:    sshconn.NewKnownHosts(filepath.Join(filepath.Dir(vp), "known_hosts")),
		sessions: map[string]*session{},
		prompts:  map[string]chan bool{},
	}, nil
}

// startup and shutdown are lowercase on purpose: Wails would expose
// exported methods to JavaScript, and the UI has no business calling these.
func (a *App) startup(ctx context.Context) { a.ctx = ctx }

func (a *App) shutdown(_ context.Context) {
	a.mu.Lock()
	ids := make([]string, 0, len(a.sessions))
	for id := range a.sessions {
		ids = append(ids, id)
	}
	a.mu.Unlock()

	for _, id := range ids {
		a.Disconnect(id)
	}
	a.vault.Lock()
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}