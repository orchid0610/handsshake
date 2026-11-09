package sshconn

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"net"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func pemKey(t *testing.T, passphrase string) []byte {
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
	return pem.EncodeToMemory(block)
}

func pubKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	s, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s.PublicKey()
}

func TestAuthMethods(t *testing.T) {
	if _, err := (Credentials{}).authMethods(); !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("got %v", err)
	}
	if m, err := (Credentials{Password: "x"}).authMethods(); err != nil || len(m) != 2 {
		t.Fatalf("password auth: %v, %v", m, err)
	}
}

func TestParseKey(t *testing.T) {
	plain := pemKey(t, "")
	locked := pemKey(t, "s3cret")

	if _, err := parseKey(plain, ""); err != nil {
		t.Fatalf("plain key: %v", err)
	}
	if _, err := parseKey(plain, "unneeded"); err != nil {
		t.Fatalf("plain key with spare passphrase: %v", err)
	}
	if _, err := parseKey(locked, ""); !errors.Is(err, ErrPassphraseRequired) {
		t.Fatalf("expected ErrPassphraseRequired, got %v", err)
	}
	if _, err := parseKey(locked, "nope"); !errors.Is(err, ErrBadPassphrase) {
		t.Fatalf("expected ErrBadPassphrase, got %v", err)
	}
	if _, err := parseKey(locked, "s3cret"); err != nil {
		t.Fatalf("right passphrase: %v", err)
	}
	if _, err := parseKey([]byte("garbage"), ""); err == nil {
		t.Fatal("garbage key was accepted")
	}
}

func TestKnownHosts(t *testing.T) {
	kh := NewKnownHosts(filepath.Join(t.TempDir(), "known_hosts"))
	addr := &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 22}
	const host = "10.0.0.1:22"
	k1, k2 := pubKey(t), pubKey(t)

	// 1. Unknown host, user says no.
	if err := kh.Callback(func(HostKeyInfo) bool { return false })(host, addr, k1); !errors.Is(err, ErrHostKeyRejected) {
		t.Fatalf("expected rejection, got %v", err)
	}

	// 2. Unknown host, user says yes: asked once, then remembered.
	asked := 0
	cb := kh.Callback(func(HostKeyInfo) bool { asked++; return true })
	if err := cb(host, addr, k1); err != nil {
		t.Fatal(err)
	}
	if err := cb(host, addr, k1); err != nil {
		t.Fatal(err)
	}
	if asked != 1 {
		t.Fatalf("expected to be asked once, asked %d times", asked)
	}

	// 3. Same host, different key: must be refused, and never even asked about.
	err := cb(host, addr, k2)
	var changed *ChangedKeyError
	if !errors.As(err, &changed) {
		t.Fatalf("expected ChangedKeyError, got %v", err)
	}
	if asked != 1 {
		t.Fatal("a changed key must not trigger a trust prompt")
	}
}