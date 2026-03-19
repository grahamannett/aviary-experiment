package kurabiye

// extractEdgeCookies extracts cookies from the Microsoft Edge browser.
func extractEdgeCookies(domain string, path string, isSecure bool) ([]Cookie, error) {
	profileDir := edgeProfileDir()
	return extractChromiumCookies(profileDir, "edge", domain, path, isSecure)
}
