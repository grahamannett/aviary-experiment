package kurabiye

// extractChromeCookies extracts cookies from the Google Chrome browser.
func extractChromeCookies(domain string, path string, isSecure bool) ([]Cookie, error) {
	profileDir := chromeProfileDir()
	return extractChromiumCookies(profileDir, "chrome", domain, path, isSecure)
}
