package ssh

import (
	"fmt"
	"strings"
)

// SSHTarget is the interface for SSH destinations.
type SSHTarget interface {
	sshTarget()
	Login() string
	Host() string
	Port() string
	SetHost(string)
	String() string
}

// sshURLTarget represents ssh://[user@]host[:port]
type sshURLTarget struct {
	user     string
	hostname string
	port     string
}

func (t *sshURLTarget) sshTarget()       {}
func (t *sshURLTarget) Login() string    { return t.user }
func (t *sshURLTarget) Host() string     { return t.hostname }
func (t *sshURLTarget) Port() string     { return t.port }
func (t *sshURLTarget) SetHost(h string) {
	if h == "" {
		panic("SetHost: empty host")
	}
	t.hostname = h
}
func (t *sshURLTarget) String() string {
	s := "ssh://"
	if t.user != "" {
		s += t.user + "@"
	}
	s += t.hostname
	if t.port != "" {
		s += ":" + t.port
	}
	return s
}

// sshSimpleTarget represents [user@]host (split at last @, mimics OpenSSH)
type sshSimpleTarget struct {
	user     string
	hostname string
}

func (t *sshSimpleTarget) sshTarget()       {}
func (t *sshSimpleTarget) Login() string    { return t.user }
func (t *sshSimpleTarget) Host() string     { return t.hostname }
func (t *sshSimpleTarget) Port() string     { return "" }
func (t *sshSimpleTarget) SetHost(h string) {
	if h == "" {
		panic("SetHost: empty host")
	}
	t.hostname = h
}
func (t *sshSimpleTarget) String() string {
	if t.user != "" {
		return t.user + "@" + t.hostname
	}
	return t.hostname
}

// NewSSHTarget parses an SSH destination string into an SSHTarget.
//
// Supported formats:
//   - Simple: [user@]hostname
//   - URL: ssh://[user@]hostname[:port]
//
// Returns ErrTarget if the hostname is empty.
func NewSSHTarget(s string) (SSHTarget, error) {
	// URL format: ssh://[user@]host[:port]
	if strings.HasPrefix(s, "ssh://") {
		s = strings.TrimPrefix(s, "ssh://")
		user, rest, _ := splitUserRest(s)
		host, port, _ := splitHostRest(rest)
		if host == "" {
			return nil, fmt.Errorf("%w: missing hostname", ErrTarget)
		}
		return &sshURLTarget{user: user, hostname: host, port: port}, nil
	}
	// Simple format: [user@]host
	user, host, _ := splitUserRest(s)
	if host == "" {
		return nil, fmt.Errorf("%w: missing hostname", ErrTarget)
	}
	return &sshSimpleTarget{user: user, hostname: host}, nil
}
