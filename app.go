package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/ssh"
)

// liveSession is one open terminal.
type liveSession struct {
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
}

type App struct {
	ctx      context.Context
	store    *Store
	mu       sync.Mutex // protects sessions
	sessions map[string]*liveSession
}

func NewApp() *App {
	return &App{sessions: map[string]*liveSession{}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	store, err := OpenStore()
	if err != nil {
		log.Fatalf("opening store: %v", err)
	}
	a.store = store
}

// ---- Hosts (called from the frontend) ----

func (a *App) ListHosts() []Host {
	hosts := make([]Host, len(a.store.Hosts))
	copy(hosts, a.store.Hosts)
	return hosts
}

func (a *App) AddHost(name, address, user, keyPath string) error {
	return a.store.Add(Host{Name: name, Address: address, User: user, KeyPath: keyPath})
}

func (a *App) RemoveHost(name string) error {
	return a.store.Remove(name)
}

// ---- Terminal sessions ----

// Connect opens a shell. The frontend picks the id so it can
// subscribe to output events BEFORE the first bytes arrive.
func (a *App) Connect(id, hostName string, cols, rows int) error {
	h, ok := a.store.Find(hostName)
	if !ok {
		return fmt.Errorf("no host named %q", hostName)
	}
	client, err := connect(h.Address, h.User, h.KeyPath)
	if err != nil {
		return err
	}
	session, stdin, stdout, err := startShell(client, cols, rows)
	if err != nil {
		client.Close()
		return err
	}

	a.mu.Lock()
	a.sessions[id] = &liveSession{client: client, session: session, stdin: stdin}
	a.mu.Unlock()

	go a.pump(id, stdout)
	return nil
}

// pump reads server output and pushes it to the frontend as events.
func (a *App) pump(id string, stdout io.Reader) {
	buf := make([]byte, 32*1024)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			// base64 keeps raw bytes intact across the JS bridge
			runtime.EventsEmit(a.ctx, "term:data:"+id, base64.StdEncoding.EncodeToString(buf[:n]))
		}
		if err != nil {
			break
		}
	}
	a.Disconnect(id)
	runtime.EventsEmit(a.ctx, "term:closed:"+id)
}

func (a *App) get(id string) *liveSession {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sessions[id]
}

func (a *App) Write(id, data string) error {
	s := a.get(id)
	if s == nil {
		return fmt.Errorf("no session %s", id)
	}
	_, err := s.stdin.Write([]byte(data))
	return err
}

func (a *App) Resize(id string, cols, rows int) {
	if s := a.get(id); s != nil {
		s.session.WindowChange(rows, cols)
	}
}

func (a *App) Disconnect(id string) {
	a.mu.Lock()
	s := a.sessions[id]
	delete(a.sessions, id)
	a.mu.Unlock()
	if s != nil {
		s.session.Close()
		s.client.Close()
	}
}