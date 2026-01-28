package common

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#7B68EE") // Medium slate blue
	ColorSecondary = lipgloss.Color("#00CED1") // Dark turquoise
	ColorAccent    = lipgloss.Color("#9370DB") // Medium purple

	// Status colors
	ColorSuccess = lipgloss.Color("#32CD32") // Lime green
	ColorWarning = lipgloss.Color("#FFD700") // Gold
	ColorError   = lipgloss.Color("#FF6347") // Tomato

	// Neutral colors
	ColorSubtle     = lipgloss.Color("#666666") // Gray
	ColorMuted      = lipgloss.Color("#888888") // Light gray
	ColorBorder     = lipgloss.Color("#444444") // Dark gray
	ColorBackground = lipgloss.Color("#1a1a1a") // Near black
	ColorForeground = lipgloss.Color("#FFFFFF") // White
)

// Base styles
var (
	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginBottom(1)

	// Text styles
	TextStyle = lipgloss.NewStyle().
			Foreground(ColorForeground)

	MutedTextStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	ErrorTextStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	SuccessTextStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	WarningTextStyle = lipgloss.NewStyle().
				Foreground(ColorWarning)

	PrimaryTextStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary)

	// Selection styles
	SelectedStyle = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(ColorForeground).
			Bold(true).
			Padding(0, 1)

	UnselectedStyle = lipgloss.NewStyle().
			Foreground(ColorForeground).
			Padding(0, 1)

	// Input styles
	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	FocusedInputStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	// Button styles
	ButtonStyle = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(ColorForeground).
			Padding(0, 2).
			MarginTop(1)

	DisabledButtonStyle = lipgloss.NewStyle().
				Background(ColorBorder).
				Foreground(ColorMuted).
				Padding(0, 2).
				MarginTop(1)

	// Container styles
	BoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	FocusedBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	// Status bar style
	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(ColorForeground).
			Padding(0, 1)

	// Help styles
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	HelpSepStyle = lipgloss.NewStyle().
			Foreground(ColorBorder)

	// Menu item styles
	MenuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	MenuItemSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(ColorPrimary).
				Bold(true)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(ColorBorder)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	TableSelectedRowStyle = lipgloss.NewStyle().
				Background(ColorPrimary).
				Foreground(ColorForeground)
)

// Logo returns the Hubble CLI ASCII art logo
func Logo() string {
	logo := `
 _   _       _     _     _
| | | |_   _| |__ | |__ | | ___
| |_| | | | | '_ \| '_ \| |/ _ \
|  _  | |_| | |_) | |_) | |  __/
|_| |_|\__,_|_.__/|_.__/|_|\___|
`
	return lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render(logo)
}

// Spinner characters for loading states
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Centered returns a style that centers content in the given dimensions
func Centered(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center)
}

// FormatHelp formats a help line with key and description
func FormatHelp(key, desc string) string {
	return HelpKeyStyle.Render(key) +
		HelpSepStyle.Render(" ") +
		HelpDescStyle.Render(desc)
}
