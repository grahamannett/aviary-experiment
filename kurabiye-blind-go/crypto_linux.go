//go:build linux

package kurabiye

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// chromiumDecryptionKey retrieves the encryption key for a Chromium-based browser on Linux.
// It tries GNOME Keyring first via secret-tool, then falls back to the hardcoded "peanuts" password.
func chromiumDecryptionKey(browser string) ([]byte, error) {
	var password string

	// Try GNOME Keyring via secret-tool
	var application string
	switch browser {
	case "chrome":
		application = "chrome"
	case "edge":
		application = "chromium" // Edge on Linux often uses "chromium"
	default:
		application = "chromium"
	}

	cmd := exec.Command("secret-tool", "lookup", "application", application)
	out, err := cmd.Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		password = strings.TrimSpace(string(out))
	} else {
		// Fallback to hardcoded password
		password = "peanuts"
	}

	// Derive AES-128 key using PBKDF2
	// Salt: "saltysalt", Iterations: 1 (Linux uses 1, not 1003 like macOS)
	salt := []byte("saltysalt")
	key := pbkdf2.Key([]byte(password), salt, 1, 16, sha1.New)

	return key, nil
}

// chromiumDecryptValue decrypts a Chromium encrypted cookie value on Linux.
// Supports two formats:
//   - Legacy: v10/v11 (3 bytes) + AES-128-CBC ciphertext (IV = 16 bytes of 0x20)
//   - Modern: v10/v11 (3 bytes) + 16-byte header + 16-byte IV + AES-128-CBC ciphertext
func chromiumDecryptValue(encryptedValue []byte, key []byte) (string, error) {
	if len(encryptedValue) == 0 {
		return "", nil
	}

	// Check for v10 or v11 prefix
	if len(encryptedValue) < 3 {
		return string(encryptedValue), nil
	}

	prefix := string(encryptedValue[:3])
	if prefix != "v10" && prefix != "v11" {
		// Not encrypted, return as-is
		return string(encryptedValue), nil
	}

	// Strip prefix
	data := encryptedValue[3:]

	if len(data) == 0 {
		return "", nil
	}

	// Try modern format first: 16-byte header + 16-byte IV + ciphertext
	if len(data) >= 48 {
		ct := data[32:]
		if len(ct)%aes.BlockSize == 0 && len(ct) > 0 {
			iv := data[16:32]
			plaintext, err := decryptCBCLinux(ct, key, iv)
			if err == nil && isValidDecryptedValueLinux(plaintext) {
				return string(plaintext), nil
			}
		}
	}

	// Fall back to legacy format: IV = 16 bytes of 0x20
	if len(data)%aes.BlockSize == 0 {
		iv := make([]byte, aes.BlockSize)
		for i := range iv {
			iv[i] = 0x20
		}
		plaintext, err := decryptCBCLinux(data, key, iv)
		if err != nil {
			return "", fmt.Errorf("decryption failed: %w", err)
		}
		return string(plaintext), nil
	}

	return "", fmt.Errorf("unexpected ciphertext length: %d", len(data))
}

// decryptCBCLinux decrypts data using AES-128-CBC and removes PKCS#7 padding.
func decryptCBCLinux(ciphertext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext not block-aligned")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// Remove PKCS#7 padding
	plaintext, err = removePKCS7Padding(plaintext)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// isValidDecryptedValueLinux checks if decrypted data looks like a valid cookie value.
func isValidDecryptedValueLinux(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	for _, b := range data {
		if b < 0x20 || b > 0x7E {
			return false
		}
	}
	return true
}

// removePKCS7Padding removes PKCS#7 padding from decrypted data.
func removePKCS7Padding(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	paddingLen := int(data[len(data)-1])
	if paddingLen == 0 || paddingLen > aes.BlockSize || paddingLen > len(data) {
		return nil, fmt.Errorf("invalid padding length: %d", paddingLen)
	}

	for i := len(data) - paddingLen; i < len(data); i++ {
		if data[i] != byte(paddingLen) {
			return nil, fmt.Errorf("invalid padding byte at position %d", i)
		}
	}

	return data[:len(data)-paddingLen], nil
}
