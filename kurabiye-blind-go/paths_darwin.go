//go:build darwin

package kurabiye

import (
	"os"
	"path/filepath"
)

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func chromeCookiePaths() []string {
	base := filepath.Join(homeDir(), "Library", "Application Support", "Google", "Chrome")
	return []string{
		filepath.Join(base, "Default", "Cookies"),
		filepath.Join(base, "Default", "Network", "Cookies"),
		filepath.Join(base, "Profile 1", "Cookies"),
		filepath.Join(base, "Profile 1", "Network", "Cookies"),
	}
}

func chromeLocalStatePath() string {
	return filepath.Join(homeDir(), "Library", "Application Support", "Google", "Chrome", "Local State")
}

func edgeCookiePaths() []string {
	base := filepath.Join(homeDir(), "Library", "Application Support", "Microsoft Edge")
	return []string{
		filepath.Join(base, "Default", "Cookies"),
		filepath.Join(base, "Default", "Network", "Cookies"),
		filepath.Join(base, "Profile 1", "Cookies"),
		filepath.Join(base, "Profile 1", "Network", "Cookies"),
	}
}

func edgeLocalStatePath() string {
	return filepath.Join(homeDir(), "Library", "Application Support", "Microsoft Edge", "Local State")
}

func firefoxProfileDir() string {
	return filepath.Join(homeDir(), "Library", "Application Support", "Firefox")
}

func safariCookiePaths() []string {
	h := homeDir()
	return []string{
		filepath.Join(h, "Library", "Cookies", "Cookies.binarycookies"),
		filepath.Join(h, "Library", "Containers", "com.apple.Safari", "Data", "Library", "Cookies", "Cookies.binarycookies"),
	}
}

func chromeKeychainName() string { return "Chrome Safe Storage" }
func edgeKeychainName() string   { return "Microsoft Edge Safe Storage" }

func platformDefaultBrowsers() []string {
	return []string{"chrome", "firefox", "safari", "edge"}
}
