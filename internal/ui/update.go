package ui

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/godbus/dbus/v5"
	musicpb "github.com/kumneger0/ytmusic-tui/gen"
	"github.com/kumneger0/ytmusic-tui/internal/types"
	"github.com/kumneger0/ytmusic-tui/internal/youtube"
	"go.dalton.dog/bubbleup"
)

type MusicMetadata struct {
	artistName string
	title      string
	length     int64
}

func getMusicMetadata(music MusicMetadata) map[string]any {
	var metadata = map[string]any{
		"mpris:trackid": "/org/mpris/MediaPlayer2/" + music.title,
		"mpris:length":  music.length,
		"xesam:title":   music.title,
		"xesam:artist":  music.artistName,
	}
	return metadata
}

func (m Model) getSearchResultModel(searchResponse *types.SearchResponse) (Model, tea.Cmd) {
	dims := CalculateLayoutDimensions(&m)
	m.SearchResult = list.New(searchResponse.Items, CustomDelegate{Model: &m}, dims.MainWidth, dims.ContentHeight)
	return m, nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case types.GetLibraryMsg:
		m.IsSearchLoading = false
		if msg.Err != nil {
			slog.Error(msg.Err.Error())
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		if msg.Result != nil {
			var items []list.Item
			for _, s := range msg.Result.Songs {
				items = append(items, types.PlaylistTrackObject{
					Track: s,
				})
			}
			for _, p := range msg.Result.Playlists {
				items = append(items, types.PlaylistItem{Playlist: p})
			}
			for _, al := range msg.Result.Albums {
				items = append(items, types.AlbumItem{Album: al})
			}
			for _, a := range msg.Result.Artists {
				items = append(items, types.FollowedArtistItem{FollowedArtist: a})
			}
			for _, c := range msg.Result.Channels {
				items = append(items, types.LibraryChannelItem{LibraryChannel: c})
			}
			for _, pod := range msg.Result.Podcasts {
				items = append(items, types.PodcastItem{Podcast: pod})
			}
			dims := CalculateLayoutDimensions(&m)
			m.SelectedPlayListItems = list.New(items, CustomDelegate{Model: &m}, dims.MainWidth, dims.ContentHeight)
			removeListDefaults(&m.SelectedPlayListItems)
			m.MainViewMode = NormalMode
			m.FocusedOn = MainView
			return m, nil
		}

	case types.PythonBackendHealthResponseMsg:
		m.IsSearchLoading = false
		if msg.Err != nil {
			slog.Error(msg.Err.Error())
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		if !msg.Response.Ok {
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, "Health Check Error")
			return m, alertCmd
		}
		homePageFeed := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			homePage, err := m.YtMusicClient.GetHomePage(ctx, &musicpb.GetHomePageRequest{})
			if err != nil {
				slog.Error(err.Error())
				return types.HomePageResponseMsg{
					Response: nil,
					Err:      err,
				}
			}
			return types.HomePageResponseMsg{
				Response: homePage,
				Err:      nil,
			}
		}
		cmds = append(cmds, SendLoadingCmd(), homePageFeed)
	case types.PlaylistDetailMsg:
		if msg.Err != nil {
			slog.Error(msg.Err.Error())
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		var tracks []*types.PlaylistTrackObject
		for _, track := range msg.Playlist.Tracks {
			tracks = append(tracks, &types.PlaylistTrackObject{
				Track: track,
			})
		}
		cmd := func() tea.Msg {
			return types.UpdatePlaylistMsg{
				Playlist: tracks,
			}
		}
		return m, cmd
	case types.UpdateHomePageContentMsg:
		var items []list.Item
		contents := m.HomePageData.Sections[msg.Item.Index]
		if contents == nil {
			return m, nil
		}
		for _, content := range contents.Contents {
			items = append(items, types.HomePageContentItem{
				ItemTitle:   content.Title,
				PlaylistID:  content.PlaylistId,
				VideoID:     content.VideoId,
				BrowseID:    content.BrowseId,
				ContentType: content.ContentType,
				Description: content.Description,
			})
		}
		dims := CalculateLayoutDimensions(&m)
		m.HomePageList = list.New(items, CustomDelegate{Model: &m}, dims.MainWidth, dims.ContentHeight)
		m.IsSearchLoading = false
		removeListDefaults(&m.HomePageList)
		m.HomePageList.Title = msg.Item.Title()
		m.HomePageViewMode = HomePageContentView
		m.MainViewMode = HomePageMode
		return m, nil
	case types.SearchAndDownloadMusicMsg:
		if msg.Err != nil {
			slog.Error(msg.Err.Error())
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		if msg.Player == nil {
			return m, nil
		}
		if m.SelectedTrack != nil && m.SelectedTrack.Track != nil && m.SelectedTrack.Track.Track != nil && msg.VideoID != m.SelectedTrack.Track.Track.VideoId {
			_ = msg.Player.Close()
			return m, nil
		}
		if m.SelectedTrack.Track.Track.DurationSeconds == 0 && msg.Duration != "" {
			if duration, err := strconv.ParseInt(msg.Duration, 10, 64); err == nil {
				m.SelectedTrack.Track.Track.DurationSeconds = int32(duration)
			} else {
				slog.Error(err.Error())
			}
		}
		likedCmd := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			resp, err := m.YtMusicClient.CheckUserSavedTrack(ctx, &musicpb.CheckUserSavedTrackRequest{
				VideoId: msg.VideoID,
			})
			if err != nil {
				return types.CheckUserSavedTrackResponseMsg{
					Saved: false,
					Err:   err,
				}
			}
			return types.CheckUserSavedTrackResponseMsg{
				Saved: resp.IsSaved,
				Err:   err,
			}
		}
		cmds = append(cmds, likedCmd)
		m.PlayerProcess = msg.Player
	case types.CheckUserSavedTrackResponseMsg:
		if msg.Err != nil {
			slog.Error(msg.Err.Error())
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		m.SelectedTrack.isLiked = msg.Saved
		return m, nil
	case types.SearchingMsg:
		m.IsSearchLoading = true
	case types.SearchResultMsg:
		var alertCmd tea.Cmd
		if msg.Err != nil {
			alertCmd = m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			cmds = append(cmds, alertCmd)
		}
		if msg.Result != nil {
			m.FocusedOn = MainView
			m.MainViewMode = SearchResultMode
			model, cmd := m.getSearchResultModel(msg.Result)
			m = model
			cmds = append(cmds, cmd)
			m.IsSearchLoading = false
		}
	case types.HomePageResponseMsg:
		var alertCmd tea.Cmd
		m.IsSearchLoading = false
		if msg.Err != nil {
			slog.Error(msg.Err.Error())
			alertCmd = m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			cmds = append(cmds, alertCmd)
			return m, tea.Batch(cmds...)
		}
		m.HomePageData = msg.Response
		var items []list.Item
		for i, section := range msg.Response.Sections {
			items = append(items, types.HomePageSectionItem{
				SectionTitle: section.Title,
				Index:        i,
			})
		}

		dims := CalculateLayoutDimensions(&m)
		m.HomePageList = list.New(items, CustomDelegate{Model: &m}, dims.MainWidth, dims.ContentHeight)
		removeListDefaults(&m.HomePageList)
		m.HomePageList.Title = "Home"
		m.HomePageViewMode = HomePageSectionView
		m.MainViewMode = HomePageMode
		return m, nil
	case types.DBusMessage:
		model, cmd := m.handleDbusMessage(msg.MessageType, cmds)
		m = model
		cmds = append(cmds, cmd)
	case types.LikeUnlikeTrackMsg:
		if m.SelectedTrack != nil && m.SelectedTrack.Track != nil && m.SelectedTrack.Track.Track != nil && msg.TrackID == m.SelectedTrack.Track.Track.VideoId {
			m.SelectedTrack.isLiked = msg.Like
		}
	case types.PlayedSecondsUpdateMsg:
		if m.SelectedTrack == nil || m.SelectedTrack.Track == nil || m.SelectedTrack.Track.Track == nil {
			return m, nil
		}
		m.PlayedSeconds = msg.CurrentSeconds
		if m.CurrentLyrics != nil {
			m.updateLyricsView()
		}
		totalDurationInSeconds := m.SelectedTrack.Track.Track.DurationSeconds
		if totalDurationInSeconds > 0 && (float64(totalDurationInSeconds)-m.PlayedSeconds) < 1 {
			m.PlayedSeconds = 0
			model, cmd := m.handleMusicChange(true)
			m = model
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width - 4
		m.Height = msg.Height - 4
		dims := calculateLayoutDimensions(&m)
		m.LibraryWidth = dims.SidebarWidth
		m.MainViewWidth = dims.MainWidth
		m.PlayerSectionHeight = dims.InputHeight
		m.LyricsView.Width = max(dims.MainWidth-6, 10)
		m.LyricsView.Height = max(dims.ContentHeight-6, 10)
		return m, nil
	case types.UpdatePlaylistMsg:
		if msg.Playlist != nil {
			var playListItemSongs []list.Item
			for _, item := range msg.Playlist {
				if msg.ShouldAppendQueue {
					item.IsItFromQueue = true
				}
				playListItemSongs = append(playListItemSongs, *item)
			}
			m.MainViewMode = NormalMode
			m.IsSearchLoading = false
			var currentItems []list.Item
			if msg.ShouldAppendQueue && m.MusicQueueList != nil {
				currentItems = m.MusicQueueList.Items()
			} else {
				currentItems = m.SelectedPlayListItems.Items()
			}
			if msg.ShouldAppend {
				playListItemSongs = append(currentItems, playListItemSongs...)
				m.IsOnPagination = false
			}

			var cmd tea.Cmd
			if msg.ShouldAppendQueue {
				if m.MusicQueueList != nil {
					return m, nil
				}
				cmd = m.MusicQueueList.SetItems(playListItemSongs)
			} else {
				cmd = m.SelectedPlayListItems.SetItems(playListItemSongs)
			}
			if msg.PaginationInfo != nil {
				m.PaginationInfo = msg.PaginationInfo
			} else {
				m.PaginationInfo = nil
			}
			cmds = append(cmds, cmd)
		}
		if msg.Err != nil {
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			cmds = append(cmds, alertCmd)
		}
	case types.RelatedSongsMsg:
		if msg.Err != nil {
			slog.Error(msg.Err.Error())
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			cmds = append(cmds, alertCmd)
		}
		if msg.Related == nil || len(msg.Related.Sections) == 0 {
			slog.Error("Failed to Fetch Related Songs")
			return m, nil
		}
		var items []list.Item
		for _, section := range msg.Related.Sections {
			if section.Title != "" {
				items = append(items, types.HomePageSectionItem{SectionTitle: section.Title})
			}
			for _, content := range section.Contents {
				items = append(items, types.SongRelatedContentItem{SongRelatedContent: content})
			}
			if section.TextContent != "" {
				items = append(items, types.HomePageContentItem{
					ItemTitle:   section.Title,
					Description: section.TextContent,
				})
			}
		}
		dims := CalculateLayoutDimensions(&m)
		m.RelatedList = list.New(items, CustomDelegate{Model: &m}, dims.SidebarWidth, dims.ContentHeight)
		removeListDefaults(&m.RelatedList)
		m.RelatedList.Title = "Related"
		m.RightColumnMode = RightColumnRelated
		return m, nil
	case types.LyricsMsg:
		if msg.Err != nil {
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			slog.Error(msg.Err.Error())
			m.LyricsView.SetContent(msg.Err.Error())
			return m, alertCmd
		}
		if msg.LyricsResponse == nil || (msg.LyricsResponse.Lyrics == "" && len(msg.LyricsResponse.Lines) == 0) {
			noLyricsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A")).Italic(true)
			m.LyricsView.SetContent(noLyricsStyle.Render("No lyrics found for this song."))
			m.CurrentLyrics = nil
			return m, nil
		}
		m.CurrentLyrics = msg.LyricsResponse
		m.updateLyricsView()
		return m, nil
	case tea.KeyMsg:
		model, cmd := m.handleKeyPress(msg)
		m = model
		cmds = append(cmds, cmd)
	case tea.MouseMsg:
		x := msg.X
		y := msg.Y
		if x > m.LibraryWidth && x <= (m.LibraryWidth+m.MainViewWidth) && y <= (m.Height-m.PlayerSectionHeight) {
			if m.MainViewMode == LyricsMode && m.FocusedOn != SearchBar {
				lyricsModel, cmd := m.LyricsView.Update(msg)
				m.LyricsView = lyricsModel
				cmds = append(cmds, cmd)
			}
		}

	default:
	}
	model, cmd := updateFocusedComponent(&m, msg, &cmds)
	m = model
	outAlert, outCmd := m.Alert.Update(msg)
	cmds = append(cmds, outCmd, cmd)
	m.Alert = outAlert.(bubbleup.AlertModel)
	return m, tea.Batch(cmds...)
}

func (m Model) handleDbusMessage(msg types.MessageType, cmds []tea.Cmd) (Model, tea.Cmd) {
	switch msg {
	case types.NextTrack:
		model, cmd := m.handleMusicChange(true)
		m = model
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case types.PreviousTrack:
		model, cmd := m.handleMusicChange(false)
		m = model
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case types.PlayPause:
		model, cmd := m.HandleMusicPausePlay()
		m = model
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) handlePagination(listModel *list.Model, ShouldAppendQueue bool, currentIndex *int) (Model, tea.Cmd) {
	if listModel == nil {
		return m, nil
	}
	if currentIndex == nil {
		index := listModel.GlobalIndex()
		currentIndex = &index
	}
	totalItems := listModel.Items()
	if *currentIndex+5 >= len(totalItems) && m.PaginationInfo != nil && m.PaginationInfo.Next != "" {
		if m.IsOnPagination {
			return m, nil
		}
		m.IsOnPagination = true
		var paginationInfo *types.PaginationInfo
		if m.FocusedOn == QueueList && m.MusicQueueList != nil && m.MusicQueueList.PaginationInfo != nil {
			paginationInfo = m.MusicQueueList.PaginationInfo
		} else {
			paginationInfo = m.PaginationInfo
		}
		return m, getNextPageItems(&m, paginationInfo, ShouldAppendQueue)
	}
	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "down", "j":
		if m.MainViewMode == LyricsMode && m.FocusedOn == MainView {
			var cmd tea.Cmd
			m.LyricsView, cmd = m.LyricsView.Update(msg)
			return m, cmd
		}
		if m.FocusedOn != MainView && m.FocusedOn != QueueList {
			return m, nil
		}
		if m.MainViewMode == HomePageMode {
			var cmd tea.Cmd
			m.HomePageList, cmd = m.HomePageList.Update(msg)
			return m, cmd
		}
		listModel := getListItemForMusicToChoose(&m, m.FocusedOn)
		return m.handlePagination(listModel, m.FocusedOn == QueueList, nil)
	case "up", "k":
		if m.MainViewMode == LyricsMode && m.FocusedOn == MainView {
			var cmd tea.Cmd
			m.LyricsView, cmd = m.LyricsView.Update(msg)
			return m, cmd
		}
		if m.FocusedOn != MainView && m.FocusedOn != QueueList {
			return m, nil
		}
		if m.MainViewMode == HomePageMode {
			var cmd tea.Cmd
			m.HomePageList, cmd = m.HomePageList.Update(msg)
			return m, cmd
		}
		return m, nil
	case "ctrl+k":
		m.FocusedOn = SearchBar
		return m, m.Search.Focus()
	case "esc", "escape":
		if m.FocusedOn == SearchBar {
			m.Search.Blur()
			m.FocusedOn = SideView
			return m, nil
		}
		if m.MainViewMode == HomePageMode && m.HomePageViewMode == HomePageContentView {
			var items []list.Item
			for i, section := range m.HomePageData.Sections {
				items = append(items, types.HomePageSectionItem{
					SectionTitle: section.Title,
					Index:        i,
				})
			}
			dims := CalculateLayoutDimensions(&m)
			m.HomePageList = list.New(items, CustomDelegate{Model: &m}, dims.MainWidth, dims.ContentHeight)
			removeListDefaults(&m.HomePageList)
			m.HomePageList.Title = "Home"
			m.HomePageViewMode = HomePageSectionView
			return m, nil
		}
	case "a":
		return m.addMusicToQueue()
	case "r":
		if m.FocusedOn == QueueList {
			showRelated := (m.RightColumnMode == RightColumnRelated || m.RightColumnMode == "") && len(m.RelatedList.Items()) > 0
			if showRelated {
				if len(m.RelatedList.Items()) > 0 {
					m.RelatedList.RemoveItem(m.RelatedList.Index())
				}
			} else if m.MusicQueueList != nil {
				if len(m.MusicQueueList.Model.Items()) > 0 {
					m.MusicQueueList.Model.RemoveItem(m.MusicQueueList.GlobalIndex())
				}
			}
		}
	case "ctrl+l":
		if m.MainViewMode == LyricsMode {
			m.MainViewMode = NormalMode
			return m, nil
		}
		return m.getMusicLyrics(m.SelectedTrack)
	case "l":
		if m.SelectedTrack != nil && m.SelectedTrack.Track != nil {
			cmd := func() tea.Msg {
				var shouldRemove bool
				if m.SelectedTrack.isLiked {
					shouldRemove = true
				} else {
					shouldRemove = false
				}
				ctx, _ := context.WithCancel(context.Background())
				_, err := m.YtMusicClient.SaveRemoveTrack(ctx, &musicpb.SaveRemoveTrackRequest{
					VideoIds: []string{},
					IsRemove: true,
				})

				if err != nil {
					slog.Error(err.Error())
				}
				likeUnlikeTrackMsg := types.LikeUnlikeTrackMsg{
					TrackID: m.SelectedTrack.Track.Track.VideoId,
					Like:    !shouldRemove,
					Err:     err,
				}
				return likeUnlikeTrackMsg
			}
			return m, cmd
		}
	case " ":
		if m.FocusedOn != Player {
			return m, nil
		}
		return m.HandleMusicPausePlay()
	case "b":
		if m.FocusedOn != Player {
			return m, nil
		}
		return m.handleMusicChange(false)
	case "n":
		if m.FocusedOn != Player {
			return m, nil
		}
		return m.handleMusicChange(true)
	case "ctrl+q":
		if m.FocusedOn != QueueList {
			return m, nil
		}
		m.toggleRightColumnMode()
		return m, nil
	case "q", "ctrl+c":
		if m.FocusedOn == SearchBar {
			return m, nil
		}
		if m.BackendProcess != nil && m.BackendProcess.Process != nil {
			_ = m.BackendProcess.Process.Signal(syscall.SIGTERM)
		}
		if m.playbackCancel != nil {
			m.playbackCancel()
			m.playbackCancel = nil
		}
		if m.PlayerProcess != nil {
			err := m.PlayerProcess.Close()
			if err != nil {
				slog.Error(err.Error())
			}
			m.PlayerProcess = nil
		}
		return m, tea.Quit
	case "tab":
		return changeFocusMode(&m, false)
	case "shift+tab":
		return changeFocusMode(&m, true)
	case "enter":
		return m.handleEnterKey()
	}
	return m, nil
}

func getNextPageItems(m *Model, paginationInfo *types.PaginationInfo, ShouldAppendQueue bool) tea.Cmd {
	switch paginationInfo.NextPageURLType {
	case types.NextPageURLTypePlaylistTracks:
		return func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			playlistItems, err := m.YtMusicClient.GetPlaylistItems(ctx, &musicpb.GetPlaylistItemsRequest{
				PlaylistId: paginationInfo.NextItemID,
				Limit:      100,
			})
			if err != nil {
				return types.UpdatePlaylistMsg{
					Playlist: nil,
					Err:      err,
				}
			}
			var tracks []*types.PlaylistTrackObject
			for _, track := range playlistItems.Tracks {
				tracks = append(tracks, &types.PlaylistTrackObject{
					Track: track,
				})
			}
			return types.UpdatePlaylistMsg{
				Playlist:          tracks,
				Err:               nil,
				ShouldAppend:      true,
				PaginationInfo:    paginationInfo,
				ShouldAppendQueue: ShouldAppendQueue,
			}
		}
	case types.NextPageURLTypeUserSavedItems:
		return func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			userSavedTracks, err := m.YtMusicClient.GetUserSavedTracks(ctx, &musicpb.GetUserSavedTracksRequest{
				Limit: 100,
			})
			if err != nil {
				return types.UpdatePlaylistMsg{
					Playlist: nil,
					Err:      err,
				}
			}
			var playlistItems []*types.PlaylistTrackObject
			for _, item := range userSavedTracks.Tracks {
				playlistItems = append(playlistItems, &types.PlaylistTrackObject{
					Track: item,
				})
			}
			return types.UpdatePlaylistMsg{
				Playlist:          playlistItems,
				Err:               nil,
				ShouldAppend:      true,
				PaginationInfo:    paginationInfo,
				ShouldAppendQueue: ShouldAppendQueue,
			}
		}
	}
	return nil
}
func (m Model) getMusicLyrics(track *SelectedTrack) (Model, tea.Cmd) {
	if m.MainViewMode == LyricsMode {
		m.MainViewMode = NormalMode
	} else {
		m.MainViewMode = LyricsMode
		m.FocusedOn = MainView
	}
	return m, nil
}

