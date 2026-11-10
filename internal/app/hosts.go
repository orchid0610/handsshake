package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"handsshake/internal/vault"
)

// HostInfo is what the UI sees: everything EXCEPT the secrets, plus flags
// saying which secrets exist.
type HostInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	User          string `json:"user"`
	Auth          string `json:"auth"` // "password" or "key"
	KeyPath       string `json:"keyPath"`
	HasPassword   bool   `json:"hasPassword"`
	KeyInVault    bool   `json:"keyInVault"`
	HasPassphrase bool   `json:"hasPassphrase"`
	OS            string `json:"os"`
	OSLike        string `json:"osLike"`
}

// HostInput is what the UI sends when saving. On edit, an empty secret
// field means "keep the current one".
type HostInput struct {
	ID              string `json:"id"` // empty = new host
	Name            string `json:"name"`
	Address         string `json:"address"`
	User            string `json:"user"`
	Auth            string `json:"auth"`
	Password        string `json:"password"`
	KeyPath         string `json:"keyPath"`
	ImportKey       bool   `json:"importKey"` // copy the key file's contents into the vault
	Passphrase      string `json:"passphrase"`
	ClearPassphrase bool   `json:"clearPassphrase"`
}

func toInfo(h vault.Host) HostInfo {
	return HostInfo{
		ID: h.ID, Name: h.Name, Address: h.Address, User: h.User,
		Auth: string(h.Auth), KeyPath: h.KeyPath,
		HasPassword: h.Password != "", KeyInVault: h.PrivateKey != "",
		HasPassphrase: h.Passphrase != "",
		OS:            h.OS, OSLike: h.OSLike,
	}
}

func (a *App) ListHosts() ([]HostInfo, error) {
	hosts, err := a.vault.Hosts()
	if err != nil {
		return nil, err
	}
	sort.Slice(hosts, func(i, j int) bool {
		return strings.ToLower(hosts[i].Name) < strings.ToLower(hosts[j].Name)
	})
	out := make([]HostInfo, 0, len(hosts)) // empty slice, not nil: JSON "[]", not "null"
	for _, h := range hosts {
		out = append(out, toInfo(h))
	}
	return out, nil
}

func (a *App) GetHost(id string) (HostInfo, error) {
	h, err := a.vault.Get(id)
	if err != nil {
		return HostInfo{}, err
	}
	return toInfo(h), nil
}

func (a *App) SaveHost(in HostInput) (string, error) {
	h := vault.Host{
		ID:      in.ID,
		Name:    strings.TrimSpace(in.Name),
		Address: strings.TrimSpace(in.Address),
		User:    strings.TrimSpace(in.User),
		Auth:    vault.AuthType(in.Auth),
	}
	switch {
	case h.Name == "":
		return "", errors.New("give the host a name")
	case h.Address == "":
		return "", errors.New("enter the server address")
	case h.User == "":
		return "", errors.New("enter a username")
	}

	editing := in.ID != ""
	var old vault.Host
	if editing {
		var err error
		if old, err = a.vault.Get(in.ID); err != nil {
			return "", err
		}
	}

	if editing {
		h.OS, h.OSLike = old.OS, old.OSLike
	}

	switch h.Auth {
	case vault.AuthPassword:
		h.Password = in.Password
		if h.Password == "" && editing && old.Auth == vault.AuthPassword {
			h.Password = old.Password
		}
		if h.Password == "" {
			return "", errors.New("enter a password")
		}

	case vault.AuthKey:
		keepOld := editing && old.Auth == vault.AuthKey
		path := strings.TrimSpace(in.KeyPath)
		switch {
		case path != "" && in.ImportKey:
			pem, err := readKeyFile(path)
			if err != nil {
				return "", err
			}
			h.PrivateKey = string(pem)
		case path != "":
			if _, err := readKeyFile(path); err != nil {
				return "", err
			}
			h.KeyPath = path
		case keepOld:
			h.KeyPath, h.PrivateKey = old.KeyPath, old.PrivateKey
		default:
			return "", errors.New("choose a private key file")
		}
		switch {
		case in.ClearPassphrase:
			// leave h.Passphrase empty
		case in.Passphrase != "":
			h.Passphrase = in.Passphrase
		case keepOld:
			h.Passphrase = old.Passphrase
		}

	default:
		return "", errors.New("unknown authentication type")
	}
	return a.vault.Put(h)
}

func (a *App) DeleteHost(id string) error { return a.vault.Delete(id) }

// PickKeyFile opens a native file dialog in ~/.ssh. It returns "" if cancelled.
func (a *App) PickKeyFile() (string, error) {
	home, _ := os.UserHomeDir()
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Choose a private key",
		DefaultDirectory: filepath.Join(home, ".ssh"),
	})
}

func readKeyFile(path string) ([]byte, error) {
	b, err := os.ReadFile(expandHome(path))
	if err != nil {
		return nil, fmt.Errorf("cannot read the key file: %w", err)
	}
	if len(b) > 1<<20 {
		return nil, errors.New("that file is too large to be a private key")
	}
	if !bytes.Contains(b, []byte("PRIVATE KEY")) {
		return nil, errors.New("that doesn't look like a private key (did you pick the .pub file?)")
	}
	return b, nil
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
