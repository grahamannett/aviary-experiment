package kurabiye

import (
	"net"
	"net/url"
	"strings"
)

// parseDomain extracts the host from a URL string.
func parseDomain(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	host := u.Hostname()
	if host == "" {
		return "", &url.Error{Op: "parse", URL: rawURL, Err: url.InvalidHostError("")}
	}
	return strings.ToLower(host), nil
}

// domainMatches returns true if a cookie with the given cookieDomain should be
// sent to the given host. This follows the standard cookie domain-matching rules:
//
//   - If cookieDomain starts with ".", it's a suffix match:
//     ".example.com" matches "example.com" and "sub.example.com"
//   - If cookieDomain does not start with ".", it must match exactly.
//   - Matching is case-insensitive.
func domainMatches(host, cookieDomain string) bool {
	host = strings.ToLower(host)
	cookieDomain = strings.ToLower(cookieDomain)

	// If cookie domain starts with ".", it's a domain cookie (suffix match).
	// Otherwise it's a host-only cookie (exact match only).
	// IP addresses never match via suffix (RFC 6265).
	if strings.HasPrefix(cookieDomain, ".") {
		if net.ParseIP(host) != nil {
			return false // no suffix matching for IP addresses
		}
		cd := cookieDomain[1:] // strip leading dot
		if host == cd {
			return true
		}
		if strings.HasSuffix(host, "."+cd) {
			return true
		}
		return false
	}

	// Host-only: exact match
	return host == cookieDomain
}