func (m *Model) toggleRightColumnMode() {
	if m.RightColumnMode == RightColumnRelated || m.RightColumnMode == "" {
		m.RightColumnMode = RightColumnQueue
		if m.MusicQueueList != nil && len(m.MusicQueueList.Model.Items()) > 0 {
			m.MusicQueueList.Model.Select(0)
		}
	} else {
		m.RightColumnMode = RightColumnRelated
		if len(m.RelatedList.Items()) > 0 {
			m.RelatedList.Select(0)
		}
	}
}

func (m *Model) updateLyricsView() {
	if m.CurrentLyrics == nil {
		return
	}

	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A855F7"))
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A1A1AA"))
	sourceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A")).Italic(true)

	if m.CurrentLyrics.HasTimestamps && len(m.CurrentLyrics.Lines) > 0 {
		currentMS := int32(m.PlayedSeconds*1000) - 750
		if currentMS < 0 {
			currentMS = 0
		}
		activeIdx := -1

		for i, line := range m.CurrentLyrics.Lines {
			if line.StartTime <= currentMS {
				activeIdx = i
			} else {
				break
			}
		}

		if activeIdx < 0 && len(m.CurrentLyrics.Lines) > 0 {
			activeIdx = 0
		}

		var lines []string
		for i, line := range m.CurrentLyrics.Lines {
			if i == activeIdx {
				lines = append(lines, activeStyle.Render("▶ "+line.Text))
			} else {
				lines = append(lines, inactiveStyle.Render("  "+line.Text))
			}
		}

		lyricsText := strings.Join(lines, "\n")
		if m.CurrentLyrics.Source != "" {
			lyricsText = lyricsText + "\n\n" + sourceStyle.Render(m.CurrentLyrics.Source)
		}

		m.LyricsView.SetContent(lyricsText)

		if activeIdx >= 0 && m.LyricsView.Height > 0 {
			targetOffset := activeIdx - (m.LyricsView.Height / 2)
			if targetOffset < 0 {
				targetOffset = 0
			}
			m.LyricsView.SetYOffset(targetOffset)
		}
	} else if m.CurrentLyrics.Lyrics != "" {
		lyricsText := m.CurrentLyrics.Lyrics
		if m.CurrentLyrics.Source != "" {
			lyricsText = lyricsText + "\n\n" + sourceStyle.Render(m.CurrentLyrics.Source)
		}
		m.LyricsView.SetContent(lyricsText)
	} else {
		noLyricsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A")).Italic(true)
		m.LyricsView.SetContent(noLyricsStyle.Render("No lyrics available for this song."))
	}
}

