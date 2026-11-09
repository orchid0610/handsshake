// Package vault stores SSH hosts and their secrets in one encrypted file.
package vault

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

var (
	ErrLocked        = errors.New("vault is locked")
	ErrWrongPassword = errors.New("wrong password, or the vault file is corrupted")
	ErrNotFound      = errors.New("host not found")
)

type AuthType string

const (
	AuthPassword AuthType = "password"
	AuthKey      AuthType = "key"
)

// Host is one saved connection, including its secrets.
type Host struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Address    string   `json:"address"` // host:port
	User       string   `json:"user"`
	Auth       AuthType `json:"auth"`
	Password   string   `json:"password,omitempty"`
	KeyPath    string   `json:"keyPath,omitempty"`    // key stored on disk...
	PrivateKey string   `json:"privateKey,omitempty"` // ...or imported into the vault
	Passphrase string   `json:"passphrase,omitempty"`
}

type data struct {
	Hosts []Host `json:"hosts"`
}

type kdfParams struct {
	Time    uint32 `json:"time"`
	Memory  uint32 `json:"memory"` // KiB
	Threads uint8  `json:"threads"`
}

// envelope is what actually sits on disk: everything needed to
// decrypt EXCEPT the password.
type envelope struct {
	Version int       `json:"version"`
	KDF     string    `json:"kdf"`
	Params  kdfParams `json:"params"`
	Salt    []byte    `json:"salt"`
	Nonce   []byte    `json:"nonce"`
	Data    []byte    `json:"data"` // encrypted JSON of `data`
}

// aad ("additional authenticated data") binds the plaintext header to the
// ciphertext, so tampering with the KDF settings or salt is detected too.
func (e *envelope) aad() []byte {
	h := fmt.Sprintf("sshgui-vault|v%d|%s|t=%d|m=%d|p=%d|",
		e.Version, e.KDF, e.Params.Time, e.Params.Memory, e.Params.Threads)
	return append([]byte(h), e.Salt...)
}

type Vault struct {
	mu   sync.Mutex
	path string
	key  []byte // nil while locked
	env  envelope
	data data
}

func New(path string) *Vault { return &Vault{path: path} }

// DefaultPath is ~/.config/sshgui/vault.json on Linux.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "sshgui")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "vault.json"), nil
}

func (v *Vault) Exists() bool {
	_, err := os.Stat(v.path)
	return err == nil
}

func (v *Vault) IsUnlocked() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.key != nil
}

// Create makes a new empty vault protected by the given master password.
func (v *Vault) Create(password string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.Exists() {
		return errors.New("a vault already exists")
	}
	if len(password) < 8 {
		return errors.New("master password must be at least 8 characters")
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	v.env = envelope{
		Version: 1,
		KDF:     "argon2id",
		Params:  kdfParams{Time: 3, Memory: 64 * 1024, Threads: 4},
		Salt:    salt,
	}
	v.key = derive(password, v.env)
	v.data = data{}
	return v.save()
}

// Unlock decrypts the vault into memory.
func (v *Vault) Unlock(password string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	raw, err := os.ReadFile(v.path)
	if err != nil {
		return err
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("vault file is damaged: %w", err)
	}
	// Don't trust the file blindly: absurd KDF settings could freeze the app.
	p := env.Params
	if env.Version != 1 || env.KDF != "argon2id" ||
		p.Time == 0 || p.Time > 20 || p.Threads == 0 ||
		p.Memory < 8*1024 || p.Memory > 1<<21 {
		return errors.New("unsupported or corrupted vault settings")
	}

	key := derive(password, env)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		wipe(key)
		return err
	}
	// Open PANICS on a wrong-sized nonce, so check it ourselves.
	if len(env.Nonce) != aead.NonceSize() {
		wipe(key)
		return ErrWrongPassword
	}
	plain, err := aead.Open(nil, env.Nonce, env.Data, env.aad())
	if err != nil {
		wipe(key)
		return ErrWrongPassword
	}
	defer wipe(plain)

	var d data
	if err := json.Unmarshal(plain, &d); err != nil {
		wipe(key)
		return fmt.Errorf("vault contents are damaged: %w", err)
	}
	v.env, v.key, v.data = env, key, d
	return nil
}

// Lock forgets the key and all decrypted data.
func (v *Vault) Lock() {
	v.mu.Lock()
	defer v.mu.Unlock()
	wipe(v.key)
	v.key = nil
	v.data = data{}
}

// ---- Hosts ----

func (v *Vault) Hosts() ([]Host, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.key == nil {
		return nil, ErrLocked
	}
	out := make([]Host, len(v.data.Hosts))
	copy(out, v.data.Hosts)
	return out, nil
}

func (v *Vault) Get(id string) (Host, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.key == nil {
		return Host{}, ErrLocked
	}
	if i := v.index(id); i >= 0 {
		return v.data.Hosts[i], nil
	}
	return Host{}, ErrNotFound
}

// Put adds a host (empty ID) or replaces an existing one, and returns its ID.
func (v *Vault) Put(h Host) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.key == nil {
		return "", ErrLocked
	}
	if h.ID == "" {
		h.ID = newID()
		v.data.Hosts = append(v.data.Hosts, h)
	} else {
		i := v.index(h.ID)
		if i < 0 {
			return "", ErrNotFound
		}
		v.data.Hosts[i] = h
	}
	return h.ID, v.save()
}

func (v *Vault) Delete(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.key == nil {
		return ErrLocked
	}
	i := v.index(id)
	if i < 0 {
		return ErrNotFound
	}
	v.data.Hosts = append(v.data.Hosts[:i], v.data.Hosts[i+1:]...)
	return v.save()
}

// ---- internals (callers must hold v.mu) ----

func (v *Vault) index(id string) int {
	for i := range v.data.Hosts {
		if v.data.Hosts[i].ID == id {
			return i
		}
	}
	return -1
}

func (v *Vault) save() error {
	plain, err := json.Marshal(v.data)
	if err != nil {
		return err
	}
	defer wipe(plain)

	aead, err := chacha20poly1305.NewX(v.key)
	if err != nil {
		return err
	}
	// A fresh random nonce for EVERY save. Reusing one would break the cipher.
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	v.env.Nonce = nonce
	v.env.Data = aead.Seal(nil, nonce, plain, v.env.aad())

	out, err := json.MarshalIndent(v.env, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(v.path, out)
}

func derive(password string, e envelope) []byte {
	return argon2.IDKey([]byte(password), e.Salt,
		e.Params.Time, e.Params.Memory, e.Params.Threads, chacha20poly1305.KeySize)
}

// writeAtomic writes to a temp file and renames it over the target, so a
// crash mid-write can never leave you with a half-written (lost) vault.
func writeAtomic(path string, b []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".vault-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // harmless after a successful rename
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
}