import threading
from typing import cast, Callable, TypeVar, Any
from ytmusicapi.type_alias import JsonDict
from ytmusicapi.models.lyrics import Lyrics, TimedLyrics
from ytmusicapi import YTMusic, LikeStatus
from yt_dlp import YoutubeDL

T = TypeVar("T")

from .types import (
    GetStreamURLResponse,
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

class MusicClient:
    def __init__(self, auth_file: str | None = None) -> None:
        self.auth_file: str | None = auth_file
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
        if self.auth_file:
            try:
                self._client = YTMusic(auth=self.auth_file)
                self._generation += 1
                return
            except Exception as e:
                print(f"Error initializing YTMusic with auth_file {self.auth_file}: {e}")
        self._client = YTMusic()
        self._generation += 1

    def reset_client(self, known_generation: int | None = None) -> None:
        with self._lock:
            if known_generation is not None and known_generation != self._generation:
                return  # Another thread already reset the client for this failure.
            print("Resetting YTMusic client due to error/session corruption...")
            self._init_client()

    def execute(self, func: Callable[[YTMusic], T]) -> T:
        generation = self._generation
        try:
            return func(self.get_client())
        except Exception as e:
            print(f"YTMusic request execution failed ({e}). Resetting client and retrying once...")
            self.reset_client(known_generation=generation)
            return func(self.get_client())
    def get_stream_url_and_duration(self, video_id:str)  -> GetStreamURLResponse:
        full_url: str = "https://www.youtube.com/watch?v=" + video_id
        options = {
            "format": "bestaudio[abr>=120][abr<=250]/bestaudio",
              "format_sort": ["abr"],
             "quiet": True,
         }
        with YoutubeDL(options) as ydl:  # pyright: ignore[reportArgumentType]
            info = ydl.extract_info(
                full_url,
                download=False,
            )
        url = info.get("url")
        duration: int | None = info.get("duration") 
        if not isinstance(url, str):        
            raise RuntimeError("Unable to extract stream URL")
        return {
            "url":url,
            "duration" : duration,
        }

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

    def get_track(self, video_id: str) -> YTSongResponse:
        def _fetch(c: YTMusic) -> YTSongResponse:
            raw_song: object = c.get_song(videoId=video_id)
            song_dict = cast(dict[str, object], raw_song)
            video_details = song_dict.get("videoDetails")
            if not isinstance(video_details, dict):
                return cast(YTSongResponse, cast(object, {}))
            return cast(YTSongResponse, cast(object, video_details))

        track = self.execute(_fetch)
        track_video_id = track.get("videoId")
        if isinstance(track_video_id, str):
            try:
                stream_url_and_duration = self.get_stream_url_and_duration(video_id=track_video_id)
                track["url"] = stream_url_and_duration.get("url")
                if stream_url_and_duration.get("duration") is not None:
                    track["lengthSeconds"] = str(stream_url_and_duration.get("duration"))
            except Exception as e:
                print(f"Error extracting stream url for track {track_video_id}: {e}")
        return track

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
        return self.execute(lambda c: cast(YTAccountInfo, cast(object, c.get_account_info())))

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
        except Exception as e:
            print(f"Error in get_song_related: {e}")
            return []

    def get_lyrics(self, browse_id: str, timestamps: bool = False) -> Lyrics | TimedLyrics | dict[str, object] | None:
        if timestamps:
            result = self._get_timed_lyrics_raw(browse_id)
            if result is not None:
                return result
            try:
                res = self.execute(lambda c: c.get_lyrics(browseId=browse_id, timestamps=True))
                if res and (res.get("hasTimestamps") or res.get("lyrics")):
                    return cast(dict[str, object], cast(object, res))
            except Exception as e:
                print(f"Failed to fetch timed lyrics via web API: {e}")

        try:
            return self.execute(lambda c: c.get_lyrics(browseId=browse_id, timestamps=False))
        except Exception as e:
            print(f"Failed to fetch plain lyrics: {e}")
            return None

    def _get_timed_lyrics_raw(self, browse_id: str) -> TimedLyrics | None:
        """Parse timed lyrics directly from the mobile API response."""
        def _fetch_mobile(c: YTMusic) -> TimedLyrics | None:
            from ytmusicapi.mixins.browsing import TIMESTAMPED_LYRICS
            from ytmusicapi.navigation import nav
            from ytmusicapi.models.lyrics import LyricLine

            with c.as_mobile():
                response = c._send_request("browse", {"browseId": browse_id})  # pyright: ignore[reportPrivateUsage]

            data = nav(response, TIMESTAMPED_LYRICS, True)
            if not isinstance(data, dict):
                return None

            data_map = cast(dict[str, object], data)
            timed_lyrics_data = data_map.get("timedLyricsData")
            if not isinstance(timed_lyrics_data, list):
                return None

            lines: list[LyricLine] = []
            for item in cast(list[object], timed_lyrics_data):
                if not isinstance(item, dict):
                    continue    
                item_map = cast(dict[str, object], item)
                text = str(item_map.get("lyricLine") or "")
                cue = item_map.get("cueRange")
                start = 0
                end = 0
                lid = 0
                if isinstance(cue, dict):
                    cue_map = cast(dict[str, object], cue)
            def _coerce_int(value: object) -> int:
                try:
                    return int(str(value))
                except (TypeError, ValueError):
                    return 0

                    start = _coerce_int(cue_map.get("startTimeMilliseconds"))
                    end = _coerce_int(cue_map.get("endTimeMilliseconds"))
                    metadata = cue_map.get("metadata")
                    if isinstance(metadata, dict):
                        metadata_map = cast(dict[str, object], metadata)
                        lid = _coerce_int(metadata_map.get("id"))
                lines.append(LyricLine(text=text, start_time=start, end_time=end, id=lid))

            return TimedLyrics(
                lyrics=lines,
                source=str(data_map.get("sourceMessage") or "") if data_map.get("sourceMessage") is not None else None,
                hasTimestamps=True,
            )
        try:
            return self.execute(_fetch_mobile)
        except Exception:
            return None

    def get_watch_playlist(self, video_id: str) -> dict[str, object]:
        if not video_id:
            return {}
        try:
            res = self.execute(lambda c: c.get_watch_playlist(videoId=video_id))
            return cast(dict[str, object], res)
        except Exception as e:
            print(f"Error in get_watch_playlist: {e}")
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
        except Exception as e:
            print(f"create_playlist failed ({e}). Resetting client without retry...")
            self.reset_client()
            raise