func (m Model) handleMusicChange(isForward bool) (Model, tea.Cmd) {
	if m.MusicQueueList == nil {
		return m, nil
	}

	if len(m.MusicQueueList.Model.Items()) <= 0 {
		return m, nil
	}

	var validItems []list.Item
	for _, item := range m.MusicQueueList.Model.Items() {
		if _, ok := item.(types.PlaylistTrackObject); ok {
			validItems = append(validItems, item)
		}
	}
	if len(validItems) != len(m.MusicQueueList.Model.Items()) {
		cmd := m.MusicQueueList.Model.SetItems(validItems)
		m.MusicQueueList.Model.Select(0)
		return m, cmd
	}

	var currentlySelectedMusicIndex int
	for index, track := range m.MusicQueueList.Model.Items() {
		playlistTrack, ok := track.(types.PlaylistTrackObject)
		if !ok {
			continue
		}
		if playlistTrack.Track != nil && m.SelectedTrack != nil && m.SelectedTrack.Track != nil && m.SelectedTrack.Track.Track != nil && playlistTrack.Track.VideoId == m.SelectedTrack.Track.Track.VideoId {
			currentlySelectedMusicIndex = index
			break
		}
	}

	if currentlySelectedMusicIndex == 0 && !isForward {
		return m, nil
	}

	var nextTrackIndex int
	if isForward && len(m.MusicQueueList.Model.Items()) == (currentlySelectedMusicIndex+1) {
		nextTrackIndex = 0
	} else if isForward {
		nextTrackIndex = currentlySelectedMusicIndex + 1
	} else {
		nextTrackIndex = currentlySelectedMusicIndex - 1
	}

	var musicToPlay types.PlaylistTrackObject
	var found bool
	for i := 0; i < len(m.MusicQueueList.Model.Items()); i++ {
		idx := (nextTrackIndex + i) % len(m.MusicQueueList.Model.Items())
		item := m.MusicQueueList.Model.Items()[idx]
		playlistTrack, ok := item.(types.PlaylistTrackObject)
		if ok {
			musicToPlay = playlistTrack
			nextTrackIndex = idx
			found = true
			break
		}
	}

	if !found {
		slog.Error("no valid PlaylistTrackObject found in music queue")
		return m, nil
	}
	m.MusicQueueList.Model.Select(nextTrackIndex)
	var paginationCmd tea.Cmd
	var model Model
	if isForward {
		if m.MusicQueueList != nil {
			model, paginationCmd = m.handlePagination(&m.MusicQueueList.Model, true, &nextTrackIndex)
			m = model
		}
	}
	model, cmd := m.PlaySelectedMusic(musicToPlay)
	m = model
	return m, tea.Batch(cmd, paginationCmd)
}

