package screens

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hubblenetwork/hubcli/internal/api"
	"github.com/hubblenetwork/hubcli/internal/ble"
	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/hubblenetwork/hubcli/internal/tui/common"
)

// BLEScanState represents the current state of the BLE scan screen
type BLEScanState int

const (
	BLEScanStateInit BLEScanState = iota // Initial state before scanning starts
	BLEScanStateScanning
	BLEScanStateError
)

// BLE scan messages
type (
	// BLEScanStartedMsg indicates scanning has started
	BLEScanStartedMsg struct {
		Results <-chan ble.ScanResult
	}

	// BLEScanPacketMsg is sent when a packet is discovered
	BLEScanPacketMsg struct {
		Packet models.EncryptedPacket
		Raw    ble.RawAdvertisement
	}

	// BLEScanStoppedMsg indicates scanning has stopped
	BLEScanStoppedMsg struct {
		Error error
	}

	// BLEScanTickMsg is sent periodically during scanning
	BLEScanTickMsg struct{}
)

// BLEScanModel is the model for the BLE scan screen
type BLEScanModel struct {
	client      *api.Client
	scanner     ble.ScannerInterface
	packets     []models.EncryptedPacket
	rawPackets  []ble.RawAdvertisement
	table       table.Model
	spinner     spinner.Model
	help        help.Model
	keys        bleScanKeyMap

	state       BLEScanState
	err         error
	scanCtx     context.Context
	cancelScan  context.CancelFunc
	width       int
	height      int
	scannerErr  error // Error from initializing scanner
	resultsChan <-chan ble.ScanResult
}

// bleScanKeyMap defines key bindings for the BLE scan screen
type bleScanKeyMap struct {
	Pause  key.Binding
	Resume key.Binding
	Clear  key.Binding
	Back   key.Binding
	Quit   key.Binding
}

func defaultBLEScanKeyMap() bleScanKeyMap {
	return bleScanKeyMap{
		Pause: key.NewBinding(
			key.WithKeys("p", " "),
			key.WithHelp("p/space", "pause"),
		),
		Resume: key.NewBinding(
			key.WithKeys("p", " ", "r"),
			key.WithHelp("p/space/r", "resume"),
		),
		Clear: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clear"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
	}
}

// NewBLEScanModel creates a new BLE scan screen model
func NewBLEScanModel(client *api.Client) BLEScanModel {
	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "Time", Width: 13},
		{Title: "RSSI", Width: 7},
		{Title: "Ver", Width: 4},
		{Title: "Seq", Width: 5},
		{Title: "Device ID", Width: 10},
		{Title: "Auth Tag", Width: 10},
		{Title: "Encrypted Payload", Width: 18},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		Foreground(common.ColorSecondary).
		BorderStyle(lipgloss.HiddenBorder())
	s.Cell = s.Cell.
		BorderStyle(lipgloss.HiddenBorder())
	s.Selected = s.Selected.
		Foreground(common.ColorForeground).
		Background(common.ColorPrimary).
		Bold(true)
	t.SetStyles(s)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(common.ColorPrimary)

	// Try to create a real scanner
	var scanner ble.ScannerInterface
	var scannerErr error
	realScanner, err := ble.NewScanner()
	if err != nil {
		scannerErr = err
		scanner = ble.NewMockScanner() // Fallback to mock
	} else {
		scanner = realScanner
	}

	return BLEScanModel{
		client:     client,
		scanner:    scanner,
		scannerErr: scannerErr,
		table:      t,
		spinner:    sp,
		help:       help.New(),
		keys:       defaultBLEScanKeyMap(),
		state:      BLEScanStateInit,
	}
}

// Init initializes the BLE scan model and starts scanning automatically
func (m BLEScanModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startScan())
}

