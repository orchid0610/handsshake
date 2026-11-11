// Package sshconfig reads the part of an OpenSSH client config that matters
// for importing hosts: Host blocks and the options inside them.
package sshconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Env supplies the user-specific values the config refers to.
type Env struct {
	Home      string // for "~", %d and relative Include paths
	LocalUser string // for %u, and the default remote user
}

// Entry is one concrete host alias with all matching options applied.
type Entry struct {
	Alias         string
	HostName      string   // after token expansion; the alias if unset
	User          string   // the local user name if unset (as OpenSSH does)
	Port          int      // 22 if unset
	IdentityFiles []string // absolute paths, in file order
	ProxyJump     string
	ProxyCommand  string
	Problem       string // set when we couldn't make sense of the entry
}

type Result struct {
	Entries  []Entry
	Warnings []string
}

type option struct {
	key  string // lowercase
	args []string
}

// block is a run of options that applies to a set of host patterns.
type block struct {
	global   bool // options before any Host line
	skip     bool // Match blocks: not supported, their options are ignored
	patterns []string
	opts     []option
}

func (b *block) applies(alias string) bool {
	if b.skip {
		return false
	}
	if b.global {
		return true
	}
	alias = strings.ToLower(alias)
	matched := false
	for _, pat := range b.patterns {
		neg := strings.HasPrefix(pat, "!")
		if neg {
			pat = pat[1:]
		}
		if !wildcard(strings.ToLower(pat), alias) {
			continue
		}
		if neg {
			return false // a matching !pattern always excludes the block
		}
		matched = true
	}
	return matched
}

type parser struct {
	env      Env
	blocks   []*block
	cur      *block
	warnings []string
}

const maxIncludeDepth = 8

// Load reads the config at path, following top-level Include lines.
// A missing file is returned as an error that satisfies os.ErrNotExist.
func Load(path string, env Env) (*Result, error) {
	p := &parser{env: env}
	p.cur = &block{global: true}
	p.blocks = []*block{p.cur}
	if err := p.parseFile(path, 0); err != nil {
		return nil, err
	}
	return p.resolve(), nil
}

func (p *parser) warn(msg string) { p.warnings = append(p.warnings, msg) }

func (p *parser) parseFile(path string, depth int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, args, ok := splitLine(line)
		if !ok {
			continue
		}
		switch key {
		case "host":
			p.cur = &block{patterns: args}
			p.blocks = append(p.blocks, p.cur)
		case "match":
			p.cur = &block{skip: true}
			p.blocks = append(p.blocks, p.cur)
		case "include":
			p.include(args, depth)
		default:
			p.cur.opts = append(p.cur.opts, option{key, args})
		}
	}
	return nil
}

func (p *parser) include(patterns []string, depth int) {
	if !p.cur.global {
		p.warn("An Include inside a Host or Match block was ignored.")
		return
	}
	if depth >= maxIncludeDepth {
		p.warn("Include files are nested too deeply; some were ignored.")
		return
	}
	saved := p.cur // Host blocks inside an included file don't leak out of it
	for _, pat := range patterns {
		pat = p.expandHome(pat)
		if !filepath.IsAbs(pat) {
			pat = filepath.Join(p.env.Home, ".ssh", pat) // OpenSSH's rule for user configs
		}
		matches, _ := filepath.Glob(pat)
		for _, m := range matches {
			if info, err := os.Stat(m); err != nil || info.IsDir() {
				continue
			}
			if err := p.parseFile(m, depth+1); err != nil {
				p.warn(fmt.Sprintf("Could not read %s: %v", m, err))
			}
		}
	}
	p.cur = saved
}

// resolve produces one Entry per concrete (non-wildcard) Host alias.
func (p *parser) resolve() *Result {
	res := &Result{}
	seen := map[string]bool{}
	for _, b := range p.blocks {
		if b.global || b.skip {
			continue
		}
		for _, pat := range b.patterns {
			key := strings.ToLower(pat)
			if strings.HasPrefix(pat, "!") || strings.ContainsAny(pat, "*?") || seen[key] {
				continue
			}
			seen[key] = true
			res.Entries = append(res.Entries, p.entryFor(pat))
		}
	}
	res.Warnings = p.warnings
	return res
}

