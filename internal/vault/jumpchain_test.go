package vault

import (
	"errors"
	"fmt"
	"testing"
)

func idsOf(hosts []Host) []string {
	out := make([]string, len(hosts))
	for i, h := range hosts {
		out[i] = h.ID
	}
	return out
}

func TestJumpChain(t *testing.T) {
	a := Host{ID: "a"}
	b := Host{ID: "b", JumpHostID: "a"}
	c := Host{ID: "c", JumpHostID: "b"}
	all := []Host{a, b, c}

	chain, err := JumpChain(all, c)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(idsOf(chain)); got != "[a b c]" {
		t.Fatalf("chain order: got %s, want [a b c]", got)
	}
	if chain, _ = JumpChain(all, a); len(chain) != 1 {
		t.Fatalf("a host with no jump host is a chain of one, got %v", idsOf(chain))
	}

	// A brand-new host (no ID yet) can be checked before it is saved.
	if chain, err = JumpChain(all, Host{JumpHostID: "c"}); err != nil || len(chain) != 4 {
		t.Fatalf("new host: %v, %v", idsOf(chain), err)
	}

	x := Host{ID: "x", JumpHostID: "y"}
	y := Host{ID: "y", JumpHostID: "x"}
	if _, err := JumpChain([]Host{x, y}, x); !errors.Is(err, ErrJumpCycle) {
		t.Errorf("loop: got %v", err)
	}
	self := Host{ID: "s", JumpHostID: "s"}
	if _, err := JumpChain([]Host{self}, self); !errors.Is(err, ErrJumpCycle) {
		t.Errorf("self: got %v", err)
	}
	lost := Host{ID: "m", JumpHostID: "gone"}
	if _, err := JumpChain([]Host{lost}, lost); !errors.Is(err, ErrJumpMissing) {
		t.Errorf("missing: got %v", err)
	}

	// MaxJumpHops jumps in a row is fine; one more is too many.
	var hosts []Host
	prev := ""
	for i := 0; i <= MaxJumpHops+1; i++ {
		id := fmt.Sprintf("h%d", i)
		hosts = append(hosts, Host{ID: id, JumpHostID: prev})
		prev = id
	}
	if _, err := JumpChain(hosts, hosts[MaxJumpHops]); err != nil {
		t.Errorf("%d jumps should be allowed: %v", MaxJumpHops, err)
	}
	if _, err := JumpChain(hosts, hosts[MaxJumpHops+1]); !errors.Is(err, ErrJumpTooDeep) {
		t.Errorf("one jump too many: got %v", err)
	}
}

func TestDeleteClearsJumpReferences(t *testing.T) {
	path, _ := newVault(t)
	v := New(path)
	if err := v.Unlock(pw); err != nil {
		t.Fatal(err)
	}
	bastion, _ := v.Put(Host{Name: "bastion", Address: "b:22", User: "u", Auth: AuthPassword, Password: "p"})
	app, _ := v.Put(Host{Name: "app", Address: "a:22", User: "u", Auth: AuthPassword, Password: "p", JumpHostID: bastion})

	if err := v.Delete(bastion); err != nil {
		t.Fatal(err)
	}
	v.Lock()
	if err := v.Unlock(pw); err != nil {
		t.Fatal(err)
	}
	got, err := v.Get(app)
	if err != nil || got.JumpHostID != "" {
		t.Fatalf("the deleted jump host is still referenced: %+v, %v", got, err)
	}
}

func TestAddManyKeepsIDs(t *testing.T) {
	path, _ := newVault(t)
	v := New(path)
	if err := v.Unlock(pw); err != nil {
		t.Fatal(err)
	}
	if err := v.AddMany([]Host{{ID: "keep-1", Name: "a"}, {Name: "b"}}); err != nil {
		t.Fatal(err)
	}
	if h, err := v.Get("keep-1"); err != nil || h.Name != "a" {
		t.Fatalf("the ID that was set must be kept: %+v, %v", h, err)
	}
	before, _ := v.Hosts()

	if err := v.AddMany([]Host{{Name: "fresh"}, {ID: "keep-1", Name: "dup"}}); err == nil {
		t.Fatal("a duplicate ID must be refused")
	}
	if after, _ := v.Hosts(); len(after) != len(before) {
		t.Fatalf("a refused batch must add nothing: %d hosts before, %d after", len(before), len(after))
	}
}