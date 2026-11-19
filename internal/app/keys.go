package app

import (
	"os"

	"handsshake/internal/keyinfo"
)

// ListKeys describes the SSH keys the saved hosts use. It reads key files to
// work out each key's type and fingerprint, but only public information
// leaves this function.
func (a *App) ListKeys() ([]keyinfo.Key, error) {
	hosts, err := a.vault.Hosts()
	if err != nil {
		return nil, err
	}
	home, _ := os.UserHomeDir()
	return keyinfo.List(hosts, home), nil
}