package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hubblenetwork/hubcli/internal/models"
)

// ListDevices returns all devices registered to the organization.
// Handles pagination automatically to retrieve all devices.
func (c *Client) ListDevices(ctx context.Context) ([]models.Device, error) {
	path := fmt.Sprintf("/org/%s/devices", c.orgID)

	var allDevices []models.Device
	var contToken string

	// Handle pagination
	for {
		body, headers, err := c.getWithContToken(ctx, path, contToken)
		if err != nil {
			return nil, err
		}

		// API returns {"devices": [...]}
		var response struct {
			Devices []models.Device `json:"devices"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse devices response: %w", err)
		}

		allDevices = append(allDevices, response.Devices...)

		// Check for continuation token in response header
		contToken = headers.Get("Continuation-Token")
		if contToken == "" {
			break
		}
	}

	return allDevices, nil
}

// RegisterDevice creates a new device with the specified encryption type.
// If encryption is empty, defaults to AES-256-CTR.
func (c *Client) RegisterDevice(ctx context.Context, req models.RegisterDeviceRequest) (*models.Device, error) {
	path := fmt.Sprintf("/v2/org/%s/devices", c.orgID)

	// Set defaults
	if req.NDevices == 0 {
		req.NDevices = 1
	}
	if req.Encryption == "" {
		req.Encryption = models.EncryptionAES256CTR
	}

	body, _, err := c.post(ctx, path, req)
	if err != nil {
		return nil, err
	}

	// The API returns a list of devices even when registering one
	var devices []models.Device
	if err := json.Unmarshal(body, &devices); err != nil {
		return nil, fmt.Errorf("failed to parse register device response: %w", err)
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no device returned from registration")
	}

	return &devices[0], nil
}

// UpdateDevice updates device metadata (name and/or tags).
func (c *Client) UpdateDevice(ctx context.Context, deviceID string, req models.UpdateDeviceRequest) (*models.Device, error) {
	path := fmt.Sprintf("/org/%s/devices/%s", c.orgID, deviceID)

	body, _, err := c.patch(ctx, path, req)
	if err != nil {
		return nil, err
	}

	var device models.Device
	if err := json.Unmarshal(body, &device); err != nil {
		return nil, fmt.Errorf("failed to parse update device response: %w", err)
	}

	return &device, nil
}

// SetDeviceName is a convenience method to update just the device name.
func (c *Client) SetDeviceName(ctx context.Context, deviceID, name string) (*models.Device, error) {
	return c.UpdateDevice(ctx, deviceID, models.UpdateDeviceRequest{
		SetName: &name,
	})
}

// SetDeviceTags is a convenience method to update just the device tags.
func (c *Client) SetDeviceTags(ctx context.Context, deviceID string, tags map[string]string) (*models.Device, error) {
	return c.UpdateDevice(ctx, deviceID, models.UpdateDeviceRequest{
		SetTags: &tags,
	})
}

// DeleteDevice deletes a device by its ID.
func (c *Client) DeleteDevice(ctx context.Context, deviceID string) error {
	path := fmt.Sprintf("/org/%s/devices/%s", c.orgID, deviceID)

	_, _, err := c.delete(ctx, path)
	if err != nil {
		return err
	}

	return nil
}
