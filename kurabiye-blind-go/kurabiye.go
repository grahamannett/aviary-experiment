// Package kurabiye extracts HTTP cookies from locally installed web browsers.
package kurabiye

import (
	"fmt"
	"strings"
	"time"
)

// Cookie represents an HTTP cookie extracted from a browser.
type Cookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires"`
	Secure   bool      `json:"secure"`
	HTTPOnly bool      `json:"httpOnly"`
	SameSite string    `json:"sameSite"` // "Strict", "Lax", "None", or ""
	Source   string    `json:"source"`   // which browser produced this cookie
}

// GetCookiesOptions configures which cookies to retrieve.
type GetCookiesOptions struct {
	URL      string   // required — base URL for origin/domain filtering
	Names    []string // optional — only return cookies matching these names
	Browsers []string // which browsers to try: "chrome", "edge", "firefox", "safari"
	Mode     string   // "merge" (default) or "first"
}

// GetCookiesResult holds the extracted cookies and any warnings.
type GetCookiesResult struct {
	Cookies  []Cookie `json:"cookies"`
	Warnings []string `json:"warnings"`
}

// browserBackend is the internal interface each browser implements.
type browserBackend interface {
	name() string
	getCookies(host string) ([]Cookie, error)
}

// defaultBrowsers returns the platform-appropriate default browser list.
func defaultBrowsers() []string {
	return platformDefaultBrowsers()
}

// GetCookies extracts cookies from the requested browsers matching the given URL.
func GetCookies(opts GetCookiesOptions) (*GetCookiesResult, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("kurabiye: URL is required")
	}

	host, err := parseDomain(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("kurabiye: invalid URL %q: %w", opts.URL, err)
	}

	if opts.Mode == "" {
		opts.Mode = "merge"
	}

	browsers := opts.Browsers
	if len(browsers) == 0 {
		browsers = defaultBrowsers()
	}

	nameSet := make(map[string]bool, len(opts.Names))
	for _, n := range opts.Names {
		nameSet[n] = true
	}

	result := &GetCookiesResult{}

	for _, bName := range browsers {
		backend, err := getBackend(bName)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", bName, err))
			continue
		}

		cookies, err := backend.getCookies(host)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", bName, err))
			continue
		}

		// Filter expired cookies
		now := time.Now()
		var filtered []Cookie
		for _, c := range cookies {
			if !c.Expires.IsZero() && c.Expires.Before(now) {
				continue
			}
			if len(nameSet) > 0 && !nameSet[c.Name] {
				continue
			}
			filtered = append(filtered, c)
		}

		result.Cookies = append(result.Cookies, filtered...)

		if opts.Mode == "first" && len(filtered) > 0 {
			break
		}
	}

	return result, nil
}

// getBackend returns the browser backend for the given name.
func getBackend(name string) (browserBackend, error) {
	switch strings.ToLower(name) {
	case "chrome":
		return newChrome()
	case "edge":
		return newEdge()
	case "firefox":
		return newFirefox()
	case "safari":
		return newSafari()
	default:
		return nil, fmt.Errorf("unknown browser: %q", name)
	}
}

// ToCookieHeader formats cookies as an HTTP Cookie header string.
// When dedupeByName is true, keep only the first occurrence of each name.
func ToCookieHeader(cookies []Cookie, dedupeByName bool) string {
	seen := make(map[string]bool)
	var parts []string
	for _, c := range cookies {
		if dedupeByName {
			if seen[c.Name] {
				continue
			}
			seen[c.Name] = true
		}
		parts = append(parts, c.Name+"="+c.Value)
	}
	return strings.Join(parts, "; ")
}
