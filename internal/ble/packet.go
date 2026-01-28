package ble

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/hubblenetwork/hubcli/internal/models"
)

// Hubble BLE Service UUID: 0xFCA6
const (
	// HubbleServiceUUID is the 16-bit UUID for Hubble BLE advertisements
	HubbleServiceUUID16 = 0xFCA6

	// HubbleServiceUUID is the full 128-bit UUID string
	HubbleServiceUUID = "0000fca6-0000-1000-8000-00805f9b34fb"

	// MinPayloadLength is the minimum expected payload length
	MinPayloadLength = 8

	// MaxPayloadLength is the maximum expected payload length
	MaxPayloadLength = 31
)

var (
	// ErrInvalidPayload indicates the BLE payload is malformed
	ErrInvalidPayload = errors.New("invalid BLE payload")

	// ErrPayloadTooShort indicates the payload is too short
	ErrPayloadTooShort = errors.New("payload too short")

	// ErrNotHubblePacket indicates the packet is not a Hubble advertisement
	ErrNotHubblePacket = errors.New("not a Hubble BLE packet")
)

// RawAdvertisement represents a raw BLE advertisement received from scanning
type RawAdvertisement struct {
	// LocalName is the advertised device name (if any)
	LocalName string

	// ServiceUUIDs contains the advertised service UUIDs
	ServiceUUIDs []string

	// ServiceData maps service UUIDs to their data
	ServiceData map[string][]byte

	// ManufacturerData contains manufacturer-specific data
	ManufacturerData []byte

	// RSSI is the received signal strength indicator
	RSSI int

	// Address is the BLE device address
	Address string

	// Timestamp when the advertisement was received
	Timestamp time.Time
}

// ParseAdvertisement extracts a Hubble packet from a raw BLE advertisement
func ParseAdvertisement(adv RawAdvertisement, loc models.Location) (*models.EncryptedPacket, error) {
	// Look for Hubble service data first
	for uuid, data := range adv.ServiceData {
		if isHubbleUUID(uuid) && len(data) >= MinPayloadLength {
			return &models.EncryptedPacket{
				Payload:   data,
				RSSI:      adv.RSSI,
				Timestamp: adv.Timestamp,
				Location:  loc,
			}, nil
		}
	}

	// Fall back to manufacturer data if present
	if len(adv.ManufacturerData) >= MinPayloadLength {
		return &models.EncryptedPacket{
			Payload:   adv.ManufacturerData,
			RSSI:      adv.RSSI,
			Timestamp: adv.Timestamp,
			Location:  loc,
		}, nil
	}

	return nil, ErrNotHubblePacket
}

// isHubbleUUID checks if a UUID string matches the Hubble service UUID
func isHubbleUUID(uuid string) bool {
	// Check full UUID
	if uuid == HubbleServiceUUID {
		return true
	}

	// Check 16-bit short form (as hex string)
	if uuid == "fca6" || uuid == "FCA6" || uuid == "0xfca6" || uuid == "0xFCA6" {
		return true
	}

	return false
}

// ContainsHubbleService checks if the advertisement contains the Hubble service UUID
func ContainsHubbleService(adv RawAdvertisement) bool {
	// Check service UUIDs list
	for _, uuid := range adv.ServiceUUIDs {
		if isHubbleUUID(uuid) {
			return true
		}
	}

	// Check service data keys
	for uuid := range adv.ServiceData {
		if isHubbleUUID(uuid) {
			return true
		}
	}

	return false
}

// ExtractDeviceID attempts to extract a device ID from the packet payload
// The device ID is typically in the first 4 bytes of the payload
func ExtractDeviceID(payload []byte) (uint32, error) {
	if len(payload) < 4 {
		return 0, ErrPayloadTooShort
	}

	// Device ID is stored in little-endian format
	return binary.LittleEndian.Uint32(payload[:4]), nil
}

// PacketInfo contains parsed information about a BLE packet
type PacketInfo struct {
	// DeviceIDBytes is the raw device ID portion
	DeviceIDBytes []byte

	// EncryptedData is the encrypted payload portion
	EncryptedData []byte

	// AuthTag is the authentication tag (if present)
	AuthTag []byte

	// FullPayload is the complete raw payload
	FullPayload []byte
}

// ParsePacketStructure breaks down a raw payload into its components
// Hubble packet structure:
// - Bytes 0-3: Device ID (4 bytes)
// - Bytes 4-N: Encrypted data + auth tag
func ParsePacketStructure(payload []byte) (*PacketInfo, error) {
	if len(payload) < MinPayloadLength {
		return nil, ErrPayloadTooShort
	}

	info := &PacketInfo{
		DeviceIDBytes: payload[:4],
		FullPayload:   payload,
	}

	// The remaining bytes are encrypted data
	if len(payload) > 4 {
		// Last 4 bytes are typically the auth tag
		if len(payload) > 8 {
			info.EncryptedData = payload[4 : len(payload)-4]
			info.AuthTag = payload[len(payload)-4:]
		} else {
			info.EncryptedData = payload[4:]
		}
	}

	return info, nil
}
