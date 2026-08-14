package ui

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
	musicpb "github.com/kumneger0/ytmusic-tui/gen"
	"github.com/kumneger0/ytmusic-tui/gen/genconnect"
	"github.com/kumneger0/ytmusic-tui/internal/queue"
	"github.com/kumneger0/ytmusic-tui/internal/types"
	"github.com/kumneger0/ytmusic-tui/internal/youtube"
	"go.dalton.dog/bubbleup"
)

type FocusedOn string

const (
	SideView  FocusedOn = "SIDE_VIEW"
	MainView  FocusedOn = "MAIN_VIEW"
	Player    FocusedOn = "PLAYER"
	SearchBar FocusedOn = "SEARCH_BAR"
	QueueList FocusedOn = "QUEUE_LIST"
)

type MainViewMode string

const (
	SearchResultMode MainViewMode = "SEARCH_RESULT_MODE"
	//currently im showing the search result in main area which is the center one
	//let's say the user searches for a song or playlist and sees the result and he chose the first result
	//at this time the previous are gone b/c i was sharing  this main new to show items in playlist and the search result
	// so by adding this MainViewMode we can switch b/c modes so that we keep the result in memory
	// meaning we can switch b/n search result and normal mode
	NormalMode   MainViewMode = "NORMAL_MODE"
	LyricsMode   MainViewMode = "LYRICS_MODE"
	HomePageMode MainViewMode = "HOME_PAGE_MODE"
)

type HomePageViewMode int

const (
	HomePageSectionView HomePageViewMode = iota
	HomePageContentView
)

type RightColumnMode string

const (
	RightColumnQueue   RightColumnMode = "RIGHT_COLUMN_QUEUE"
	RightColumnRelated RightColumnMode = "RIGHT_COLUMN_RELATED"
)

type SpotifySearchResult struct {
	Tracks, Artists, Albums, Playlists list.Model
}

type SelectedTrack struct {
	types.PlaylistTrackObject
	isLiked bool
}

type MusicQueueList struct {
	list.Model
	PaginationInfo *types.PaginationInfo
}

type Model struct {
	BreadcrumbItems       []types.Breadcrumb
	SideBarList           list.Model
	Alert                 bubbleup.AlertModel
	SelectedPlayListItems list.Model
	LyricsView            viewport.Model
	FocusedOn             FocusedOn
	MainViewMode
	PlayerProcess        *types.Player
	playbackCancel       context.CancelFunc
	SelectedTrack        *SelectedTrack
	PlayedSeconds        float64
	Height               int
	Width                int
	LibraryWidth         int
	MainViewWidth        int
	PlayerSectionHeight  int
	Search               textinput.Model
	Queue                *queue.RingQueue
	QueueList            list.Model
	PlaybackContext      []*types.PlaylistTrackObject
	PlaybackContextName  string
	PendingContextName   string
	PlaylistContextIndex int
	PlayHistory          []*types.PlaylistTrackObject
	PlayHistoryIndex     int
	YtMusicClient        genconnect.MusicServiceClient
	DBusConn             *Instance
	//actually i need this b/c if user searches and selects playlist or artist
	//at that time when he selects artist or playlist the search were hidden from mainView
	//so that if search again we can show the previous result by comparing the query
	// TODO: find a better way than this looks very ugly
	SearchQuery string
	// SearchResult                             *SpotifySearchResult
	IsSearchLoading  bool
	SearchResult     list.Model
	PaginationInfo   *types.PaginationInfo
	IsOnPagination   bool
	CoreDepsPath     *youtube.CoreDepsPath
	HomePageData     *musicpb.GetHomePageResponse
	HomePageList     list.Model
	HomePageViewMode HomePageViewMode
	RelatedList      list.Model
	RightColumnMode  RightColumnMode
	CurrentLyrics    *musicpb.GetLyricsResponse
}

type Instance struct {
	Props *prop.Properties
	Conn  *dbus.Conn
}

func (m Model) Init() tea.Cmd {
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
	return tea.Batch(m.Alert.Init(), SendLoadingCmd(), homePageFeed)
}

