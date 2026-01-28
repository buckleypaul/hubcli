package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hubblenetwork/hubcli/internal/tui/common"
)

// MenuItem represents a menu option on the home screen
type MenuItem struct {
	Title       string
	Description string
	Icon        string
	Screen      string // Screen identifier to navigate to
}

// HomeModel is the model for the home/menu screen
type HomeModel struct {
	items    []MenuItem
	cursor   int
	keys     common.MenuKeyMap
	help     help.Model
	orgName  string
	showHelp bool
	width    int
	height   int
}

// NewHomeModel creates a new home screen model
func NewHomeModel(orgName string) HomeModel {
	items := []MenuItem{
		{
			Title:       "Devices",
			Description: "View and manage registered devices",
			Icon:        "📱",
			Screen:      "devices",
		},
		{
			Title:       "Packets",
			Description: "View packet history from your devices",
			Icon:        "📦",
			Screen:      "packets",
		},
		{
			Title:       "BLE Scan",
			Description: "Scan for local BLE advertisements",
			Icon:        "📡",
			Screen:      "ble_scan",
		},
		{
			Title:       "Organization",
			Description: "View organization information",
			Icon:        "🏢",
			Screen:      "org_info",
		},
		{
			Title:       "Settings",
			Description: "Manage credentials and preferences",
			Icon:        "⚙️",
			Screen:      "settings",
		},
	}

	return HomeModel{
		items:   items,
		cursor:  0,
		keys:    common.DefaultMenuKeyMap(),
		help:    help.New(),
		orgName: orgName,
	}
}

// Init initializes the home model
func (m HomeModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the home screen
func (m HomeModel) Update(msg tea.Msg) (HomeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.items) - 1
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			m.cursor++
			if m.cursor >= len(m.items) {
				m.cursor = 0
			}
			return m, nil

		case key.Matches(msg, m.keys.Select):
			// Return a navigation command
			return m, m.navigateToSelected()

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the home screen
func (m HomeModel) View() string {
	var content strings.Builder

	// Header
	header := m.renderHeader()
	content.WriteString(header)
	content.WriteString("\n\n")

	// Menu items
	menu := m.renderMenu()
	content.WriteString(menu)
	content.WriteString("\n\n")

	// Help
	if m.showHelp {
		content.WriteString(m.help.FullHelpView(m.keys.FullHelp()))
	} else {
		content.WriteString(m.help.ShortHelpView(m.keys.ShortHelp()))
	}

	// Center the content
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content.String(),
	)
}

func (m HomeModel) renderHeader() string {
	var b strings.Builder

	// Title
	title := common.TitleStyle.Copy().MarginBottom(0).Render("Hubble CLI")
	b.WriteString(title)

	// Organization name
	if m.orgName != "" {
		b.WriteString("\n")
		orgText := fmt.Sprintf("Organization: %s", m.orgName)
		b.WriteString(common.MutedTextStyle.Render(orgText))
	}

	return b.String()
}

func (m HomeModel) renderMenu() string {
	var b strings.Builder

	menuWidth := 50

	for i, item := range m.items {
		isSelected := i == m.cursor

		// Build menu item
		var itemContent strings.Builder

		// Icon and title
		titleLine := fmt.Sprintf("%s  %s", item.Icon, item.Title)

		// Description on second line
		descLine := item.Description

		if isSelected {
			// Selected style
			titleStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(common.ColorPrimary)

			descStyle := lipgloss.NewStyle().
				Foreground(common.ColorMuted).
				PaddingLeft(4)

			itemContent.WriteString(titleStyle.Render("▸ " + titleLine))
			itemContent.WriteString("\n")
			itemContent.WriteString(descStyle.Render(descLine))
		} else {
			// Unselected style
			titleStyle := lipgloss.NewStyle().
				Foreground(common.ColorForeground)

			itemContent.WriteString(titleStyle.Render("  " + titleLine))
		}

		// Box around the item if selected
		itemStr := itemContent.String()
		if isSelected {
			itemStr = common.FocusedBoxStyle.Copy().
				Width(menuWidth).
				Render(itemStr)
		} else {
			itemStr = lipgloss.NewStyle().
				Width(menuWidth).
				Padding(0, 2).
				Render(itemStr)
		}

		b.WriteString(itemStr)
		if i < len(m.items)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m HomeModel) navigateToSelected() tea.Cmd {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		screen := m.items[m.cursor].Screen
		return func() tea.Msg {
			return NavigateMsg{Screen: screen}
		}
	}
	return nil
}

// NavigateMsg is sent when navigating to a new screen
type NavigateMsg struct {
	Screen string
	Data   interface{} // Optional data to pass to the target screen
}

// SelectedItem returns the currently selected menu item
func (m HomeModel) SelectedItem() MenuItem {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return m.items[m.cursor]
	}
	return MenuItem{}
}

// SetOrgName sets the organization name
func (m *HomeModel) SetOrgName(name string) {
	m.orgName = name
}
