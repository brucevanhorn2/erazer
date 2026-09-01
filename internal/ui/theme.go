package ui

import "github.com/charmbracelet/lipgloss"

const (
	primaryColorHex   = "#B341F5"
	secondaryColorHex = "#6E6E6E"
)

// Theme holds erazer's static cyberpunk styling — matching exfil's and
// sneakernet's house colors. There's no Settings screen to change these at
// runtime; that's out of scope here.
type Theme struct {
	PrimaryColor        lipgloss.Color
	SecondaryColor      lipgloss.Color
	MutedPrimaryColor   lipgloss.Color
	MutedSecondaryColor lipgloss.Color

	BrowserDir      lipgloss.Style
	BrowserFile     lipgloss.Style
	BrowserSelected lipgloss.Style

	Header lipgloss.Style // section headers, e.g. "Confirm erasure"

	EraseTrigger        lipgloss.Style // the ERAZE button, unfocused
	EraseTriggerFocused lipgloss.Style // the ERAZE button, focused — the "dramatically red" state

	StatusBar   lipgloss.Style
	StatusKey   lipgloss.Style
	StatusValue lipgloss.Style
	StatusError lipgloss.Style
}

func NewTheme() Theme {
	primary := lipgloss.Color(primaryColorHex)
	secondary := lipgloss.Color(secondaryColorHex)
	return Theme{
		PrimaryColor:        primary,
		SecondaryColor:      secondary,
		MutedPrimaryColor:   mutedColor(primary),
		MutedSecondaryColor: mutedColor(secondary),

		BrowserDir: lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Bold(true),
		BrowserFile: lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")),
		BrowserSelected: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true),

		Header: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true),

		EraseTrigger: lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true),
		EraseTriggerFocused: lipgloss.NewStyle().
			Background(lipgloss.Color("1")).
			Foreground(lipgloss.Color("15")).
			Bold(true).
			Padding(0, 2),

		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")),
		StatusKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Bold(true),
		StatusValue: lipgloss.NewStyle().
			Foreground(primary),
		StatusError: lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")),
	}
}
