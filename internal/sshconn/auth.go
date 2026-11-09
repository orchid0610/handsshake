// Package sshconn dials SSH servers and opens interactive shells.
// It knows nothing about the vault or the UI.
package sshconn

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/ssh"
)

var (
	ErrNoCredentials      = errors.New("no password or private key provided")
	ErrPassphraseRequired = errors.New("this key is protected by a passphrase")
	ErrBadPassphrase      = errors.New("could not unlock the key: wrong passphrase or unsupported key")
	ErrAuthFailed         = errors.New("authentication failed: check the username, password or key")
)

// Credentials is whatever we have to prove who we are. A private key takes
// precedence over a password.
type Credentials struct {
	Password   string
	PrivateKey []byte // PEM / OpenSSH format
	Passphrase string // for encrypted keys
}

func (c Credentials) authMethods() ([]ssh.AuthMethod, error) {
	switch {
	case len(c.PrivateKey) > 0:
		signer, err := parseKey(c.PrivateKey, c.Passphrase)
		if err != nil {
			return nil, err
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil

	case c.Password != "":
		return []ssh.AuthMethod{
			ssh.Password(c.Password),
			// Many servers only offer "keyboard-interactive" for passwords.
			ssh.KeyboardInteractive(func(_, _ string, questions []string, _ []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = c.Password
				}
				return answers, nil
			}),
		}, nil
	}
	return nil, ErrNoCredentials
}

// parseKey tries the key without a passphrase first, and only asks for one
// if the key turns out to be encrypted.
func parseKey(pem []byte, passphrase string) (ssh.Signer, error) {
	signer, err := ssh.ParsePrivateKey(pem)
	if err == nil {
		return signer, nil
	}
	var missing *ssh.PassphraseMissingError
	if !errors.As(err, &missing) {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	if passphrase == "" {
		return nil, ErrPassphraseRequired
	}
	signer, err = ssh.ParsePrivateKeyWithPassphrase(pem, []byte(passphrase))
	if err != nil {
		return nil, ErrBadPassphrase
	}
	return signer, nil
}