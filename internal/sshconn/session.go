package sshconn

import (
	"errors"
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
}

// Conn is an authenticated connection. Shells (and later SFTP) are
// channels opened on top of it.
type Conn struct {
	client *ssh.Client
	done   chan struct{}
	once   sync.Once
}

func Dial(cfg Config) (*Conn, error) {
	if cfg.HostKey == nil {
		return nil, errors.New("sshconn: a HostKey callback is required")
	}
	methods, err := cfg.Creds.authMethods()
	if err != nil {
		return nil, err
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	// Remember what the host key check said: ssh wraps it in a generic
	// "handshake failed" error, and we want to show the real reason.
	var hostErr error
	sc := &ssh.ClientConfig{
		User: cfg.User,
		Auth: methods,
		HostKeyCallback: func(h string, r net.Addr, k ssh.PublicKey) error {
			hostErr = cfg.HostKey(h, r, k)
			return hostErr
		},
		Timeout: timeout,
	}

	client, err := ssh.Dial("tcp", withDefaultPort(cfg.Address), sc)
	if err != nil {
		if hostErr != nil {
			return nil, hostErr
		}
		if strings.Contains(err.Error(), "unable to authenticate") {
			return nil, ErrAuthFailed
		}
		return nil, err
	}

	c := &Conn{client: client, done: make(chan struct{})}
	go c.keepAlive(30 * time.Second)
	return c, nil
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

func (c *Conn) Close() error {
	c.once.Do(func() { close(c.done) })
	return c.client.Close()
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
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	fail := func(err error) (*Shell, error) { s.Close(); return nil, err }

	if err := s.RequestPty("xterm-256color", rows, cols, modes); err != nil {
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