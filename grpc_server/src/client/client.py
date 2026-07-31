from typing import cast, Any
from ytmusicapi.models.lyrics import Lyrics, TimedLyrics
from ytmusicapi import  YTMusic, LikeStatus
from yt_dlp import YoutubeDL

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
    client: YTMusic

    def __init__(self, auth_file: str) -> None:
        try:
            self.client = YTMusic(auth=auth_file)
        except Exception:
            self.client =YTMusic()
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
        res: object = self.client.get_home(limit=10)
        return cast(list[YTHomeSection], res)

    def get_library(self, limit: int = 25) -> YTLibraryResponse:
        albums = cast(list[YTLibraryAlbum], self.client.get_library_albums(limit))
        playlists = cast(list[YTLibraryPlaylist], self.client.get_library_playlists(limit))
        channels = cast(list[YTLibraryChannel], self.client.get_library_channels(limit))
        artists = cast(list[YTLibraryArtist], self.client.get_library_subscriptions(limit))
        podcasts = cast(list[YTLibraryPlaylist], self.client.get_library_podcasts(limit))
        return {
            "albums": albums,
            "playlists": playlists,
            "channels": channels,
            "artists": artists,
            "podcasts": podcasts,
        }
    

    def get_user_saved_tracks(self, limit: int = 100) -> list[YTSong]:
        raw_songs: object = self.client.get_liked_songs(limit)
        songs = cast(YTLikedSongsResponse, cast(object, raw_songs))
        tracks = songs.get('tracks')
        if isinstance(tracks, list):
            return tracks
        return []

    def get_user_saved_albums(self, limit: int = 25) -> list[YTLibraryAlbum]:
        raw_albums: object = self.client.get_library_albums(limit=limit)
        return cast(list[YTLibraryAlbum], raw_albums)

    def get_user_playlists(self, limit: int = 25) -> list[YTLibraryPlaylist]:
        raw_playlists: object = self.client.get_library_playlists(limit=limit)
        return cast(list[YTLibraryPlaylist], raw_playlists)

    def get_track(self, video_id: str) -> YTSongResponse:
        raw_song: object = self.client.get_song(videoId=video_id)
        song_dict = cast(dict[str, object], raw_song)
        video_details = song_dict.get("videoDetails")
        if not isinstance(video_details, dict):
            return cast(YTSongResponse, cast(object, {}))
        track: YTSongResponse = cast(YTSongResponse, cast(object, video_details))
        track_video_id = track.get('videoId')
        if isinstance(track_video_id, str):
                stream_url_and_duration = self.get_stream_url_and_duration(video_id=track_video_id)
                track['url'] = stream_url_and_duration.get('url')
                if stream_url_and_duration.get('duration') != None:
                    track['lengthSeconds'] = str(stream_url_and_duration.get('duration'))
        
        return track

    def get_album_tracks(self, browse_id: str) -> YTAlbumResponse:
        raw_album: object = self.client.get_album(browseId=browse_id)
        return cast(YTAlbumResponse, cast(object, raw_album))

    def get_playlist_items(self, playlist_id: str, limit: int = 100) -> YTLikedSongsResponse:
        raw_playlist: object = self.client.get_playlist(playlistId=playlist_id, limit=limit)
        return cast(YTLikedSongsResponse, cast(object, raw_playlist))

    def get_search_results(self, query: str, filter_type: YTSearchFilter | None = None, limit: int = 20) -> list[YTSearchResult]:
        raw_results: object = self.client.search(query=query, filter=filter_type, limit=limit)
        return cast(list[YTSearchResult], raw_results)

    def get_artist_top_tracks(self, channel_id: str) -> YTArtistResponse:
        raw_artist: object = self.client.get_artist(channelId=channel_id)
        return cast(YTArtistResponse, cast(object, raw_artist))

    def get_followed_artists(self, limit: int = 25) -> list[YTLibraryArtist]:
        raw_artists: object = self.client.get_library_subscriptions(limit=limit)
        return cast(list[YTLibraryArtist], raw_artists)

    def get_user_profile(self) -> YTAccountInfo:
            raw_info: object = self.client.get_account_info()
            return cast(YTAccountInfo, cast(object, raw_info))

    def get_user_top_items(self) -> list[YTSong]:
        raw_history: object = self.client.get_history()
        return cast(list[YTSong], raw_history)

    def check_user_saved_track(self, video_id: str) -> bool:
        try:
            raw_song = cast(dict[str, object], self.client.get_song(videoId=video_id))
            if raw_song:
                return True
        except Exception:
            pass
        return False

    def save_remove_track(self, video_ids: list[str], is_remove: bool) -> None:
        rating = LikeStatus.INDIFFERENT if is_remove else LikeStatus.LIKE
        for video_id in video_ids:
            _ = self.client.rate_song(videoId=video_id, rating=rating)

    def search(self, query: str) -> list[YTSong]:
        res = self.client.search(
            query,
            filter="songs"
        )
        return cast(list[YTSong], res)

    def like_song(self, video_id: str) -> object:
        return self.client.rate_song(
            video_id,
            LikeStatus.LIKE
        )

    def unlike_song(self, video_id: str) -> object:
        return self.client.rate_song(
            video_id,
            LikeStatus.INDIFFERENT
        )

    def get_song_related(self, browse_id: str) -> list[dict[str, Any]]:
        raw_related: object = self.client.get_song_related(browseId=browse_id)
        return cast(list[dict[str, Any]], cast(object, raw_related))

    def get_lyrics(self, browse_id: str, timestamps: bool = False) -> Lyrics | TimedLyrics | None:
        if timestamps:
            try:
                return self.client.get_lyrics(browseId=browse_id, timestamps=True)
            except (KeyError, Exception):
                result = self._get_timed_lyrics_raw(browse_id)
                if result is not None:
                    return result

        try:
            return self.client.get_lyrics(browseId=browse_id, timestamps=False)
        except Exception:
            return None

    def _get_timed_lyrics_raw(self, browse_id: str) -> TimedLyrics | None:
        """Parse timed lyrics directly from the mobile API response."""
        try:
            from ytmusicapi.mixins.browsing import TIMESTAMPED_LYRICS
            from ytmusicapi.navigation import nav
            from ytmusicapi.models.lyrics import LyricLine

            with self.client.as_mobile():
                response = self.client._send_request("browse", {"browseId": browse_id})

            data = nav(response, TIMESTAMPED_LYRICS, True)
            if not isinstance(data, dict) or "timedLyricsData" not in data:
                return None

            lines: list[LyricLine] = []
            for item in data["timedLyricsData"]:
                text = item.get("lyricLine", "")
                cue = item.get("cueRange") or {}
                start = int(cue.get("startTimeMilliseconds", 0)) if isinstance(cue, dict) else 0
                end = int(cue.get("endTimeMilliseconds", 0)) if isinstance(cue, dict) else 0
                meta = cue.get("metadata") if isinstance(cue, dict) else None
                lid = int(meta["id"]) if isinstance(meta, dict) and "id" in meta else 0
                lines.append(LyricLine(text=text, start_time=start, end_time=end, id=lid))

            return TimedLyrics(
                lyrics=lines,
                source=data.get("sourceMessage"),
                hasTimestamps=True,
            )
        except Exception:
            return None

    def get_watch_playlist(self, video_id: str) -> dict[str, Any]:
        raw_watch: object = self.client.get_watch_playlist(videoId=video_id)
        if isinstance(raw_watch, dict):
            return cast(dict[str, Any], raw_watch)
        return {}