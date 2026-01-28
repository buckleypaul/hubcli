package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hubblenetwork/hubcli/internal/api"
	"github.com/hubblenetwork/hubcli/internal/auth"
	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/hubblenetwork/hubcli/internal/tui/common"
	"github.com/hubblenetwork/hubcli/internal/tui/screens"
)

// Screen represents the current screen in the TUI.
type Screen int

const (
	ScreenLogin Screen = iota
	ScreenHome
	ScreenDevices
	ScreenPackets
	ScreenBLEScan
	ScreenOrgInfo
	ScreenSettings
)

// App is the main application model.
type App struct {
	screen      Screen
	prevScreen  Screen
	width       int
	height      int
	ready       bool
	err         error
	credentials *models.Credentials
	orgName     string
	client      *api.Client

	// Screen models
	loginModel    screens.LoginModel
	homeModel     screens.HomeModel
	devicesModel  screens.DevicesModel
	packetsModel  screens.PacketsModel
	orgInfoModel  screens.OrgInfoModel
	bleScanModel  screens.BLEScanModel
	settingsModel screens.SettingsModel
}

// NewApp creates a new application instance.
func NewApp() *App {
	app := &App{
		screen:     ScreenLogin,
		loginModel: screens.NewLoginModel(),
	}

	// Check for existing credentials
	creds, err := auth.GetCredentials()
	if err == nil && creds != nil && creds.IsValid() {
		app.credentials = creds
		app.client = api.NewClientFromCredentials(*creds)
		app.screen = ScreenHome
		app.homeModel = screens.NewHomeModel("")
	}

	return app
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Initialize the current screen
	switch a.screen {
	case ScreenLogin:
		cmds = append(cmds, a.loginModel.Init())
	case ScreenHome:
		cmds = append(cmds, a.homeModel.Init())
		// Fetch org name in background
		if a.credentials != nil {
			cmds = append(cmds, a.fetchOrgName())
		}
	}

	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		// Forward to current screen
		return a, a.forwardToCurrentScreen(msg)

	case screens.LoginSuccessMsg:
		// Login was successful, switch to home screen
		a.credentials = &msg.Credentials
		a.client = api.NewClientFromCredentials(msg.Credentials)
		a.orgName = msg.OrgName
		a.homeModel = screens.NewHomeModel(msg.OrgName)
		a.screen = ScreenHome
		// Forward window size to new screen
		return a, a.forwardToCurrentScreen(tea.WindowSizeMsg{
			Width:  a.width,
			Height: a.height,
		})

	case screens.NavigateMsg:
		return a.handleNavigation(msg.Screen, msg.Data)

	case orgNameMsg:
		a.orgName = msg.Name
		a.homeModel.SetOrgName(msg.Name)
		return a, nil
	}

	// Forward message to current screen
	cmd := a.forwardToCurrentScreen(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

// View implements tea.Model.
func (a *App) View() string {
	if !a.ready {
		return "Loading..."
	}

	var content string

	switch a.screen {
	case ScreenLogin:
		content = a.loginModel.View()
	case ScreenHome:
		content = a.homeModel.View()
	case ScreenDevices:
		content = a.devicesModel.View()
	case ScreenPackets:
		content = a.packetsModel.View()
	case ScreenBLEScan:
		content = a.bleScanModel.View()
	case ScreenOrgInfo:
		content = a.orgInfoModel.View()
	case ScreenSettings:
		content = a.settingsModel.View()
	default:
		content = "Unknown screen"
	}

	return content
}

func (a *App) forwardToCurrentScreen(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch a.screen {
	case ScreenLogin:
		a.loginModel, cmd = a.loginModel.Update(msg)
	case ScreenHome:
		a.homeModel, cmd = a.homeModel.Update(msg)
	case ScreenDevices:
		a.devicesModel, cmd = a.devicesModel.Update(msg)
	case ScreenPackets:
		a.packetsModel, cmd = a.packetsModel.Update(msg)
	case ScreenOrgInfo:
		a.orgInfoModel, cmd = a.orgInfoModel.Update(msg)
	case ScreenBLEScan:
		a.bleScanModel, cmd = a.bleScanModel.Update(msg)
	case ScreenSettings:
		a.settingsModel, cmd = a.settingsModel.Update(msg)
	}

	return cmd
}

func (a *App) handleNavigation(screen string, data interface{}) (tea.Model, tea.Cmd) {
	// Handle "back" separately to avoid overwriting prevScreen
	if screen == "back" {
		a.screen = a.prevScreen
		return a, a.forwardToCurrentScreen(tea.WindowSizeMsg{
			Width:  a.width,
			Height: a.height,
		})
	}

	// Save current screen before navigating
	a.prevScreen = a.screen

	var initCmd tea.Cmd

	switch screen {
	case "devices":
		a.screen = ScreenDevices
		a.devicesModel = screens.NewDevicesModel(a.client)
		initCmd = a.devicesModel.Init()
	case "packets":
		deviceID := ""
		if data != nil {
			if id, ok := data.(string); ok {
				deviceID = id
			}
		}
		a.screen = ScreenPackets
		a.packetsModel = screens.NewPacketsModel(a.client, deviceID)
		initCmd = a.packetsModel.Init()
	case "ble_scan":
		a.screen = ScreenBLEScan
		a.bleScanModel = screens.NewBLEScanModel(a.client)
		initCmd = a.bleScanModel.Init()
	case "org_info":
		a.screen = ScreenOrgInfo
		a.orgInfoModel = screens.NewOrgInfoModel(a.client)
		initCmd = a.orgInfoModel.Init()
	case "settings":
		a.screen = ScreenSettings
		a.settingsModel = screens.NewSettingsModel()
		initCmd = a.settingsModel.Init()
	case "home":
		a.screen = ScreenHome
	}

	// Forward window size to new screen
	sizeCmd := a.forwardToCurrentScreen(tea.WindowSizeMsg{
		Width:  a.width,
		Height: a.height,
	})

	if initCmd != nil {
		return a, tea.Batch(initCmd, sizeCmd)
	}
	return a, sizeCmd
}

func (a *App) renderPlaceholder(title, description string) string {
	content := common.TitleStyle.Render(title) + "\n\n" +
		common.SubtitleStyle.Render(description) + "\n\n" +
		common.MutedTextStyle.Render("This screen will be implemented in Phase 4.") + "\n\n" +
		common.FormatHelp("esc", "back") + "  " + common.FormatHelp("q", "quit")

	return lipgloss.Place(
		a.width,
		a.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// orgNameMsg is sent when the org name is fetched
type orgNameMsg struct {
	Name string
}

func (a *App) fetchOrgName() tea.Cmd {
	return func() tea.Msg {
		if a.credentials == nil {
			return nil
		}

		client := api.NewClientFromCredentials(*a.credentials)
		org, err := client.GetOrganization(context.Background())
		if err != nil {
			return nil
		}

		return orgNameMsg{Name: org.Name}
	}
}
