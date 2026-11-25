package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"handsshake/internal/sshconn"
	"handsshake/internal/vault"
)

type session struct {
	conn  *sshconn.Conn
	shell *sshconn.Shell
}

// ConnectInfo is what the UI learns when a session opens.
type ConnectInfo struct {
	HostKeyType string `json:"hostKeyType"` // e.g. "ssh-ed25519"
}

// Connect opens a terminal for a saved host. The UI chooses the id so it can
// start listening for output events BEFORE the first bytes arrive. If the
// server is unknown, this call waits while the UI asks the user to verify
// the fingerprint.
// Connect opens a terminal for a saved host, going through its jump hosts
// first if it has any. The UI chooses the id so it can start listening for
// output events BEFORE the first bytes arrive. If a server is unknown, this
// call waits while the UI asks the user to verify the fingerprint.
func (a *App) Connect(id, hostID string, cols, rows int) (ConnectInfo, error) {
	if cols < 1 || rows < 1 {
		cols, rows = 80, 24
	}
	h, err := a.vault.Get(hostID)
	if err != nil {
		return ConnectInfo{}, err
	}
	hosts, err := a.vault.Hosts()
	if err != nil {
		return ConnectInfo{}, err
	}
	chain, err := vault.JumpChain(hosts, h) // jump hosts first, the host itself last
	if err != nil {
		return ConnectInfo{}, err
	}

	conn, err := a.dialChain(chain)
	if err != nil {
		return ConnectInfo{}, err
	}
	shell, err := conn.OpenShell(cols, rows)
	if err != nil {
		conn.Close()
		return ConnectInfo{}, err
	}

	a.mu.Lock()
	a.sessions[id] = &session{conn: conn, shell: shell}
	a.mu.Unlock()

	go a.pump(id, shell)
	_ = a.vault.Update(hostID, func(h *vault.Host) { h.LastUsed = time.Now().Unix() })
	go a.detectOS(id, hostID, conn)
	return ConnectInfo{HostKeyType: conn.HostKeyType()}, nil
}

// Latency measures one round trip to the server, in milliseconds.
func (a *App) Latency(id string) (int, error) {
	s := a.get(id)
	if s == nil {
		return 0, fmt.Errorf("no session %s", id)
	}
	d, err := s.conn.Ping()
	if err != nil {
		return 0, err
	}
	return max(1, int(d.Milliseconds())), nil // sub-millisecond still shows as 1 ms
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

// detectOS finds out what the server runs, tells the UI, and remembers it
// on the host so the sidebar can show the logo next time.
// detectOS finds out what the server runs, tells the UI, and remembers it
// on the host so the UI can show the logo and name before the next connect.
func (a *App) detectOS(id, hostID string, conn *sshconn.Conn) {
	info, err := conn.DetectOS()
	if err != nil || (info.ID == "" && info.Name == "") {
		return // restricted shells, routers and the like just keep a generic icon
	}
	runtime.EventsEmit(a.ctx, "term:os:"+id, info)

	if h, err := a.vault.Get(hostID); err == nil &&
		h.OS == info.ID && h.OSLike == info.Like && h.OSName == info.Name {
		return // nothing changed, so don't rewrite the vault
	}
	_ = a.vault.Update(hostID, func(h *vault.Host) {
		h.OS, h.OSLike, h.OSName = info.ID, info.Like, info.Name
	})
}

// credentialsFor turns a saved host's login details into what sshconn needs.
func credentialsFor(h vault.Host) (sshconn.Credentials, error) {
	switch h.Auth {
	case vault.AuthKey:
		pem := []byte(h.PrivateKey)
		if len(pem) == 0 {
			var err error
			if pem, err = os.ReadFile(expandHome(h.KeyPath)); err != nil {
				return sshconn.Credentials{}, fmt.Errorf("cannot read the key file: %w", err)
			}
		}
		return sshconn.Credentials{PrivateKey: pem, Passphrase: h.Passphrase}, nil
	default:
		return sshconn.Credentials{Password: h.Password}, nil
	}
}

// dialChain connects through every host in the chain, in order. Each hop signs
// in with its own credentials, and each host key is checked (and, if new,
// confirmed by the user) separately.
func (a *App) dialChain(chain []vault.Host) (*sshconn.Conn, error) {
	var via *sshconn.Conn
	for i, h := range chain {
		creds, err := credentialsFor(h)
		if err != nil {
			if via != nil {
				via.Close()
			}
			return nil, hopError(chain, i, err)
		}
		conn, err := sshconn.Dial(sshconn.Config{
			Address: h.Address,
			User:    h.User,
			Creds:   creds,
			HostKey: a.known.Callback(a.askTrust),
			Via:     via, // Dial takes ownership of it, even when it fails
		})
		if err != nil {
			return nil, hopError(chain, i, err)
		}
		via = conn
	}
	return via, nil
}

// hopError says which host in a chain failed. A direct connection keeps its plain error.
func hopError(chain []vault.Host, i int, err error) error {
	switch {
	case len(chain) == 1:
		return err
	case i < len(chain)-1:
		return fmt.Errorf("jump host “%s”: %w", chain[i].Name, err)
	default:
		return fmt.Errorf("“%s” (through “%s”): %w", chain[i].Name, chain[i-1].Name, err)
	}
}
