package screens

import (
	"context"
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
	BLEScanStateIdle BLEScanState = iota
	BLEScanStateScanning
	BLEScanStateIngesting
	BLEScanStateError
)

// BLE scan messages
type (
	// BLEScanStartedMsg indicates scanning has started
	BLEScanStartedMsg struct{}

	// BLEScanPacketMsg is sent when a packet is discovered
	BLEScanPacketMsg struct {
		Packet models.EncryptedPacket
		Raw    ble.RawAdvertisement
	}

	// BLEScanStoppedMsg indicates scanning has stopped
	BLEScanStoppedMsg struct {
		Error error
	}

	// BLEIngestCompleteMsg indicates packet ingestion is complete
	BLEIngestCompleteMsg struct {
		Count int
		Error error
	}

	// BLEScanTickMsg is sent periodically during scanning
	BLEScanTickMsg struct{}
)

// BLEScanModel is the model for the BLE scan screen
type BLEScanModel struct {
	client      *api.Client
	scanner     *ble.MockScanner // Using mock for now, will be replaced with real scanner
	packets     []models.EncryptedPacket
	rawPackets  []ble.RawAdvertisement
	table       table.Model
	spinner     spinner.Model
	help        help.Model
	keys        bleScanKeyMap

	state      BLEScanState
	err        error
	scanCtx    context.Context
	cancelScan context.CancelFunc
	timeout    time.Duration
	startTime  time.Time
	width      int
	height     int
}

// bleScanKeyMap defines key bindings for the BLE scan screen
type bleScanKeyMap struct {
	Start   key.Binding
	Stop    key.Binding
	Ingest  key.Binding
	Clear   key.Binding
	Back    key.Binding
	Quit    key.Binding
	Timeout key.Binding
}

func defaultBLEScanKeyMap() bleScanKeyMap {
	return bleScanKeyMap{
		Start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start scan"),
		),
		Stop: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "stop scan"),
		),
		Ingest: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "ingest to cloud"),
		),
		Clear: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clear packets"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Timeout: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "set timeout"),
		),
	}
}

