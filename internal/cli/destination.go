package cli

import "strings"

// ParseSSHDestination parses an SSH destination string into login, host, and port.
// Supports formats: user@host, host, ssh://user@host:port
func ParseSSHDestination(destination string) (login, host, port string) {
	if strings.HasPrefix(destination, "ssh://") {
		login, host, port = parseSSHURL(destination)
		host = stripIPv6Brackets(host)
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

func stripIPv6Brackets(host string) string {
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return host[1 : len(host)-1]
	}

	return host
}
