package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sshgui/internal/sshconn"
	"sshgui/internal/vault"
)

type session struct {
	conn  *sshconn.Conn
	shell *sshconn.Shell
}

// Connect opens a terminal for a saved host. The UI chooses the id so it can
// start listening for output events BEFORE the first bytes arrive. If the
// server is unknown, this call waits while the UI asks the user to verify
// the fingerprint.
func (a *App) Connect(id, hostID string, cols, rows int) error {
	if cols < 1 || rows < 1 {
		cols, rows = 80, 24
	}
	h, err := a.vault.Get(hostID)
	if err != nil {
		return err
	}

	var creds sshconn.Credentials
	switch h.Auth {
	case vault.AuthKey:
		pem := []byte(h.PrivateKey)
		if len(pem) == 0 {
			if pem, err = os.ReadFile(expandHome(h.KeyPath)); err != nil {
				return fmt.Errorf("cannot read the key file: %w", err)
			}
		}
		creds = sshconn.Credentials{PrivateKey: pem, Passphrase: h.Passphrase}
	default:
		creds = sshconn.Credentials{Password: h.Password}
	}

	conn, err := sshconn.Dial(sshconn.Config{
		Address: h.Address,
		User:    h.User,
		Creds:   creds,
		HostKey: a.known.Callback(a.askTrust),
	})
	if err != nil {
		return err
	}
	shell, err := conn.OpenShell(cols, rows)
	if err != nil {
		conn.Close()
		return err
	}

	a.mu.Lock()
	a.sessions[id] = &session{conn: conn, shell: shell}
	a.mu.Unlock()

	go a.pump(id, shell)
	return nil
}

// pump forwards everything the server prints to the UI as events.
func (a *App) pump(id string, shell *sshconn.Shell) {
	buf := make([]byte, 32*1024)
	for {
		n, err := shell.Read(buf)
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

func (a *App) get(id string) *session {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sessions[id]
}

func (a *App) Write(id, data string) error {
	s := a.get(id)
	if s == nil {
		return fmt.Errorf("no session %s", id)
	}
	_, err := s.shell.Write([]byte(data))
	return err
}

func (a *App) Resize(id string, cols, rows int) {
	if s := a.get(id); s != nil && cols > 0 && rows > 0 {
		_ = s.shell.Resize(cols, rows)
	}
}

// Disconnect closes a terminal. Calling it for an unknown id is harmless.
func (a *App) Disconnect(id string) {
	a.mu.Lock()
	s := a.sessions[id]
	delete(a.sessions, id)
	a.mu.Unlock()
	if s != nil {
		s.shell.Close()
		s.conn.Close()
	}
}

// ---- Host key trust prompt ----

// askTrust runs on the connecting goroutine. It tells the UI to show a
// dialog, then waits for AnswerHostKey (or gives up).
func (a *App) askTrust(info sshconn.HostKeyInfo) bool {
	promptID := randomID()
	answer := make(chan bool, 1)

	a.mu.Lock()
	a.prompts[promptID] = answer
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.prompts, promptID)
		a.mu.Unlock()
	}()

	runtime.EventsEmit(a.ctx, "hostkey:prompt", map[string]string{
		"promptId":    promptID,
		"host":        info.Host,
		"keyType":     info.KeyType,
		"fingerprint": info.Fingerprint,
	})

	select {
	case trusted := <-answer:
		return trusted
	case <-time.After(2 * time.Minute):
		return false
	case <-a.ctx.Done():
		return false
	}
}

// AnswerHostKey is called by the UI when the user clicks Trust or Cancel.
func (a *App) AnswerHostKey(promptID string, trust bool) {
	a.mu.Lock()
	answer := a.prompts[promptID]
	a.mu.Unlock()
	if answer != nil {
		select {
		case answer <- trust:
		default:
		}
	}
}