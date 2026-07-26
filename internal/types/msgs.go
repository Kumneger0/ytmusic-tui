package types // nolint:revive

import (
	"io"
	"log/slog"
	"sync/atomic"

	"github.com/ebitengine/oto/v3"
	musicpb "github.com/kumneger0/clispot/gen"
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
	Playlist          []*PlaylistTrackObject
	Err               error
	PaginationInfo    *PaginationInfo
	ShouldAppendQueue bool
	ShouldAppend      bool
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

type SpotifySearchResultMsg struct {
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

type LikeUnlikeTrackMsg struct {
	TrackID string
	Like    bool
	Err     error
}

type SearchAndDownloadMusicMsg struct {
	Player  *Player
	VideoID string
	Err     error
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
