package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	musicpb "github.com/kumneger0/ytmusic-tui/gen"
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
	if fg.ActiveModal != types.ModalTypeNone {
		t.Fatalf("expected ActiveModal to be ModalTypeNone after Esc, got %v", fg.ActiveModal)
	}
	if cmd == nil {
		t.Fatalf("expected CloseModalMsg command on Esc")
	}
}

func TestForegroundModel_AddToPlaylistModal(t *testing.T) {
	fg := NewForegroundModel()

	model, _ := fg.Update(types.OpenAddToPlaylistLoadingMsg{
		TrackID:    "track123",
		TrackTitle: "Blinding Lights",
	})
	fg = model.(*ForegroundModel)

	if fg.ActiveModal != types.ModalTypeAddToPlaylist || !fg.IsLoading {
		t.Fatalf("expected ModalTypeAddToPlaylist with IsLoading=true")
	}

	loadingView := fg.View()
	if loadingView == "" {
		t.Fatalf("expected non-empty loading view")
	}

	pls := []*musicpb.Playlist{
		{PlaylistId: "PL1", Title: "Chill Vibes", Count: 10},
		{PlaylistId: "PL2", Title: "Workout Hits", Count: 25},
	}

	model, _ = fg.Update(types.OpenAddToPlaylistModalMsg{
		TrackID:    "track123",
		TrackTitle: "Blinding Lights",
		Playlists:  pls,
	})
	fg = model.(*ForegroundModel)

	if fg.IsLoading {
		t.Fatalf("expected IsLoading=false after playlists arrive")
	}

	model, _ = fg.Update(tea.KeyMsg{Type: tea.KeyDown})
	fg = model.(*ForegroundModel)
	if fg.PlaylistSelectIndex != 1 {
		t.Fatalf("expected PlaylistSelectIndex to be 1 after Down, got %d", fg.PlaylistSelectIndex)
	}

	model, cmd := fg.Update(tea.KeyMsg{Type: tea.KeyEnter})
	fg = model.(*ForegroundModel)
	if !fg.IsSubmitting {
		t.Fatalf("expected IsSubmitting to be true on Enter")
	}
	if cmd == nil {
		t.Fatalf("expected command on Enter")
	}

	model, _ = fg.Update(types.AddToPlaylistResponseMsg{Success: true})
	fg = model.(*ForegroundModel)
	if fg.ActiveModal != types.ModalTypeNone {
		t.Fatalf("expected modal to close on Success, got %v", fg.ActiveModal)
	}
}

func TestForegroundModel_AddDuplicateShortcut(t *testing.T) {
	fg := NewForegroundModel()

	pls := []*musicpb.Playlist{
		{PlaylistId: "PL1", Title: "Chill Vibes", Count: 10},
	}

	model, _ := fg.Update(types.OpenAddToPlaylistModalMsg{
		TrackID:    "track123",
		TrackTitle: "Blinding Lights",
		Playlists:  pls,
	})
	fg = model.(*ForegroundModel)

	model, cmd := fg.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	fg = model.(*ForegroundModel)

	if !fg.IsSubmitting {
		t.Fatalf("expected IsSubmitting to be true after pressing 'a'")
	}
	if cmd == nil {
		t.Fatalf("expected command on pressing 'a'")
	}
	msg := cmd()
	addMsg, ok := msg.(types.AddToPlaylistMsg)
	if !ok || !addMsg.Duplicates {
		t.Fatalf("expected AddToPlaylistMsg with Duplicates=true, got %v", msg)
	}
}
