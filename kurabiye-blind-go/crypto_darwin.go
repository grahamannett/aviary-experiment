//go:build darwin

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

// chromiumDecryptionKey retrieves the encryption key for a Chromium-based browser on macOS.
// It reads the password from the macOS Keychain and derives an AES key.
func chromiumDecryptionKey(browser string) ([]byte, error) {
	var service string
	switch browser {
	case "chrome":
		service = "Chrome Safe Storage"
	case "edge":
		service = "Microsoft Edge Safe Storage"
	default:
		service = "Chromium Safe Storage"
	}

	// Retrieve password from macOS Keychain
	cmd := exec.Command("security", "find-generic-password", "-s", service, "-w")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get %s password from Keychain: %w", service, err)
	}

	password := strings.TrimSpace(string(out))

	// Derive AES-128 key using PBKDF2
	// Salt: "saltysalt", Iterations: 1003, Key length: 16
	salt := []byte("saltysalt")
	key := pbkdf2.Key([]byte(password), salt, 1003, 16, sha1.New)

	return key, nil
}

// chromiumDecryptValue decrypts a Chromium encrypted cookie value on macOS.
// Two formats are supported:
//   - Legacy: v10 (3 bytes) + AES-128-CBC ciphertext (IV = 16 bytes of 0x20)
//   - Modern (Chrome ~v127+): v10 (3 bytes) + 16-byte header + 16-byte IV + AES-128-CBC ciphertext
func chromiumDecryptValue(encryptedValue []byte, key []byte) (string, error) {
	if len(encryptedValue) == 0 {
		return "", nil
	}

	// Check for v10 prefix
	if len(encryptedValue) < 3 || string(encryptedValue[:3]) != "v10" {
		// Not encrypted, return as-is (might be plaintext)
		return string(encryptedValue), nil
	}

	// Strip v10 prefix
	data := encryptedValue[3:]

	if len(data) == 0 {
		return "", nil
	}

	// Try modern format first: 16-byte header + 16-byte IV + ciphertext
	// Modern Chrome (v127+) on macOS prepends a 16-byte header and uses the
	// next 16 bytes as the CBC IV instead of the hardcoded 0x20 IV.
	if len(data) >= 48 { // need at least 16 (header) + 16 (IV) + 16 (min ciphertext)
		ct := data[32:]
		if len(ct)%aes.BlockSize == 0 && len(ct) > 0 {
			iv := data[16:32]
			plaintext, err := decryptCBC(ct, key, iv)
			if err == nil && isValidDecryptedValue(plaintext) {
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
		plaintext, err := decryptCBC(data, key, iv)
		if err != nil {
			return "", fmt.Errorf("decryption failed: %w", err)
		}
		return string(plaintext), nil
	}

	return "", fmt.Errorf("unexpected ciphertext length: %d", len(data))
}

// decryptCBC decrypts data using AES-128-CBC and removes PKCS#7 padding.
func decryptCBC(ciphertext, key, iv []byte) ([]byte, error) {
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

// isValidDecryptedValue checks if decrypted data looks like a valid cookie value.
// Valid cookie values should be printable ASCII.
func isValidDecryptedValue(data []byte) bool {
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
