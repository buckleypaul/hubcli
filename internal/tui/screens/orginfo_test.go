package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewOrgInfoModel(t *testing.T) {
	m := NewOrgInfoModel(nil)

	assert.Equal(t, OrgInfoStateLoading, m.state)
	assert.Nil(t, m.client)
	assert.Nil(t, m.org)
	assert.Nil(t, m.credsValid)
}

func TestOrgInfoModel_Init(t *testing.T) {
	m := NewOrgInfoModel(nil)
	cmd := m.Init()

	// Init should return commands for spinner tick and loading
	assert.NotNil(t, cmd)
}

func TestOrgInfoModel_WindowSizeMsg(t *testing.T) {
	m := NewOrgInfoModel(nil)

	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

func TestOrgInfoModel_OrgInfoLoadedMsg(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.state = OrgInfoStateLoading

	org := &models.Organization{
		ID:   "org-123",
		Name: "Test Organization",
	}

	m, _ = m.Update(OrgInfoLoadedMsg{Org: org, DeviceCount: 5})

	assert.Equal(t, OrgInfoStateReady, m.state)
	assert.NotNil(t, m.org)
	assert.Equal(t, "org-123", m.org.ID)
	assert.Equal(t, "Test Organization", m.org.Name)
	assert.Equal(t, 5, m.deviceCount)
	// Credentials should be automatically marked as valid
	assert.NotNil(t, m.credsValid)
	assert.True(t, *m.credsValid)
}

func TestOrgInfoModel_OrgInfoErrorMsg(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.state = OrgInfoStateLoading

	m, _ = m.Update(OrgInfoErrorMsg{Err: assert.AnError})

	assert.Equal(t, OrgInfoStateError, m.state)
	assert.Error(t, m.err)
	// Credentials should be automatically marked as invalid on error
	assert.NotNil(t, m.credsValid)
	assert.False(t, *m.credsValid)
}

func TestOrgInfoModel_CredsValidMsg_Valid(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.state = OrgInfoStateCheckingCreds

	m, _ = m.Update(CredsValidMsg{Valid: true})

	assert.Equal(t, OrgInfoStateReady, m.state)
	assert.NotNil(t, m.credsValid)
	assert.True(t, *m.credsValid)
}

func TestOrgInfoModel_CredsValidMsg_Invalid(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.state = OrgInfoStateCheckingCreds

	m, _ = m.Update(CredsValidMsg{Valid: false})

	assert.Equal(t, OrgInfoStateReady, m.state)
	assert.NotNil(t, m.credsValid)
	assert.False(t, *m.credsValid)
}

func TestOrgInfoModel_CredsValidMsg_Error(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.state = OrgInfoStateCheckingCreds

	m, _ = m.Update(CredsValidMsg{Valid: false, Err: assert.AnError})

	assert.Equal(t, OrgInfoStateReady, m.state)
	assert.NotNil(t, m.credsValid)
	assert.False(t, *m.credsValid)
}

func TestOrgInfoModel_BackNavigation(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.state = OrgInfoStateReady

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Should return a command that sends NavigateMsg
	assert.NotNil(t, cmd)
	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	assert.True(t, ok)
	assert.Equal(t, "home", navMsg.Screen)
}

func TestOrgInfoModel_QuitKey(t *testing.T) {
	m := NewOrgInfoModel(nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Should return tea.Quit
	assert.NotNil(t, cmd)
}

func TestOrgInfoModel_RefreshKey(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.state = OrgInfoStateReady

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	assert.Equal(t, OrgInfoStateLoading, m.state)
	assert.Nil(t, m.credsValid) // Should reset creds validation
	assert.NotNil(t, cmd)
}

func TestOrgInfoModel_RefreshFromError(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.state = OrgInfoStateError

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	assert.Equal(t, OrgInfoStateLoading, m.state)
	assert.NotNil(t, cmd)
}

func TestOrgInfoModel_VKey_NoOp(t *testing.T) {
	// 'v' key should no longer trigger validation (it's automatic now)
	m := NewOrgInfoModel(nil)
	m.state = OrgInfoStateReady

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})

	// Should not change state
	assert.Equal(t, OrgInfoStateReady, m.state)
	assert.Nil(t, cmd)
}

func TestOrgInfoModel_View(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.width = 80
	m.height = 24
	m.state = OrgInfoStateReady
	m.org = &models.Organization{
		ID:   "org-123",
		Name: "Test Organization",
	}

	view := m.View()

	assert.Contains(t, view, "Organization")
	assert.Contains(t, view, "org-123")
	assert.Contains(t, view, "Test Organization")
	assert.Contains(t, view, "refresh")
}

func TestOrgInfoModel_ViewLoading(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.width = 80
	m.height = 24
	m.state = OrgInfoStateLoading

	view := m.View()

	assert.Contains(t, view, "Loading")
}

func TestOrgInfoModel_ViewCheckingCreds(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.width = 80
	m.height = 24
	m.state = OrgInfoStateCheckingCreds

	view := m.View()

	assert.Contains(t, view, "Validating")
}

func TestOrgInfoModel_ViewError(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.width = 80
	m.height = 24
	m.state = OrgInfoStateError
	m.err = assert.AnError

	view := m.View()

	assert.Contains(t, view, "Error")
	assert.Contains(t, view, "retry")
}

func TestOrgInfoModel_ViewCredsValid(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.width = 80
	m.height = 24
	m.state = OrgInfoStateReady
	m.org = &models.Organization{ID: "org-123"}
	valid := true
	m.credsValid = &valid

	view := m.View()

	assert.Contains(t, view, "Valid")
}

func TestOrgInfoModel_ViewCredsInvalid(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.width = 80
	m.height = 24
	m.state = OrgInfoStateReady
	m.org = &models.Organization{ID: "org-123"}
	valid := false
	m.credsValid = &valid

	view := m.View()

	assert.Contains(t, view, "Invalid")
}

func TestOrgInfoModel_ViewNoOrgName(t *testing.T) {
	m := NewOrgInfoModel(nil)
	m.width = 80
	m.height = 24
	m.state = OrgInfoStateReady
	m.org = &models.Organization{ID: "org-123", Name: ""}

	view := m.View()

	assert.Contains(t, view, "Not set")
}
