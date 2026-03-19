// Package kurabiye extracts HTTP cookies from locally installed web browsers.
package kurabiye

import (
	"fmt"
	"net/url"
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

// GetCookiesOptions configures cookie extraction.
type GetCookiesOptions struct {
	URL      string   // required — base URL for origin/domain filtering
	Names    []string // optional — only return cookies matching these names
	Browsers []string // which browsers to try: "chrome", "edge", "firefox", "safari"
	Mode     string   // "merge" (default) or "first"
}

// GetCookiesResult contains extracted cookies and any warnings.
type GetCookiesResult struct {
	Cookies  []Cookie `json:"cookies"`
	Warnings []string `json:"warnings"`
}

// browserExtractor is the signature for functions that extract cookies for a specific browser.
type browserExtractor func(domain string, path string, isSecure bool) ([]Cookie, error)

// GetCookies extracts cookies from the user's locally installed browsers.
func GetCookies(opts GetCookiesOptions) (*GetCookiesResult, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("kurabiye: URL is required")
	}

	parsed, err := url.Parse(opts.URL)
	if err != nil {
		return nil, fmt.Errorf("kurabiye: invalid URL: %w", err)
	}

	domain := parsed.Hostname()
	if domain == "" {
		return nil, fmt.Errorf("kurabiye: could not extract hostname from URL")
	}

	cookiePath := parsed.Path
	if cookiePath == "" {
		cookiePath = "/"
	}

	isSecure := parsed.Scheme == "https"

	mode := opts.Mode
	if mode == "" {
		mode = "merge"
	}

	browsers := opts.Browsers
	if len(browsers) == 0 {
		browsers = defaultBrowsers()
	}

	extractors := map[string]browserExtractor{
		"chrome":  extractChromeCookies,
		"edge":    extractEdgeCookies,
		"firefox": extractFirefoxCookies,
		"safari":  extractSafariCookies,
	}

	result := &GetCookiesResult{}

	for _, browser := range browsers {
		extractor, ok := extractors[strings.ToLower(browser)]
		if !ok {
			result.Warnings = append(result.Warnings, fmt.Sprintf("unknown browser: %s", browser))
			continue
		}

		cookies, err := extractor(domain, cookiePath, isSecure)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", browser, err))
			continue
		}

		// Filter expired cookies
		now := time.Now()
		var valid []Cookie
		for _, c := range cookies {
			// Zero time means session cookie (no expiry)
			if !c.Expires.IsZero() && c.Expires.Before(now) {
				continue
			}
			valid = append(valid, c)
		}

		// Filter by name if requested
		if len(opts.Names) > 0 {
			nameSet := make(map[string]bool, len(opts.Names))
			for _, n := range opts.Names {
				nameSet[n] = true
			}
			var filtered []Cookie
			for _, c := range valid {
				if nameSet[c.Name] {
					filtered = append(filtered, c)
				}
			}
			valid = filtered
		}

		if mode == "first" && len(valid) > 0 {
			result.Cookies = valid
			return result, nil
		}

		result.Cookies = append(result.Cookies, valid...)
	}

	return result, nil
}

// ToCookieHeader formats cookies as an HTTP Cookie header string.
// When dedupeByName is true, keep only the first occurrence of each name.
func ToCookieHeader(cookies []Cookie, dedupeByName bool) string {
	var parts []string
	seen := make(map[string]bool)

	for _, c := range cookies {
		if dedupeByName {
			if seen[c.Name] {
				continue
			}
			seen[c.Name] = true
		}
		parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
	}

	return strings.Join(parts, "; ")
}

// defaultBrowsers returns the list of browsers to try based on the OS.
func defaultBrowsers() []string {
	return platformDefaultBrowsers()
}
