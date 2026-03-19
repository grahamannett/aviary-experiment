//go:build windows

package kurabiye

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// chromiumGetDecryptionKey retrieves the AES-GCM key from the Local State file on Windows.
func chromiumGetDecryptionKey(b *chromiumBrowser) ([]byte, error) {
	data, err := os.ReadFile(b.localStatePath)
	if err != nil {
		return nil, fmt.Errorf("reading Local State: %w", err)
	}

	var state struct {
		OSCrypt struct {
			EncryptedKey string `json:"encrypted_key"`
		} `json:"os_crypt"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing Local State JSON: %w", err)
	}

	if state.OSCrypt.EncryptedKey == "" {
		return nil, fmt.Errorf("no encrypted_key in Local State")
	}

	encKey, err := base64.StdEncoding.DecodeString(state.OSCrypt.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("base64 decode encrypted_key: %w", err)
	}

	if len(encKey) < 5 || string(encKey[:5]) != "DPAPI" {
		return nil, fmt.Errorf("encrypted_key missing DPAPI prefix")
	}
	encKey = encKey[5:]

	key, err := dpapiDecrypt(encKey)
	if err != nil {
		return nil, fmt.Errorf("DPAPI decrypt: %w", err)
	}

	return key, nil
}

// dpapiDecrypt uses PowerShell to call DPAPI Unprotect on the given data.
func dpapiDecrypt(data []byte) ([]byte, error) {
	b64 := base64.StdEncoding.EncodeToString(data)

	script := fmt.Sprintf(`
Add-Type -AssemblyName System.Security
$bytes = [Convert]::FromBase64String('%s')
$decrypted = [Security.Cryptography.ProtectedData]::Unprotect($bytes, $null, [Security.Cryptography.DataProtectionScope]::CurrentUser)
[Convert]::ToBase64String($decrypted)
`, b64)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("powershell DPAPI: %w", err)
	}

	result := strings.TrimSpace(string(out))
	return base64.StdEncoding.DecodeString(result)
}

// chromiumDecryptValue decrypts an encrypted cookie value from a Chromium browser on Windows.
func chromiumDecryptValue(encryptedValue []byte, key []byte, domain string) (string, error) {
	if len(encryptedValue) == 0 {
		return "", nil
	}

	if len(encryptedValue) >= 3 && string(encryptedValue[:3]) == "v10" {
		plaintext, err := decryptAESGCM(encryptedValue[3:], key)
		if err != nil {
			return "", err
		}
		// Chrome 130+ prepends SHA256(domain) to the plaintext
		return string(stripDomainHash([]byte(plaintext), domain)), nil
	}

	// Fallback: try raw DPAPI decryption (older Chrome)
	plaintext, err := dpapiDecrypt(encryptedValue)
	if err != nil {
		return "", fmt.Errorf("DPAPI fallback decrypt: %w", err)
	}
	return string(plaintext), nil
}

// decryptAESGCM decrypts AES-256-GCM encrypted data.
func decryptAESGCM(data []byte, key []byte) (string, error) {
	if len(data) < 12+16 {
		return "", fmt.Errorf("AES-GCM data too short")
	}

	nonce := data[:12]
	ciphertext := data[12:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("gcm.Open: %w", err)
	}

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
