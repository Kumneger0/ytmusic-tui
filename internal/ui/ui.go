package ui

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
	musicpb "github.com/kumneger0/clispot/gen"
	"github.com/kumneger0/clispot/internal/types"
	"github.com/kumneger0/clispot/internal/youtube"
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
	isLiked bool
	Track   *types.PlaylistTrackObject
}

type MusicQueueList struct {
	list.Model
	PaginationInfo *types.PaginationInfo
}

type Model struct {
	BackendProcess        *exec.Cmd
	BreadcrumbItems       []types.Breadcrumb
	SideBarList           list.Model
	Alert                 bubbleup.AlertModel
	SelectedPlayListItems list.Model
	LyricsView            viewport.Model
	FocusedOn             FocusedOn
	MainViewMode
	PlayerProcess       *types.Player
	playbackCancel      context.CancelFunc
	SelectedTrack       *SelectedTrack
	PlayedSeconds       float64
	Height              int
	Width               int
	LibraryWidth        int
	MainViewWidth       int
	PlayerSectionHeight int
	Search              textinput.Model
	MusicQueueList      *MusicQueueList
	YtMusicClient       musicpb.MusicServiceClient
	DBusConn            *Instance
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

type SafeModel struct {
	Mu sync.RWMutex
	*Model
}

func (m Model) Init() tea.Cmd {
	cmd := func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		followedArtist, err := m.YtMusicClient.GetFollowedArtists(ctx, &musicpb.GetFollowedArtistsRequest{})
		if err != nil {
			return nil
		}
		return followedArtist
	}
	pythonBackendHealthCheckCmd := func() tea.Msg {
		var count int
		for {
			time.Sleep(time.Second * 5)
			count++
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			response, err := m.YtMusicClient.HealthCheck(ctx, &musicpb.HealthCheckRequest{})
			if err != nil && count <= 5 {
				continue
			}
			return types.PythonBackendHealthResponseMsg{
				Response: response,
				Err:      err,
			}
		}
	}
	return tea.Batch(cmd, m.Alert.Init(), SendLoadingCmd(), pythonBackendHealthCheckCmd)
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
	m.SideBarList.Title = "Youtube Music tui"
	m.MusicQueueList.Model.Title = "Queue"
	removeListDefaults(&m.SideBarList)
	removeListDefaults(&m.SelectedPlayListItems)
	removeListDefaults(&m.SearchResult)
	removeListDefaults(&m.HomePageList)
	removeListDefaults(&m.MusicQueueList.Model)
	if len(m.RelatedList.Items()) > 0 {
		m.RelatedList.Title = "Related"
		removeListDefaults(&m.RelatedList)
	}
	m.SearchResult.SetShowTitle(false)
	m.SelectedPlayListItems.SetShowTitle(false)
	m.HomePageList.SetShowTitle(false)
	dimensions := calculateLayoutDimensions(&m)
	m.SideBarList.SetSize(dimensions.sidebarWidth, dimensions.contentHeight)
	m.SelectedPlayListItems.SetSize(dimensions.mainWidth, dimensions.contentHeight-4)
	m.SearchResult.SetSize(dimensions.mainWidth, dimensions.contentHeight-4)
	m.HomePageList.SetSize(dimensions.mainWidth, dimensions.contentHeight-4)
	if len(m.RelatedList.Items()) > 0 {
		m.RelatedList.SetSize(dimensions.sidebarWidth, dimensions.contentHeight)
	}
	if m.MusicQueueList != nil {
		m.MusicQueueList.Model.SetSize(dimensions.sidebarWidth, dimensions.contentHeight)
	}
	sideBarView := getStyle(&m, dimensions.contentHeight, dimensions.sidebarWidth, SideView, false).Render(m.SideBarList.View())
	searchBar := renderSearchBar(&m, dimensions.mainWidth)
	breadcrumb := renderBreadcrumbs(m.BreadcrumbItems)
	var mainView string
	if m.IsSearchLoading {
		loadingText := dimmerStyle.Render("  ⟳ Loading...")
		mainView = getStyle(&m, dimensions.contentHeight, dimensions.mainWidth, MainView, false).Render(
			lipgloss.JoinVertical(lipgloss.Top, searchBar, breadcrumb, loadingText),
		)
	} else if m.MainViewMode == SearchResultMode {
		resultHeader := titleStyle.Render("  Search Results")
		mainView = getStyle(&m, dimensions.contentHeight, dimensions.mainWidth, MainView, false).Render(
			lipgloss.JoinVertical(lipgloss.Top, searchBar, resultHeader, lipgloss.NewStyle().Padding(1, 0, 0, 0).Render(m.SearchResult.View())),
		)
	} else if m.MainViewMode == LyricsMode {
		trackName := ""
		if m.SelectedTrack != nil && m.SelectedTrack.Track != nil {
			trackName = " • " + m.SelectedTrack.Track.Track.Name
		}
		lyricsHeader := titleStyle.Render("  📝 Lyrics" + trackName)
		lyricsPadded := lipgloss.NewStyle().Padding(1, 2).Render(m.LyricsView.View())
		mainView = getStyle(&m, dimensions.contentHeight, dimensions.mainWidth, MainView, false).Render(
			lipgloss.JoinVertical(lipgloss.Top, searchBar, breadcrumb, lyricsHeader, lyricsPadded),
		)
	} else if m.MainViewMode == HomePageMode {
		mainView = getStyle(&m, dimensions.contentHeight, dimensions.mainWidth, MainView, false).Render(
			lipgloss.JoinVertical(lipgloss.Top, searchBar, breadcrumb, lipgloss.NewStyle().Padding(1, 0, 0, 0).Render(m.HomePageList.View())),
		)
	} else {
		mainView = getStyle(&m, dimensions.contentHeight, dimensions.mainWidth, MainView, false).
			Render(lipgloss.JoinVertical(lipgloss.Top, searchBar, breadcrumb, lipgloss.NewStyle().Padding(1, 0, 0, 0).Render(m.SelectedPlayListItems.View())))
	}

	var playingView string

	if m.SelectedTrack != nil && m.SelectedTrack.Track != nil {
		playedSeconds := int(m.PlayedSeconds)
		currentPosition := time.Second * time.Duration(playedSeconds)
		total := time.Duration(m.SelectedTrack.Track.Track.DurationMS) * time.Millisecond
		playingView = renderNowPlaying(&m, currentPosition, total)
	}

	controls := renderPlayerControls(&m)
	playingCombined := strings.TrimSpace(playingView) + "\n" + controls

	playing := getPlayerStyles(&m, dimensions).
		Foreground(playerFg).
		Render(playingCombined)
	var rightColumnView string
	showRelated := (m.RightColumnMode == RightColumnRelated || m.RightColumnMode == "") && len(m.RelatedList.Items()) > 0
	if showRelated {
		rightColumnView = m.RelatedList.View()
	} else if m.MusicQueueList != nil {
		rightColumnView = m.MusicQueueList.View()
	}
	queueList := getStyle(&m, dimensions.contentHeight, dimensions.sidebarWidth, QueueList, false).Render(rightColumnView)
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

type layoutDimensions struct {
	sidebarWidth  int
	mainWidth     int
	contentHeight int
	inputHeight   int
}

func calculateLayoutDimensions(m *Model) layoutDimensions {
	sidebarWidth := m.Width * 22 / 100
	inputHeight := min(max(m.Height*10/100, 2), 3)
	mainCenterArea := m.Width - (sidebarWidth * 2) - 2
	if mainCenterArea < 10 {
		mainCenterArea = 10
	}

	return layoutDimensions{
		sidebarWidth:  sidebarWidth,
		mainWidth:     mainCenterArea,
		contentHeight: m.Height * 90 / 100,
		inputHeight:   inputHeight,
	}
}

func removeListDefaults(listToRemoveDefaults *list.Model) {
	if listToRemoveDefaults != nil {
		listToRemoveDefaults.SetShowFilter(false)
		listToRemoveDefaults.SetShowPagination(false)
		listToRemoveDefaults.SetShowHelp(false)
		listToRemoveDefaults.SetShowStatusBar(false)
	}
}

func (m *Model) IsPlaying() bool {
	return m != nil && m.PlayerProcess != nil && m.PlayerProcess.OtoPlayer != nil && m.PlayerProcess.OtoPlayer.IsPlaying()
}
