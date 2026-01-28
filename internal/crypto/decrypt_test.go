package crypto

import (
	"testing"
	"time"

	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePacket(t *testing.T) {
	t.Run("parses valid packet", func(t *testing.T) {
		// Construct a test packet:
		// Bytes 0-1: sequence number (0x0042 = 66, masked to 10 bits = 66)
		// Bytes 2-5: reserved
		// Bytes 6-9: auth tag
		// Bytes 10+: encrypted payload
		packet := []byte{
			0x00, 0x42, // sequence number
			0x00, 0x00, 0x00, 0x00, // reserved
			0xAA, 0xBB, 0xCC, 0xDD, // auth tag
			0x01, 0x02, 0x03, 0x04, // encrypted payload
		}

		parsed, err := ParsePacket(packet)
		require.NoError(t, err)

		assert.Equal(t, uint16(66), parsed.SequenceNumber)
		assert.Equal(t, []byte{0xAA, 0xBB, 0xCC, 0xDD}, parsed.AuthTag)
		assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, parsed.EncryptedPayload)
	})

	t.Run("extracts 10-bit sequence number", func(t *testing.T) {
		// 0x03FF = 1023, the maximum 10-bit value
		packet := []byte{
			0x03, 0xFF, // sequence number (all 10 bits set)
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
		}

		parsed, err := ParsePacket(packet)
		require.NoError(t, err)
		assert.Equal(t, uint16(1023), parsed.SequenceNumber)
	})

	t.Run("masks upper bits of sequence number", func(t *testing.T) {
		// 0xFFFF should be masked to 0x03FF (1023)
		packet := []byte{
			0xFF, 0xFF, // high bits should be masked
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
		}

		parsed, err := ParsePacket(packet)
		require.NoError(t, err)
		assert.Equal(t, uint16(1023), parsed.SequenceNumber)
	})

	t.Run("handles minimum size packet (no payload)", func(t *testing.T) {
		packet := make([]byte, MinPacketSize)

		parsed, err := ParsePacket(packet)
		require.NoError(t, err)
		assert.Empty(t, parsed.EncryptedPayload)
	})

	t.Run("rejects packet too short", func(t *testing.T) {
		shortPacket := make([]byte, MinPacketSize-1)

		_, err := ParsePacket(shortPacket)
		assert.ErrorIs(t, err, ErrPacketTooShort)
	})

	t.Run("preserves raw packet", func(t *testing.T) {
		packet := []byte{
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05,
			0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B,
		}

		parsed, err := ParsePacket(packet)
		require.NoError(t, err)
		assert.Equal(t, packet, parsed.RawPacket)
	})
}

func TestTimeToCounter(t *testing.T) {
	t.Run("converts to days since epoch", func(t *testing.T) {
		// 2024-01-15 is day 19738 since Unix epoch
		timestamp := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		counter := TimeToCounter(timestamp)

		expectedDay := uint32(timestamp.Unix() / SecondsPerDay)
		assert.Equal(t, expectedDay, counter)
	})

	t.Run("same day returns same counter", func(t *testing.T) {
		t1 := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC)

		assert.Equal(t, TimeToCounter(t1), TimeToCounter(t2))
	})

	t.Run("different days return different counters", func(t *testing.T) {
		t1 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC)

		assert.Equal(t, TimeToCounter(t1)+1, TimeToCounter(t2))
	})
}

func TestCounterToTime(t *testing.T) {
	t.Run("converts to midnight UTC", func(t *testing.T) {
		counter := uint32(19738) // Some day
		result := CounterToTime(counter)

		assert.Equal(t, 0, result.Hour())
		assert.Equal(t, 0, result.Minute())
		assert.Equal(t, 0, result.Second())
	})

	t.Run("roundtrips correctly", func(t *testing.T) {
		original := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		counter := TimeToCounter(original)
		result := CounterToTime(counter)

		assert.Equal(t, original, result)
	})
}

func TestDecryptOptions(t *testing.T) {
	t.Run("WithSearchWindow sets days", func(t *testing.T) {
		opts := DecryptOptions{}
		WithSearchWindow(5)(&opts)
		assert.Equal(t, 5, opts.SearchWindowDays)
	})

	t.Run("WithExpectedTime sets time", func(t *testing.T) {
		expected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		opts := DecryptOptions{}
		WithExpectedTime(expected)(&opts)
		assert.Equal(t, expected, opts.ExpectedTime)
	})
}