func (m Model) addMusicToQueue() (Model, tea.Cmd) {
	var itemToAdd list.Item
	var currentlyPlayingTrackID string
	if m.FocusedOn != MainView {
		return m, nil
	}

	if m.MainViewMode == NormalMode {
		itemToAdd = m.SelectedPlayListItems.SelectedItem()
	}
	if m.MainViewMode == SearchResultMode {
		if len(m.SearchResult.Items()) > 0 {
			if song, ok := m.SearchResult.SelectedItem().(types.SongItem); ok {
				itemToAdd = types.PlaylistTrackObject{Track: song.Song}
			} else if srSong, ok := m.SearchResult.SelectedItem().(types.SearchResultSongItem); ok {
				song := &musicpb.Song{
					VideoId:         srSong.VideoId,
					Title:           srSong.Title,
					Artists:         srSong.Artists,
					Album:           srSong.Album,
					AlbumId:         srSong.AlbumId,
					DurationSeconds: srSong.DurationSeconds,
					Liked:           srSong.Liked,
					Thumbnails:      srSong.Thumbnails,
					IsExplicit:      srSong.IsExplicit,
					Url:             srSong.Url,
				}
				itemToAdd = types.PlaylistTrackObject{Track: song}
			}
		}
	}

	if _, ok := itemToAdd.(types.PlaylistTrackObject); !ok {
		return m, nil
	}

	if m.SelectedTrack != nil && m.SelectedTrack.Track != nil && m.SelectedTrack.Track.Track != nil {
		currentlyPlayingTrackID = m.SelectedTrack.Track.Track.VideoId
	}

	var musicQueue = m.MusicQueueList.Items()

	if len(musicQueue) == 0 {
		var validItems []list.Item
		if _, ok := itemToAdd.(types.PlaylistTrackObject); ok {
			validItems = append(validItems, itemToAdd)
		}
		return m, m.MusicQueueList.SetItems(validItems)
	}

	item, ok := itemToAdd.(types.PlaylistTrackObject)
	if !ok {
		slog.Error("failed to cast itemToAdd to PlaylistTrackObject")
		return m, nil
	}

	item.IsItFromQueue = true
	itemToAdd = item

	var validQueueItems []list.Item
	for _, queueItem := range m.MusicQueueList.Items() {
		if _, ok := queueItem.(types.PlaylistTrackObject); ok {
			validQueueItems = append(validQueueItems, queueItem)
		}
	}

	var currentlyPlayingTrackIndex int
	for index, item := range validQueueItems {
		playlistTrackObject, ok := item.(types.PlaylistTrackObject)
		if !ok {
			continue
		}
		if playlistTrackObject.Track != nil && playlistTrackObject.Track.VideoId == currentlyPlayingTrackID {
			currentlyPlayingTrackIndex = index
		}
	}

	var itemsAfterCurrentlyPlayingTrack = validQueueItems[currentlyPlayingTrackIndex+1:]
	var itemsBeforeCurrentlyPlayingTrack = validQueueItems[:currentlyPlayingTrackIndex+1]
	cmd := m.MusicQueueList.SetItems(append(itemsBeforeCurrentlyPlayingTrack, append([]list.Item{itemToAdd}, itemsAfterCurrentlyPlayingTrack...)...))
	return m, cmd
}

