package screens

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hubblenetwork/hubcli/internal/api"
	"github.com/hubblenetwork/hubcli/internal/auth"
	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/hubblenetwork/hubcli/internal/tui/common"
)

// LoginState represents the current state of the login screen
type LoginState int

const (
	LoginStateInput LoginState = iota
	LoginStateValidating
	LoginStateSuccess
	LoginStateError
)

// Login messages
type (
	// LoginSuccessMsg is sent when credentials are validated successfully
	LoginSuccessMsg struct {
		Credentials models.Credentials
		OrgName     string
	}

	// LoginErrorMsg is sent when validation fails
	LoginErrorMsg struct {
		Err error
	}

	// ValidateCredentialsMsg triggers credential validation
	ValidateCredentialsMsg struct {
		Credentials models.Credentials
	}
)

// LoginModel is the model for the login screen
type LoginModel struct {
	orgIDInput textinput.Model
	tokenInput textinput.Model
	spinner    spinner.Model
	help       help.Model
	keys       common.LoginKeyMap

	focusIndex int
	state      LoginState
	err        error
	orgName    string

	width  int
	height int
}

// NewLoginModel creates a new login screen model
func NewLoginModel() LoginModel {
	// Organization ID input
	orgID := textinput.New()
	orgID.Placeholder = "your-organization-id"
	orgID.CharLimit = 64
	orgID.Width = 50
	orgID.Focus()

	// API Token input
	token := textinput.New()
	token.Placeholder = "your-api-token"
	token.CharLimit = 256
	token.Width = 50
	token.EchoMode = textinput.EchoPassword
	token.EchoCharacter = '•'

	// Spinner for validation
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(common.ColorPrimary)

	return LoginModel{
		orgIDInput: orgID,
		tokenInput: token,
		spinner:    sp,
		help:       help.New(),
		keys:       common.DefaultLoginKeyMap(),
		focusIndex: 0,
		state:      LoginStateInput,
	}
}

