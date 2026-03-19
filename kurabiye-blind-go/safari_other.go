//go:build !darwin

package kurabiye

import "fmt"

// extractSafariCookies is a stub for non-macOS platforms where Safari is not available.
func extractSafariCookies(domain string, path string, isSecure bool) ([]Cookie, error) {
	return nil, fmt.Errorf("Safari is only available on macOS")
}