func (m Model) HandleMusicPausePlay() (Model, tea.Cmd) {
	if m.PlayerProcess == nil {
		return m, nil
	}
	if m.PlayerProcess.OtoPlayer == nil {
		return m, nil
	}
	if m.PlayerProcess.OtoPlayer.IsPlaying() {
		m.PlayerProcess.OtoPlayer.Pause()

		if m.DBusConn != nil {
			dbusErr := m.DBusConn.Props.Set("org.mpris.MediaPlayer2.Player",
				"PlaybackStatus",
				dbus.MakeVariant("Paused"),
			)
			if dbusErr != nil {
				slog.Error(dbusErr.Error())
			}
		}

		return m, nil
	}

	if m.DBusConn != nil {
		dbusErr := m.DBusConn.Props.Set("org.mpris.MediaPlayer2.Player",
			"PlaybackStatus",
			dbus.MakeVariant("Playing"),
		)

		if dbusErr != nil {
			slog.Error(dbusErr.Error())
		}
	}

	m.PlayerProcess.OtoPlayer.Play()
	return m, nil
}

func getListItemForMusicToChoose(m *Model, focusedOn FocusedOn) *list.Model {
	if focusedOn == MainView && m.MainViewMode == HomePageMode {
		if m.HomePageViewMode == HomePageSectionView {
			return &m.HomePageList
		}
	}
	if focusedOn == MainView && m.MainViewMode == SearchResultMode {
		return &m.SearchResult
	}
	if focusedOn == MainView && m.MainViewMode == NormalMode {
		return &m.SelectedPlayListItems
	}
	if focusedOn == QueueList {
		showRelated := (m.RightColumnMode == RightColumnRelated || m.RightColumnMode == "") && len(m.RelatedList.Items()) > 0
		if showRelated {
			return &m.RelatedList
		}
		if m.MusicQueueList != nil {
			return &m.MusicQueueList.Model
		}
	}
	return nil
}

func (m Model) handleEnterKey() (Model, tea.Cmd) {
	switch m.FocusedOn {
	case SideView:
		return m.handleSidebarEnter()
	case MainView, QueueList:
		return m.handleMainViewOrQueueEnter()
	case SearchBar:
		return m.handleSearchBarEnter()
	default:
		return m, nil
	}
}

func (m Model) handleSidebarEnter() (Model, tea.Cmd) {
	item, ok := m.SideBarList.SelectedItem().(types.SidebarItem)
	if !ok {
		return m, nil
	}
	m.BreadcrumbItems = []types.Breadcrumb{{Name: item.Name, Icon: item.Icon}}
	itemName := strings.ToLower(strings.TrimSpace(item.Name))

	if itemName == "home" {
		homePageFeed := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			homePage, err := m.YtMusicClient.GetHomePage(ctx, &musicpb.GetHomePageRequest{})
			return types.HomePageResponseMsg{
				Response: homePage,
				Err:      err,
			}
		}
		return m, tea.Batch(SendLoadingCmd(), homePageFeed)
	}

	if itemName == "library" {
		libraryCmd := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			library, err := m.YtMusicClient.GetLibrary(ctx, &musicpb.GetLibraryRequest{Limit: 100})
			return types.GetLibraryMsg{
				Result: library,
				Err:    err,
			}
		}
		return m, tea.Batch(SendLoadingCmd(), libraryCmd)
	}

	return m, nil
}

