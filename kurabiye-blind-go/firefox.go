package kurabiye

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type firefoxBrowser struct{}

func newFirefox() (*firefoxBrowser, error) {
	return &firefoxBrowser{}, nil
}

func (b *firefoxBrowser) name() string {
	return "firefox"
}

func (b *firefoxBrowser) getCookies(host string) ([]Cookie, error) {
	profileDir, err := findFirefoxProfile()
	if err != nil {
		return nil, err
	}

	cookiesDB := filepath.Join(profileDir, "cookies.sqlite")
	if _, err := os.Stat(cookiesDB); err != nil {
		return nil, fmt.Errorf("firefox cookies.sqlite not found at %s", cookiesDB)
	}

	// Copy to temp to avoid lock issues
	tmpFile, err := copyToTemp(cookiesDB)
	if err != nil {
		return nil, fmt.Errorf("copying firefox cookie db: %w", err)
	}
	defer os.Remove(tmpFile)

	// Also copy WAL and SHM if present
	for _, ext := range []string{"-wal", "-shm"} {
		src := cookiesDB + ext
		if _, err := os.Stat(src); err == nil {
			dst := tmpFile + ext
			if err := copyFile(src, dst); err == nil {
				defer os.Remove(dst)
			}
		}
	}

	return readFirefoxCookies(tmpFile, host)
}

func readFirefoxCookies(dbPath, host string) ([]Cookie, error) {
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("opening firefox sqlite: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT host, name, value, path, expiry, isSecure, isHttpOnly, sameSite
		FROM moz_cookies
	`)
	if err != nil {
		return nil, fmt.Errorf("querying firefox cookies: %w", err)
	}
	defer rows.Close()

	var cookies []Cookie
	for rows.Next() {
		var (
			cookieHost string
			name       string
			value      string
			path       string
			expiry     int64
			isSecure   int
			isHTTPOnly int
			sameSite   int
		)
		err := rows.Scan(&cookieHost, &name, &value, &path, &expiry, &isSecure, &isHTTPOnly, &sameSite)
		if err != nil {
			continue
		}

		if !domainMatches(host, cookieHost) {
			continue
		}

		var expires time.Time
		if expiry > 0 {
			expires = time.Unix(expiry, 0)
		}

		cookies = append(cookies, Cookie{
			Name:     name,
			Value:    value,
			Domain:   cookieHost,
			Path:     path,
			Expires:  expires,
			Secure:   isSecure != 0,
			HTTPOnly: isHTTPOnly != 0,
			SameSite: firefoxSameSite(sameSite),
			Source:   "firefox",
		})
	}

	return cookies, rows.Err()
}

func firefoxSameSite(v int) string {
	switch v {
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
	baseDir := firefoxProfileDir()

	// Try profiles.ini first
	profilesIni := filepath.Join(baseDir, "profiles.ini")
	if _, err := os.Stat(profilesIni); err == nil {
		profile, err := parseProfilesIni(profilesIni, baseDir)
		if err == nil {
			return profile, nil
		}
	}

	// Fallback: glob for default-release profile
	return findFirefoxProfileByGlob(baseDir)
}

// parseProfilesIni reads profiles.ini and returns the default profile path.
func parseProfilesIni(iniPath, baseDir string) (string, error) {
	f, err := os.Open(iniPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	type profileSection struct {
		name       string
		path       string
		isRelative bool
		isDefault  bool
	}

	var profiles []profileSection
	var current *profileSection
	var installDefault string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := line[1 : len(line)-1]
			if strings.HasPrefix(section, "Profile") {
				profiles = append(profiles, profileSection{name: section})
				current = &profiles[len(profiles)-1]
			} else {
				current = nil
			}
			// Check Install sections for Default
			if strings.HasPrefix(section, "Install") {
				// The next Default= in this section is the install default
				current = nil // don't treat as profile
				// We'll capture the Default= below
			}
			continue
		}

		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			if current != nil {
				switch key {
				case "Path":
					current.path = val
				case "IsRelative":
					current.isRelative = val == "1"
				case "Default":
					current.isDefault = val == "1"
				}
			} else if key == "Default" && installDefault == "" {
				installDefault = val
			}
		}
	}

	// Prefer install default
	if installDefault != "" {
		p := resolvePath(baseDir, installDefault, true)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Fall back to profile marked as Default=1
	for _, p := range profiles {
		if p.isDefault && p.path != "" {
			resolved := resolvePath(baseDir, p.path, p.isRelative)
			if _, err := os.Stat(resolved); err == nil {
				return resolved, nil
			}
		}
	}

	// Fall back to first profile with a path
	for _, p := range profiles {
		if p.path != "" {
			resolved := resolvePath(baseDir, p.path, p.isRelative)
			if _, err := os.Stat(resolved); err == nil {
				return resolved, nil
			}
		}
	}

	return "", fmt.Errorf("no valid profile found in profiles.ini")
}

func resolvePath(baseDir, path string, isRelative bool) string {
	if isRelative {
		return filepath.Join(baseDir, path)
	}
	return path
}

func findFirefoxProfileByGlob(baseDir string) (string, error) {
	profilesDir := filepath.Join(baseDir, "Profiles")

	// Try *.default-release first (modern Firefox)
	matches, _ := filepath.Glob(filepath.Join(profilesDir, "*.default-release"))
	if len(matches) > 0 {
		return matches[0], nil
	}

	// Try *.default
	matches, _ = filepath.Glob(filepath.Join(profilesDir, "*.default"))
	if len(matches) > 0 {
		return matches[0], nil
	}

	// Try directly in baseDir (some systems put profiles there)
	matches, _ = filepath.Glob(filepath.Join(baseDir, "*.default-release"))
	if len(matches) > 0 {
		return matches[0], nil
	}

	matches, _ = filepath.Glob(filepath.Join(baseDir, "*.default"))
	if len(matches) > 0 {
		return matches[0], nil
	}

	return "", fmt.Errorf("no firefox profile directory found in %s", baseDir)
}
