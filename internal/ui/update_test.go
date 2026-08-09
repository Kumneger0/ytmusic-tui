package ui

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	musicpb "github.com/kumneger0/ytmusic-tui/gen"
	"github.com/kumneger0/ytmusic-tui/internal/queue"
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
	m.QueueList = list.New(nil, d, dims.SidebarWidth, dims.ContentHeight)
	m.Queue = queue.NewRingQueue()
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

	result, _ := m.Update(types.LikeUnlikeTrackResponseMsg{TrackID: "vid1", Liked: true})
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

	result, _ := m.Update(types.LikeUnlikeTrackResponseMsg{TrackID: "other", Liked: true})
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

func TestUpdate_LyricsView_JKScrolling(t *testing.T) {
	m := newTestModel()
	m.FocusedOn = MainView
	m.MainViewMode = LyricsMode
	m.LyricsView.SetContent("Line 1\nLine 2\nLine 3\nLine 4\nLine 5")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updated := result.(Model)
	if updated.MainViewMode != LyricsMode {
		t.Errorf("MainViewMode: want LyricsMode, got %v", updated.MainViewMode)
	}

	result2, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	updated2 := result2.(Model)
	if updated2.MainViewMode != LyricsMode {
		t.Errorf("MainViewMode: want LyricsMode, got %v", updated2.MainViewMode)
	}
}

func TestUpdate_QueueList_RemovalAndNavigation(t *testing.T) {
	m := newTestModel()
	m.FocusedOn = QueueList
	m.RightColumnMode = RightColumnQueue

	trackObj := types.PlaylistTrackObject{Track: &musicpb.Song{VideoId: "song1", Title: "Song 1"}}
	m.Queue = queue.NewRingQueue()
	m.Queue.AddTrack(&trackObj)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	updated := result.(Model)
	if updated.Queue.Len() != 0 {
		t.Errorf("Queue items: want 0 after removal, got %d", updated.Queue.Len())
	}

	resultTab, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updatedTab := resultTab.(Model)
	if updatedTab.FocusedOn != Player {
		t.Errorf("FocusedOn after tab: want Player, got %s", updatedTab.FocusedOn)
	}

	resultShiftTab, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	updatedShiftTab := resultShiftTab.(Model)
	if updatedShiftTab.FocusedOn != MainView {
		t.Errorf("FocusedOn after shift+tab: want MainView, got %s", updatedShiftTab.FocusedOn)
	}
}

func TestUpdate_QueueList_MultiTrackRemovalAndPlayback(t *testing.T) {
	m := newTestModel()
	m.FocusedOn = QueueList
	m.RightColumnMode = RightColumnQueue

	t1 := types.PlaylistTrackObject{Track: &musicpb.Song{VideoId: "song1", Title: "Song 1"}}
	t2 := types.PlaylistTrackObject{Track: &musicpb.Song{VideoId: "song2", Title: "Song 2"}}
	t3 := types.PlaylistTrackObject{Track: &musicpb.Song{VideoId: "song3", Title: "Song 3"}}

	m.Queue = queue.NewRingQueue()
	m.Queue.AddTrack(&t1)
	m.Queue.AddTrack(&t2)
	m.Queue.AddTrack(&t3)
	m.SyncQueueList()

	// Select row 2 (which is track 2, since row 0 is header "Queue", row 1 is track 1, row 2 is track 2)
	m.QueueList.Select(2)

	// Press 'r' to remove selected row 2 (Song 2)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	updated := result.(Model)

	if updated.Queue.Len() != 2 {
		t.Fatalf("Queue items: want 2 after removal of middle track, got %d", updated.Queue.Len())
	}

	tracks := updated.Queue.AllTracks()
	if len(tracks) == 2 && (tracks[0].Track.VideoId != "song1" || tracks[1].Track.VideoId != "song3") {
		t.Errorf("Remaining tracks want [song1, song3], got [%s, %s]", tracks[0].Track.VideoId, tracks[1].Track.VideoId)
	}

	// Select row 1 (song1) and press Enter
	updated.QueueList.Select(1)
	resEnter, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mPlayed := resEnter.(Model)
	if mPlayed.SelectedTrack == nil || mPlayed.SelectedTrack.Track == nil || mPlayed.SelectedTrack.Track.Track.VideoId != "song1" {
		t.Errorf("Played track: want song1, got %v", mPlayed.SelectedTrack)
	}
}

