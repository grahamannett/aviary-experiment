//go:build darwin

package kurabiye

import (
	"os"
	"path/filepath"
)

func chromeProfileDir() string {
	return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Google", "Chrome", "Default")
}

func edgeProfileDir() string {
	return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Microsoft Edge", "Default")
}

func firefoxBaseDir() string {
	return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Firefox", "Profiles")
}

func firefoxIniPath() string {
	return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Firefox", "profiles.ini")
}

func safariCookiesPath() string {
	return filepath.Join(os.Getenv("HOME"), "Library", "Cookies", "Cookies.binarycookies")
}

func platformDefaultBrowsers() []string {
	return []string{"chrome", "edge", "firefox", "safari"}
}