func (m Model) navigateToDetailView(cmd tea.Cmd) (Model, tea.Cmd) {
	m.MainViewMode = NormalMode
	m.FocusedOn = MainView
	updateDelegate(&m)
	return m, tea.Batch(cmd, SendLoadingCmd())
}

func (m Model) playTrackFromList(track types.PlaylistTrackObject, rawItems []list.Item) (Model, tea.Cmd) {
	if m.MusicQueueList == nil {
		return m, nil
	}
	var items []list.Item
	for _, rawItem := range rawItems {
		if homeContent, ok := rawItem.(types.HomePageContentItem); ok && homeContent.VideoID != "" {
			playlistTrack := types.PlaylistTrackObject{
				Track: &musicpb.Song{
					VideoId: homeContent.VideoID,
					Title:   homeContent.ItemTitle,
				},
				IsItFromQueue: true,
			}
			items = append(items, playlistTrack)
		} else if songItem, ok := rawItem.(types.SongItem); ok {
			playlistTrack := types.PlaylistTrackObject{
				Track:         songItem.Song,
				IsItFromQueue: true,
			}
			items = append(items, playlistTrack)
		} else if srSongItem, ok := rawItem.(types.SearchResultSongItem); ok {
			song := &musicpb.Song{
				VideoId:         srSongItem.VideoId,
				Title:           srSongItem.Title,
				Artists:         srSongItem.Artists,
				Album:           srSongItem.Album,
				AlbumId:         srSongItem.AlbumId,
				DurationSeconds: srSongItem.DurationSeconds,
				Liked:           srSongItem.Liked,
				Thumbnails:      srSongItem.Thumbnails,
				IsExplicit:      srSongItem.IsExplicit,
				Url:             srSongItem.Url,
			}
			playlistTrack := types.PlaylistTrackObject{
				Track:         song,
				IsItFromQueue: true,
			}
			items = append(items, playlistTrack)
		} else if playlistTrack, ok := rawItem.(types.PlaylistTrackObject); ok {
			playlistItem := types.PlaylistTrackObject{
				Track:         playlistTrack.Track,
				IsItFromQueue: true,
			}
			items = append(items, playlistItem)
		}
	}
	m.MusicQueueList.Model.SetItems(items)
	m.MusicQueueList.Model.Select(m.MusicQueueList.GlobalIndex())
	return m.PlaySelectedMusic(track)
}

func (m Model) handleHomePageEnter() (Model, tea.Cmd) {
	if m.HomePageViewMode == HomePageSectionView {
		listItemToChooseMusicFrom := getListItemForMusicToChoose(&m, m.FocusedOn)
		if listItemToChooseMusicFrom == nil {
			return m, nil
		}
		item, ok := listItemToChooseMusicFrom.SelectedItem().(types.HomePageSectionItem)
		if !ok {
			return m, nil
		}
		cmd := func() tea.Msg {
			return types.UpdateHomePageContentMsg{Item: item}
		}
		m.BreadcrumbItems = append(m.BreadcrumbItems, types.Breadcrumb{Name: item.SectionTitle, Icon: ""})
		return m, cmd
	}

	if m.HomePageViewMode == HomePageContentView {
		item, ok := m.HomePageList.SelectedItem().(types.HomePageContentItem)
		if !ok {
			return m, nil
		}
		if item.VideoID != "" || item.ContentType == "song" || item.ContentType == "video" {
			trackID := item.VideoID
			if trackID == "" {
				trackID = item.PlaylistID
			}
			playlistTrack := types.PlaylistTrackObject{
				Track: &musicpb.Song{
					VideoId: trackID,
					Title:   item.ItemTitle,
				},
			}
			return m.playTrackFromList(playlistTrack, m.HomePageList.Items())
		} else if item.ContentType == "album" || strings.HasPrefix(item.BrowseID, "MPRE") {
			browseID := item.BrowseID
			if browseID == "" {
				browseID = item.PlaylistID
			}
			return m.navigateToDetailView(m.getAlbumTracks(browseID))
		}
		playlistID := item.PlaylistID
		if playlistID == "" {
			playlistID = item.BrowseID
		}
		playlistDetailMsg := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			playlistItems, err := m.YtMusicClient.GetPlaylistItems(ctx, &musicpb.GetPlaylistItemsRequest{
				PlaylistId: playlistID,
			})
			return types.PlaylistDetailMsg{
				Playlist: playlistItems,
				Err:      err,
			}
		}
		m.BreadcrumbItems = append(m.BreadcrumbItems, types.Breadcrumb{Name: item.ItemTitle, Icon: ""})
		return m, tea.Batch(SendLoadingCmd(), playlistDetailMsg)
	}

	return m, nil
}