// Update handles messages for the BLE scan screen
func (m BLEScanModel) Update(msg tea.Msg) (BLEScanModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.table.SetHeight(m.height - 20)
		m.updateTableColumns()
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			if m.state == BLEScanStateScanning {
				m.stopScan()
			}
			return m, func() tea.Msg {
				return NavigateMsg{Screen: "home"}
			}

		case key.Matches(msg, m.keys.Quit):
			if m.state == BLEScanStateScanning {
				m.stopScan()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keys.Pause) || key.Matches(msg, m.keys.Resume):
			// Toggle between scanning and paused states
			if m.state == BLEScanStateScanning {
				m.stopScan()
				m.state = BLEScanStateInit
				return m, nil
			} else if m.state == BLEScanStateInit || m.state == BLEScanStateError {
				return m, m.startScan()
			}

		case key.Matches(msg, m.keys.Clear):
			m.packets = nil
			m.rawPackets = nil
			m.updateTable()
			return m, nil
		}

	case BLEScanStartedMsg:
		m.state = BLEScanStateScanning
		m.resultsChan = msg.Results // Store the channel from the message
		// Start tick loop for continuous polling
		return m, tea.Batch(m.spinner.Tick, m.tickCmd())

	case BLEScanPacketMsg:
		m.packets = append(m.packets, msg.Packet)
		m.rawPackets = append(m.rawPackets, msg.Raw)
		m.updateTable()
		// Continue polling for more results
		if m.state == BLEScanStateScanning {
			return m, m.pollResults()
		}
		return m, nil

	case BLEScanStoppedMsg:
		m.state = BLEScanStateInit
		if msg.Error != nil && msg.Error != ble.ErrScanStopped {
			m.state = BLEScanStateError
			m.err = msg.Error
		}
		return m, nil

	case BLEScanTickMsg:
		// Continuous polling while scanning
		if m.state == BLEScanStateScanning {
			// Poll for results and schedule next tick
			result := m.pollResultsSync()
			if result != nil {
				// Process the result through Update recursively
				m, cmd := m.Update(result)
				return m, tea.Batch(cmd, m.tickCmd())
			}
			// No result yet, just schedule next tick
			return m, m.tickCmd()
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == BLEScanStateScanning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Update table
	if (m.state == BLEScanStateInit || m.state == BLEScanStateScanning) && len(m.packets) > 0 {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the BLE scan screen
func (m BLEScanModel) View() string {
	var content strings.Builder

	// Helper to center text within terminal width
	centerText := func(s string) string {
		return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(s)
	}

	// Header (centered)
	content.WriteString(centerText(common.TitleStyle.Render("BLE Scanner")))
	content.WriteString("\n")
	content.WriteString(centerText(common.SubtitleStyle.Render("Scan for Hubble BLE advertisements")))
	content.WriteString("\n\n")

	// Status bar (centered)
	content.WriteString(centerText(m.renderStatus()))
	content.WriteString("\n\n")

	// Main content
	switch m.state {
	case BLEScanStateScanning:
		content.WriteString(centerText(fmt.Sprintf("%s Scanning...", m.spinner.View())))
		content.WriteString("\n\n")
		content.WriteString(centerText(fmt.Sprintf("Found %d packet(s)", len(m.packets))))
		content.WriteString("\n\n")
		content.WriteString(m.table.View())

	case BLEScanStateError:
		content.WriteString(centerText(common.ErrorTextStyle.Render("Error: " + m.err.Error())))
		content.WriteString("\n\n")
		content.WriteString(centerText(common.MutedTextStyle.Render("Press 'r' to retry")))

	case BLEScanStateInit:
		if m.scannerErr != nil {
			content.WriteString(centerText(common.ErrorTextStyle.Render("Scanner Error: " + m.scannerErr.Error())))
			content.WriteString("\n\n")
			content.WriteString(centerText(common.MutedTextStyle.Render("BLE scanning may not be available.")))
		} else {
			content.WriteString(centerText(fmt.Sprintf("Scan paused. %d packet(s) captured", len(m.packets))))
			content.WriteString("\n\n")
			content.WriteString(m.table.View())
		}
	}

	// Help (centered)
	content.WriteString("\n\n")
	content.WriteString(centerText(m.renderHelp()))

	// Center vertically, but use full width
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Left,
		lipgloss.Center,
		lipgloss.NewStyle().Width(m.width).Render(content.String()),
	)
}

func (m BLEScanModel) renderStatus() string {
	var parts []string

	// Packet count
	countStyle := lipgloss.NewStyle().
		Foreground(common.ColorForeground).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1)

	countStr := fmt.Sprintf("Packets: %d", len(m.packets))
	parts = append(parts, countStyle.Render(countStr))

	// State indicator
	var stateStr string
	var stateStyle lipgloss.Style

	switch m.state {
	case BLEScanStateInit:
		stateStr = "PAUSED"
		stateStyle = lipgloss.NewStyle().
			Foreground(common.ColorMuted).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1)
	case BLEScanStateScanning:
		stateStr = "SCANNING"
		stateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(common.ColorPrimary).
			Bold(true).
			Padding(0, 1)
	case BLEScanStateError:
		stateStr = "ERROR"
		stateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(common.ColorError).
			Bold(true).
			Padding(0, 1)
	}

	parts = append(parts, stateStyle.Render(stateStr))

	return strings.Join(parts, "  ")
}

