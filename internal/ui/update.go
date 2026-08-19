package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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
	artistName []string
	title      string
	length     int64
	albumName  string
}

func getMusicMetadata(music MusicMetadata) map[string]any {
	var metadata = map[string]any{
		"mpris:trackid": "/org/mpris/MediaPlayer2/" + music.title,
		"xesam:title":   music.title,
		"xesam:artist":  music.artistName,
		"xesam:length":  music.length,
		"xesam:album":   music.albumName,
	}
	return metadata
}

func (m Model) getSearchResultModel(searchResponse *types.SearchResponse) (Model, tea.Cmd) {
	dims := CalculateLayoutDimensions(&m)
	m.SearchResult = list.New(searchResponse.Items, CustomDelegate{Model: &m}, dims.MainWidth, dims.ContentHeight-4)
	m.SearchResult.SetShowTitle(false)
	removeListDefaults(&m.SearchResult)
	return m, nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case types.WatchPlaylistItemsMsg:
		if msg.Err != nil {
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		if msg.WatchPlaylistItems == nil {
			slog.Error("Failed to fetch watch playlist items")
			return m, nil
		}
		var currentlyPlayingTrackID string
		if m.SelectedTrack != nil && m.SelectedTrack.Track != nil {
			currentlyPlayingTrackID = m.SelectedTrack.Track.VideoId
		}
		if currentlyPlayingTrackID != "" && currentlyPlayingTrackID != msg.SourceID {
			return m, nil
		}
		for _, song := range msg.WatchPlaylistItems.Tracks {
			if song.VideoId == currentlyPlayingTrackID {
				continue
			}
			m.PlaybackContext = append(m.PlaybackContext, &types.PlaylistTrackObject{
				Track: song,
			})
		}
		m.PlaybackContextName = ""
		return m, m.SyncQueueList()
	case types.CreatePlaylistMsg:
		cmd := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			response, err := m.YtMusicClient.CreatePlaylist(ctx, &musicpb.CreatePlaylistRequest{
				Title:         msg.Title,
				Description:   msg.Description,
				PrivacyStatus: msg.PrivacyStatus,
			})
			if err != nil {
				slog.Error(err.Error())
				return types.CreatePlaylistResponseMsg{Success: false, Err: err}
			}
			if response == nil {
				err := errors.New("failed to create playlist: nil response")
				slog.Error(err.Error())
				return types.CreatePlaylistResponseMsg{Success: false, Err: err}
			}
			if response.PlaylistId == "" {
				err := errors.New(response.GetError())
				slog.Error(err.Error())
				return types.CreatePlaylistResponseMsg{Success: false, Err: err}
			}
			return types.CreatePlaylistResponseMsg{Success: true, PlaylistID: response.PlaylistId}
		}
		return m, cmd
	case types.CreatePlaylistResponseMsg:
		if msg.Success {
			alertCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, "Playlist created successfully!")
			return m, alertCmd
		} else if msg.Err != nil {
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		return m, nil
	case types.AddToPlaylistMsg:
		addCmd := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			response, err := m.YtMusicClient.AddPlaylistItems(ctx, &musicpb.AddPlaylistItemsRequest{
				PlaylistId: msg.PlaylistID,
				VideoIds:   []string{msg.TrackID},
				Duplicates: msg.Duplicates,
			})

			isDup := false
			if err != nil || response == nil || !response.Success {
				itemsRes, itemsErr := m.YtMusicClient.GetPlaylistItems(ctx, &musicpb.GetPlaylistItemsRequest{
					PlaylistId: msg.PlaylistID,
					Limit:      200,
				})
				if itemsErr == nil && itemsRes != nil {
					for _, t := range itemsRes.Tracks {
						if t.VideoId == msg.TrackID {
							isDup = true
							break
						}
					}
				}
			}

			if !msg.Duplicates && isDup {
				return types.PromptDuplicateConfirmMsg{
					PlaylistID:   msg.PlaylistID,
					PlaylistName: msg.PlaylistName,
					TrackID:      msg.TrackID,
					TrackTitle:   msg.TrackTitle,
				}
			}

			if err != nil {
				slog.Error(err.Error())
				return types.AddToPlaylistResponseMsg{
					PlaylistID:   msg.PlaylistID,
					PlaylistName: msg.PlaylistName,
					TrackID:      msg.TrackID,
					TrackTitle:   msg.TrackTitle,
					Success:      false,
					Err:          err,
				}
			}
			if response == nil || !response.Success {
				errStr := "Failed to add song to playlist"
				if response != nil && response.Error != "" {
					errStr = response.Error
				}
				return types.AddToPlaylistResponseMsg{
					PlaylistID:   msg.PlaylistID,
					PlaylistName: msg.PlaylistName,
					TrackID:      msg.TrackID,
					TrackTitle:   msg.TrackTitle,
					Success:      false,
					Err:          fmt.Errorf("%s", errStr),
				}
			}
			return types.AddToPlaylistResponseMsg{
				PlaylistID:   msg.PlaylistID,
				PlaylistName: msg.PlaylistName,
				TrackID:      msg.TrackID,
				TrackTitle:   msg.TrackTitle,
				Status:       response.Status,
				Success:      true,
			}
		}
		return m, addCmd
	case types.AddToPlaylistResponseMsg:
		if msg.Success {
			alertCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, fmt.Sprintf("Added \"%s\" to %s", msg.TrackTitle, msg.PlaylistName))
			return m, alertCmd
		} else if msg.Err != nil {
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		return m, nil
	case types.RemoveFromPlaylistMsg:
		removeCmd := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result, err := m.YtMusicClient.GetPlaylistItems(ctx, &musicpb.GetPlaylistItemsRequest{
				PlaylistId: msg.PlaylistID,
				Limit:      200,
			})
			if err != nil {
				return types.RemoveFromPlaylistResponseMsg{
					PlaylistID:   msg.PlaylistID,
					PlaylistName: msg.PlaylistName,
					TrackID:      msg.TrackID,
					TrackTitle:   msg.TrackTitle,
					Success:      false,
					Err:          err,
				}
			}

			var setVideoID *string
			for _, track := range result.Tracks {
				if track.VideoId == msg.TrackID {
					setVideoID = &track.SetVideoId
					break
				}
			}

			if setVideoID == nil {
				return types.RemoveFromPlaylistResponseMsg{
					PlaylistID:   msg.PlaylistID,
					PlaylistName: msg.PlaylistName,
					TrackID:      msg.TrackID,
					TrackTitle:   msg.TrackTitle,
					Success:      false,
					Err:          errors.New("Failed to Find the track in this playlist"),
				}
			}
			response, err := m.YtMusicClient.RemovePlaylistItems(ctx, &musicpb.RemovePlaylistItemsRequest{
				PlaylistId: msg.PlaylistID,
				Videos: []*musicpb.PlaylistItemRef{
					{
						VideoId:    msg.TrackID,
						SetVideoId: *setVideoID,
					},
				},
			})

			if err != nil {
				slog.Error(err.Error())
				return types.RemoveFromPlaylistResponseMsg{
					PlaylistID:   msg.PlaylistID,
					PlaylistName: msg.PlaylistName,
					TrackID:      msg.TrackID,
					TrackTitle:   msg.TrackTitle,
					Success:      false,
					Err:          err,
				}
			}
			if response == nil || !response.Success {
				errStr := "Failed to remove song from playlist"
				if response != nil && response.Error != "" {
					errStr = response.Error
				}
				return types.RemoveFromPlaylistResponseMsg{
					PlaylistID:   msg.PlaylistID,
					PlaylistName: msg.PlaylistName,
					TrackID:      msg.TrackID,
					TrackTitle:   msg.TrackTitle,
					Success:      false,
					Err:          fmt.Errorf("%s", errStr),
				}
			}
			return types.RemoveFromPlaylistResponseMsg{
				PlaylistID:   msg.PlaylistID,
				PlaylistName: msg.PlaylistName,
				TrackID:      msg.TrackID,
				TrackTitle:   msg.TrackTitle,
				Success:      true,
			}
		}
		return m, removeCmd
	case types.RemoveFromPlaylistResponseMsg:
		if msg.Success {
			alertCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, fmt.Sprintf("Removed \"%s\" from %s", msg.TrackTitle, msg.PlaylistName))
			return m, alertCmd
		} else if msg.Err != nil {
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		return m, nil
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
			m.SelectedPlayListItems = list.New(items, CustomDelegate{Model: &m}, dims.MainWidth, dims.ContentHeight-4)
			m.SelectedPlayListItems.Title = "Library"
			m.SelectedPlayListItems.SetShowTitle(false)
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
		return m, tea.Batch(SendLoadingCmd(), homePageFeed)
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
				ItemTitle:       content.Title,
				PlaylistID:      content.PlaylistId,
				VideoID:         content.VideoId,
				BrowseID:        content.BrowseId,
				ContentType:     content.ContentType,
				Description:     content.Description,
				Artists:         content.Artists,
				DurationSeconds: content.DurationSeconds,
			})
		}
		dims := CalculateLayoutDimensions(&m)
		m.HomePageList = list.New(items, CustomDelegate{Model: &m}, dims.MainWidth, dims.ContentHeight-4)
		m.HomePageList.SetShowTitle(false)
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
		if m.SelectedTrack == nil || m.SelectedTrack.Track == nil {
			_ = msg.Player.Close()
			return m, nil
		}
		if msg.VideoID != m.SelectedTrack.Track.VideoId {
			_ = msg.Player.Close()
			return m, nil
		}
		if m.SelectedTrack.Track.DurationSeconds == 0 && msg.Duration != "" {
			if duration, err := strconv.ParseInt(msg.Duration, 10, 64); err == nil {
				m.SelectedTrack.Track.DurationSeconds = int32(duration)
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
		m.PlayerProcess = msg.Player
		return m, likedCmd
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
			return m, alertCmd
		}
		if msg.Result != nil {
			m.FocusedOn = MainView
			m.MainViewMode = SearchResultMode
			model, cmd := m.getSearchResultModel(msg.Result)
			m = model
			m.IsSearchLoading = false
			m.Search.Blur()
			return m, cmd
		}
	case types.HomePageResponseMsg:
		var alertCmd tea.Cmd
		m.IsSearchLoading = false
		if msg.Err != nil {
			slog.Error(msg.Err.Error())
			alertCmd = m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
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
		m.HomePageList = list.New(items, CustomDelegate{Model: &m}, dims.MainWidth, dims.ContentHeight-4)
		m.HomePageList.SetShowTitle(false)
		removeListDefaults(&m.HomePageList)
		m.HomePageList.Title = "Home"
		m.HomePageViewMode = HomePageSectionView
		m.MainViewMode = HomePageMode
		return m, nil
	case types.DBusMessage:
		model, cmd := m.handleDbusMessage(msg.MessageType)
		m = model
		return m, cmd
	case types.LikeUnlikeTrackResponseMsg:
		if msg.Err != nil {
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		if m.SelectedTrack != nil && m.SelectedTrack.Track != nil && m.SelectedTrack.Track.VideoId == msg.TrackID {
			m.SelectedTrack.isLiked = msg.Liked
		}
	case types.PlayedSecondsUpdateMsg:
		if m.SelectedTrack == nil || m.SelectedTrack.Track == nil {
			return m, nil
		}
		oldSec := int(m.PlayedSeconds)
		m.PlayedSeconds = msg.CurrentSeconds
		if int(m.PlayedSeconds) == oldSec {
			return m, nil
		}

		if m.CurrentLyrics != nil {
			m.updateLyricsView()
		}
		totalDurationInSeconds := m.SelectedTrack.Track.DurationSeconds
		if totalDurationInSeconds > 0 && (float64(totalDurationInSeconds)-m.PlayedSeconds) < 1 {
			model, cmd := m.handleMusicChange(true)
			m = model
			return m, cmd
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
		m.UpdateListDimensions()
		return m, nil
	case types.UpdatePlaylistMsg:
		m.IsSearchLoading = false
		if msg.Playlist != nil {
			var playListItemSongs []list.Item
			for _, item := range msg.Playlist {
				playListItemSongs = append(playListItemSongs, *item)
			}
			m.MainViewMode = NormalMode
			m.IsSearchLoading = false
			var cmd tea.Cmd
			if m.IsOnPagination {
				m.IsOnPagination = false
				existingItems := m.SelectedPlayListItems.Items()
				allItems := append(existingItems, playListItemSongs...)
				cmd = m.SelectedPlayListItems.SetItems(allItems)
			} else {
				cmd = m.SelectedPlayListItems.SetItems(playListItemSongs)
			}
			if m.PendingContextName != "" {
				m.SelectedPlayListItems.Title = m.PendingContextName
			}
			if msg.PaginationInfo != nil {
				m.PaginationInfo = msg.PaginationInfo
			} else {
				m.PaginationInfo = nil
			}
			return m, cmd
		}
		if msg.Err != nil {
			m.IsOnPagination = false
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
		}
		dims := CalculateLayoutDimensions(&m)
		m.SelectedPlayListItems = list.New([]list.Item{}, CustomDelegate{Model: &m}, dims.MainWidth, dims.ContentHeight-4)
		m.SelectedPlayListItems.SetShowTitle(false)
		removeListDefaults(&m.SelectedPlayListItems)
	case types.RelatedSongsMsg:
		if msg.Err != nil {
			slog.Error(msg.Err.Error())
			alertCmd := m.Alert.NewAlertCmd(bubbleup.ErrorKey, msg.Err.Error())
			return m, alertCmd
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
		m.RelatedList.Title = "Related"
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
		m.MainViewMode = LyricsMode
		m.CurrentLyrics = msg.LyricsResponse
		m.updateLyricsView()
		return m, nil
	case tea.KeyMsg:
		model, cmd := m.handleKeyPress(msg)
		m = model
		if cmd != nil {
			var searchCmd tea.Cmd
			if m.FocusedOn == SearchBar {
				m.Search, searchCmd = m.Search.Update(msg)
			}
			return m, tea.Batch(cmd, searchCmd)
		}

	case tea.MouseMsg:
		x := msg.X
		y := msg.Y
		if x > m.LibraryWidth && x <= (m.LibraryWidth+m.MainViewWidth) && y <= (m.Height-m.PlayerSectionHeight) {
			if m.MainViewMode == LyricsMode && m.FocusedOn != SearchBar {
				lyricsModel, cmd := m.LyricsView.Update(msg)
				m.LyricsView = lyricsModel
				return m, cmd
			}
		}

	default:
	}
	model, cmd := updateFocusedComponent(&m, msg)
	m = model
	outAlert, outCmd := m.Alert.Update(msg)
	m.Alert = outAlert.(bubbleup.AlertModel)
	return m, tea.Batch(outCmd, cmd)
}

func (m Model) handleDbusMessage(msg types.MessageType) (Model, tea.Cmd) {
	switch msg {
	case types.NextTrack:
		model, cmd := m.handleMusicChange(true)
		m = model
		return m, cmd
	case types.PreviousTrack:
		model, cmd := m.handleMusicChange(false)
		m = model
		return m, cmd
	case types.PlayPause:
		model, cmd := m.HandleMusicPausePlay()
		m = model
		return m, cmd
	}
	return m, nil
}

func (m Model) handlePagination(listModel *list.Model, currentIndex *int) (Model, tea.Cmd) {
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
		paginationInfo = m.PaginationInfo
		return m, getNextPageItems(&m, paginationInfo)
	}
	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+t":
		openCreatePlaylistModal := func() tea.Msg {
			return types.OpenModalMsg{
				ModalType: types.ModalTypeCreatePlaylist,
			}
		}
		return m, openCreatePlaylistModal
	case "ctrl+p":
		trackID, trackTitle := m.getCurrentSelectedTrack()
		if trackID == "" {
			return m, nil
		}
		openAddToPlaylistCmd := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			userPlaylists, err := m.YtMusicClient.GetUserPlaylists(ctx, &musicpb.GetUserPlaylistsRequest{Limit: 100})
			if err != nil {
				return types.OpenAddToPlaylistModalMsg{
					TrackID:    trackID,
					TrackTitle: trackTitle,
					Err:        err,
				}
			}
			var pls []*musicpb.Playlist
			if userPlaylists != nil {
				pls = userPlaylists.Playlists
			}
			return types.OpenAddToPlaylistModalMsg{
				TrackID:    trackID,
				TrackTitle: trackTitle,
				Playlists:  pls,
			}
		}
		return m, tea.Batch(
			tea.Sequence(
				func() tea.Msg {
					return types.OpenAddToPlaylistLoadingMsg{
						TrackID:    trackID,
						TrackTitle: trackTitle,
					}
				},
				func() tea.Msg {
					return types.OpenModalMsg{ModalType: types.ModalTypePlaylistManagement}
				},
			),
			openAddToPlaylistCmd,
		)
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
		return m.handlePagination(listModel, nil)
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
			m.HomePageList = list.New(items, CustomDelegate{Model: &m}, dims.MainWidth, dims.ContentHeight-4)
			m.HomePageList.SetShowTitle(false)
			removeListDefaults(&m.HomePageList)
			m.HomePageList.Title = "Home"
			m.HomePageViewMode = HomePageSectionView
			return m, nil
		}
	case "a":
		return m.addMusicToQueue()
	case "r":
		var cmd tea.Cmd
		showRelated := (m.RightColumnMode == RightColumnRelated || m.RightColumnMode == "") && len(m.RelatedList.Items()) > 0
		if m.FocusedOn == QueueList && !showRelated {
			shouldIRemoveFromPlaybackContext := true
			if m.Queue != nil && m.Queue.Len() > 0 {
				selectedIdx := m.QueueList.Index()
				userQueueTracks := m.Queue.AllTracks()
				trackIdx := selectedIdx - 1
				if trackIdx >= 0 && trackIdx < len(userQueueTracks) {
					shouldIRemoveFromPlaybackContext = false
					m.Queue.RemoveTrackAtIndex(trackIdx)
				}
				cmd = m.SyncQueueList()
			}

			if shouldIRemoveFromPlaybackContext {
				itemIndex := -1
				if selectedTrack, ok := m.QueueList.SelectedItem().(types.PlaylistTrackObject); ok {
					if selectedTrack.Track != nil {
						for i, track := range m.PlaybackContext {
							if track != nil && track.Track != nil &&
								track.Track.VideoId == selectedTrack.Track.VideoId {
								itemIndex = i
								break
							}
						}
					}
				}
				if itemIndex != -1 && itemIndex < len(m.PlaybackContext) {
					itemsToKeep := m.PlaybackContext[:itemIndex]
					if itemIndex+1 < len(m.PlaybackContext) {
						itemsToKeep = append(itemsToKeep, m.PlaybackContext[itemIndex+1:]...)
					}
					m.PlaybackContext = itemsToKeep
					cmd = m.SyncQueueList()
				}
			}
			return m, cmd
		}
	case "ctrl+l":
		if m.MainViewMode == LyricsMode {
			m.MainViewMode = NormalMode
			return m, nil
		}

		return m.getMusicLyrics()
	case "l":
		if m.SelectedTrack != nil && m.SelectedTrack.Track != nil {
			trackID := m.SelectedTrack.Track.VideoId
			shouldRemove := m.SelectedTrack.isLiked
			targetLikedState := !shouldRemove

			cmd := func() tea.Msg {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				_, err := m.YtMusicClient.SaveRemoveTrack(ctx, &musicpb.SaveRemoveTrackRequest{
					VideoIds: []string{trackID},
					IsRemove: shouldRemove,
				})

				if err != nil {
					slog.Error(err.Error())
				}
				return types.LikeUnlikeTrackResponseMsg{
					TrackID: trackID,
					Liked:   targetLikedState,
					Err:     err,
				}
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

func getNextPageItems(m *Model, paginationInfo *types.PaginationInfo) tea.Cmd {
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
				Playlist:       tracks,
				Err:            nil,
				PaginationInfo: paginationInfo,
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
				Playlist:       playlistItems,
				Err:            nil,
				PaginationInfo: paginationInfo,
			}
		}
	}
	return nil
}

