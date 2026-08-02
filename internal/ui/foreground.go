package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/ytmusic-tui/internal/types"
)

type FormField int

const (
	FieldTitle FormField = iota
	FieldDescription
	FieldPrivacy
	FieldSubmit
)

type ForegroundModel struct {
	ActiveModal    types.ModalType
	FocusIndex     FormField
	TitleInput     textinput.Model
	DescInput      textinput.Model
	PrivacyIndex   int
	PrivacyOptions []string
	Width          int
	Height         int
	ErrorMsg       string
	SuccessMsg     string
	IsSubmitting   bool
}

func NewForegroundModel() *ForegroundModel {
	ti := textinput.New()
	ti.Placeholder = "My Awesome Playlist"
	ti.Prompt = ""
	ti.CharLimit = 100
	ti.Width = 34
	ti.Focus()

	di := textinput.New()
	di.Placeholder = "Optional description..."
	di.Prompt = ""
	di.CharLimit = 200
	di.Width = 34

	return &ForegroundModel{
		ActiveModal:    types.ModalTypeCreatePlaylist,
		FocusIndex:     FieldTitle,
		TitleInput:     ti,
		DescInput:      di,
		PrivacyIndex:   0,
		PrivacyOptions: []string{"PRIVATE", "PUBLIC", "UNLISTED"},
	}
}