// entryFor applies every matching block in file order. As in OpenSSH, the
// first value found for an option wins (IdentityFile accumulates instead).
func (p *parser) entryFor(alias string) Entry {
	var hostName, user, port, proxyJump, proxyCmd string
	var ids []string
	first := func(dst *string, o option) {
		if *dst == "" && len(o.args) > 0 {
			*dst = o.args[0]
		}
	}
	for _, b := range p.blocks {
		if !b.applies(alias) {
			continue
		}
		for _, o := range b.opts {
			switch o.key {
			case "hostname":
				first(&hostName, o)
			case "user":
				first(&user, o)
			case "port":
				first(&port, o)
			case "identityfile":
				if len(o.args) > 0 {
					ids = append(ids, o.args[0])
				}
			case "proxyjump":
				first(&proxyJump, o)
			case "proxycommand":
				if proxyCmd == "" && len(o.args) > 0 {
					proxyCmd = strings.Join(o.args, " ")
				}
			}
		}
	}

	e := Entry{Alias: alias, User: user, Port: 22}
	if e.User == "" {
		e.User = p.env.LocalUser
	}
	if port != "" {
		if n, err := strconv.Atoi(port); err == nil && n > 0 && n < 65536 {
			e.Port = n
		} else {
			p.warn(fmt.Sprintf("%s: invalid port %q, using 22.", alias, port))
		}
	}
	if hostName == "" {
		hostName = alias
	}
	if h, ok := expand(hostName, map[byte]string{'h': alias}); ok {
		e.HostName = h
	} else {
		e.HostName = hostName
		e.Problem = "The address uses a % placeholder that can't be resolved."
	}

	vars := map[byte]string{
		'h': e.HostName, 'p': strconv.Itoa(e.Port), 'r': e.User,
		'u': p.env.LocalUser, 'd': p.env.Home,
	}
	for _, f := range ids {
		if strings.EqualFold(f, "none") {
			continue
		}
		f, _ = expand(f, vars) // anything unresolved stays literal and shows up as a missing file
		f = p.expandHome(f)
		if !filepath.IsAbs(f) {
			f = filepath.Join(p.env.Home, ".ssh", f)
		}
		e.IdentityFiles = append(e.IdentityFiles, f)
	}
	if !strings.EqualFold(proxyJump, "none") {
		e.ProxyJump = proxyJump
	}
	if !strings.EqualFold(proxyCmd, "none") {
		e.ProxyCommand = proxyCmd
	}
	return e
}

func (p *parser) expandHome(s string) string {
	switch {
	case s == "~":
		return p.env.Home
	case strings.HasPrefix(s, "~/"):
		return filepath.Join(p.env.Home, s[2:])
	}
	return s
}

// expand replaces %x tokens. ok is false if an unknown token was left behind.
func expand(s string, vars map[byte]string) (string, bool) {
	if !strings.Contains(s, "%") {
		return s, true
	}
	var b strings.Builder
	ok := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '%' {
			b.WriteByte(c)
			continue
		}
		if i+1 >= len(s) { // a lone % at the end
			b.WriteByte(c)
			ok = false
			continue
		}
		i++
		switch n := s[i]; {
		case n == '%':
			b.WriteByte('%')
		case vars[n] != "":
			b.WriteString(vars[n])
		default:
			b.WriteByte('%')
			b.WriteByte(n)
			ok = false
		}
	}
	return b.String(), ok
}

// splitLine separates "Keyword args", "Keyword=args" and "Keyword = args".
func splitLine(line string) (key string, args []string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || line[0] == '#' {
		return "", nil, false
	}
	i := strings.IndexAny(line, " \t=")
	if i < 0 {
		return strings.ToLower(line), nil, true
	}
	key = strings.ToLower(line[:i])
	rest := strings.TrimLeft(line[i:], " \t")
	rest = strings.TrimPrefix(rest, "=")
	rest = strings.TrimLeft(rest, " \t")
	return key, tokenize(rest), true
}

// tokenize splits on whitespace, honours "double quotes", and stops at a
// # that starts a token.
func tokenize(s string) []string {
	var out []string
	var cur strings.Builder
	inTok, inQuote := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inQuote:
			switch {
			case c == '"':
				inQuote = false
			case c == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\'):
				i++
				cur.WriteByte(s[i])
			default:
				cur.WriteByte(c)
			}
		case c == '"':
			inQuote, inTok = true, true
		case c == ' ' || c == '\t':
			if inTok {
				out = append(out, cur.String())
				cur.Reset()
				inTok = false
			}
		case c == '#' && !inTok:
			return out
		default:
			inTok = true
			cur.WriteByte(c)
		}
	}
	if inTok {
		out = append(out, cur.String())
	}
	return out
}

// wildcard matches s against a pattern with * (any run) and ? (any one character).
func wildcard(p, s string) bool {
	pi, si := 0, 0
	star, mark := -1, 0
	for si < len(s) {
		switch {
		case pi < len(p) && (p[pi] == '?' || p[pi] == s[si]):
			pi++
			si++
		case pi < len(p) && p[pi] == '*':
			star, mark = pi, si
			pi++
		case star >= 0: // backtrack: let the last * swallow one more character
			pi = star + 1
			mark++
			si = mark
		default:
			return false
		}
	}
	for pi < len(p) && p[pi] == '*' {
		pi++
	}
	return pi == len(p)
}