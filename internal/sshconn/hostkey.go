package sshconn

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var ErrHostKeyRejected = errors.New("host key was not trusted")

// HostKeyInfo describes a server we have never seen before.
type HostKeyInfo struct {
	Host        string // host:port as dialed
	KeyType     string // e.g. ssh-ed25519
	Fingerprint string // e.g. SHA256:abc...
}

// ChangedKeyError means we know this host, but it presented a DIFFERENT key.
// That is either a reinstalled server or a man-in-the-middle attack, so we
// never offer to "just accept" it.
type ChangedKeyError struct {
	Host        string
	Fingerprint string // fingerprint of the NEW key
}

func (e *ChangedKeyError) Error() string {
	return fmt.Sprintf("the host key for %s has CHANGED (new fingerprint %s). "+
		"This could be an attack, or the server was reinstalled. Connection refused.",
		e.Host, e.Fingerprint)
}

// TrustFunc is asked about a never-seen host. Return true to trust it and
// remember the key. In the app this will show a dialog to the user.
type TrustFunc func(HostKeyInfo) bool

// KnownHosts is the app's own known_hosts file (same format OpenSSH uses).
type KnownHosts struct {
	mu   sync.Mutex
	path string
}

func NewKnownHosts(path string) *KnownHosts { return &KnownHosts{path: path} }

// Callback returns a host key verifier for ssh.ClientConfig.
func (k *KnownHosts) Callback(trust TrustFunc) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := k.check(hostname, remote, key)
		if err == nil {
			return nil // known host, key matches
		}
		var ke *knownhosts.KeyError
		if !errors.As(err, &ke) {
			return err
		}
		if len(ke.Want) > 0 { // we know this host, but the key differs
			return &ChangedKeyError{Host: hostname, Fingerprint: ssh.FingerprintSHA256(key)}
		}
		// Never seen before: ask. The lock is NOT held while the user thinks.
		info := HostKeyInfo{Host: hostname, KeyType: key.Type(), Fingerprint: ssh.FingerprintSHA256(key)}
		if trust == nil || !trust(info) {
			return ErrHostKeyRejected
		}
		return k.remember(hostname, key)
	}
}

func (k *KnownHosts) check(hostname string, remote net.Addr, key ssh.PublicKey) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	f, err := os.OpenFile(k.path, os.O_CREATE|os.O_RDONLY, 0o600) // make sure it exists
	if err != nil {
		return err
	}
	f.Close()
	cb, err := knownhosts.New(k.path)
	if err != nil {
		return err
	}
	return cb(hostname, remote, key)
}

func (k *KnownHosts) remember(hostname string, key ssh.PublicKey) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	f, err := os.OpenFile(k.path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, knownhosts.Line([]string{hostname}, key))
	return err
}