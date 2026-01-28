//go:build integration

package api

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests require the following environment variables:
// - HUBBLE_ORG_ID: Your organization ID
// - HUBBLE_API_TOKEN: Your API token
//
// Run with: go test -tags=integration ./internal/api/...

func getTestClient(t *testing.T) *Client {
	orgID := os.Getenv("HUBBLE_ORG_ID")
	token := os.Getenv("HUBBLE_API_TOKEN")

	if orgID == "" || token == "" {
		t.Skip("Integration tests require HUBBLE_ORG_ID and HUBBLE_API_TOKEN environment variables")
	}

	return NewClient(orgID, token)
}

func TestIntegration_CheckCredentials(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := client.CheckCredentials(ctx)
	require.NoError(t, err, "credentials should be valid")
}

func TestIntegration_CheckCredentials_Invalid(t *testing.T) {
	client := NewClient("invalid-org", "invalid-token")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := client.CheckCredentials(ctx)
	assert.Error(t, err, "invalid credentials should return error")
}

func TestIntegration_GetOrganization(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	org, err := client.GetOrganization(ctx)
	require.NoError(t, err)
	require.NotNil(t, org)

	assert.NotEmpty(t, org.ID, "organization ID should not be empty")
	t.Logf("Organization: ID=%s, Name=%s", org.ID, org.Name)
}

func TestIntegration_ListDevices(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	devices, err := client.ListDevices(ctx)
	require.NoError(t, err)
	require.NotNil(t, devices)

	t.Logf("Found %d devices", len(devices))

	for _, d := range devices {
		t.Logf("Device: ID=%s, Name=%s, Encryption=%s", d.ID, d.Name, d.Encryption)
	}
}

func TestIntegration_RegisterDevice(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Register a new device with AES-256-CTR encryption
	req := models.RegisterDeviceRequest{
		Encryption: models.EncryptionAES256CTR,
	}

	device, err := client.RegisterDevice(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, device)

	assert.NotEmpty(t, device.ID, "device ID should not be empty")
	assert.NotEmpty(t, device.Key, "device key should not be empty")
	assert.Equal(t, models.EncryptionAES256CTR, device.Encryption)

	t.Logf("Registered device: ID=%s", device.ID)

	// Verify device appears in list
	devices, err := client.ListDevices(ctx)
	require.NoError(t, err)

	found := false
	for _, d := range devices {
		if d.ID == device.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "newly registered device should appear in device list")
}

func TestIntegration_UpdateDevice(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First, register a device
	device, err := client.RegisterDevice(ctx, models.RegisterDeviceRequest{
		Encryption: models.EncryptionAES256CTR,
	})
	require.NoError(t, err)
	require.NotNil(t, device)

	// Update the device name
	testName := "Integration Test Device"
	err = client.SetDeviceName(ctx, device.ID, testName)
	require.NoError(t, err)

	// Verify the name was updated by listing devices
	devices, err := client.ListDevices(ctx)
	require.NoError(t, err)

	var updatedDevice *models.Device
	for i := range devices {
		if devices[i].ID == device.ID {
			updatedDevice = &devices[i]
			break
		}
	}

	require.NotNil(t, updatedDevice, "device should still exist")
	assert.Equal(t, testName, updatedDevice.Name)

	t.Logf("Updated device name to: %s", testName)
}

func TestIntegration_SetDeviceTags(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First, register a device
	device, err := client.RegisterDevice(ctx, models.RegisterDeviceRequest{
		Encryption: models.EncryptionAES256CTR,
	})
	require.NoError(t, err)
	require.NotNil(t, device)

	// Set tags on the device
	tags := map[string]string{
		"environment": "test",
		"owner":       "integration-tests",
	}
	err = client.SetDeviceTags(ctx, device.ID, tags)
	require.NoError(t, err)

	t.Logf("Set tags on device %s: %v", device.ID, tags)
}

