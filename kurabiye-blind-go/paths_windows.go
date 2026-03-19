//go:build windows

package kurabiye

import (
	"os"
	"path/filepath"
)

func chromeProfileDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data", "Default")
}

func edgeProfileDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "User Data", "Default")
}

func firefoxBaseDir() string {
	return filepath.Join(os.Getenv("APPDATA"), "Mozilla", "Firefox", "Profiles")
}

func firefoxIniPath() string {
	return filepath.Join(os.Getenv("APPDATA"), "Mozilla", "Firefox", "profiles.ini")
}

func safariCookiesPath() string {
	// Safari is not available on Windows
	return ""
}

func platformDefaultBrowsers() []string {
	return []string{"chrome", "edge", "firefox"}
}
