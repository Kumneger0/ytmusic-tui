package ui

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	musicpb "github.com/kumneger0/ytmusic-tui/gen"
	"github.com/kumneger0/ytmusic-tui/internal/types"
	"go.dalton.dog/bubbleup"
)

func newTestModel() Model {
	m := Model{
		FocusedOn:    SideView,
		MainViewMode: HomePageMode,
		Alert:        *bubbleup.NewAlertModel(80, false, 5*time.Second),
		Width:        120,
		Height:       40,
	}
	dims := CalculateLayoutDimensions(&m)
	d := CustomDelegate{Model: &m}
	m.SideBarList = list.New(nil, d, dims.SidebarWidth, dims.ContentHeight)
	m.SelectedPlayListItems = list.New(nil, d, dims.MainWidth, dims.ContentHeight)
	m.HomePageList = list.New(nil, d, dims.MainWidth, dims.ContentHeight)
	m.SearchResult = list.New(nil, d, dims.MainWidth, dims.ContentHeight)
	m.RelatedList = list.New(nil, d, dims.SidebarWidth, dims.ContentHeight)
	m.Search = textinput.New()
	return m
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := newTestModel()
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	result, _ := m.Update(msg)
	updated := result.(Model)

	if updated.Width != 116 {
		t.Errorf("Width: want 116, got %d", updated.Width)
	}
	if updated.Height != 36 {
		t.Errorf("Height: want 36, got %d", updated.Height)
	}
	if updated.LibraryWidth == 0 {
		t.Error("LibraryWidth was not set")
	}
	if updated.MainViewWidth == 0 {
		t.Error("MainViewWidth was not set")
	}
}

func TestUpdate_SearchingMsg(t *testing.T) {
	m := newTestModel()
	m.IsSearchLoading = false

	result, _ := m.Update(types.SearchingMsg{})
	updated := result.(Model)

	if !updated.IsSearchLoading {
		t.Error("IsSearchLoading should be true after SearchingMsg")
	}
}

func TestUpdate_PlayedSecondsUpdateMsg_NilTrack(t *testing.T) {
	m := newTestModel()
	m.SelectedTrack = nil

	result, _ := m.Update(types.PlayedSecondsUpdateMsg{CurrentSeconds: 42.0})
	updated := result.(Model)

	if updated.PlayedSeconds != 0 {
		t.Errorf("PlayedSeconds: want 0, got %f", updated.PlayedSeconds)
	}
}

func TestUpdate_PlayedSecondsUpdateMsg_UpdatesSeconds(t *testing.T) {
	m := newTestModel()
	m.SelectedTrack = &SelectedTrack{
		Track: &types.PlaylistTrackObject{
			Track: &musicpb.Song{DurationSeconds: 200},
		},
	}

	result, _ := m.Update(types.PlayedSecondsUpdateMsg{CurrentSeconds: 55.5})
	updated := result.(Model)

	if updated.PlayedSeconds != 55.5 {
		t.Errorf("PlayedSeconds: want 55.5, got %f", updated.PlayedSeconds)
	}
}

func TestUpdate_LikeUnlikeTrackMsg(t *testing.T) {
	m := newTestModel()
	m.SelectedTrack = &SelectedTrack{
		isLiked: false,
		Track: &types.PlaylistTrackObject{
			Track: &musicpb.Song{VideoId: "vid1"},
		},
	}

	result, _ := m.Update(types.LikeUnlikeTrackMsg{TrackID: "vid1", Like: true})
	updated := result.(Model)

	if !updated.SelectedTrack.isLiked {
		t.Error("track should be liked after LikeUnlikeTrackMsg{Like: true}")
	}
}

func TestUpdate_LikeUnlikeTrackMsg_DifferentID(t *testing.T) {
	m := newTestModel()
	m.SelectedTrack = &SelectedTrack{
		isLiked: false,
		Track: &types.PlaylistTrackObject{
			Track: &musicpb.Song{VideoId: "vid1"},
		},
	}

	result, _ := m.Update(types.LikeUnlikeTrackMsg{TrackID: "other", Like: true})
	updated := result.(Model)

	if updated.SelectedTrack.isLiked {
		t.Error("track should remain unliked when TrackID doesn't match")
	}
}

func TestUpdate_CheckUserSavedTrackMsg(t *testing.T) {
	m := newTestModel()
	m.SelectedTrack = &SelectedTrack{
		isLiked: false,
		Track: &types.PlaylistTrackObject{
			Track: &musicpb.Song{VideoId: "vid1"},
		},
	}

	result, _ := m.Update(types.CheckUserSavedTrackResponseMsg{Saved: true})
	updated := result.(Model)

	if !updated.SelectedTrack.isLiked {
		t.Error("isLiked should be true after CheckUserSavedTrackResponseMsg{Saved: true}")
	}
}