func (m *ForegroundModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *ForegroundModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case types.OpenModalMsg:
		m.ActiveModal = msg.ModalType
		if msg.ModalType == types.ModalTypeCreatePlaylist {
			m.TitleInput.Reset()
			m.TitleInput.Focus()
			m.DescInput.Reset()
			m.DescInput.Blur()
			m.FocusIndex = FieldTitle
			m.PrivacyIndex = 0
			m.ErrorMsg = ""
			m.IsSubmitting = false
			return m, textinput.Blink
		}
		return m, nil

	case types.CloseModalMsg:
		m.ActiveModal = types.ModalTypeCreatePlaylist
		m.TitleInput.Blur()
		m.DescInput.Blur()
		return m, nil
	case types.CreatePlaylistResponseMsg:
		if msg.Success {
			m.IsSubmitting = false
			m.ErrorMsg = ""
			m.TitleInput.Blur()
			m.TitleInput.Reset()
			m.DescInput.Reset()
			m.SuccessMsg = "Playlist created successfully! You can find it in your library."
			m.DescInput.Blur()
		} else {
			m.ErrorMsg = "Failed to create playlist. Please try again."
			m.IsSubmitting = false
			return m, nil
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			currentModal := m.ActiveModal
			m.ActiveModal = types.ModalTypeCreatePlaylist
			return m, func() tea.Msg {
				return types.CloseModalMsg{ModalType: currentModal}
			}

		case "tab", "down":
			m.FocusIndex = (m.FocusIndex + 1) % 4
			m.updateFocus()
			return m, nil

		case "shift+tab", "up":
			m.FocusIndex = (m.FocusIndex - 1 + 4) % 4
			m.updateFocus()
			return m, nil

		case "left", "h":
			if m.FocusIndex == FieldPrivacy {
				m.PrivacyIndex = (m.PrivacyIndex - 1 + len(m.PrivacyOptions)) % len(m.PrivacyOptions)
				return m, nil
			}

		case "right", "l", "space", " ":
			if m.FocusIndex == FieldPrivacy {
				m.PrivacyIndex = (m.PrivacyIndex + 1) % len(m.PrivacyOptions)
				return m, nil
			}

		case "enter":
			if m.FocusIndex == FieldSubmit || m.FocusIndex == FieldTitle || m.FocusIndex == FieldDescription {
				title := strings.TrimSpace(m.TitleInput.Value())
				if title == "" {
					m.ErrorMsg = "Playlist title is required"
					return m, nil
				}
				m.IsSubmitting = true
				m.ErrorMsg = ""
				desc := strings.TrimSpace(m.DescInput.Value())
				privacy := m.PrivacyOptions[m.PrivacyIndex]
				return m, func() tea.Msg {
					return types.CreatePlaylistMsg{
						Title:         title,
						Description:   desc,
						PrivacyStatus: privacy,
					}
				}
			}
		}

		var cmd tea.Cmd
		switch m.FocusIndex {
		case FieldTitle:
			m.TitleInput, cmd = m.TitleInput.Update(msg)
		case FieldDescription:
			m.DescInput, cmd = m.DescInput.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m *ForegroundModel) updateFocus() {
	switch m.FocusIndex {
	case FieldTitle:
		m.TitleInput.Focus()
		m.DescInput.Blur()
	case FieldDescription:
		m.TitleInput.Blur()
		m.DescInput.Focus()
	default:
		m.TitleInput.Blur()
		m.DescInput.Blur()
	}
}

func (m *ForegroundModel) View() string {
	if m.ActiveModal == types.ModalTypeCreatePlaylist {
		return m.renderCreatePlaylistModal()
	}

	return ""
}

func (m *ForegroundModel) renderCreatePlaylistModal() string {
	if m.IsSubmitting {
		return lipgloss.NewStyle().
			Width(54).
			Height(12).
			Align(lipgloss.Center).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			Background(lipgloss.Color("#18181B")).
			Render("Creating playlist...")
	}
	modalWidth := 54
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A1A1AA")).Width(14)
	focusedLabelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Width(14)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Bold(true)

	header := accentStyle.Render("➕ Create New Playlist")

	var titleLabel string
	if m.FocusIndex == FieldTitle {
		titleLabel = focusedLabelStyle.Render("Title ")
	} else {
		titleLabel = labelStyle.Render("Title *")
	}
	titleRow := fmt.Sprintf("%s %s", titleLabel, m.TitleInput.View())

	var descLabel string
	if m.FocusIndex == FieldDescription {
		descLabel = focusedLabelStyle.Render("Description")
	} else {
		descLabel = labelStyle.Render("Description")
	}
	descRow := fmt.Sprintf("%s %s", descLabel, m.DescInput.View())

	var privacyLabel string
	if m.FocusIndex == FieldPrivacy {
		privacyLabel = focusedLabelStyle.Render("Privacy")
	} else {
		privacyLabel = labelStyle.Render("Privacy")
	}

	var privacyPills []string
	for i, opt := range m.PrivacyOptions {
		if i == m.PrivacyIndex {
			if m.FocusIndex == FieldPrivacy {
				pill := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FAFAFA")).
					Background(lipgloss.Color("#7D56F4")).
					Bold(true).
					Padding(0, 1).
					Render("● " + opt)
				privacyPills = append(privacyPills, pill)
			} else {
				pill := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#7D56F4")).
					Bold(true).
					Padding(0, 1).
					Render("● " + opt)
				privacyPills = append(privacyPills, pill)
			}
		} else {
			pill := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#71717A")).
				Padding(0, 1).
				Render("○ " + opt)
			privacyPills = append(privacyPills, pill)
		}
	}
	privacyRow := fmt.Sprintf("%s %s", privacyLabel, strings.Join(privacyPills, " "))

	var submitBtn string
	if m.FocusIndex == FieldSubmit {
		submitBtn = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Bold(true).
			Padding(0, 3).
			Render("✓ Create Playlist")
	} else {
		submitBtn = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E4E4E7")).
			Background(lipgloss.Color("#3F3F46")).
			Padding(0, 3).
			Render("  Create Playlist")
	}
	submitContainer := lipgloss.NewStyle().Align(lipgloss.Center).Width(modalWidth - 4).Render(submitBtn)

	errContent := ""
	if m.ErrorMsg != "" {
		errContent = errorStyle.Render("⚠ " + m.ErrorMsg)
	}

	successContent := ""
	if m.SuccessMsg != "" {
		successContent = successStyle.Render("✓ " + m.SuccessMsg)
	}

	divider := dimStyle.Render(strings.Repeat("─", modalWidth-4))
	keybindHelp := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A1A1AA")).
		Align(lipgloss.Center).
		Width(modalWidth - 4).
		Render("Tab/↑↓ Focus • Space Privacy • Enter Submit • Esc Cancel")

	contentParts := []string{
		header,
		divider,
		titleRow,
		descRow,
		privacyRow,
	}
	if errContent != "" {
		contentParts = append(contentParts, errContent)
	}
	if successContent != "" {
		contentParts = append(contentParts, successContent)
	}
	contentParts = append(contentParts, submitContainer, divider, keybindHelp)

	body := strings.Join(contentParts, "\n\n")

	modalBox := lipgloss.NewStyle().
		Width(modalWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		Background(lipgloss.Color("#18181B")).
		Render(body)

	return modalBox
}
