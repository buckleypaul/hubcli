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
	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/hubblenetwork/hubcli/internal/tui/common"
)

// PacketsState represents the current state of the packets screen
type PacketsState int

const (
	PacketsStateLoading PacketsState = iota
	PacketsStateReady
	PacketsStateError
)

// Packets screen messages
type (
	// PacketsLoadedMsg is sent when packets are fetched
	PacketsLoadedMsg struct {
		Packets           []models.RetrievedPacket
		ContinuationToken string
		Append            bool // If true, append to existing packets
	}

	// PacketsErrorMsg is sent when fetching fails
	PacketsErrorMsg struct {
		Err error
	}
)

// PacketsModel is the model for the packets screen
type PacketsModel struct {
	client   *api.Client
	packets  []models.RetrievedPacket
	table    table.Model
	spinner  spinner.Model
	help     help.Model
	keys     common.ListKeyMap

	state             PacketsState
	err               error
	deviceID          string // Optional filter by device ID
	days              int    // Number of days to query
	width             int
	height            int
	continuationToken string // Token for loading more packets
	hasMore           bool   // Whether more packets are available
	loadingMore       bool   // Whether currently loading more packets
}

// NewPacketsModel creates a new packets screen model
func NewPacketsModel(client *api.Client, deviceID string) PacketsModel {
	columns := []table.Column{
		{Title: "Device ID", Width: 18},
		{Title: "Timestamp", Width: 20},
		{Title: "Location", Width: 25},
		{Title: "Payload", Width: 30},
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

	return PacketsModel{
		client:   client,
		table:    t,
		spinner:  sp,
		help:     help.New(),
		keys:     common.DefaultListKeyMap(),
		state:    PacketsStateLoading,
		deviceID: deviceID,
		days:     7, // Default to 7 days
	}
}

// Init initializes the packets model
func (m PacketsModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadPackets(false),
	)
}

// Update handles messages for the packets screen
func (m PacketsModel) Update(msg tea.Msg) (PacketsModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		m.table.SetHeight(m.height - 15)
		m.updateColumnWidths()
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			return m, func() tea.Msg {
				return NavigateMsg{Screen: "back"}
			}

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Refresh):
			if m.state == PacketsStateReady || m.state == PacketsStateError {
				m.state = PacketsStateLoading
				m.continuationToken = ""
				return m, tea.Batch(m.spinner.Tick, m.loadPackets(false))
			}

		case msg.String() == "1":
			m.days = 1
			m.state = PacketsStateLoading
			m.continuationToken = ""
			return m, tea.Batch(m.spinner.Tick, m.loadPackets(false))

		case msg.String() == "7":
			m.days = 7
			m.state = PacketsStateLoading
			m.continuationToken = ""
			return m, tea.Batch(m.spinner.Tick, m.loadPackets(false))

		case msg.String() == "3" && msg.Alt:
			m.days = 30
			m.state = PacketsStateLoading
			m.continuationToken = ""
			return m, tea.Batch(m.spinner.Tick, m.loadPackets(false))

		case msg.String() == "c":
			// Clear device filter
			if m.deviceID != "" {
				m.deviceID = ""
				m.continuationToken = ""
				m.state = PacketsStateLoading
				return m, tea.Batch(m.spinner.Tick, m.loadPackets(false))
			}

		case msg.String() == "m":
			// Load more packets
			if m.state == PacketsStateReady && m.hasMore && !m.loadingMore {
				m.loadingMore = true
				return m, m.loadPackets(true)
			}
		}

	case PacketsLoadedMsg:
		m.state = PacketsStateReady
		m.loadingMore = false
		if msg.Append {
			m.packets = append(m.packets, msg.Packets...)
		} else {
			m.packets = msg.Packets
		}
		m.continuationToken = msg.ContinuationToken
		m.hasMore = msg.ContinuationToken != ""
		m.updateTable()
		return m, nil

	case PacketsErrorMsg:
		m.state = PacketsStateError
		m.err = msg.Err
		return m, nil

	case spinner.TickMsg:
		if m.state == PacketsStateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Update table
	if m.state == PacketsStateReady {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the packets screen
func (m PacketsModel) View() string {
	var content strings.Builder

	// Header
	content.WriteString(common.TitleStyle.Render("Packets"))
	content.WriteString("\n")

	subtitle := "View packet history"
	if m.deviceID != "" {
		subtitle = fmt.Sprintf("Packets for device: %s", truncate(m.deviceID, 20))
	}
	content.WriteString(common.SubtitleStyle.Render(subtitle))
	content.WriteString("\n\n")

	// Time range indicator
	timeRange := fmt.Sprintf("Showing last %d day(s)", m.days)
	content.WriteString(common.MutedTextStyle.Render(timeRange))
	content.WriteString("\n\n")

	switch m.state {
	case PacketsStateLoading:
		content.WriteString(fmt.Sprintf("%s Loading packets...", m.spinner.View()))

	case PacketsStateError:
		content.WriteString(common.ErrorTextStyle.Render("Error: " + m.err.Error()))
		content.WriteString("\n\n")
		content.WriteString(common.MutedTextStyle.Render("Press 'r' to retry"))

	case PacketsStateReady:
		if len(m.packets) == 0 {
			content.WriteString(common.MutedTextStyle.Render("No packets found in the selected time range."))
		} else {
			// Packet count
			countText := fmt.Sprintf("%d packet(s)", len(m.packets))
			if m.hasMore {
				countText += " (more available)"
			}
			if m.loadingMore {
				countText += " - loading more..."
			}
			content.WriteString(common.MutedTextStyle.Render(countText))
			content.WriteString("\n\n")

			// Table
			content.WriteString(m.table.View())
		}
	}

	// Help
	content.WriteString("\n\n")
	helpText := []string{
		common.FormatHelp("↑/↓", "navigate"),
		common.FormatHelp("1/7", "1/7 days"),
		common.FormatHelp("r", "refresh"),
	}
	if m.hasMore && !m.loadingMore {
		helpText = append(helpText, common.FormatHelp("m", "load more"))
	}
	if m.deviceID != "" {
		helpText = append(helpText, common.FormatHelp("c", "clear filter"))
	}
	helpText = append(helpText, common.FormatHelp("esc", "back"))
	content.WriteString(strings.Join(helpText, "  "))

	// Use full width with padding
	style := lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2)

	return style.Render(content.String())
}

