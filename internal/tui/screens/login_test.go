package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewLoginModel(t *testing.T) {
	m := NewLoginModel()

	assert.Equal(t, LoginStateInput, m.state)
	assert.Equal(t, 0, m.focusIndex)
	assert.Empty(t, m.orgIDInput.Value())
	assert.Empty(t, m.tokenInput.Value())
}

func TestLoginModel_Init(t *testing.T) {
	m := NewLoginModel()
	cmd := m.Init()

	// Init should return a command for text input blinking
	assert.NotNil(t, cmd)
}

func TestLoginModel_TabNavigation(t *testing.T) {
	m := NewLoginModel()

	// Initial focus should be on org ID (index 0)
	assert.Equal(t, 0, m.focusIndex)

	// Tab to token field
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 1, m.focusIndex)

	// Tab to submit button
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 2, m.focusIndex)

	// Tab wraps to org ID
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 0, m.focusIndex)
}

func TestLoginModel_ShiftTabNavigation(t *testing.T) {
	m := NewLoginModel()

	// Shift+Tab from first field wraps to submit button
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	assert.Equal(t, 2, m.focusIndex)

	// Shift+Tab to token field
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	assert.Equal(t, 1, m.focusIndex)

	// Shift+Tab to org ID field
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	assert.Equal(t, 0, m.focusIndex)
}

func TestLoginModel_CanSubmit(t *testing.T) {
	m := NewLoginModel()

	// Empty fields - cannot submit
	assert.False(t, m.canSubmit())

	// Set org ID only
	m.orgIDInput.SetValue("test-org")
	assert.False(t, m.canSubmit())

	// Set token too
	m.tokenInput.SetValue("test-token")
	assert.True(t, m.canSubmit())

	// Whitespace only doesn't count
	m.orgIDInput.SetValue("   ")
	assert.False(t, m.canSubmit())
}

func TestLoginModel_GetCredentials(t *testing.T) {
	m := NewLoginModel()
	m.orgIDInput.SetValue("  my-org  ")
	m.tokenInput.SetValue("  my-token  ")

	creds := m.GetCredentials()

	// Should trim whitespace
	assert.Equal(t, "my-org", creds.OrgID)
	assert.Equal(t, "my-token", creds.Token)
}

func TestLoginModel_WindowSizeMsg(t *testing.T) {
	m := NewLoginModel()

	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

func TestLoginModel_IsSuccess(t *testing.T) {
	m := NewLoginModel()

	assert.False(t, m.IsSuccess())

	m.state = LoginStateSuccess
	assert.True(t, m.IsSuccess())
}

func TestLoginModel_GetOrgName(t *testing.T) {
	m := NewLoginModel()

	assert.Empty(t, m.GetOrgName())

	m.orgName = "Test Org"
	assert.Equal(t, "Test Org", m.GetOrgName())
}

func TestLoginModel_LoginSuccessMsg(t *testing.T) {
	m := NewLoginModel()
	m.state = LoginStateValidating

	msg := LoginSuccessMsg{
		OrgName: "My Organization",
	}

	m, _ = m.Update(msg)

	assert.Equal(t, LoginStateSuccess, m.state)
	assert.Equal(t, "My Organization", m.orgName)
}

func TestLoginModel_LoginErrorMsg(t *testing.T) {
	m := NewLoginModel()
	m.state = LoginStateValidating

	msg := LoginErrorMsg{
		Err: assert.AnError,
	}

	m, _ = m.Update(msg)

	assert.Equal(t, LoginStateError, m.state)
	assert.Error(t, m.err)
}

func TestLoginModel_View(t *testing.T) {
	m := NewLoginModel()
	m.width = 80
	m.height = 24

	view := m.View()

	// Should contain key elements
	assert.Contains(t, view, "Hubble")
	assert.Contains(t, view, "Organization ID")
	assert.Contains(t, view, "API Token")
	assert.Contains(t, view, "Login")
}

func TestLoginModel_NoKeysDuringValidation(t *testing.T) {
	m := NewLoginModel()
	m.state = LoginStateValidating
	originalFocus := m.focusIndex

	// Tab should not change focus during validation
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, originalFocus, m.focusIndex)
}
