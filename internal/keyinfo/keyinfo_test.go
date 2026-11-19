package keyinfo

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"golang.org/x/crypto/ssh"

	"handsshake/internal/vault"
)

func fingerprint(t *testing.T, pub any) string {
	t.Helper()
	k, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return ssh.FingerprintSHA256(k)
}

func marshal(t *testing.T, priv any, passphrase string) []byte {
	t.Helper()
	var block *pem.Block
	var err error
	if passphrase == "" {
		block, err = ssh.MarshalPrivateKey(priv, "test")
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(priv, "test", []byte(passphrase))
	}
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(block)
}

// legacyEncrypted makes an old-style PEM key ("Proc-Type: 4,ENCRYPTED") whose
// public half is NOT readable without the passphrase.
func legacyEncrypted(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	block, err := x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY",
		x509.MarshalPKCS1PrivateKey(key), []byte("pw"), x509.PEMCipherAES256)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(block)
}

func find(t *testing.T, keys []Key, name string) Key {
	t.Helper()
	for _, k := range keys {
		if k.Name == name {
			return k
		}
	}
	t.Fatalf("no key named %q in %+v", name, keys)
	return Key{}
}

func TestList(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	put := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	edPub, edPriv, _ := ed25519.GenerateKey(rand.Reader)
	put("id_ed25519", marshal(t, edPriv, ""))

	lockedPub, lockedPriv, _ := ed25519.GenerateKey(rand.Reader)
	put("locked_key", marshal(t, lockedPriv, "pw"))

	rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	legacy1, _ := rsa.GenerateKey(rand.Reader, 2048)
	legacy2, _ := rsa.GenerateKey(rand.Reader, 2048)
	put("legacy_key", legacyEncrypted(t, legacy1))
	put("legacy_with_pub", legacyEncrypted(t, legacy2))
	pub2, _ := ssh.NewPublicKey(&legacy2.PublicKey)
	put("legacy_with_pub.pub", ssh.MarshalAuthorizedKey(pub2))
	put("notes.txt", []byte("hello"))

	hosts := []vault.Host{
		{Name: "alpha", Auth: vault.AuthKey, KeyPath: "~/.ssh/id_ed25519"}, // "~" must expand
		{Name: "beta", Auth: vault.AuthKey, KeyPath: filepath.Join(dir, "id_ed25519")},
		{Name: "gamma", Auth: vault.AuthKey, KeyPath: filepath.Join(dir, "locked_key")},
		{Name: "delta", Auth: vault.AuthKey, KeyPath: filepath.Join(dir, "locked_key"), Passphrase: "pw"},
		{Name: "epsilon", Auth: vault.AuthKey, PrivateKey: string(marshal(t, rsaKey, ""))},
		{Name: "zeta", Auth: vault.AuthKey, KeyPath: filepath.Join(dir, "nope")},
		{Name: "eta", Auth: vault.AuthKey, KeyPath: filepath.Join(dir, "notes.txt")},
		{Name: "iota", Auth: vault.AuthKey, KeyPath: filepath.Join(dir, "legacy_key"), Passphrase: "pw"},
		{Name: "kappa", Auth: vault.AuthKey, KeyPath: filepath.Join(dir, "legacy_with_pub")},
		{Name: "password-host", Auth: vault.AuthPassword, Password: "x"},
		{Name: "no-key-yet", Auth: vault.AuthKey},
	}

	keys := List(hosts, home)
	if len(keys) != 7 {
		t.Fatalf("expected 7 distinct keys, got %d: %+v", len(keys), keys)
	}

	// A key shared by two hosts appears once; "~" was expanded.
	ed := find(t, keys, "id_ed25519")
	if ed.Source != "file" || ed.Type != "ED25519" || ed.Status != StatusOK || ed.Encrypted ||
		ed.Fingerprint != fingerprint(t, edPub) || ed.Path != "~/.ssh/id_ed25519" ||
		!reflect.DeepEqual(ed.UsedBy, []string{"alpha", "beta"}) {
		t.Errorf("id_ed25519: %+v", ed)
	}

	// Encrypted OpenSSH key: fingerprint is available without the passphrase.
	// One host has no saved passphrase, so the key is "locked".
	locked := find(t, keys, "locked_key")
	if !locked.Encrypted || locked.Status != StatusLocked || locked.Fingerprint != fingerprint(t, lockedPub) ||
		!reflect.DeepEqual(locked.UsedBy, []string{"delta", "gamma"}) {
		t.Errorf("locked_key: %+v", locked)
	}
	// With every user's passphrase saved, the same key is fine.
	onlyDelta := List([]vault.Host{hosts[3]}, home)
	if len(onlyDelta) != 1 || onlyDelta[0].Status != StatusOK || !onlyDelta[0].Encrypted {
		t.Errorf("locked_key with saved passphrase: %+v", onlyDelta)
	}

	// Stored in the vault.
	emb := find(t, keys, "epsilon key")
	if emb.Source != "vault" || emb.Path != "" || emb.Type != "RSA 2048" || emb.Status != StatusOK {
		t.Errorf("embedded key: %+v", emb)
	}

	if k := find(t, keys, "nope"); k.Status != StatusMissing || k.Fingerprint != "" {
		t.Errorf("missing file: %+v", k)
	}
	if k := find(t, keys, "notes.txt"); k.Status != StatusInvalid {
		t.Errorf("invalid file: %+v", k)
	}

	// Old-format encrypted key: no fingerprint, but the type comes from the PEM header.
	if k := find(t, keys, "legacy_key"); !k.Encrypted || k.Fingerprint != "" || k.Type != "RSA" || k.Status != StatusOK {
		t.Errorf("legacy_key: %+v", k)
	}
	// ...unless a .pub file sits next to it.
	if k := find(t, keys, "legacy_with_pub"); !k.Encrypted || k.Fingerprint != fingerprint(t, &legacy2.PublicKey) ||
		k.Type != "RSA 2048" || k.Status != StatusLocked {
		t.Errorf("legacy_with_pub: %+v", k)
	}
}

func TestListEmpty(t *testing.T) {
	if keys := List(nil, "/home/x"); keys == nil || len(keys) != 0 {
		t.Fatalf("expected an empty, non-nil slice, got %#v", keys)
	}
}
