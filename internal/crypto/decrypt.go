package crypto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/hubblenetwork/hubcli/internal/models"
)

const (
	// SecondsPerDay is the number of seconds in a day.
	SecondsPerDay = 86400

	// MinPacketSize is the minimum valid packet size.
	// 2 bytes header + 4 bytes reserved + 4 bytes auth tag = 10 bytes minimum
	MinPacketSize = 10

	// HeaderSize is the size of the packet header (sequence number bytes).
	HeaderSize = 2

	// ReservedSize is the size of reserved bytes before the auth tag.
	ReservedSize = 4

	// AuthTagOffset is the byte offset where the auth tag starts.
	AuthTagOffset = HeaderSize + ReservedSize // 6

	// PayloadOffset is the byte offset where the encrypted payload starts.
	PayloadOffset = AuthTagOffset + AuthTagSize // 10

	// SequenceNumberMask extracts the 10-bit sequence number.
	SequenceNumberMask = 0x3FF

	// DefaultSearchWindowDays is the default number of days to search in each direction.
	DefaultSearchWindowDays = 2
)

// Common errors
var (
	ErrPacketTooShort     = errors.New("packet too short")
	ErrDecryptionFailed   = errors.New("decryption failed: no valid time counter found")
	ErrInvalidKey         = errors.New("invalid key size")
	ErrAuthenticationFail = errors.New("authentication tag mismatch")
)

// ParsedPacket contains the parsed components of an encrypted BLE advertisement.
type ParsedPacket struct {
	SequenceNumber   uint16 // 10-bit sequence counter
	AuthTag          []byte // 4-byte truncated CMAC
	EncryptedPayload []byte // Encrypted data
	RawPacket        []byte // Original packet bytes
}

// ParsePacket extracts the components from a BLE advertisement payload.
func ParsePacket(payload []byte) (*ParsedPacket, error) {
	if len(payload) < MinPacketSize {
		return nil, fmt.Errorf("%w: got %d bytes, need at least %d", ErrPacketTooShort, len(payload), MinPacketSize)
	}

	// Extract 10-bit sequence number from first 2 bytes (big-endian)
	seqRaw := binary.BigEndian.Uint16(payload[0:2])
	seqNum := seqRaw & SequenceNumberMask

	// Extract 4-byte auth tag at offset 6
	authTag := make([]byte, AuthTagSize)
	copy(authTag, payload[AuthTagOffset:AuthTagOffset+AuthTagSize])

	// Extract encrypted payload starting at offset 10
	var encPayload []byte
	if len(payload) > PayloadOffset {
		encPayload = make([]byte, len(payload)-PayloadOffset)
		copy(encPayload, payload[PayloadOffset:])
	}

	return &ParsedPacket{
		SequenceNumber:   seqNum,
		AuthTag:          authTag,
		EncryptedPayload: encPayload,
		RawPacket:        payload,
	}, nil
}

// DecryptResult contains the result of a successful decryption.
type DecryptResult struct {
	Payload     []byte // Decrypted payload
	TimeCounter uint32 // The time counter that worked
	SeqCounter  uint32 // The sequence counter from the packet
}

// DecryptOptions configures the decryption behavior.
type DecryptOptions struct {
	// SearchWindowDays is the number of days to search in each direction.
	// Default is 2 (searches -2 to +2 days from expected time).
	SearchWindowDays int

	// ExpectedTime is the expected timestamp for the packet.
	// If zero, uses the packet's timestamp or current time.
	ExpectedTime time.Time
}

// DecryptOption is a functional option for configuring decryption.
type DecryptOption func(*DecryptOptions)

// WithSearchWindow sets the number of days to search in each direction.
func WithSearchWindow(days int) DecryptOption {
	return func(o *DecryptOptions) {
		o.SearchWindowDays = days
	}
}

// WithExpectedTime sets the expected timestamp for the packet.
func WithExpectedTime(t time.Time) DecryptOption {
	return func(o *DecryptOptions) {
		o.ExpectedTime = t
	}
}

// TimeToCounter converts a Unix timestamp to a time counter (days since epoch).
func TimeToCounter(t time.Time) uint32 {
	return uint32(t.Unix() / SecondsPerDay)
}

// CounterToTime converts a time counter back to a time (midnight of that day).
func CounterToTime(counter uint32) time.Time {
	return time.Unix(int64(counter)*SecondsPerDay, 0).UTC()
}