// Init initializes the login model
func (m LoginModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the login screen
func (m LoginModel) Update(msg tea.Msg) (LoginModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case tea.KeyMsg:
		// Don't handle keys during validation
		if m.state == LoginStateValidating {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Tab):
			m.focusIndex = (m.focusIndex + 1) % 3 // 2 inputs + submit button
			m.updateFocus()
			return m, nil

		case key.Matches(msg, m.keys.ShiftTab):
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 2
			}
			m.updateFocus()
			return m, nil

		case key.Matches(msg, m.keys.Submit):
			if m.focusIndex == 2 || m.canSubmit() {
				return m.submit()
			}
			// If on input field, move to next
			m.focusIndex = (m.focusIndex + 1) % 3
			m.updateFocus()
			return m, nil
		}

	case LoginSuccessMsg:
		m.state = LoginStateSuccess
		m.orgName = msg.OrgName
		return m, nil

	case LoginErrorMsg:
		m.state = LoginStateError
		m.err = msg.Err
		return m, nil

	case spinner.TickMsg:
		if m.state == LoginStateValidating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Update focused input
	if m.state == LoginStateInput {
		var cmd tea.Cmd
		if m.focusIndex == 0 {
			m.orgIDInput, cmd = m.orgIDInput.Update(msg)
			cmds = append(cmds, cmd)
		} else if m.focusIndex == 1 {
			m.tokenInput, cmd = m.tokenInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the login screen
func (m LoginModel) View() string {
	var content strings.Builder

	// Logo
	content.WriteString(common.Logo())
	content.WriteString("\n")

	// Title
	content.WriteString(common.TitleStyle.Render("Welcome to Hubble CLI"))
	content.WriteString("\n")
	content.WriteString(common.SubtitleStyle.Render("Enter your credentials to continue"))
	content.WriteString("\n\n")

	switch m.state {
	case LoginStateInput, LoginStateError:
		content.WriteString(m.renderForm())

	case LoginStateValidating:
		content.WriteString(m.renderValidating())

	case LoginStateSuccess:
		content.WriteString(m.renderSuccess())
	}

	// Help
	content.WriteString("\n\n")
	content.WriteString(m.help.ShortHelpView(m.keys.ShortHelp()))

	// Center the content
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content.String(),
	)
}

func (m LoginModel) renderForm() string {
	var b strings.Builder

	// Organization ID field
	orgIDLabel := "Organization ID"
	if m.focusIndex == 0 {
		orgIDLabel = common.SelectedStyle.Render(orgIDLabel)
	} else {
		orgIDLabel = common.UnselectedStyle.Render(orgIDLabel)
	}
	b.WriteString(orgIDLabel)
	b.WriteString("\n")

	inputStyle := common.InputStyle
	if m.focusIndex == 0 {
		inputStyle = common.FocusedInputStyle
	}
	b.WriteString(inputStyle.Render(m.orgIDInput.View()))
	b.WriteString("\n\n")

	// Token field
	tokenLabel := "API Token"
	if m.focusIndex == 1 {
		tokenLabel = common.SelectedStyle.Render(tokenLabel)
	} else {
		tokenLabel = common.UnselectedStyle.Render(tokenLabel)
	}
	b.WriteString(tokenLabel)
	b.WriteString("\n")

	inputStyle = common.InputStyle
	if m.focusIndex == 1 {
		inputStyle = common.FocusedInputStyle
	}
	b.WriteString(inputStyle.Render(m.tokenInput.View()))
	b.WriteString("\n\n")

	// Submit button
	buttonText := "  Login  "
	if m.focusIndex == 2 {
		b.WriteString(common.ButtonStyle.Render(buttonText))
	} else if m.canSubmit() {
		b.WriteString(common.ButtonStyle.Copy().Background(common.ColorBorder).Render(buttonText))
	} else {
		b.WriteString(common.DisabledButtonStyle.Render(buttonText))
	}

	// Error message
	if m.state == LoginStateError && m.err != nil {
		b.WriteString("\n\n")
		b.WriteString(common.ErrorTextStyle.Render("Error: " + m.err.Error()))
	}

	return b.String()
}

func (m LoginModel) renderValidating() string {
	return fmt.Sprintf("%s Validating credentials...", m.spinner.View())
}

func (m LoginModel) renderSuccess() string {
	var b strings.Builder
	b.WriteString(common.SuccessTextStyle.Render("✓ Login successful!"))
	b.WriteString("\n\n")
	if m.orgName != "" {
		b.WriteString(fmt.Sprintf("Organization: %s", m.orgName))
	}
	b.WriteString("\n\n")
	b.WriteString(common.MutedTextStyle.Render("Credentials saved to keychain."))
	return b.String()
}

func (m *LoginModel) updateFocus() {
	m.orgIDInput.Blur()
	m.tokenInput.Blur()

	switch m.focusIndex {
	case 0:
		m.orgIDInput.Focus()
	case 1:
		m.tokenInput.Focus()
	}
}

func (m LoginModel) canSubmit() bool {
	return strings.TrimSpace(m.orgIDInput.Value()) != "" &&
		strings.TrimSpace(m.tokenInput.Value()) != ""
}

func (m LoginModel) submit() (LoginModel, tea.Cmd) {
	if !m.canSubmit() {
		return m, nil
	}

	m.state = LoginStateValidating
	m.err = nil

	creds := models.Credentials{
		OrgID: strings.TrimSpace(m.orgIDInput.Value()),
		Token: strings.TrimSpace(m.tokenInput.Value()),
	}

	return m, tea.Batch(
		m.spinner.Tick,
		validateCredentials(creds),
	)
}

// validateCredentials returns a command that validates the credentials
func validateCredentials(creds models.Credentials) tea.Cmd {
	return func() tea.Msg {
		client := api.NewClientFromCredentials(creds)
		ctx := context.Background()

		// Validate credentials by fetching the organization
		// If this succeeds, the credentials are valid
		org, err := client.GetOrganization(ctx)
		if err != nil {
			return LoginErrorMsg{Err: fmt.Errorf("invalid credentials: %w", err)}
		}

		orgName := ""
		if org != nil {
			orgName = org.Name
		}

		// Save to keychain
		if err := auth.SaveCredentials(&creds); err != nil {
			return LoginErrorMsg{Err: fmt.Errorf("failed to save credentials: %w", err)}
		}

		return LoginSuccessMsg{
			Credentials: creds,
			OrgName:     orgName,
		}
	}
}

// GetCredentials returns the entered credentials
func (m LoginModel) GetCredentials() models.Credentials {
	return models.Credentials{
		OrgID: strings.TrimSpace(m.orgIDInput.Value()),
		Token: strings.TrimSpace(m.tokenInput.Value()),
	}
}

// IsSuccess returns true if login was successful
func (m LoginModel) IsSuccess() bool {
	return m.state == LoginStateSuccess
}

// GetOrgName returns the organization name after successful login
func (m LoginModel) GetOrgName() string {
	return m.orgName
}
