package kurabiye

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// extractFirefoxCookies extracts cookies from Firefox.
func extractFirefoxCookies(domain string, path string, isSecure bool) ([]Cookie, error) {
	profileDir, err := findFirefoxProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to find Firefox profile: %w", err)
	}

	cookieDBPath := filepath.Join(profileDir, "cookies.sqlite")
	if _, err := os.Stat(cookieDBPath); err != nil {
		return nil, fmt.Errorf("Firefox cookie database not found at %s", cookieDBPath)
	}

	// Copy to temp to avoid locking issues
	tmpDir, err := os.MkdirTemp("", "kurabiye-ff-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpDBPath := filepath.Join(tmpDir, "cookies.sqlite")
	if err := copyFirefoxDB(cookieDBPath, tmpDBPath); err != nil {
		return nil, fmt.Errorf("failed to copy cookie database: %w", err)
	}

	// Open the database
	db, err := sql.Open("sqlite", tmpDBPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open cookie database: %w", err)
	}
	defer db.Close()

	// Query cookies — Firefox stores values in plaintext
	rows, err := db.Query(`
		SELECT host, name, value, path, expiry, isSecure, isHttpOnly, sameSite
		FROM moz_cookies
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query cookies: %w", err)
	}
	defer rows.Close()

	var cookies []Cookie
	for rows.Next() {
		var (
			host        string
			name        string
			value       string
			cookiePath  string
			expiry      int64
			isSecureInt int
			isHTTPOnly  int
			sameSiteInt int
		)

		if err := rows.Scan(&host, &name, &value, &cookiePath, &expiry,
			&isSecureInt, &isHTTPOnly, &sameSiteInt); err != nil {
			continue
		}

		// Check domain match
		if !domainMatches(host, domain) {
			continue
		}

		// Check path match
		if !pathMatches(cookiePath, path) {
			continue
		}

		cookies = append(cookies, Cookie{
			Name:     name,
			Value:    value,
			Domain:   host,
			Path:     cookiePath,
			Expires:  firefoxExpiryToTime(expiry),
			Secure:   isSecureInt == 1,
			HTTPOnly: isHTTPOnly == 1,
			SameSite: firefoxSameSite(sameSiteInt),
			Source:   "firefox",
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cookies: %w", err)
	}

	return cookies, nil
}

// firefoxExpiryToTime converts a Firefox expiry timestamp (Unix seconds) to time.Time.
func firefoxExpiryToTime(expiry int64) time.Time {
	if expiry == 0 {
		return time.Time{} // session cookie
	}
	return time.Unix(expiry, 0)
}

// firefoxSameSite converts Firefox's sameSite integer to a string.
// Firefox: 0=None, 1=Lax, 2=Strict
func firefoxSameSite(val int) string {
	switch val {
	case 0:
		return "None"
	case 1:
		return "Lax"
	case 2:
		return "Strict"
	default:
		return ""
	}
}

// findFirefoxProfile locates the default Firefox profile directory.
func findFirefoxProfile() (string, error) {
	// First try to parse profiles.ini
	iniPath := firefoxIniPath()
	if profile, err := parseFirefoxProfilesIni(iniPath); err == nil {
		return profile, nil
	}

	// Fallback: glob for profile directories
	baseDir := firefoxBaseDir()
	patterns := []string{
		filepath.Join(baseDir, "*.default-release"),
		filepath.Join(baseDir, "*.default"),
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err == nil && len(matches) > 0 {
			return matches[0], nil
		}
	}

	return "", fmt.Errorf("no Firefox profile found in %s", baseDir)
}

// parseFirefoxProfilesIni parses profiles.ini to find the default profile path.
func parseFirefoxProfilesIni(iniPath string) (string, error) {
	f, err := os.Open(iniPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	baseDir := filepath.Dir(iniPath)
	scanner := bufio.NewScanner(f)

	var currentPath string
	var currentIsRelative bool
	var currentIsDefault bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "[") {
			// If previous section was the default profile, return it
			if currentIsDefault && currentPath != "" {
				if currentIsRelative {
					return filepath.Join(baseDir, currentPath), nil
				}
				return currentPath, nil
			}
			// Reset for new section
			currentPath = ""
			currentIsRelative = false
			currentIsDefault = false
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Path":
			currentPath = val
		case "IsRelative":
			currentIsRelative = val == "1"
		case "Default":
			currentIsDefault = val == "1"
		}
	}

	// Check last section
	if currentIsDefault && currentPath != "" {
		if currentIsRelative {
			return filepath.Join(baseDir, currentPath), nil
		}
		return currentPath, nil
	}

	return "", fmt.Errorf("no default profile found in %s", iniPath)
}

// copyFirefoxDB copies the Firefox cookie database and associated files.
func copyFirefoxDB(src, dst string) error {
	if err := copyFileFF(src, dst); err != nil {
		return err
	}

	// Also copy WAL and SHM files if they exist
	for _, suffix := range []string{"-wal", "-shm"} {
		srcFile := src + suffix
		if _, err := os.Stat(srcFile); err == nil {
			_ = copyFileFF(srcFile, dst+suffix)
		}
	}

	return nil
}

// copyFileFF copies a single file.
func copyFileFF(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
