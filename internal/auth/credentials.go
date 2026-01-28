package auth

import (
	"errors"
	"os"

	"github.com/hubblenetwork/hubcli/internal/models"
)

const (
	// Environment variable names
	EnvOrgID = "HUBBLE_ORG_ID"
	EnvToken = "HUBBLE_API_TOKEN"
)

// Common errors
var (
	ErrNoCredentials = errors.New("no credentials found")
)

// CredentialStore defines the interface for credential persistence.
type CredentialStore interface {
	// Get retrieves stored credentials.
	Get() (*models.Credentials, error)
	// Save persists credentials.
	Save(creds *models.Credentials) error
	// Delete removes stored credentials.
	Delete() error
	// Exists returns true if credentials are stored.
	Exists() bool
}

// GetCredentials retrieves credentials from all available sources.
// Priority: environment variables > keychain
func GetCredentials() (*models.Credentials, error) {
	// First, try environment variables
	envCreds := GetCredentialsFromEnv()
	if envCreds.IsValid() {
		return envCreds, nil
	}

	// Then try keychain
	keychainStore := NewKeychainStore()
	if keychainStore.Exists() {
		return keychainStore.Get()
	}

	return nil, ErrNoCredentials
}

// GetCredentialsFromEnv reads credentials from environment variables.
func GetCredentialsFromEnv() *models.Credentials {
	return &models.Credentials{
		OrgID: os.Getenv(EnvOrgID),
		Token: os.Getenv(EnvToken),
	}
}

// SaveCredentials saves credentials to the keychain.
func SaveCredentials(creds *models.Credentials) error {
	store := NewKeychainStore()
	return store.Save(creds)
}

// DeleteCredentials removes credentials from the keychain.
func DeleteCredentials() error {
	store := NewKeychainStore()
	return store.Delete()
}

// HasCredentials returns true if credentials exist in env or keychain.
func HasCredentials() bool {
	creds := GetCredentialsFromEnv()
	if creds.IsValid() {
		return true
	}

	store := NewKeychainStore()
	return store.Exists()
}
