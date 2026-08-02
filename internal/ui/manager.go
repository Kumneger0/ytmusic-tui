package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/ytmusic-tui/internal/types"
)

type SessionState int

const (
	Foreground SessionState = iota
	Background
)

type OverlayMode string

const (
	Search OverlayMode = "SEARCH"
)

type Manager struct {
	State        SessionState
	WindowWidth  int
	WindowHeight int
	Foreground   tea.Model
	Background   tea.Model
	Overlay      tea.Model
	OverlayMode  OverlayMode
}

func (m Manager) Init() tea.Cmd {
	return tea.Batch(
		m.Foreground.Init(),
		m.Background.Init(),
	)
}

func (m Manager) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case types.CreatePlaylistMsg:
		var cmd tea.Cmd
		m.Background, cmd = m.Background.Update(msg)
		return m, cmd
	case tea.WindowSizeMsg:
		m.WindowWidth = msg.Width
		m.WindowHeight = msg.Height
		var fgCmd, bgCmd tea.Cmd
		m.Foreground, fgCmd = m.Foreground.Update(msg)
		m.Background, bgCmd = m.Background.Update(msg)
		return m, tea.Batch(fgCmd, bgCmd)

	case types.OpenModalMsg:
		m.State = Foreground
		var fgCmd, bgCmd tea.Cmd
		m.Foreground, fgCmd = m.Foreground.Update(msg)
		m.Background, bgCmd = m.Background.Update(msg)
		return m, tea.Batch(fgCmd, bgCmd)

	case types.CloseModalMsg:
		m.State = Background
		var fgCmd, bgCmd tea.Cmd
		m.Foreground, fgCmd = m.Foreground.Update(msg)
		m.Background, bgCmd = m.Background.Update(msg)
		return m, tea.Batch(fgCmd, bgCmd)
	}

	if m.State == Foreground {
		var cmd tea.Cmd
		m.Foreground, cmd = m.Foreground.Update(message)
		return m, cmd
	}

	var cmd tea.Cmd
	m.Background, cmd = m.Background.Update(message)
	return m, cmd
}

func (m Manager) View() string {
	if m.State == Foreground {
		return m.Overlay.View()
	}
	return m.Background.View()
}
