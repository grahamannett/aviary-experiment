package kurabiye

import "strings"

// domainMatches checks if a cookie's domain applies to the given request domain.
// Cookie domain rules per RFC 6265:
// - If cookie domain starts with ".", it matches the domain and all subdomains.
// - If cookie domain equals the request domain exactly, it matches.
// - If the request domain is a subdomain of the cookie domain, it matches.
func domainMatches(cookieDomain, requestDomain string) bool {
	// Normalize: lowercase and strip leading dot
	cookieDomain = strings.ToLower(cookieDomain)
	requestDomain = strings.ToLower(requestDomain)

	cookieDomainTrimmed := strings.TrimPrefix(cookieDomain, ".")

	// Exact match
	if cookieDomainTrimmed == requestDomain {
		return true
	}

	// Subdomain match: request domain ends with ".cookieDomain"
	if strings.HasSuffix(requestDomain, "."+cookieDomainTrimmed) {
		return true
	}

	return false
}

// pathMatches checks if a cookie's path applies to the given request path.
// Per RFC 6265 section 5.1.4:
// - The cookie path is a prefix of the request path
// - If cookie path is "/", it matches everything
// - If cookie path equals the request path, it matches
// - If cookie path is a prefix and the next char in request path is "/"
func pathMatches(cookiePath, requestPath string) bool {
	if cookiePath == "" || cookiePath == "/" {
		return true
	}

	if requestPath == "" {
		requestPath = "/"
	}

	if cookiePath == requestPath {
		return true
	}

	if strings.HasPrefix(requestPath, cookiePath) {
		// The next character must be "/" for the match to be valid
		if strings.HasSuffix(cookiePath, "/") {
			return true
		}
		if len(requestPath) > len(cookiePath) && requestPath[len(cookiePath)] == '/' {
			return true
		}
	}

	return false
}
