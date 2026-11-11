// Package vault stores SSH hosts and their secrets in one encrypted file.
package vault

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"handsshake/internal/fileutil"
)

var (
	ErrLocked         = errors.New("vault is locked")
	ErrWrongPassword  = errors.New("wrong password")
	ErrBadRecoveryKey = errors.New("invalid recovery key")
	ErrNotFound       = errors.New("host not found")
)

// Each encrypted blob is bound to its purpose, so an attacker can't
// swap one blob into another slot.
const (
	purposePassword = "password-slot"
	purposeRecovery = "recovery-slot"
	purposeData     = "data"
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
	OS         string   `json:"os,omitempty"`     // detected remote OS id, e.g. "ubuntu"
	OSLike     string   `json:"osLike,omitempty"` // related families, e.g. "debian"
}

type data struct {
	Hosts []Host `json:"hosts"`
}

type kdfParams struct {
	Time    uint32 `json:"time"`
	Memory  uint32 `json:"memory"` // KiB
	Threads uint8  `json:"threads"`
}

// envelope is what sits on disk. It holds the vault key twice (wrapped by
// the password and by the recovery key) plus the encrypted data.
type envelope struct {
	Version int       `json:"version"`
	KDF     string    `json:"kdf"`
	Params  kdfParams `json:"params"`

	PwSalt  []byte `json:"pwSalt"`
	PwNonce []byte `json:"pwNonce"`
	PwWrap  []byte `json:"pwWrap"`

	RecNonce []byte `json:"recNonce"`
	RecWrap  []byte `json:"recWrap"`

	Nonce []byte `json:"nonce"`
	Data  []byte `json:"data"`
}

// aad is the "additional authenticated data" for each blob. The password
// slot also binds the KDF settings and salt, so weakening them breaks unlock
// instead of silently lowering the protection.
func (e *envelope) aad(purpose string) []byte {
	s := fmt.Sprintf("handsshake-vault|v%d|%s", e.Version, purpose)
	if purpose == purposePassword {
		s += fmt.Sprintf("|%s|t=%d|m=%d|p=%d|", e.KDF, e.Params.Time, e.Params.Memory, e.Params.Threads)
		return append([]byte(s), e.PwSalt...)
	}
	return []byte(s)
}

type Vault struct {
	mu   sync.Mutex
	path string
	key  []byte // the vault key; nil while locked
	env  envelope
	data data
}

func New(path string) *Vault { return &Vault{path: path} }

// DefaultPath is ~/.config/handsshake/vault.json on Linux.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "handsshake")
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

// Create makes a new vault and returns the recovery key. The recovery key
// is shown to the user ONCE; it is never stored anywhere in readable form.
func (v *Vault) Create(password string) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.Exists() {
		return "", errors.New("a vault already exists")
	}
	if err := checkPassword(password); err != nil {
		return "", err
	}
	vk, err := randomBytes(chacha20poly1305.KeySize)
	if err != nil {
		return "", err
	}
	rk, err := randomBytes(chacha20poly1305.KeySize)
	if err != nil {
		return "", err
	}
	defer wipe(rk)

	v.env = envelope{Version: 2, KDF: "argon2id",
		Params: kdfParams{Time: 3, Memory: 64 * 1024, Threads: 4}}
	v.key, v.data = vk, data{}

	if err := v.wrapPassword(password); err != nil {
		v.lockLocked()
		return "", err
	}
	nonce, wrap, err := seal(rk, vk, v.env.aad(purposeRecovery))
	if err != nil {
		v.lockLocked()
		return "", err
	}
	v.env.RecNonce, v.env.RecWrap = nonce, wrap

	if err := v.save(); err != nil {
		v.lockLocked()
		return "", err
	}
	return formatRecoveryKey(rk), nil
}

// Unlock opens the vault with the master password.
func (v *Vault) Unlock(password string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	env, err := v.load()
	if err != nil {
		return err
	}
	pk := derive(password, env)
	defer wipe(pk)

	vk, err := open(pk, env.PwNonce, env.PwWrap, env.aad(purposePassword))
	if err != nil {
		return ErrWrongPassword
	}
	return v.finishUnlock(env, vk)
}

// UnlockWithRecovery opens the vault with the recovery key and sets a new
// master password. The recovery key itself stays valid.
func (v *Vault) UnlockWithRecovery(recoveryKey, newPassword string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if err := checkPassword(newPassword); err != nil {
		return err
	}
	rk, err := parseRecoveryKey(recoveryKey)
	if err != nil {
		return err
	}
	defer wipe(rk)

	env, err := v.load()
	if err != nil {
		return err
	}
	vk, err := open(rk, env.RecNonce, env.RecWrap, env.aad(purposeRecovery))
	if err != nil {
		return ErrBadRecoveryKey
	}
	if err := v.finishUnlock(env, vk); err != nil {
		return err
	}
	if err := v.wrapPassword(newPassword); err != nil {
		v.lockLocked()
		return err
	}
	if err := v.save(); err != nil {
		v.lockLocked()
		return err
	}
	return nil
}