func (m *PacketsModel) updateTable() {
	deviceWidth, _, locationWidth, payloadWidth := m.calculateColumnWidths()

	rows := make([]table.Row, len(m.packets))
	for i, p := range m.packets {
		location := formatRetrievedLocation(p.Location)
		rows[i] = table.Row{
			truncate(p.DeviceID(), deviceWidth),
			p.Timestamp().Format("2006-01-02 15:04:05"),
			truncate(location, locationWidth),
			truncate(p.Payload(), payloadWidth),
		}
	}
	m.table.SetRows(rows)
}

// updateColumnWidths updates table column widths based on screen width
func (m *PacketsModel) updateColumnWidths() {
	deviceWidth, timestampWidth, locationWidth, payloadWidth := m.calculateColumnWidths()

	columns := []table.Column{
		{Title: "Device ID", Width: deviceWidth},
		{Title: "Timestamp", Width: timestampWidth},
		{Title: "Location", Width: locationWidth},
		{Title: "Payload", Width: payloadWidth},
	}
	m.table.SetColumns(columns)
}

// calculateColumnWidths returns column widths based on screen width
func (m *PacketsModel) calculateColumnWidths() (deviceWidth, timestampWidth, locationWidth, payloadWidth int) {
	// Fixed width for timestamp
	timestampWidth = 20

	// Available width for other columns (account for padding/borders)
	availableWidth := m.width - timestampWidth - 10

	if availableWidth < 80 {
		// Minimum widths
		deviceWidth = 36
		locationWidth = 20
		payloadWidth = 24
	} else {
		// Device ID needs 36 chars for full UUID
		deviceWidth = 36
		remaining := availableWidth - deviceWidth
		// Split remaining between location and payload
		locationWidth = remaining * 35 / 100
		payloadWidth = remaining - locationWidth
	}

	return
}

func (m PacketsModel) loadPackets(append bool) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return PacketsErrorMsg{Err: fmt.Errorf("no API client")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		opts := api.RetrievePacketsOptions{
			Days: m.days,
		}
		if m.deviceID != "" {
			opts.DeviceID = &m.deviceID
		} else {
			// When no device filter, limit to 100 packets per request
			opts.Limit = 100
		}

		// If appending, use the continuation token
		if append && m.continuationToken != "" {
			opts.ContinuationToken = m.continuationToken
		}

		result, err := m.client.RetrievePacketsWithPagination(ctx, opts)
		if err != nil {
			return PacketsErrorMsg{Err: err}
		}

		return PacketsLoadedMsg{
			Packets:           result.Packets,
			ContinuationToken: result.ContinuationToken,
			Append:            append,
		}
	}
}

// SetDeviceFilter sets the device ID filter
func (m *PacketsModel) SetDeviceFilter(deviceID string) {
	m.deviceID = deviceID
}

// formatLocation formats a location for display
func formatLocation(loc models.Location) string {
	if loc.Fake {
		return "Local scan"
	}
	if loc.Latitude == 0 && loc.Longitude == 0 {
		return "Unknown"
	}
	return fmt.Sprintf("%.4f, %.4f", loc.Latitude, loc.Longitude)
}

// formatRetrievedLocation formats a retrieved packet location for display
func formatRetrievedLocation(loc models.RetrievedLocation) string {
	if loc.Latitude == 0 && loc.Longitude == 0 {
		return "Unknown"
	}
	return fmt.Sprintf("%.4f, %.4f", loc.Latitude, loc.Longitude)
}
