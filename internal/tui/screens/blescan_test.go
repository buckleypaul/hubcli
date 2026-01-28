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

	assert.Equal(t, BLEScanStateIdle, m.state)
	assert.Nil(t, m.client)
	assert.Empty(t, m.packets)
	assert.Equal(t, 30*time.Second, m.timeout)
}

func TestBLEScanModel_Init(t *testing.T) {
	m := NewBLEScanModel(nil)
	cmd := m.Init()

	// Init should return a spinner tick command
	assert.NotNil(t, cmd)
}

func TestBLEScanModel_WindowSizeMsg(t *testing.T) {
	m := NewBLEScanModel(nil)

	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

func TestBLEScanModel_StartScan(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateIdle

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	assert.NotNil(t, cmd)
}

func TestBLEScanModel_StartScan_WhenScanning(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateScanning

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	// Should remain in scanning state (no-op)
	assert.Equal(t, BLEScanStateScanning, m.state)
}

func TestBLEScanModel_StopScan(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateScanning

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Equal(t, BLEScanStateIdle, m.state)
}

func TestBLEScanModel_StopScan_WhenIdle(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateIdle

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	// Should remain idle (no-op)
	assert.Equal(t, BLEScanStateIdle, m.state)
}

func TestBLEScanModel_BackNavigation(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateIdle

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
	m.state = BLEScanStateIdle
	m.packets = []models.EncryptedPacket{{Payload: []byte{0x01}}}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	assert.Empty(t, m.packets)
}

func TestBLEScanModel_ClearPackets_WhenScanning(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateScanning
	m.packets = []models.EncryptedPacket{{Payload: []byte{0x01}}}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	// Should not clear when scanning
	assert.Len(t, m.packets, 1)
}

func TestBLEScanModel_ChangeTimeout(t *testing.T) {
	tests := []struct {
		key      rune
		expected time.Duration
	}{
		{'1', 10 * time.Second},
		{'3', 30 * time.Second},
		{'6', 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(string(tt.key), func(t *testing.T) {
			m := NewBLEScanModel(nil)
			m.state = BLEScanStateIdle

			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tt.key}})

			assert.Equal(t, tt.expected, m.timeout)
		})
	}
}

func TestBLEScanModel_ChangeTimeout_WhenScanning(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateScanning
	originalTimeout := m.timeout

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})

	// Should not change timeout when scanning
	assert.Equal(t, originalTimeout, m.timeout)
}

func TestBLEScanModel_IngestPackets(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateIdle
	m.packets = []models.EncryptedPacket{{Payload: []byte{0x01}}}

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	assert.Equal(t, BLEScanStateIngesting, m.state)
	assert.NotNil(t, cmd)
}

func TestBLEScanModel_IngestPackets_NoPackets(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateIdle
	m.packets = []models.EncryptedPacket{}

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	// Should not change state when no packets
	assert.Equal(t, BLEScanStateIdle, m.state)
	assert.Nil(t, cmd)
}

func TestBLEScanModel_BLEScanStartedMsg(t *testing.T) {
	m := NewBLEScanModel(nil)

	m, _ = m.Update(BLEScanStartedMsg{})

	assert.Equal(t, BLEScanStateScanning, m.state)
	assert.False(t, m.startTime.IsZero())
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

	assert.Equal(t, BLEScanStateIdle, m.state)
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
	assert.Equal(t, BLEScanStateIdle, m.state)
}

func TestBLEScanModel_BLEIngestCompleteMsg(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateIngesting
	m.packets = []models.EncryptedPacket{{Payload: []byte{0x01}}}

	m, _ = m.Update(BLEIngestCompleteMsg{Count: 1})

	assert.Equal(t, BLEScanStateIdle, m.state)
	assert.Empty(t, m.packets) // Packets cleared after ingestion
}

func TestBLEScanModel_BLEIngestCompleteMsg_WithError(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateIngesting
	m.packets = []models.EncryptedPacket{{Payload: []byte{0x01}}}

	m, _ = m.Update(BLEIngestCompleteMsg{Error: assert.AnError})

	assert.Equal(t, BLEScanStateError, m.state)
	assert.Error(t, m.err)
	assert.Len(t, m.packets, 1) // Packets not cleared on error
}

func TestBLEScanModel_View(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.width = 80
	m.height = 24
	m.state = BLEScanStateIdle

	view := m.View()

	assert.Contains(t, view, "BLE Scanner")
	assert.Contains(t, view, "IDLE")
	assert.Contains(t, view, "start")
}

func TestBLEScanModel_ViewScanning(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.width = 80
	m.height = 24
	m.state = BLEScanStateScanning
	m.startTime = time.Now()

	view := m.View()

	assert.Contains(t, view, "Scanning")
	assert.Contains(t, view, "SCANNING")
	assert.Contains(t, view, "stop")
}

func TestBLEScanModel_ViewIngesting(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.width = 80
	m.height = 24
	m.state = BLEScanStateIngesting
	m.packets = []models.EncryptedPacket{{Payload: []byte{0x01}}}

	view := m.View()

	assert.Contains(t, view, "Ingesting")
	assert.Contains(t, view, "INGESTING")
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
	m.state = BLEScanStateIdle
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
	assert.Contains(t, view, "ingest")
	assert.Contains(t, view, "clear")
}

func TestBLEScanModel_SetScanner(t *testing.T) {
	m := NewBLEScanModel(nil)
	mockScanner := ble.NewMockScanner()

	m.SetScanner(mockScanner)

	assert.Equal(t, mockScanner, m.scanner)
}

func TestBLEScanModel_BLEScanTickMsg(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateScanning
	m.startTime = time.Now()
	m.timeout = 30 * time.Second

	m, cmd := m.Update(BLEScanTickMsg{})

	// Should return another tick command
	assert.NotNil(t, cmd)
	assert.Equal(t, BLEScanStateScanning, m.state)
}

func TestBLEScanModel_BLEScanTickMsg_Timeout(t *testing.T) {
	m := NewBLEScanModel(nil)
	m.state = BLEScanStateScanning
	m.startTime = time.Now().Add(-60 * time.Second) // Started 60 seconds ago
	m.timeout = 30 * time.Second

	m, _ = m.Update(BLEScanTickMsg{})

	// Should stop scanning due to timeout
	assert.Equal(t, BLEScanStateIdle, m.state)
}