// ChangePassword re-wraps the vault key under a new master password.
func (v *Vault) ChangePassword(current, next string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.key == nil {
		return ErrLocked
	}
	if err := checkPassword(next); err != nil {
		return err
	}
	pk := derive(current, v.env)
	defer wipe(pk)
	check, err := open(pk, v.env.PwNonce, v.env.PwWrap, v.env.aad(purposePassword))
	if err != nil {
		return ErrWrongPassword
	}
	wipe(check)

	if err := v.wrapPassword(next); err != nil {
		return err
	}
	return v.save()
}

// Lock forgets the key and all decrypted data.
func (v *Vault) Lock() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.lockLocked()
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

// AddMany adds several new hosts (any IDs are replaced) and saves once.
func (v *Vault) AddMany(hosts []Host) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.key == nil {
		return ErrLocked
	}
	before := len(v.data.Hosts)
	for _, h := range hosts {
		h.ID = newID()
		v.data.Hosts = append(v.data.Hosts, h)
	}
	if err := v.save(); err != nil {
		v.data.Hosts = v.data.Hosts[:before] // don't keep what we failed to save
		return err
	}
	return nil
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

func (v *Vault) lockLocked() {
	wipe(v.key)
	v.key = nil
	v.data = data{}
}

func (v *Vault) index(id string) int {
	for i := range v.data.Hosts {
		if v.data.Hosts[i].ID == id {
			return i
		}
	}
	return -1
}

func (v *Vault) load() (envelope, error) {
	raw, err := os.ReadFile(v.path)
	if err != nil {
		return envelope{}, err
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return envelope{}, fmt.Errorf("vault file is damaged: %w", err)
	}
	// Don't trust the file blindly: absurd KDF settings could freeze the app.
	p := env.Params
	if env.Version != 2 || env.KDF != "argon2id" ||
		p.Time == 0 || p.Time > 20 || p.Threads == 0 ||
		p.Memory < 8*1024 || p.Memory > 1<<21 {
		return envelope{}, errors.New("unsupported or corrupted vault settings")
	}
	return env, nil
}

// finishUnlock decrypts the data with the vault key and, on success, keeps
// that key in memory.
func (v *Vault) finishUnlock(env envelope, vk []byte) error {
	plain, err := open(vk, env.Nonce, env.Data, env.aad(purposeData))
	if err != nil {
		wipe(vk)
		return errors.New("vault data is damaged")
	}
	defer wipe(plain)

	var d data
	if err := json.Unmarshal(plain, &d); err != nil {
		wipe(vk)
		return errors.New("vault data is damaged")
	}
	v.env, v.key, v.data = env, vk, d
	return nil
}

// wrapPassword (re)creates the password slot with a fresh salt.
func (v *Vault) wrapPassword(password string) error {
	salt, err := randomBytes(16)
	if err != nil {
		return err
	}
	v.env.PwSalt = salt
	pk := derive(password, v.env)
	defer wipe(pk)

	nonce, wrap, err := seal(pk, v.key, v.env.aad(purposePassword))
	if err != nil {
		return err
	}
	v.env.PwNonce, v.env.PwWrap = nonce, wrap
	return nil
}

func (v *Vault) save() error {
	plain, err := json.Marshal(v.data)
	if err != nil {
		return err
	}
	defer wipe(plain)

	nonce, ct, err := seal(v.key, plain, v.env.aad(purposeData))
	if err != nil {
		return err
	}
	v.env.Nonce, v.env.Data = nonce, ct

	out, err := json.MarshalIndent(v.env, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.WriteAtomic(v.path, out, 0o600)
}

// ---- crypto helpers ----

func derive(password string, e envelope) []byte {
	return argon2.IDKey([]byte(password), e.PwSalt,
		e.Params.Time, e.Params.Memory, e.Params.Threads, chacha20poly1305.KeySize)
}

// seal encrypts with a fresh random nonce. Never reuse a nonce with one key.
func seal(key, plaintext, aad []byte) (nonce, ciphertext []byte, err error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil, err
	}
	nonce, err = randomBytes(aead.NonceSize())
	if err != nil {
		return nil, nil, err
	}
	return nonce, aead.Seal(nil, nonce, plaintext, aad), nil
}

func open(key, nonce, ciphertext, aad []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	// Open PANICS on a wrong-sized nonce, so check it ourselves.
	if len(nonce) != aead.NonceSize() {
		return nil, errors.New("bad nonce")
	}
	return aead.Open(nil, nonce, ciphertext, aad)
}

func checkPassword(p string) error {
	if len(p) < 8 {
		return errors.New("master password must be at least 8 characters")
	}
	return nil
}

// ---- recovery key encoding: 32 random bytes as XXXX-XXXX-... ----

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func formatRecoveryKey(rk []byte) string {
	s := b32.EncodeToString(rk)
	var parts []string
	for i := 0; i < len(s); i += 4 {
		parts = append(parts, s[i:min(i+4, len(s))])
	}
	return strings.Join(parts, "-")
}

func parseRecoveryKey(s string) ([]byte, error) {
	s = strings.NewReplacer("-", "", " ", "", "\n", "", "\t", "").Replace(s)
	rk, err := b32.DecodeString(strings.ToUpper(s))
	if err != nil || len(rk) != chacha20poly1305.KeySize {
		return nil, ErrBadRecoveryKey
	}
	return rk, nil
}

// ---- file + misc helpers ----

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

func newID() string {
	b, _ := randomBytes(8)
	return hex.EncodeToString(b)
}

func wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
