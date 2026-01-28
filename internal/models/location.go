package models

import "time"

// Location represents geographic coordinates with accuracy metadata.
type Location struct {
	Latitude           float64   `json:"latitude"`
	Longitude          float64   `json:"longitude"`
	Timestamp          time.Time `json:"timestamp"`
	HorizontalAccuracy float64   `json:"horizontal_accuracy,omitempty"`
	Altitude           float64   `json:"altitude,omitempty"`
	VerticalAccuracy   float64   `json:"vertical_accuracy,omitempty"`
	Fake               bool      `json:"fake,omitempty"`
}

// NewFakeLocation returns a placeholder location for local BLE scans
// where real GPS coordinates are not available.
func NewFakeLocation() Location {
	return Location{
		Latitude:  90,
		Longitude: 0,
		Timestamp: time.Now().UTC(),
		Fake:      true,
	}
}
