//go:build linux

package kurabiye

import (
	"os"
	"path/filepath"
)

func chromeProfileDir() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "google-chrome", "Default")
}

func edgeProfileDir() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "microsoft-edge", "Default")
}

func firefoxBaseDir() string {
	return filepath.Join(os.Getenv("HOME"), ".mozilla", "firefox")
}

func firefoxIniPath() string {
	return filepath.Join(os.Getenv("HOME"), ".mozilla", "firefox", "profiles.ini")
}

func safariCookiesPath() string {
	// Safari is not available on Linux
	return ""
}

func platformDefaultBrowsers() []string {
	return []string{"chrome", "edge", "firefox"}
}