func (m Model) handleMainViewOrQueueEnter() (Model, tea.Cmd) {
	if m.MainViewMode == HomePageMode && m.FocusedOn == MainView {
		return m.handleHomePageEnter()
	}

	listItemToChooseMusicFrom := getListItemForMusicToChoose(&m, m.FocusedOn)
	if listItemToChooseMusicFrom == nil {
		return m, nil
	}

	switch selectedItem := listItemToChooseMusicFrom.SelectedItem().(type) {
	case types.PlaylistTrackObject:
		if selectedItem.Track == nil {
			return m, nil
		}
		relatedSongsCmd := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			relatedSongs, err := m.YtMusicClient.GetSongRelated(ctx, &musicpb.GetSongRelatedRequest{
				VideoId: selectedItem.Track.VideoId,
			})
			return types.RelatedSongsMsg{
				Related: relatedSongs,
				Err:     err,
			}
		}
		m, cmd := m.playTrackFromList(selectedItem, listItemToChooseMusicFrom.Items())
		return m, tea.Batch(cmd, relatedSongsCmd)

	case types.SongItem:
		playlistTrack := types.PlaylistTrackObject{Track: selectedItem.Song}
		relatedSongsCmd := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			relatedSongs, err := m.YtMusicClient.GetSongRelated(ctx, &musicpb.GetSongRelatedRequest{
				VideoId: selectedItem.VideoId,
			})
			return types.RelatedSongsMsg{
				Related: relatedSongs,
				Err:     err,
			}
		}
		m, cmd := m.playTrackFromList(playlistTrack, listItemToChooseMusicFrom.Items())
		return m, tea.Batch(cmd, relatedSongsCmd)

	case types.SearchResultSongItem:
		song := &musicpb.Song{
			VideoId:         selectedItem.VideoId,
			Title:           selectedItem.Title,
			Artists:         selectedItem.Artists,
			Album:           selectedItem.Album,
			AlbumId:         selectedItem.AlbumId,
			DurationSeconds: selectedItem.DurationSeconds,
			Liked:           selectedItem.Liked,
			Thumbnails:      selectedItem.Thumbnails,
			IsExplicit:      selectedItem.IsExplicit,
			Url:             selectedItem.Url,
		}
		playlistTrack := types.PlaylistTrackObject{Track: song}
		relatedSongsCmd := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			relatedSongs, err := m.YtMusicClient.GetSongRelated(ctx, &musicpb.GetSongRelatedRequest{
				VideoId: selectedItem.VideoId,
			})
			return types.RelatedSongsMsg{
				Related: relatedSongs,
				Err:     err,
			}
		}
		m, cmd := m.PlaySelectedMusic(playlistTrack)
		return m, tea.Batch(cmd, relatedSongsCmd)

	case types.SearchResultPlaylistItem:
		return m.navigateToDetailView(m.getPlaylistItems(selectedItem.BrowseId))

	case types.SearchResultAlbumItem:
		return m.navigateToDetailView(m.getAlbumTracks(selectedItem.BrowseId))

	case types.SearchResultArtistItem:
		return m.navigateToDetailView(m.getArtistTracks(selectedItem.BrowseId))

	case types.SearchResultPodcastItem:
		return m.navigateToDetailView(m.getPlaylistItems(selectedItem.BrowseId))

	case types.SearchResultEpisodeItem:
		return m.PlaySelectedMusic(types.PlaylistTrackObject{
			Track: &musicpb.Song{
				VideoId: selectedItem.VideoId,
				Title:   selectedItem.Title,
			},
		})

	case types.PlaylistItem:
		return m.navigateToDetailView(m.getPlaylistItems(selectedItem.PlaylistId))

	case types.AlbumItem:
		return m.navigateToDetailView(m.getAlbumTracks(selectedItem.BrowseId))

	case types.ArtistItem:
		return m.navigateToDetailView(m.getArtistTracks(selectedItem.Id))

	case types.FollowedArtistItem:
		return m.navigateToDetailView(m.getArtistTracks(selectedItem.ChannelId))

	case types.LibraryChannelItem:
		return m.navigateToDetailView(m.getArtistTracks(selectedItem.BrowseId))

	case types.PodcastItem:
		return m.navigateToDetailView(m.getPlaylistItems(selectedItem.PodcastId))

	case types.SongRelatedContentItem:
		if selectedItem.VideoId != "" || selectedItem.ContentType == "song" || selectedItem.ContentType == "video" {
			playlistTrack := types.PlaylistTrackObject{
				Track: &musicpb.Song{
					VideoId: selectedItem.VideoId,
					Title:   selectedItem.Title,
					Artists: selectedItem.Artists,
				},
			}
			relatedSongsCmd := func() tea.Msg {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				relatedSongs, err := m.YtMusicClient.GetSongRelated(ctx, &musicpb.GetSongRelatedRequest{
					VideoId: selectedItem.VideoId,
				})
				return types.RelatedSongsMsg{
					Related: relatedSongs,
					Err:     err,
				}
			}
			m, cmd := m.PlaySelectedMusic(playlistTrack)
			return m, tea.Batch(cmd, relatedSongsCmd)
		} else if selectedItem.ContentType == "album" || strings.HasPrefix(selectedItem.BrowseId, "MPRE") {
			return m.navigateToDetailView(m.getAlbumTracks(selectedItem.BrowseId))
		} else if selectedItem.PlaylistId != "" || selectedItem.ContentType == "playlist" {
			return m.navigateToDetailView(m.getPlaylistItems(selectedItem.PlaylistId))
		} else if selectedItem.BrowseId != "" {
			return m.navigateToDetailView(m.getArtistTracks(selectedItem.BrowseId))
		}
	}

	return m, nil
}

func (m Model) handleSearchBarEnter() (Model, tea.Cmd) {
	query := m.Search.Value()
	if query == m.SearchQuery && len(m.SearchResult.Items()) > 0 {
		m.MainViewMode = SearchResultMode
		return m, nil
	}

	loadingCmd := SendLoadingCmd()
	searchingCmd := func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		searchResults, err := m.YtMusicClient.GetSearchResults(ctx, &musicpb.GetSearchResultsRequest{
			Query: query,
		})

		if err != nil {
			slog.Error(err.Error())
			return types.SearchResultMsg{
				Result: nil,
				Err:    err,
			}
		}

		var items []list.Item
		for _, s := range searchResults.Songs {
			items = append(items, types.SearchResultSongItem{SearchResultSong: s})
		}
		for _, a := range searchResults.Artists {
			items = append(items, types.SearchResultArtistItem{SearchResultArtist: a})
		}
		for _, p := range searchResults.Playlists {
			items = append(items, types.SearchResultPlaylistItem{SearchResultPlaylist: p})
		}
		for _, al := range searchResults.Albums {
			items = append(items, types.SearchResultAlbumItem{SearchResultAlbum: al})
		}
		for _, pod := range searchResults.Podcasts {
			items = append(items, types.SearchResultPodcastItem{SearchResultPodcast: pod})
		}
		for _, ep := range searchResults.Episodes {
			items = append(items, types.SearchResultEpisodeItem{SearchResultEpisode: ep})
		}

		searchResult := &types.SearchResponse{
			Items: items,
		}

		return types.SearchResultMsg{
			Result: searchResult,
			Err:    nil,
		}
	}
	m.SearchQuery = query
	return m, tea.Batch(loadingCmd, searchingCmd)
}

func (m Model) getArtistTracks(artistID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		artistSongs, err := m.YtMusicClient.GetArtistTopTracks(ctx, &musicpb.GetArtistTopTracksRequest{
			ChannelId: artistID,
		})
		if err != nil {
			slog.Error(err.Error())
			return types.UpdatePlaylistMsg{
				Playlist: nil,
				Err:      err,
			}
		}
		var tracks []*types.PlaylistTrackObject
		for _, track := range artistSongs.Tracks {
			tracks = append(tracks, &types.PlaylistTrackObject{
				Track: track,
			})
		}
		return types.UpdatePlaylistMsg{
			Playlist: tracks,
			Err:      nil,
		}
	}
}

func (m Model) getAlbumTracks(albumID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		albumResp, err := m.YtMusicClient.GetAlbumTracks(ctx, &musicpb.GetAlbumTracksRequest{
			BrowseId: albumID,
		})
		if err != nil {
			slog.Error(err.Error())
			return types.UpdatePlaylistMsg{
				Playlist: nil,
				Err:      err,
			}
		}
		var tracks []*types.PlaylistTrackObject
		for _, track := range albumResp.Tracks {
			tracks = append(tracks, &types.PlaylistTrackObject{
				Track: track,
			})
		}
		return types.UpdatePlaylistMsg{
			Playlist: tracks,
			Err:      nil,
		}
	}
}