func (m BLEScanModel) renderHelp() string {
	var helpText []string

	switch m.state {
	case BLEScanStateInit:
		helpText = []string{
			common.FormatHelp("r/space", "resume"),
			common.FormatHelp("c", "clear"),
		}
	case BLEScanStateScanning:
		helpText = []string{
			common.FormatHelp("p/space", "pause"),
			common.FormatHelp("c", "clear"),
		}
	case BLEScanStateError:
		helpText = []string{
			common.FormatHelp("r", "retry"),
		}
	}

	helpText = append(helpText, common.FormatHelp("esc", "back"))

	return strings.Join(helpText, "  ")
}

func (m *BLEScanModel) updateTableColumns() {
	if m.width == 0 {
		return
	}

	// Fixed minimum widths for each column
	const (
		minNum       = 4
		minTime      = 13
		minRSSI      = 7
		minVer       = 4
		minSeq       = 5
		minDeviceID  = 10
		minAuthTag   = 10
		minEncrypted = 18
	)

	// Calculate extra space to distribute
	minTotal := minNum + minTime + minRSSI + minVer + minSeq + minDeviceID + minAuthTag + minEncrypted
	extraSpace := m.width - minTotal

	if extraSpace < 0 {
		extraSpace = 0
	}

	// Distribute extra space to the Encrypted Payload column
	colEncrypted := minEncrypted + extraSpace

	columns := []table.Column{
		{Title: "#", Width: minNum},
		{Title: "Time", Width: minTime},
		{Title: "RSSI", Width: minRSSI},
		{Title: "Ver", Width: minVer},
		{Title: "Seq", Width: minSeq},
		{Title: "Device ID", Width: minDeviceID},
		{Title: "Auth Tag", Width: minAuthTag},
		{Title: "Encrypted Payload", Width: colEncrypted},
	}
	m.table.SetColumns(columns)
	// Set table width to sum of column widths
	tableWidth := minNum + minTime + minRSSI + minVer + minSeq + minDeviceID + minAuthTag + colEncrypted
	m.table.SetWidth(tableWidth)
}

func (m *BLEScanModel) updateTable() {
	rows := make([]table.Row, len(m.packets))

	// Calculate encrypted payload display width based on current terminal width (matches updateTableColumns)
	const minEncrypted = 18
	encryptedDisplayWidth := minEncrypted
	if m.width > 0 {
		minTotal := 4 + 13 + 7 + 4 + 5 + 10 + 10 + minEncrypted
		extraSpace := m.width - minTotal
		if extraSpace < 0 {
			extraSpace = 0
		}
		encryptedDisplayWidth = minEncrypted + extraSpace
	}

	// Display newest packets first (time-descending order)
	for i := len(m.packets) - 1; i >= 0; i-- {
		p := m.packets[i]
		rowIdx := len(m.packets) - 1 - i // Row index for the table (0 = newest)

		rssiStr := fmt.Sprintf("%d", p.RSSI)

		// Parse payload structure:
		// Byte 0–1 : [Protocol Version (6 bits) | SeqNo (10 bits)]
		// Byte 2–5 : Ephemeral Device Identifier (32 bits)
		// Byte 6–9 : Authentication Tag (32 bits)
		// Bytes 10+: Encrypted payload (0-13 bytes)
		verStr, seqStr, deviceIDStr, authTagStr, encryptedStr := parsePayloadFields(p.Payload, encryptedDisplayWidth)

		rows[rowIdx] = table.Row{
			fmt.Sprintf("%d", i+1), // Keep original packet number for reference
			p.Timestamp.Format("15:04:05.000"),
			rssiStr,
			verStr,
			seqStr,
			deviceIDStr,
			authTagStr,
			encryptedStr,
		}
	}
	m.table.SetRows(rows)
}

