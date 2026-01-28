package crypto

import (
	"crypto/aes"
	"crypto/subtle"
	"fmt"

	"github.com/aead/cmac"
)

const (
	// AuthTagSize is the size of the truncated authentication tag in bytes.
	AuthTagSize = 4
)

// ComputeAuthTag computes a 4-byte truncated AES-CMAC authentication tag.
// The full 16-byte CMAC is computed and then truncated to 4 bytes.
func ComputeAuthTag(key, data []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 32 {
		return nil, fmt.Errorf("key must be 16 or 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	mac, err := cmac.New(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create CMAC: %w", err)
	}

	mac.Write(data)
	fullTag := mac.Sum(nil)

	// Truncate to 4 bytes
	return fullTag[:AuthTagSize], nil
}

// VerifyAuthTag checks if the provided tag matches the computed tag.
// Uses constant-time comparison to prevent timing attacks.
func VerifyAuthTag(key, data, expectedTag []byte) (bool, error) {
	if len(expectedTag) != AuthTagSize {
		return false, fmt.Errorf("expected tag must be %d bytes, got %d", AuthTagSize, len(expectedTag))
	}

	computedTag, err := ComputeAuthTag(key, data)
	if err != nil {
		return false, err
	}

	return subtle.ConstantTimeCompare(computedTag, expectedTag) == 1, nil
}

// ComputeFullCMAC computes the full 16-byte AES-CMAC (not truncated).
// Useful for testing and debugging.
func ComputeFullCMAC(key, data []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 32 {
		return nil, fmt.Errorf("key must be 16 or 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	mac, err := cmac.New(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create CMAC: %w", err)
	}

	mac.Write(data)
	return mac.Sum(nil), nil
}
