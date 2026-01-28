package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewHomeModel(t *testing.T) {
	m := NewHomeModel("Test Org")

	assert.Equal(t, "Test Org", m.orgName)
	assert.Equal(t, 0, m.cursor)
	assert.Len(t, m.items, 5) // Devices, Packets, BLE Scan, Organization, Settings
}

func TestHomeModel_Init(t *testing.T) {
	m := NewHomeModel("")
	cmd := m.Init()

	// Init returns nil - no startup command needed
	assert.Nil(t, cmd)
}

func TestHomeModel_UpDownNavigation(t *testing.T) {
	m := NewHomeModel("")

	// Initial cursor at 0
	assert.Equal(t, 0, m.cursor)

	// Down moves to 1
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.cursor)

	// Down again
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, m.cursor)

	// Up goes back
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, m.cursor)
}

func TestHomeModel_NavigationWrapping(t *testing.T) {
	m := NewHomeModel("")
	lastIndex := len(m.items) - 1

	// Up from 0 wraps to last
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, lastIndex, m.cursor)

	// Down from last wraps to 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, m.cursor)
}

func TestHomeModel_VimKeysNavigation(t *testing.T) {
	m := NewHomeModel("")

	// j moves down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, m.cursor)

	// k moves up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.cursor)
}

func TestHomeModel_SelectItem(t *testing.T) {
	m := NewHomeModel("")

	// Select first item (Devices)
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.NotNil(t, cmd)

	// Execute the command
	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	assert.True(t, ok)
	assert.Equal(t, "devices", navMsg.Screen)
}

func TestHomeModel_SelectDifferentItems(t *testing.T) {
	m := NewHomeModel("")

	// Move to Packets (index 1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	msg := cmd()
	navMsg := msg.(NavigateMsg)
	assert.Equal(t, "packets", navMsg.Screen)

	// Move to BLE Scan (index 2)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	msg = cmd()
	navMsg = msg.(NavigateMsg)
	assert.Equal(t, "ble_scan", navMsg.Screen)
}

func TestHomeModel_SelectedItem(t *testing.T) {
	m := NewHomeModel("")

	// First item should be Devices
	item := m.SelectedItem()
	assert.Equal(t, "Devices", item.Title)
	assert.Equal(t, "devices", item.Screen)

	// Move to second item
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	item = m.SelectedItem()
	assert.Equal(t, "Packets", item.Title)
}

func TestHomeModel_WindowSizeMsg(t *testing.T) {
	m := NewHomeModel("")

	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestHomeModel_ToggleHelp(t *testing.T) {
	m := NewHomeModel("")

	assert.False(t, m.showHelp)

	// Press ? to toggle help
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.True(t, m.showHelp)

	// Press ? again to hide
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.False(t, m.showHelp)
}

func TestHomeModel_QuitKey(t *testing.T) {
	m := NewHomeModel("")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Should return quit command
	assert.NotNil(t, cmd)
	// The cmd is tea.Quit which is a function
}

func TestHomeModel_SetOrgName(t *testing.T) {
	m := NewHomeModel("")

	assert.Empty(t, m.orgName)

	m.SetOrgName("New Org Name")
	assert.Equal(t, "New Org Name", m.orgName)
}

func TestHomeModel_View(t *testing.T) {
	m := NewHomeModel("Test Organization")
	m.width = 80
	m.height = 24

	view := m.View()

	// Should contain key elements
	assert.Contains(t, view, "Hubble CLI")
	assert.Contains(t, view, "Test Organization")
	assert.Contains(t, view, "Devices")
	assert.Contains(t, view, "Packets")
	assert.Contains(t, view, "BLE Scan")
}

func TestNavigateMsg(t *testing.T) {
	msg := NavigateMsg{Screen: "devices"}
	assert.Equal(t, "devices", msg.Screen)
}
