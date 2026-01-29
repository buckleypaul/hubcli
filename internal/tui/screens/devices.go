package screens

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hubblenetwork/hubcli/internal/api"
	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/hubblenetwork/hubcli/internal/tui/common"
)

// DevicesState represents the current state of the devices screen
type DevicesState int

const (
	DevicesStateLoading DevicesState = iota
	DevicesStateReady
	DevicesStateError
	DevicesStateRegistering
	DevicesStateDeleteConfirm
	DevicesStateDeleting
)

// SortColumn represents which column to sort by
type SortColumn int

const (
	SortByID SortColumn = iota
	SortByName
	SortByCreated
	SortByLastPacket
)

func (s SortColumn) String() string {
	switch s {
	case SortByID:
		return "ID"
	case SortByName:
		return "Name"
	case SortByCreated:
		return "Created"
	case SortByLastPacket:
		return "Last Packet"
	default:
		return "ID"
	}
}

// Device screen messages
type (
	// DevicesLoadedMsg is sent when devices are fetched
	DevicesLoadedMsg struct {
		Devices []models.Device
	}

	// DevicesErrorMsg is sent when fetching fails
	DevicesErrorMsg struct {
		Err error
	}

	// DeviceRegisteredMsg is sent when a device is registered
	DeviceRegisteredMsg struct {
		Device *models.Device
	}

	// DeviceDeletedMsg is sent when a device is deleted
	DeviceDeletedMsg struct {
		DeviceID string
	}
)

// DevicesModel is the model for the devices screen
type DevicesModel struct {
	client  *api.Client
	devices []models.Device
	table   table.Model
	spinner spinner.Model
	help    help.Model
	keys    common.ListKeyMap

	state        DevicesState
	err          error
	showRegister bool
	width        int
	height       int

	// Filtering
	filterInput   textinput.Model
	filterActive  bool
	filterText    string
	filteredDevs  []models.Device

	// Sorting
	sortColumn     SortColumn // Column currently being sorted
	sortAsc        bool
	selectedColumn SortColumn // Column selected for potential sorting (with brackets)

	// Delete confirmation
	deleteInput       textinput.Model
	deleteDevice      *models.Device // Device being deleted
	deleteConfirmText string         // Text user must type to confirm (first 4 chars of UUID)
}

// NewDevicesModel creates a new devices screen model
func NewDevicesModel(client *api.Client) DevicesModel {
	columns := []table.Column{
		{Title: "ID", Width: 20},
		{Title: "Name", Width: 24},
		{Title: "Created", Width: 18},
		{Title: "Last Packet", Width: 18},
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

	// Initialize filter input
	fi := textinput.New()
	fi.Placeholder = "Filter by name or ID..."
	fi.CharLimit = 64
	fi.Width = 40
	fi.PromptStyle = lipgloss.NewStyle().Foreground(common.ColorSecondary)
	fi.TextStyle = lipgloss.NewStyle().Foreground(common.ColorForeground)

	// Initialize delete confirmation input
	di := textinput.New()
	di.Placeholder = "xxxx"
	di.CharLimit = 4
	di.Width = 10
	di.PromptStyle = lipgloss.NewStyle().Foreground(common.ColorSecondary)
	di.TextStyle = lipgloss.NewStyle().Foreground(common.ColorForeground)

	return DevicesModel{
		client:         client,
		table:          t,
		spinner:        sp,
		help:           help.New(),
		keys:           common.DefaultListKeyMap(),
		state:          DevicesStateLoading,
		filterInput:    fi,
		deleteInput:    di,
		sortColumn:     SortByLastPacket,
		sortAsc:        false, // Default: most recent first
		selectedColumn: SortByLastPacket,
	}
}

// Init initializes the devices model
func (m DevicesModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadDevices(),
	)
}

