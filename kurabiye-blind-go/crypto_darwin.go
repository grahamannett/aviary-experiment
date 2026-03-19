//go:build darwin

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

// chromiumGetDecryptionKey retrieves the encryption key for a Chromium browser on macOS.
func chromiumGetDecryptionKey(b *chromiumBrowser) ([]byte, error) {
	cmd := exec.Command("security", "find-generic-password", "-s", b.keychainName, "-w")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("keychain lookup for %q failed: %w", b.keychainName, err)
	}
	password := strings.TrimSpace(string(out))

	salt := []byte("saltysalt")
	key := pbkdf2.Key([]byte(password), salt, 1003, 16, sha1.New)
	return key, nil
}

// chromiumDecryptValue decrypts an encrypted cookie value from a Chromium browser on macOS.
// domain is used to strip the SHA256(domain) prefix added by Chrome 130+.
func chromiumDecryptValue(encryptedValue []byte, key []byte, domain string) (string, error) {
	if len(encryptedValue) == 0 {
		return "", nil
	}

	if len(encryptedValue) < 3 || string(encryptedValue[:3]) != "v10" {
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

	// IV is 16 bytes of space (0x20)
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
	// Detect and strip it.
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
