package sshconn

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// ---- a tiny in-process SSH server ----

type testServer struct {
	name    string
	addr    string
	hostKey ssh.PublicKey
	conns   atomic.Int32 // SSH connections currently open
	opened  atomic.Int32 // direct-tcpip channels opened (jump traffic)
}

func startTestServer(t *testing.T, name, password string) *testServer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(_ ssh.ConnMetadata, pw []byte) (*ssh.Permissions, error) {
			if string(pw) == password {
				return nil, nil
			}
			return nil, errors.New("wrong password")
		},
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })

	s := &testServer{name: name, addr: ln.Addr().String(), hostKey: signer.PublicKey()}
	go func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				return
			}
			go s.serve(nc, cfg)
		}
	}()
	return s
}

func (s *testServer) serve(nc net.Conn, cfg *ssh.ServerConfig) {
	conn, chans, reqs, err := ssh.NewServerConn(nc, cfg)
	if err != nil {
		nc.Close()
		return
	}
	defer conn.Close()
	s.conns.Add(1)
	defer s.conns.Add(-1)
	go ssh.DiscardRequests(reqs)

	for nch := range chans {
		switch nch.ChannelType() {
		case "direct-tcpip":
			s.forward(nch)
		case "session":
			s.session(nch)
		default:
			nch.Reject(ssh.UnknownChannelType, "unsupported")
		}
	}
}

// forward does what a bastion does: connect onwards and pipe the bytes.
func (s *testServer) forward(nch ssh.NewChannel) {
	var req struct {
		Host     string
		Port     uint32
		OrigHost string
		OrigPort uint32
	}
	if err := ssh.Unmarshal(nch.ExtraData(), &req); err != nil {
		nch.Reject(ssh.ConnectionFailed, "bad request")
		return
	}
	target, err := net.Dial("tcp", net.JoinHostPort(req.Host, strconv.Itoa(int(req.Port))))
	if err != nil {
		nch.Reject(ssh.ConnectionFailed, err.Error())
		return
	}
	ch, reqs, err := nch.Accept()
	if err != nil {
		target.Close()
		return
	}
	s.opened.Add(1)
	go ssh.DiscardRequests(reqs)
	go func() { io.Copy(ch, target); ch.Close() }()
	go func() { io.Copy(target, ch); target.Close() }()
}

// session answers any command with "hello from <name>".
func (s *testServer) session(nch ssh.NewChannel) {
	ch, reqs, err := nch.Accept()
	if err != nil {
		return
	}
	go func() {
		defer ch.Close()
		for r := range reqs {
			if r.Type != "exec" {
				r.Reply(false, nil)
				continue
			}
			r.Reply(true, nil)
			io.WriteString(ch, "hello from "+s.name)
			ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
			return
		}
	}()
}

// ---- helpers ----

// recordingHostKey accepts every key and remembers which fingerprint each host showed.
func recordingHostKey() (ssh.HostKeyCallback, func() map[string]string) {
	var mu sync.Mutex
	seen := map[string]string{}
	cb := func(host string, _ net.Addr, key ssh.PublicKey) error {
		mu.Lock()
		defer mu.Unlock()
		seen[host] = ssh.FingerprintSHA256(key)
		return nil
	}
	get := func() map[string]string {
		mu.Lock()
		defer mu.Unlock()
		cp := map[string]string{}
		for k, v := range seen {
			cp[k] = v
		}
		return cp
	}
	return cb, get
}

func dialServer(s *testServer, password string, hk ssh.HostKeyCallback, via *Conn) (*Conn, error) {
	return Dial(Config{
		Address: s.addr, User: "u", Creds: Credentials{Password: password},
		HostKey: hk, Via: via, Timeout: 5 * time.Second,
	})
}

