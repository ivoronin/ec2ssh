package cli

import "strings"

// ParseSSHDestination parses an SSH destination string into login, host, and port.
// Supports formats: user@host, host, ssh://user@host:port
func ParseSSHDestination(destination string) (login, host, port string) {
	if strings.HasPrefix(destination, "ssh://") {
		login, host, port = parseSSHURL(destination)
		host = StripIPv6Brackets(host)
	} else {
		login, host = parseLoginHost(destination)
	}

	return login, host, port
}

func parseSSHURL(url string) (string, string, string) {
	loginhostport := strings.TrimPrefix(url, "ssh://")
	login, hostport := parseLoginHost(loginhostport)
	host, port := parseHostPort(hostport)

	return login, host, port
}

func parseLoginHost(loginhost string) (string, string) {
	atIdx := strings.LastIndex(loginhost, "@")
	if atIdx != -1 {
		return loginhost[:atIdx], loginhost[atIdx+1:]
	}

	return "", loginhost
}

func parseHostPort(hostport string) (string, string) {
	colonIdx := strings.LastIndex(hostport, ":")
	/* handle square brackets around IPv6 addresses */
	if colonIdx != -1 && strings.LastIndex(hostport, "]") < colonIdx {
		return hostport[:colonIdx], hostport[colonIdx+1:]
	}

	return hostport, ""
}

// StripIPv6Brackets removes surrounding brackets from an IPv6 address.
// Returns the original string if no brackets are present.
func StripIPv6Brackets(host string) string {
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return host[1 : len(host)-1]
	}

	return host
}

// ParseSFTPDestination parses an SFTP destination string into login, host, port, and path.
// Supports formats: user@host:path, host:path, host, sftp://user@host:port/path
func ParseSFTPDestination(destination string) (login, host, port, path string) {
	if strings.HasPrefix(destination, "sftp://") {
		return parseSFTPURL(destination)
	}

	// user@host:path or host:path
	login, hostpath := parseLoginHost(destination)
	host, path = parseHostPath(hostpath)

	return login, host, "", path
}

func parseSFTPURL(url string) (login, host, port, path string) {
	// sftp://[user@]host[:port][/path]
	loginhostportpath := strings.TrimPrefix(url, "sftp://")

	// Split path first (after first /)
	slashIdx := strings.Index(loginhostportpath, "/")
	loginhostport := loginhostportpath
	if slashIdx != -1 {
		loginhostport = loginhostportpath[:slashIdx]
		path = loginhostportpath[slashIdx+1:]
	}

	login, hostport := parseLoginHost(loginhostport)
	host, port = parseHostPort(hostport)
	host = StripIPv6Brackets(host)

	return login, host, port, path
}

func parseHostPath(hostpath string) (string, string) {
	// For non-URL format, colon separates host:path (not port)
	// Handle IPv6: [::1]:path
	if strings.HasPrefix(hostpath, "[") {
		bracketIdx := strings.Index(hostpath, "]")
		if bracketIdx != -1 {
			host := hostpath[1:bracketIdx]
			rest := hostpath[bracketIdx+1:]
			if strings.HasPrefix(rest, ":") {
				return host, rest[1:]
			}

			return host, ""
		}
	}

	colonIdx := strings.Index(hostpath, ":")
	if colonIdx != -1 {
		return hostpath[:colonIdx], hostpath[colonIdx+1:]
	}

	return hostpath, ""
}
