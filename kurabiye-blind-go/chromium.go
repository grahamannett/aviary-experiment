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

// chromiumBrowser holds configuration for a specific Chromium-based browser.
type chromiumBrowser struct {
	browserName    string
	cookiePaths    []string
	keychainName   string // macOS keychain service name, or Linux app name
	localStatePath string // Windows Local State path
}

func (b *chromiumBrowser) name() string {
	return b.browserName
}

func (b *chromiumBrowser) getCookies(host string) ([]Cookie, error) {
	dbPath, err := b.findCookieDB()
	if err != nil {
		return nil, err
	}

	// Copy DB to temp file to avoid lock issues
	tmpFile, err := copyToTemp(dbPath)
	if err != nil {
		return nil, fmt.Errorf("copying cookie db: %w", err)
	}
	defer os.Remove(tmpFile)

	// Also copy WAL and SHM files if they exist
	for _, ext := range []string{"-wal", "-shm"} {
		src := dbPath + ext
		if _, err := os.Stat(src); err == nil {
			dst := tmpFile + ext
			if err := copyFile(src, dst); err == nil {
				defer os.Remove(dst)
			}
		}
	}

	key, err := b.getDecryptionKey()
	if err != nil {
		return nil, fmt.Errorf("getting decryption key: %w", err)
	}

	return b.readCookies(tmpFile, host, key)
}

func (b *chromiumBrowser) findCookieDB() (string, error) {
	for _, p := range b.cookiePaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("%s cookie database not found", b.browserName)
}

func (b *chromiumBrowser) getDecryptionKey() ([]byte, error) {
	return chromiumGetDecryptionKey(b)
}

func (b *chromiumBrowser) readCookies(dbPath, host string, key []byte) ([]Cookie, error) {
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT host_key, name, value, encrypted_value, path,
		       expires_utc, is_secure, is_httponly, samesite, has_expires
		FROM cookies
	`)
	if err != nil {
		return nil, fmt.Errorf("querying cookies: %w", err)
	}
	defer rows.Close()

	var cookies []Cookie
	for rows.Next() {
		var (
			hostKey        string
			name           string
			value          string
			encryptedValue []byte
			path           string
			expiresUTC     int64
			isSecure       int
			isHTTPOnly     int
			sameSite       int
			hasExpires     int
		)
		err := rows.Scan(&hostKey, &name, &value, &encryptedValue, &path,
			&expiresUTC, &isSecure, &isHTTPOnly, &sameSite, &hasExpires)
		if err != nil {
			continue // skip malformed rows
		}

		if !domainMatches(host, hostKey) {
			continue
		}

		// Decrypt value if needed
		cookieValue := value
		if cookieValue == "" && len(encryptedValue) > 0 {
			decrypted, err := chromiumDecryptValue(encryptedValue, key, hostKey)
			if err != nil {
				continue // skip cookies we can't decrypt
			}
			cookieValue = decrypted
		}

		var expires time.Time
		if hasExpires != 0 && expiresUTC != 0 {
			expires = chromiumTimestampToTime(expiresUTC)
		}

		cookies = append(cookies, Cookie{
			Name:     name,
			Value:    cookieValue,
			Domain:   hostKey,
			Path:     path,
			Expires:  expires,
			Secure:   isSecure != 0,
			HTTPOnly: isHTTPOnly != 0,
			SameSite: chromiumSameSite(sameSite),
			Source:   b.browserName,
		})
	}

	return cookies, rows.Err()
}

// chromiumTimestampToTime converts a Chromium timestamp (microseconds since 1601-01-01)
// to a Go time.Time.
func chromiumTimestampToTime(usec int64) time.Time {
	if usec == 0 {
		return time.Time{}
	}
	// Chromium epoch: 1601-01-01 00:00:00 UTC
	// Unix epoch: 1970-01-01 00:00:00 UTC
	// Difference: 11644473600 seconds = 11644473600000000 microseconds
	const chromiumEpochDelta = 11644473600000000
	unixUsec := usec - chromiumEpochDelta
	return time.Unix(unixUsec/1000000, (unixUsec%1000000)*1000)
}

// chromiumSameSite converts Chromium's integer samesite to a string.
func chromiumSameSite(v int) string {
	switch v {
	case -1:
		return "None"
	case 0:
		return "" // unspecified
	case 1:
		return "Lax"
	case 2:
		return "Strict"
	default:
		return ""
	}
}

// copyToTemp copies a file to a temporary location and returns the temp path.
func copyToTemp(src string) (string, error) {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("kurabiye_%d_%s", os.Getpid(), filepath.Base(src)))
	if err := copyFile(src, tmpFile); err != nil {
		return "", err
	}
	return tmpFile, nil
}

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
