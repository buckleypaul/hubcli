package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-org", "test-token")

	assert.Equal(t, "test-org", client.OrgID())
	assert.NotNil(t, client.httpClient)
}

func TestClient_RequestSetsHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, userAgent, r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
	_, _, err := client.get(context.Background(), "/test")

	require.NoError(t, err)
}

func TestClient_HandlesErrorResponses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    error
	}{
		{
			name:       "401 unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   `{"message": "invalid token"}`,
			wantErr:    ErrInvalidCredentials,
		},
		{
			name:       "404 not found",
			statusCode: http.StatusNotFound,
			response:   `{"message": "resource not found"}`,
			wantErr:    ErrNotFound,
		},
		{
			name:       "429 rate limited",
			statusCode: http.StatusTooManyRequests,
			response:   `{"message": "rate limit exceeded"}`,
			wantErr:    ErrRateLimited,
		},
		{
			name:       "400 bad request",
			statusCode: http.StatusBadRequest,
			response:   `{"message": "invalid request"}`,
			wantErr:    ErrBadRequest,
		},
		{
			name:       "500 server error",
			statusCode: http.StatusInternalServerError,
			response:   `{"message": "internal error"}`,
			wantErr:    ErrServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
			_, _, err := client.get(context.Background(), "/test")

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)

			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.statusCode, apiErr.StatusCode)
		})
	}
}

func TestClient_PostSendsBody(t *testing.T) {
	type testBody struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		var body testBody
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)

		assert.Equal(t, "test", body.Name)
		assert.Equal(t, 42, body.Value)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result": "ok"}`))
	}))
	defer server.Close()

	client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
	_, _, err := client.post(context.Background(), "/test", testBody{Name: "test", Value: 42})

	require.NoError(t, err)
}

func TestClient_PatchSendsBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient("test-org", "test-token", WithBaseURL(server.URL))
	_, _, err := client.patch(context.Background(), "/test", map[string]string{"key": "value"})

	require.NoError(t, err)
}

func TestWithHTTPClient(t *testing.T) {
	customClient := &http.Client{}
	client := NewClient("org", "token", WithHTTPClient(customClient))

	assert.Same(t, customClient, client.httpClient)
}
