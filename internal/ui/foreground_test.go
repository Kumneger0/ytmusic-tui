package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kumneger0/ytmusic-tui/internal/types"
)

func TestForegroundModel_CreatePlaylistModal(t *testing.T) {
	fg := NewForegroundModel()

	model, _ := fg.Update(types.OpenModalMsg{ModalType: types.ModalTypeCreatePlaylist})
	fg = model.(*ForegroundModel)

	if fg.ActiveModal != types.ModalTypeCreatePlaylist {
		t.Fatalf("expected ActiveModal to be ModalTypeCreatePlaylist, got %v", fg.ActiveModal)
	}
	if fg.FocusIndex != FieldTitle {
		t.Fatalf("expected initial FocusIndex to be FieldTitle, got %v", fg.FocusIndex)
	}

	view := fg.View()
	if view == "" {
		t.Fatalf("expected non-empty view when modal is active")
	}
	if !testing.Short() {
		t.Logf("Modal View:\n%s", view)
	}

	model, _ = fg.Update(tea.KeyMsg{Type: tea.KeyTab})
	fg = model.(*ForegroundModel)
	if fg.FocusIndex != FieldDescription {
		t.Fatalf("expected FocusIndex to be FieldDescription after Tab, got %v", fg.FocusIndex)
	}

	model, _ = fg.Update(tea.KeyMsg{Type: tea.KeyTab})
	fg = model.(*ForegroundModel)
	if fg.FocusIndex != FieldPrivacy {
		t.Fatalf("expected FocusIndex to be FieldPrivacy after second Tab, got %v", fg.FocusIndex)
	}

	model, _ = fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	fg = model.(*ForegroundModel)
	if fg.PrivacyOptions[fg.PrivacyIndex] != "PUBLIC" {
		t.Fatalf("expected Privacy status to be PUBLIC after Space, got %s", fg.PrivacyOptions[fg.PrivacyIndex])
	}

	model, cmd := fg.Update(tea.KeyMsg{Type: tea.KeyEsc})
	fg = model.(*ForegroundModel)
	if fg.ActiveModal != types.ModalTypeCreatePlaylist {
		t.Fatalf("expected ActiveModal to be ModalTypeCreatePlaylist after Esc, got %v", fg.ActiveModal)
	}
	if cmd == nil {
		t.Fatalf("expected CloseModalMsg command on Esc")
	}
}