// Update handles messages for the devices screen
func (m DevicesModel) Update(msg tea.Msg) (DevicesModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		// Adjust table height (account for filter input when active)
		tableHeight := m.height - 15
		if m.filterActive {
			tableHeight -= 2
		}
		m.table.SetHeight(tableHeight)
		// Update column widths to fill screen
		m.updateColumnHeaders()
		return m, nil

	case tea.KeyMsg:
		// Handle delete confirmation mode
		if m.state == DevicesStateDeleteConfirm {
			switch msg.String() {
			case "esc":
				m.state = DevicesStateReady
				m.deleteInput.Blur()
				m.deleteInput.SetValue("")
				m.deleteDevice = nil
				m.table.Focus()
				return m, nil
			case "enter":
				// Check if input matches first 4 characters of device UUID
				if strings.EqualFold(m.deleteInput.Value(), m.deleteConfirmText) {
					m.state = DevicesStateDeleting
					m.deleteInput.Blur()
					deviceID := m.deleteDevice.ID
					m.deleteDevice = nil
					m.deleteInput.SetValue("")
					return m, tea.Batch(m.spinner.Tick, m.deleteDeviceCmd(deviceID))
				}
				// Wrong input - stay in confirmation mode
				return m, nil
			default:
				var cmd tea.Cmd
				m.deleteInput, cmd = m.deleteInput.Update(msg)
				return m, cmd
			}
		}

		// Handle filter input mode
		if m.filterActive {
			switch msg.String() {
			case "esc":
				m.filterActive = false
				m.filterInput.Blur()
				m.table.Focus()
				return m, nil
			case "enter":
				m.filterActive = false
				m.filterInput.Blur()
				m.table.Focus()
				m.filterText = m.filterInput.Value()
				m.applyFilterAndSort()
				return m, nil
			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				// Apply filter as user types
				m.filterText = m.filterInput.Value()
				m.applyFilterAndSort()
				return m, cmd
			}
		}

		switch {
		case key.Matches(msg, m.keys.Back):
			// If filter has text, clear it first
			if m.filterText != "" {
				m.filterText = ""
				m.filterInput.SetValue("")
				m.applyFilterAndSort()
				return m, nil
			}
			return m, func() tea.Msg {
				return NavigateMsg{Screen: "home"}
			}

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Refresh):
			if m.state == DevicesStateReady || m.state == DevicesStateError {
				m.state = DevicesStateLoading
				return m, tea.Batch(m.spinner.Tick, m.loadDevices())
			}

		case key.Matches(msg, m.keys.Search):
			// Toggle filter input
			if m.state == DevicesStateReady {
				m.filterActive = true
				m.filterInput.Focus()
				return m, textinput.Blink
			}

		case key.Matches(msg, m.keys.Select):
			if m.state == DevicesStateReady && len(m.filteredDevs) > 0 {
				// Get the full device to access the complete ID
				device := m.SelectedDevice()
				if device != nil {
					// Navigate to packets for this device
					return m, func() tea.Msg {
						return NavigateMsg{Screen: "packets", Data: device.ID}
					}
				}
			}

		case msg.String() == "n":
			// Register new device
			if m.state == DevicesStateReady && !m.filterActive {
				m.state = DevicesStateRegistering
				return m, tea.Batch(m.spinner.Tick, m.registerDevice())
			}

		case msg.String() == "d":
			// Delete device - initiate confirmation
			if m.state == DevicesStateReady && !m.filterActive && len(m.filteredDevs) > 0 {
				device := m.SelectedDevice()
				if device != nil {
					m.state = DevicesStateDeleteConfirm
					m.deleteDevice = device
					m.deleteConfirmText = device.ID[:4]
					m.deleteInput.SetValue("")
					m.deleteInput.Focus()
					return m, textinput.Blink
				}
			}

		// Select sort column with left/right arrows
		case key.Matches(msg, m.keys.Left):
			if m.state == DevicesStateReady {
				if m.selectedColumn > 0 {
					m.selectedColumn--
				} else {
					m.selectedColumn = SortByLastPacket // Wrap to last column
				}
				m.updateColumnHeaders()
				return m, nil
			}
		case key.Matches(msg, m.keys.Right):
			if m.state == DevicesStateReady {
				m.selectedColumn = (m.selectedColumn + 1) % 4
				m.updateColumnHeaders()
				return m, nil
			}

		// Sort by selected column with 's'
		case msg.String() == "s":
			if m.state == DevicesStateReady {
				m.toggleSort(m.selectedColumn)
				return m, nil
			}
		}

	case DevicesLoadedMsg:
		m.state = DevicesStateReady
		m.devices = msg.Devices
		m.applyFilterAndSort()
		return m, nil

	case DevicesErrorMsg:
		m.state = DevicesStateError
		m.err = msg.Err
		return m, nil

	case DeviceRegisteredMsg:
		m.state = DevicesStateLoading
		return m, tea.Batch(m.spinner.Tick, m.loadDevices())

	case DeviceDeletedMsg:
		m.state = DevicesStateLoading
		return m, tea.Batch(m.spinner.Tick, m.loadDevices())

	case spinner.TickMsg:
		if m.state == DevicesStateLoading || m.state == DevicesStateRegistering || m.state == DevicesStateDeleting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Update table
	if m.state == DevicesStateReady && !m.filterActive {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// toggleSort sets sort column and toggles direction if same column
func (m *DevicesModel) toggleSort(col SortColumn) {
	if m.sortColumn == col {
		m.sortAsc = !m.sortAsc
	} else {
		m.sortColumn = col
		// Default to ascending for name/ID, descending for dates
		m.sortAsc = col != SortByCreated && col != SortByLastPacket
	}
	m.applyFilterAndSort()
}

// applyFilterAndSort filters and sorts devices, then updates the table
func (m *DevicesModel) applyFilterAndSort() {
	// Filter
	m.filteredDevs = m.filterDevices()

	// Sort
	m.sortDevices()

	// Update column headers with sort indicator
	m.updateColumnHeaders()

	// Update table
	m.updateTableFromFiltered()
}

// updateColumnHeaders updates column titles to show sort indicator and selection brackets
func (m *DevicesModel) updateColumnHeaders() {
	sortIndicator := " ↓"
	if m.sortAsc {
		sortIndicator = " ↑"
	}

	titles := []string{"ID", "Name", "Created", "Last Packet"}

	// Add sort indicator to sorted column
	if m.sortColumn >= 0 && int(m.sortColumn) < len(titles) {
		titles[m.sortColumn] += sortIndicator
	}

	// Add brackets around selected column
	if m.selectedColumn >= 0 && int(m.selectedColumn) < len(titles) {
		titles[m.selectedColumn] = "[" + titles[m.selectedColumn] + "]"
	}

	// Calculate dynamic column widths
	idWidth, nameWidth, createdWidth, lastPacketWidth := m.calculateColumnWidths()

	columns := []table.Column{
		{Title: titles[0], Width: idWidth},
		{Title: titles[1], Width: nameWidth},
		{Title: titles[2], Width: createdWidth},
		{Title: titles[3], Width: lastPacketWidth},
	}
	m.table.SetColumns(columns)
}

// calculateColumnWidths returns column widths based on screen width
func (m *DevicesModel) calculateColumnWidths() (idWidth, nameWidth, createdWidth, lastPacketWidth int) {
	// Fixed widths for date columns
	createdWidth = 18
	lastPacketWidth = 18

	// Available width for ID and Name (account for padding/borders)
	availableWidth := m.width - createdWidth - lastPacketWidth - 10

	if availableWidth < 60 {
		// Minimum widths
		idWidth = 36
		nameWidth = 24
	} else {
		// Give 60% to ID (UUIDs are 36 chars), 40% to Name
		idWidth = availableWidth * 60 / 100
		nameWidth = availableWidth - idWidth
		// Ensure ID can fit a full UUID
		if idWidth < 36 {
			idWidth = 36
			nameWidth = availableWidth - idWidth
		}
	}

	return
}

// filterDevices returns devices matching the filter text
func (m *DevicesModel) filterDevices() []models.Device {
	if m.filterText == "" {
		// Return a copy to avoid modifying original
		result := make([]models.Device, len(m.devices))
		copy(result, m.devices)
		return result
	}

	filter := strings.ToLower(m.filterText)
	var result []models.Device
	for _, d := range m.devices {
		// Match against ID or Name
		if strings.Contains(strings.ToLower(d.ID), filter) ||
			strings.Contains(strings.ToLower(d.Name), filter) {
			result = append(result, d)
		}
	}
	return result
}

// sortDevices sorts the filtered devices in place
func (m *DevicesModel) sortDevices() {
	sort.SliceStable(m.filteredDevs, func(i, j int) bool {
		var less bool
		switch m.sortColumn {
		case SortByID:
			less = m.filteredDevs[i].ID < m.filteredDevs[j].ID
		case SortByName:
			less = strings.ToLower(m.filteredDevs[i].Name) < strings.ToLower(m.filteredDevs[j].Name)
		case SortByCreated:
			less = m.filteredDevs[i].CreatedTS < m.filteredDevs[j].CreatedTS
		case SortByLastPacket:
			// Handle nil values - devices with no packets sort last
			iVal := float64(0)
			jVal := float64(0)
			if m.filteredDevs[i].MostRecentPacket != nil && m.filteredDevs[i].MostRecentPacket.Terrestrial != nil {
				iVal = m.filteredDevs[i].MostRecentPacket.Terrestrial.Timestamp
			}
			if m.filteredDevs[j].MostRecentPacket != nil && m.filteredDevs[j].MostRecentPacket.Terrestrial != nil {
				jVal = m.filteredDevs[j].MostRecentPacket.Terrestrial.Timestamp
			}
			less = iVal < jVal
		}
		if m.sortAsc {
			return less
		}
		return !less
	})
}

// View renders the devices screen
func (m DevicesModel) View() string {
	var content strings.Builder

	// Header
	content.WriteString(common.TitleStyle.Render("Devices"))
	content.WriteString("\n")
	content.WriteString(common.SubtitleStyle.Render("Manage your registered devices"))
	content.WriteString("\n\n")

	switch m.state {
	case DevicesStateLoading:
		content.WriteString(fmt.Sprintf("%s Loading devices...", m.spinner.View()))

	case DevicesStateRegistering:
		content.WriteString(fmt.Sprintf("%s Registering new device...", m.spinner.View()))

	case DevicesStateDeleting:
		content.WriteString(fmt.Sprintf("%s Deleting device...", m.spinner.View()))

	case DevicesStateDeleteConfirm:
		// Show confirmation prompt
		deviceName := m.deleteDevice.Name
		if deviceName == "" {
			deviceName = "(unnamed)"
		}
		content.WriteString(common.ErrorTextStyle.Render("⚠ Delete Device"))
		content.WriteString("\n\n")
		content.WriteString(fmt.Sprintf("Device: %s\n", deviceName))
		content.WriteString(fmt.Sprintf("ID: %s\n\n", m.deleteDevice.ID))
		content.WriteString("Type the first 4 characters of the device ID to confirm deletion:\n\n")
		content.WriteString(fmt.Sprintf("  %s ", m.deleteInput.View()))
		if m.deleteInput.Value() != "" && !strings.EqualFold(m.deleteInput.Value(), m.deleteConfirmText) && len(m.deleteInput.Value()) == 4 {
			content.WriteString(common.ErrorTextStyle.Render(" ✗ Does not match"))
		}

	case DevicesStateError:
		content.WriteString(common.ErrorTextStyle.Render("Error: " + m.err.Error()))
		content.WriteString("\n\n")
		content.WriteString(common.MutedTextStyle.Render("Press 'r' to retry"))

	case DevicesStateReady:
		if len(m.devices) == 0 {
			content.WriteString(common.MutedTextStyle.Render("No devices found."))
			content.WriteString("\n\n")
			content.WriteString(common.MutedTextStyle.Render("Press 'n' to register a new device."))
		} else {
			// Filter input
			if m.filterActive {
				content.WriteString(common.PrimaryTextStyle.Render("Filter: "))
				content.WriteString(m.filterInput.View())
				content.WriteString("\n\n")
			} else if m.filterText != "" {
				content.WriteString(common.MutedTextStyle.Render(fmt.Sprintf("Filter: %q", m.filterText)))
				content.WriteString("\n\n")
			}

			// Device count
			countText := fmt.Sprintf("%d of %d device(s)", len(m.filteredDevs), len(m.devices))
			content.WriteString(common.MutedTextStyle.Render(countText))
			content.WriteString("\n\n")

			// Table
			content.WriteString(m.table.View())
		}
	}

	// Help
	content.WriteString("\n\n")
	var helpText []string
	if m.state == DevicesStateDeleteConfirm {
		helpText = []string{
			common.FormatHelp("enter", "confirm delete"),
			common.FormatHelp("esc", "cancel"),
		}
	} else if m.filterActive {
		helpText = []string{
			common.FormatHelp("enter", "apply"),
			common.FormatHelp("esc", "cancel"),
		}
	} else {
		helpText = []string{
			common.FormatHelp("↑/↓", "navigate"),
			common.FormatHelp("←/→", "select column"),
			common.FormatHelp("s", "sort"),
			common.FormatHelp("enter", "view packets"),
			common.FormatHelp("/", "filter"),
			common.FormatHelp("n", "new"),
			common.FormatHelp("d", "delete"),
			common.FormatHelp("r", "refresh"),
			common.FormatHelp("esc", "back"),
		}
	}
	content.WriteString(strings.Join(helpText, "  "))

	// Use full width with padding
	style := lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2)

	return style.Render(content.String())
}

