package types // nolint:revive

import (
	musicpb "github.com/kumneger0/ytmusic-tui/gen"
)

type PlaylistItemsResponse struct {
	Total int                    `json:"total"`
	Items []*PlaylistTrackObject `json:"items"`
}

type PlaylistTrackObject struct {
	Track          *musicpb.Song `json:"track"`
	IsItFromQueue  bool          `json:"isItFromQueue"`
	IsItFromSearch bool          `json:"-"`
}

func (playlist PlaylistTrackObject) FilterValue() string {
	if playlist.Track == nil {
		return ""
	}
	return playlist.Track.Title
}

func (playlist PlaylistTrackObject) Title() string {
	if playlist.Track == nil {
		return ""
	}
	return playlist.Track.Title
}
