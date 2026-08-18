"""TypedDict definitions matching ytmusicapi return structures."""

from typing import TypedDict, Literal

# Search filters supported by ytmusicapi
YTSearchFilter = Literal["songs", "videos", "albums", "artists", "playlists", "podcasts", "episodes"]


class YTThumbnail(TypedDict, total=False):
    url: str
    width: int
    height: int


class YTArtist(TypedDict, total=False):
    id: str | None
    name: str


class YTAlbumInfo(TypedDict, total=False):
    """Album reference embedded inside a song/track dict."""
    id: str | None
    name: str


class YTSong(TypedDict, total=False):
    """A song/track item as returned by get_playlist, get_liked_songs, get_history, search, etc."""
    videoId: str | None
    title: str
    artists: list[YTArtist]
    album: YTAlbumInfo | str | None
    duration: str | None
    duration_seconds: int | None
    likeStatus: str | None
    thumbnails: list[YTThumbnail]
    isExplicit: bool | None
    isAvailable: bool | None
    played: str | None  # present in history items


class YTLikedSongsResponse(TypedDict, total=False):
    """Return type of get_liked_songs / get_playlist."""
    id: str
    title: str
    description: str | None
    author: dict[str, str] | str | None
    year: str | None
    trackCount: int | None
    duration: str | None
    duration_seconds: int | None
    thumbnails: list[YTThumbnail]
    tracks: list[YTSong]

class YTWatchPlaylistResponse(TypedDict, total=False):
    playlistId:str
    tracks: list[YTSong]    


class YTLibraryAlbum(TypedDict, total=False):
    """An album item as returned by get_library_albums."""
    browseId: str
    playlistId: str | None
    title: str
    type: str | None
    thumbnails: list[YTThumbnail]
    artists: list[YTArtist]
    year: str | None
    isExplicit: bool | None


class YTLibraryPlaylist(TypedDict, total=False):
    """A playlist item as returned by get_library_playlists."""
    playlistId: str
    title: str
    thumbnails: list[YTThumbnail]
    count: int | str | None
    description: str | None
    author: list[YTArtist] | str | None


class YTLibraryChannel(TypedDict, total=False):
    """A channel item as returned by get_library_channels."""
    browseId: str
    artist: str
    subscribers: str | None
    thumbnails: list[YTThumbnail]


class YTLibraryArtist(TypedDict, total=False):
    """An artist item as returned by get_library_subscriptions."""
    browseId: str
    artist: str
    subscribers: str | None
    thumbnails: list[YTThumbnail]


class YTLibraryResponse(TypedDict, total=False):
    """Return type of get_library bundling albums, playlists, channels, artists, and podcasts."""
    albums: list[YTLibraryAlbum]
    playlists: list[YTLibraryPlaylist]
    channels: list[YTLibraryChannel]
    artists: list[YTLibraryArtist]
    podcasts: list[YTLibraryPlaylist]

class GetStreamURLResponse(TypedDict, total=False):
    url:str
    duration: int | None


class YTAlbumResponse(TypedDict, total=False):
    """Return type of get_album."""
    title: str
    type: str | None
    thumbnails: list[YTThumbnail]
    description: str | None
    artists: list[YTArtist]
    year: str | None
    trackCount: int | None
    duration: str | None
    duration_seconds: int | None
    audioPlaylistId: str | None
    tracks: list[YTSong]


class YTArtistSongsSection(TypedDict, total=False):
    browseId: str | None
    results: list[YTSong]


class YTArtistResponse(TypedDict, total=False):
    """Return type of get_artist."""
    name: str
    channelId: str | None
    description: str | None
    subscribers: str | None
    thumbnails: list[YTThumbnail]
    songs: YTArtistSongsSection


class YTAccountInfo(TypedDict, total=False):
    """Return type of get_account_info."""
    accountName: str
    channelHandle: str | None
    accountPhotoUrl: str | None


class YTSearchResult(TypedDict, total=False):
    """A mixed/filtered search result from search()."""
    category: str | None
    resultType: str | None
    videoId: str | None
    title: str
    artists: list[YTArtist]
    album: YTAlbumInfo | str | None
    duration: str | None
    duration_seconds: int | None
    thumbnails: list[YTThumbnail]
    isExplicit: bool | None
    browseId: str
    year: str | None
    type: str | None
    artist: str
    subscribers: str | None
    author: str | None
    itemCount: str | None


class YTHomeSection(TypedDict, total=False):
    title: str
    contents: list[dict[str, object]]


class YTSongResponse(TypedDict, total=False):
    """Return type of get_song (videoDetails sub-dict)."""
    videoId: str
    title: str
    lengthSeconds: str | None
    channelId: str | None
    author: str | None
    url: str | None
    thumbnail: dict[str, list[YTThumbnail]] | None