func TestIntegration_RetrievePackets(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	opts := RetrievePacketsOptions{
		Days: 7,
	}

	packets, err := client.RetrievePackets(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, packets)

	t.Logf("Retrieved %d packets from the last 7 days", len(packets))

	// Log first few packets for inspection
	for i, p := range packets {
		if i >= 5 {
			break
		}
		t.Logf("Packet %d: DeviceID=%s, Timestamp=%s, Location=(%.4f, %.4f)",
			i+1, p.DeviceID, p.Timestamp.Format(time.RFC3339),
			p.Location.Latitude, p.Location.Longitude)
	}
}

func TestIntegration_RetrievePackets_WithDeviceFilter(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First get list of devices
	devices, err := client.ListDevices(ctx)
	require.NoError(t, err)

	if len(devices) == 0 {
		t.Skip("No devices found, skipping device-filtered packet retrieval test")
	}

	// Use first device ID as filter
	deviceID := devices[0].ID
	opts := RetrievePacketsOptions{
		Days:     7,
		DeviceID: &deviceID,
	}

	packets, err := client.RetrievePackets(ctx, opts)
	require.NoError(t, err)

	t.Logf("Retrieved %d packets for device %s", len(packets), deviceID)

	// Verify all packets are for the specified device
	for _, p := range packets {
		assert.Equal(t, deviceID, p.DeviceID, "packet should be from the filtered device")
	}
}

func TestIntegration_FullWorkflow(t *testing.T) {
	client := getTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Step 1: Validate credentials
	t.Log("Step 1: Validating credentials...")
	err := client.CheckCredentials(ctx)
	require.NoError(t, err, "credentials should be valid")

	// Step 2: Get organization info
	t.Log("Step 2: Getting organization info...")
	org, err := client.GetOrganization(ctx)
	require.NoError(t, err)
	t.Logf("Organization: %s (%s)", org.Name, org.ID)

	// Step 3: List existing devices
	t.Log("Step 3: Listing devices...")
	devicesBefore, err := client.ListDevices(ctx)
	require.NoError(t, err)
	t.Logf("Found %d existing devices", len(devicesBefore))

	// Step 4: Register a new device
	t.Log("Step 4: Registering new device...")
	newDevice, err := client.RegisterDevice(ctx, models.RegisterDeviceRequest{
		Encryption: models.EncryptionAES256CTR,
	})
	require.NoError(t, err)
	t.Logf("Registered device: %s", newDevice.ID)

	// Step 5: Update device name
	t.Log("Step 5: Updating device name...")
	deviceName := "Workflow Test Device"
	err = client.SetDeviceName(ctx, newDevice.ID, deviceName)
	require.NoError(t, err)

	// Step 6: Verify device count increased
	t.Log("Step 6: Verifying device list...")
	devicesAfter, err := client.ListDevices(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(devicesBefore)+1, len(devicesAfter), "device count should increase by 1")

	// Step 7: Retrieve packets
	t.Log("Step 7: Retrieving packets...")
	packets, err := client.RetrievePackets(ctx, RetrievePacketsOptions{Days: 1})
	require.NoError(t, err)
	t.Logf("Retrieved %d packets from the last day", len(packets))

	t.Log("Full workflow completed successfully!")
}

// TestIntegration_RateLimiting tests behavior when rate limited
// This test is skipped by default as it intentionally triggers rate limiting
func TestIntegration_RateLimiting(t *testing.T) {
	t.Skip("Skipping rate limit test to avoid hitting API limits")

	client := getTestClient(t)
	ctx := context.Background()

	// Make many rapid requests to trigger rate limiting
	var rateLimited bool
	for i := 0; i < 100; i++ {
		_, err := client.ListDevices(ctx)
		if err != nil {
			if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 429 {
				rateLimited = true
				t.Logf("Rate limited after %d requests", i+1)
				break
			}
		}
	}

	if !rateLimited {
		t.Log("Did not hit rate limit after 100 requests")
	}
}
