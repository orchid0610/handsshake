// Package keyinfo describes the SSH keys that saved hosts use. It never needs
// a passphrase, and nothing it returns contains private key material.
package keyinfo

import (
	"crypto/rsa"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/crypto/ssh"

	"handsshake/internal/vault"
)

const (
	StatusOK      = "ok"      // usable: not encrypted, or its passphrase is saved
	StatusLocked  = "locked"  // encrypted, and a host using it has no saved passphrase: connecting will fail
	StatusMissing = "missing" // the key file isn't there
	StatusInvalid = "invalid" // the file isn't a private key we can read
)

// Key is one distinct key, with every host that uses it.
type Key struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`        // file name, or "<host> key" for one stored in the vault
	Source      string   `json:"source"`      // "file" or "vault"
	Path        string   `json:"path"`        // as shown to the user (~/...); files only
	Type        string   `json:"type"`        // e.g. "ED25519", "RSA 4096"; "" if unknown
	Fingerprint string   `json:"fingerprint"` // SHA256:...; "" if it can't be determined
	Encrypted   bool     `json:"encrypted"`   // protected by a passphrase
	Status      string   `json:"status"`
	UsedBy      []string `json:"usedBy"` // host names, sorted
}

type inspected struct {
	pub       ssh.PublicKey // nil when it can't be determined
	typeName  string
	encrypted bool
	status    string // "", StatusMissing or StatusInvalid
}

// List describes every key the given hosts use. A key shared by several hosts
// (found by its fingerprint) appears once. home is used to expand "~".
func List(hosts []vault.Host, home string) []Key {
	type item struct {
		key    Key
		status string
		locked bool
		used   map[string]bool
	}
	items := map[string]*item{}
	var order []string

	for _, h := range hosts {
		if h.Auth != vault.AuthKey {
			continue
		}
		s, ok := readSource(h, home)
		if !ok {
			continue
		}

		fp := ""
		if s.ins.pub != nil {
			fp = ssh.FingerprintSHA256(s.ins.pub)
		}
		id := fp
		if id == "" {
			if s.kind == "file" {
				id = "file:" + s.path
			} else {
				id = "vault:" + h.ID
			}
		}

		it := items[id]
		if it == nil {
			it = &item{
				key: Key{
					ID: id, Name: s.name, Source: s.kind, Path: s.path,
					Type: s.ins.typeName, Fingerprint: fp, Encrypted: s.ins.encrypted,
				},
				status: s.ins.status,
				used:   map[string]bool{},
			}
			items[id] = it
			order = append(order, id)
		}
		it.used[h.Name] = true
		if s.ins.encrypted && h.Passphrase == "" {
			it.locked = true
		}
	}

	out := make([]Key, 0, len(items))
	for _, id := range order {
		it := items[id]
		k := it.key
		switch {
		case it.status != "":
			k.Status = it.status
		case it.locked:
			k.Status = StatusLocked
		default:
			k.Status = StatusOK
		}
		for name := range it.used {
			k.UsedBy = append(k.UsedBy, name)
		}
		sort.Slice(k.UsedBy, func(i, j int) bool { return strings.ToLower(k.UsedBy[i]) < strings.ToLower(k.UsedBy[j]) })
		out = append(out, k)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		if a != b {
			return a < b
		}
		return out[i].Path < out[j].Path
	})
	return out
}

type source struct {
	name, kind, path string
	ins              inspected
}

// readSource finds a host's key (inside the vault, or on disk) and inspects it.
func readSource(h vault.Host, home string) (source, bool) {
	switch {
	case h.PrivateKey != "":
		return source{name: h.Name + " key", kind: "vault", ins: inspectPEM([]byte(h.PrivateKey), "")}, true

	case h.KeyPath != "":
		full := expandHome(h.KeyPath, home)
		s := source{name: filepath.Base(full), kind: "file", path: shortPath(full, home)}
		info, err := os.Stat(full)
		switch {
		case err != nil || info.IsDir():
			s.ins = inspected{status: StatusMissing}
		case info.Size() > 1<<20:
			s.ins = inspected{status: StatusInvalid}
		default:
			data, err := os.ReadFile(full)
			if err != nil {
				s.ins = inspected{status: StatusMissing}
			} else {
				s.ins = inspectPEM(data, full+".pub")
			}
		}
		return s, true
	}
	return source{}, false
}

// inspectPEM works out a key's public half. pubPath, if set, is where a
// matching .pub file might be, used for old-format encrypted keys.
func inspectPEM(b []byte, pubPath string) inspected {
	signer, err := ssh.ParsePrivateKey(b)
	if err == nil {
		pub := signer.PublicKey()
		return inspected{pub: pub, typeName: label(pub)}
	}

	var needPass *ssh.PassphraseMissingError
	if !errors.As(err, &needPass) {
		return inspected{status: StatusInvalid}
	}

	// Encrypted. OpenSSH-format keys carry their public half unencrypted.
	res := inspected{encrypted: true, pub: needPass.PublicKey}
	if res.pub == nil && pubPath != "" {
		if data, err := os.ReadFile(pubPath); err == nil {
			if pk, _, _, _, err := ssh.ParseAuthorizedKey(data); err == nil {
				res.pub = pk
			}
		}
	}
	if res.pub != nil {
		res.typeName = label(res.pub)
	} else {
		res.typeName = pemTypeLabel(b)
	}
	return res
}

func label(pub ssh.PublicKey) string {
	switch pub.Type() {
	case ssh.KeyAlgoED25519:
		return "ED25519"
	case ssh.KeyAlgoRSA:
		if cp, ok := pub.(ssh.CryptoPublicKey); ok {
			if k, ok := cp.CryptoPublicKey().(*rsa.PublicKey); ok {
				return fmt.Sprintf("RSA %d", k.N.BitLen())
			}
		}
		return "RSA"
	case ssh.KeyAlgoECDSA256:
		return "ECDSA 256"
	case ssh.KeyAlgoECDSA384:
		return "ECDSA 384"
	case ssh.KeyAlgoECDSA521:
		return "ECDSA 521"
	case ssh.KeyAlgoSKED25519:
		return "ED25519-SK"
	case ssh.KeyAlgoSKECDSA256:
		return "ECDSA-SK"
	case "ssh-dss":
		return "DSA"
	}
	return pub.Type()
}

// pemTypeLabel is the fallback for old encrypted PEM keys: the block header names the type.
func pemTypeLabel(b []byte) string {
	block, _ := pem.Decode(b)
	if block == nil {
		return ""
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return "RSA"
	case "EC PRIVATE KEY":
		return "ECDSA"
	case "DSA PRIVATE KEY":
		return "DSA"
	}
	return ""
}

func expandHome(p, home string) string {
	switch {
	case p == "~":
		return home
	case strings.HasPrefix(p, "~/"):
		return filepath.Join(home, p[2:])
	}
	return p
}

// shortPath shows a path under the home directory as ~/...
func shortPath(p, home string) string {
	if home != "" && strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + p[len(home):]
	}
	return p
}