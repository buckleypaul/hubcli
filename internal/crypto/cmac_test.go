package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeAuthTag(t *testing.T) {
	t.Run("produces 4-byte tag", func(t *testing.T) {
		key := make([]byte, 16)
		data := []byte("test data")

		tag, err := ComputeAuthTag(key, data)
		require.NoError(t, err)
		assert.Len(t, tag, AuthTagSize)
	})

	t.Run("deterministic output", func(t *testing.T) {
		key := make([]byte, 16)
		data := []byte("test data")

		tag1, err := ComputeAuthTag(key, data)
		require.NoError(t, err)

		tag2, err := ComputeAuthTag(key, data)
		require.NoError(t, err)

		assert.Equal(t, tag1, tag2)
	})

	t.Run("different data produces different tags", func(t *testing.T) {
		key := make([]byte, 16)

		tag1, err := ComputeAuthTag(key, []byte("data1"))
		require.NoError(t, err)

		tag2, err := ComputeAuthTag(key, []byte("data2"))
		require.NoError(t, err)

		assert.NotEqual(t, tag1, tag2)
	})

	t.Run("different keys produce different tags", func(t *testing.T) {
		key1 := make([]byte, 16)
		key2 := make([]byte, 16)
		key2[0] = 1
		data := []byte("test data")

		tag1, err := ComputeAuthTag(key1, data)
		require.NoError(t, err)

		tag2, err := ComputeAuthTag(key2, data)
		require.NoError(t, err)

		assert.NotEqual(t, tag1, tag2)
	})

	t.Run("supports 256-bit keys", func(t *testing.T) {
		key := make([]byte, 32)
		data := []byte("test data")

		tag, err := ComputeAuthTag(key, data)
		require.NoError(t, err)
		assert.Len(t, tag, AuthTagSize)
	})

	t.Run("rejects invalid key sizes", func(t *testing.T) {
		invalidSizes := []int{0, 8, 15, 17, 24, 31, 33}

		for _, size := range invalidSizes {
			key := make([]byte, size)
			_, err := ComputeAuthTag(key, []byte("data"))
			assert.Error(t, err, "should reject key size %d", size)
		}
	})
}

func TestVerifyAuthTag(t *testing.T) {
	key := make([]byte, 16)
	data := []byte("test data")

	t.Run("returns true for valid tag", func(t *testing.T) {
		tag, err := ComputeAuthTag(key, data)
		require.NoError(t, err)

		valid, err := VerifyAuthTag(key, data, tag)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("returns false for invalid tag", func(t *testing.T) {
		invalidTag := []byte{0x00, 0x00, 0x00, 0x00}

		valid, err := VerifyAuthTag(key, data, invalidTag)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("returns false for modified data", func(t *testing.T) {
		tag, err := ComputeAuthTag(key, data)
		require.NoError(t, err)

		modifiedData := []byte("modified data")
		valid, err := VerifyAuthTag(key, modifiedData, tag)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("rejects wrong tag length", func(t *testing.T) {
		_, err := VerifyAuthTag(key, data, []byte{0x00, 0x00, 0x00})
		assert.Error(t, err)

		_, err = VerifyAuthTag(key, data, []byte{0x00, 0x00, 0x00, 0x00, 0x00})
		assert.Error(t, err)
	})
}

func TestComputeFullCMAC(t *testing.T) {
	t.Run("produces 16-byte output", func(t *testing.T) {
		key := make([]byte, 16)
		data := []byte("test data")

		mac, err := ComputeFullCMAC(key, data)
		require.NoError(t, err)
		assert.Len(t, mac, 16)
	})

	t.Run("truncated tag matches full CMAC prefix", func(t *testing.T) {
		key := make([]byte, 16)
		data := []byte("test data")

		fullMAC, err := ComputeFullCMAC(key, data)
		require.NoError(t, err)

		truncatedTag, err := ComputeAuthTag(key, data)
		require.NoError(t, err)

		assert.Equal(t, fullMAC[:AuthTagSize], truncatedTag)
	})
}

// TestCMACKnownVector tests against AES-CMAC test vectors from RFC 4493.
func TestCMACKnownVector(t *testing.T) {
	// RFC 4493 Example 1: AES-CMAC with 128-bit key
	key, _ := hex.DecodeString("2b7e151628aed2a6abf7158809cf4f3c")

	t.Run("RFC 4493 empty message", func(t *testing.T) {
		expected, _ := hex.DecodeString("bb1d6929e95937287fa37d129b756746")

		mac, err := ComputeFullCMAC(key, []byte{})
		require.NoError(t, err)
		assert.Equal(t, expected, mac)
	})

	t.Run("RFC 4493 16-byte message", func(t *testing.T) {
		message, _ := hex.DecodeString("6bc1bee22e409f96e93d7e117393172a")
		expected, _ := hex.DecodeString("070a16b46b4d4144f79bdd9dd04a287c")

		mac, err := ComputeFullCMAC(key, message)
		require.NoError(t, err)
		assert.Equal(t, expected, mac)
	})

	t.Run("RFC 4493 40-byte message", func(t *testing.T) {
		message, _ := hex.DecodeString("6bc1bee22e409f96e93d7e117393172aae2d8a571e03ac9c9eb76fac45af8e5130c81c46a35ce411")
		expected, _ := hex.DecodeString("dfa66747de9ae63030ca32611497c827")

		mac, err := ComputeFullCMAC(key, message)
		require.NoError(t, err)
		assert.Equal(t, expected, mac)
	})
}