type lrclibQuery struct {
	videoID     string
	trackName   string
	albumName   string
	artistName  string
	durationSec int32
}

func (m Model) getMusicLyrics() (Model, tea.Cmd) {
	m.MainViewMode = LyricsMode
	m.FocusedOn = MainView
	if m.SelectedTrack == nil || m.SelectedTrack.Track == nil {
		noLyricsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#71717A")).Italic(true)
		m.LyricsView.SetContent(noLyricsStyle.Render("No track selected."))
		return m, nil
	}

	m.LyricsView.SetContent("  ⟳ Loading lyrics...")
	q := lrclibQuery{
		videoID:     m.SelectedTrack.Track.VideoId,
		trackName:   m.SelectedTrack.Track.Title,
		albumName:   m.SelectedTrack.Track.Album,
		durationSec: m.SelectedTrack.Track.DurationSeconds,
	}
	if len(m.SelectedTrack.Track.Artists) > 0 {
		q.artistName = m.SelectedTrack.Track.Artists[0].Name
	}

	ytClient := m.YtMusicClient

	lyricsCmd := func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var backendResp *musicpb.GetLyricsResponse
		lyricsResponse, err := ytClient.GetLyrics(ctx, &musicpb.GetLyricsRequest{
			VideoId:    q.videoID,
			Timestamps: true,
		})
		if err == nil && lyricsResponse != nil {
			backendResp = lyricsResponse
		}

		if err != nil {
			slog.Debug("backend GetLyrics failed", "videoId", q.videoID, "err", err)
		}

		if backendResp != nil && backendResp.HasTimestamps && len(backendResp.Lines) > 0 {
			return types.LyricsMsg{LyricsResponse: backendResp}
		}

		lrclibResp := fetchLrclib(ctx, q)

		if lrclibResp != nil && lrclibResp.SyncedLyrics != "" {
			lines := parseSyncedLyrics(lrclibResp.SyncedLyrics)
			if len(lines) > 0 {
				return types.LyricsMsg{
					LyricsResponse: &musicpb.GetLyricsResponse{
						HasTimestamps: true,
						Lines:         lines,
						Source:        "lrclib.net",
					},
				}
			}
		}

		if backendResp != nil && backendResp.Lyrics != "" {
			return types.LyricsMsg{LyricsResponse: backendResp}
		}

		if lrclibResp != nil && lrclibResp.PlainLyrics != "" {
			return types.LyricsMsg{
				LyricsResponse: &musicpb.GetLyricsResponse{
					Lyrics: lrclibResp.PlainLyrics,
					Source: "lrclib.net",
				},
			}
		}

		return types.LyricsMsg{LyricsResponse: nil, Err: err}
	}

	return m, lyricsCmd
}

