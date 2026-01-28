package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/hubblenetwork/hubcli/internal/models"
)

// RetrievePacketsOptions configures packet retrieval.
type RetrievePacketsOptions struct {
	DeviceID          *string
	Start             *time.Time
	Days              int    // If Start is nil, query from (now - Days) to now
	Limit             int    // Maximum number of packets to retrieve (0 = no limit)
	ContinuationToken string // Token to continue from a previous request
}

// RetrievePacketsResult contains packets and pagination info.
type RetrievePacketsResult struct {
	Packets           []models.RetrievedPacket
	ContinuationToken string // Non-empty if more packets are available
}

// RetrievePackets fetches decrypted packets from the cloud.
// By default, retrieves packets from the last 7 days.
func (c *Client) RetrievePackets(ctx context.Context, opts RetrievePacketsOptions) ([]models.RetrievedPacket, error) {
	result, err := c.RetrievePacketsWithPagination(ctx, opts)
	if err != nil {
		return nil, err
	}
	return result.Packets, nil
}

// RetrievePacketsWithPagination fetches packets with pagination support.
// Returns packets and a continuation token if more are available.
func (c *Client) RetrievePacketsWithPagination(ctx context.Context, opts RetrievePacketsOptions) (*RetrievePacketsResult, error) {
	path := fmt.Sprintf("/org/%s/packets", c.orgID)

	// Build query parameters
	params := url.Values{}

	if opts.DeviceID != nil {
		params.Set("device_id", *opts.DeviceID)
	}

	// Calculate start time
	var start time.Time
	if opts.Start != nil {
		start = *opts.Start
	} else {
		days := opts.Days
		if days == 0 {
			days = 7 // Default to 7 days
		}
		start = time.Now().UTC().AddDate(0, 0, -days)
	}
	params.Set("start", strconv.FormatInt(start.Unix(), 10))

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var allPackets []models.RetrievedPacket
	contToken := opts.ContinuationToken

	// Handle pagination
	for {
		body, headers, err := c.getWithContToken(ctx, path, contToken)
		if err != nil {
			return nil, err
		}

		// API returns {"packets": [...]}
		var response struct {
			Packets []models.RetrievedPacket `json:"packets"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse packets response: %w", err)
		}

		allPackets = append(allPackets, response.Packets...)

		// Check for continuation token in response header
		contToken = headers.Get("Continuation-Token")

		// Stop if we've reached the limit
		if opts.Limit > 0 && len(allPackets) >= opts.Limit {
			// Trim to exact limit
			if len(allPackets) > opts.Limit {
				allPackets = allPackets[:opts.Limit]
			}
			// Keep the continuation token to indicate more are available
			break
		}

		if contToken == "" {
			break
		}
	}

	return &RetrievePacketsResult{
		Packets:           allPackets,
		ContinuationToken: contToken,
	}, nil
}

// IngestPacket uploads encrypted BLE packets to the cloud for processing.
func (c *Client) IngestPacket(ctx context.Context, req models.IngestPacketRequest) error {
	path := fmt.Sprintf("/org/%s/packets", c.orgID)
	_, _, err := c.post(ctx, path, req)
	return err
}

// IngestEncryptedPackets is a convenience method to ingest multiple EncryptedPacket structs.
func (c *Client) IngestEncryptedPackets(ctx context.Context, packets []models.EncryptedPacket) error {
	if len(packets) == 0 {
		return nil
	}

	// Group packets by location (for now, treat each packet as its own location)
	var bleLocations []models.BLELocation

	for _, p := range packets {
		loc := models.BLELocation{
			Location: models.LocationPayload{
				Latitude:           p.Location.Latitude,
				Longitude:          p.Location.Longitude,
				Timestamp:          p.Location.Timestamp.Unix(),
				HorizontalAccuracy: p.Location.HorizontalAccuracy,
				Altitude:           p.Location.Altitude,
				VerticalAccuracy:   p.Location.VerticalAccuracy,
			},
			Adverstments: []models.BLEAdvertisement{
				{
					Payload:   encodeBase64(p.Payload),
					RSSI:      p.RSSI,
					Timestamp: p.Timestamp.Unix(),
				},
			},
		}
		bleLocations = append(bleLocations, loc)
	}

	return c.IngestPacket(ctx, models.IngestPacketRequest{
		BLELocations: bleLocations,
	})
}

// encodeBase64 encodes bytes to base64 string.
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
