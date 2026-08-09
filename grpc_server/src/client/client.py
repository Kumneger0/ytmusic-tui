import json
import os
import threading
from typing import TypeAlias, cast, Callable, TypeVar, Any
from click import Path
from ytmusicapi.models.lyrics import Lyrics, TimedLyrics
from ytmusicapi.type_alias import JsonDict
from ytmusicapi import YTMusic, LikeStatus

T = TypeVar("T")

from .types import (
    YTSong,
    YTLikedSongsResponse,
    YTHomeSection,
    YTLibraryAlbum,
    YTLibraryPlaylist,
    YTLibraryChannel,
    YTLibraryResponse,
    YTAlbumResponse,
    YTLibraryArtist,
    YTAccountInfo,
    YTSongResponse,
    YTArtistResponse,
    YTSearchResult,
    YTSearchFilter
)

JSONValue: TypeAlias = (
    str
    | int
    | float
    | bool
    | None
    | list["JSONValue"]
    | dict[str, "JSONValue"]
)

JSONDict = dict[str, JSONValue]


def get_ytmusic_client(auth_input: str | JsonDict | Path | None = None) -> YTMusic:
    """
    Creates a YTMusic instance from a file path, raw JSON string, or dict object.
    Supports in-memory / virtual auth without requiring disk persistence.
    """
    if not auth_input:
        return YTMusic()

    if isinstance(auth_input, Path):
        return YTMusic(auth=str(auth_input))

    if isinstance(auth_input, str):
        trimmed = auth_input.strip()
        if trimmed.startswith("{") and trimmed.endswith("}"):
            try:
                _ = json.loads(trimmed)
                return YTMusic(auth=trimmed)
            except Exception as e:
                print(f"Error parsing auth JSON string: {e}")
                return YTMusic()
        elif os.path.isfile(trimmed):
            return YTMusic(auth=trimmed)
        else:
            try:
                return YTMusic(auth=trimmed)
            except Exception:
                return YTMusic()

    return YTMusic(auth=auth_input)





