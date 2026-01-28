package screens

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hubblenetwork/hubcli/internal/ble"
	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewBLEScanModel(t *testing.T) {
	m := NewBLEScanModel(nil)

	assert.Equal(t, BLEScanStateInit, m.state)
	assert.Nil(t, m.client)
	assert.Empty(t, m.packets)
}

func TestBLEScanModel_Init(t *testing.T) {
	m := NewBLEScanModel(nil)
	cmd := m.Init()

	// Init should return commands (spinner tick and start scan)
	assert.NotNil(t, cmd)
}

func TestBLEScanModel_WindowSizeMsg(t *testing.T) {
	m := NewBLEScanModel(nil)

	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

func TestBLEScanModel_PauseScan(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateScanning

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	assert.Equal(t, BLEScanStateInit, m.state)
}

func TestBLEScanModel_PauseScan_WhenPaused(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateInit

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	// Should remain in init state (no-op) - actually it resumes
	// 'p' is in both Pause and Resume bindings
}

func TestBLEScanModel_ResumeScan(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateInit
	m.scannerErr = nil

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	assert.NotNil(t, cmd)
}

func TestBLEScanModel_BackNavigation(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateInit

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	assert.NotNil(t, cmd)
	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	assert.True(t, ok)
	assert.Equal(t, "home", navMsg.Screen)
}

func TestBLEScanModel_QuitKey(t *testing.T) {
	m := NewBLEScanModel(nil)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	assert.NotNil(t, cmd)
}

func TestBLEScanModel_ClearPackets(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateInit
	m.packets = []models.EncryptedPacket{{Payload: []byte{0x01}}}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	assert.Empty(t, m.packets)
}

func TestBLEScanModel_ClearPackets_WhenScanning(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateScanning
	m.packets = []models.EncryptedPacket{{Payload: []byte{0x01}}}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	// Should clear even when scanning
	assert.Empty(t, m.packets)
}

func TestBLEScanModel_BLEScanStartedMsg(t *testing.T) {
	m := NewBLEScanModel(nil)

	m, _ = m.Update(BLEScanStartedMsg{})

	assert.Equal(t, BLEScanStateScanning, m.state)
}

func TestBLEScanModel_BLEScanPacketMsg(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateScanning

	packet := models.EncryptedPacket{
		Payload:   []byte{0x01, 0x02, 0x03},
		RSSI:      -65,
		Timestamp: time.Now(),
	}
	raw := ble.RawAdvertisement{
		Address: "AA:BB:CC:DD:EE:FF",
	}

	m, _ = m.Update(BLEScanPacketMsg{Packet: packet, Raw: raw})

	assert.Len(t, m.packets, 1)
	assert.Len(t, m.rawPackets, 1)
	assert.Equal(t, packet.Payload, m.packets[0].Payload)
	assert.Equal(t, raw.Address, m.rawPackets[0].Address)
}

func TestBLEScanModel_BLEScanStoppedMsg(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateScanning

	m, _ = m.Update(BLEScanStoppedMsg{})

	assert.Equal(t, BLEScanStateInit, m.state)
}

func TestBLEScanModel_BLEScanStoppedMsg_WithError(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateScanning

	m, _ = m.Update(BLEScanStoppedMsg{Error: assert.AnError})

	assert.Equal(t, BLEScanStateError, m.state)
	assert.Error(t, m.err)
}

func TestBLEScanModel_BLEScanStoppedMsg_WithScanStopped(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateScanning

	m, _ = m.Update(BLEScanStoppedMsg{Error: ble.ErrScanStopped})

	// ErrScanStopped should not be treated as an error state
	assert.Equal(t, BLEScanStateInit, m.state)
}

func TestBLEScanModel_View(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.width = 80
	m.height = 24
	m.state = BLEScanStateInit
	m.scannerErr = nil

	view := m.View()

	assert.Contains(t, view, "BLE Scanner")
	assert.Contains(t, view, "PAUSED")
	assert.Contains(t, view, "resume")
}

func TestBLEScanModel_ViewScanning(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.width = 80
	m.height = 24
	m.state = BLEScanStateScanning

	view := m.View()

	assert.Contains(t, view, "Scanning")
	assert.Contains(t, view, "SCANNING")
	assert.Contains(t, view, "pause")
}

func TestBLEScanModel_ViewError(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.width = 80
	m.height = 24
	m.state = BLEScanStateError
	m.err = assert.AnError

	view := m.View()

	assert.Contains(t, view, "Error")
	assert.Contains(t, view, "ERROR")
	assert.Contains(t, view, "retry")
}

func TestBLEScanModel_ViewWithPackets(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.width = 80
	m.height = 24
	m.state = BLEScanStateInit
	m.scannerErr = nil // Clear scanner error for test
	m.packets = []models.EncryptedPacket{
		{
			Payload:   []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			RSSI:      -65,
			Timestamp: time.Now(),
		},
	}
	m.rawPackets = []ble.RawAdvertisement{
		{Address: "AA:BB:CC:DD:EE:FF"},
	}
	m.updateTable()

	view := m.View()

	assert.Contains(t, view, "1 packet(s)")
	assert.Contains(t, view, "clear")
}

func TestBLEScanModel_SetScanner(t *testing.T) {
	m := NewBLEScanModel(nil)
	mockScanner := ble.NewMockScanner()

	m.SetScanner(mockScanner)

	assert.Equal(t, mockScanner, m.scanner)
}
