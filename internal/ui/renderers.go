package ui

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kumneger0/ytmusic-tui/internal/types"
)

type CustomDelegate struct {
	list.DefaultDelegate
	*Model
}

func (d CustomDelegate) Height() int {
	return 1
}

func (d CustomDelegate) Spacing() int {
	return 0
}

func (d CustomDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d CustomDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var title string
	var isSelected bool
	var icon string
	var subtitle string

	if d.Model != nil {
		switch d.Model.FocusedOn {
		case SideView:
			if m.Title == "Youtube Music tui" || m.Title == "Library" {
				isSelected = m.Index() == index
			}
		case MainView:
			if m.Title != "Related" && m.Title != "Queue" && m.Title != "Youtube Music tui" {
				isSelected = m.Index() == index
			}
		case QueueList:
			if m.Title == "Related" || m.Title == "Queue" {
				isSelected = m.Index() == index
			}
		}
	}

	switch item := item.(type) {
	case types.SongItem:
		icon = "♫"
		if item.Song != nil {
			title = item.Title
			if len(item.Artists) > 0 {
				var names []string
				for _, a := range item.Artists {
					names = append(names, a.Name)
				}
				subtitle = strings.Join(names, ", ")
			}
		}
	case types.SearchResultSongItem:
		icon = "♫"
		if item.SearchResultSong != nil {
			title = item.Title
			if len(item.Artists) > 0 {
				var names []string
				for _, a := range item.Artists {
					names = append(names, a.Name)
				}
				subtitle = strings.Join(names, ", ")
			}
		}
	case types.SearchResultArtistItem:
		icon = "♪"
		if item.SearchResultArtist != nil {
			title = item.Name
			if item.Subscribers != "" {
				subtitle = fmt.Sprintf("%s — %s subscribers", item.Name, item.Subscribers)
			} else {
				subtitle = "Artist"
			}
		}
	case types.SearchResultPlaylistItem:
		icon = "☰"
		if item.SearchResultPlaylist != nil {
			title = item.Title
			if item.Author != "" {
				subtitle = item.Author
			} else {
				subtitle = "Playlist"
			}
		}
	case types.SearchResultAlbumItem:
		icon = "◉"
		if item.SearchResultAlbum != nil {
			title = item.Title
			subtitle = fmt.Sprintf("%s • %s", item.Type, item.Year)
		}
	case types.SearchResultPodcastItem:
		icon = "📻"
		if item.SearchResultPodcast != nil {
			title = item.Title
			if item.Author != "" {
				subtitle = fmt.Sprintf("%s — podcast", item.Author)
			} else {
				subtitle = "Podcast Show"
			}
		}
	case types.SearchResultEpisodeItem:
		icon = "🎙"
		if item.SearchResultEpisode != nil {
			title = item.Title
			if item.PodcastName != "" && item.Date != "" {
				subtitle = fmt.Sprintf("%s • %s", item.PodcastName, item.Date)
			} else if item.PodcastName != "" {
				subtitle = item.PodcastName
			} else {
				subtitle = "Podcast Episode"
			}
		}
	case types.AlbumItem:
		icon = "◉"
		if item.Album != nil {
			title = item.Title
			subtitle = fmt.Sprintf("%s • %s", item.Type, item.Year)
		}
	case types.PlaylistItem:
		icon = "☰"
		if item.Playlist != nil {
			title = item.Title
			if item.Author != "" {
				subtitle = item.Author
			} else {
				subtitle = "Playlist"
			}
		}
	case types.FollowedArtistItem:
		icon = "♪"
		if item.FollowedArtist != nil {
			title = item.Name
			if item.Subscribers != "" {
				subtitle = item.Subscribers
			} else {
				subtitle = "Artist"
			}
		}
	case types.LibraryChannelItem:
		icon = "♪"
		if item.LibraryChannel != nil {
			title = item.Name
			subtitle = "Channel"
		}
	case types.PodcastItem:
		icon = "📻"
		if item.Podcast != nil {
			title = item.Title
			subtitle = item.Author
		}
	case types.SongRelatedContentItem:
		if item.SongRelatedContent != nil {
			if item.VideoId != "" || item.ContentType == "song" || item.ContentType == "video" {
				icon = "♫"
			} else if item.ContentType == "artist" || strings.HasPrefix(item.BrowseId, "UC") || item.Subscribers != "" {
				icon = "♪"
			} else if item.ContentType == "album" || strings.HasPrefix(item.BrowseId, "MPRE") {
				icon = "◉"
			} else {
				icon = "☰"
			}
			title = item.Title
			subtitle = item.Description
		}
	case types.PlaylistTrackObject:
		icon = "♫"
		if item.Track != nil {
			title = item.Track.Title
			if d.Model != nil && d.Model.SelectedTrack != nil && d.Model.SelectedTrack.Track != nil &&
				item.Track.VideoId == d.Model.SelectedTrack.Track.VideoId {
				title += " (current)"
			}
			if len(item.Track.Artists) > 0 {
				var names []string
				for _, a := range item.Track.Artists {
					names = append(names, a.Name)
				}
				subtitle = strings.Join(names, ", ")
			}
		}
	case types.SidebarItem:
		icon = item.Icon
		title = item.Name
	case types.HomePageContentItem:
		if item.VideoID != "" || item.ContentType == "song" || item.ContentType == "video" {
			icon = "♫"
		} else if item.ContentType == "album" || strings.HasPrefix(item.BrowseID, "MPRE") {
			icon = "◉"
		} else {
			icon = "☰"
		}
		title = item.ItemTitle
		subtitle = item.Description
	case types.HomePageSectionItem:
		icon = "▸"
		title = item.SectionTitle
	case types.UserSavedTracksListItem:
		title = item.FilterValue()
		icon = "♥"
	}

	availableWidth := m.Width()
	if availableWidth <= 0 {
		availableWidth = 40
	}

	prefix := fmt.Sprintf(" %s %s", icon, title)
	prefixWidth := lipgloss.Width(prefix)

	var rendered string
	if subtitle != "" && availableWidth >= prefixWidth+8 {
		sep := " · "
		sepWidth := lipgloss.Width(sep)
		maxSubWidth := availableWidth - prefixWidth - sepWidth - 1
		if maxSubWidth > 3 {
			sub := truncateText(subtitle, maxSubWidth)
			if isSelected {
				rendered = selectedStyle.Render(prefix) +
					selectedStyle.Foreground(lipgloss.Color("#D4D4D8")).Render(sep+sub+" ")
			} else {
				rendered = normalStyle.Render(prefix) +
					dimStyle.Render(sep+sub+" ")
			}
		} else {
			str := prefix + " "
			if isSelected {
				rendered = selectedStyle.Render(str)
			} else {
				rendered = normalStyle.Render(str)
			}
		}
	} else {
		if prefixWidth > availableWidth-1 {
			iconWidth := lipgloss.Width(fmt.Sprintf(" %s ", icon))
			maxTitleWidth := availableWidth - iconWidth - 1
			if maxTitleWidth > 3 {
				title = truncateText(title, maxTitleWidth)
				prefix = fmt.Sprintf(" %s %s", icon, title)
			}
		}
		str := prefix + " "
		if isSelected {
			rendered = selectedStyle.Render(str)
		} else {
			rendered = normalStyle.Render(str)
		}
	}

	fmt.Fprint(w, rendered)
}