func (m Model) getPlaylistItems(playlistID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		playlistItems, err := m.YtMusicClient.GetPlaylistItems(ctx, &musicpb.GetPlaylistItemsRequest{
			PlaylistId: playlistID,
			Limit:      100,
		})
		if err != nil {
			slog.Error(err.Error())
			return types.UpdatePlaylistMsg{
				Playlist: nil,
				Err:      err,
			}
		}
		var tracks []*types.PlaylistTrackObject
		for _, track := range playlistItems.Tracks {
			tracks = append(tracks, &types.PlaylistTrackObject{
				Track: track,
			})
		}
		return types.UpdatePlaylistMsg{
			Playlist:     tracks,
			Err:          nil,
			ShouldAppend: false,
		}
	}
}

func (m Model) PlaySelectedMusic(selectedMusic types.PlaylistTrackObject) (Model, tea.Cmd) {
	if selectedMusic.Track == nil {
		return m, nil
	}
	var cmds []tea.Cmd
	var artistNames []string
	for _, artist := range selectedMusic.Track.Artists {
		artistNames = append(artistNames, artist.Name)
	}
	if m.playbackCancel != nil {
		m.playbackCancel()
		m.playbackCancel = nil
	}
	if m.PlayerProcess != nil {
		err := m.PlayerProcess.Close()
		if err != nil {
			slog.Error(err.Error())
		}
		m.PlayerProcess = nil
	}

	playCtx, cancel := context.WithCancel(context.Background())
	m.playbackCancel = cancel

	cmd := youtube.SearchAndDownloadMusic(playCtx, selectedMusic.Track.VideoId, m.CoreDepsPath, func() (*musicpb.GetVideoStreamURLAndDurationResponse, error) {
		getStreamURLResponse, err := m.YtMusicClient.GetVideoStreamURLAndDuration(playCtx, &musicpb.GetVideoStreamURLAndDurationRequest{
			VideoId: selectedMusic.Track.VideoId,
		})
		if err != nil {
			return nil, err
		}
		return getStreamURLResponse, nil
	})

	m.CurrentLyrics = nil
	m.LyricsView.SetContent("  ⟳ Loading lyrics...")

	lyricsCmd := func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		lyricsResponse, err := m.YtMusicClient.GetLyrics(ctx, &musicpb.GetLyricsRequest{
			VideoId:    selectedMusic.Track.VideoId,
			Timestamps: true,
		})
		return types.LyricsMsg{
			LyricsResponse: lyricsResponse,
			Err:            err,
		}
	}

	cmds = append(cmds, cmd, lyricsCmd)
	metadata := getMusicMetadata(MusicMetadata{
		artistName: strings.Join(artistNames, ","),
		length:     int64(selectedMusic.Track.DurationSeconds * 1000),
		title:      selectedMusic.Track.Title,
	})

	if m.DBusConn != nil {
		dbusErr := m.DBusConn.Props.Set(
			"org.mpris.MediaPlayer2.Player",
			"Metadata",
			dbus.MakeVariant(metadata),
		)

		if dbusErr != nil {
			slog.Error(dbusErr.Error())
		}

		dbusErr = m.DBusConn.Props.Set("org.mpris.MediaPlayer2.Player",
			"PlaybackStatus",
			dbus.MakeVariant("Playing"),
		)

		if dbusErr != nil {
			slog.Error(dbusErr.Error())
		}
	}
	m.SelectedTrack = &SelectedTrack{
		isLiked: false,
		Track:   &selectedMusic,
	}

	return m, tea.Batch(cmds...)
}

func changeFocusMode(m *Model, shift bool) (Model, tea.Cmd) {
	var next, prev FocusedOn
	switch m.FocusedOn {
	case SideView:
		next, prev = MainView, Player
	case MainView:
		next = QueueList
		prev = SideView
	case QueueList:
		prev = MainView
		next = Player
	case Player:
		next, prev = SideView, QueueList
	default:
		if shift {
			items := m.SelectedPlayListItems.Items()
			if len(items) > 0 {
				m.FocusedOn = MainView
				m.SelectedPlayListItems.Select(len(items) - 1)
			} else {
				m.FocusedOn = SideView
			}
			return *m, nil
		}
		m.FocusedOn = SideView
		return *m, nil
	}

	if shift {
		m.FocusedOn = prev
	} else {
		m.FocusedOn = next
	}

	updateDelegate(m)
	return *m, nil
}

func updateDelegate(m *Model) {
	if m == nil {
		return
	}
	m.SelectedPlayListItems.SetDelegate(CustomDelegate{Model: m})
	if m.MusicQueueList != nil {
		m.MusicQueueList.SetDelegate(CustomDelegate{Model: m})
	}
	m.SideBarList.SetDelegate(CustomDelegate{Model: m})
	m.HomePageList.SetDelegate(CustomDelegate{Model: m})
	m.SearchResult.SetDelegate(CustomDelegate{Model: m})
	m.RelatedList.SetDelegate(CustomDelegate{Model: m})
}

func updateFocusedComponent(m *Model, msg tea.Msg, cmdsFromParent *[]tea.Cmd) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds = *cmdsFromParent
	cmds = append(cmds, cmd)
	switch m.FocusedOn {
	case SearchBar:
		m.Search.Focus()
		m.Search, cmd = m.Search.Update(msg)
		cmds = append(cmds, cmd)
	case SideView:
		m.Search.Blur()
		m.SideBarList, cmd = m.SideBarList.Update(msg)
		cmds = append(cmds, cmd)
	case QueueList:
		var cmd tea.Cmd
		showRelated := (m.RightColumnMode == RightColumnRelated || m.RightColumnMode == "") && len(m.RelatedList.Items()) > 0
		if showRelated {
			m.RelatedList, cmd = m.RelatedList.Update(msg)
		} else if m.MusicQueueList != nil {
			m.MusicQueueList.Model, cmd = m.MusicQueueList.Model.Update(msg)
		}
		cmds = append(cmds, cmd)
	case MainView:
		switch m.MainViewMode {
		case NormalMode:
			m.SelectedPlayListItems, cmd = m.SelectedPlayListItems.Update(msg)
			cmds = append(cmds, cmd)
		case SearchResultMode:
			m.SearchResult, cmd = m.SearchResult.Update(msg)
			cmds = append(cmds, cmd)
		}
	default:
	}
	return *m, tea.Batch(cmds...)
}

func SendLoadingCmd() tea.Cmd {
	return func() tea.Msg {
		return types.SearchingMsg{}
	}
}
