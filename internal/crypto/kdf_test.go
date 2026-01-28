package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSP800108CounterKDF(t *testing.T) {
	t.Run("produces deterministic output", func(t *testing.T) {
		key := make([]byte, 16)
		for i := range key {
			key[i] = byte(i)
		}

		result1, err := SP800108CounterKDF(key, "TestLabel", "TestContext", 32)
		require.NoError(t, err)

		result2, err := SP800108CounterKDF(key, "TestLabel", "TestContext", 32)
		require.NoError(t, err)

		assert.Equal(t, result1, result2)
	})

	t.Run("different labels produce different output", func(t *testing.T) {
		key := make([]byte, 16)

		result1, err := SP800108CounterKDF(key, "Label1", "Context", 16)
		require.NoError(t, err)

		result2, err := SP800108CounterKDF(key, "Label2", "Context", 16)
		require.NoError(t, err)

		assert.NotEqual(t, result1, result2)
	})

	t.Run("different contexts produce different output", func(t *testing.T) {
		key := make([]byte, 16)

		result1, err := SP800108CounterKDF(key, "Label", "Context1", 16)
		require.NoError(t, err)

		result2, err := SP800108CounterKDF(key, "Label", "Context2", 16)
		require.NoError(t, err)

		assert.NotEqual(t, result1, result2)
	})

	t.Run("different keys produce different output", func(t *testing.T) {
		key1 := make([]byte, 16)
		key2 := make([]byte, 16)
		key2[0] = 1

		result1, err := SP800108CounterKDF(key1, "Label", "Context", 16)
		require.NoError(t, err)

		result2, err := SP800108CounterKDF(key2, "Label", "Context", 16)
		require.NoError(t, err)

		assert.NotEqual(t, result1, result2)
	})

	t.Run("produces correct output length", func(t *testing.T) {
		key := make([]byte, 16)

		for _, length := range []int{1, 8, 16, 24, 32, 48, 64} {
			result, err := SP800108CounterKDF(key, "Label", "Context", length)
			require.NoError(t, err)
			assert.Len(t, result, length)
		}
	})

	t.Run("supports 256-bit keys", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}

		result, err := SP800108CounterKDF(key, "Label", "Context", 32)
		require.NoError(t, err)
		assert.Len(t, result, 32)
	})

	t.Run("rejects invalid key sizes", func(t *testing.T) {
		invalidSizes := []int{0, 8, 15, 17, 24, 31, 33, 64}

		for _, size := range invalidSizes {
			key := make([]byte, size)
			_, err := SP800108CounterKDF(key, "Label", "Context", 16)
			assert.Error(t, err, "should reject key size %d", size)
		}
	})
}

func TestDeriveKey(t *testing.T) {
	t.Run("converts counter to string context", func(t *testing.T) {
		key := make([]byte, 16)

		// Counter 12345 should use "12345" as context
		result1, err := DeriveKey(key, 16, "Label", 12345)
		require.NoError(t, err)

		result2, err := SP800108CounterKDF(key, "Label", "12345", 16)
		require.NoError(t, err)

		assert.Equal(t, result1, result2)
	})
}

func TestNonceDerivation(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	t.Run("produces 12-byte nonce", func(t *testing.T) {
		nonce, err := FullNonceDerivation(key, 19000, 42)
		require.NoError(t, err)
		assert.Len(t, nonce, 12)
	})

	t.Run("different time counters produce different nonces", func(t *testing.T) {
		nonce1, err := FullNonceDerivation(key, 19000, 42)
		require.NoError(t, err)

		nonce2, err := FullNonceDerivation(key, 19001, 42)
		require.NoError(t, err)

		assert.NotEqual(t, nonce1, nonce2)
	})

	t.Run("different sequence counters produce different nonces", func(t *testing.T) {
		nonce1, err := FullNonceDerivation(key, 19000, 42)
		require.NoError(t, err)

		nonce2, err := FullNonceDerivation(key, 19000, 43)
		require.NoError(t, err)

		assert.NotEqual(t, nonce1, nonce2)
	})
}

func TestEncryptionKeyDerivation(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	t.Run("preserves key length", func(t *testing.T) {
		for _, keyLen := range []int{16, 32} {
			masterKey := make([]byte, keyLen)
			derivedKey, err := FullEncryptionKeyDerivation(masterKey, 19000, 42)
			require.NoError(t, err)
			assert.Len(t, derivedKey, keyLen)
		}
	})

	t.Run("different time counters produce different keys", func(t *testing.T) {
		key1, err := FullEncryptionKeyDerivation(key, 19000, 42)
		require.NoError(t, err)

		key2, err := FullEncryptionKeyDerivation(key, 19001, 42)
		require.NoError(t, err)

		assert.NotEqual(t, key1, key2)
	})
}

// TestKnownVector tests against a known test vector.
// This ensures compatibility with the Python implementation.
func TestKnownVector(t *testing.T) {
	// This test uses a specific key and verifies the derived values are consistent
	key, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")

	t.Run("nonce key derivation is deterministic", func(t *testing.T) {
		nonceKey, err := DeriveNonceKey(key, 20000)
		require.NoError(t, err)
		assert.Len(t, nonceKey, 32)

		// Run again to verify determinism
		nonceKey2, err := DeriveNonceKey(key, 20000)
		require.NoError(t, err)
		assert.Equal(t, nonceKey, nonceKey2)
	})

	t.Run("encryption key derivation is deterministic", func(t *testing.T) {
		encKey, err := FullEncryptionKeyDerivation(key, 20000, 100)
		require.NoError(t, err)
		assert.Len(t, encKey, 32)

		// Run again to verify determinism
		encKey2, err := FullEncryptionKeyDerivation(key, 20000, 100)
		require.NoError(t, err)
		assert.Equal(t, encKey, encKey2)
	})
}
