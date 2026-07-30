package types

import (
	musicpb "github.com/kumneger0/ytmusic-tui/gen"
)

func MapThumbnailToImage(pb *musicpb.Thumbnail) Image {
	if pb == nil {
		return Image{}
	}
	return Image{
		URL:    pb.Url,
		Width:  int(pb.Width),
		Height: int(pb.Height),
	}
}

func MapThumbnailsToImages(pbs []*musicpb.Thumbnail) []Image {
	var imgs []Image
	for _, pb := range pbs {
		imgs = append(imgs, MapThumbnailToImage(pb))
	}
	return imgs
}

func MapArtistToArtist(pb *musicpb.Artist) Artist {
	if pb == nil {
		return Artist{}
	}
	return Artist{
		ID:   pb.Id,
		Name: pb.Name,
	}
}

func MapArtistsToArtists(pbs []*musicpb.Artist) []Artist {
	var arts []Artist
	for _, pb := range pbs {
		arts = append(arts, MapArtistToArtist(pb))
	}
	return arts
}

func MapAlbumToAlbum(pb *musicpb.Album) Album {
	if pb == nil {
		return Album{}
	}
	return Album{
		ID:       pb.BrowseId,
		Name:     pb.Title,
		Artists:  MapArtistsToArtists(pb.Artists),
		Images:   MapThumbnailsToImages(pb.Thumbnails),
		Year:     pb.Year,
		Type:     pb.Type,
		Explicit: pb.IsExplicit,
	}
}

func MapSongToTrack(pb *musicpb.Song) Track {
	if pb == nil {
		return Track{}
	}
	return Track{
		ID:      pb.VideoId,
		Name:    pb.Title,
		Artists: MapArtistsToArtists(pb.Artists),
		Album: Album{
			ID:   pb.AlbumId,
			Name: pb.Album,
		},
		DurationMS: int(pb.DurationSeconds * 1000),
		Explicit:   pb.IsExplicit,
		URL:        pb.Url,
	}
}

func MapSearchResultSongToTrack(pb *musicpb.SearchResultSong) Track {
	if pb == nil {
		return Track{}
	}
	return Track{
		ID:      pb.VideoId,
		Name:    pb.Title,
		Artists: MapArtistsToArtists(pb.Artists),
		Album: Album{
			ID:   pb.AlbumId,
			Name: pb.Album,
		},
		DurationMS: int(pb.DurationSeconds * 1000),
		Explicit:   pb.IsExplicit,
		URL:        pb.Url,
	}
}

func MapPlaylistToPlaylist(pb *musicpb.Playlist) Playlist {
	if pb == nil {
		return Playlist{}
	}
	return Playlist{
		ID:          pb.PlaylistId,
		Name:        pb.Title,
		Description: pb.Description,
		Images:      MapThumbnailsToImages(pb.Thumbnails),
		Count:       int(pb.Count),
		Author:      pb.Author,
	}
}
