package screens

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewDevicesModel(t *testing.T) {
	m := NewDevicesModel(nil)

	assert.Equal(t, DevicesStateLoading, m.state)
	assert.Nil(t, m.client)
	assert.Empty(t, m.devices)
}

func TestDevicesModel_Init(t *testing.T) {
	m := NewDevicesModel(nil)
	cmd := m.Init()

	// Init should return commands for spinner tick and loading
	assert.NotNil(t, cmd)
}

func TestDevicesModel_WindowSizeMsg(t *testing.T) {
	m := NewDevicesModel(nil)

	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

func TestDevicesModel_DevicesLoadedMsg(t *testing.T) {
	m := NewDevicesModel(nil)
	m.state = DevicesStateLoading

	devices := []models.Device{
		{
			ID:         "device-1",
			Name:       "Test Device",
			Encryption: models.EncryptionAES256CTR,
			CreatedAt:  time.Now(),
		},
	}

	m, _ = m.Update(DevicesLoadedMsg{Devices: devices})

	assert.Equal(t, DevicesStateReady, m.state)
	assert.Len(t, m.devices, 1)
	assert.Equal(t, "device-1", m.devices[0].ID)
}

func TestDevicesModel_DevicesErrorMsg(t *testing.T) {
	m := NewDevicesModel(nil)
	m.state = DevicesStateLoading

	m, _ = m.Update(DevicesErrorMsg{Err: assert.AnError})

	assert.Equal(t, DevicesStateError, m.state)
	assert.Error(t, m.err)
}

func TestDevicesModel_BackNavigation(t *testing.T) {
	m := NewDevicesModel(nil)
	m.state = DevicesStateReady

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Should return a command that sends NavigateMsg
	assert.NotNil(t, cmd)
	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	assert.True(t, ok)
	assert.Equal(t, "home", navMsg.Screen)
}

func TestDevicesModel_QuitKey(t *testing.T) {
	m := NewDevicesModel(nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Should return tea.Quit
	assert.NotNil(t, cmd)
}

func TestDevicesModel_RefreshKey(t *testing.T) {
	m := NewDevicesModel(nil)
	m.state = DevicesStateReady

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	assert.Equal(t, DevicesStateLoading, m.state)
	assert.NotNil(t, cmd)
}

func TestDevicesModel_RefreshFromError(t *testing.T) {
	m := NewDevicesModel(nil)
	m.state = DevicesStateError

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	assert.Equal(t, DevicesStateLoading, m.state)
	assert.NotNil(t, cmd)
}

func TestDevicesModel_RegisterNewDevice(t *testing.T) {
	m := NewDevicesModel(nil)
	m.state = DevicesStateReady

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	assert.Equal(t, DevicesStateRegistering, m.state)
	assert.NotNil(t, cmd)
}

func TestDevicesModel_DeviceRegisteredMsg(t *testing.T) {
	m := NewDevicesModel(nil)
	m.state = DevicesStateRegistering

	device := &models.Device{ID: "new-device"}
	m, cmd := m.Update(DeviceRegisteredMsg{Device: device})

	assert.Equal(t, DevicesStateLoading, m.state)
	assert.NotNil(t, cmd) // Should reload devices
}

func TestDevicesModel_SelectDevice(t *testing.T) {
	m := NewDevicesModel(nil)
	m.state = DevicesStateReady
	m.devices = []models.Device{
		{ID: "device-1", Name: "Test Device"},
	}
	m.updateTable()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should return a command that navigates to packets with device ID
	assert.NotNil(t, cmd)
	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	assert.True(t, ok)
	assert.Equal(t, "packets", navMsg.Screen)
}

func TestDevicesModel_SelectedDevice(t *testing.T) {
	m := NewDevicesModel(nil)
	m.state = DevicesStateReady
	m.devices = []models.Device{
		{ID: "device-1", Name: "Test Device 1"},
		{ID: "device-2", Name: "Test Device 2"},
	}
	m.updateTable()

	// No selection in wrong state
	m.state = DevicesStateLoading
	assert.Nil(t, m.SelectedDevice())

	// No selection with empty devices
	m.state = DevicesStateReady
	m.devices = nil
	assert.Nil(t, m.SelectedDevice())
}

func TestDevicesModel_View(t *testing.T) {
	m := NewDevicesModel(nil)
	m.width = 80
	m.height = 24
	m.state = DevicesStateReady

	view := m.View()

	assert.Contains(t, view, "Devices")
	assert.Contains(t, view, "navigate")
	assert.Contains(t, view, "refresh")
}

func TestDevicesModel_ViewLoading(t *testing.T) {
	m := NewDevicesModel(nil)
	m.width = 80
	m.height = 24
	m.state = DevicesStateLoading

	view := m.View()

	assert.Contains(t, view, "Loading")
}

func TestDevicesModel_ViewError(t *testing.T) {
	m := NewDevicesModel(nil)
	m.width = 80
	m.height = 24
	m.state = DevicesStateError
	m.err = assert.AnError

	view := m.View()

	assert.Contains(t, view, "Error")
	assert.Contains(t, view, "retry")
}

func TestDevicesModel_ViewRegistering(t *testing.T) {
	m := NewDevicesModel(nil)
	m.width = 80
	m.height = 24
	m.state = DevicesStateRegistering

	view := m.View()

	assert.Contains(t, view, "Registering")
}

func TestDevicesModel_ViewEmpty(t *testing.T) {
	m := NewDevicesModel(nil)
	m.width = 80
	m.height = 24
	m.state = DevicesStateReady
	m.devices = []models.Device{}

	view := m.View()

	assert.Contains(t, view, "No devices found")
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 3, "hi"},
		{"hello", 5, "hello"},
		{"ab", 2, "ab"},
		{"abc", 3, "abc"},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		assert.Equal(t, tt.expected, result, "truncate(%q, %d)", tt.input, tt.maxLen)
	}
}
