package screens

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewPacketsModel(t *testing.T) {
	m := NewPacketsModel(nil, "")

	assert.Equal(t, PacketsStateLoading, m.state)
	assert.Nil(t, m.client)
	assert.Empty(t, m.packets)
	assert.Equal(t, 7, m.days) // Default to 7 days
	assert.Empty(t, m.deviceID)
}

func TestNewPacketsModel_WithDeviceFilter(t *testing.T) {
	m := NewPacketsModel(nil, "device-123")

	assert.Equal(t, "device-123", m.deviceID)
}

func TestPacketsModel_Init(t *testing.T) {
	m := NewPacketsModel(nil, "")
	cmd := m.Init()

	// Init should return commands for spinner tick and loading
	assert.NotNil(t, cmd)
}

func TestPacketsModel_WindowSizeMsg(t *testing.T) {
	m := NewPacketsModel(nil, "")

	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

func TestPacketsModel_PacketsLoadedMsg(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.state = PacketsStateLoading

	packets := []models.RetrievedPacket{
		{
			Location: models.RetrievedLocation{
				Timestamp: float64(time.Now().Unix()),
				Latitude:  37.7749,
				Longitude: -122.4194,
			},
			Device: models.RetrievedDevice{
				ID:        "device-1",
				Timestamp: float64(time.Now().Unix()),
				Payload:   "test payload",
			},
			NetworkType: "TERRESTRIAL",
		},
	}

	m, _ = m.Update(PacketsLoadedMsg{Packets: packets})

	assert.Equal(t, PacketsStateReady, m.state)
	assert.Len(t, m.packets, 1)
	assert.Equal(t, "device-1", m.packets[0].DeviceID())
}

func TestPacketsModel_PacketsErrorMsg(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.state = PacketsStateLoading

	m, _ = m.Update(PacketsErrorMsg{Err: assert.AnError})

	assert.Equal(t, PacketsStateError, m.state)
	assert.Error(t, m.err)
}

func TestPacketsModel_BackNavigation(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.state = PacketsStateReady

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	// Should return a command that sends NavigateMsg
	assert.NotNil(t, cmd)
	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	assert.True(t, ok)
	assert.Equal(t, "back", navMsg.Screen)
}

func TestPacketsModel_QuitKey(t *testing.T) {
	m := NewPacketsModel(nil, "")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Should return tea.Quit
	assert.NotNil(t, cmd)
}

func TestPacketsModel_RefreshKey(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.state = PacketsStateReady

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	assert.Equal(t, PacketsStateLoading, m.state)
	assert.NotNil(t, cmd)
}

func TestPacketsModel_RefreshFromError(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.state = PacketsStateError

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	assert.Equal(t, PacketsStateLoading, m.state)
	assert.NotNil(t, cmd)
}

func TestPacketsModel_ChangeDays1(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.state = PacketsStateReady

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})

	assert.Equal(t, 1, m.days)
	assert.Equal(t, PacketsStateLoading, m.state)
	assert.NotNil(t, cmd)
}

func TestPacketsModel_ChangeDays7(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.state = PacketsStateReady
	m.days = 1

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'7'}})

	assert.Equal(t, 7, m.days)
	assert.Equal(t, PacketsStateLoading, m.state)
	assert.NotNil(t, cmd)
}

func TestPacketsModel_ClearDeviceFilter(t *testing.T) {
	m := NewPacketsModel(nil, "device-123")
	m.state = PacketsStateReady

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	assert.Empty(t, m.deviceID)
	assert.Equal(t, PacketsStateLoading, m.state)
	assert.NotNil(t, cmd)
}

func TestPacketsModel_ClearDeviceFilter_NoOp(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.state = PacketsStateReady

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	// Should not change state if no filter
	assert.Equal(t, PacketsStateReady, m.state)
	assert.Nil(t, cmd)
}

func TestPacketsModel_SetDeviceFilter(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.SetDeviceFilter("device-456")

	assert.Equal(t, "device-456", m.deviceID)
}

