package sshconn

import (
	"errors"
	"strings"
	"time"
)

// OSInfo describes the operating system of the remote machine.
type OSInfo struct {
	ID   string `json:"id"`   // lowercase: "ubuntu", "debian", "macos"... ("" if unknown)
	Name string `json:"name"` // human-friendly, e.g. "Ubuntu 24.04.3 LTS"
	Like string `json:"like"` // related families from ID_LIKE, e.g. "debian"
}

// What we ask the server. Both parts tolerate being missing.
const osProbe = "cat /etc/os-release 2>/dev/null; uname -s 2>/dev/null"

// DetectOS asks the server what it runs. It opens a separate channel on the
// existing connection, so it never disturbs the interactive shell.
func (c *Conn) DetectOS() (OSInfo, error) {
	s, err := c.client.NewSession()
	if err != nil {
		return OSInfo{}, err
	}
	defer s.Close() // also unblocks the goroutine below on timeout

	type result struct {
		out []byte
		err error
	}
	done := make(chan result, 1)
	go func() {
		out, err := s.Output(osProbe)
		done <- result{out, err}
	}()

	select {
	case r := <-done:
		if len(r.out) == 0 {
			if r.err == nil {
				r.err = errors.New("empty response")
			}
			return OSInfo{}, r.err
		}
		return parseOSInfo(string(r.out)), nil
	case <-time.After(5 * time.Second):
		return OSInfo{}, errors.New("OS detection timed out")
	}
}

// Systems without /etc/os-release identify themselves via uname -s.
var kernels = map[string]OSInfo{
	"darwin":  {ID: "macos", Name: "macOS"},
	"freebsd": {ID: "freebsd", Name: "FreeBSD"},
	"openbsd": {ID: "openbsd", Name: "OpenBSD"},
	"netbsd":  {ID: "netbsd", Name: "NetBSD"},
	"linux":   {ID: "linux", Name: "Linux"},
}

// parseOSInfo reads the probe output: KEY=value lines from os-release,
// then (usually) one bare line from uname.
func parseOSInfo(out string) OSInfo {
	kv := map[string]string{}
	kernel := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok && isEnvKey(k) {
			kv[k] = strings.Trim(v, `"'`)
		} else {
			kernel = line
		}
	}

	id := strings.ToLower(kv["ID"])
	if id == "" {
		if k, ok := kernels[strings.ToLower(kernel)]; ok {
			return k
		}
		return OSInfo{Name: kernel}
	}
	name := kv["PRETTY_NAME"]
	if name == "" {
		name = kv["NAME"]
	}
	if name == "" {
		name = id
	}
	return OSInfo{ID: id, Name: name, Like: strings.ToLower(kv["ID_LIKE"])}
}

func isEnvKey(k string) bool {
	if k == "" {
		return false
	}
	for _, r := range k {
		if !(r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_') {
			return false
		}
	}
	return true
}