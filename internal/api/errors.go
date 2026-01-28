package api

import (
	"errors"
	"fmt"
)

// Common API errors.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotFound           = errors.New("resource not found")
	ErrRateLimited        = errors.New("rate limited")
	ErrServerError        = errors.New("server error")
	ErrBadRequest         = errors.New("bad request")
)

// APIError represents an error response from the Hubble API.
type APIError struct {
	StatusCode int
	Message    string
	Details    map[string]interface{}
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error %d", e.StatusCode)
}

// Is implements error matching for APIError.
func (e *APIError) Is(target error) bool {
	switch e.StatusCode {
	case 401:
		return errors.Is(target, ErrInvalidCredentials)
	case 404:
		return errors.Is(target, ErrNotFound)
	case 429:
		return errors.Is(target, ErrRateLimited)
	case 400:
		return errors.Is(target, ErrBadRequest)
	}
	if e.StatusCode >= 500 {
		return errors.Is(target, ErrServerError)
	}
	return false
}

// NewAPIError creates an APIError from an HTTP status code.
func NewAPIError(statusCode int, message string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    message,
	}
}
