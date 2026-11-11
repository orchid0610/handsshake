package sshconfig

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func setup(t *testing.T, config string) (Env, string) {
	t.Helper()
	home := t.TempDir()
	path := filepath.Join(home, ".ssh", "config")
	write(t, path, config)
	return Env{Home: home, LocalUser: "me"}, path
}

func load(t *testing.T, config string) (*Result, Env) {
	t.Helper()
	env, path := setup(t, config)
	res, err := Load(path, env)
	if err != nil {
		t.Fatal(err)
	}
	return res, env
}

func find(t *testing.T, res *Result, alias string) Entry {
	t.Helper()
	for _, e := range res.Entries {
		if e.Alias == alias {
			return e
		}
	}
	t.Fatalf("no entry %q in %+v", alias, res.Entries)
	return Entry{}
}

func TestBasic(t *testing.T) {
	res, env := load(t, `
Host web
    HostName 10.0.0.5
    User deploy
    Port 2222
    IdentityFile ~/.ssh/web_key
`)
	want := Entry{Alias: "web", HostName: "10.0.0.5", User: "deploy", Port: 2222,
		IdentityFiles: []string{filepath.Join(env.Home, ".ssh", "web_key")}}
	if len(res.Entries) != 1 || !reflect.DeepEqual(res.Entries[0], want) {
		t.Fatalf("got %+v, want %+v", res.Entries, want)
	}
}

func TestDefaultsApplyButFirstValueWins(t *testing.T) {
	res, env := load(t, `
Host db
    User dbadmin
Host *
    User fallback
    IdentityFile ~/.ssh/default
Host app
    HostName app.example.com
`)
	if len(res.Entries) != 2 {
		t.Fatalf("Host * must not become an entry: %+v", res.Entries)
	}
	db, app := find(t, res, "db"), find(t, res, "app")
	if db.User != "dbadmin" {
		t.Errorf("first value must win, got %q", db.User)
	}
	if app.User != "fallback" || app.HostName != "app.example.com" {
		t.Errorf("defaults not applied: %+v", app)
	}
	key := filepath.Join(env.Home, ".ssh", "default")
	if !reflect.DeepEqual(app.IdentityFiles, []string{key}) || !reflect.DeepEqual(db.IdentityFiles, []string{key}) {
		t.Errorf("identity file default missing: %+v / %+v", app.IdentityFiles, db.IdentityFiles)
	}
}

func TestWildcardsAndNegation(t *testing.T) {
	res, _ := load(t, `
Host *.internal !secret.internal
    User ops
Host a.internal secret.internal other
    Port 22
`)
	if find(t, res, "a.internal").User != "ops" {
		t.Error("a.internal should match *.internal")
	}
	if find(t, res, "secret.internal").User != "me" {
		t.Error("!secret.internal must exclude the block")
	}
	if find(t, res, "other").User != "me" {
		t.Error("other matches nothing")
	}
}

func TestSyntaxVariations(t *testing.T) {
	res, _ := load(t, "# a comment\r\nHost   Foo   # trailing comment\r\n  hostname = \"my server.example.com\"   # note\r\n  USER=root\r\n")
	e := find(t, res, "Foo")
	if e.HostName != "my server.example.com" || e.User != "root" {
		t.Fatalf("got %+v", e)
	}
}

func TestMatchBlocksAreIgnored(t *testing.T) {
	res, _ := load(t, `
Host a
    User x
Match host a
    User y
Host b
`)
	if find(t, res, "a").User != "x" {
		t.Error("Match options must not apply")
	}
	if find(t, res, "b").User != "me" {
		t.Error("options of a Match block leaked into the next Host")
	}
}

func TestIncludeRelativeToSSHDir(t *testing.T) {
	env, path := setup(t, "Include conf.d/*.conf\nHost main\n    HostName main.example.com\n")
	write(t, filepath.Join(env.Home, ".ssh", "conf.d", "x.conf"),
		"Host included\n    HostName inc.example.com\n    User inc\n")

	res, err := Load(path, env)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Entries) != 2 || res.Entries[0].Alias != "included" || res.Entries[1].Alias != "main" {
		t.Fatalf("got %+v", res.Entries)
	}
	if res.Entries[1].User != "me" {
		t.Error("options from the included file leaked into the parent")
	}
}

func TestIncludeInsideHostBlockIsIgnoredWithWarning(t *testing.T) {
	res, _ := load(t, "Host a\n    Include whatever\n")
	if len(res.Warnings) != 1 {
		t.Fatalf("warnings: %v", res.Warnings)
	}
}

func TestTokensAndKeyPaths(t *testing.T) {
	res, env := load(t, `
Host srv
    HostName %h.example.com
    IdentityFile ~/.ssh/%h_key
    IdentityFile keys/rel
    IdentityFile none
`)
	e := find(t, res, "srv")
	if e.HostName != "srv.example.com" {
		t.Errorf("HostName = %q", e.HostName)
	}
	want := []string{
		filepath.Join(env.Home, ".ssh", "srv.example.com_key"),
		filepath.Join(env.Home, ".ssh", "keys", "rel"),
	}
	if !reflect.DeepEqual(e.IdentityFiles, want) {
		t.Errorf("IdentityFiles = %v, want %v", e.IdentityFiles, want)
	}
}

func TestUnresolvableTokenIsAProblem(t *testing.T) {
	res, _ := load(t, "Host x\n    HostName %q.example.com\n")
	if find(t, res, "x").Problem == "" {
		t.Fatal("expected a Problem")
	}
}

func TestInvalidPortFallsBackWithWarning(t *testing.T) {
	res, _ := load(t, "Host p\n    Port banana\n")
	if find(t, res, "p").Port != 22 || len(res.Warnings) != 1 {
		t.Fatalf("port %d, warnings %v", find(t, res, "p").Port, res.Warnings)
	}
}

func TestProxyOptions(t *testing.T) {
	res, _ := load(t, `
Host j
    ProxyJump bastion
Host k
    ProxyCommand ssh -W %h:%p gateway
Host m
    ProxyJump none
`)
	if find(t, res, "j").ProxyJump != "bastion" {
		t.Error("ProxyJump lost")
	}
	if find(t, res, "k").ProxyCommand != "ssh -W %h:%p gateway" {
		t.Errorf("ProxyCommand = %q", find(t, res, "k").ProxyCommand)
	}
	if find(t, res, "m").ProxyJump != "" {
		t.Error("ProxyJump none must clear it")
	}
}

func TestDuplicateHostLinesGiveOneEntry(t *testing.T) {
	res, _ := load(t, "Host dup\n    HostName one\nHost dup\n    HostName two\n")
	if len(res.Entries) != 1 || res.Entries[0].HostName != "one" {
		t.Fatalf("got %+v", res.Entries)
	}
}

func TestMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope"), Env{})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
}

func TestWildcard(t *testing.T) {
	cases := []struct {
		pat, s string
		want   bool
	}{
		{"*", "anything", true},
		{"*", "", true},
		{"", "", true},
		{"*.internal", "a.internal", true},
		{"*.internal", "a.external", false},
		{"web?", "web1", true},
		{"web?", "web12", false},
		{"a*b*c", "aXXbYYc", true},
		{"a*b*c", "aXXbYY", false},
	}
	for _, c := range cases {
		if got := wildcard(c.pat, c.s); got != c.want {
			t.Errorf("wildcard(%q, %q) = %v, want %v", c.pat, c.s, got, c.want)
		}
	}
}