type lrclibResponse struct {
	SyncedLyrics string `json:"syncedLyrics"`
	PlainLyrics  string `json:"plainLyrics"`
}

func fetchLrclib(ctx context.Context, q lrclibQuery) *lrclibResponse {
	params := url.Values{}
	params.Add("track_name", q.trackName)
	if q.albumName != "" {
		params.Add("album_name", q.albumName)
	}
	if q.artistName != "" {
		params.Add("artist_name", q.artistName)
	}
	if q.durationSec > 0 {
		params.Add("duration", strconv.FormatInt(int64(q.durationSec), 10))
	}

	reqCtx, reqCancel := context.WithTimeout(ctx, 5*time.Second)
	defer reqCancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", "https://lrclib.net/api/get?"+params.Encode(), nil)
	if err != nil {
		slog.Debug("lrclib NewRequest error", "err", err)
		return nil
	}
	req.Header.Set("User-Agent", "ytmusic-tui/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Debug("lrclib request failed", "err", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("lrclib non-200 status code", "status", resp.StatusCode)
		return nil
	}

	limitedReader := io.LimitReader(resp.Body, 512*1024)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		slog.Debug("lrclib read body error", "err", err)
		return nil
	}

	var result lrclibResponse
	if err := json.Unmarshal(data, &result); err != nil {
		slog.Debug("lrclib unmarshal error", "err", err)
		return nil
	}
	return &result
}

