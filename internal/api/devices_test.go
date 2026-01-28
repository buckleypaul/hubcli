package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListDevices(t *testing.T) {
	t.Run("success with devices", func(t *testing.T) {
		createdAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/org/test-org/devices", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)

			devices := []models.Device{
				{
					ID:         "dev-001",
					Name:       "Test Device 1",
					Key:        "base64encodedkey1==",
					Encryption: models.EncryptionAES256CTR,
					CreatedAt:  createdAt,
				},
				{
					ID:         "dev-002",
					Name:       "Test Device 2",
					Key:        "base64encodedkey2==",
					Encryption: models.EncryptionAES128CTR,
					CreatedAt:  createdAt,
					Tags:       map[string]string{"env": "production"},
				},
			}

			// API returns {"devices": [...]}
			response := struct {
				Devices []models.Device `json:"devices"`
			}{Devices: devices}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		devices, err := client.ListDevices(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 2)

		assert.Equal(t, "dev-001", devices[0].ID)
		assert.Equal(t, "Test Device 1", devices[0].Name)
		assert.Equal(t, models.EncryptionAES256CTR, devices[0].Encryption)

		assert.Equal(t, "dev-002", devices[1].ID)
		assert.Equal(t, "production", devices[1].Tags["env"])
	})

	t.Run("success with empty list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"devices":[]}`))
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		devices, err := client.ListDevices(context.Background())

		require.NoError(t, err)
		assert.Empty(t, devices)
	})

	t.Run("with pagination", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++

			if requestCount == 1 {
				// First page - no continuation token in request header
				assert.Empty(t, r.Header.Get("Continuation-Token"))

				response := struct {
					Devices []models.Device `json:"devices"`
				}{
					Devices: []models.Device{
						{ID: "dev-001", Name: "Device 1"},
						{ID: "dev-002", Name: "Device 2"},
					},
				}
				// Return continuation token in response header
				w.Header().Set("Continuation-Token", "token123")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			} else {
				// Second page - continuation token should be in request header
				assert.Equal(t, "token123", r.Header.Get("Continuation-Token"))

				response := struct {
					Devices []models.Device `json:"devices"`
				}{
					Devices: []models.Device{
						{ID: "dev-003", Name: "Device 3"},
					},
				}
				// No continuation token means last page
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			}
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		devices, err := client.ListDevices(context.Background())

		require.NoError(t, err)
		require.Len(t, devices, 3)
		assert.Equal(t, "dev-001", devices[0].ID)
		assert.Equal(t, "dev-002", devices[1].ID)
		assert.Equal(t, "dev-003", devices[2].ID)
		assert.Equal(t, 2, requestCount)
	})
}

func TestClient_RegisterDevice(t *testing.T) {
	t.Run("success with defaults", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v2/org/test-org/devices", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)

			var req models.RegisterDeviceRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			assert.Equal(t, 1, req.NDevices)
			assert.Equal(t, models.EncryptionAES256CTR, req.Encryption)

			devices := []models.Device{
				{
					ID:         "new-dev-001",
					Key:        "newdevicekey==",
					Encryption: models.EncryptionAES256CTR,
					CreatedAt:  time.Now().UTC(),
				},
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(devices)
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		device, err := client.RegisterDevice(context.Background(), models.RegisterDeviceRequest{})

		require.NoError(t, err)
		assert.Equal(t, "new-dev-001", device.ID)
		assert.Equal(t, "newdevicekey==", device.Key)
	})

	t.Run("success with custom encryption", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req models.RegisterDeviceRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			assert.Equal(t, models.EncryptionAES128CTR, req.Encryption)

			devices := []models.Device{
				{
					ID:         "new-dev-002",
					Key:        "128bitkey==",
					Encryption: models.EncryptionAES128CTR,
				},
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(devices)
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		device, err := client.RegisterDevice(context.Background(), models.RegisterDeviceRequest{
			Encryption: models.EncryptionAES128CTR,
		})

		require.NoError(t, err)
		assert.Equal(t, models.EncryptionAES128CTR, device.Encryption)
	})
}

func TestClient_UpdateDevice(t *testing.T) {
	t.Run("update name", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/org/test-org/devices/dev-001", r.URL.Path)
			assert.Equal(t, http.MethodPatch, r.Method)

			var req models.UpdateDeviceRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			require.NotNil(t, req.SetName)
			assert.Equal(t, "New Device Name", *req.SetName)

			device := models.Device{
				ID:   "dev-001",
				Name: "New Device Name",
				Key:  "existingkey==",
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(device)
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		name := "New Device Name"
		device, err := client.UpdateDevice(context.Background(), "dev-001", models.UpdateDeviceRequest{
			SetName: &name,
		})

		require.NoError(t, err)
		assert.Equal(t, "New Device Name", device.Name)
	})

	t.Run("update tags", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req models.UpdateDeviceRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			require.NotNil(t, req.SetTags)
			assert.Equal(t, "production", (*req.SetTags)["env"])

			device := models.Device{
				ID:   "dev-001",
				Tags: map[string]string{"env": "production"},
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(device)
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		tags := map[string]string{"env": "production"}
		device, err := client.UpdateDevice(context.Background(), "dev-001", models.UpdateDeviceRequest{
			SetTags: &tags,
		})

		require.NoError(t, err)
		assert.Equal(t, "production", device.Tags["env"])
	})
}

func TestClient_SetDeviceName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.UpdateDeviceRequest
		json.NewDecoder(r.Body).Decode(&req)

		require.NotNil(t, req.SetName)
		assert.Equal(t, "My Device", *req.SetName)

		device := models.Device{ID: "dev-001", Name: "My Device"}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(device)
	}))
	defer server.Close()

	client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
	device, err := client.SetDeviceName(context.Background(), "dev-001", "My Device")

	require.NoError(t, err)
	assert.Equal(t, "My Device", device.Name)
}

func TestClient_SetDeviceTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.UpdateDeviceRequest
		json.NewDecoder(r.Body).Decode(&req)

		require.NotNil(t, req.SetTags)
		assert.Equal(t, "test", (*req.SetTags)["env"])

		device := models.Device{ID: "dev-001", Tags: map[string]string{"env": "test"}}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(device)
	}))
	defer server.Close()

	client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
	device, err := client.SetDeviceTags(context.Background(), "dev-001", map[string]string{"env": "test"})

	require.NoError(t, err)
	assert.Equal(t, "test", device.Tags["env"])
}
