package screens

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hubblenetwork/hubcli/internal/api"
	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/hubblenetwork/hubcli/internal/tui/common"
)

// OrgInfoState represents the current state of the org info screen
type OrgInfoState int

const (
	OrgInfoStateLoading OrgInfoState = iota
	OrgInfoStateReady
	OrgInfoStateError
	OrgInfoStateCheckingCreds
)

// Org info messages
type (
	// OrgInfoLoadedMsg is sent when org info is fetched
	OrgInfoLoadedMsg struct {
		Org         *models.Organization
		DeviceCount int
	}

	// OrgInfoErrorMsg is sent when fetching fails
	OrgInfoErrorMsg struct {
		Err error
	}

	// CredsValidMsg is sent when credentials are validated
	CredsValidMsg struct {
		Valid bool
		Err   error
	}
)

// OrgInfoModel is the model for the organization info screen
type OrgInfoModel struct {
	client      *api.Client
	org         *models.Organization
	deviceCount int
	credsValid  *bool
	spinner     spinner.Model
	help        help.Model
	keys        common.ListKeyMap

	state  OrgInfoState
	err    error
	width  int
	height int
}

// NewOrgInfoModel creates a new org info screen model
func NewOrgInfoModel(client *api.Client) OrgInfoModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(common.ColorPrimary)

	return OrgInfoModel{
		client:  client,
		spinner: sp,
		help:    help.New(),
		keys:    common.DefaultListKeyMap(),
		state:   OrgInfoStateLoading,
	}
}

// Init initializes the org info model
func (m OrgInfoModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadOrgInfo(),
	)
}

// Update handles messages for the org info screen
func (m OrgInfoModel) Update(msg tea.Msg) (OrgInfoModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			return m, func() tea.Msg {
				return NavigateMsg{Screen: "home"}
			}

		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Refresh):
			if m.state == OrgInfoStateReady || m.state == OrgInfoStateError {
				m.state = OrgInfoStateLoading
				m.credsValid = nil
				return m, tea.Batch(m.spinner.Tick, m.loadOrgInfo())
			}
		}

	case OrgInfoLoadedMsg:
		m.state = OrgInfoStateReady
		m.org = msg.Org
		m.deviceCount = msg.DeviceCount
		// If we successfully loaded org info, credentials are valid
		valid := true
		m.credsValid = &valid
		return m, nil

	case OrgInfoErrorMsg:
		m.state = OrgInfoStateError
		m.err = msg.Err
		// If we got an error, credentials may be invalid
		valid := false
		m.credsValid = &valid
		return m, nil

	case CredsValidMsg:
		m.state = OrgInfoStateReady
		if msg.Err != nil {
			valid := false
			m.credsValid = &valid
		} else {
			m.credsValid = &msg.Valid
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == OrgInfoStateLoading || m.state == OrgInfoStateCheckingCreds {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View renders the org info screen
func (m OrgInfoModel) View() string {
	var content strings.Builder

	// Header
	content.WriteString(common.TitleStyle.Render("Organization"))
	content.WriteString("\n")
	content.WriteString(common.SubtitleStyle.Render("View organization information"))
	content.WriteString("\n\n")

	switch m.state {
	case OrgInfoStateLoading:
		content.WriteString(fmt.Sprintf("%s Loading organization info...", m.spinner.View()))

	case OrgInfoStateCheckingCreds:
		content.WriteString(fmt.Sprintf("%s Validating credentials...", m.spinner.View()))

	case OrgInfoStateError:
		content.WriteString(common.ErrorTextStyle.Render("Error: " + m.err.Error()))
		content.WriteString("\n\n")
		content.WriteString(common.MutedTextStyle.Render("Press 'r' to retry"))

	case OrgInfoStateReady:
		content.WriteString(m.renderInfo())
	}

	// Help
	content.WriteString("\n\n")
	helpText := []string{
		common.FormatHelp("r", "refresh"),
		common.FormatHelp("esc", "back"),
	}
	content.WriteString(strings.Join(helpText, "  "))

	// Use full width with padding
	style := lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2)

	return style.Render(content.String())
}

func (m OrgInfoModel) renderInfo() string {
	var b strings.Builder

	// Use full width minus outer padding (2*2=4) and some buffer
	boxWidth := m.width - 8
	if boxWidth < 40 {
		boxWidth = 40
	}

	boxStyle := common.BoxStyle.Copy().Width(boxWidth)

	// Organization details
	b.WriteString(boxStyle.Render(m.renderOrgDetails()))
	b.WriteString("\n\n")

	// Credential status
	b.WriteString(boxStyle.Render(m.renderCredStatus()))

	return b.String()
}

func (m OrgInfoModel) renderOrgDetails() string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(common.ColorSecondary)
	labelStyle := lipgloss.NewStyle().Foreground(common.ColorMuted).Width(12)
	valueStyle := lipgloss.NewStyle().Foreground(common.ColorForeground)

	b.WriteString(headerStyle.Render("Organization Details"))
	b.WriteString("\n\n")

	// Org ID
	b.WriteString(labelStyle.Render("Org ID:"))
	if m.org != nil {
		b.WriteString(valueStyle.Render(m.org.ID))
	} else if m.client != nil {
		b.WriteString(valueStyle.Render(m.client.OrgID()))
	} else {
		b.WriteString(common.MutedTextStyle.Render("Unknown"))
	}
	b.WriteString("\n")

	// Org Name
	b.WriteString(labelStyle.Render("Name:"))
	if m.org != nil && m.org.Name != "" {
		b.WriteString(valueStyle.Render(m.org.Name))
	} else {
		b.WriteString(common.MutedTextStyle.Render("Not set"))
	}
	b.WriteString("\n")

	// Device Count
	b.WriteString(labelStyle.Render("Devices:"))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d", m.deviceCount)))

	return b.String()
}

func (m OrgInfoModel) renderCredStatus() string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(common.ColorSecondary)
	labelStyle := lipgloss.NewStyle().Foreground(common.ColorMuted).Width(15)

	b.WriteString(headerStyle.Render("Credential Status"))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("API Status:"))
	if m.credsValid == nil {
		b.WriteString(common.MutedTextStyle.Render("Checking..."))
	} else if *m.credsValid {
		b.WriteString(common.SuccessTextStyle.Render("Valid"))
	} else {
		b.WriteString(common.ErrorTextStyle.Render("Invalid"))
	}

	return b.String()
}

func (m OrgInfoModel) loadOrgInfo() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return OrgInfoErrorMsg{Err: fmt.Errorf("no API client")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get org info
		org, err := m.client.GetOrganization(ctx)
		if err != nil {
			return OrgInfoErrorMsg{Err: err}
		}

		// Get device count
		devices, err := m.client.ListDevices(ctx)
		deviceCount := 0
		if err == nil {
			deviceCount = len(devices)
		}

		return OrgInfoLoadedMsg{
			Org:         org,
			DeviceCount: deviceCount,
		}
	}
}

func (m OrgInfoModel) validateCredentials() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return CredsValidMsg{Valid: false, Err: fmt.Errorf("no API client")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := m.client.CheckCredentials(ctx)
		if err != nil {
			return CredsValidMsg{Valid: false, Err: err}
		}

		return CredsValidMsg{Valid: true}
	}
}
