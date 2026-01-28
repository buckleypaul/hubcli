package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_CheckCredentials(t *testing.T) {
	t.Run("valid credentials", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CheckCredentials now uses GetOrganization internally
			assert.Equal(t, "/org/test-org", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"org_id": "test-org", "name": "Test Organization"}`))
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		err := client.CheckCredentials(context.Background())

		require.NoError(t, err)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message": "invalid credentials"}`))
		}))
		defer server.Close()

		client := NewClient("test-org", "bad-token", WithBaseURL(server.URL))
		err := client.CheckCredentials(context.Background())

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})
}

func TestClient_GetOrganization(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/org/test-org", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"org_id": "test-org", "name": "Test Organization"}`))
		}))
		defer server.Close()

		client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
		org, err := client.GetOrganization(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "test-org", org.ID)
		assert.Equal(t, "Test Organization", org.Name)
	})

	t.Run("not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "organization not found"}`))
		}))
		defer server.Close()

		client := NewClient("nonexistent", "test-token", WithBaseURL(server.URL))
		_, err := client.GetOrganization(context.Background())

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}
