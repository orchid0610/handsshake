package sshconn

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Address string // "host" or "host:port" (port defaults to 22)
	User    string
	Creds   Credentials
	HostKey ssh.HostKeyCallback // REQUIRED: there is deliberately no insecure default
	Timeout time.Duration       // default 10s

	// Via, if set, is an open connection to a jump host: the new connection is
	// tunnelled through it. Dial takes ownership of Via: it is closed when the
	// new connection is closed, or at once if Dial fails.
	Via *Conn
}

// Conn is an authenticated connection. Shells (and later SFTP) are
// channels opened on top of it.
type Conn struct {
	client      *ssh.Client
	via         *Conn // the jump host this connection runs through, if any
	done        chan struct{}
	once        sync.Once
	hostKeyType string // e.g. "ssh-ed25519": the key the server proved its identity with
}

func Dial(cfg Config) (*Conn, error) {
	fail := func(err error) (*Conn, error) {
		if cfg.Via != nil {
			cfg.Via.Close()
		}
		return nil, err
	}

	if cfg.HostKey == nil {
		return fail(errors.New("sshconn: a HostKey callback is required"))
	}
	methods, err := cfg.Creds.authMethods()
	if err != nil {
		return fail(err)
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	addr := withDefaultPort(cfg.Address)

	nc, err := openTransport(cfg.Via, addr, timeout)
	if err != nil {
		return fail(err)
	}

	// The handshake gets `timeout` to finish, but the clock stops while the
	// user is deciding whether to trust a new host key.
	var (
		mu       sync.Mutex
		timedOut bool
		hostErr  error
		keyType  string
	)
	timer := time.AfterFunc(timeout, func() {
		mu.Lock()
		timedOut = true
		mu.Unlock()
		nc.Close() // unblocks the handshake
	})

	sc := &ssh.ClientConfig{
		User: cfg.User,
		Auth: methods,
		HostKeyCallback: func(h string, r net.Addr, k ssh.PublicKey) error {
			timer.Stop()
			err := cfg.HostKey(h, r, k)
			mu.Lock()
			keyType, hostErr = k.Type(), err
			mu.Unlock()
			timer.Reset(timeout)
			return err
		},
	}

	conn, chans, reqs, err := ssh.NewClientConn(nc, addr, sc)
	timer.Stop()
	mu.Lock()
	to, he, kt := timedOut, hostErr, keyType
	mu.Unlock()

	if err != nil {
		nc.Close()
		// ssh wraps the real reason in a generic "handshake failed"; show the real one.
		switch {
		case he != nil:
			return fail(he)
		case to:
			return fail(errors.New("the server did not complete the SSH handshake in time"))
		case strings.Contains(err.Error(), "unable to authenticate"):
			return fail(ErrAuthFailed)
		default:
			return fail(err)
		}
	}

	c := &Conn{
		client:      ssh.NewClient(conn, chans, reqs),
		via:         cfg.Via,
		done:        make(chan struct{}),
		hostKeyType: kt,
	}
	go c.keepAlive(30 * time.Second)
	return c, nil
}

// openTransport opens the raw connection to addr: directly, or as a channel
// through a jump host.
func openTransport(via *Conn, addr string, timeout time.Duration) (net.Conn, error) {
	if via == nil {
		return net.DialTimeout("tcp", addr, timeout)
	}

	type result struct {
		nc  net.Conn
		err error
	}
	done := make(chan result, 1)
	go func() {
		nc, err := via.client.Dial("tcp", addr)
		done <- result{nc, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			return nil, fmt.Errorf("the jump host could not reach %s: %w", addr, r.err)
		}
		return r.nc, nil
	case <-time.After(timeout):
		go func() { // don't leak the channel if it opens after all
			if r := <-done; r.nc != nil {
				r.nc.Close()
			}
		}()
		return nil, fmt.Errorf("the jump host did not answer for %s in time", addr)
	}
}

// HostKeyType is the key algorithm the server identified itself with.
func (c *Conn) HostKeyType() string { return c.hostKeyType }

// Ping measures one round trip to the server using an SSH keepalive request.
func (c *Conn) Ping() (time.Duration, error) {
	start := time.Now()
	done := make(chan error, 1)
	go func() {
		_, _, err := c.client.SendRequest("keepalive@openssh.com", true, nil)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			return 0, err
		}
		return time.Since(start), nil
	case <-time.After(5 * time.Second):
		return 0, errors.New("timed out")
	}
}

// keepAlive pings the server so dead connections (Wi-Fi dropped, laptop
// slept) are noticed instead of hanging forever.
func (c *Conn) keepAlive(every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-t.C:
			if _, _, err := c.client.SendRequest("keepalive@openssh.com", true, nil); err != nil {
				c.client.Close()
				return
			}
		}
	}
}

// Close closes this connection and, after it, the jump host it ran through.
func (c *Conn) Close() error {
	c.once.Do(func() { close(c.done) })
	err := c.client.Close()
	if c.via != nil {
		c.via.Close()
	}
	return err
}

// Shell is an interactive terminal on the server. Read returns what the
// server prints; Write sends keystrokes.
type Shell struct {
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
}

func (c *Conn) OpenShell(cols, rows int) (*Shell, error) {
	s, err := c.client.NewSession()
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*Shell, error) { s.Close(); return nil, err }

	if err := s.RequestPty("xterm-256color", rows, cols, ptyModes()); err != nil {
		return fail(err)
	}
	stdin, err := s.StdinPipe()
	if err != nil {
		return fail(err)
	}
	stdout, err := s.StdoutPipe()
	if err != nil {
		return fail(err)
	}
	if err := s.Shell(); err != nil {
		return fail(err)
	}
	return &Shell{session: s, stdin: stdin, stdout: stdout}, nil
}

// ptyModes are the terminal modes requested for every shell. Flow control is
// turned off so Ctrl+S reaches the shell (forward history search) instead of
// freezing the session until Ctrl+Q.
func ptyModes() ssh.TerminalModes {
	return ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.IXON:          0,
		ssh.IXOFF:         0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
}

func (s *Shell) Read(p []byte) (int, error)  { return s.stdout.Read(p) }
func (s *Shell) Write(p []byte) (int, error) { return s.stdin.Write(p) }
func (s *Shell) Resize(cols, rows int) error { return s.session.WindowChange(rows, cols) }
func (s *Shell) Close() error                { return s.session.Close() }

func withDefaultPort(addr string) string {
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return addr
	}
	return net.JoinHostPort(strings.Trim(addr, "[]"), "22")
}