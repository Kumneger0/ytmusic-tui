package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	accentColor    = lipgloss.Color("#7D56F4")
	accentDim      = lipgloss.Color("#5A3DB5")
	textPrimary    = lipgloss.Color("#E4E4E7")
	textSecondary  = lipgloss.Color("#A1A1AA")
	textDim        = lipgloss.Color("#71717A")
	borderFocused  = lipgloss.Color("#7D56F4")
	borderNormal   = lipgloss.Color("#3F3F46")
	bgSelected     = lipgloss.Color("#7D56F4")
	fgSelected     = lipgloss.Color("#FAFAFA")
	progressFilled = lipgloss.Color("#7D56F4")
	progressEmpty  = lipgloss.Color("#3F3F46")
	playerFg       = lipgloss.Color("#E4E4E7")
)

var (
	normalStyle = lipgloss.NewStyle().
			Foreground(textPrimary)

	selectedStyle = lipgloss.NewStyle().
			Foreground(fgSelected).
			Background(bgSelected).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(textSecondary)

	dimmerStyle = lipgloss.NewStyle().
			Foreground(textDim)

	titleStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)
)

func getBorderColor(isFocused bool) lipgloss.Color {
	if isFocused {
		return borderFocused
	}
	return borderNormal
}

func getPlayerStyles(m *Model, dims layoutDimensions) lipgloss.Style {
	width := dims.mainWidth + (dims.sidebarWidth*2 + 2)
	inputStyle := getStyle(m, dims.inputHeight, width, Player, false)
	return inputStyle
}

func getStyle(m *Model, height, width int, focusedOn FocusedOn, isSearchResult bool) lipgloss.Style {
	if isSearchResult {
		return lipgloss.NewStyle().
			Width(width).
			Height(height-5).
			Padding(1, 0, 0, 0)
	}
	isFocused := m.FocusedOn == focusedOn
	border := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0).
		Border(border).
		BorderForeground(getBorderColor(isFocused))
	return style
}