func truncateText(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxW {
		runes = runes[:len(runes)-1]
	}
	if len(runes) == 0 {
		return ""
	}
	return string(runes) + "…"
}

func renderSearchBar(m *Model, width int) string {
	if width < 20 {
		width = 20
	}
	m.Search.Width = width - 6

	box := lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Margin(0).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderNormal).
		Foreground(textPrimary)

	var content string
	if m.Search.Value() == "" && !m.Search.Focused() {
		content = dimmerStyle.Render("🔍 Search tracks, artists, playlists...")
	} else {
		content = strings.TrimRight(m.Search.View(), "\n")
	}
	return strings.TrimRight(box.Render(content), "\n")
}

func renderNowPlaying(m *Model, currentPosition, TotalDuration time.Duration) string {
	selectedTrack := m.SelectedTrack
	if selectedTrack == nil || selectedTrack.Track == nil {
		return ""
	}
	var artistNames []string
	var artists = selectedTrack.Track.Artists
	for _, artist := range artists {
		artistNames = append(artistNames, artist.Name)
	}
	artistName := strings.Join(artistNames, ", ")
	trackName := selectedTrack.Track.Title

	var likedIndicator string
	if selectedTrack.isLiked {
		likedIndicator = " ♥"
	} else {
		likedIndicator = " ♡"
	}

	barWidth := m.Width
	var progressFloat float64
	if TotalDuration == 0 {
		progressFloat = 1.0
	} else {
		progressFloat = float64(currentPosition.Abs()) / float64(TotalDuration.Abs()) * float64(barWidth)
	}
	progress := max(min(int(math.Max(progressFloat, 1)), barWidth), 0)

	filled := lipgloss.NewStyle().Foreground(progressFilled).Render(strings.Repeat("━", progress))
	empty := lipgloss.NewStyle().Foreground(progressEmpty).Render(strings.Repeat("─", max(barWidth-progress, 0)))

	playIcon := "⏸"
	if m.IsPlaying() {
		playIcon = "▶"
	}

	trackInfo := lipgloss.NewStyle().Foreground(textPrimary).Bold(true).Render(
		fmt.Sprintf("%s %s", playIcon, trackName),
	)
	artistInfo := dimStyle.Render(fmt.Sprintf(" — %s", artistName))
	timeInfo := dimStyle.Render(fmt.Sprintf("  %s / %s", formatTime(currentPosition), formatTime(TotalDuration)))
	likeInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171")).Render(likedIndicator)

	return fmt.Sprintf("%s%s%s%s\n%s%s\n",
		trackInfo,
		artistInfo,
		timeInfo,
		likeInfo,
		filled, empty,
	)
}