// NewBLEScanModel creates a new BLE scan screen model
func NewBLEScanModel(client *api.Client) BLEScanModel {
	columns := []table.Column{
		{Title: "#", Width: 4},
		{Title: "Time", Width: 12},
		{Title: "RSSI", Width: 6},
		{Title: "Address", Width: 18},
		{Title: "Payload (hex)", Width: 40},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(common.ColorBorder).
		BorderBottom(true).
		Bold(true).
		Foreground(common.ColorSecondary)
	s.Selected = s.Selected.
		Foreground(common.ColorForeground).
		Background(common.ColorPrimary).
		Bold(true)
	t.SetStyles(s)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(common.ColorPrimary)

	return BLEScanModel{
		client:  client,
		scanner: ble.NewMockScanner(),
		table:   t,
		spinner: sp,
		help:    help.New(),
		keys:    defaultBLEScanKeyMap(),
		state:   BLEScanStateIdle,
		timeout: 30 * time.Second,
	}
}

// Init initializes the BLE scan model
func (m BLEScanModel) Init() tea.Cmd {
	return m.spinner.Tick
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

		case key.Matches(msg, m.keys.Start):
			if m.state == BLEScanStateIdle || m.state == BLEScanStateError {
				return m, m.startScan()
			}

		case key.Matches(msg, m.keys.Stop):
			if m.state == BLEScanStateScanning {
				m.stopScan()
				m.state = BLEScanStateIdle
				return m, nil
			}

		case key.Matches(msg, m.keys.Ingest):
			if m.state == BLEScanStateIdle && len(m.packets) > 0 {
				m.state = BLEScanStateIngesting
				return m, tea.Batch(m.spinner.Tick, m.ingestPackets())
			}

		case key.Matches(msg, m.keys.Clear):
			if m.state == BLEScanStateIdle {
				m.packets = nil
				m.rawPackets = nil
				m.updateTable()
				return m, nil
			}

		case msg.String() == "1":
			if m.state == BLEScanStateIdle {
				m.timeout = 10 * time.Second
				return m, nil
			}
		case msg.String() == "3":
			if m.state == BLEScanStateIdle {
				m.timeout = 30 * time.Second
				return m, nil
			}
		case msg.String() == "6":
			if m.state == BLEScanStateIdle {
				m.timeout = 60 * time.Second
				return m, nil
			}
		}

	case BLEScanStartedMsg:
		m.state = BLEScanStateScanning
		m.startTime = time.Now()
		return m, tea.Batch(m.spinner.Tick, m.tickCmd())

	case BLEScanPacketMsg:
		m.packets = append(m.packets, msg.Packet)
		m.rawPackets = append(m.rawPackets, msg.Raw)
		m.updateTable()
		return m, nil

	case BLEScanStoppedMsg:
		m.state = BLEScanStateIdle
		if msg.Error != nil && msg.Error != ble.ErrScanStopped {
			m.state = BLEScanStateError
			m.err = msg.Error
		}
		return m, nil

	case BLEScanTickMsg:
		if m.state == BLEScanStateScanning {
			// Check if timeout reached
			if time.Since(m.startTime) >= m.timeout {
				m.stopScan()
				m.state = BLEScanStateIdle
				return m, nil
			}
			return m, m.tickCmd()
		}

	case BLEIngestCompleteMsg:
		m.state = BLEScanStateIdle
		if msg.Error != nil {
			m.state = BLEScanStateError
			m.err = msg.Error
		} else {
			// Clear packets after successful ingestion
			m.packets = nil
			m.rawPackets = nil
			m.updateTable()
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == BLEScanStateScanning || m.state == BLEScanStateIngesting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Update table
	if m.state == BLEScanStateIdle && len(m.packets) > 0 {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the BLE scan screen
func (m BLEScanModel) View() string {
	var content strings.Builder

	// Header
	content.WriteString(common.TitleStyle.Render("BLE Scanner"))
	content.WriteString("\n")
	content.WriteString(common.SubtitleStyle.Render("Scan for Hubble BLE advertisements"))
	content.WriteString("\n\n")

	// Status bar
	content.WriteString(m.renderStatus())
	content.WriteString("\n\n")

	// Main content
	switch m.state {
	case BLEScanStateScanning:
		content.WriteString(fmt.Sprintf("%s Scanning... (%s elapsed)\n\n",
			m.spinner.View(),
			time.Since(m.startTime).Round(time.Second)))
		content.WriteString(fmt.Sprintf("Found %d packet(s)\n\n", len(m.packets)))
		if len(m.packets) > 0 {
			content.WriteString(m.table.View())
		}

	case BLEScanStateIngesting:
		content.WriteString(fmt.Sprintf("%s Ingesting %d packet(s) to cloud...\n",
			m.spinner.View(), len(m.packets)))

	case BLEScanStateError:
		content.WriteString(common.ErrorTextStyle.Render("Error: " + m.err.Error()))
		content.WriteString("\n\n")
		content.WriteString(common.MutedTextStyle.Render("Press 's' to try again"))

	case BLEScanStateIdle:
		if len(m.packets) == 0 {
			content.WriteString(common.MutedTextStyle.Render("No packets captured yet."))
			content.WriteString("\n\n")
			content.WriteString(common.MutedTextStyle.Render("Press 's' to start scanning."))
		} else {
			content.WriteString(fmt.Sprintf("%d packet(s) captured\n\n", len(m.packets)))
			content.WriteString(m.table.View())
		}
	}

	// Help
	content.WriteString("\n\n")
	content.WriteString(m.renderHelp())

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content.String(),
	)
}

func (m BLEScanModel) renderStatus() string {
	var parts []string

	// Timeout setting
	timeoutStyle := lipgloss.NewStyle().
		Foreground(common.ColorMuted).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1)

	timeoutStr := fmt.Sprintf("Timeout: %ds", int(m.timeout.Seconds()))
	parts = append(parts, timeoutStyle.Render(timeoutStr))

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
	case BLEScanStateIdle:
		stateStr = "IDLE"
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
	case BLEScanStateIngesting:
		stateStr = "INGESTING"
		stateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(common.ColorWarning).
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
	case BLEScanStateIdle:
		helpText = []string{
			common.FormatHelp("s", "start"),
			common.FormatHelp("1/3/6", "timeout 10/30/60s"),
		}
		if len(m.packets) > 0 {
			helpText = append(helpText,
				common.FormatHelp("i", "ingest"),
				common.FormatHelp("c", "clear"),
			)
		}
	case BLEScanStateScanning:
		helpText = []string{
			common.FormatHelp("x", "stop"),
		}
	case BLEScanStateIngesting:
		helpText = []string{
			common.FormatHelp("", "please wait..."),
		}
	case BLEScanStateError:
		helpText = []string{
			common.FormatHelp("s", "retry"),
		}
	}

	helpText = append(helpText, common.FormatHelp("esc", "back"))

	return strings.Join(helpText, "  ")
}

func (m *BLEScanModel) updateTable() {
	rows := make([]table.Row, len(m.packets))
	for i, p := range m.packets {
		payloadHex := fmt.Sprintf("%x", p.Payload)
		if len(payloadHex) > 40 {
			payloadHex = payloadHex[:37] + "..."
		}

		rssiStr := fmt.Sprintf("%d", p.RSSI)
		if p.RSSI >= -50 {
			rssiStr = fmt.Sprintf("%d ++", p.RSSI)
		} else if p.RSSI >= -70 {
			rssiStr = fmt.Sprintf("%d +", p.RSSI)
		}

		address := ""
		if i < len(m.rawPackets) {
			address = truncate(m.rawPackets[i].Address, 18)
		}

		rows[i] = table.Row{
			fmt.Sprintf("%d", i+1),
			p.Timestamp.Format("15:04:05.000"),
			rssiStr,
			address,
			payloadHex,
		}
	}
	m.table.SetRows(rows)
}

func (m *BLEScanModel) startScan() tea.Cmd {
	m.scanCtx, m.cancelScan = context.WithCancel(context.Background())

	return func() tea.Msg {
		// For now, return a started message
		// In production, this would start the actual BLE scan
		return BLEScanStartedMsg{}
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
}

func (m BLEScanModel) tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return BLEScanTickMsg{}
	})
}

func (m BLEScanModel) ingestPackets() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return BLEIngestCompleteMsg{Error: fmt.Errorf("no API client")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Convert packets to the format expected by the API
		encryptedPackets := make([]models.EncryptedPacket, len(m.packets))
		copy(encryptedPackets, m.packets)

		err := m.client.IngestEncryptedPackets(ctx, encryptedPackets)
		if err != nil {
			return BLEIngestCompleteMsg{Error: err}
		}

		return BLEIngestCompleteMsg{Count: len(m.packets)}
	}
}

// SetScanner allows setting a custom scanner (useful for testing)
func (m *BLEScanModel) SetScanner(scanner *ble.MockScanner) {
	m.scanner = scanner
}
