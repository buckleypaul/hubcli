package crypto

import (
	"crypto/aes"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/aead/cmac"
)

// SP800108CounterKDF implements NIST SP 800-108 Key Derivation Function in Counter Mode
// using AES-CMAC as the pseudo-random function (PRF).
//
// The KDF produces output of the requested length by concatenating PRF outputs:
// K(i) = PRF(KI, [i]₂ || Label || 0x00 || Context || [L]₂)
// where:
//   - [i]₂ is a 32-bit big-endian counter starting at 1
//   - Label is the purpose string
//   - 0x00 is a separator byte
//   - Context is additional context data
//   - [L]₂ is a 32-bit big-endian representation of output length in bits
func SP800108CounterKDF(key []byte, label, context string, outputLen int) ([]byte, error) {
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

	blockSize := mac.Size() // 16 bytes for AES-CMAC
	numBlocks := (outputLen + blockSize - 1) / blockSize

	// Build the fixed portion of the input: Label || 0x00 || Context || [L]₂
	labelBytes := []byte(label)
	contextBytes := []byte(context)
	outputBits := uint32(outputLen * 8)

	fixedInput := make([]byte, 0, len(labelBytes)+1+len(contextBytes)+4)
	fixedInput = append(fixedInput, labelBytes...)
	fixedInput = append(fixedInput, 0x00) // separator
	fixedInput = append(fixedInput, contextBytes...)
	fixedInput = binary.BigEndian.AppendUint32(fixedInput, outputBits)

	// Generate output blocks
	result := make([]byte, 0, numBlocks*blockSize)
	counterBytes := make([]byte, 4)

	for i := 1; i <= numBlocks; i++ {
		mac.Reset()

		// [i]₂ || fixedInput
		binary.BigEndian.PutUint32(counterBytes, uint32(i))
		mac.Write(counterBytes)
		mac.Write(fixedInput)

		result = append(result, mac.Sum(nil)...)
	}

	// Truncate to requested length
	return result[:outputLen], nil
}

// DeriveKey is a convenience wrapper around SP800108CounterKDF that converts
// a numeric context to a string, matching the pyhubblenetwork implementation.
func DeriveKey(masterKey []byte, outputLen int, label string, counter uint32) ([]byte, error) {
	context := strconv.FormatUint(uint64(counter), 10)
	return SP800108CounterKDF(masterKey, label, context, outputLen)
}

// DeriveNonceKey derives the intermediate nonce key from the master key.
func DeriveNonceKey(masterKey []byte, timeCounter uint32) ([]byte, error) {
	return DeriveKey(masterKey, len(masterKey), "NonceKey", timeCounter)
}

// DeriveNonce derives the 12-byte nonce from the nonce key and sequence counter.
func DeriveNonce(nonceKey []byte, seqCounter uint32) ([]byte, error) {
	return DeriveKey(nonceKey, 12, "Nonce", seqCounter)
}

// DeriveEncryptionKeyIntermediate derives the intermediate encryption key.
func DeriveEncryptionKeyIntermediate(masterKey []byte, timeCounter uint32) ([]byte, error) {
	return DeriveKey(masterKey, len(masterKey), "EncryptionKey", timeCounter)
}

// DeriveEncryptionKey derives the final encryption key from the intermediate key.
func DeriveEncryptionKey(intermediateKey []byte, seqCounter uint32) ([]byte, error) {
	return DeriveKey(intermediateKey, len(intermediateKey), "Key", seqCounter)
}

// FullNonceDerivation performs the complete two-stage nonce derivation.
func FullNonceDerivation(masterKey []byte, timeCounter, seqCounter uint32) ([]byte, error) {
	nonceKey, err := DeriveNonceKey(masterKey, timeCounter)
	if err != nil {
		return nil, fmt.Errorf("failed to derive nonce key: %w", err)
	}

	return DeriveNonce(nonceKey, seqCounter)
}

// FullEncryptionKeyDerivation performs the complete two-stage encryption key derivation.
func FullEncryptionKeyDerivation(masterKey []byte, timeCounter, seqCounter uint32) ([]byte, error) {
	intermediateKey, err := DeriveEncryptionKeyIntermediate(masterKey, timeCounter)
	if err != nil {
		return nil, fmt.Errorf("failed to derive intermediate key: %w", err)
	}

	return DeriveEncryptionKey(intermediateKey, seqCounter)
}
