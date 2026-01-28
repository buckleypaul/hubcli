package auth

import (
	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/zalando/go-keyring"
)

const (
	// KeychainService is the service name used in the macOS Keychain.
	KeychainService = "hubcli"
	// Keychain item names
	keychainOrgID = "org_id"
	keychainToken = "api_token"
)

// KeychainStore implements CredentialStore using the macOS Keychain.
type KeychainStore struct{}

// NewKeychainStore creates a new KeychainStore.
func NewKeychainStore() *KeychainStore {
	return &KeychainStore{}
}

// Get retrieves credentials from the keychain.
func (s *KeychainStore) Get() (*models.Credentials, error) {
	orgID, err := keyring.Get(KeychainService, keychainOrgID)
	if err != nil {
		return nil, err
	}

	token, err := keyring.Get(KeychainService, keychainToken)
	if err != nil {
		return nil, err
	}

	return &models.Credentials{
		OrgID: orgID,
		Token: token,
	}, nil
}

// Save stores credentials in the keychain.
func (s *KeychainStore) Save(creds *models.Credentials) error {
	if err := keyring.Set(KeychainService, keychainOrgID, creds.OrgID); err != nil {
		return err
	}

	if err := keyring.Set(KeychainService, keychainToken, creds.Token); err != nil {
		// Try to clean up the org ID if token save fails
		_ = keyring.Delete(KeychainService, keychainOrgID)
		return err
	}

	return nil
}

// Delete removes credentials from the keychain.
func (s *KeychainStore) Delete() error {
	// Delete both, ignoring errors if they don't exist
	_ = keyring.Delete(KeychainService, keychainOrgID)
	_ = keyring.Delete(KeychainService, keychainToken)
	return nil
}

// Exists returns true if credentials are stored in the keychain.
func (s *KeychainStore) Exists() bool {
	_, err := keyring.Get(KeychainService, keychainOrgID)
	if err != nil {
		return false
	}

	_, err = keyring.Get(KeychainService, keychainToken)
	return err == nil
}
