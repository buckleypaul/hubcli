package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hubblenetwork/hubcli/internal/models"
)

// CheckCredentials validates the API credentials.
// Returns nil if credentials are valid.
func (c *Client) CheckCredentials(ctx context.Context) error {
	// Validate credentials by attempting to fetch the organization
	_, err := c.GetOrganization(ctx)
	return err
}

// GetOrganization retrieves organization metadata.
func (c *Client) GetOrganization(ctx context.Context) (*models.Organization, error) {
	path := fmt.Sprintf("/org/%s", c.orgID)
	body, _, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}

	var org models.Organization
	if err := json.Unmarshal(body, &org); err != nil {
		return nil, fmt.Errorf("failed to parse organization response: %w", err)
	}

	return &org, nil
}
