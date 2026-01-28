package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hubblenetwork/hubcli/internal/auth"
	"github.com/hubblenetwork/hubcli/internal/tui/common"
)

// SettingsState represents the current state of the settings screen
type SettingsState int

const (
	SettingsStateReady SettingsState = iota
	SettingsStateConfirmClear
	SettingsStateClearing
	SettingsStateSuccess
	SettingsStateError
)

// Settings messages
type (
	// CredentialsClearedMsg is sent when credentials are cleared
	CredentialsClearedMsg struct {
		Error error
	}
)

// SettingsModel is the model for the settings screen
type SettingsModel struct {
	help   help.Model
	keys   settingsKeyMap
	store  *auth.KeychainStore

	state          SettingsState
	err            error
	hasKeychain    bool
	hasEnvVars     bool
	keychainOrgID  string
	envOrgID       string
	width          int
	height         int
}

// settingsKeyMap defines key bindings for the settings screen
type settingsKeyMap struct {
	Clear   key.Binding
	Confirm key.Binding
	Cancel  key.Binding
	Back    key.Binding
	Quit    key.Binding
}

func defaultSettingsKeyMap() settingsKeyMap {
	return settingsKeyMap{
		Clear: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clear keychain"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "cancel"),
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

// NewSettingsModel creates a new settings screen model
func NewSettingsModel() SettingsModel {
	m := SettingsModel{
		help:  help.New(),
		keys:  defaultSettingsKeyMap(),
		store: auth.NewKeychainStore(),
		state: SettingsStateReady,
	}

	// Check credential sources
	m.checkCredentials()

	return m
}

// Init initializes the settings model
func (m SettingsModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the settings screen
func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case SettingsStateConfirmClear:
			switch {
			case key.Matches(msg, m.keys.Confirm):
				m.state = SettingsStateClearing
				return m, m.clearCredentials()
			case key.Matches(msg, m.keys.Cancel):
				m.state = SettingsStateReady
				return m, nil
			}

		case SettingsStateSuccess, SettingsStateError:
			// Any key returns to ready state
			m.state = SettingsStateReady
			m.checkCredentials()
			return m, nil

		default:
			switch {
			case key.Matches(msg, m.keys.Back):
				return m, func() tea.Msg {
					return NavigateMsg{Screen: "home"}
				}

			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit

			case key.Matches(msg, m.keys.Clear):
				if m.hasKeychain {
					m.state = SettingsStateConfirmClear
					return m, nil
				}
			}
		}

	case CredentialsClearedMsg:
		if msg.Error != nil {
			m.state = SettingsStateError
			m.err = msg.Error
		} else {
			m.state = SettingsStateSuccess
		}
		m.checkCredentials()
		return m, nil
	}

	return m, nil
}

// View renders the settings screen
func (m SettingsModel) View() string {
	var content strings.Builder

	// Header
	content.WriteString(common.TitleStyle.Render("Settings"))
	content.WriteString("\n")
	content.WriteString(common.SubtitleStyle.Render("Manage credentials and preferences"))
	content.WriteString("\n\n")

	boxStyle := common.BoxStyle.Copy().Width(60)

	// Credential status section
	content.WriteString(boxStyle.Render(m.renderCredentialStatus()))
	content.WriteString("\n\n")

	// Environment variables section
	content.WriteString(boxStyle.Render(m.renderEnvVarInfo()))
	content.WriteString("\n\n")

	// State-specific content
	switch m.state {
	case SettingsStateConfirmClear:
		confirmBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(common.ColorWarning).
			Padding(1, 2).
			Width(60)

		confirmContent := common.WarningTextStyle.Render("Clear stored credentials?") + "\n\n" +
			common.MutedTextStyle.Render("This will remove credentials from the keychain.") + "\n" +
			common.MutedTextStyle.Render("You will need to log in again.") + "\n\n" +
			common.FormatHelp("y", "confirm") + "  " + common.FormatHelp("n", "cancel")

		content.WriteString(confirmBox.Render(confirmContent))

	case SettingsStateClearing:
		content.WriteString(common.MutedTextStyle.Render("Clearing credentials..."))

	case SettingsStateSuccess:
		content.WriteString(common.SuccessTextStyle.Render("Credentials cleared successfully!"))
		content.WriteString("\n")
		content.WriteString(common.MutedTextStyle.Render("Press any key to continue."))

	case SettingsStateError:
		content.WriteString(common.ErrorTextStyle.Render("Error: " + m.err.Error()))
		content.WriteString("\n")
		content.WriteString(common.MutedTextStyle.Render("Press any key to continue."))

	default:
		// Help
		content.WriteString(m.renderHelp())
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content.String(),
	)
}

