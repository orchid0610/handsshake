package vault

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pw = "correct horse battery"

// newVault creates a vault holding one host and returns its path and recovery key.
func newVault(t *testing.T) (path, recovery string) {
	t.Helper()
	path = filepath.Join(t.TempDir(), "vault.json")
	v := New(path)
	rk, err := v.Create(pw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Put(Host{Name: "srv", Address: "10.1.2.3:22", User: "root",
		Auth: AuthPassword, Password: "hunter2"}); err != nil {
		t.Fatal(err)
	}
	v.Lock()
	return path, rk
}

func TestRoundTrip(t *testing.T) {
	path, _ := newVault(t)

	v := New(path)
	if _, err := v.Hosts(); !errors.Is(err, ErrLocked) {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
	if err := v.Unlock("wrong password"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("expected ErrWrongPassword, got %v", err)
	}
	if err := v.Unlock(pw); err != nil {
		t.Fatal(err)
	}
	hosts, err := v.Hosts()
	if err != nil || len(hosts) != 1 || hosts[0].Password != "hunter2" {
		t.Fatalf("host did not survive the round trip: %+v, %v", hosts, err)
	}

	// Nothing sensitive may appear in plaintext on disk.
	raw, _ := os.ReadFile(path)
	for _, secret := range []string{"hunter2", "10.1.2.3", "root", "srv"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatalf("%q leaked into the vault file", secret)
		}
	}
}

func TestRecoveryKey(t *testing.T) {
	path, rk := newVault(t)

	if err := New(path).UnlockWithRecovery("AAAA-BBBB", "new password 123"); !errors.Is(err, ErrBadRecoveryKey) {
		t.Fatalf("garbage recovery key: got %v", err)
	}

	// Typing it in lowercase without dashes must still work.
	sloppy := strings.ToLower(strings.ReplaceAll(rk, "-", ""))
	v := New(path)
	if err := v.UnlockWithRecovery(sloppy, "new password 123"); err != nil {
		t.Fatal(err)
	}
	v.Lock()

	if err := v.Unlock(pw); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("old password should be dead, got %v", err)
	}
	if err := v.Unlock("new password 123"); err != nil {
		t.Fatal(err)
	}
	if hosts, _ := v.Hosts(); len(hosts) != 1 {
		t.Fatalf("data lost after recovery: %+v", hosts)
	}
}

func TestChangePassword(t *testing.T) {
	path, _ := newVault(t)
	v := New(path)
	if err := v.Unlock(pw); err != nil {
		t.Fatal(err)
	}
	if err := v.ChangePassword("not the password", "another password 1"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("got %v", err)
	}
	if err := v.ChangePassword(pw, "another password 1"); err != nil {
		t.Fatal(err)
	}
	v.Lock()
	if err := v.Unlock("another password 1"); err != nil {
		t.Fatal(err)
	}
}

func TestTamperingIsDetected(t *testing.T) {
	cases := map[string]func(*envelope){
		"data":   func(e *envelope) { e.Data[0] ^= 0xff },
		"pwWrap": func(e *envelope) { e.PwWrap[0] ^= 0xff },
		"params": func(e *envelope) { e.Params.Time = 2 }, // weakening the KDF
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			path, _ := newVault(t)
			raw, _ := os.ReadFile(path)
			var e envelope
			if err := json.Unmarshal(raw, &e); err != nil {
				t.Fatal(err)
			}
			mutate(&e)
			raw, _ = json.Marshal(e)
			os.WriteFile(path, raw, 0o600)

			if err := New(path).Unlock(pw); err == nil {
				t.Fatal("tampering was not detected")
			}
		})
	}
}

func TestAddMany(t *testing.T) {
	path, _ := newVault(t) // starts with one host

	v := New(path)
	if err := v.AddMany([]Host{{Name: "a"}}); !errors.Is(err, ErrLocked) {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
	if err := v.Unlock(pw); err != nil {
		t.Fatal(err)
	}
	if err := v.AddMany([]Host{{Name: "a"}, {Name: "b"}}); err != nil {
		t.Fatal(err)
	}

	v.Lock()
	if err := v.Unlock(pw); err != nil {
		t.Fatal(err)
	}
	hosts, _ := v.Hosts()
	if len(hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d", len(hosts))
	}
	ids := map[string]bool{}
	for _, h := range hosts {
		if h.ID == "" || ids[h.ID] {
			t.Fatalf("IDs must be unique and non-empty: %+v", hosts)
		}
		ids[h.ID] = true
	}
}
