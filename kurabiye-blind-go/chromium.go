package kurabiye

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// chromiumEpochToTime converts a Chromium timestamp (microseconds since 1601-01-01)
// to a Go time.Time.
func chromiumEpochToTime(chromiumTime int64) time.Time {
	if chromiumTime == 0 {
		return time.Time{}
	}
	// Chromium epoch is January 1, 1601.
	// Unix epoch is January 1, 1970.
	// Difference: 11644473600 seconds.
	unixMicro := chromiumTime - 11644473600*1000000
	return time.Unix(unixMicro/1000000, (unixMicro%1000000)*1000)
}

// chromiumSameSite converts the integer samesite value from Chromium's DB to a string.
func chromiumSameSite(val int) string {
	switch val {
	case -1:
		return "" // unspecified
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

// extractChromiumCookies extracts cookies from a Chromium-based browser.
func extractChromiumCookies(profileDir string, browserName string, domain string, path string, isSecure bool) ([]Cookie, error) {
	// Find the cookie database file
	cookieDBPath := findChromiumCookieDB(profileDir)
	if cookieDBPath == "" {
		return nil, fmt.Errorf("cookie database not found in %s", profileDir)
	}

	// Copy the database to a temp file to avoid locking issues
	tmpDir, err := os.MkdirTemp("", "kurabiye-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpDBPath := filepath.Join(tmpDir, "Cookies")
	if err := copyFile(cookieDBPath, tmpDBPath); err != nil {
		return nil, fmt.Errorf("failed to copy cookie database: %w", err)
	}

	// Also copy WAL and SHM files if they exist
	for _, suffix := range []string{"-wal", "-shm"} {
		src := cookieDBPath + suffix
		if _, err := os.Stat(src); err == nil {
			dst := tmpDBPath + suffix
			_ = copyFile(src, dst) // best effort
		}
	}

	// Open the database
	db, err := sql.Open("sqlite", tmpDBPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open cookie database: %w", err)
	}
	defer db.Close()

	// Get the decryption key
	key, err := chromiumDecryptionKey(browserName)
	if err != nil {
		return nil, fmt.Errorf("failed to get decryption key: %w", err)
	}

	// Query cookies
	rows, err := db.Query(`
		SELECT host_key, name, value, encrypted_value, path, expires_utc, 
		       is_secure, is_httponly, samesite
		FROM cookies
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query cookies: %w", err)
	}
	defer rows.Close()

	var cookies []Cookie
	for rows.Next() {
		var (
			hostKey        string
			name           string
			value          string
			encryptedValue []byte
			cookiePath     string
			expiresUTC     int64
			isSecureInt    int
			isHTTPOnlyInt  int
			sameSiteInt    int
		)

		if err := rows.Scan(&hostKey, &name, &value, &encryptedValue, &cookiePath,
			&expiresUTC, &isSecureInt, &isHTTPOnlyInt, &sameSiteInt); err != nil {
			continue // skip malformed rows
		}

		// Check domain match
		if !domainMatches(hostKey, domain) {
			continue
		}

		// Check path match
		if !pathMatches(cookiePath, path) {
			continue
		}

		// Decrypt the value if needed
		cookieValue := value
		if cookieValue == "" && len(encryptedValue) > 0 {
			decrypted, err := chromiumDecryptValue(encryptedValue, key)
			if err != nil {
				continue // skip cookies we can't decrypt
			}
			cookieValue = decrypted
		}

		cookies = append(cookies, Cookie{
			Name:     name,
			Value:    cookieValue,
			Domain:   hostKey,
			Path:     cookiePath,
			Expires:  chromiumEpochToTime(expiresUTC),
			Secure:   isSecureInt == 1,
			HTTPOnly: isHTTPOnlyInt == 1,
			SameSite: chromiumSameSite(sameSiteInt),
			Source:   browserName,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cookies: %w", err)
	}

	return cookies, nil
}

// findChromiumCookieDB finds the cookie database file in a Chromium profile directory.
// Newer versions use "Network/Cookies", older versions use "Cookies" directly.
func findChromiumCookieDB(profileDir string) string {
	// Try newer path first
	networkPath := filepath.Join(profileDir, "Network", "Cookies")
	if _, err := os.Stat(networkPath); err == nil {
		return networkPath
	}

	// Fall back to older path
	directPath := filepath.Join(profileDir, "Cookies")
	if _, err := os.Stat(directPath); err == nil {
		return directPath
	}

	return ""
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
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