func (m SettingsModel) renderCredentialStatus() string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(common.ColorSecondary)
	labelStyle := lipgloss.NewStyle().Foreground(common.ColorMuted).Width(20)
	valueStyle := lipgloss.NewStyle().Foreground(common.ColorForeground)

	b.WriteString(headerStyle.Render("Credential Status"))
	b.WriteString("\n\n")

	// Keychain status
	b.WriteString(labelStyle.Render("Keychain:"))
	if m.hasKeychain {
		b.WriteString(common.SuccessTextStyle.Render("Stored"))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("  Org ID:"))
		b.WriteString(valueStyle.Render(maskString(m.keychainOrgID)))
	} else {
		b.WriteString(common.MutedTextStyle.Render("Not stored"))
	}
	b.WriteString("\n")

	// Environment variable status
	b.WriteString(labelStyle.Render("Environment:"))
	if m.hasEnvVars {
		b.WriteString(common.SuccessTextStyle.Render("Set"))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("  Org ID:"))
		b.WriteString(valueStyle.Render(maskString(m.envOrgID)))
	} else {
		b.WriteString(common.MutedTextStyle.Render("Not set"))
	}
	b.WriteString("\n\n")

	// Active source
	b.WriteString(labelStyle.Render("Active Source:"))
	if m.hasEnvVars {
		b.WriteString(common.PrimaryTextStyle.Render("Environment variables"))
	} else if m.hasKeychain {
		b.WriteString(common.PrimaryTextStyle.Render("Keychain"))
	} else {
		b.WriteString(common.ErrorTextStyle.Render("None"))
	}

	return b.String()
}

func (m SettingsModel) renderEnvVarInfo() string {
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(common.ColorSecondary)
	codeStyle := lipgloss.NewStyle().
		Foreground(common.ColorForeground).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1)

	b.WriteString(headerStyle.Render("Environment Variables"))
	b.WriteString("\n\n")
	b.WriteString(common.MutedTextStyle.Render("Set these to use environment-based credentials:"))
	b.WriteString("\n\n")
	b.WriteString(codeStyle.Render("export HUBBLE_ORG_ID=\"your-org-id\""))
	b.WriteString("\n")
	b.WriteString(codeStyle.Render("export HUBBLE_API_TOKEN=\"your-api-token\""))
	b.WriteString("\n\n")
	b.WriteString(common.MutedTextStyle.Render("Environment variables take priority over keychain."))

	return b.String()
}

func (m SettingsModel) renderHelp() string {
	var helpText []string

	if m.hasKeychain {
		helpText = append(helpText, common.FormatHelp("c", "clear keychain"))
	}
	helpText = append(helpText, common.FormatHelp("esc", "back"))

	return strings.Join(helpText, "  ")
}

func (m *SettingsModel) checkCredentials() {
	// Check keychain
	if m.store != nil && m.store.Exists() {
		creds, err := m.store.Get()
		if err == nil && creds != nil {
			m.hasKeychain = true
			m.keychainOrgID = creds.OrgID
		} else {
			m.hasKeychain = false
			m.keychainOrgID = ""
		}
	}

	// Check environment variables
	envCreds := auth.GetCredentialsFromEnv()
	if envCreds != nil && envCreds.IsValid() {
		m.hasEnvVars = true
		m.envOrgID = envCreds.OrgID
	} else {
		m.hasEnvVars = false
		m.envOrgID = ""
	}
}

func (m SettingsModel) clearCredentials() tea.Cmd {
	return func() tea.Msg {
		if m.store == nil {
			return CredentialsClearedMsg{Error: fmt.Errorf("keychain not available")}
		}

		err := m.store.Delete()
		return CredentialsClearedMsg{Error: err}
	}
}

// maskString masks a string, showing only first and last 2 characters
func maskString(s string) string {
	if len(s) <= 6 {
		return "****"
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}
