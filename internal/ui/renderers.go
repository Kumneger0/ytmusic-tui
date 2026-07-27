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
	"github.com/kumneger0/clispot/internal/types"
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

	switch item := item.(type) {
	case types.SearchResultItem:
		isSelected = d.Model != nil && d.Model.FocusedOn == MainView && m.Index() == index
		title = item.Title()
		switch item.Kind() {
		case types.SearchResultTrack:
			icon = "♫"
			if t, ok := item.(types.Track); ok && len(t.Artists) > 0 {
				var names []string
				for _, a := range t.Artists {
					names = append(names, a.Name)
				}
				subtitle = strings.Join(names, ", ")
			}
		case types.SearchResultArtist:
			icon = "♪"
			if a, ok := item.(types.Artist); ok {
				subtitle = fmt.Sprintf("%s — artist", a.Name)
			} else {
				subtitle = "Artist"
			}
		case types.SearchResultPlaylist:
			icon = "☰"
			if p, ok := item.(types.Playlist); ok {
				if p.Author != "" {
					subtitle = p.Author
				} else {
					subtitle = "Playlist"
				}
			}
		case types.SearchResultAlbum:
			icon = "◉"
			if a, ok := item.(types.Album); ok {
				subtitle = fmt.Sprintf("%s • %s", a.Type, a.Year)
			}
		case types.SearchResultPodcast:
			icon = "📻"
			if p, ok := item.(types.Podcast); ok {
				if p.Author != "" {
					subtitle = fmt.Sprintf("%s — podcast", p.Author)
				} else {
					subtitle = "Podcast Show"
				}
			}
		case types.SearchResultEpisode:
			icon = "🎙"
			if ep, ok := item.(types.Episode); ok {
				if ep.PodcastName != "" && ep.Date != "" {
					subtitle = fmt.Sprintf("%s • %s", ep.PodcastName, ep.Date)
				} else if ep.PodcastName != "" {
					subtitle = ep.PodcastName
				} else {
					subtitle = "Podcast Episode"
				}
			}
		}
	case types.PlaylistTrackObject:
		icon = "♫"
		title = item.Track.Name
		if len(item.Track.Artists) > 0 {
			var names []string
			for _, a := range item.Track.Artists {
				names = append(names, a.Name)
			}
			subtitle = strings.Join(names, ", ")
		}
		if d.Model != nil {
			switch d.Model.FocusedOn {
			case QueueList:
				if item.IsItFromQueue {
					isSelected = m.Index() == index
				}
			case MainView:
				if !item.IsItFromQueue && item.IsItFromSearch == false {
					isSelected = m.Index() == index
				}
			}
		}
	case types.SidebarItem:
		icon = item.Icon
		title = item.Name
		if d.Model != nil && d.Model.FocusedOn == SideView {
			isSelected = m.Index() == index
		}
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
		if d.Model != nil && d.Model.FocusedOn == MainView && d.Model.MainViewMode == HomePageMode {
			isSelected = m.Index() == index
		}
	case types.HomePageSectionItem:
		icon = "▸"
		title = item.SectionTitle
		if d.Model != nil && d.Model.FocusedOn == MainView && d.Model.MainViewMode == HomePageMode {
			isSelected = m.Index() == index
		}
	case types.UserSavedTracksListItem:
		title = item.FilterValue()
		if d.Model != nil {
			icon = "♥"
			isSelected = d.Model.FocusedOn == SideView && m.Index() == index
		}
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
	if selectedTrack == nil {
		return ""
	}
	var artistNames []string
	var artists = selectedTrack.Track.Track.Artists
	for _, artist := range artists {
		artistNames = append(artistNames, artist.Name)
	}
	artistName := strings.Join(artistNames, ", ")
	trackName := selectedTrack.Track.Track.Name

	var likedIndicator string
	if selectedTrack.isLiked {
		likedIndicator = " ♥"
	} else {
		likedIndicator = ""
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

	trackInfo := lipgloss.NewStyle().Foreground(textPrimary).Bold(true).Render(
		fmt.Sprintf("▶ %s", trackName),
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

func renderPlayerControls() string {
	key := lipgloss.NewStyle().Foreground(accentColor).Bold(true)
	sep := dimmerStyle.Render("  │  ")
	label := lipgloss.NewStyle().Foreground(textSecondary)

	var parts []string
	parts = append(parts,
		key.Render("⏮")+label.Render(" prev")+dimmerStyle.Render("(b)"),
		key.Render("⏯")+label.Render(" play/pause")+dimmerStyle.Render("(space)"),
		key.Render("⏭")+label.Render(" next")+dimmerStyle.Render("(n)"),
		key.Render("♥")+label.Render(" like")+dimmerStyle.Render("(l)"),
		key.Render("✕")+label.Render(" quit")+dimmerStyle.Render("(q)"),
		key.Render("📝")+label.Render(" lyrics")+dimmerStyle.Render("(ctrl+l)"),
	)
	return strings.Join(parts, sep)
}