func TestDecrypt_Errors(t *testing.T) {
	t.Run("rejects invalid key size", func(t *testing.T) {
		invalidKey := make([]byte, 24) // Invalid size
		packet := models.EncryptedPacket{
			Payload:   make([]byte, MinPacketSize),
			Timestamp: time.Now(),
		}

		_, err := Decrypt(invalidKey, packet)
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("rejects short packet", func(t *testing.T) {
		key := make([]byte, 16)
		packet := models.EncryptedPacket{
			Payload:   make([]byte, MinPacketSize-1),
			Timestamp: time.Now(),
		}

		_, err := Decrypt(key, packet)
		assert.ErrorIs(t, err, ErrPacketTooShort)
	})
}

func TestDecryptWithKnownCounter_Errors(t *testing.T) {
	t.Run("rejects invalid key size", func(t *testing.T) {
		invalidKey := make([]byte, 24)
		packet := models.EncryptedPacket{
			Payload: make([]byte, MinPacketSize),
		}

		_, err := DecryptWithKnownCounter(invalidKey, packet, 19000)
		assert.ErrorIs(t, err, ErrInvalidKey)
	})
}

func TestFindTimeCounter_Errors(t *testing.T) {
	t.Run("rejects invalid key size", func(t *testing.T) {
		invalidKey := make([]byte, 24)
		packet := models.EncryptedPacket{
			Payload: make([]byte, MinPacketSize),
		}

		_, err := FindTimeCounter(invalidKey, packet)
		assert.ErrorIs(t, err, ErrInvalidKey)
	})
}

// TestDecryptIntegration tests the full decryption flow with a synthetic packet.
// This creates a packet, encrypts it, and verifies decryption works.
func TestDecryptIntegration(t *testing.T) {
	t.Run("encrypt and decrypt roundtrip", func(t *testing.T) {
		// Master key
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}

		timeCounter := uint32(20000)
		seqCounter := uint32(42)
		plaintext := []byte("Hello, Hubble!")

		// Derive keys
		encKey, err := FullEncryptionKeyDerivation(key, timeCounter, seqCounter)
		require.NoError(t, err)

		nonce, err := FullNonceDerivation(key, timeCounter, seqCounter)
		require.NoError(t, err)

		// Encrypt the payload
		ciphertext, err := AESCTREncrypt(encKey, nonce, plaintext)
		require.NoError(t, err)

		// Build the packet
		// Header: sequence number (2 bytes) + reserved (4 bytes) = 6 bytes
		// Then auth tag (4 bytes), then ciphertext
		header := make([]byte, 6)
		header[0] = byte(seqCounter >> 8)
		header[1] = byte(seqCounter & 0xFF)

		// Compute auth tag over header
		authTag, err := ComputeAuthTag(encKey, header)
		require.NoError(t, err)

		// Assemble packet
		packet := make([]byte, 0, len(header)+len(authTag)+len(ciphertext))
		packet = append(packet, header...)
		packet = append(packet, authTag...)
		packet = append(packet, ciphertext...)

		// Create EncryptedPacket
		encPacket := models.EncryptedPacket{
			Payload:   packet,
			Timestamp: CounterToTime(timeCounter),
		}

		// Decrypt
		result, err := Decrypt(key, encPacket, WithSearchWindow(1))
		require.NoError(t, err)

		assert.Equal(t, plaintext, result.Payload)
		assert.Equal(t, timeCounter, result.TimeCounter)
		assert.Equal(t, seqCounter, result.SeqCounter)
	})

	t.Run("finds correct time counter within search window", func(t *testing.T) {
		key := make([]byte, 16)
		for i := range key {
			key[i] = byte(i)
		}

		actualTimeCounter := uint32(20002)
		seqCounter := uint32(100)
		plaintext := []byte("test payload")

		// Create encrypted packet at actualTimeCounter
		encKey, _ := FullEncryptionKeyDerivation(key, actualTimeCounter, seqCounter)
		nonce, _ := FullNonceDerivation(key, actualTimeCounter, seqCounter)
		ciphertext, _ := AESCTREncrypt(encKey, nonce, plaintext)

		header := make([]byte, 6)
		header[0] = byte(seqCounter >> 8)
		header[1] = byte(seqCounter & 0xFF)
		authTag, _ := ComputeAuthTag(encKey, header)

		packet := append(header, authTag...)
		packet = append(packet, ciphertext...)

		// Set expected time to 2 days before actual
		expectedTime := CounterToTime(actualTimeCounter - 2)

		encPacket := models.EncryptedPacket{
			Payload:   packet,
			Timestamp: expectedTime,
		}

		// Should find the correct counter within ±2 day window
		result, err := Decrypt(key, encPacket, WithSearchWindow(2))
		require.NoError(t, err)

		assert.Equal(t, plaintext, result.Payload)
		assert.Equal(t, actualTimeCounter, result.TimeCounter)
	})

	t.Run("DecryptWithKnownCounter success", func(t *testing.T) {
		key := make([]byte, 16)
		for i := range key {
			key[i] = byte(i)
		}

		timeCounter := uint32(20000)
		seqCounter := uint32(50)
		plaintext := []byte("known counter test")

		encKey, _ := FullEncryptionKeyDerivation(key, timeCounter, seqCounter)
		nonce, _ := FullNonceDerivation(key, timeCounter, seqCounter)
		ciphertext, _ := AESCTREncrypt(encKey, nonce, plaintext)

		header := make([]byte, 6)
		header[0] = byte(seqCounter >> 8)
		header[1] = byte(seqCounter & 0xFF)
		authTag, _ := ComputeAuthTag(encKey, header)

		packet := append(header, authTag...)
		packet = append(packet, ciphertext...)

		encPacket := models.EncryptedPacket{
			Payload: packet,
		}

		result, err := DecryptWithKnownCounter(key, encPacket, timeCounter)
		require.NoError(t, err)
		assert.Equal(t, plaintext, result.Payload)
	})

	t.Run("DecryptWithKnownCounter wrong counter fails", func(t *testing.T) {
		key := make([]byte, 16)
		timeCounter := uint32(20000)
		seqCounter := uint32(50)

		encKey, _ := FullEncryptionKeyDerivation(key, timeCounter, seqCounter)
		header := make([]byte, 6)
		header[0] = byte(seqCounter >> 8)
		header[1] = byte(seqCounter & 0xFF)
		authTag, _ := ComputeAuthTag(encKey, header)

		packet := append(header, authTag...)
		packet = append(packet, []byte("encrypted")...)

		encPacket := models.EncryptedPacket{
			Payload: packet,
		}

		// Use wrong time counter
		_, err := DecryptWithKnownCounter(key, encPacket, timeCounter+10)
		assert.Error(t, err)
	})

	t.Run("FindTimeCounter success", func(t *testing.T) {
		key := make([]byte, 16)
		for i := range key {
			key[i] = byte(i)
		}

		timeCounter := uint32(20001)
		seqCounter := uint32(75)

		encKey, _ := FullEncryptionKeyDerivation(key, timeCounter, seqCounter)
		header := make([]byte, 6)
		header[0] = byte(seqCounter >> 8)
		header[1] = byte(seqCounter & 0xFF)
		authTag, _ := ComputeAuthTag(encKey, header)

		packet := append(header, authTag...)
		packet = append(packet, []byte("payload")...)

		encPacket := models.EncryptedPacket{
			Payload:   packet,
			Timestamp: CounterToTime(timeCounter),
		}

		found, err := FindTimeCounter(key, encPacket, WithSearchWindow(1))
		require.NoError(t, err)
		assert.Equal(t, timeCounter, found)
	})

	t.Run("FindTimeCounter not found", func(t *testing.T) {
		key := make([]byte, 16)
		// Create packet with random auth tag that won't match
		packet := make([]byte, MinPacketSize+4)
		packet[6] = 0xFF
		packet[7] = 0xFF
		packet[8] = 0xFF
		packet[9] = 0xFF

		encPacket := models.EncryptedPacket{
			Payload:   packet,
			Timestamp: time.Now(),
		}

		_, err := FindTimeCounter(key, encPacket, WithSearchWindow(1))
		assert.ErrorIs(t, err, ErrDecryptionFailed)
	})

	t.Run("FindTimeCounter rejects short packet", func(t *testing.T) {
		key := make([]byte, 16)
		packet := models.EncryptedPacket{
			Payload: make([]byte, MinPacketSize-1),
		}

		_, err := FindTimeCounter(key, packet)
		assert.ErrorIs(t, err, ErrPacketTooShort)
	})

	t.Run("Decrypt with zero timestamp uses current time", func(t *testing.T) {
		key := make([]byte, 16)
		for i := range key {
			key[i] = byte(i)
		}

		// Use current day's counter
		timeCounter := TimeToCounter(time.Now().UTC())
		seqCounter := uint32(99)
		plaintext := []byte("current time test")

		encKey, _ := FullEncryptionKeyDerivation(key, timeCounter, seqCounter)
		nonce, _ := FullNonceDerivation(key, timeCounter, seqCounter)
		ciphertext, _ := AESCTREncrypt(encKey, nonce, plaintext)

		header := make([]byte, 6)
		header[0] = byte(seqCounter >> 8)
		header[1] = byte(seqCounter & 0xFF)
		authTag, _ := ComputeAuthTag(encKey, header)

		packet := append(header, authTag...)
		packet = append(packet, ciphertext...)

		// Zero timestamp - should default to current time
		encPacket := models.EncryptedPacket{
			Payload:   packet,
			Timestamp: time.Time{},
		}

		result, err := Decrypt(key, encPacket, WithSearchWindow(1))
		require.NoError(t, err)
		assert.Equal(t, plaintext, result.Payload)
	})

	t.Run("Decrypt fails when counter outside search window", func(t *testing.T) {
		key := make([]byte, 16)
		timeCounter := uint32(20000)
		seqCounter := uint32(50)

		encKey, _ := FullEncryptionKeyDerivation(key, timeCounter, seqCounter)
		header := make([]byte, 6)
		header[0] = byte(seqCounter >> 8)
		header[1] = byte(seqCounter & 0xFF)
		authTag, _ := ComputeAuthTag(encKey, header)

		packet := append(header, authTag...)
		packet = append(packet, []byte("payload")...)

		// Set expected time 10 days away with only 2 day search window
		encPacket := models.EncryptedPacket{
			Payload:   packet,
			Timestamp: CounterToTime(timeCounter + 10),
		}

		_, err := Decrypt(key, encPacket, WithSearchWindow(2))
		assert.ErrorIs(t, err, ErrDecryptionFailed)
	})
}