func parseSyncedLyrics(synced string) []*musicpb.LyricLine {
	var lines []*musicpb.LyricLine
	for _, rawLine := range strings.Split(synced, "\n") {
		rawLine = strings.TrimSpace(rawLine)
		if rawLine == "" || !strings.HasPrefix(rawLine, "[") {
			continue
		}
		closeBracket := strings.Index(rawLine, "]")
		if closeBracket == -1 {
			continue
		}
		timestamp := rawLine[1:closeBracket]
		text := strings.TrimSpace(rawLine[closeBracket+1:])

		ms, ok := parseLRCTimestamp(timestamp)
		if !ok {
			continue
		}
		lines = append(lines, &musicpb.LyricLine{
			Text:      text,
			StartTime: ms,
		})
	}
	for i := 0; i < len(lines)-1; i++ {
		lines[i].EndTime = lines[i+1].StartTime
	}
	return lines
}

func parseLRCTimestamp(ts string) (int32, bool) {
	parts := strings.Split(ts, ":")
	if len(parts) != 2 {
		return 0, false
	}
	if strings.HasPrefix(parts[0], "-") {
		return 0, false
	}
	minutes, err := strconv.Atoi(parts[0])
	if err != nil || minutes < 0 {
		return 0, false
	}
	secParts := strings.Split(parts[1], ".")
	if len(secParts) > 2 {
		return 0, false
	}
	if strings.HasPrefix(secParts[0], "-") {
		return 0, false
	}
	seconds, err := strconv.Atoi(secParts[0])
	if err != nil || seconds < 0 || seconds >= 60 {
		return 0, false
	}
	var fracMs int
	if len(secParts) == 2 {
		frac := secParts[1]
		if len(frac) == 0 || len(frac) > 3 || strings.HasPrefix(frac, "-") {
			return 0, false
		}
		fracVal, err := strconv.Atoi(frac)
		if err != nil || fracVal < 0 {
			return 0, false
		}
		switch len(frac) {
		case 1:
			fracMs = fracVal * 100
		case 2:
			fracMs = fracVal * 10
		case 3:
			fracMs = fracVal
		}
	}

	return int32(minutes*60000 + seconds*1000 + fracMs), true
}

