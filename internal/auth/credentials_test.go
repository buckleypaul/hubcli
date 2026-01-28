package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCredentialsFromEnv(t *testing.T) {
	t.Run("with env vars set", func(t *testing.T) {
		os.Setenv(EnvOrgID, "test-org")
		os.Setenv(EnvToken, "test-token")
		defer func() {
			os.Unsetenv(EnvOrgID)
			os.Unsetenv(EnvToken)
		}()

		creds := GetCredentialsFromEnv()

		assert.Equal(t, "test-org", creds.OrgID)
		assert.Equal(t, "test-token", creds.Token)
		assert.True(t, creds.IsValid())
	})

	t.Run("without env vars", func(t *testing.T) {
		os.Unsetenv(EnvOrgID)
		os.Unsetenv(EnvToken)

		creds := GetCredentialsFromEnv()

		assert.Empty(t, creds.OrgID)
		assert.Empty(t, creds.Token)
		assert.False(t, creds.IsValid())
	})

	t.Run("partial env vars", func(t *testing.T) {
		os.Setenv(EnvOrgID, "test-org")
		os.Unsetenv(EnvToken)
		defer os.Unsetenv(EnvOrgID)

		creds := GetCredentialsFromEnv()

		assert.Equal(t, "test-org", creds.OrgID)
		assert.Empty(t, creds.Token)
		assert.False(t, creds.IsValid())
	})
}

func TestHasCredentials_WithEnvVars(t *testing.T) {
	os.Setenv(EnvOrgID, "test-org")
	os.Setenv(EnvToken, "test-token")
	defer func() {
		os.Unsetenv(EnvOrgID)
		os.Unsetenv(EnvToken)
	}()

	assert.True(t, HasCredentials())
}

func TestHasCredentials_WithoutEnvVars(t *testing.T) {
	os.Unsetenv(EnvOrgID)
	os.Unsetenv(EnvToken)

	// This will check keychain, which may or may not have credentials
	// Just verify it doesn't panic
	_ = HasCredentials()
}
