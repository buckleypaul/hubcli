package models

// Organization represents Hubble organization metadata.
type Organization struct {
	ID   string `json:"org_id"`
	Name string `json:"name"`
}

// Credentials holds authentication data for the Hubble API.
type Credentials struct {
	OrgID string
	Token string
}

// IsValid returns true if both OrgID and Token are non-empty.
func (c Credentials) IsValid() bool {
	return c.OrgID != "" && c.Token != ""
}

// Environment represents the API environment.
type Environment string

const (
	EnvProduction  Environment = "production"
	EnvStaging     Environment = "staging"
	EnvDevelopment Environment = "development"
)

// BaseURL returns the API base URL for the environment.
func (e Environment) BaseURL() string {
	switch e {
	case EnvStaging:
		return "https://api.staging.hubblenetwork.com/api"
	case EnvDevelopment:
		return "http://localhost:8080/api"
	default:
		return "https://api.hubblenetwork.com/api"
	}
}
