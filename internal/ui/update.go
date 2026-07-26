package ui

import (
	"context"
	"log/slog"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/godbus/dbus/v5"
	musicpb "github.com/kumneger0/clispot/gen"
	"github.com/kumneger0/clispot/internal/types"
	"github.com/kumneger0/clispot/internal/youtube"
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
	m.SearchResult = list.New(searchResponse.Items, CustomDelegate{Model: &m}, 10, 20)
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
					Track: types.MapSongToTrack(s),
				})
			}
			for _, p := range msg.Result.Playlists {
				items = append(items, types.MapPlaylistToPlaylist(p))
			}
			for _, al := range msg.Result.Albums {
				items = append(items, types.MapAlbumToAlbum(al))
			}
			for _, a := range msg.Result.Artists {
				items = append(items, types.Artist{
					ID:     a.ChannelId,
					Name:   a.Name,
					Images: types.MapThumbnailsToImages(a.Thumbnails),
				})
			}
			for _, c := range msg.Result.Channels {
				items = append(items, types.Artist{
					ID:     c.BrowseId,
					Name:   c.Name,
					Images: types.MapThumbnailsToImages(c.Thumbnails),
				})
			}
			m.SelectedPlayListItems = list.New(items, CustomDelegate{Model: &m}, 10, 20)
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
				Track: types.MapSongToTrack(track),
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
				Description: content.Description,
			})
		}
		m.HomePageList = list.New(items, CustomDelegate{Model: &m}, 10, 20)
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
		if m.SelectedTrack != nil && m.SelectedTrack.Track != nil && msg.VideoID != m.SelectedTrack.Track.Track.ID {
			_ = msg.Player.Close()
			return m, nil
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
	case types.SpotifySearchResultMsg:
		var alertCmd tea.Cmd
		if msg.Err != nil {
			alertCmd = m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			cmds = append(cmds, alertCmd)
		}
		if msg.Result != nil {
			m.FocusedOn = SearchResult
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

		m.HomePageList = list.New(items, CustomDelegate{Model: &m}, 10, 20)
		removeListDefaults(&m.HomePageList)
		m.HomePageList.Title = "Home"
		m.HomePageViewMode = HomePageSectionView
		m.MainViewMode = HomePageMode
		return m, nil
	case *types.UserFollowedArtistResponse:
		// TODO: implement later
	case types.DBusMessage:
		model, cmd := m.handleDbusMessage(msg.MessageType, cmds)
		m = model
		cmds = append(cmds, cmd)
	case types.LikeUnlikeTrackMsg:
		if msg.TrackID == m.SelectedTrack.Track.Track.ID {
			m.SelectedTrack.isLiked = msg.Like
		}
	case types.PlayedSecondsUpdateMsg:
		if m.SelectedTrack == nil || m.SelectedTrack.Track == nil {
			return m, nil
		}
		m.PlayedSeconds = msg.CurrentSeconds
		totalDurationInSeconds := m.SelectedTrack.Track.Track.DurationMS / 1000
		if totalDurationInSeconds > 0 && (float64(totalDurationInSeconds)-m.PlayedSeconds) < 1 {
			m.PlayedSeconds = 0
			model, cmd := m.handleMusicChange(true, false)
			m = model
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width - 4
		m.Height = msg.Height - 4
		dims := calculateLayoutDimensions(&m)
		m.LibraryWidth = dims.sidebarWidth
		m.MainViewWidth = dims.mainWidth
		m.PlayerSectionHeight = dims.inputHeight
		m.LyricsView.Width = dims.mainWidth
		m.LyricsView.Height = dims.contentHeight
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
		model, cmd := m.handleMusicChange(true, true)
		m = model
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case types.PreviousTrack:
		model, cmd := m.handleMusicChange(false, true)
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
	case "escape":
		if m.MainViewMode == HomePageMode && m.HomePageViewMode == HomePageContentView {
			var items []list.Item
			for i, section := range m.HomePageData.Sections {
				items = append(items, types.HomePageSectionItem{
					SectionTitle: section.Title,
					Index:        i,
				})
			}
			m.HomePageList = list.New(items, CustomDelegate{Model: &m}, 10, 20)
			removeListDefaults(&m.HomePageList)
			m.HomePageList.Title = "Home"
			m.HomePageViewMode = HomePageSectionView
			return m, nil
		}
	case "a":
		return m.addMusicToQueue()
	case "r":
		if m.FocusedOn == QueueList && m.MusicQueueList != nil {
			if len(m.MusicQueueList.Model.Items()) > 0 {
				m.MusicQueueList.Model.RemoveItem(m.MusicQueueList.GlobalIndex())
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
					TrackID: m.SelectedTrack.Track.Track.ID,
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
		return m.handleMusicChange(false, true)
	case "n":
		if m.FocusedOn != Player {
			return m, nil
		}
		return m.handleMusicChange(true, true)
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
					Track: types.MapSongToTrack(track),
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
					Track: types.MapSongToTrack(item),
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
	//TODO: implement lyrics later
	return m, nil
}

func (m Model) handleMusicChange(isForward, shouldRemoveTheCacheFile bool) (Model, tea.Cmd) {
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
		if playlistTrack.Track.ID == m.SelectedTrack.Track.Track.ID {
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
	if m.FocusedOn == MainView && m.MainViewMode == NormalMode {
		itemToAdd = m.SelectedPlayListItems.SelectedItem()
	} else if m.FocusedOn == SearchResult && m.MainViewMode == SearchResultMode {
		if len(m.SearchResult.Items()) > 0 {
			if track, ok := m.SearchResult.SelectedItem().(types.Track); ok {
				itemToAdd = types.PlaylistTrackObject{Track: track}
			}
		}
	} else {
		return m, nil
	}

	if _, ok := itemToAdd.(types.PlaylistTrackObject); !ok {
		return m, nil
	}

	if m.SelectedTrack != nil && m.SelectedTrack.Track != nil {
		currentlyPlayingTrackID = m.SelectedTrack.Track.Track.ID
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
		if playlistTrackObject.Track.ID == currentlyPlayingTrackID {
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
	if focusedOn == MainView && m.MainViewMode == NormalMode {
		return &m.SelectedPlayListItems
	}
	if focusedOn == QueueList && m.MusicQueueList != nil {
		return &m.MusicQueueList.Model
	}
	return nil
}

func (m Model) handleEnterKey() (Model, tea.Cmd) {
	if m.FocusedOn == SideView {
		if item, ok := m.SideBarList.SelectedItem().(types.SidebarItem); ok {
			newBreadcrumbItems := []types.Breadcrumb{{Name: item.Name, Icon: item.Icon}}
			m.BreadcrumbItems = newBreadcrumbItems
			if strings.ToLower(strings.Trim(item.Name, " ")) == "home" {
				homePageFeed := func() tea.Msg {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					homePage, err := m.YtMusicClient.GetHomePage(ctx, &musicpb.GetHomePageRequest{})
					if err != nil {
						slog.Error(err.Error())
						return types.HomePageResponseMsg{
							Response: homePage,
							Err:      err,
						}
					}
					return types.HomePageResponseMsg{
						Response: homePage,
						Err:      nil,
					}
				}
				return m, tea.Batch(SendLoadingCmd(), homePageFeed)
			}
			if strings.ToLower(strings.Trim(item.Name, " ")) == "library" {
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
		}
	}
	if m.FocusedOn == MainView || m.FocusedOn == QueueList {
		if m.MainViewMode == HomePageMode && m.HomePageViewMode == HomePageSectionView {
			listItemToChooseMusicFrom := getListItemForMusicToChoose(&m, m.FocusedOn)
			if listItemToChooseMusicFrom == nil {
				return m, nil
			}
			item, ok := listItemToChooseMusicFrom.SelectedItem().(types.HomePageSectionItem)
			if !ok {
				return m, nil
			}
			cmd := func() tea.Msg {
				return types.UpdateHomePageContentMsg{
					Item: item,
				}
			}
			newBreadcrumbItems := []types.Breadcrumb{{Name: item.SectionTitle, Icon: ""}}
			m.BreadcrumbItems = append(m.BreadcrumbItems, newBreadcrumbItems...)
			return m, cmd
		}

		if m.MainViewMode == HomePageMode && m.HomePageViewMode == HomePageContentView {
			item, ok := m.HomePageList.SelectedItem().(types.HomePageContentItem)
			if !ok {
				return m, nil
			}
			playlistDetailMsg := func() tea.Msg {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				playlistItems, err := m.YtMusicClient.GetPlaylistItems(ctx, &musicpb.GetPlaylistItemsRequest{
					PlaylistId: item.PlaylistID,
				})
				return types.PlaylistDetailMsg{
					Playlist: playlistItems,
					Err:      err,
				}
			}
			newBreadcrumbItems := []types.Breadcrumb{{Name: item.ItemTitle, Icon: ""}}
			m.BreadcrumbItems = append(m.BreadcrumbItems, newBreadcrumbItems...)
			return m, tea.Batch(SendLoadingCmd(), playlistDetailMsg)
		}

		listItemToChooseMusicFrom := getListItemForMusicToChoose(&m, m.FocusedOn)
		if listItemToChooseMusicFrom == nil {
			return m, nil
		}

		switch selectedItem := listItemToChooseMusicFrom.SelectedItem().(type) {
		case types.PlaylistTrackObject:
			var items []list.Item
			for _, item := range m.SelectedPlayListItems.Items() {
				playlistTrack, ok := item.(types.PlaylistTrackObject)
				if !ok {
					continue
				}
				playlistItem := types.PlaylistTrackObject{
					Track:         playlistTrack.Track,
					IsItFromQueue: true,
				}
				items = append(items, playlistItem)
			}
			if m.MusicQueueList == nil {
				return m, nil
			}
			m.MusicQueueList.Model.SetItems(items)
			m.MusicQueueList.Model.Select(m.MusicQueueList.GlobalIndex())
			return m.PlaySelectedMusic(selectedItem)

		case types.Playlist:
			loadingCmd := SendLoadingCmd()
			cmd := m.getPlaylistItems(selectedItem.ID)
			m.MainViewMode = NormalMode
			m.FocusedOn = MainView
			updateDelegate(&m)
			return m, tea.Batch(cmd, loadingCmd)

		case types.Album:
			loadingCmd := SendLoadingCmd()
			cmd := m.getAlbumTracks(selectedItem.ID)
			m.MainViewMode = NormalMode
			m.FocusedOn = MainView
			updateDelegate(&m)
			return m, tea.Batch(cmd, loadingCmd)

		case types.Artist:
			loadingCmd := SendLoadingCmd()
			cmd := m.getArtistTracks(selectedItem.ID)
			m.MainViewMode = NormalMode
			m.FocusedOn = MainView
			updateDelegate(&m)
			return m, tea.Batch(cmd, loadingCmd)
		}
	}

	if m.FocusedOn == SearchBar {
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
				return types.SpotifySearchResultMsg{
					Result: nil,
					Err:    err,
				}
			}

			var items []list.Item
			for _, s := range searchResults.Songs {
				track := types.MapSearchResultSongToTrack(s)
				items = append(items, track)
			}
			for _, a := range searchResults.Artists {
				artist := types.Artist{
					ID:     a.BrowseId,
					Name:   a.Name,
					Images: types.MapThumbnailsToImages(a.Thumbnails),
				}
				items = append(items, artist)
			}
			for _, p := range searchResults.Playlists {
				playlist := types.Playlist{
					ID:          p.BrowseId,
					Name:        p.Title,
					Description: p.ItemCount,
					Images:      types.MapThumbnailsToImages(p.Thumbnails),
					Author:      p.Author,
				}
				items = append(items, playlist)
			}
			for _, al := range searchResults.Albums {
				album := types.Album{
					ID:      al.BrowseId,
					Name:    al.Title,
					Artists: types.MapArtistsToArtists(al.Artists),
					Images:  types.MapThumbnailsToImages(al.Thumbnails),
					Year:    al.Year,
					Type:    al.Type,
				}
				items = append(items, album)
			}

			searchResult := &types.SearchResponse{
				Items: items,
			}

			return types.SpotifySearchResultMsg{
				Result: searchResult,
				Err:    nil,
			}
		}
		m.SearchQuery = query
		return m, tea.Batch(loadingCmd, searchingCmd)
	}

	if m.FocusedOn == SideView {
		m.PaginationInfo = nil
		loadingCmd := SendLoadingCmd()
		switch selectedItem := m.SideBarList.SelectedItem().(type) {
		case types.SidebarItem:
			m.MainViewMode = HomePageMode
			m.FocusedOn = MainView
			updateDelegate(&m)
			return m, nil
		case types.Playlist:
			cmd := m.getPlaylistItems(selectedItem.ID)
			return m, tea.Batch(loadingCmd, cmd)
		case types.Artist:
			cmd := m.getArtistTracks(selectedItem.ID)
			return m, tea.Batch(loadingCmd, cmd)
		case types.UserSavedTracksListItem:
			cmd := m.getUserSavedTracks()
			return m, tea.Batch(loadingCmd, cmd)
		}
	}
	if m.FocusedOn == MainView && m.MainViewMode == HomePageMode {
		switch m.HomePageViewMode {
		case HomePageSectionView:
			selectedItem, ok := m.HomePageList.SelectedItem().(types.HomePageSectionItem)
			if !ok {
				slog.Error("failed to cast the selected item to types.HomePageSectionItem")
				return m, nil
			}
			if m.HomePageData != nil && selectedItem.Index < len(m.HomePageData.Sections) {
				section := m.HomePageData.Sections[selectedItem.Index]
				var items []list.Item
				for _, content := range section.Contents {
					items = append(items, types.HomePageContentItem{
						ItemTitle:   content.Title,
						PlaylistID:  content.PlaylistId,
						Description: content.Description,
					})
				}
				m.HomePageList = list.New(items, CustomDelegate{Model: &m}, 10, 20)
				removeListDefaults(&m.HomePageList)
				m.HomePageList.Title = section.Title
				m.HomePageViewMode = HomePageContentView
				return m, nil
			}
		case HomePageContentView:
			selectedItem, ok := m.HomePageList.SelectedItem().(types.HomePageContentItem)
			if !ok {
				slog.Error("failed to cast the selected item to types.HomePageContentItem")
				return m, nil
			}
			if selectedItem.PlaylistID != "" {
				loadingCmd := SendLoadingCmd()
				cmd := m.getPlaylistItems(selectedItem.PlaylistID)
				m.MainViewMode = NormalMode
				m.FocusedOn = MainView
				updateDelegate(&m)
				return m, tea.Batch(cmd, loadingCmd)
			}
		}
	}
	if m.FocusedOn == SearchResult {
		if selectedItem, ok := m.SearchResult.SelectedItem().(types.SearchResultItem); ok {
			switch selectedItem.Kind() {
			case types.SearchResultTrack:
				track, ok := selectedItem.(types.Track)
				if !ok {
					slog.Error("failed to cast the selected item to types.Track")
					return m, nil
				}
				return m.PlaySelectedMusic(types.PlaylistTrackObject{
					Track: track,
				})
			case types.SearchResultPlaylist:
				playlist, ok := selectedItem.(types.Playlist)
				if !ok {
					slog.Error("failed to cast the selected item to types.Playlist")
					return m, nil
				}
				loadingCmd := SendLoadingCmd()
				cmd := m.getPlaylistItems(playlist.ID)
				m.MainViewMode = NormalMode
				m.FocusedOn = MainView
				updateDelegate(&m)
				return m, tea.Batch(cmd, loadingCmd)
			case types.SearchResultArtist:
				artist, ok := selectedItem.(types.Artist)
				if !ok {
					slog.Error("failed to cast the selected item to types.Artist")
					return m, nil
				}
				loadingCmd := SendLoadingCmd()
				cmd := m.getArtistTracks(artist.ID)
				m.MainViewMode = NormalMode
				m.FocusedOn = MainView
				updateDelegate(&m)
				return m, tea.Batch(cmd, loadingCmd)
			case types.SearchResultAlbum:
				album, ok := selectedItem.(types.Album)
				if !ok {
					slog.Error("failed to cast the selected item to types.Album")
					return m, nil
				}
				loadingCmd := SendLoadingCmd()
				cmd := m.getAlbumTracks(album.ID)
				m.MainViewMode = NormalMode
				m.FocusedOn = MainView
				updateDelegate(&m)
				return m, tea.Batch(cmd, loadingCmd)
			}
		}
	}
	return m, nil
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
				Track: types.MapSongToTrack(track),
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
				Track: types.MapSongToTrack(track),
			})
		}
		return types.UpdatePlaylistMsg{
			Playlist: tracks,
			Err:      nil,
		}
	}
}

func (m Model) getUserSavedTracks() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		savedTracks, err := m.YtMusicClient.GetUserSavedTracks(ctx, &musicpb.GetUserSavedTracksRequest{
			Limit: 100,
		})
		if err != nil {
			slog.Error(err.Error())
			return types.UpdatePlaylistMsg{
				Playlist: nil,
				Err:      err,
			}
		}
		var tracks []*types.PlaylistTrackObject
		for _, track := range savedTracks.Tracks {
			tracks = append(tracks, &types.PlaylistTrackObject{
				Track: types.MapSongToTrack(track),
			})
		}
		return types.UpdatePlaylistMsg{
			Playlist:     tracks,
			Err:          nil,
			ShouldAppend: false,
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
				Track: types.MapSongToTrack(track),
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

	cmd := youtube.SearchAndDownloadMusic(playCtx, selectedMusic.Track.ID, m.CoreDepsPath, func() (string, error) {
		getStreamURLResponse, err := m.YtMusicClient.GetVideoStreamURL(context.Background(), &musicpb.GetVideoStreamURLRequest{
			VideoId: selectedMusic.Track.ID,
		})
		if err != nil {
			return "", err
		}
		return getStreamURLResponse.Url, nil
	})

	cmds = append(cmds, cmd)
	metadata := getMusicMetadata(MusicMetadata{
		artistName: strings.Join(artistNames, ","),
		length:     int64(selectedMusic.Track.DurationMS),
		title:      selectedMusic.Track.Name,
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

	if m.MainViewMode == LyricsMode {
		model, cmd := m.getMusicLyrics(m.SelectedTrack)
		m = model
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func changeFocusMode(m *Model, shift bool) (Model, tea.Cmd) {
	var next, prev FocusedOn
	switch m.FocusedOn {
	case SideView:
		next, prev = MainView, Player
	case MainView:
		if m.MainViewMode == SearchResultMode {
			next = SearchResult
		} else {
			next = QueueList
		}
		prev = SideView
	case SearchResult:
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
		if m.MusicQueueList != nil {
			m.MusicQueueList.Model, cmd = m.MusicQueueList.Model.Update(msg)
		}
		cmds = append(cmds, cmd)
	case MainView:
		switch m.MainViewMode {
		case NormalMode:
			m.SelectedPlayListItems, cmd = m.SelectedPlayListItems.Update(msg)
			cmds = append(cmds, cmd)
		}
	case SearchResult:
		m.SearchResult, cmd = m.SearchResult.Update(msg)
		cmds = append(cmds, cmd)
	default:
	}
	return *m, tea.Batch(cmds...)
}

func SendLoadingCmd() tea.Cmd {
	return func() tea.Msg {
		return types.SearchingMsg{}
	}
}