func renderBreadcrumbs(items []types.Breadcrumb) string {
	if len(items) == 0 {
		return ""
	}

	parts := make([]string, 0, len(items)*2)
	for i, item := range items {
		label := item.Name
		if item.Icon != "" {
			label = fmt.Sprintf("%s %s", item.Icon, item.Name)
		}
		itemStyle := lipgloss.NewStyle().Foreground(accentColor).Bold(true)
		if i == len(items)-1 {
			itemStyle = itemStyle.Foreground(textPrimary).Bold(true)
		} else {
			itemStyle = itemStyle.Foreground(accentColor)
		}

		parts = append(parts, itemStyle.Render(label))
		if i < len(items)-1 {
			parts = append(parts, lipgloss.NewStyle().Foreground(textDim).Render("▸"))
		}
	}

	return lipgloss.NewStyle().Padding(0, 0, 0, 1).Render(strings.Join(parts, " "))
}

func (m Model) View() string {
	dimensions := calculateLayoutDimensions(&m)
	sideBarView := getStyle(&m, dimensions.ContentHeight, dimensions.SidebarWidth, SideView, false).Render(m.SideBarList.View())
	searchBar := renderSearchBar(&m, dimensions.MainWidth)
	breadcrumb := renderBreadcrumbs(m.BreadcrumbItems)
	var mainView string
	if m.IsSearchLoading {
		loadingText := dimmerStyle.Render("  ⟳ Loading...")
		mainView = getStyle(&m, dimensions.ContentHeight, dimensions.MainWidth, MainView, false).Render(
			lipgloss.JoinVertical(lipgloss.Top, searchBar, breadcrumb, loadingText),
		)
	} else if m.MainViewMode == SearchResultMode {
		resultHeader := titleStyle.Render("  Search Results")
		mainView = getStyle(&m, dimensions.ContentHeight, dimensions.MainWidth, MainView, false).Render(
			lipgloss.JoinVertical(lipgloss.Top, searchBar, resultHeader, lipgloss.NewStyle().Padding(1, 0, 0, 0).Render(m.SearchResult.View())),
		)
	} else if m.MainViewMode == LyricsMode {
		trackName := ""
		if m.SelectedTrack != nil && m.SelectedTrack.Track != nil {
			trackName = " • " + m.SelectedTrack.Track.Title
		}
		lyricsHeader := titleStyle.Render("  📝 Lyrics" + trackName)
		lyricsPadded := lipgloss.NewStyle().Padding(1, 2).Render(m.LyricsView.View())
		mainView = getStyle(&m, dimensions.ContentHeight, dimensions.MainWidth, MainView, false).Render(
			lipgloss.JoinVertical(lipgloss.Top, searchBar, breadcrumb, lyricsHeader, lyricsPadded),
		)
	} else if m.MainViewMode == HomePageMode {
		mainView = getStyle(&m, dimensions.ContentHeight, dimensions.MainWidth, MainView, false).Render(
			lipgloss.JoinVertical(lipgloss.Top, searchBar, breadcrumb, lipgloss.NewStyle().Padding(1, 0, 0, 0).Render(m.HomePageList.View())),
		)
	} else {
		mainView = getStyle(&m, dimensions.ContentHeight, dimensions.MainWidth, MainView, false).
			Render(lipgloss.JoinVertical(lipgloss.Top, searchBar, breadcrumb, lipgloss.NewStyle().Padding(1, 0, 0, 0).Render(m.SelectedPlayListItems.View())))
	}

	var playingView string

	if m.SelectedTrack != nil && m.SelectedTrack.Track != nil {
		playedSeconds := int(m.PlayedSeconds)
		currentPosition := time.Second * time.Duration(playedSeconds)
		total := time.Duration(m.SelectedTrack.Track.DurationSeconds) * time.Second
		playingView = renderNowPlaying(&m, currentPosition, total)
	}

	controls := renderPlayerControls(&m)
	playingCombined := strings.TrimSpace(playingView) + "\n" + controls

	playing := getPlayerStyles(&m, dimensions).
		Foreground(playerFg).
		Render(playingCombined)
	var rightColumnView string
	showRelated := m.RightColumnMode == RightColumnRelated
	if showRelated {
		rightColumnView = m.RelatedList.View()
	} else {
		rightColumnView = m.QueueList.View()
	}
	queueList := getStyle(&m, dimensions.ContentHeight, dimensions.SidebarWidth, QueueList, false).Render(rightColumnView)
	combinedView := lipgloss.JoinVertical(lipgloss.Top,
		lipgloss.JoinHorizontal(lipgloss.Top, sideBarView, mainView, queueList),
		playing,
	)
	return m.Alert.Render(combinedView)
}

