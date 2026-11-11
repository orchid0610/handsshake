// Package hostimport turns an OpenSSH client config into HandSSHake hosts.
package hostimport

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"

	"handsshake/internal/sshconfig"
	"handsshake/internal/vault"
)

// Candidate status: what importing this entry would do.
const (
	StatusNew         = "new"
	StatusExists      = "exists"
	StatusUnsupported = "unsupported"
)

// Key status: what we found at the host's IdentityFile.
const (
	KeyNone      = "none"      // no key configured or found: the host will need a password
	KeyOK        = "ok"
	KeyEncrypted = "encrypted" // protected by a passphrase
	KeyMissing   = "missing"
	KeyInvalid   = "invalid"
)

type Candidate struct {
	Alias     string `json:"alias"`
	Address   string `json:"address"` // host:port
	User      string `json:"user"`
	KeyPath   string `json:"keyPath"`
	KeyStatus string `json:"keyStatus"`
	Status    string `json:"status"`
	Note      string `json:"note"`
}

type Preview struct {
	Path       string      `json:"path"`
	Candidates []Candidate `json:"candidates"`
	Warnings   []string    `json:"warnings"`
}

// DefaultPath is ~/.ssh/config.
func DefaultPath(home string) string { return filepath.Join(home, ".ssh", "config") }

// Build parses the config (the default one if path is empty) and describes
// every host in it. The returned map holds the vault hosts that importing
// would add, keyed by alias; only entries with StatusNew are in it.
func Build(path string, env sshconfig.Env, existing []vault.Host) (Preview, map[string]vault.Host, error) {
	if path == "" {
		path = DefaultPath(env.Home)
	}
	res, err := sshconfig.Load(path, env)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Preview{}, nil, fmt.Errorf("no SSH config file found at %s", path)
		}
		return Preview{}, nil, err
	}

	pv := Preview{Path: path, Candidates: []Candidate{}, Warnings: append([]string{}, res.Warnings...)}
	hosts := map[string]vault.Host{}
	for _, e := range res.Entries {
		c, h := describe(e, env, existing)
		pv.Candidates = append(pv.Candidates, c)
		if c.Status == StatusNew {
			hosts[e.Alias] = h
		}
	}
	return pv, hosts, nil
}

func describe(e sshconfig.Entry, env sshconfig.Env, existing []vault.Host) (Candidate, vault.Host) {
	addr := net.JoinHostPort(e.HostName, strconv.Itoa(e.Port))
	keyPath, keyStatus := pickKey(e.IdentityFiles, env.Home)

	h := vault.Host{Name: e.Alias, Address: addr, User: e.User, Auth: vault.AuthPassword}
	if keyPath != "" {
		h.Auth, h.KeyPath = vault.AuthKey, keyPath
	}
	c := Candidate{Alias: e.Alias, Address: addr, User: e.User,
		KeyPath: keyPath, KeyStatus: keyStatus, Status: StatusNew}

	switch {
	case e.Problem != "":
		c.Status, c.Note = StatusUnsupported, e.Problem
	case e.ProxyJump != "":
		c.Status, c.Note = StatusUnsupported, "Goes through a jump host (ProxyJump), which isn't supported yet."
	case e.ProxyCommand != "":
		c.Status, c.Note = StatusUnsupported, "Uses a ProxyCommand, which isn't supported yet."
	case alreadyThere(existing, h):
		c.Status, c.Note = StatusExists, "Already in your vault."
	}
	return c, h
}

func alreadyThere(existing []vault.Host, h vault.Host) bool {
	for _, x := range existing {
		if strings.EqualFold(x.Name, h.Name) {
			return true
		}
		if x.User == h.User && strings.EqualFold(normAddr(x.Address), normAddr(h.Address)) {
			return true
		}
	}
	return false
}

// normAddr adds the default port, because "host" and "host:22" are the same server.
func normAddr(a string) string {
	if _, _, err := net.SplitHostPort(a); err != nil {
		return net.JoinHostPort(strings.Trim(a, "[]"), "22")
	}
	return a
}

// pickKey chooses which key file the host should use. Like ssh, it falls
// back to the default key names when the config names none.
func pickKey(configured []string, home string) (path, status string) {
	if len(configured) == 0 {
		for _, name := range []string{"id_ed25519", "id_ecdsa", "id_rsa"} {
			p := filepath.Join(home, ".ssh", name)
			if st := inspect(p); st != KeyMissing {
				return p, st
			}
		}
		return "", KeyNone
	}
	for _, p := range configured {
		if st := inspect(p); st != KeyMissing {
			return p, st
		}
	}
	return configured[0], KeyMissing
}

// inspect checks a key file without ever needing its passphrase.
func inspect(path string) string {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return KeyMissing
	}
	if info.Size() > 1<<20 {
		return KeyInvalid
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return KeyMissing
	}
	_, err = ssh.ParsePrivateKey(b)
	if err == nil {
		return KeyOK
	}
	var needPass *ssh.PassphraseMissingError
	if errors.As(err, &needPass) {
		return KeyEncrypted
	}
	return KeyInvalid
}

// EmbedKey copies the key file's contents into the host, so the key lives
// inside the encrypted vault and no longer depends on the file.
func EmbedKey(h vault.Host) (vault.Host, error) {
	if h.KeyPath == "" {
		return h, nil
	}
	b, err := os.ReadFile(h.KeyPath)
	if err != nil {
		return h, err
	}
	if len(b) > 1<<20 || !bytes.Contains(b, []byte("PRIVATE KEY")) {
		return h, errors.New("that file doesn't look like a private key")
	}
	h.PrivateKey, h.KeyPath = string(b), ""
	return h, nil
}