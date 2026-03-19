//go:build darwin

package kurabiye

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

func TestChromiumDecryptRoundTrip(t *testing.T) {
	// Simulate what Chrome does: encrypt a value and verify we can decrypt it.
	password := "testpassword"
	salt := []byte("saltysalt")
	key := pbkdf2.Key([]byte(password), salt, 1003, 16, sha1.New)

	plaintext := "my_secret_cookie_value"

	// Encrypt: PKCS#7 pad, then AES-128-CBC with IV = 16 * 0x20
	padded := pkcs7Pad([]byte(plaintext), aes.BlockSize)

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}

	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = 0x20
	}

	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padded)

	// Prepend "v10" prefix
	encrypted := append([]byte("v10"), ciphertext...)

	// Decrypt
	got, err := chromiumDecryptValue(encrypted, key, "")
	if err != nil {
		t.Fatalf("chromiumDecryptValue error: %v", err)
	}

	if got != plaintext {
		t.Errorf("decrypted = %q, want %q", got, plaintext)
	}
}

func TestChromiumDecryptEmpty(t *testing.T) {
	got, err := chromiumDecryptValue(nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestChromiumDecryptNoPrefix(t *testing.T) {
	// Data without v10 prefix should be returned as-is
	data := []byte("plainvalue")
	got, err := chromiumDecryptValue(data, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "plainvalue" {
		t.Errorf("expected 'plainvalue', got %q", got)
	}
}

func TestPkcs7Unpad(t *testing.T) {
	tests := []struct {
		input     []byte
		blockSize int
		want      []byte
		hasErr    bool
	}{
		{[]byte{1, 2, 3, 4, 4, 4, 4, 4}, 8, []byte{1, 2, 3, 4}, false},
		{[]byte{8, 8, 8, 8, 8, 8, 8, 8}, 8, []byte{}, false},
		{[]byte{1, 2, 3, 4, 5, 6, 7, 1}, 8, []byte{1, 2, 3, 4, 5, 6, 7}, false},
		{nil, 8, nil, true},                   // empty
		{[]byte{0, 0, 0, 0}, 8, nil, true},    // zero padding byte
		{[]byte{1, 2, 3, 9}, 8, nil, true},    // pad > blockSize
		{[]byte{1, 2, 3, 2}, 8, nil, true},    // inconsistent padding
	}

	for i, tt := range tests {
		got, err := pkcs7Unpad(tt.input, tt.blockSize)
		if tt.hasErr {
			if err == nil {
				t.Errorf("case %d: expected error", i)
			}
			continue
		}
		if err != nil {
			t.Errorf("case %d: unexpected error: %v", i, err)
			continue
		}
		if string(got) != string(tt.want) {
			t.Errorf("case %d: got %v, want %v", i, got, tt.want)
		}
	}
}

// pkcs7Pad adds PKCS#7 padding.
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	pad := make([]byte, padding)
	for i := range pad {
		pad[i] = byte(padding)
	}
	return append(data, pad...)
}