func TestUpdate_QueueList_ContextItemSelection(t *testing.T) {
	m := newTestModel()
	m.FocusedOn = QueueList
	m.RightColumnMode = RightColumnQueue

	t1 := &types.PlaylistTrackObject{Track: &musicpb.Song{VideoId: "ctx1", Title: "Context 1"}}
	t2 := &types.PlaylistTrackObject{Track: &musicpb.Song{VideoId: "ctx2", Title: "Context 2"}}
	t3 := &types.PlaylistTrackObject{Track: &musicpb.Song{VideoId: "ctx3", Title: "Context 3"}}

	m.SetPlaybackContext([]*types.PlaylistTrackObject{t1, t2, t3}, "My Playlist", 0)
	m.SyncQueueList()

	// Select row 2 in QueueList (row 0 is header "Next from My Playlist", row 1 is ctx1, row 2 is ctx2)
	m.QueueList.Select(2)

	resEnter, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mPlayed := resEnter.(Model)

	if mPlayed.PlaylistContextIndex != 1 {
		t.Errorf("PlaylistContextIndex: want 1 for ctx2, got %d", mPlayed.PlaylistContextIndex)
	}
	if mPlayed.SelectedTrack == nil || mPlayed.SelectedTrack.Track.Track.VideoId != "ctx2" {
		t.Errorf("Played track: want ctx2, got %v", mPlayed.SelectedTrack)
	}
}

func TestUpdate_PlayerActions(t *testing.T) {
	m := newTestModel()
	m.FocusedOn = Player
	m.SelectedTrack = &SelectedTrack{
		Track: &types.PlaylistTrackObject{
			Track: &musicpb.Song{VideoId: "v1", Title: "Test Song"},
		},
	}

	// Space (Play/Pause)
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	// Prev ('b')
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	// Next ('n')
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Like ('l')
	_, cmdLike := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if cmdLike == nil {
		t.Error("cmdLike should not be nil when pressing 'l' on selected track")
	}

	// LyricsKey (ctrl+l)
	resultLyrics, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	updatedLyrics := resultLyrics.(Model)
	if updatedLyrics.MainViewMode != LyricsMode {
		t.Errorf("MainViewMode after ctrl+l: want LyricsMode, got %v", updatedLyrics.MainViewMode)
	}
	if updatedLyrics.FocusedOn != MainView {
		t.Errorf("FocusedOn after ctrl+l: want MainView, got %s", updatedLyrics.FocusedOn)
	}
}

func TestUpdate_SearchBar_FocusSubmitCancel(t *testing.T) {
	m := newTestModel()
	m.FocusedOn = MainView

	resFocus, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlK})
	updatedFocus := resFocus.(Model)
	if updatedFocus.FocusedOn != SearchBar {
		t.Fatalf("FocusedOn: want SearchBar, got %s", updatedFocus.FocusedOn)
	}

	updatedFocus.Search.SetValue("Kendrick")

	resSubmit, cmdSubmit := updatedFocus.Update(tea.KeyMsg{Type: tea.KeyEnter})
	submittedModel := resSubmit.(Model)
	if cmdSubmit == nil {
		t.Error("cmdSubmit should not be nil when pressing enter in SearchBar")
	}

	_, cmdQ := submittedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmdQ != nil {
		msg := cmdQ()
		if msg == tea.Quit() {
			t.Error("q in SearchBar should not quit application")
		}
	}

	resCancel, _ := submittedModel.Update(tea.KeyMsg{Type: tea.KeyEscape})
	updatedCancel := resCancel.(Model)
	if updatedCancel.FocusedOn == SearchBar {
		t.Errorf("FocusedOn after escape: should not be SearchBar, got %s", updatedCancel.FocusedOn)
	}
}

func TestUpdate_SongRelatedContent_ArtistSelection(t *testing.T) {
	m := newTestModel()
	m.FocusedOn = QueueList
	m.RightColumnMode = RightColumnRelated

	artistItem := types.SongRelatedContentItem{
		SongRelatedContent: &musicpb.SongRelatedContent{
			Title:       "Kendrick Lamar",
			BrowseId:    "UCq3V-gFmAdGlc8c-1M8-g",
			ContentType: "artist",
			Subscribers: "10M",
		},
	}

	m.RelatedList = list.New([]list.Item{artistItem}, CustomDelegate{Model: &m}, 20, 10)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("cmd should not be nil when selecting related artist")
	}
}
