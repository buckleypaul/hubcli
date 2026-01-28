package ble

import (
	"testing"
	"time"

	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestIsHubbleUUID(t *testing.T) {
	tests := []struct {
		uuid     string
		expected bool
	}{
		{HubbleServiceUUID, true},
		{"fca6", true},
		{"FCA6", true},
		{"0xfca6", true},
		{"0xFCA6", true},
		{"0000fca6-0000-1000-8000-00805f9b34fb", true},
		{"00001234-0000-1000-8000-00805f9b34fb", false},
		{"1234", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isHubbleUUID(tt.uuid)
		assert.Equal(t, tt.expected, result, "isHubbleUUID(%q)", tt.uuid)
	}
}

func TestContainsHubbleService(t *testing.T) {
	tests := []struct {
		name     string
		adv      RawAdvertisement
		expected bool
	}{
		{
			name: "has hubble service UUID in list",
			adv: RawAdvertisement{
				ServiceUUIDs: []string{"fca6"},
			},
			expected: true,
		},
		{
			name: "has hubble service data",
			adv: RawAdvertisement{
				ServiceData: map[string][]byte{
					HubbleServiceUUID: {0x01, 0x02},
				},
			},
			expected: true,
		},
		{
			name: "no hubble service",
			adv: RawAdvertisement{
				ServiceUUIDs: []string{"1234"},
				ServiceData: map[string][]byte{
					"5678": {0x01},
				},
			},
			expected: false,
		},
		{
			name:     "empty advertisement",
			adv:      RawAdvertisement{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsHubbleService(tt.adv)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseAdvertisement(t *testing.T) {
	now := time.Now()
	loc := models.Location{Latitude: 37.7749, Longitude: -122.4194}

	tests := []struct {
		name        string
		adv         RawAdvertisement
		loc         models.Location
		expectError error
		checkPacket func(*testing.T, *models.EncryptedPacket)
	}{
		{
			name: "valid service data",
			adv: RawAdvertisement{
				ServiceData: map[string][]byte{
					HubbleServiceUUID: {0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				},
				RSSI:      -65,
				Timestamp: now,
			},
			loc: loc,
			checkPacket: func(t *testing.T, p *models.EncryptedPacket) {
				assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, p.Payload)
				assert.Equal(t, -65, p.RSSI)
				assert.Equal(t, loc, p.Location)
			},
		},
		{
			name: "valid manufacturer data fallback",
			adv: RawAdvertisement{
				ManufacturerData: []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22},
				RSSI:             -70,
				Timestamp:        now,
			},
			loc: loc,
			checkPacket: func(t *testing.T, p *models.EncryptedPacket) {
				assert.Equal(t, []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22}, p.Payload)
				assert.Equal(t, -70, p.RSSI)
			},
		},
		{
			name: "service data too short",
			adv: RawAdvertisement{
				ServiceData: map[string][]byte{
					HubbleServiceUUID: {0x01, 0x02},
				},
			},
			loc:         loc,
			expectError: ErrNotHubblePacket,
		},
		{
			name:        "no data",
			adv:         RawAdvertisement{},
			loc:         loc,
			expectError: ErrNotHubblePacket,
		},
		{
			name: "non-hubble service data only",
			adv: RawAdvertisement{
				ServiceData: map[string][]byte{
					"00001234-0000-1000-8000-00805f9b34fb": {0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				},
			},
			loc:         loc,
			expectError: ErrNotHubblePacket,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet, err := ParseAdvertisement(tt.adv, tt.loc)

			if tt.expectError != nil {
				assert.ErrorIs(t, err, tt.expectError)
				assert.Nil(t, packet)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, packet)
				if tt.checkPacket != nil {
					tt.checkPacket(t, packet)
				}
			}
		})
	}
}

func TestExtractDeviceID(t *testing.T) {
	tests := []struct {
		name        string
		payload     []byte
		expected    uint32
		expectError error
	}{
		{
			name:     "valid device ID",
			payload:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
			expected: 0x04030201, // Little endian
		},
		{
			name:     "minimum length",
			payload:  []byte{0xFF, 0xFF, 0xFF, 0xFF},
			expected: 0xFFFFFFFF,
		},
		{
			name:        "payload too short",
			payload:     []byte{0x01, 0x02, 0x03},
			expectError: ErrPayloadTooShort,
		},
		{
			name:        "empty payload",
			payload:     []byte{},
			expectError: ErrPayloadTooShort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deviceID, err := ExtractDeviceID(tt.payload)

			if tt.expectError != nil {
				assert.ErrorIs(t, err, tt.expectError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, deviceID)
			}
		})
	}
}

func TestParsePacketStructure(t *testing.T) {
	tests := []struct {
		name        string
		payload     []byte
		expectError error
		check       func(*testing.T, *PacketInfo)
	}{
		{
			name:    "valid packet with auth tag",
			payload: []byte{0x01, 0x02, 0x03, 0x04, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11, 0x22},
			check: func(t *testing.T, info *PacketInfo) {
				assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, info.DeviceIDBytes)
				assert.Equal(t, []byte{0xAA, 0xBB, 0xCC, 0xDD}, info.EncryptedData)
				assert.Equal(t, []byte{0xEE, 0xFF, 0x11, 0x22}, info.AuthTag)
			},
		},
		{
			name:    "minimum valid packet",
			payload: []byte{0x01, 0x02, 0x03, 0x04, 0xAA, 0xBB, 0xCC, 0xDD},
			check: func(t *testing.T, info *PacketInfo) {
				assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, info.DeviceIDBytes)
				assert.Equal(t, []byte{0xAA, 0xBB, 0xCC, 0xDD}, info.EncryptedData)
				assert.Empty(t, info.AuthTag)
			},
		},
		{
			name:        "payload too short",
			payload:     []byte{0x01, 0x02, 0x03},
			expectError: ErrPayloadTooShort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParsePacketStructure(tt.payload)

			if tt.expectError != nil {
				assert.ErrorIs(t, err, tt.expectError)
				assert.Nil(t, info)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, info)
				if tt.check != nil {
					tt.check(t, info)
				}
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Verify constants are set correctly
	assert.Equal(t, uint16(0xFCA6), uint16(HubbleServiceUUID16))
	assert.Equal(t, "0000fca6-0000-1000-8000-00805f9b34fb", HubbleServiceUUID)
	assert.Equal(t, 8, MinPayloadLength)
	assert.Equal(t, 31, MaxPayloadLength)
}
