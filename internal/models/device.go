package models

import "time"

// EncryptionType represents supported encryption algorithms.
type EncryptionType string

const (
	EncryptionAES256CTR EncryptionType = "AES-256-CTR"
	EncryptionAES128CTR EncryptionType = "AES-128-CTR"
)

// PacketTimestamp contains a timestamp for a packet.
type PacketTimestamp struct {
	Timestamp float64 `json:"timestamp"`
}

// MostRecentPacketInfo contains info about the most recent packet from a device.
type MostRecentPacketInfo struct {
	Terrestrial *PacketTimestamp `json:"terrestrial,omitempty"`
	Satellite   *PacketTimestamp `json:"satellite,omitempty"`
}

// Device represents a registered Hubble device.
type Device struct {
	ID               string                `json:"id"`
	Name             string                `json:"name,omitempty"`
	Key              string                `json:"key,omitempty"`  // Base64-encoded encryption key
	Encryption       EncryptionType        `json:"encryption,omitempty"` // "AES-256-CTR" or "AES-128-CTR"
	Tags             map[string]string     `json:"tags,omitempty"`
	Active           bool                  `json:"active,omitempty"`
	CreatedTS        int64                 `json:"created_ts,omitempty"`         // Unix timestamp
	MostRecentPacket *MostRecentPacketInfo `json:"most_recent_packet,omitempty"` // Info about last packet
	CreatedAt        time.Time             `json:"-"`                            // Computed from CreatedTS
}

// RegisterDeviceRequest is the payload for registering a new device.
type RegisterDeviceRequest struct {
	NDevices   int            `json:"n_devices,omitempty"`
	Encryption EncryptionType `json:"encryption,omitempty"`
}

// UpdateDeviceRequest is the payload for updating device metadata.
type UpdateDeviceRequest struct {
	SetName *string            `json:"set_name,omitempty"`
	SetTags *map[string]string `json:"set_tags,omitempty"`
}
