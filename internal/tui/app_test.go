package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hubblenetwork/hubcli/internal/models"
	"github.com/hubblenetwork/hubcli/internal/tui/screens"
	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	// NewApp checks for credentials in env/keychain
	// Without credentials, it should start at login screen
	app := NewApp()

	// Depending on env, it may be login or home
	// Just verify app is created
	assert.NotNil(t, app)
}

func TestApp_Init(t *testing.T) {
	app := NewApp()
	cmd := app.Init()

	// Init should return some command(s)
	// May be nil or a batch depending on screen
	_ = cmd
}

func TestApp_WindowSizeMsg(t *testing.T) {
	app := NewApp()

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	updatedApp := model.(*App)

	assert.Equal(t, 100, updatedApp.width)
	assert.Equal(t, 50, updatedApp.height)
	assert.True(t, updatedApp.ready)
}

func TestApp_View_NotReady(t *testing.T) {
	app := NewApp()
	app.ready = false

	view := app.View()
	assert.Equal(t, "Loading...", view)
}

func TestApp_HandleNavigation(t *testing.T) {
	app := NewApp()
	app.screen = ScreenHome
	app.ready = true
	app.width = 80
	app.height = 24

	tests := []struct {
		screen   string
		expected Screen
	}{
		{"devices", ScreenDevices},
		{"packets", ScreenPackets},
		{"ble_scan", ScreenBLEScan},
		{"org_info", ScreenOrgInfo},
		{"settings", ScreenSettings},
		{"home", ScreenHome},
	}

	for _, tt := range tests {
		model, _ := app.handleNavigation(tt.screen, nil)
		updatedApp := model.(*App)
		assert.Equal(t, tt.expected, updatedApp.screen, "navigation to %s", tt.screen)
	}
}

func TestApp_HandleNavigation_Back(t *testing.T) {
	app := NewApp()
	// Simulate navigating from Home to Devices
	app.screen = ScreenHome
	app.handleNavigation("devices", nil) // This sets prevScreen = ScreenHome

	// Now navigate back
	model, _ := app.handleNavigation("back", nil)
	updatedApp := model.(*App)

	assert.Equal(t, ScreenHome, updatedApp.screen)
}

func TestApp_LoginSuccessMsg(t *testing.T) {
	app := NewApp()
	app.screen = ScreenLogin
	app.ready = true
	app.width = 80
	app.height = 24

	msg := screens.LoginSuccessMsg{
		Credentials: models.Credentials{
			OrgID: "test-org",
			Token: "test-token",
		},
		OrgName: "Test Organization",
	}

	model, _ := app.Update(msg)
	updatedApp := model.(*App)

	assert.Equal(t, ScreenHome, updatedApp.screen)
	assert.Equal(t, "Test Organization", updatedApp.orgName)
	assert.NotNil(t, updatedApp.credentials)
}

func TestApp_NavigateMsg(t *testing.T) {
	app := NewApp()
	app.screen = ScreenHome
	app.ready = true
	app.width = 80
	app.height = 24

	msg := screens.NavigateMsg{Screen: "devices"}

	model, _ := app.Update(msg)
	updatedApp := model.(*App)

	assert.Equal(t, ScreenDevices, updatedApp.screen)
}

func TestApp_OrgNameMsg(t *testing.T) {
	app := NewApp()
	app.screen = ScreenHome
	app.homeModel = screens.NewHomeModel("")

	msg := orgNameMsg{Name: "Fetched Org Name"}

	model, _ := app.Update(msg)
	updatedApp := model.(*App)

	assert.Equal(t, "Fetched Org Name", updatedApp.orgName)
}

func TestApp_RenderPlaceholder(t *testing.T) {
	app := NewApp()
	app.width = 80
	app.height = 24

	content := app.renderPlaceholder("Test Title", "Test description")

	assert.Contains(t, content, "Test Title")
	assert.Contains(t, content, "Test description")
	assert.Contains(t, content, "Phase 4")
}

func TestScreenConstants(t *testing.T) {
	// Verify screen constants are distinct
	screens := []Screen{
		ScreenLogin,
		ScreenHome,
		ScreenDevices,
		ScreenPackets,
		ScreenBLEScan,
		ScreenOrgInfo,
		ScreenSettings,
	}

	seen := make(map[Screen]bool)
	for _, s := range screens {
		assert.False(t, seen[s], "duplicate screen constant")
		seen[s] = true
	}
}
