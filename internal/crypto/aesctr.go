package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

const (
	// NonceSize is the size of the AES-CTR nonce in bytes.
	NonceSize = 12
	// AES128KeySize is the key size for AES-128 in bytes.
	AES128KeySize = 16
	// AES256KeySize is the key size for AES-256 in bytes.
	AES256KeySize = 32
)

// AESCTRDecrypt decrypts ciphertext using AES in CTR mode.
// The nonce must be 12 bytes. The remaining 4 bytes of the IV are used as a counter
// starting at 0.
func AESCTRDecrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	if len(key) != AES128KeySize && len(key) != AES256KeySize {
		return nil, fmt.Errorf("key must be %d or %d bytes, got %d", AES128KeySize, AES256KeySize, len(key))
	}

	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("nonce must be %d bytes, got %d", NonceSize, len(nonce))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Build the full 16-byte IV: nonce (12 bytes) + counter (4 bytes starting at 0)
	iv := make([]byte, aes.BlockSize)
	copy(iv, nonce)
	// Last 4 bytes are already 0 (counter starts at 0)

	stream := cipher.NewCTR(block, iv)

	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

// AESCTREncrypt encrypts plaintext using AES in CTR mode.
// Since CTR mode is symmetric, this is the same as decryption.
func AESCTREncrypt(key, nonce, plaintext []byte) ([]byte, error) {
	return AESCTRDecrypt(key, nonce, plaintext)
}
