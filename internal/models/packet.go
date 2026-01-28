package models

import (
	"encoding/hex"
	"time"
)

// EncryptedPacket represents a raw BLE advertisement captured from scanning.
type EncryptedPacket struct {
	Payload   []byte    `json:"payload"`
	RSSI      int       `json:"rssi"`
	Timestamp time.Time `json:"timestamp"`
	Location  Location  `json:"location"`
}

// PayloadHex returns the payload as a hexadecimal string.
func (p EncryptedPacket) PayloadHex() string {
	return hex.EncodeToString(p.Payload)
}

// DecryptedPacket represents a successfully decrypted packet.
type DecryptedPacket struct {
	DeviceID    string    `json:"device_id"`
	Payload     []byte    `json:"payload"`
	TimeCounter uint32    `json:"time_counter"`
	Timestamp   time.Time `json:"timestamp"`
	Location    Location  `json:"location"`
}

// PayloadHex returns the decrypted payload as a hexadecimal string.
func (p DecryptedPacket) PayloadHex() string {
	return hex.EncodeToString(p.Payload)
}

// BLEAdvertisement represents a single BLE advertisement for ingestion.
type BLEAdvertisement struct {
	Payload   string `json:"payload"` // Base64-encoded
	RSSI      int    `json:"rssi"`
	Timestamp int64  `json:"timestamp"` // Unix timestamp
}

// BLELocation represents a location with associated BLE advertisements.
type BLELocation struct {
	Location     LocationPayload    `json:"location"`
	Adverstments []BLEAdvertisement `json:"adv"`
}

// LocationPayload is the JSON structure for location in packet ingestion.
type LocationPayload struct {
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	Timestamp          int64   `json:"timestamp"`
	HorizontalAccuracy float64 `json:"horizontal_accuracy,omitempty"`
	Altitude           float64 `json:"altitude,omitempty"`
	VerticalAccuracy   float64 `json:"vertical_accuracy,omitempty"`
}

// IngestPacketRequest is the payload for ingesting encrypted packets.
type IngestPacketRequest struct {
	BLELocations []BLELocation `json:"ble_locations"`
}

// RetrievedPacket represents a packet returned from the cloud API.
type RetrievedPacket struct {
	Location    RetrievedLocation `json:"location"`
	Device      RetrievedDevice   `json:"device"`
	NetworkType string            `json:"network_type"`
}

// RetrievedLocation is the location data in a retrieved packet.
type RetrievedLocation struct {
	Timestamp          float64 `json:"timestamp"` // Unix timestamp as float
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	Altitude           float64 `json:"altitude"`
	HorizontalAccuracy float64 `json:"horizontal_accuracy"`
	VerticalAccuracy   float64 `json:"vertical_accuracy"`
}

// RetrievedDevice is the device data in a retrieved packet.
type RetrievedDevice struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Tags           map[string]string `json:"tags"`
	Payload        string            `json:"payload"` // Base64-encoded
	Timestamp      float64           `json:"timestamp"`
	RSSI           int               `json:"rssi"`
	SequenceNumber int               `json:"sequence_number"`
	Counter        int               `json:"counter"`
}

// DeviceID returns the device ID from the nested device object.
func (p RetrievedPacket) DeviceID() string {
	return p.Device.ID
}

// Payload returns the payload from the nested device object.
func (p RetrievedPacket) Payload() string {
	return p.Device.Payload
}

// Timestamp returns the timestamp as time.Time.
func (p RetrievedPacket) Timestamp() time.Time {
	return time.Unix(int64(p.Device.Timestamp), int64((p.Device.Timestamp-float64(int64(p.Device.Timestamp)))*1e9))
}

// GetLocation returns the location as a Location struct.
func (p RetrievedPacket) GetLocation() Location {
	return Location{
		Latitude:           p.Location.Latitude,
		Longitude:          p.Location.Longitude,
		Altitude:           p.Location.Altitude,
		HorizontalAccuracy: p.Location.HorizontalAccuracy,
		VerticalAccuracy:   p.Location.VerticalAccuracy,
		Timestamp:          time.Unix(int64(p.Location.Timestamp), 0),
	}
}
