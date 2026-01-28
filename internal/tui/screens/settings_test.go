package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewSettingsModel(t *testing.T) {
	m := NewSettingsModel()

	assert.Equal(t, SettingsStateReady, m.state)
	assert.NotNil(t, m.store)
}

func TestSettingsModel_Init(t *testing.T) {
	m := NewSettingsModel()
	cmd := m.Init()

	// Init should return nil for settings screen
	assert.Nil(t, cmd)
}

func TestSettingsModel_WindowSizeMsg(t *testing.T) {
	m := NewSettingsModel()

	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

func TestSettingsModel_BackNavigation(t *testing.T) {
	m := NewSettingsModel()
	m.state = SettingsStateReady

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	assert.NotNil(t, cmd)
	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	assert.True(t, ok)
	assert.Equal(t, "home", navMsg.Screen)
}

func TestSettingsModel_QuitKey(t *testing.T) {
	m := NewSettingsModel()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	assert.NotNil(t, cmd)
}

func TestSettingsModel_ClearKey_NoKeychain(t *testing.T) {
	m := NewSettingsModel()
	m.state = SettingsStateReady
	m.hasKeychain = false

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	// Should not change state when no keychain
	assert.Equal(t, SettingsStateReady, m.state)
	assert.Nil(t, cmd)
}

func TestSettingsModel_ClearKey_WithKeychain(t *testing.T) {
	m := NewSettingsModel()
	m.state = SettingsStateReady
	m.hasKeychain = true

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	assert.Equal(t, SettingsStateConfirmClear, m.state)
}

func TestSettingsModel_ConfirmClear_Confirm(t *testing.T) {
	m := NewSettingsModel()
	m.state = SettingsStateConfirmClear

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	assert.Equal(t, SettingsStateClearing, m.state)
	assert.NotNil(t, cmd)
}

func TestSettingsModel_ConfirmClear_Cancel(t *testing.T) {
	m := NewSettingsModel()
	m.state = SettingsStateConfirmClear

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	assert.Equal(t, SettingsStateReady, m.state)
}

func TestSettingsModel_CredentialsClearedMsg_Success(t *testing.T) {
	m := NewSettingsModel()
	m.state = SettingsStateClearing

	m, _ = m.Update(CredentialsClearedMsg{Error: nil})

	assert.Equal(t, SettingsStateSuccess, m.state)
}

func TestSettingsModel_CredentialsClearedMsg_Error(t *testing.T) {
	m := NewSettingsModel()
	m.state = SettingsStateClearing

	m, _ = m.Update(CredentialsClearedMsg{Error: assert.AnError})

	assert.Equal(t, SettingsStateError, m.state)
	assert.Error(t, m.err)
}

func TestSettingsModel_AnyKeyFromSuccess(t *testing.T) {
	m := NewSettingsModel()
	m.state = SettingsStateSuccess

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	assert.Equal(t, SettingsStateReady, m.state)
}

func TestSettingsModel_AnyKeyFromError(t *testing.T) {
	m := NewSettingsModel()
	m.state = SettingsStateError

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	assert.Equal(t, SettingsStateReady, m.state)
}

func TestSettingsModel_View(t *testing.T) {
	m := NewSettingsModel()
	m.width = 80
	m.height = 24
	m.state = SettingsStateReady

	view := m.View()

	assert.Contains(t, view, "Settings")
	assert.Contains(t, view, "Credential Status")
	assert.Contains(t, view, "Environment Variables")
	assert.Contains(t, view, "HUBBLE_ORG_ID")
	assert.Contains(t, view, "HUBBLE_API_TOKEN")
}

func TestSettingsModel_ViewConfirmClear(t *testing.T) {
	m := NewSettingsModel()
	m.width = 80
	m.height = 24
	m.state = SettingsStateConfirmClear

	view := m.View()

	assert.Contains(t, view, "Clear stored credentials")
	assert.Contains(t, view, "confirm")
	assert.Contains(t, view, "cancel")
}

func TestSettingsModel_ViewClearing(t *testing.T) {
	m := NewSettingsModel()
	m.width = 80
	m.height = 24
	m.state = SettingsStateClearing

	view := m.View()

	assert.Contains(t, view, "Clearing credentials")
}

func TestSettingsModel_ViewSuccess(t *testing.T) {
	m := NewSettingsModel()
	m.width = 80
	m.height = 24
	m.state = SettingsStateSuccess

	view := m.View()

	assert.Contains(t, view, "cleared successfully")
}

func TestSettingsModel_ViewError(t *testing.T) {
	m := NewSettingsModel()
	m.width = 80
	m.height = 24
	m.state = SettingsStateError
	m.err = assert.AnError

	view := m.View()

	assert.Contains(t, view, "Error")
}

func TestSettingsModel_ViewWithKeychain(t *testing.T) {
	m := NewSettingsModel()
	m.width = 80
	m.height = 24
	m.state = SettingsStateReady
	m.hasKeychain = true
	m.keychainOrgID = "test-org-123"

	view := m.View()

	assert.Contains(t, view, "Stored")
	assert.Contains(t, view, "clear keychain")
}

func TestSettingsModel_ViewWithEnvVars(t *testing.T) {
	m := NewSettingsModel()
	m.width = 80
	m.height = 24
	m.state = SettingsStateReady
	m.hasEnvVars = true
	m.envOrgID = "env-org-456"

	view := m.View()

	assert.Contains(t, view, "Environment variables")
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abc", "****"},
		{"abcdef", "****"},
		{"abcdefg", "ab***fg"},
		{"test-org-id-123", "te***********23"},
	}

	for _, tt := range tests {
		result := maskString(tt.input)
		assert.Equal(t, tt.expected, result, "maskString(%q)", tt.input)
	}
}