func renderPlayerControls(m *Model) string {
	key := lipgloss.NewStyle().Foreground(accentColor).Bold(true)
	sep := dimmerStyle.Render("  │  ")
	label := lipgloss.NewStyle().Foreground(textSecondary)

	playPauseIcon := "▶"
	playPauseLabel := " play"
	if m.IsPlaying() {
		playPauseIcon = "⏸"
		playPauseLabel = " pause"
	}

	var parts []string
	hasTrack := m.isCurrentFocusTrack()

	switch m.FocusedOn {
	case SideView:
		parts = append(parts,
			key.Render("↵")+label.Render(" select")+dimmerStyle.Render("(enter)"),
			key.Render("✨")+label.Render(" new playlist")+dimmerStyle.Render("(ctrl+t)"),
			key.Render("→")+label.Render(" main view")+dimmerStyle.Render("(tab)"),
			key.Render("↓")+label.Render(" player")+dimmerStyle.Render("(shift+tab)"),
			key.Render("🔍")+label.Render(" search")+dimmerStyle.Render("(ctrl+k)"),
			key.Render("✕")+label.Render(" quit")+dimmerStyle.Render("(q)"),
		)
	case MainView:
		if m.MainViewMode == LyricsMode {
			parts = append(parts,
				key.Render("↕")+label.Render(" scroll")+dimmerStyle.Render("(j/k)"),
				key.Render("→")+label.Render(" queue")+dimmerStyle.Render("(tab)"),
				key.Render("←")+label.Render(" sidebar")+dimmerStyle.Render("(shift+tab)"),
				key.Render("🔍")+label.Render(" search")+dimmerStyle.Render("(ctrl+k)"),
				key.Render("✕")+label.Render(" quit")+dimmerStyle.Render("(q)"),
			)
		} else {
			parts = append(parts,
				key.Render("▶")+label.Render(" play")+dimmerStyle.Render("(enter)"),
				key.Render("+")+label.Render(" add queue")+dimmerStyle.Render("(a)"),
			)
			if hasTrack {
				parts = append(parts, key.Render("📋")+label.Render(" playlist")+dimmerStyle.Render("(ctrl+p)"))
			}
			parts = append(parts,
				key.Render("✨")+label.Render(" new playlist")+dimmerStyle.Render("(ctrl+t)"),
				key.Render("→")+label.Render(" queue")+dimmerStyle.Render("(tab)"),
				key.Render("←")+label.Render(" sidebar")+dimmerStyle.Render("(shift+tab)"),
				key.Render("🔍")+label.Render(" search")+dimmerStyle.Render("(ctrl+k)"),
				key.Render("✕")+label.Render(" quit")+dimmerStyle.Render("(q)"),
			)
		}
	case QueueList:
		parts = append(parts,
			key.Render("▶")+label.Render(" play")+dimmerStyle.Render("(enter)"),
		)
		if hasTrack {
			parts = append(parts, key.Render("📋")+label.Render(" playlist")+dimmerStyle.Render("(ctrl+p)"))
		}
		parts = append(parts,
			key.Render("✕")+label.Render(" remove")+dimmerStyle.Render("(r)"),
			key.Render("↓")+label.Render(" player")+dimmerStyle.Render("(tab)"),
			key.Render("←")+label.Render(" main view")+dimmerStyle.Render("(shift+tab)"),
			key.Render("🔍")+label.Render(" search")+dimmerStyle.Render("(ctrl+k)"),
			key.Render("✕")+label.Render(" quit")+dimmerStyle.Render("(q)"),
		)
	case Player:
		parts = append(parts,
			key.Render(playPauseIcon)+label.Render(playPauseLabel)+dimmerStyle.Render("(space)"),
			key.Render("⏮")+label.Render(" prev")+dimmerStyle.Render("(b)"),
			key.Render("⏭")+label.Render(" next")+dimmerStyle.Render("(n)"),
			key.Render("♥")+label.Render(" like")+dimmerStyle.Render("(l)"),
		)
		if hasTrack {
			parts = append(parts, key.Render("📋")+label.Render(" playlist")+dimmerStyle.Render("(ctrl+p)"))
		}
		parts = append(parts,
			key.Render("✨")+label.Render(" new playlist")+dimmerStyle.Render("(ctrl+t)"),
			key.Render("📝")+label.Render(" lyrics")+dimmerStyle.Render("(ctrl+l)"),
			key.Render("←")+label.Render(" sidebar")+dimmerStyle.Render("(tab)"),
			key.Render("→")+label.Render(" queue")+dimmerStyle.Render("(shift+tab)"),
			key.Render("✕")+label.Render(" quit")+dimmerStyle.Render("(q)"),
		)
	case SearchBar:
		parts = append(parts,
			key.Render("🔍")+label.Render(" search")+dimmerStyle.Render("(enter)"),
			key.Render("✕")+label.Render(" cancel")+dimmerStyle.Render("(esc)"),
		)
		if hasTrack {
			parts = append(parts, key.Render("📋")+label.Render(" playlist")+dimmerStyle.Render("(ctrl+p)"))
		}
	default:
		parts = append(parts,
			key.Render(playPauseIcon)+label.Render(playPauseLabel)+dimmerStyle.Render("(space)"),
			key.Render("🔍")+label.Render(" search")+dimmerStyle.Render("(ctrl+k)"),
			key.Render("✕")+label.Render(" quit")+dimmerStyle.Render("(q)"),
		)
		if hasTrack {
			parts = append(parts, key.Render("📋")+label.Render(" playlist")+dimmerStyle.Render("(ctrl+p)"))
		}
	}

	return strings.Join(parts, sep)
}
