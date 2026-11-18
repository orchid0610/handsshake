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

func TestUpdate(t *testing.T) {
	path, _ := newVault(t)
	v := New(path)
	if err := v.Update("x", func(*Host) {}); !errors.Is(err, ErrLocked) {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
	if err := v.Unlock(pw); err != nil {
		t.Fatal(err)
	}
	hosts, _ := v.Hosts()
	id := hosts[0].ID

	if err := v.Update(id, func(h *Host) { h.OSName = "Ubuntu"; h.LastUsed = 42 }); err != nil {
		t.Fatal(err)
	}
	if err := v.Update("missing", func(*Host) {}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	v.Lock()
	if err := v.Unlock(pw); err != nil {
		t.Fatal(err)
	}
	got, _ := v.Get(id)
	if got.OSName != "Ubuntu" || got.LastUsed != 42 || got.Password != "hunter2" {
		t.Fatalf("update did not persist, or clobbered other fields: %+v", got)
	}
}

func TestSnippets(t *testing.T) {
	path, _ := newVault(t)
	v := New(path)
	if _, err := v.Snippets(); !errors.Is(err, ErrLocked) {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
	if err := v.Unlock(pw); err != nil {
		t.Fatal(err)
	}

	a, err := v.PutSnippet(Snippet{Title: "  Disk usage ", Code: "df -h\r\ndu -sh /var/log/*\n"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := v.PutSnippet(Snippet{Title: "Logs", Code: "tail -f /var/log/syslog"})
	if err != nil {
		t.Fatal(err)
	}

	got, _ := v.Snippets()
	if len(got) != 2 || got[0].ID != b || got[1].ID != a {
		t.Fatalf("newest must come first: %+v", got)
	}
	if got[1].Title != "Disk usage" || got[1].Code != "df -h\ndu -sh /var/log/*" {
		t.Fatalf("title/code were not cleaned up: %+v", got[1])
	}

	// Editing keeps the position.
	if _, err := v.PutSnippet(Snippet{ID: a, Title: "Disk", Code: "df -h"}); err != nil {
		t.Fatal(err)
	}
	if got, _ = v.Snippets(); got[1].ID != a || got[1].Title != "Disk" {
		t.Fatalf("edit moved or lost the snippet: %+v", got)
	}

	// Validation.
	bad := []Snippet{
		{Title: "", Code: "x"},
		{Title: "x", Code: "   "},
		{Title: strings.Repeat("a", MaxSnippetTitle+1), Code: "x"},
		{Title: "x", Code: strings.Repeat("a", MaxSnippetCode+1)},
	}
	for i, s := range bad {
		if _, err := v.PutSnippet(s); err == nil {
			t.Errorf("bad snippet %d was accepted", i)
		}
	}
	if _, err := v.PutSnippet(Snippet{ID: "nope", Title: "x", Code: "y"}); !errors.Is(err, ErrSnippetNotFound) {
		t.Fatalf("expected ErrSnippetNotFound, got %v", err)
	}

	// Survives a lock, and is encrypted on disk.
	v.Lock()
	if err := v.Unlock(pw); err != nil {
		t.Fatal(err)
	}
	if got, _ = v.Snippets(); len(got) != 2 || got[1].Code != "df -h" {
		t.Fatalf("snippets did not survive a lock/unlock: %+v", got)
	}
	raw, _ := os.ReadFile(path)
	if bytes.Contains(raw, []byte("tail -f")) {
		t.Fatal("snippet text leaked into the vault file")
	}

	if err := v.DeleteSnippet(b); err != nil {
		t.Fatal(err)
	}
	if got, _ = v.Snippets(); len(got) != 1 {
		t.Fatalf("expected 1 snippet, got %d", len(got))
	}
	if err := v.DeleteSnippet(b); !errors.Is(err, ErrSnippetNotFound) {
		t.Fatalf("expected ErrSnippetNotFound, got %v", err)
	}
}