func runCmd(t *testing.T, c *Conn) string {
	t.Helper()
	sess, err := c.client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	out, err := sess.Output("whoami")
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// closedAddr returns an address where nothing is listening.
func closedAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

// ---- tests ----

func TestDialThroughJumpHost(t *testing.T) {
	bastion := startTestServer(t, "bastion", "bp")
	target := startTestServer(t, "target", "tp")
	hk, seen := recordingHostKey()

	b, err := dialServer(bastion, "bp", hk, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := dialServer(target, "tp", hk, b)
	if err != nil {
		t.Fatal(err)
	}

	if out := runCmd(t, c); out != "hello from target" {
		t.Fatalf("the command ran on the wrong machine: %q", out)
	}
	got := seen()
	if got[bastion.addr] != ssh.FingerprintSHA256(bastion.hostKey) {
		t.Error("the jump host's key was not verified")
	}
	if got[target.addr] != ssh.FingerprintSHA256(target.hostKey) {
		t.Error("the TARGET's key was not verified (it must be checked too, not just the jump host's)")
	}
	if bastion.opened.Load() != 1 {
		t.Errorf("the connection did not go through the jump host: %d channels", bastion.opened.Load())
	}
	if c.HostKeyType() != "ssh-ed25519" {
		t.Errorf("host key type: %q", c.HostKeyType())
	}

	// Closing the target closes the jump connection too.
	c.Close()
	waitFor(t, "both connections to close", func() bool {
		return bastion.conns.Load() == 0 && target.conns.Load() == 0
	})
}

func TestJumpHostCannotReachTarget(t *testing.T) {
	bastion := startTestServer(t, "bastion", "bp")
	hk, _ := recordingHostKey()
	b, err := dialServer(bastion, "bp", hk, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Dial(Config{Address: closedAddr(t), User: "u", Creds: Credentials{Password: "x"},
		HostKey: hk, Via: b, Timeout: 5 * time.Second})
	if err == nil || !strings.Contains(err.Error(), "jump host") {
		t.Fatalf("expected an error that blames the jump host's route, got %v", err)
	}
	waitFor(t, "the jump connection to be closed", func() bool { return bastion.conns.Load() == 0 })
}

func TestJumpWrongPasswordAtTarget(t *testing.T) {
	bastion := startTestServer(t, "bastion", "bp")
	target := startTestServer(t, "target", "tp")
	hk, _ := recordingHostKey()
	b, err := dialServer(bastion, "bp", hk, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := dialServer(target, "WRONG", hk, b); !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
	waitFor(t, "the jump connection to be closed", func() bool { return bastion.conns.Load() == 0 })
}

func TestJumpHostKeyRejectedAtTarget(t *testing.T) {
	bastion := startTestServer(t, "bastion", "bp")
	target := startTestServer(t, "target", "tp")
	hk := func(host string, _ net.Addr, _ ssh.PublicKey) error {
		if host == target.addr {
			return ErrHostKeyRejected
		}
		return nil
	}
	b, err := dialServer(bastion, "bp", hk, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := dialServer(target, "tp", hk, b); !errors.Is(err, ErrHostKeyRejected) {
		t.Fatalf("expected ErrHostKeyRejected, got %v", err)
	}
	waitFor(t, "the jump connection to be closed", func() bool { return bastion.conns.Load() == 0 })
}

// The user may take a while to read a fingerprint: that must not count as a timeout.
func TestSlowHostKeyConfirmationIsNotATimeout(t *testing.T) {
	target := startTestServer(t, "target", "tp")
	slow := func(string, net.Addr, ssh.PublicKey) error {
		time.Sleep(600 * time.Millisecond)
		return nil
	}
	c, err := Dial(Config{Address: target.addr, User: "u", Creds: Credentials{Password: "tp"},
		HostKey: slow, Timeout: 200 * time.Millisecond})
	if err != nil {
		t.Fatalf("a slow confirmation was treated as a timeout: %v", err)
	}
	c.Close()
}

// A server that accepts the connection but never speaks SSH must not hang us forever.
func TestHandshakeTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c // accept, then stay silent
		}
	}()

	hk, _ := recordingHostKey()
	start := time.Now()
	_, err = Dial(Config{Address: ln.Addr().String(), User: "u", Creds: Credentials{Password: "x"},
		HostKey: hk, Timeout: 200 * time.Millisecond})
	if err == nil || !strings.Contains(err.Error(), "handshake") {
		t.Fatalf("expected a handshake timeout, got %v", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("took %v to give up", time.Since(start))
	}
}