class MusicClient:
    def __init__(self, auth: str | JsonDict | None = None) -> None:
        self.auth: str | JsonDict | None = auth
        self._client: YTMusic | None = None
        self._lock = threading.Lock()
        self._generation: int = 0

    def get_client(self) -> YTMusic:
        if self._client is None:
            with self._lock:
                if self._client is None:
                    self._init_client()
        assert self._client is not None
        return self._client

    def _init_client(self) -> None:
        if self.auth:
            try:
                self._client = get_ytmusic_client(self.auth)
                self._generation += 1
                return
            except Exception:
                pass
        self._client = YTMusic()
        self._generation += 1

    def reset_client(self, known_generation: int | None = None) -> None:
        with self._lock:
            if known_generation is not None and known_generation != self._generation:
                return
            self._init_client()

    def execute(self, func: Callable[[YTMusic], T]) -> T:
        generation = self._generation
        try:
            return func(self.get_client())
        except Exception:
            self.reset_client(known_generation=generation)
            return func(self.get_client())

    def get_home(self) -> list[YTHomeSection]:
        return self.execute(lambda c: cast(list[YTHomeSection], c.get_home(limit=10)))

    def get_library(self, limit: int = 25) -> YTLibraryResponse:
        def _fetch(c: YTMusic) -> YTLibraryResponse:
            albums = cast(list[YTLibraryAlbum], c.get_library_albums(limit))
            playlists = cast(list[YTLibraryPlaylist], c.get_library_playlists(limit))
            channels = cast(list[YTLibraryChannel], c.get_library_channels(limit))
            artists = cast(list[YTLibraryArtist], c.get_library_subscriptions(limit))
            podcasts = cast(list[YTLibraryPlaylist], c.get_library_podcasts(limit))
            return {
                "albums": albums,
                "playlists": playlists,
                "channels": channels,
                "artists": artists,
                "podcasts": podcasts,
            }
        return self.execute(_fetch)

    def get_user_saved_tracks(self, limit: int = 100) -> list[YTSong]:
        def _fetch(c: YTMusic) -> list[YTSong]:
            raw_songs: object = c.get_liked_songs(limit)
            songs = cast(YTLikedSongsResponse, cast(object, raw_songs))
            tracks = songs.get("tracks")
            if isinstance(tracks, list):
                return tracks
            return []
        return self.execute(_fetch)

    def get_user_saved_albums(self, limit: int = 25) -> list[YTLibraryAlbum]:
        return self.execute(lambda c: cast(list[YTLibraryAlbum], c.get_library_albums(limit=limit)))

    def get_user_playlists(self, limit: int = 25) -> list[YTLibraryPlaylist]:
        return self.execute(lambda c: cast(list[YTLibraryPlaylist], c.get_library_playlists(limit=limit)))

    def get_track(self, video_id: str, user_cookie: str | None = None) -> YTSongResponse:
        def _fetch(c: YTMusic) -> YTSongResponse:
            raw_song: object = c.get_song(videoId=video_id)
            song_dict = cast(dict[str, object], raw_song)
            video_details = song_dict.get("videoDetails")
            if not isinstance(video_details, dict):
                return cast(YTSongResponse, cast(object, {}))
            return cast(YTSongResponse, cast(object, video_details))

        return self.execute(_fetch)

    def get_album_tracks(self, browse_id: str) -> YTAlbumResponse:
        return self.execute(lambda c: cast(YTAlbumResponse, cast(object, c.get_album(browseId=browse_id))))

    def get_playlist_items(self, playlist_id: str, limit: int = 100) -> YTLikedSongsResponse:
        return self.execute(lambda c: cast(YTLikedSongsResponse, cast(object, c.get_playlist(playlistId=playlist_id, limit=limit))))

    def get_search_results(self, query: str, filter_type: YTSearchFilter | None = None, limit: int = 20) -> list[YTSearchResult]:
        return self.execute(lambda c: cast(list[YTSearchResult], c.search(query=query, filter=filter_type, limit=limit)))

    def get_artist_top_tracks(self, channel_id: str) -> YTArtistResponse:
        return self.execute(lambda c: cast(YTArtistResponse, cast(object, c.get_artist(channelId=channel_id))))

    def get_followed_artists(self, limit: int = 25) -> list[YTLibraryArtist]:
        return self.execute(lambda c: cast(list[YTLibraryArtist], c.get_library_subscriptions(limit=limit)))

    def get_user_profile(self) -> YTAccountInfo:
        try:
            return self.execute(lambda c: cast(YTAccountInfo, cast(object, c.get_account_info())))
        except Exception:
            return cast(YTAccountInfo, cast(object, {}))

    def get_user_top_items(self) -> list[YTSong]:
        return self.execute(lambda c: cast(list[YTSong], c.get_history()))

    def check_user_saved_track(self, video_id: str) -> bool:
        if not video_id:
            return False

        def _check(c: YTMusic) -> bool:
            raw_playlist = cast(object, c.get_watch_playlist(videoId=video_id))
            if not isinstance(raw_playlist, dict):
                return False
            watch_playlist = cast(dict[str, object], raw_playlist)
            tracks = cast(list[object] | None, watch_playlist.get("tracks"))
            if isinstance(tracks, list) and len(tracks) > 0:
                first_track: object = tracks[0]
                if isinstance(first_track, dict):
                    track_dict = cast(dict[str, object], first_track)
                    return str(track_dict.get("likeStatus") or "") == "LIKE"
            return False

        try:
            return self.execute(_check)
        except Exception:
            return False

    def save_remove_track(self, video_ids: list[str], is_remove: bool)  -> JsonDict | None:
        rating = LikeStatus.INDIFFERENT if is_remove else LikeStatus.LIKE
        print(f"save_remove_track: video_ids={video_ids}, is_remove={is_remove}, rating={rating}")
        last_res: JsonDict | None = None
        for video_id in video_ids:
            res = self.execute(lambda c, vid=video_id: c.rate_song(videoId=vid, rating=rating))
            if isinstance(res, dict):
                last_res = res
        return last_res

    def search(self, query: str) -> list[YTSong]:
        return self.execute(lambda c: cast(list[YTSong], c.search(query, filter="songs")))

    def like_song(self, video_id: str) -> object:
        print(f"like_song: video_id={video_id}")
        return self.execute(lambda c: c.rate_song(video_id, LikeStatus.LIKE))

    def unlike_song(self, video_id: str) -> object:
        print(f"unlike_song: video_id={video_id}")
        return self.execute(lambda c: c.rate_song(video_id, LikeStatus.INDIFFERENT))

    def get_song_related(self, browse_id: str) -> list[dict[str, object]]:
        if not browse_id or not browse_id.startswith("MPTRt_"):
            return []
        try:
            return self.execute(lambda c: cast(list[dict[str, object]], cast(object, c.get_song_related(browseId=browse_id))))
        except Exception:
            return []

    def get_lyrics(self, browse_id: str, timestamps: bool = False)  -> Lyrics | TimedLyrics | None:
        try:
            res = self.execute(lambda c: c.get_lyrics(browseId=browse_id, timestamps=timestamps))
            if res is not None:
                return res
        except Exception as e:
            import traceback
            traceback.print_exc()
            print(f"failed to fetch timed lyrics via web API: {e}")

        try:
            return self.execute(lambda c: c.get_lyrics(browseId=browse_id, timestamps=not timestamps))
        except Exception as e:
            import traceback
            traceback.print_exc()
            print(f"oops failed to fetch lyrics: {e}")
            return None

    def get_watch_playlist(self, video_id: str) -> dict[str, object]:
        if not video_id:
            return {}
        try:
            res = self.execute(lambda c: c.get_watch_playlist(videoId=video_id))
            return cast(dict[str, object], res)
        except Exception:
            return {}

    def create_playlist(
        self,
        title: str,
        description: str,
        privacy_status: str = "PRIVATE",
        video_ids: list[str] | None = None,
        source_playlist: str | None = None,
    ):
        try:
            return self.get_client().create_playlist(
                    title=title,
                    description=description,
                    privacy_status=privacy_status,
                    video_ids=video_ids,
                    source_playlist=source_playlist,
            )
        except Exception:
            self.reset_client()
            raise

    def add_playlist_items(
        self,
        playlist_id: str,
        video_ids: list[str] | None = None,
        source_playlist: str | None = None,
        duplicates: bool = False,
    )  :
        try:
            return self.get_client().add_playlist_items(
                    playlistId=playlist_id,
                    videoIds=video_ids,
                    source_playlist=source_playlist,
                    duplicates=duplicates,
            )
        except Exception as e:
            self.reset_client()
            raise

    def remove_playlist_items(
        self,
        playlist_id: str,
        videos: list[dict[str, Any]],
    ) :
        try:
            return self.get_client().remove_playlist_items(
                    playlistId=playlist_id,
                    videos=videos,
            )
        except Exception:
            self.reset_client()
            raise