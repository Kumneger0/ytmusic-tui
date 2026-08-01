package types // nolint:revive

import musicpb "github.com/kumneger0/ytmusic-tui/gen"

type SongItem struct {
	*musicpb.Song
}

func (s SongItem) FilterValue() string {
	if s.Song == nil {
		return ""
	}
	return s.Song.Title
}

type AlbumItem struct {
	*musicpb.Album
}

func (a AlbumItem) FilterValue() string {
	if a.Album == nil {
		return ""
	}
	return a.Album.Title + " (album)"
}

type PlaylistItem struct {
	*musicpb.Playlist
}

func (p PlaylistItem) FilterValue() string {
	if p.Playlist == nil {
		return ""
	}
	return p.Playlist.Title + " (playlist)"
}

type ArtistItem struct {
	*musicpb.Artist
}

func (a ArtistItem) FilterValue() string {
	if a.Artist == nil {
		return ""
	}
	return a.Artist.Name + " (artist)"
}

type SearchResultSongItem struct {
	*musicpb.SearchResultSong
}

func (s SearchResultSongItem) FilterValue() string {
	if s.SearchResultSong == nil {
		return ""
	}
	return s.SearchResultSong.Title
}

type SearchResultAlbumItem struct {
	*musicpb.SearchResultAlbum
}

func (s SearchResultAlbumItem) FilterValue() string {
	if s.SearchResultAlbum == nil {
		return ""
	}
	return s.SearchResultAlbum.Title + " (album)"
}

type SearchResultArtistItem struct {
	*musicpb.SearchResultArtist
}

func (s SearchResultArtistItem) FilterValue() string {
	if s.SearchResultArtist == nil {
		return ""
	}
	return s.SearchResultArtist.Name + " (artist)"
}

type SearchResultPlaylistItem struct {
	*musicpb.SearchResultPlaylist
}

func (s SearchResultPlaylistItem) FilterValue() string {
	if s.SearchResultPlaylist == nil {
		return ""
	}
	return s.SearchResultPlaylist.Title + " (playlist)"
}

type SearchResultPodcastItem struct {
	*musicpb.SearchResultPodcast
}

func (s SearchResultPodcastItem) FilterValue() string {
	if s.SearchResultPodcast == nil {
		return ""
	}
	return s.SearchResultPodcast.Title + " (podcast)"
}

type SearchResultEpisodeItem struct {
	*musicpb.SearchResultEpisode
}

func (s SearchResultEpisodeItem) FilterValue() string {
	if s.SearchResultEpisode == nil {
		return ""
	}
	return s.SearchResultEpisode.Title + " (episode)"
}

type SongRelatedContentItem struct {
	*musicpb.SongRelatedContent
}

func (s SongRelatedContentItem) FilterValue() string {
	if s.SongRelatedContent == nil {
		return ""
	}
	return s.SongRelatedContent.Title
}

func (f FollowedArtistItem) FilterValue() string {
	if f.FollowedArtist == nil {
		return ""
	}
	return f.FollowedArtist.Name + " (artist)"
}

type FollowedArtistItem struct {
	*musicpb.FollowedArtist
}

type LibraryChannelItem struct {
	*musicpb.LibraryChannel
}

func (c LibraryChannelItem) FilterValue() string {
	if c.LibraryChannel == nil {
		return ""
	}
	return c.LibraryChannel.Name + " (channel)"
}

type PodcastItem struct {
	*musicpb.Podcast
}

func (p PodcastItem) FilterValue() string {
	if p.Podcast == nil {
		return ""
	}
	return p.Podcast.Title + " (podcast)"
}
