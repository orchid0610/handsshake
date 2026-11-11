package hostimport

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"handsshake/internal/sshconfig"
	"handsshake/internal/vault"
)

func writeKey(t *testing.T, path, passphrase string) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var block *pem.Block
	if passphrase == "" {
		block, err = ssh.MarshalPrivateKey(priv, "test")
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(priv, "test", []byte(passphrase))
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
}

func newHome(t *testing.T, config string) sshconfig.Env {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	if config != "" {
		if err := os.WriteFile(filepath.Join(home, ".ssh", "config"), []byte(config), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return sshconfig.Env{Home: home, LocalUser: "me"}
}

func TestBuild(t *testing.T) {
	env := newHome(t, `
Host web
    HostName 10.0.0.5
    User deploy
    Port 2222
    IdentityFile ~/.ssh/locked_key

Host plain
    HostName plain.example.com

Host gone
    HostName gone.example.com
    IdentityFile ~/.ssh/nope

Host jump
    HostName inner
    ProxyJump bastion

Host dup
    HostName dup.example.com
`)
	ssh := filepath.Join(env.Home, ".ssh")
	writeKey(t, filepath.Join(ssh, "locked_key"), "s3cret")
	writeKey(t, filepath.Join(ssh, "id_ed25519"), "") // default key

	// "dup" is already in the vault, saved without a port.
	existing := []vault.Host{{Name: "other", Address: "dup.example.com", User: "me"}}

	pv, hosts, err := Build("", env, existing)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Candidate{}
	for _, c := range pv.Candidates {
		got[c.Alias] = c
	}

	want := map[string]Candidate{
		"web":   {Address: "10.0.0.5:2222", User: "deploy", KeyPath: filepath.Join(ssh, "locked_key"), KeyStatus: KeyEncrypted, Status: StatusNew},
		"plain": {Address: "plain.example.com:22", User: "me", KeyPath: filepath.Join(ssh, "id_ed25519"), KeyStatus: KeyOK, Status: StatusNew},
		"gone":  {Address: "gone.example.com:22", User: "me", KeyPath: filepath.Join(ssh, "nope"), KeyStatus: KeyMissing, Status: StatusNew},
	}
	for alias, w := range want {
		c := got[alias]
		if c.Address != w.Address || c.User != w.User || c.KeyPath != w.KeyPath ||
			c.KeyStatus != w.KeyStatus || c.Status != w.Status {
			t.Errorf("%s: got %+v, want %+v", alias, c, w)
		}
	}
	if got["jump"].Status != StatusUnsupported {
		t.Errorf("jump: got %+v", got["jump"])
	}
	if got["dup"].Status != StatusExists {
		t.Errorf("dup (same user and address as an existing host): got %+v", got["dup"])
	}

	if h := hosts["web"]; h.Auth != vault.AuthKey || h.KeyPath != want["web"].KeyPath || h.Name != "web" {
		t.Errorf("web host to import: %+v", h)
	}
	for _, alias := range []string{"jump", "dup"} {
		if _, ok := hosts[alias]; ok {
			t.Errorf("%s must not be importable", alias)
		}
	}
}

func TestNoKeyMeansPassword(t *testing.T) {
	env := newHome(t, "Host p\n    HostName p.example.com\n")
	pv, hosts, err := Build("", env, nil)
	if err != nil {
		t.Fatal(err)
	}
	if pv.Candidates[0].KeyStatus != KeyNone {
		t.Fatalf("got %+v", pv.Candidates[0])
	}
	if h := hosts["p"]; h.Auth != vault.AuthPassword || h.KeyPath != "" {
		t.Fatalf("got %+v", h)
	}
}

func TestMissingConfig(t *testing.T) {
	env := newHome(t, "")
	if _, _, err := Build("", env, nil); err == nil || !strings.Contains(err.Error(), "no SSH config") {
		t.Fatalf("got %v", err)
	}
}

func TestEmbedKey(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "k")
	writeKey(t, key, "")

	h, err := EmbedKey(vault.Host{KeyPath: key})
	if err != nil || h.KeyPath != "" || !strings.Contains(h.PrivateKey, "PRIVATE KEY") {
		t.Fatalf("got %+v, %v", h, err)
	}

	notAKey := filepath.Join(dir, "notes.txt")
	os.WriteFile(notAKey, []byte("hello"), 0o600)
	if _, err := EmbedKey(vault.Host{KeyPath: notAKey}); err == nil {
		t.Fatal("a text file must not be embedded as a key")
	}
}