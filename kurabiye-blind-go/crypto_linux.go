//go:build linux

package kurabiye

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// chromiumGetDecryptionKey retrieves the encryption key for a Chromium browser on Linux.
func chromiumGetDecryptionKey(b *chromiumBrowser) ([]byte, error) {
	password, err := secretToolLookup(b.keychainName)
	if err != nil {
		// Fallback: Chromium uses "peanuts" when no keyring is available
		password = "peanuts"
	}

	salt := []byte("saltysalt")
	key := pbkdf2.Key([]byte(password), salt, 1, 16, sha1.New)
	return key, nil
}

func secretToolLookup(application string) (string, error) {
	// Try the v2 schema first (modern Chromium)
	cmd := exec.Command("secret-tool", "lookup",
		"xdg:schema", "chrome_libsecret_os_crypt_password_v2",
		"application", application)
	out, err := cmd.Output()
	if err == nil {
		if s := strings.TrimSpace(string(out)); s != "" {
			return s, nil
		}
	}

	// Fall back to v1 schema
	cmd = exec.Command("secret-tool", "lookup",
		"xdg:schema", "chrome_libsecret_os_crypt_password_v1",
		"application", application)
	out, err = cmd.Output()
	if err == nil {
		if s := strings.TrimSpace(string(out)); s != "" {
			return s, nil
		}
	}

	// Fall back to simple lookup
	cmd = exec.Command("secret-tool", "lookup", "application", application)
	out, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("secret-tool lookup for %q failed: %w", application, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// chromiumDecryptValue decrypts an encrypted cookie value from a Chromium browser on Linux.
func chromiumDecryptValue(encryptedValue []byte, key []byte, domain string) (string, error) {
	if len(encryptedValue) == 0 {
		return "", nil
	}

	if len(encryptedValue) < 3 {
		return string(encryptedValue), nil
	}
	prefix := string(encryptedValue[:3])
	if prefix != "v10" && prefix != "v11" {
		return string(encryptedValue), nil
	}

	ciphertext := encryptedValue[3:]

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}

	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = 0x20
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	if len(ciphertext)%aes.BlockSize != 0 {
		return "", fmt.Errorf("ciphertext not multiple of block size")
	}

	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	plaintext, err = pkcs7Unpad(plaintext, aes.BlockSize)
	if err != nil {
		return "", fmt.Errorf("padding removal failed: %w", err)
	}

	// Chrome 130+ prepends SHA256(domain) to the plaintext before encrypting.
	plaintext = stripDomainHash(plaintext, domain)

	return string(plaintext), nil
}

// stripDomainHash removes the 32-byte SHA256(domain) prefix if present.
func stripDomainHash(plaintext []byte, domain string) []byte {
	if len(plaintext) <= 32 || domain == "" {
		return plaintext
	}
	domainHash := sha256.Sum256([]byte(domain))
	if string(plaintext[:32]) == string(domainHash[:]) {
		return plaintext[32:]
	}
	return plaintext
}

// pkcs7Unpad removes PKCS#7 padding.
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > blockSize || padLen > len(data) {
		return nil, fmt.Errorf("invalid padding length %d", padLen)
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if data[i] != byte(padLen) {
			return nil, fmt.Errorf("invalid padding byte at position %d", i)
		}
	}
	return data[:len(data)-padLen], nil
}
