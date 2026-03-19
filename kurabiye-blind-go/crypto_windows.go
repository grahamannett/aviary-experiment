//go:build windows

package kurabiye

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	dllCrypt32  = syscall.NewLazyDLL("crypt32.dll")
	dllKernel32 = syscall.NewLazyDLL("kernel32.dll")

	procDecryptData = dllCrypt32.NewProc("CryptUnprotectData")
	procLocalFree   = dllKernel32.NewProc("LocalFree")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func newDataBlob(d []byte) *dataBlob {
	if len(d) == 0 {
		return &dataBlob{}
	}
	return &dataBlob{
		cbData: uint32(len(d)),
		pbData: &d[0],
	}
}

// dpApiDecrypt decrypts data using Windows DPAPI.
func dpApiDecrypt(data []byte) ([]byte, error) {
	var outBlob dataBlob
	inBlob := newDataBlob(data)

	r, _, err := procDecryptData.Call(
		uintptr(unsafe.Pointer(inBlob)),
		0, // pOptionalEntropy
		0, // pvReserved
		0, // pPromptStruct
		0, // dwFlags
		uintptr(unsafe.Pointer(&outBlob)),
	)

	if r == 0 {
		return nil, fmt.Errorf("CryptUnprotectData failed: %w", err)
	}

	defer procLocalFree.Call(uintptr(unsafe.Pointer(outBlob.pbData)))

	result := make([]byte, outBlob.cbData)
	copy(result, unsafe.Slice(outBlob.pbData, outBlob.cbData))

	return result, nil
}

// chromiumDecryptionKey retrieves the AES-GCM master key for Chromium on Windows.
// The key is stored in the browser's Local State file, encrypted with DPAPI.
func chromiumDecryptionKey(browser string) ([]byte, error) {
	var profileDir string
	switch browser {
	case "chrome":
		profileDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data")
	case "edge":
		profileDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "User Data")
	default:
		return nil, fmt.Errorf("unsupported browser for Windows decryption: %s", browser)
	}

	localStatePath := filepath.Join(profileDir, "Local State")
	data, err := os.ReadFile(localStatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Local State: %w", err)
	}

	var localState struct {
		OSCrypt struct {
			EncryptedKey string `json:"encrypted_key"`
		} `json:"os_crypt"`
	}

	if err := json.Unmarshal(data, &localState); err != nil {
		return nil, fmt.Errorf("failed to parse Local State: %w", err)
	}

	if localState.OSCrypt.EncryptedKey == "" {
		return nil, fmt.Errorf("no encrypted_key found in Local State")
	}

	// Base64 decode the key
	encryptedKey, err := base64.StdEncoding.DecodeString(localState.OSCrypt.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to base64 decode encrypted key: %w", err)
	}

	// Strip "DPAPI" prefix (5 bytes)
	if len(encryptedKey) < 5 || string(encryptedKey[:5]) != "DPAPI" {
		return nil, fmt.Errorf("encrypted key missing DPAPI prefix")
	}
	encryptedKey = encryptedKey[5:]

	// Decrypt with DPAPI
	key, err := dpApiDecrypt(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt master key with DPAPI: %w", err)
	}

	return key, nil
}

// chromiumDecryptValue decrypts a Chromium encrypted cookie value on Windows.
// For v10-prefixed values: AES-256-GCM with 12-byte nonce.
// For other values: direct DPAPI decryption.
func chromiumDecryptValue(encryptedValue []byte, key []byte) (string, error) {
	if len(encryptedValue) == 0 {
		return "", nil
	}

	// Check for v10 prefix (AES-256-GCM)
	if len(encryptedValue) >= 3 && string(encryptedValue[:3]) == "v10" {
		ciphertext := encryptedValue[3:]

		if len(ciphertext) < 12+16 {
			return "", fmt.Errorf("v10 encrypted value too short")
		}

		// 12-byte nonce, then ciphertext+tag
		nonce := ciphertext[:12]
		encrypted := ciphertext[12:]

		block, err := aes.NewCipher(key)
		if err != nil {
			return "", fmt.Errorf("failed to create AES cipher: %w", err)
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return "", fmt.Errorf("failed to create GCM: %w", err)
		}

		plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
		if err != nil {
			return "", fmt.Errorf("AES-GCM decryption failed: %w", err)
		}

		return string(plaintext), nil
	}

	// Fallback: DPAPI decryption
	plaintext, err := dpApiDecrypt(encryptedValue)
	if err != nil {
		return "", fmt.Errorf("DPAPI decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// removePKCS7Padding removes PKCS#7 padding from decrypted data.
// (Not used on Windows since we use AES-GCM, but included for completeness.)
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
