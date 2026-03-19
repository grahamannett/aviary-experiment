//go:build windows

package kurabiye

import (
	"os"
	"path/filepath"
)

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func localAppData() string {
	if v := os.Getenv("LOCALAPPDATA"); v != "" {
		return v
	}
	return filepath.Join(homeDir(), "AppData", "Local")
}

func appData() string {
	if v := os.Getenv("APPDATA"); v != "" {
		return v
	}
	return filepath.Join(homeDir(), "AppData", "Roaming")
}

func chromeCookiePaths() []string {
	base := filepath.Join(localAppData(), "Google", "Chrome", "User Data")
	return []string{
		filepath.Join(base, "Default", "Network", "Cookies"),
		filepath.Join(base, "Default", "Cookies"),
		filepath.Join(base, "Profile 1", "Network", "Cookies"),
		filepath.Join(base, "Profile 1", "Cookies"),
	}
}

func chromeLocalStatePath() string {
	return filepath.Join(localAppData(), "Google", "Chrome", "User Data", "Local State")
}

func edgeCookiePaths() []string {
	base := filepath.Join(localAppData(), "Microsoft", "Edge", "User Data")
	return []string{
		filepath.Join(base, "Default", "Network", "Cookies"),
		filepath.Join(base, "Default", "Cookies"),
		filepath.Join(base, "Profile 1", "Network", "Cookies"),
		filepath.Join(base, "Profile 1", "Cookies"),
	}
}

func edgeLocalStatePath() string {
	return filepath.Join(localAppData(), "Microsoft", "Edge", "User Data", "Local State")
}

func firefoxProfileDir() string {
	return filepath.Join(appData(), "Mozilla", "Firefox")
}

func safariCookiePaths() []string {
	return nil // Safari not supported on Windows
}

func chromeKeychainName() string { return "" }
func edgeKeychainName() string   { return "" }

func platformDefaultBrowsers() []string {
	return []string{"chrome", "edge", "firefox"}
}
