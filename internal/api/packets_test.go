package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_RetrievePackets(t *testing.T) {
	t.Run("success with packets", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.True(t, strings.HasPrefix(r.URL.Path, "/org/test-org/packets"))

			// Verify start parameter is set
			startParam := r.URL.Query().Get("start")
			assert.NotEmpty(t, startParam)

			// API returns {"packets": [...]} wrapper
			response := map[string]interface{}{
				"packets": []map[string]interface{}{
					{
						"location": map[string]interface{}{
							"timestamp":           float64(time.Now().Unix()),
							"latitude":            37.7749,
							"longitude":           -122.4194,
							"altitude":            0,
							"horizontal_accuracy": 10,
							"vertical_accuracy":   10,
						},
						"device": map[string]interface{}{
							"id":              "dev-001",
							"name":            "Test Device",
							"tags":            map[string]string{},
							"payload":         "0102030405",
							"timestamp":       float64(time.Now().Unix()),
							"rssi":            -65,
							"sequence_number": 1,
							"counter":         100,
						},
						"network_type": "TERRESTRIAL",
					},
				},
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		packets, err := client.RetrievePackets(context.Background(), RetrievePacketsOptions{})

		require.NoError(t, err)
		require.Len(t, packets, 1)
		assert.Equal(t, "dev-001", packets[0].DeviceID())
		assert.Equal(t, "0102030405", packets[0].Payload())
	})

	t.Run("with device filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			deviceID := r.URL.Query().Get("device_id")
			assert.Equal(t, "dev-specific", deviceID)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"packets":[]}`))
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		deviceID := "dev-specific"
		_, err := client.RetrievePackets(context.Background(), RetrievePacketsOptions{
			DeviceID: &deviceID,
		})

		require.NoError(t, err)
	})

	t.Run("with custom days", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startParam := r.URL.Query().Get("start")
			startUnix, err := strconv.ParseInt(startParam, 10, 64)
			require.NoError(t, err)

			startTime := time.Unix(startUnix, 0)
			expectedStart := time.Now().UTC().AddDate(0, 0, -14)

			// Allow 1 minute tolerance
			assert.WithinDuration(t, expectedStart, startTime, time.Minute)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"packets":[]}`))
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		_, err := client.RetrievePackets(context.Background(), RetrievePacketsOptions{
			Days: 14,
		})

		require.NoError(t, err)
	})

	t.Run("with pagination", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++

			if requestCount == 1 {
				// First page - no continuation token in request header
				assert.Empty(t, r.Header.Get("Continuation-Token"))

				response := map[string]interface{}{
					"packets": []map[string]interface{}{
						{
							"location":     map[string]interface{}{"timestamp": float64(time.Now().Unix())},
							"device":       map[string]interface{}{"id": "dev-001", "payload": "page1", "timestamp": float64(time.Now().Unix())},
							"network_type": "TERRESTRIAL",
						},
					},
				}
				// Return continuation token in response header
				w.Header().Set("Continuation-Token", "token123")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			} else {
				// Second page - continuation token should be in request header
				assert.Equal(t, "token123", r.Header.Get("Continuation-Token"))

				response := map[string]interface{}{
					"packets": []map[string]interface{}{
						{
							"location":     map[string]interface{}{"timestamp": float64(time.Now().Unix())},
							"device":       map[string]interface{}{"id": "dev-002", "payload": "page2", "timestamp": float64(time.Now().Unix())},
							"network_type": "TERRESTRIAL",
						},
					},
				}
				// No continuation token means last page
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
			}
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		packets, err := client.RetrievePackets(context.Background(), RetrievePacketsOptions{})

		require.NoError(t, err)
		require.Len(t, packets, 2)
		assert.Equal(t, "dev-001", packets[0].DeviceID())
		assert.Equal(t, "dev-002", packets[1].DeviceID())
		assert.Equal(t, 2, requestCount)
	})
}

func TestClient_IngestPacket(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/org/test-org/packets", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)

			var req models.IngestPacketRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			require.Len(t, req.BLELocations, 1)
			assert.Equal(t, float64(37.7749), req.BLELocations[0].Location.Latitude)
			require.Len(t, req.BLELocations[0].Adverstments, 1)
			assert.Equal(t, -70, req.BLELocations[0].Adverstments[0].RSSI)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		err := client.IngestPacket(context.Background(), models.IngestPacketRequest{
			BLELocations: []models.BLELocation{
				{
					Location: models.LocationPayload{
						Latitude:  37.7749,
						Longitude: -122.4194,
						Timestamp: time.Now().Unix(),
					},
					Adverstments: []models.BLEAdvertisement{
						{
							Payload:   "dGVzdHBheWxvYWQ=",
							RSSI:      -70,
							Timestamp: time.Now().Unix(),
						},
					},
				},
			},
		})

		require.NoError(t, err)
	})
}

func TestClient_IngestEncryptedPackets(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req models.IngestPacketRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			require.Len(t, req.BLELocations, 2)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))

		packets := []models.EncryptedPacket{
			{
				Payload:   []byte{0x01, 0x02, 0x03},
				RSSI:      -65,
				Timestamp: time.Now().UTC(),
				Location:  models.NewFakeLocation(),
			},
			{
				Payload:   []byte{0x04, 0x05, 0x06},
				RSSI:      -72,
				Timestamp: time.Now().UTC(),
				Location:  models.NewFakeLocation(),
			},
		}

		err := client.IngestEncryptedPackets(context.Background(), packets)
		require.NoError(t, err)
	})

	t.Run("empty packets does nothing", func(t *testing.T) {
		serverCalled := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serverCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		err := client.IngestEncryptedPackets(context.Background(), []models.EncryptedPacket{})

		require.NoError(t, err)
		assert.False(t, serverCalled)
	})
}