func (m *Model) toggleRightColumnMode() {
	if m.RightColumnMode == RightColumnRelated || m.RightColumnMode == "" {
		m.RightColumnMode = RightColumnQueue
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
	var track *types.PlaylistTrackObject
	fromHistory := false

	if isForward {
		if m.Queue != nil && m.Queue.Len() > 0 {
			track = m.Queue.PopFirst()
		} else if len(m.PlayHistory) > 0 && m.PlayHistoryIndex < len(m.PlayHistory)-1 {
			m.PlayHistoryIndex++
			track = m.PlayHistory[m.PlayHistoryIndex]
			fromHistory = true
		} else if len(m.PlaybackContext) > 0 {
			m.PlaylistContextIndex = (m.PlaylistContextIndex + 1) % len(m.PlaybackContext)
			track = m.PlaybackContext[m.PlaylistContextIndex]
		} else if len(m.PlayHistory) > 0 {
			m.PlayHistoryIndex = 0
			track = m.PlayHistory[m.PlayHistoryIndex]
			fromHistory = true
		}

		if !fromHistory && m.PlayedSeconds > 30 &&
			m.SelectedTrack != nil && m.SelectedTrack.Track != nil {
			appendToPlayHistory(&m, &m.SelectedTrack.PlaylistTrackObject)
		}
	} else {
		if len(m.PlayHistory) > 0 && m.PlayHistoryIndex > 0 {
			m.PlayHistoryIndex--
			track = m.PlayHistory[m.PlayHistoryIndex]
		}
	}

	if track != nil {
		m.PlayedSeconds = 0
		model, cmd := m.PlaySelectedMusic(*track)
		m = model
		return m, cmd
	}

	return m, nil
}

func extractTrackObject(item list.Item) *types.PlaylistTrackObject {
	if item == nil {
		return nil
	}
	switch v := item.(type) {
	case types.PlaylistTrackObject:
		if v.Track != nil {
			return &v
		}
	case types.SongItem:
		if v.Song != nil {
			return &types.PlaylistTrackObject{Track: v.Song}
		}
	case types.SearchResultSongItem:
		if v.SearchResultSong != nil {
			song := &musicpb.Song{
				VideoId:         v.VideoId,
				Title:           v.Title,
				Artists:         v.Artists,
				Album:           v.Album,
				AlbumId:         v.AlbumId,
				DurationSeconds: v.DurationSeconds,
				Liked:           v.Liked,
				Thumbnails:      v.Thumbnails,
				IsExplicit:      v.IsExplicit,
				Url:             v.Url,
			}
			return &types.PlaylistTrackObject{Track: song}
		}
	case types.HomePageContentItem:
		vID := v.VideoID
		if vID == "" {
			vID = v.PlaylistID
		}
		if vID != "" {
			return &types.PlaylistTrackObject{
				Track: &musicpb.Song{
					VideoId: vID,
					Title:   v.ItemTitle,
				},
			}
		}
	case types.SongRelatedContentItem:
		if v.SongRelatedContent != nil && v.VideoId != "" {
			return &types.PlaylistTrackObject{
				Track: &musicpb.Song{
					VideoId: v.VideoId,
					Title:   v.Title,
					Artists: v.Artists,
				},
			}
		}
	}
	return nil
}

func (m Model) addMusicToQueue() (Model, tea.Cmd) {
	var selectedItem list.Item

	switch m.FocusedOn {
	case MainView:
		switch m.MainViewMode {
		case NormalMode:
			selectedItem = m.SelectedPlayListItems.SelectedItem()
		case SearchResultMode:
			if len(m.SearchResult.Items()) > 0 {
				selectedItem = m.SearchResult.SelectedItem()
			}
		case HomePageMode:
			if m.HomePageViewMode == HomePageContentView && len(m.HomePageList.Items()) > 0 {
				selectedItem = m.HomePageList.SelectedItem()
			}
		}
	case QueueList:
		showRelated := (m.RightColumnMode == RightColumnRelated || m.RightColumnMode == "") && len(m.RelatedList.Items()) > 0
		if len(m.RelatedList.Items()) > 0 && showRelated {
			selectedItem = m.RelatedList.SelectedItem()
		}
	}

	track := extractTrackObject(selectedItem)
	if track == nil || track.Track == nil {
		return m, nil
	}

	m.Queue.AddTrack(track)
	toastCmd := m.Alert.NewAlertCmd(bubbleup.InfoKey, fmt.Sprintf("Added \"%s\" to queue", track.Title()))
	return m, tea.Batch(toastCmd, m.SyncQueueList())
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
		return &m.QueueList
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
			var resp *musicpb.GetHomePageResponse
			if homePage != nil {
				resp = homePage
			}
			return types.HomePageResponseMsg{
				Response: resp,
				Err:      err,
			}
		}
		return m, tea.Batch(SendLoadingCmd(), homePageFeed)
	}

	if itemName == "library" {
		m.PendingContextName = "Library"
		libraryCmd := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			library, err := m.YtMusicClient.GetLibrary(ctx, &musicpb.GetLibraryRequest{Limit: 100})
			var libResp *musicpb.GetLibraryResponse
			if library != nil {
				libResp = library
			}
			return types.GetLibraryMsg{
				Result: libResp,
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

func appendToPlayHistory(m *Model, track *types.PlaylistTrackObject) {
	if track == nil || track.Track == nil {
		return
	}
	if len(m.PlayHistory) > 0 {
		lastTrack := m.PlayHistory[len(m.PlayHistory)-1]
		if lastTrack != nil && lastTrack.Track != nil && lastTrack.Track.VideoId == track.Track.VideoId {
			m.PlayHistoryIndex = len(m.PlayHistory) - 1
			return
		}
	}
	trackCopy := *track
	m.PlayHistory = append(m.PlayHistory, &trackCopy)
	if len(m.PlayHistory) > 200 {
		trim := len(m.PlayHistory) - 200
		m.PlayHistory = m.PlayHistory[trim:]
	}
	m.PlayHistoryIndex = len(m.PlayHistory) - 1
}

func (m Model) playStandaloneTrack(track types.PlaylistTrackObject) (Model, tea.Cmd) {
	m.PlaybackContext = nil
	m.PlaybackContextName = ""
	m.PlaylistContextIndex = 0

	return m.PlaySelectedMusic(track)
}

func (m Model) playTrackFromList(track types.PlaylistTrackObject) (Model, tea.Cmd) {
	playlistItems := m.SelectedPlayListItems.Items()
	var contextTracks []*types.PlaylistTrackObject
	for _, item := range playlistItems {
		if pt, ok := item.(types.PlaylistTrackObject); ok {
			copy := pt
			contextTracks = append(contextTracks, &copy)
		}
	}

	contextName := m.SelectedPlayListItems.Title
	if contextName == "" {
		contextName = m.PendingContextName
	}
	if contextName == "" {
		contextName = "Playlist"
	}

	selectedIdx := m.SelectedPlayListItems.Index()
	m.SetPlaybackContext(contextTracks, contextName, selectedIdx)

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
		if item.VideoID != "" {
			var contextTracks []*types.PlaylistTrackObject
			for _, hpItem := range m.HomePageList.Items() {
				if contentItem, ok := hpItem.(types.HomePageContentItem); ok {
					if contentItem.VideoID != "" {
						contextTracks = append(contextTracks, &types.PlaylistTrackObject{
							Track: &musicpb.Song{
								VideoId:         contentItem.VideoID,
								Title:           contentItem.ItemTitle,
								Artists:         contentItem.Artists,
								DurationSeconds: contentItem.DurationSeconds,
							},
						})
					}
				}
			}

			trackID := item.VideoID
			playlistTrack := types.PlaylistTrackObject{
				Track: &musicpb.Song{
					VideoId: trackID,
					Title:   item.ItemTitle,
					Artists: item.Artists,
				},
			}

			contextName := m.HomePageList.Title
			if contextName == "" {
				contextName = "Home"
			}
			selectedIdx := m.HomePageList.Index()
			if len(contextTracks) > 0 {
				m.SetPlaybackContext(contextTracks, contextName, selectedIdx)
			}

			return m.PlaySelectedMusic(playlistTrack)
		} else if item.ContentType == "album" || strings.HasPrefix(item.BrowseID, "MPRE") {
			browseID := item.BrowseID
			if browseID == "" {
				browseID = item.PlaylistID
			}
			m.PendingContextName = item.ItemTitle
			return m.navigateToDetailView(m.getAlbumTracks(browseID))
		}
		playlistID := item.PlaylistID
		if playlistID == "" {
			playlistID = item.BrowseID
		}
		m.PendingContextName = item.ItemTitle
		playlistDetailMsg := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			playlistItems, err := m.YtMusicClient.GetPlaylistItems(ctx, &musicpb.GetPlaylistItemsRequest{
				PlaylistId: playlistID,
			})
			var plResp *musicpb.GetPlaylistItemsResponse
			if playlistItems != nil {
				plResp = playlistItems
			}
			return types.PlaylistDetailMsg{
				Playlist: plResp,
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
			var relResp *musicpb.GetSongRelatedResponse
			if relatedSongs != nil {
				relResp = relatedSongs
			}
			return types.RelatedSongsMsg{
				Related: relResp,
				Err:     err,
			}
		}

		var queueListUpdateCmd tea.Cmd
		if m.FocusedOn == QueueList {
			selectedIdx := m.QueueList.Index()
			inQueue := false
			if m.Queue != nil && m.Queue.Len() > 0 {
				userQueueTracks := m.Queue.AllTracks()
				trackIdx := selectedIdx - 1
				if trackIdx >= 0 && trackIdx < len(userQueueTracks) {
					inQueue = true
					for i := 0; i <= trackIdx; i++ {
						m.Queue.PopFirst()
					}
					queueListUpdateCmd = m.SyncQueueList()
					if len(m.QueueList.Items()) > 0 {
						m.QueueList.Select(0)
					}
				}
			}

			if !inQueue && len(m.PlaybackContext) > 0 {
				for idx, ctxTrack := range m.PlaybackContext {
					if ctxTrack != nil && ctxTrack.Track != nil && selectedItem.Track != nil && ctxTrack.Track.VideoId == selectedItem.Track.VideoId {
						m.PlaylistContextIndex = idx
						queueListUpdateCmd = m.SyncQueueList()
						if len(m.QueueList.Items()) > 0 {
							m.QueueList.Select(0)
						}
						break
					}
				}
			}

			m, cmd := m.PlaySelectedMusic(selectedItem)
			return m, tea.Batch(cmd, relatedSongsCmd, queueListUpdateCmd)
		}

		m, cmd := m.playTrackFromList(selectedItem)
		return m, tea.Batch(cmd, relatedSongsCmd, queueListUpdateCmd)

	case types.SongItem:
		playlistTrack := types.PlaylistTrackObject{Track: selectedItem.Song}
		relatedSongsCmd := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			relatedSongs, err := m.YtMusicClient.GetSongRelated(ctx, &musicpb.GetSongRelatedRequest{
				VideoId: selectedItem.VideoId,
			})
			var relResp *musicpb.GetSongRelatedResponse
			if relatedSongs != nil {
				relResp = relatedSongs
			}
			return types.RelatedSongsMsg{
				Related: relResp,
				Err:     err,
			}
		}
		m, cmd := m.playTrackFromList(playlistTrack)
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

		watchPlaylistCmd := func() tea.Msg {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			watchPlaylistItems, err := m.YtMusicClient.GetWatchPlaylistItems(ctx, &musicpb.GetWatchPlaylistItemsRequest{
				VideoId: selectedItem.VideoId,
				Limit:   100,
			})
			return types.WatchPlaylistItemsMsg{
				SourceID:           selectedItem.VideoId,
				WatchPlaylistItems: watchPlaylistItems,
				Err:                err,
			}
		}
		m, cmd := m.playStandaloneTrack(playlistTrack)
		return m, tea.Batch(cmd, watchPlaylistCmd)

	case types.SearchResultPlaylistItem:
		if selectedItem.SearchResultPlaylist != nil {
			m.PendingContextName = selectedItem.SearchResultPlaylist.Title
		}
		return m.navigateToDetailView(m.getPlaylistItems(selectedItem.BrowseId))

	case types.SearchResultAlbumItem:
		if selectedItem.SearchResultAlbum != nil {
			m.PendingContextName = selectedItem.SearchResultAlbum.Title
		}
		return m.navigateToDetailView(m.getAlbumTracks(selectedItem.BrowseId))

	case types.SearchResultArtistItem:
		if selectedItem.SearchResultArtist != nil {
			m.PendingContextName = selectedItem.SearchResultArtist.Name
		}
		return m.navigateToDetailView(m.getArtistTracks(selectedItem.BrowseId))

	case types.SearchResultPodcastItem:
		if selectedItem.SearchResultPodcast != nil {
			m.PendingContextName = selectedItem.SearchResultPodcast.Title
		}
		return m.navigateToDetailView(m.getPlaylistItems(selectedItem.BrowseId))

	case types.SearchResultEpisodeItem:
		return m.playStandaloneTrack(types.PlaylistTrackObject{
			Track: &musicpb.Song{
				VideoId: selectedItem.VideoId,
				Title:   selectedItem.Title,
			},
		})

	case types.PlaylistItem:
		if selectedItem.Playlist != nil {
			m.PendingContextName = selectedItem.Playlist.Title
		}
		return m.navigateToDetailView(m.getPlaylistItems(selectedItem.PlaylistId))

	case types.AlbumItem:
		if selectedItem.Album != nil {
			m.PendingContextName = selectedItem.Album.Title
		}
		return m.navigateToDetailView(m.getAlbumTracks(selectedItem.BrowseId))

	case types.ArtistItem:
		if selectedItem.Artist != nil {
			m.PendingContextName = selectedItem.Artist.Name
		}
		return m.navigateToDetailView(m.getArtistTracks(selectedItem.Id))

	case types.FollowedArtistItem:
		if selectedItem.FollowedArtist != nil {
			m.PendingContextName = selectedItem.FollowedArtist.Name
		}
		return m.navigateToDetailView(m.getArtistTracks(selectedItem.ChannelId))

	case types.LibraryChannelItem:
		if selectedItem.LibraryChannel != nil {
			m.PendingContextName = selectedItem.LibraryChannel.Name
		}
		return m.navigateToDetailView(m.getArtistTracks(selectedItem.BrowseId))

	case types.PodcastItem:
		if selectedItem.Podcast != nil {
			m.PendingContextName = selectedItem.Podcast.Title
		}
		return m.navigateToDetailView(m.getPlaylistItems(selectedItem.PodcastId))

	case types.SongRelatedContentItem:
		if selectedItem.SongRelatedContent != nil {
			m.PendingContextName = selectedItem.SongRelatedContent.Title
		}
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
				var relResp *musicpb.GetSongRelatedResponse
				if relatedSongs != nil {
					relResp = relatedSongs
				}
				return types.RelatedSongsMsg{
					Related: relResp,
					Err:     err,
				}
			}
			m, cmd := m.PlaySelectedMusic(playlistTrack)
			return m, tea.Batch(cmd, relatedSongsCmd)
		} else if selectedItem.ContentType == "artist" || strings.HasPrefix(selectedItem.BrowseId, "UC") || selectedItem.Subscribers != "" {
			return m.navigateToDetailView(m.getArtistTracks(selectedItem.BrowseId))
		} else if selectedItem.ContentType == "album" || strings.HasPrefix(selectedItem.BrowseId, "MPRE") {
			return m.navigateToDetailView(m.getAlbumTracks(selectedItem.BrowseId))
		} else if selectedItem.ContentType == "playlist" || (selectedItem.PlaylistId != "" && selectedItem.ContentType == "") {
			playlistID := selectedItem.PlaylistId
			if playlistID == "" {
				playlistID = selectedItem.BrowseId
			}
			return m.navigateToDetailView(m.getPlaylistItems(playlistID))
		} else if selectedItem.BrowseId != "" {
			return m.navigateToDetailView(m.getArtistTracks(selectedItem.BrowseId))
		}
	}

	return m, nil
}