func TestPacketsModel_View(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.width = 80
	m.height = 24
	m.state = PacketsStateReady

	view := m.View()

	assert.Contains(t, view, "Packets")
	assert.Contains(t, view, "7 day(s)")
	assert.Contains(t, view, "navigate")
	assert.Contains(t, view, "refresh")
}

func TestPacketsModel_ViewWithDeviceFilter(t *testing.T) {
	m := NewPacketsModel(nil, "device-123")
	m.width = 80
	m.height = 24
	m.state = PacketsStateReady

	view := m.View()

	assert.Contains(t, view, "device")
	assert.Contains(t, view, "clear filter")
}

func TestPacketsModel_ViewLoading(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.width = 80
	m.height = 24
	m.state = PacketsStateLoading

	view := m.View()

	assert.Contains(t, view, "Loading")
}

func TestPacketsModel_ViewError(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.width = 80
	m.height = 24
	m.state = PacketsStateError
	m.err = assert.AnError

	view := m.View()

	assert.Contains(t, view, "Error")
	assert.Contains(t, view, "retry")
}

func TestPacketsModel_ViewEmpty(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.width = 80
	m.height = 24
	m.state = PacketsStateReady
	m.packets = []models.RetrievedPacket{}

	view := m.View()

	assert.Contains(t, view, "No packets found")
}

func TestPacketsModel_LoadMore(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.state = PacketsStateReady
	m.hasMore = true
	m.continuationToken = "token-123"

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

	assert.True(t, m.loadingMore)
	assert.NotNil(t, cmd)
}

func TestPacketsModel_LoadMore_NoMore(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.state = PacketsStateReady
	m.hasMore = false

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

	assert.False(t, m.loadingMore)
	assert.Nil(t, cmd)
}

func TestPacketsModel_PacketsLoadedMsg_WithPagination(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.state = PacketsStateLoading
	m.packets = []models.RetrievedPacket{
		{
			Device: models.RetrievedDevice{ID: "device-1"},
		},
	}

	// Load more packets with continuation token
	m, _ = m.Update(PacketsLoadedMsg{
		Packets: []models.RetrievedPacket{
			{
				Device: models.RetrievedDevice{ID: "device-2"},
			},
		},
		ContinuationToken: "next-token",
		Append:            true,
	})

	assert.Equal(t, PacketsStateReady, m.state)
	assert.Len(t, m.packets, 2)
	assert.True(t, m.hasMore)
	assert.Equal(t, "next-token", m.continuationToken)
	assert.False(t, m.loadingMore)
}

func TestPacketsModel_PacketsLoadedMsg_NoMorePackets(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.state = PacketsStateLoading

	m, _ = m.Update(PacketsLoadedMsg{
		Packets: []models.RetrievedPacket{
			{
				Device: models.RetrievedDevice{ID: "device-1"},
			},
		},
		ContinuationToken: "",
		Append:            false,
	})

	assert.Equal(t, PacketsStateReady, m.state)
	assert.Len(t, m.packets, 1)
	assert.False(t, m.hasMore)
	assert.Empty(t, m.continuationToken)
}

func TestPacketsModel_ViewWithLoadMore(t *testing.T) {
	m := NewPacketsModel(nil, "")
	m.width = 80
	m.height = 24
	m.state = PacketsStateReady
	m.hasMore = true
	m.packets = []models.RetrievedPacket{
		{
			Device: models.RetrievedDevice{ID: "device-1"},
		},
	}

	view := m.View()

	assert.Contains(t, view, "more available")
	assert.Contains(t, view, "load more")
}

func TestFormatLocation(t *testing.T) {
	tests := []struct {
		loc      models.Location
		expected string
	}{
		{
			loc:      models.Location{Latitude: 37.7749, Longitude: -122.4194},
			expected: "37.7749, -122.4194",
		},
		{
			loc:      models.Location{Latitude: 0, Longitude: 0},
			expected: "Unknown",
		},
		{
			loc:      models.Location{Fake: true},
			expected: "Local scan",
		},
		{
			loc:      models.Location{Latitude: 40.7128, Longitude: -74.0060, Fake: true},
			expected: "Local scan",
		},
	}

	for _, tt := range tests {
		result := formatLocation(tt.loc)
		assert.Equal(t, tt.expected, result)
	}
}