func formatTime(d time.Duration) string {
	totalSeconds := int(time.Duration(math.Max(float64(d), 0)).Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

type LayoutDimensions struct {
	SidebarWidth  int
	MainWidth     int
	ContentHeight int
	InputHeight   int
}

func CalculateLayoutDimensions(m *Model) LayoutDimensions {
	if m.Width <= 0 || m.Height <= 0 {
		m.Width = 100
		m.Height = 30
	}
	sidebarWidth := m.Width * 22 / 100
	inputHeight := min(max(m.Height*10/100, 2), 3)
	mainCenterArea := m.Width - (sidebarWidth * 2) - 2
	if mainCenterArea < 10 {
		mainCenterArea = 10
	}

	return LayoutDimensions{
		SidebarWidth:  sidebarWidth,
		MainWidth:     mainCenterArea,
		ContentHeight: m.Height * 90 / 100,
		InputHeight:   inputHeight,
	}
}

func calculateLayoutDimensions(m *Model) LayoutDimensions {
	return CalculateLayoutDimensions(m)
}

func (m *Model) UpdateListDimensions() {
	dimensions := CalculateLayoutDimensions(m)
	m.SideBarList.SetSize(dimensions.SidebarWidth, dimensions.ContentHeight)
	m.SelectedPlayListItems.SetSize(dimensions.MainWidth, dimensions.ContentHeight-4)
	m.SearchResult.SetSize(dimensions.MainWidth, dimensions.ContentHeight-4)
	m.HomePageList.SetSize(dimensions.MainWidth, dimensions.ContentHeight-4)
	m.QueueList.SetSize(dimensions.SidebarWidth, dimensions.ContentHeight)
	if len(m.RelatedList.Items()) > 0 {
		m.RelatedList.SetSize(dimensions.SidebarWidth, dimensions.ContentHeight)
	}
}

func (m *Model) SetPlaybackContext(tracks []*types.PlaylistTrackObject, name string, currentIndex int) {
	m.PlaybackContext = tracks
	m.PlaybackContextName = name
	m.PlaylistContextIndex = currentIndex
}

func (m *Model) SyncQueueList() tea.Cmd {
	var items []list.Item
	if m.Queue != nil && m.Queue.Len() > 0 {
		userQueueTracks := m.Queue.AllTracks()
		items = append(items, types.HomePageSectionItem{SectionTitle: "Queue"})
		for _, t := range userQueueTracks {
			if t != nil {
				items = append(items, *t)
			}
		}
	}

	if len(m.PlaybackContext) > 0 {
		contextName := m.PlaybackContextName
		if contextName == "" {
			contextName = "Playlist"
		}
		items = append(items, types.HomePageSectionItem{SectionTitle: "Next from " + contextName})

		n := len(m.PlaybackContext)
		startIdx := m.PlaylistContextIndex
		limit := n
		if m.SelectedTrack != nil {
			startIdx++
			limit = n - 1
		}
		for i := 0; i < limit; i++ {
			idx := (startIdx + i) % n
			if m.PlaybackContext[idx] != nil {
				items = append(items, *m.PlaybackContext[idx])
			}
		}
	}

	if len(items) == 0 && len(m.PlayHistory) > 0 {
		items = append(items, types.HomePageSectionItem{SectionTitle: "Recently Played"})
		for i := len(m.PlayHistory) - 1; i >= 0; i-- {
			if m.PlayHistory[i] != nil {
				items = append(items, *m.PlayHistory[i])
			}
		}
	}
	return m.QueueList.SetItems(items)
}

func RemoveListDefaults(listToRemoveDefaults *list.Model) {
	if listToRemoveDefaults != nil {
		listToRemoveDefaults.SetShowFilter(false)
		listToRemoveDefaults.SetShowPagination(false)
		listToRemoveDefaults.SetShowHelp(false)
		listToRemoveDefaults.SetShowStatusBar(false)
	}
}

func removeListDefaults(listToRemoveDefaults *list.Model) {
	RemoveListDefaults(listToRemoveDefaults)
}

func (m *Model) IsPlaying() bool {
	return m != nil && m.PlayerProcess != nil && m.PlayerProcess.OtoPlayer != nil && m.PlayerProcess.OtoPlayer.IsPlaying()
}