// Decrypt attempts to decrypt an encrypted packet using the provided key.
// It searches a time window around the expected time to find the correct counter.
func Decrypt(key []byte, packet models.EncryptedPacket, opts ...DecryptOption) (*DecryptResult, error) {
	if len(key) != AES128KeySize && len(key) != AES256KeySize {
		return nil, ErrInvalidKey
	}

	// Apply options
	options := DecryptOptions{
		SearchWindowDays: DefaultSearchWindowDays,
		ExpectedTime:     packet.Timestamp,
	}
	for _, opt := range opts {
		opt(&options)
	}

	if options.ExpectedTime.IsZero() {
		options.ExpectedTime = time.Now().UTC()
	}

	// Parse the packet
	parsed, err := ParsePacket(packet.Payload)
	if err != nil {
		return nil, err
	}

	// Calculate the base time counter and search range
	baseCounter := TimeToCounter(options.ExpectedTime)
	minCounter := baseCounter - uint32(options.SearchWindowDays)
	maxCounter := baseCounter + uint32(options.SearchWindowDays)

	// Search for a valid time counter
	for tc := minCounter; tc <= maxCounter; tc++ {
		result, err := tryDecrypt(key, parsed, tc)
		if err == nil {
			return result, nil
		}
		// Continue searching if authentication failed
	}

	return nil, ErrDecryptionFailed
}

// tryDecrypt attempts decryption with a specific time counter.
func tryDecrypt(key []byte, parsed *ParsedPacket, timeCounter uint32) (*DecryptResult, error) {
	seqCounter := uint32(parsed.SequenceNumber)

	// Derive the authentication key and verify the tag
	// The auth tag is computed over the data portion before the auth tag
	authData := parsed.RawPacket[:AuthTagOffset]

	// Derive keys for this time counter
	encKey, err := FullEncryptionKeyDerivation(key, timeCounter, seqCounter)
	if err != nil {
		return nil, fmt.Errorf("key derivation failed: %w", err)
	}

	// Compute expected auth tag using the encryption key
	expectedTag, err := ComputeAuthTag(encKey, authData)
	if err != nil {
		return nil, fmt.Errorf("auth tag computation failed: %w", err)
	}

	// Verify auth tag
	valid, err := VerifyAuthTag(encKey, authData, parsed.AuthTag)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrAuthenticationFail
	}
	_ = expectedTag // Used in verification

	// Auth tag matches, proceed with decryption
	nonce, err := FullNonceDerivation(key, timeCounter, seqCounter)
	if err != nil {
		return nil, fmt.Errorf("nonce derivation failed: %w", err)
	}

	plaintext, err := AESCTRDecrypt(encKey, nonce, parsed.EncryptedPayload)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return &DecryptResult{
		Payload:     plaintext,
		TimeCounter: timeCounter,
		SeqCounter:  seqCounter,
	}, nil
}

// DecryptWithKnownCounter decrypts a packet when the time counter is already known.
// This is faster than Decrypt() as it doesn't search.
func DecryptWithKnownCounter(key []byte, packet models.EncryptedPacket, timeCounter uint32) (*DecryptResult, error) {
	if len(key) != AES128KeySize && len(key) != AES256KeySize {
		return nil, ErrInvalidKey
	}

	parsed, err := ParsePacket(packet.Payload)
	if err != nil {
		return nil, err
	}

	return tryDecrypt(key, parsed, timeCounter)
}

// FindTimeCounter searches for the correct time counter without decrypting.
// Returns the time counter if found, or an error if no valid counter is found.
func FindTimeCounter(key []byte, packet models.EncryptedPacket, opts ...DecryptOption) (uint32, error) {
	if len(key) != AES128KeySize && len(key) != AES256KeySize {
		return 0, ErrInvalidKey
	}

	options := DecryptOptions{
		SearchWindowDays: DefaultSearchWindowDays,
		ExpectedTime:     packet.Timestamp,
	}
	for _, opt := range opts {
		opt(&options)
	}

	if options.ExpectedTime.IsZero() {
		options.ExpectedTime = time.Now().UTC()
	}

	parsed, err := ParsePacket(packet.Payload)
	if err != nil {
		return 0, err
	}

	baseCounter := TimeToCounter(options.ExpectedTime)
	minCounter := baseCounter - uint32(options.SearchWindowDays)
	maxCounter := baseCounter + uint32(options.SearchWindowDays)

	for tc := minCounter; tc <= maxCounter; tc++ {
		seqCounter := uint32(parsed.SequenceNumber)

		encKey, err := FullEncryptionKeyDerivation(key, tc, seqCounter)
		if err != nil {
			continue
		}

		authData := parsed.RawPacket[:AuthTagOffset]
		valid, err := VerifyAuthTag(encKey, authData, parsed.AuthTag)
		if err == nil && valid {
			return tc, nil
		}
	}

	return 0, ErrDecryptionFailed
}