func (m *DevicesModel) updateTable() {
	m.applyFilterAndSort()
}

func (m *DevicesModel) updateTableFromFiltered() {
	idWidth, nameWidth, _, _ := m.calculateColumnWidths()

	rows := make([]table.Row, len(m.filteredDevs))
	for i, d := range m.filteredDevs {
		name := d.Name
		if name == "" {
			name = "-"
		}
		// Convert unix timestamp (seconds) to formatted date
		created := "-"
		if d.CreatedTS > 0 {
			created = time.Unix(d.CreatedTS, 0).Format("2006-01-02 15:04")
		}
		lastPacket := "-"
		if d.MostRecentPacket != nil && d.MostRecentPacket.Terrestrial != nil && d.MostRecentPacket.Terrestrial.Timestamp > 0 {
			ts := int64(d.MostRecentPacket.Terrestrial.Timestamp)
			lastPacket = time.Unix(ts, 0).Format("2006-01-02 15:04")
		}
		rows[i] = table.Row{
			truncate(d.ID, idWidth),
			truncate(name, nameWidth),
			created,
			lastPacket,
		}
	}
	m.table.SetRows(rows)
}

func (m DevicesModel) loadDevices() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return DevicesErrorMsg{Err: fmt.Errorf("no API client")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		devices, err := m.client.ListDevices(ctx)
		if err != nil {
			return DevicesErrorMsg{Err: err}
		}

		return DevicesLoadedMsg{Devices: devices}
	}
}

