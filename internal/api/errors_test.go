package api

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name:     "with message",
			err:      &APIError{StatusCode: 401, Message: "invalid token"},
			expected: "API error 401: invalid token",
		},
		{
			name:     "without message",
			err:      &APIError{StatusCode: 500},
			expected: "API error 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestAPIError_Is(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		target     error
		expected   bool
	}{
		{"401 matches ErrInvalidCredentials", 401, ErrInvalidCredentials, true},
		{"404 matches ErrNotFound", 404, ErrNotFound, true},
		{"429 matches ErrRateLimited", 429, ErrRateLimited, true},
		{"400 matches ErrBadRequest", 400, ErrBadRequest, true},
		{"500 matches ErrServerError", 500, ErrServerError, true},
		{"502 matches ErrServerError", 502, ErrServerError, true},
		{"503 matches ErrServerError", 503, ErrServerError, true},
		{"401 does not match ErrNotFound", 401, ErrNotFound, false},
		{"200 does not match any", 200, ErrServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode}
			assert.Equal(t, tt.expected, errors.Is(err, tt.target))
		})
	}
}

func TestNewAPIError(t *testing.T) {
	err := NewAPIError(404, "not found")

	assert.Equal(t, 404, err.StatusCode)
	assert.Equal(t, "not found", err.Message)
}