func (m Model) handleSearchBarEnter() (Model, tea.Cmd) {
	query := m.Search.Value()
	if query == m.SearchQuery && len(m.SearchResult.Items()) > 0 {
		if query == m.SearchQuery && len(m.SearchResult.Items()) > 0 {
			m.MainViewMode = SearchResultMode
			m.Search.Blur()
			m.FocusedOn = MainView
			updateDelegate(&m)
			return m, nil
		}
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
		if searchResults != nil {
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
			Playlist: tracks,
			Err:      nil,
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
	cmd := youtube.SearchAndDownloadMusic(playCtx, selectedMusic.Track.VideoId, m.CoreDepsPath)
	m.CurrentLyrics = nil
	m.LyricsView.SetContent("")

	cmds = append(cmds, cmd)
	metadata := getMusicMetadata(MusicMetadata{
		artistName: artistNames,
		length:     int64(selectedMusic.Track.DurationSeconds * 1000),
		title:      selectedMusic.Track.Title,
		albumName:  selectedMusic.Track.Album,
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
	m.PlayedSeconds = 0
	m.SelectedTrack = &SelectedTrack{
		isLiked:             false,
		PlaylistTrackObject: selectedMusic,
	}

	cmds = append(cmds, m.SyncQueueList())
	if m.MainViewMode == LyricsMode {
		model, cmd := m.getMusicLyrics()
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
	m.QueueList.SetDelegate(CustomDelegate{Model: m})
	m.SideBarList.SetDelegate(CustomDelegate{Model: m})
	m.HomePageList.SetDelegate(CustomDelegate{Model: m})
	m.SearchResult.SetDelegate(CustomDelegate{Model: m})
	m.RelatedList.SetDelegate(CustomDelegate{Model: m})
}

func updateFocusedComponent(m *Model, msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.FocusedOn {
	case SearchBar:
		m.Search.Focus()
		m.Search, cmd = m.Search.Update(msg)
		return *m, cmd
	case SideView:
		m.Search.Blur()
		m.SideBarList, cmd = m.SideBarList.Update(msg)
		return *m, cmd
	case QueueList:
		var cmd tea.Cmd
		showRelated := (m.RightColumnMode == RightColumnRelated || m.RightColumnMode == "") && len(m.RelatedList.Items()) > 0
		if showRelated {
			m.RelatedList, cmd = m.RelatedList.Update(msg)
		} else {
			m.QueueList, cmd = m.QueueList.Update(msg)
		}
		return *m, cmd
	case MainView:
		switch m.MainViewMode {
		case NormalMode:
			m.SelectedPlayListItems, cmd = m.SelectedPlayListItems.Update(msg)
			return *m, cmd
		case SearchResultMode:
			m.SearchResult, cmd = m.SearchResult.Update(msg)
			return *m, cmd
		}
	default:
	}
	return *m, nil
}

func SendLoadingCmd() tea.Cmd {
	return func() tea.Msg {
		return types.SearchingMsg{}
	}
}

func (m Model) getCurrentSelectedTrack() (string, string) {
	if m.FocusedOn == Player {
		if m.SelectedTrack != nil && m.SelectedTrack.Track != nil {
			return m.SelectedTrack.Track.VideoId, m.SelectedTrack.Track.Title
		}
		return "", ""
	}

	listModel := getListItemForMusicToChoose(&m, m.FocusedOn)
	if listModel != nil && len(listModel.Items()) > 0 {
		selectedItem := listModel.SelectedItem()
		if selectedItem != nil {
			switch item := selectedItem.(type) {
			case types.PlaylistTrackObject:
				if item.Track != nil && item.Track.VideoId != "" {
					return item.Track.VideoId, item.Track.Title
				}
			case types.SongItem:
				if item.Song != nil && item.Song.VideoId != "" {
					return item.Song.VideoId, item.Song.Title
				}
			case types.SearchResultSongItem:
				if item.SearchResultSong != nil && item.SearchResultSong.VideoId != "" {
					return item.SearchResultSong.VideoId, item.SearchResultSong.Title
				}
			case types.HomePageContentItem:
				if item.VideoID != "" {
					return item.VideoID, item.Title()
				}
			case types.SongRelatedContentItem:
				if item.SongRelatedContent != nil && item.SongRelatedContent.VideoId != "" {
					return item.SongRelatedContent.VideoId, item.SongRelatedContent.Title
				}
			}
		}
	}

	if m.SelectedTrack != nil && m.SelectedTrack.Track != nil {
		return m.SelectedTrack.Track.VideoId, m.SelectedTrack.Track.Title
	}

	return "", ""
}

func (m Model) isCurrentFocusTrack() bool {
	trackID, _ := m.getCurrentSelectedTrack()
	return trackID != ""
}
