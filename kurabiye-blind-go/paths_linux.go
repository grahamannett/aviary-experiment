//go:build linux

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
	base := filepath.Join(homeDir(), ".config", "google-chrome")
	return []string{
		filepath.Join(base, "Default", "Cookies"),
		filepath.Join(base, "Default", "Network", "Cookies"),
		filepath.Join(base, "Profile 1", "Cookies"),
		filepath.Join(base, "Profile 1", "Network", "Cookies"),
	}
}

func chromeLocalStatePath() string {
	return filepath.Join(homeDir(), ".config", "google-chrome", "Local State")
}

func edgeCookiePaths() []string {
	base := filepath.Join(homeDir(), ".config", "microsoft-edge")
	return []string{
		filepath.Join(base, "Default", "Cookies"),
		filepath.Join(base, "Default", "Network", "Cookies"),
		filepath.Join(base, "Profile 1", "Cookies"),
		filepath.Join(base, "Profile 1", "Network", "Cookies"),
	}
}

func edgeLocalStatePath() string {
	return filepath.Join(homeDir(), ".config", "microsoft-edge", "Local State")
}

func firefoxProfileDir() string {
	return filepath.Join(homeDir(), ".mozilla", "firefox")
}

func safariCookiePaths() []string {
	return nil // Safari not supported on Linux
}

func chromeKeychainName() string { return "chrome" }
func edgeKeychainName() string   { return "chromium" }

func platformDefaultBrowsers() []string {
	return []string{"chrome", "firefox", "edge"}
}
