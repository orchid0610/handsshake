package app

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"handsshake/internal/hostimport"
	"handsshake/internal/sshconfig"
	"handsshake/internal/vault"
)

func sshEnv() (sshconfig.Env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return sshconfig.Env{}, err
	}
	name := os.Getenv("USER")
	if u, err := user.Current(); err == nil {
		name = u.Username
	}
	return sshconfig.Env{Home: home, LocalUser: name}, nil
}

// PickConfigFile opens a native file dialog in ~/.ssh. It returns "" if cancelled.
func (a *App) PickConfigFile() (string, error) {
	home, _ := os.UserHomeDir()
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Choose an SSH config file",
		DefaultDirectory: filepath.Join(home, ".ssh"),
	})
}

// PreviewSSHConfig describes what importing would do. An empty path means ~/.ssh/config.
func (a *App) PreviewSSHConfig(path string) (hostimport.Preview, error) {
	env, err := sshEnv()
	if err != nil {
		return hostimport.Preview{}, err
	}
	existing, err := a.vault.Hosts()
	if err != nil {
		return hostimport.Preview{}, err
	}
	pv, _, err := hostimport.Build(path, env, existing)
	return pv, err
}

// ImportHosts adds the chosen hosts. It re-reads the config instead of
// trusting host data from the UI: the UI only says WHICH aliases to import.
func (a *App) ImportHosts(path string, aliases []string, copyKeys bool) (int, error) {
	env, err := sshEnv()
	if err != nil {
		return 0, err
	}
	existing, err := a.vault.Hosts()
	if err != nil {
		return 0, err
	}
	pv, hosts, err := hostimport.Build(path, env, existing)
	if err != nil {
		return 0, err
	}
	keyStatus := map[string]string{}
	for _, c := range pv.Candidates {
		keyStatus[c.Alias] = c.KeyStatus
	}

	seen := map[string]bool{}
	var add []vault.Host
	for _, alias := range aliases {
		h, ok := hosts[alias] // only hosts that are importable are in the map
		if !ok || seen[alias] {
			continue
		}
		seen[alias] = true
		if copyKeys && (keyStatus[alias] == hostimport.KeyOK || keyStatus[alias] == hostimport.KeyEncrypted) {
			if embedded, err := hostimport.EmbedKey(h); err == nil {
				h = embedded
			}
		}
		add = append(add, h)
	}
	if len(add) == 0 {
		return 0, nil
	}
	if err := a.vault.AddMany(add); err != nil {
		return 0, err
	}
	return len(add), nil
}