func (m DevicesModel) registerDevice() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return DevicesErrorMsg{Err: fmt.Errorf("no API client")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		device, err := m.client.RegisterDevice(ctx, models.RegisterDeviceRequest{
			Encryption: models.EncryptionAES256CTR,
		})
		if err != nil {
			return DevicesErrorMsg{Err: err}
		}

		return DeviceRegisteredMsg{Device: device}
	}
}

func (m DevicesModel) deleteDeviceCmd(deviceID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return DevicesErrorMsg{Err: fmt.Errorf("no API client")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := m.client.DeleteDevice(ctx, deviceID)
		if err != nil {
			return DevicesErrorMsg{Err: err}
		}

		return DeviceDeletedMsg{DeviceID: deviceID}
	}
}

// SelectedDevice returns the currently selected device, if any
func (m DevicesModel) SelectedDevice() *models.Device {
	if m.state != DevicesStateReady || len(m.devices) == 0 || len(m.filteredDevs) == 0 {
		return nil
	}

	selectedRow := m.table.SelectedRow()
	if len(selectedRow) == 0 {
		return nil
	}

	// Get the displayed ID, removing "..." suffix if truncated
	displayedID := selectedRow[0]
	if strings.HasSuffix(displayedID, "...") {
		displayedID = displayedID[:len(displayedID)-3]
	}

	// Find device by ID prefix match in filtered list
	for i := range m.filteredDevs {
		if strings.HasPrefix(m.filteredDevs[i].ID, displayedID) {
			return &m.filteredDevs[i]
		}
	}

	return nil
}

// truncate shortens a string to maxLen, adding "..." if needed
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
