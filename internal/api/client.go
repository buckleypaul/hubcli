package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hubblenetwork/hubcli/internal/models"
)

const (
	defaultTimeout = 30 * time.Second
	userAgent      = "hubcli/1.0"
)

// Client is an HTTP client for the Hubble API.
type Client struct {
	baseURL    string
	orgID      string
	token      string
	httpClient *http.Client
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = c
	}
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) ClientOption {
	return func(client *Client) {
		client.baseURL = url
	}
}

// NewClient creates a new Hubble API client.
func NewClient(orgID, token string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: models.EnvProduction.BaseURL(),
		orgID:   orgID,
		token:   token,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// NewClientFromCredentials creates a client from a Credentials struct.
func NewClientFromCredentials(creds models.Credentials, opts ...ClientOption) *Client {
	return NewClient(creds.OrgID, creds.Token, opts...)
}

// request performs an HTTP request and returns the response body.
func (c *Client) request(ctx context.Context, method, path string, body interface{}) ([]byte, http.Header, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Message string                 `json:"message"`
			Error   string                 `json:"error"`
			Details map[string]interface{} `json:"details"`
		}
		_ = json.Unmarshal(respBody, &errResp)

		msg := errResp.Message
		if msg == "" {
			msg = errResp.Error
		}

		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Message:    msg,
			Details:    errResp.Details,
		}
		return nil, resp.Header, apiErr
	}

	return respBody, resp.Header, nil
}

// get performs a GET request.
func (c *Client) get(ctx context.Context, path string) ([]byte, http.Header, error) {
	return c.getWithContToken(ctx, path, "")
}

// getWithContToken performs a GET request with an optional continuation token header.
func (c *Client) getWithContToken(ctx context.Context, path string, contToken string) ([]byte, http.Header, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if contToken != "" {
		req.Header.Set("Continuation-Token", contToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Message string                 `json:"message"`
			Error   string                 `json:"error"`
			Details map[string]interface{} `json:"details"`
		}
		_ = json.Unmarshal(respBody, &errResp)

		msg := errResp.Message
		if msg == "" {
			msg = errResp.Error
		}

		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Message:    msg,
			Details:    errResp.Details,
		}
		return nil, resp.Header, apiErr
	}

	return respBody, resp.Header, nil
}

// post performs a POST request.
func (c *Client) post(ctx context.Context, path string, body interface{}) ([]byte, http.Header, error) {
	return c.request(ctx, http.MethodPost, path, body)
}

// patch performs a PATCH request.
func (c *Client) patch(ctx context.Context, path string, body interface{}) ([]byte, http.Header, error) {
	return c.request(ctx, http.MethodPatch, path, body)
}

// delete performs a DELETE request.
func (c *Client) delete(ctx context.Context, path string) ([]byte, http.Header, error) {
	return c.request(ctx, http.MethodDelete, path, nil)
}

// OrgID returns the configured organization ID.
func (c *Client) OrgID() string {
	return c.orgID
}
