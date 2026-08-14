import asyncio
import hashlib
import os
import sys
from typing import cast, override

try:
    from dotenv import load_dotenv

    _ = load_dotenv()
except ImportError:
    pass

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "gen")))
sys.path.insert(0, os.path.abspath(os.path.dirname(__file__)))
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

from connectrpc.method import REQ, RES
import uvicorn
from connectrpc.request import RequestContext
from .gen import music_pb2
from gen.music_connect import MusicService, MusicServiceASGIApplication
from .src.client.client import (
    MusicClient,
    get_ytmusic_client,
)
from .src.client.types import (
    YTHomeSection,
    YTSearchFilter,
    YTSearchResult,
    YTThumbnail,
)

from .src.mappers import (
    coerce_int,
    coerce_str,
    parse_auth_metadata,
    to_proto_album,
    to_proto_artist,
    to_proto_channel,
    to_proto_followed_artist,
    to_proto_playlist,
    to_proto_podcast,
    to_proto_song,
    to_proto_song_related_content,
    to_proto_thumbnail,
)


class Service(MusicService):
    def __init__(self) -> None:
        self._clients: dict[str, MusicClient] = {}

    def _get_client_for_request(self, ctx: RequestContext[REQ, RES] | None) -> MusicClient:
        auth_data = ""
        if ctx is not None:
            raw_auth = ctx.request_headers.get("x-auth-json")
            if raw_auth:
                auth_data = parse_auth_metadata(raw_auth)

        if not auth_data:
            cache_key = ""
        else:
            cache_key = hashlib.sha256(auth_data.encode("utf-8")).hexdigest()

        if cache_key not in self._clients:
            if len(self._clients) >= 100:
                _ = self._clients.pop(next(iter(self._clients)))
            self._clients[cache_key] = MusicClient(auth=auth_data if auth_data else None)
        return self._clients[cache_key]
    @override
    async def health_check(
        self, request, ctx
    ) -> music_pb2.HealthCheckResponse:
        return music_pb2.HealthCheckResponse(ok=True)
    
    @override
    async def login(self, request, ctx) -> music_pb2.LoginResponse:
        auth_json = request.auth_json or ctx.request_headers.get("x-auth-json", "")
        auth_json = parse_auth_metadata(auth_json)

        if not auth_json:
            return music_pb2.LoginResponse(
                authenticated=False,
                error="No auth credentials provided.",
                user_name="",
            )

        try:
            yt = await asyncio.to_thread(get_ytmusic_client, str(auth_json))
            err: str | None = None

            try:
                playlists = await asyncio.to_thread(yt.get_library_playlists, limit=5)
                if playlists and len(playlists) > 0:
                    return music_pb2.LoginResponse(authenticated=True, error="", user_name="")
            except Exception as e:
                err = str(e)

            try:
                songs_data = await asyncio.to_thread(yt.get_liked_songs, limit=5)
                tracks = songs_data.get("tracks")
                if isinstance(tracks, list) and len(tracks) > 0:
                    return music_pb2.LoginResponse(authenticated=True, error="", user_name="")
            except Exception as e:
                if err is None:
                    err = str(e)

            user_name = ""
            try:
                info = await asyncio.to_thread(yt.get_account_info)
                user_name = str(info.get("accountName") or "")
            except Exception as e:
                if err is None:
                    err = str(e)

            if user_name:
                return music_pb2.LoginResponse(authenticated=True, error="", user_name=user_name)

            return music_pb2.LoginResponse(
                authenticated=False,
                error=err or "Failed to Authenticate User",
                user_name="",
            )
        except Exception as e:
            return music_pb2.LoginResponse(
                authenticated=False,
                error=f"Verification failed: {e}",
                user_name="",
            )
    
    @override
    async def get_library(self, request, ctx) -> music_pb2.GetLibraryResponse:
        limit = request.limit if request.limit > 0 else 25
        client = self._get_client_for_request(ctx)
        library_data = await asyncio.to_thread(client.get_library, limit=limit)
        return music_pb2.GetLibraryResponse(
            albums=[to_proto_album(album) for album in library_data.get("albums", [])],
            playlists=[to_proto_playlist(pl) for pl in library_data.get("playlists", [])],
            channels=[to_proto_channel(ch) for ch in library_data.get("channels", [])],
            artists=[to_proto_followed_artist(art) for art in library_data.get("artists", [])],
            podcasts=[to_proto_podcast(pod) for pod in library_data.get("podcasts", [])],
        )
    @override
    async def get_user_saved_tracks(self, request, ctx) -> music_pb2.GetUserSavedTracksResponse:
        limit = request.limit if request.limit > 0 else 100
        client = self._get_client_for_request(ctx)
        songs_data = await asyncio.to_thread(client.get_user_saved_tracks, limit=limit)
        songs_list = [to_proto_song(song) for song in songs_data]
        return music_pb2.GetUserSavedTracksResponse(tracks=songs_list, total=len(songs_list))
    @override
    async def get_user_saved_albums(self, request, ctx) -> music_pb2.GetUserSavedAlbumsResponse:
        limit = request.limit if request.limit > 0 else 25
        client = self._get_client_for_request(ctx)
        albums_data = await asyncio.to_thread(client.get_user_saved_albums, limit=limit)
        albums_list = [to_proto_album(album) for album in albums_data]
        return music_pb2.GetUserSavedAlbumsResponse(albums=albums_list, total=len(albums_list))
    @override
    async def get_user_playlists(self, request, ctx) -> music_pb2.GetUserPlaylistsResponse:
        limit = request.limit if request.limit > 0 else 25
        client = self._get_client_for_request(ctx)
        playlists_data = await asyncio.to_thread(client.get_user_playlists, limit=limit)
        playlists_list = [to_proto_playlist(pl) for pl in playlists_data]
        return music_pb2.GetUserPlaylistsResponse(playlists=playlists_list, total=len(playlists_list))
    @override
    async def get_track(self, request, ctx) -> music_pb2.GetTrackResponse:
        client = self._get_client_for_request(ctx)
        raw_auth = ctx.request_headers.get("x-auth-json")
        auth_data = parse_auth_metadata(raw_auth) if raw_auth else None
        track_details = await asyncio.to_thread(
            client.get_track, video_id=request.video_id, user_cookie=auth_data
        )
        if not track_details:
            return music_pb2.GetTrackResponse()

        song_msg = music_pb2.Song(
            video_id=track_details.get("videoId") or request.video_id,
            title=track_details.get("title") or "",
            url=track_details.get("url") or "",
            album="",
            album_id="",
            duration_seconds=0,
            liked=False,
            is_explicit=False,
        )

        author = track_details.get("author") or ""
        if author:
            song_msg.artists.append(music_pb2.Artist(id="", name=author))

        thumbs_dict: dict[str, list[YTThumbnail]] = track_details.get("thumbnail") or {}
        if thumbs_dict:
            for thumb in thumbs_dict.get("thumbnails", []):
                song_msg.thumbnails.append(to_proto_thumbnail(thumb))

        len_str: str | None = track_details.get("lengthSeconds")
        if len_str:
            try:
                song_msg.duration_seconds = int(len_str)
            except ValueError:
                pass

        return music_pb2.GetTrackResponse(track=song_msg)

    @override
    async def get_album_tracks(self, request, ctx) -> music_pb2.GetAlbumTracksResponse:
        client = self._get_client_for_request(ctx)
        album_data = (
            await asyncio.to_thread(client.get_album_tracks, browse_id=request.browse_id)
        ) or {}

        response = music_pb2.GetAlbumTracksResponse(
            title=album_data.get("title") or "",
            year=album_data.get("year") or "",
            total=coerce_int(album_data.get("trackCount")),
            description=album_data.get("description") or "",
        )
        for artist in album_data.get("artists") or []:
            response.artists.append(to_proto_artist(artist))
        for thumbnail in album_data.get("thumbnails") or []:
            response.thumbnails.append(to_proto_thumbnail(thumbnail))
        for track in album_data.get("tracks") or []:
            response.tracks.append(to_proto_song(track))

        return response

    @override
    async def get_playlist_items(self, request, ctx) -> music_pb2.GetPlaylistItemsResponse:
        limit = request.limit if request.limit > 0 else 100
        client = self._get_client_for_request(ctx)
        playlist_data = (
            await asyncio.to_thread(
                client.get_playlist_items, playlist_id=request.playlist_id, limit=limit
            )
        ) or {}

        author_name = ""
        author_val = playlist_data.get("author")
        if isinstance(author_val, dict):
            author_name = author_val.get("name") or ""
        elif isinstance(author_val, str):
            author_name = author_val

        response = music_pb2.GetPlaylistItemsResponse(
            title=playlist_data.get("title") or "",
            description=playlist_data.get("description") or "",
            author=author_name,
            year=playlist_data.get("year") or "",
            track_count=coerce_int(playlist_data.get("trackCount")),
        )
        for thumbnail in playlist_data.get("thumbnails") or []:
            response.thumbnails.append(to_proto_thumbnail(thumbnail))
        for track in playlist_data.get("tracks") or []:
            response.tracks.append(to_proto_song(track))

        return response

    @override
    async def get_search_results(self, request, ctx) -> music_pb2.GetSearchResultsResponse:
        limit = request.limit if request.limit > 0 else 50
        filter_val: YTSearchFilter | None = None
        if request.filter in ("songs", "videos", "albums", "artists", "playlists", "podcasts", "episodes"):
            filter_val = request.filter

        client = self._get_client_for_request(ctx)
        raw_results: list[YTSearchResult] = await asyncio.to_thread(
            client.get_search_results, query=request.query, filter_type=filter_val, limit=limit
        )
        response = music_pb2.GetSearchResultsResponse()

        for result in raw_results:
            result_type = result.get("resultType")
            if result_type in ("song", "video"):
                dur_val = result.get("duration_seconds")
                dur_int = coerce_int(dur_val)

                song_item = music_pb2.SearchResultSong(
                    video_id=str(result.get("videoId") or ""),
                    title=str(result.get("title") or ""),
                    album="",
                    album_id="",
                    duration_seconds=dur_int,
                    is_explicit=bool(result.get("isExplicit")),
                )
                album_info = result.get("album")
                if isinstance(album_info, dict):
                    song_item.album = str(album_info.get("name") or "")
                    song_item.album_id = str(album_info.get("id") or "")
                elif isinstance(album_info, str):
                    song_item.album = album_info

                for artist in result.get("artists") or []:
                    song_item.artists.append(to_proto_artist(artist))
                for thumbnail in result.get("thumbnails") or []:
                    song_item.thumbnails.append(to_proto_thumbnail(thumbnail))
                response.songs.append(song_item)

            elif result_type == "album":
                album_item = music_pb2.SearchResultAlbum(
                    browse_id=str(result.get("browseId") or ""),
                    title=str(result.get("title") or ""),
                    year=str(result.get("year") or ""),
                    type=str(result.get("type") or ""),
                    is_explicit=bool(result.get("isExplicit")),
                )
                for artist in result.get("artists") or []:
                    album_item.artists.append(to_proto_artist(artist))
                for thumbnail in result.get("thumbnails") or []:
                    album_item.thumbnails.append(to_proto_thumbnail(thumbnail))
                response.albums.append(album_item)

            elif result_type == "artist":
                artist_item = music_pb2.SearchResultArtist(
                    browse_id=str(result.get("browseId") or ""),
                    name=str(result.get("artist") or ""),
                    subscribers=str(result.get("subscribers") or ""),
                )
                for thumbnail in result.get("thumbnails") or []:
                    artist_item.thumbnails.append(to_proto_thumbnail(thumbnail))
                response.artists.append(artist_item)

            elif result_type == "playlist":
                playlist_item = music_pb2.SearchResultPlaylist(
                    browse_id=str(result.get("browseId") or ""),
                    title=str(result.get("title") or ""),
                    author=str(result.get("author") or ""),
                    item_count=str(result.get("itemCount") or ""),
                )
                for thumbnail in result.get("thumbnails") or []:
                    playlist_item.thumbnails.append(to_proto_thumbnail(thumbnail))
                response.playlists.append(playlist_item)

            elif result_type == "podcast":
                author_val = result.get("author")
                author_str = author_val if isinstance(author_val, str) else ""

                podcast_item = music_pb2.SearchResultPodcast(
                    browse_id=str(result.get("browseId") or ""),
                    title=str(result.get("title") or ""),
                    author=author_str,
                )
                for thumbnail in result.get("thumbnails") or []:
                    podcast_item.thumbnails.append(to_proto_thumbnail(thumbnail))
                response.podcasts.append(podcast_item)

            elif result_type == "episode":
                podcast_info = result.get("podcast")
                podcast_name = ""
                podcast_id = ""
                if isinstance(podcast_info, dict):
                    podcast_map = cast(dict[str, object], podcast_info)
                    name_val = podcast_map.get("name")
                    id_val = podcast_map.get("id")
                    if isinstance(name_val, str):
                        podcast_name = name_val
                    if isinstance(id_val, str):
                        podcast_id = id_val
                elif isinstance(podcast_info, str):
                    podcast_name = podcast_info

                episode_item = music_pb2.SearchResultEpisode(
                    video_id=str(result.get("videoId") or ""),
                    title=str(result.get("title") or ""),
                    podcast_name=podcast_name,
                    podcast_id=podcast_id,
                    date=str(result.get("date") or ""),
                )
                for thumbnail in result.get("thumbnails") or []:
                    episode_item.thumbnails.append(to_proto_thumbnail(thumbnail))
                response.episodes.append(episode_item)

        return response
    @override
    async def get_artist_top_tracks(self, request, ctx) -> music_pb2.GetArtistTopTracksResponse:
        client = self._get_client_for_request(ctx)
        artist_data = (
            await asyncio.to_thread(client.get_artist_top_tracks, channel_id=request.channel_id)
        ) or {}
        response = music_pb2.GetArtistTopTracksResponse(
            name=artist_data.get("name") or "",
            subscribers=artist_data.get("subscribers") or "",
        )
        for thumbnail in artist_data.get("thumbnails") or []:
            response.thumbnails.append(to_proto_thumbnail(thumbnail))

        songs_sec = artist_data.get("songs") or {}
        if songs_sec:
            for song in songs_sec.get("results") or []:
                response.tracks.append(to_proto_song(song))

        return response

    @override
    async def get_followed_artists(self, request, ctx) -> music_pb2.GetFollowedArtistsResponse:
        limit = request.limit if request.limit > 0 else 25
        client = self._get_client_for_request(ctx)
        artists_data = (await asyncio.to_thread(client.get_followed_artists, limit=limit)) or []

        response = music_pb2.GetFollowedArtistsResponse(total=len(artists_data))
        for artist in artists_data:
            response.artists.append(to_proto_followed_artist(artist))

        return response
    @override
    async def get_user_profile(self, request, ctx) -> music_pb2.GetUserProfileResponse:
        client = self._get_client_for_request(ctx)
        user_info = await asyncio.to_thread(client.get_user_profile)
        response = music_pb2.GetUserProfileResponse(
            name=user_info.get("accountName") or "",
            channel_id=user_info.get("channelHandle") or "",
        )
        photo_url = user_info.get("accountPhotoUrl") or ""
        if photo_url:
            response.thumbnails.append(music_pb2.Thumbnail(url=photo_url, width=0, height=0))
        return response
    @override
    async def get_user_top_items(self, request, ctx) -> music_pb2.GetUserTopItemsResponse:
        limit = request.limit if request.limit > 0 else 25
        client = self._get_client_for_request(ctx)
        songs_data = await asyncio.to_thread(client.get_user_top_items)
        songs_list = [to_proto_song(song) for song in songs_data[:limit]]
        return music_pb2.GetUserTopItemsResponse(tracks=songs_list, total=len(songs_list))
    @override
    async def check_user_saved_track(self, request, ctx) -> music_pb2.CheckUserSavedTrackResponse:
        client = self._get_client_for_request(ctx)
        is_saved = await asyncio.to_thread(client.check_user_saved_track, video_id=request.video_id)
        return music_pb2.CheckUserSavedTrackResponse(is_saved=is_saved)
    @override
    async def save_remove_track(self, request, ctx) -> music_pb2.SaveRemoveTrackResponse:
        client = self._get_client_for_request(ctx)
        _ = await asyncio.to_thread(
            client.save_remove_track, video_ids=list(request.video_ids), is_remove=request.is_remove
        )
        return music_pb2.SaveRemoveTrackResponse()
    @override
    async def search_songs(self, request, ctx) -> music_pb2.SearchSongsResponse:
        client = self._get_client_for_request(ctx)
        songs_data = await asyncio.to_thread(client.search, query=request.query)
        songs_list = [to_proto_song(song) for song in songs_data]
        return music_pb2.SearchSongsResponse(songs=songs_list)
    @override
    async def like_song(self, request, ctx) -> music_pb2.LikeSongResponse:
        client = self._get_client_for_request(ctx)
        _ = await asyncio.to_thread(client.like_song, request.video_id)
        return music_pb2.LikeSongResponse()
    @override
    async def unlike_song(self, request, ctx) -> music_pb2.UnlikeSongResponse:
        client = self._get_client_for_request(ctx)
        _ = await asyncio.to_thread(client.unlike_song, request.video_id)
        return music_pb2.UnlikeSongResponse()
    @override
    async def get_home_page(self, request, ctx) -> music_pb2.GetHomePageResponse:
        client = self._get_client_for_request(ctx)
        home_sections: list[YTHomeSection] = await asyncio.to_thread(client.get_home)

        response = music_pb2.GetHomePageResponse()

        for section in home_sections:
            section_msg = music_pb2.HomePageSection(title=section.get("title") or "")
            contents: object = section.get("contents")
            if isinstance(contents, list):
                for content in cast(list[object], contents):
                    if not isinstance(content, dict):
                        continue
                    content_map = cast(dict[str, object], content)
                    playlist_id = str(content_map.get("playlistId") or "")
                    video_id = str(content_map.get("videoId") or "")
                    browse_id = str(content_map.get("browseId") or "")
                    content_type = "playlist"
                    if video_id:
                        content_type = "song"
                    elif browse_id and browse_id.startswith("MPRE"):
                        content_type = "album"
                    elif browse_id and (browse_id.startswith("UC") or browse_id.startswith("FKYt")):
                        content_type = "artist"
                    elif playlist_id:
                        content_type = "playlist"
                    elif browse_id:
                        playlist_id = browse_id
                        content_type = "playlist"

                    content_msg = music_pb2.HomePageContent(
                        title=str(content_map.get("title") or ""),
                        playlist_id=playlist_id,
                        video_id=video_id,
                        browse_id=browse_id,
                        content_type=content_type,
                        description=str(content_map.get("description") or ""),
                    )

                    thumbnails = content_map.get("thumbnails")
                    if isinstance(thumbnails, list):
                        for thumbnail in cast(list[object], thumbnails):
                            if isinstance(thumbnail, dict):
                                content_msg.thumbnails.append(
                                    to_proto_thumbnail(cast(dict[str, object], thumbnail))
                                )

                    section_msg.contents.append(content_msg)

            response.sections.append(section_msg)

        return response
    @override
    async def get_song_related(self, request, ctx) -> music_pb2.GetSongRelatedResponse:
        client = self._get_client_for_request(ctx)
        if not request.video_id and not getattr(request, "browse_id", None):
            return music_pb2.GetSongRelatedResponse()

        browse_id = getattr(request, "browse_id", None) or getattr(request, "videoId", None)
        browse_id_str = str(browse_id or "")
        if not browse_id_str.startswith("MPTRt_"):
            watch_data = await asyncio.to_thread(client.get_watch_playlist, request.video_id)
            related_id = watch_data.get("related")
            if isinstance(related_id, str) and related_id:
                browse_id_str = related_id

        if not browse_id_str.startswith("MPTRt_"):
            return music_pb2.GetSongRelatedResponse()

        sections_data = await asyncio.to_thread(client.get_song_related, browse_id_str)
        response = music_pb2.GetSongRelatedResponse()

        for section in sections_data:
            sec_msg = music_pb2.SongRelatedSection(title=str(section.get("title") or ""))
            contents_data = section.get("contents")
            if isinstance(contents_data, str):
                sec_msg.text_content = contents_data
            elif isinstance(contents_data, list):
                for item in cast(list[object], contents_data):
                    if isinstance(item, dict):
                        sec_msg.contents.append(to_proto_song_related_content(cast(dict[str, object], item)))
            response.sections.append(sec_msg)

        return response
    @override
    async def get_lyrics(self, request, ctx) -> music_pb2.GetLyricsResponse:
        client = self._get_client_for_request(ctx)
        watch_data = await asyncio.to_thread(client.get_watch_playlist, request.video_id)
        lyrics_id = watch_data.get("lyrics")
        if not isinstance(lyrics_id, str) or not lyrics_id:
            return music_pb2.GetLyricsResponse()

        raw_lyrics = await asyncio.to_thread(
            client.get_lyrics, browse_id=lyrics_id, timestamps=request.timestamps
        )
        if raw_lyrics is None:
            return music_pb2.GetLyricsResponse()

        def _get_val(obj: object, key: str) -> object:
            if isinstance(obj, dict):
                return cast(dict[str, object], obj).get(key)
            return getattr(obj, key, None)

        source = str(_get_val(raw_lyrics, "source") or "")
        has_timestamps = bool(_get_val(raw_lyrics, "hasTimestamps") or False)
        lyrics_data = _get_val(raw_lyrics, "lyrics")

        response = music_pb2.GetLyricsResponse(source=source, has_timestamps=has_timestamps)

        if has_timestamps and isinstance(lyrics_data, list):
            for line in cast(list[object], lyrics_data):
                if isinstance(line, dict):
                    line_map = cast(dict[str, object], line)
                    line_text = coerce_str(line_map.get("text"))
                    start_time = coerce_int(line_map.get("start_time"))
                    end_time = coerce_int(line_map.get("end_time"))
                    line_id = coerce_int(line_map.get("id"))
                else:
                    line_text = coerce_str(getattr(line, "text", None))
                    start_time = coerce_int(getattr(line, "start_time", 0))
                    end_time = coerce_int(getattr(line, "end_time", 0))
                    line_id = coerce_int(getattr(line, "id", 0))

                response.lines.append(
                    music_pb2.LyricLine(
                        text=line_text,
                        start_time=start_time,
                        end_time=end_time,
                        id=line_id,
                    )
                )
        elif isinstance(lyrics_data, str):
            response.lyrics = lyrics_data

        return response
    
    @override
    async def create_playlist(self, request, ctx) -> music_pb2.CreatePlaylistResponse:
        client = self._get_client_for_request(ctx)
        result = await asyncio.to_thread(
            client.create_playlist,
            title=request.title,
            description=request.description,
            privacy_status=request.privacy_status or "PRIVATE",
            video_ids=list(request.video_ids) if request.video_ids else None,
            source_playlist=request.source_playlist if request.source_playlist else None,
        )

        if isinstance(result, str):
            return music_pb2.CreatePlaylistResponse(playlist_id=result, success=True)

        playlist_id = str(result.get("playlistId") or result.get("id") or "")
        error_msg = str(result.get("error") or "")
        return music_pb2.CreatePlaylistResponse(
            playlist_id=playlist_id,
            success=bool(playlist_id and not error_msg),
            error=error_msg,
        )
    @override
    async def add_playlist_items(self, request, ctx) -> music_pb2.AddPlaylistItemsResponse:
        client = self._get_client_for_request(ctx)
        result = await asyncio.to_thread(
            client.add_playlist_items,
            playlist_id=request.playlist_id,
            video_ids=list(request.video_ids) if request.video_ids else None,
            source_playlist=request.source_playlist if request.source_playlist else None,
            duplicates=request.duplicates,
        )

        if isinstance(result, str):
            success = result == "STATUS_SUCCEEDED"
            final_status = result if result else "STATUS_FAILED"
            return music_pb2.AddPlaylistItemsResponse(
                status=final_status,
                success=success,
                error="" if success else final_status,
            )

        status_str = str(result.get("status") or "")
        error_msg = str(result.get("error") or "")
        success = status_str == "STATUS_SUCCEEDED" and not error_msg
        final_status = status_str if status_str else "STATUS_FAILED"
        if not success and not error_msg:
            error_msg = final_status

        return music_pb2.AddPlaylistItemsResponse(
            status=final_status,
            success=success,
            error=error_msg,
        )
    @override
    async def remove_playlist_items(self, request, ctx) -> music_pb2.RemovePlaylistItemsResponse:
        client = self._get_client_for_request(ctx)
        videos = [{"videoId": v.video_id, "setVideoId": v.set_video_id} for v in request.videos]
        result = await asyncio.to_thread(
            client.remove_playlist_items, playlist_id=request.playlist_id, videos=videos
        )

        if isinstance(result, str):
            success = result == "STATUS_SUCCEEDED"
            final_status = result if result else "STATUS_FAILED"
            return music_pb2.RemovePlaylistItemsResponse(
                status=final_status,
                success=success,
                error="" if success else final_status,
            )

        status_str = str(result.get("status") or "")
        error_msg = str(result.get("error") or "")
        success = status_str == "STATUS_SUCCEEDED" and not error_msg
        final_status = status_str if status_str else "STATUS_FAILED"
        if not success and not error_msg:
            error_msg = final_status

        return music_pb2.RemovePlaylistItemsResponse(
            status=final_status,
            success=success,
            error=error_msg,
        )


def serve() -> None:
    port = int(os.environ.get("PORT", "8080"))
    app = MusicServiceASGIApplication(Service())
    print(f"Server started, listening on {port}")
    uvicorn.run(app, host="0.0.0.0", port=port, log_level="info")


if __name__ == "__main__":
        serve() 