func TestUpdate_HomePageResponseMsg_Success(t *testing.T) {
	m := newTestModel()
	m.IsSearchLoading = true

	resp := &musicpb.GetHomePageResponse{
		Sections: []*musicpb.HomePageSection{
			{Title: "Quick picks"},
			{Title: "Listen again"},
		},
	}

	result, _ := m.Update(types.HomePageResponseMsg{Response: resp})
	updated := result.(Model)

	if updated.IsSearchLoading {
		t.Error("IsSearchLoading should be false after HomePageResponseMsg")
	}
	if updated.MainViewMode != HomePageMode {
		t.Errorf("MainViewMode: want %q, got %q", HomePageMode, updated.MainViewMode)
	}
	if updated.HomePageViewMode != HomePageSectionView {
		t.Errorf("HomePageViewMode: want HomePageSectionView, got %d", updated.HomePageViewMode)
	}
	if updated.HomePageData == nil {
		t.Fatal("HomePageData should not be nil")
	}
	items := updated.HomePageList.Items()
	if len(items) != 2 {
		t.Fatalf("HomePageList items: want 2, got %d", len(items))
	}
}

func TestUpdate_HomePageResponseMsg_Error(t *testing.T) {
	m := newTestModel()
	m.IsSearchLoading = true

	result, cmd := m.Update(types.HomePageResponseMsg{Err: errors.New("network down")})
	updated := result.(Model)

	if updated.IsSearchLoading {
		t.Error("IsSearchLoading should be false on error")
	}
	if cmd == nil {
		t.Error("cmd should not be nil (alert batch)")
	}
}

func TestUpdate_LyricsMsg_NoLyrics(t *testing.T) {
	m := newTestModel()
	m.CurrentLyrics = &musicpb.GetLyricsResponse{}

	result, _ := m.Update(types.LyricsMsg{LyricsResponse: nil})
	updated := result.(Model)

	if updated.CurrentLyrics != nil {
		t.Error("CurrentLyrics should be nil when no lyrics found")
	}
}

func TestUpdate_LyricsMsg_WithLyrics(t *testing.T) {
	m := newTestModel()

	lyrics := &musicpb.GetLyricsResponse{
		Lyrics: "Hello world",
	}

	result, _ := m.Update(types.LyricsMsg{LyricsResponse: lyrics})
	updated := result.(Model)

	if updated.CurrentLyrics == nil {
		t.Fatal("CurrentLyrics should not be nil")
	}
	if updated.CurrentLyrics.Lyrics != "Hello world" {
		t.Errorf("Lyrics: want %q, got %q", "Hello world", updated.CurrentLyrics.Lyrics)
	}
}

func TestUpdate_GetLibraryMsg_Error(t *testing.T) {
	m := newTestModel()
	m.IsSearchLoading = true

	result, cmd := m.Update(types.GetLibraryMsg{Err: errors.New("auth failed")})
	updated := result.(Model)

	if updated.IsSearchLoading {
		t.Error("IsSearchLoading should be false on error")
	}
	if cmd == nil {
		t.Error("cmd should not be nil (alert)")
	}
}

func TestUpdate_KeyMsg_Quit(t *testing.T) {
	m := newTestModel()
	m.FocusedOn = MainView

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if cmd == nil {
		t.Fatal("cmd should not be nil after quit key")
	}
}

func TestUpdate_KeyMsg_TabCyclesFocus(t *testing.T) {
	tests := []struct {
		name     string
		start    FocusedOn
		key      tea.KeyType
		wantNext FocusedOn
	}{
		{"SideView->tab->MainView", SideView, tea.KeyTab, MainView},
		{"MainView->tab->QueueList", MainView, tea.KeyTab, QueueList},
		{"QueueList->tab->Player", QueueList, tea.KeyTab, Player},
		{"Player->tab->SideView", Player, tea.KeyTab, SideView},
		{"MainView->shift+tab->SideView", MainView, tea.KeyShiftTab, SideView},
		{"Player->shift+tab->QueueList", Player, tea.KeyShiftTab, QueueList},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.FocusedOn = tc.start
			result, _ := m.Update(tea.KeyMsg{Type: tc.key})
			updated := result.(Model)
			if updated.FocusedOn != tc.wantNext {
				t.Errorf("FocusedOn: want %q, got %q", tc.wantNext, updated.FocusedOn)
			}
		})
	}
}

func TestUpdate_KeyMsg_CtrlK_OpensSearchBar(t *testing.T) {
	m := newTestModel()
	m.FocusedOn = MainView

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	updated := result.(Model)

	if updated.FocusedOn != SearchBar {
		t.Errorf("FocusedOn: want SearchBar, got %q", updated.FocusedOn)
	}
}

func TestUpdate_KeyMsg_CtrlQ_TogglesRightColumn(t *testing.T) {
	m := newTestModel()
	m.FocusedOn = QueueList
	m.RightColumnMode = RightColumnQueue

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	updated := result.(Model)

	if updated.RightColumnMode == RightColumnQueue {
		t.Error("RightColumnMode should have toggled away from RightColumnQueue")
	}
}