// parsePayloadFields extracts the structured fields from the payload
// Byte 0–1 : [Protocol Version (6 bits) | SeqNo (10 bits)]
// Byte 2–5 : Ephemeral Device Identifier (32 bits)
// Byte 6–9 : Authentication Tag (32 bits)
// Bytes 10+: Encrypted payload (0-13 bytes)
func parsePayloadFields(payload []byte, maxEncryptedWidth int) (ver, seq, deviceID, authTag, encrypted string) {
	if len(payload) < 2 {
		return "-", "-", "-", "-", "-"
	}

	// Byte 0-1: Protocol Version (6 bits) | SeqNo (10 bits)
	// Big-endian (network byte order):
	// Byte 0: [V5 V4 V3 V2 V1 V0 S9 S8] - version (6 bits) + seq high (2 bits)
	// Byte 1: [S7 S6 S5 S4 S3 S2 S1 S0] - seq low (8 bits)
	header := binary.BigEndian.Uint16(payload[0:2])
	version := (header >> 10) & 0x3F // Top 6 bits
	seqNo := header & 0x03FF         // Bottom 10 bits
	ver = fmt.Sprintf("%d", version)
	seq = fmt.Sprintf("%d", seqNo)

	// Byte 2-5: Ephemeral Device Identifier (32 bits)
	if len(payload) >= 6 {
		deviceID = fmt.Sprintf("%08x", payload[2:6])
	} else {
		deviceID = "-"
	}

	// Byte 6-9: Authentication Tag (32 bits)
	if len(payload) >= 10 {
		authTag = fmt.Sprintf("%08x", payload[6:10])
	} else {
		authTag = "-"
	}

	// Bytes 10+: Encrypted payload (0-13 bytes)
	if len(payload) > 10 {
		encPayload := payload[10:]
		encrypted = fmt.Sprintf("%x", encPayload)
		if len(encrypted) > maxEncryptedWidth {
			encrypted = encrypted[:maxEncryptedWidth-3] + "..."
		}
	} else {
		encrypted = "-"
	}

	return
}

func (m *BLEScanModel) startScan() tea.Cmd {
	// Check if scanner initialization failed
	if m.scannerErr != nil {
		return func() tea.Msg {
			return BLEScanStoppedMsg{Error: m.scannerErr}
		}
	}

	m.scanCtx, m.cancelScan = context.WithCancel(context.Background())

	return func() tea.Msg {
		opts := ble.ScanOptions{
			Timeout:          0, // No timeout - scan continuously
			FilterHubbleOnly: true,
			Location: models.Location{
				Fake:      true,
				Timestamp: time.Now(),
			},
		}

		results, err := m.scanner.ScanStream(m.scanCtx, opts)
		if err != nil {
			return BLEScanStoppedMsg{Error: err}
		}

		// Return the channel in the message - it will be stored in Update
		return BLEScanStartedMsg{Results: results}
	}
}

func (m *BLEScanModel) stopScan() {
	if m.cancelScan != nil {
		m.cancelScan()
		m.cancelScan = nil
	}
	if m.scanner != nil {
		m.scanner.Stop()
	}
	m.resultsChan = nil
}

func (m BLEScanModel) tickCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return BLEScanTickMsg{}
	})
}

// pollResultsSync checks for results synchronously (non-blocking)
func (m *BLEScanModel) pollResultsSync() tea.Msg {
	if m.resultsChan == nil {
		return nil
	}

	select {
	case result, ok := <-m.resultsChan:
		if !ok {
			// Channel closed, scan complete
			return BLEScanStoppedMsg{}
		}
		if result.Packet != nil {
			return BLEScanPacketMsg{
				Packet: *result.Packet,
				Raw:    result.Raw,
			}
		}
		// Parse error or non-matching advertisement - continue scanning
		// Don't treat ErrNotHubblePacket as a fatal error
		return nil
	default:
		// No result available yet
		return nil
	}
}

func (m *BLEScanModel) pollResults() tea.Cmd {
	return func() tea.Msg {
		if m.resultsChan == nil {
			return nil
		}

		select {
		case result, ok := <-m.resultsChan:
			if !ok {
				// Channel closed, scan complete
				return BLEScanStoppedMsg{}
			}
			if result.Error != nil {
				return BLEScanStoppedMsg{Error: result.Error}
			}
			if result.Packet != nil {
				return BLEScanPacketMsg{
					Packet: *result.Packet,
					Raw:    result.Raw,
				}
			}
			return nil
		default:
			// No result available yet
			return nil
		}
	}
}

// SetScanner allows setting a custom scanner (useful for testing)
func (m *BLEScanModel) SetScanner(scanner ble.ScannerInterface) {
	m.scanner = scanner
	m.scannerErr = nil
}
