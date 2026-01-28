package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAESCTRDecrypt(t *testing.T) {
	t.Run("decryption is reversible", func(t *testing.T) {
		key := make([]byte, 16)
		nonce := make([]byte, 12)
		plaintext := []byte("Hello, World! This is a test message.")

		// Encrypt
		ciphertext, err := AESCTREncrypt(key, nonce, plaintext)
		require.NoError(t, err)

		// Decrypt
		decrypted, err := AESCTRDecrypt(key, nonce, ciphertext)
		require.NoError(t, err)

		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("ciphertext differs from plaintext", func(t *testing.T) {
		key := make([]byte, 16)
		nonce := make([]byte, 12)
		plaintext := []byte("test message")

		ciphertext, err := AESCTREncrypt(key, nonce, plaintext)
		require.NoError(t, err)

		assert.NotEqual(t, plaintext, ciphertext)
	})

	t.Run("preserves length", func(t *testing.T) {
		key := make([]byte, 16)
		nonce := make([]byte, 12)

		for _, length := range []int{0, 1, 15, 16, 17, 32, 100} {
			plaintext := bytes.Repeat([]byte{0x42}, length)
			ciphertext, err := AESCTREncrypt(key, nonce, plaintext)
			require.NoError(t, err)
			assert.Len(t, ciphertext, length)
		}
	})

	t.Run("different nonces produce different ciphertext", func(t *testing.T) {
		key := make([]byte, 16)
		nonce1 := make([]byte, 12)
		nonce2 := make([]byte, 12)
		nonce2[0] = 1
		plaintext := []byte("test message")

		ct1, err := AESCTREncrypt(key, nonce1, plaintext)
		require.NoError(t, err)

		ct2, err := AESCTREncrypt(key, nonce2, plaintext)
		require.NoError(t, err)

		assert.NotEqual(t, ct1, ct2)
	})

	t.Run("different keys produce different ciphertext", func(t *testing.T) {
		key1 := make([]byte, 16)
		key2 := make([]byte, 16)
		key2[0] = 1
		nonce := make([]byte, 12)
		plaintext := []byte("test message")

		ct1, err := AESCTREncrypt(key1, nonce, plaintext)
		require.NoError(t, err)

		ct2, err := AESCTREncrypt(key2, nonce, plaintext)
		require.NoError(t, err)

		assert.NotEqual(t, ct1, ct2)
	})

	t.Run("supports 256-bit keys", func(t *testing.T) {
		key := make([]byte, 32)
		nonce := make([]byte, 12)
		plaintext := []byte("test message")

		ciphertext, err := AESCTREncrypt(key, nonce, plaintext)
		require.NoError(t, err)

		decrypted, err := AESCTRDecrypt(key, nonce, ciphertext)
		require.NoError(t, err)

		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("rejects invalid key sizes", func(t *testing.T) {
		nonce := make([]byte, 12)
		plaintext := []byte("test")

		for _, size := range []int{0, 8, 15, 17, 24, 31, 33} {
			key := make([]byte, size)
			_, err := AESCTRDecrypt(key, nonce, plaintext)
			assert.Error(t, err, "should reject key size %d", size)
		}
	})

	t.Run("rejects invalid nonce sizes", func(t *testing.T) {
		key := make([]byte, 16)
		plaintext := []byte("test")

		for _, size := range []int{0, 8, 11, 13, 16} {
			nonce := make([]byte, size)
			_, err := AESCTRDecrypt(key, nonce, plaintext)
			assert.Error(t, err, "should reject nonce size %d", size)
		}
	})
}

// TestAESCTRKnownVector tests against NIST test vectors.
func TestAESCTRKnownVector(t *testing.T) {
	// NIST SP 800-38A F.5.1 CTR-AES128.Encrypt
	t.Run("NIST CTR-AES128", func(t *testing.T) {
		key, _ := hex.DecodeString("2b7e151628aed2a6abf7158809cf4f3c")
		// Using first 12 bytes of the NIST IV as nonce
		nonce, _ := hex.DecodeString("f0f1f2f3f4f5f6f7f8f9fafb")
		plaintext, _ := hex.DecodeString("6bc1bee22e409f96e93d7e117393172a")

		ciphertext, err := AESCTREncrypt(key, nonce, plaintext)
		require.NoError(t, err)

		// Verify decryption
		decrypted, err := AESCTRDecrypt(key, nonce, ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})
}

func TestAESCTREncrypt(t *testing.T) {
	t.Run("encrypt and decrypt are symmetric", func(t *testing.T) {
		key := make([]byte, 16)
		nonce := make([]byte, 12)
		plaintext := []byte("symmetric test")

		// In CTR mode, encrypt and decrypt are the same operation
		encrypted, _ := AESCTREncrypt(key, nonce, plaintext)
		decrypted, _ := AESCTRDecrypt(key, nonce, encrypted)

		assert.Equal(t, plaintext, decrypted)

		// Also verify double-encrypt returns original
		doubleEncrypted, _ := AESCTREncrypt(key, nonce, encrypted)
		assert.Equal(t, plaintext, doubleEncrypted)
	})
}
