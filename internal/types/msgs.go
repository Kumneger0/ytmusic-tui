package types // nolint:revive

import (
	"io"
	"log/slog"
	"sync/atomic"

	"github.com/ebitengine/oto/v3"
	musicpb "github.com/kumneger0/ytmusic-tui/gen"
)

type NextPageURLType string

const (
	NextPageURLTypePlaylistTracks NextPageURLType = "playlistTracks"
	NextPageURLTypeUserSavedItems NextPageURLType = "userSavedItems"
)

type PaginationInfo struct {
	Next            string
	NextPageURLType NextPageURLType
	NextItemID      string
}

type UpdatePlaylistMsg struct {
	Playlist       []*PlaylistTrackObject
	Err            error
	PaginationInfo *PaginationInfo
}

var PlayedSecondsUpdateChan = make(chan PlayedSecondsUpdateMsg)

type PlayedSecondsUpdateMsg struct {
	CurrentSeconds float64
}

type MessageType string

const (
	NextTrack     MessageType = "nextTrack"
	PreviousTrack MessageType = "previousTrack"
	PlayPause     MessageType = "playPause"
)

type DBusMessage struct {
	MessageType
}

type SearchingMsg struct{}

type SearchResultMsg struct {
	Result *SearchResponse
	Err    error
}

type PythonBackendHealthResponseMsg struct {
	Response *musicpb.HealthCheckResponse
	Err      error
}

type CheckUserSavedTrackResponseMsg struct {
	Saved bool
	Err   error
}

type LikeUnlikeTrackResponseMsg struct {
	TrackID string
	Liked   bool
	Err     error
}

type SearchAndDownloadMusicMsg struct {
	Player   *Player
	VideoID  string
	Err      error
	Duration string
}

type Player struct {
	OtoPlayer         *oto.Player
	Close             func() error
	ByteCounterReader *ByteCounterReader
}

type ByteCounterReader struct {
	R     io.Reader
	total int64
}

func (b *ByteCounterReader) Read(p []byte) (int, error) {
	n, err := b.R.Read(p)
	if n > 0 {
		atomic.AddInt64(&b.total, int64(n))
		currentSeconds := b.CurrentSeconds()
		go func() {
			PlayedSecondsUpdateChan <- PlayedSecondsUpdateMsg{
				CurrentSeconds: currentSeconds,
			}
		}()
	}
	if err != nil && err != io.EOF {
		slog.Error(err.Error())
	}
	return n, err
}

func (b *ByteCounterReader) CurrentSeconds() float64 {
	return float64(atomic.LoadInt64(&b.total)) / 176400.0
}

type HomePageResponseMsg struct {
	Response *musicpb.GetHomePageResponse
	Err      error
}

type UpdateHomePageContentMsg struct {
	Item HomePageSectionItem
}

type PlaylistDetailMsg struct {
	Playlist *musicpb.GetPlaylistItemsResponse
	Err      error
}

type GetLibraryMsg struct {
	Result *musicpb.GetLibraryResponse
	Err    error
}

type RelatedSongsMsg struct {
	Related *musicpb.GetSongRelatedResponse
	Err     error
}
type WatchPlaylistItemsMsg struct {
	SourceID           string
	WatchPlaylistItems *musicpb.GetWatchPlaylistItemsResponse
	Err                error
}

type LyricsMsg struct {
	LyricsResponse *musicpb.GetLyricsResponse
	Err            error
}

type ModalType int

const (
	ModalTypeNone ModalType = iota
	ModalTypeCreatePlaylist
	ModalTypePlaylistManagement
	ModalTypeDuplicateConfirm
)

type OpenModalMsg struct {
	ModalType ModalType
}

type CloseModalMsg struct {
	ModalType ModalType
}

type CreatePlaylistMsg struct {
	Title         string
	Description   string
	PrivacyStatus string
	VideoIDs      []string
}

type CreatePlaylistResponseMsg struct {
	PlaylistID string
	Success    bool
	Err        error
}

type OpenAddToPlaylistLoadingMsg struct {
	TrackID    string
	TrackTitle string
}

type OpenAddToPlaylistModalMsg struct {
	TrackID    string
	TrackTitle string
	Playlists  []*musicpb.Playlist
	Membership map[string]string
	Err        error
}

type AddToPlaylistMsg struct {
	PlaylistID   string
	PlaylistName string
	TrackID      string
	TrackTitle   string
	Duplicates   bool
}

type AddToPlaylistResponseMsg struct {
	PlaylistID   string
	PlaylistName string
	TrackID      string
	TrackTitle   string
	Status       string
	Success      bool
	IsDuplicate  bool
	Err          error
}

type RemoveFromPlaylistMsg struct {
	PlaylistID   string
	PlaylistName string
	TrackID      string
	TrackTitle   string
}

type RemoveFromPlaylistResponseMsg struct {
	PlaylistID   string
	PlaylistName string
	TrackID      string
	TrackTitle   string
	Success      bool
	Err          error
}

type PromptDuplicateConfirmMsg struct {
	PlaylistID   string
	PlaylistName string
	TrackID      string
	TrackTitle   string
}
