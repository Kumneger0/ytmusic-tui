from concurrent import futures
from types import FrameType
import grpc
import os
import sys
from typing import Callable, override
import signal
from dotenv import load_dotenv
_= load_dotenv()


sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "gen")))

from grpc_server.gen import music_pb2, music_pb2_grpc  # pyright: ignore[reportImplicitRelativeImport]

from grpc_server.src.auth import get_browser_json_path, run_login_flow  # pyright: ignore[reportImplicitRelativeImport]
from grpc_server.src.client.client import MusicClient  # pyright: ignore[reportImplicitRelativeImport]
from grpc_server.src.client.types import (  # pyright: ignore[reportImplicitRelativeImport]
    YTHomeSection,
    YTSearchResult,
    YTSong,
    YTThumbnail,
    YTArtist,
    YTLibraryAlbum,
    YTLibraryPlaylist,
    YTLibraryChannel,
    YTLibraryArtist,
    YTSearchFilter
)

def _to_proto_thumbnail(thumb: YTThumbnail) -> music_pb2.Thumbnail:
    return music_pb2.Thumbnail(
        url=thumb.get("url") or "",
        width=thumb.get("width") or 0,
        height=thumb.get("height") or 0,
    )


def _to_proto_artist(artist: YTArtist) -> music_pb2.Artist:
    return music_pb2.Artist(
        id=artist.get("id") or "",
        name=artist.get("name") or "",
    )


def _to_proto_song(song: YTSong) -> music_pb2.Song:
    """Helper function to map a song dictionary from ytmusicapi to a Protobuf Song message."""
    album_name = ""
    album_id = ""
    album_data = song.get("album")
    if isinstance(album_data, dict):
        album_name = album_data.get("name") or ""
        album_id = album_data.get("id") or ""
    elif isinstance(album_data, str):
        album_name = album_data

    song_msg = music_pb2.Song(
        video_id=song.get("videoId") or "",
        title=song.get("title") or "",
        album=album_name,
        album_id=album_id,
        duration_seconds=song.get("duration_seconds") or 0,
        liked=(song.get("likeStatus") == "LIKE"),
        is_explicit=bool(song.get("isExplicit")),
    )

    for artist in (song.get("artists") or []):
        song_msg.artists.append(_to_proto_artist(artist))

    for thumbnail in (song.get("thumbnails") or []):
        song_msg.thumbnails.append(_to_proto_thumbnail(thumbnail))

    return song_msg


def _to_proto_album(album: YTLibraryAlbum) -> music_pb2.Album:
    album_msg = music_pb2.Album(
        browse_id=album.get("browseId") or "",
        title=album.get("title") or "",
        year=album.get("year") or "",
        is_explicit=bool(album.get("isExplicit")),
        type=album.get("type") or "",
    )
    for artist in (album.get("artists") or []):
        album_msg.artists.append(_to_proto_artist(artist))
    for thumbnail in (album.get("thumbnails") or []):
        album_msg.thumbnails.append(_to_proto_thumbnail(thumbnail))
    return album_msg


def _to_proto_playlist(playlist: YTLibraryPlaylist) -> music_pb2.Playlist:
    count_val = playlist.get("count")
    count_int = 0
    if isinstance(count_val, int):
        count_int = count_val
    elif isinstance(count_val, str):
        try:
            count_int = int(count_val)
        except ValueError:
            pass

    author_name = ""
    author_val = playlist.get("author")
    if isinstance(author_val, list) and len(author_val) > 0:
        author_name = author_val[0].get("name") or ""
    elif isinstance(author_val, str):
        author_name = author_val

    playlist_msg = music_pb2.Playlist(
        playlist_id=playlist.get("playlistId") or "",
        title=playlist.get("title") or "",
        description=playlist.get("description") or "",
        count=count_int,
        author=author_name,
    )
    for thumbnail in (playlist.get("thumbnails") or []):
        playlist_msg.thumbnails.append(_to_proto_thumbnail(thumbnail))
    return playlist_msg


def _to_proto_channel(channel: YTLibraryChannel) -> music_pb2.LibraryChannel:
    channel_msg = music_pb2.LibraryChannel(
        browse_id=channel.get("browseId") or "",
        name=channel.get("artist") or "",
        subscribers=channel.get("subscribers") or "",
    )
    for thumbnail in (channel.get("thumbnails") or []):
        channel_msg.thumbnails.append(_to_proto_thumbnail(thumbnail))
    return channel_msg


def _to_proto_followed_artist(artist: YTLibraryArtist) -> music_pb2.FollowedArtist:
    artist_msg = music_pb2.FollowedArtist(
        channel_id=artist.get("browseId") or "",
        name=artist.get("artist") or "",
        subscribers=artist.get("subscribers") or "",
    )
    for thumbnail in (artist.get("thumbnails") or []):
        artist_msg.thumbnails.append(_to_proto_thumbnail(thumbnail))
    return artist_msg


def _to_proto_podcast(podcast: YTLibraryPlaylist) -> music_pb2.Podcast:
    author_name = ""
    channel = podcast.get("channel")
    if isinstance(channel, dict):
        author_name = channel.get("name") or ""

    if not author_name:
        author_val = podcast.get("author")
        if isinstance(author_val, list) and len(author_val) > 0:
            author_name = author_val[0].get("name") or ""
        elif isinstance(author_val, str):
            author_name = author_val

    podcast_id = (
        podcast.get("podcastId")
        or podcast.get("playlistId")
        or podcast.get("browseId")
        or ""
    )

    podcast_msg = music_pb2.Podcast(
        podcast_id=podcast_id,
        title=podcast.get("title") or "",
        author=author_name,
    )
    for thumbnail in (podcast.get("thumbnails") or []):
        podcast_msg.thumbnails.append(_to_proto_thumbnail(thumbnail))
    return podcast_msg


class MusicService(music_pb2_grpc.MusicServiceServicer): # type: ignore
    """gRPC Servicer implementing the MusicService interface, mapping requests to MusicClient."""

    def __init__(self, auth_file: str) -> None:
            self.client: MusicClient = MusicClient(auth_file)

    @override
    def HealthCheck(self, request:music_pb2.HealthCheckRequest, context:grpc.ServicerContext) -> music_pb2.HealthCheckResponse:
        return music_pb2.HealthCheckResponse(
            ok=True
        )
    @override
    def Login(self, request: music_pb2.LoginRequest, context: grpc.ServicerContext) -> music_pb2.LoginResponse:
        return music_pb2.LoginResponse(authenticated=True)

    @override
    def GetLibrary(self, request: music_pb2.GetLibraryRequest, context: grpc.ServicerContext) -> music_pb2.GetLibraryResponse:
        limit = request.limit if request.limit > 0 else 25
        library_data = self.client.get_library(limit=limit)
        albums_list = [_to_proto_album(album) for album in library_data.get("albums", [])]
        playlists_list = [_to_proto_playlist(playlist) for playlist in library_data.get("playlists", [])]
        channels_list = [_to_proto_channel(channel) for channel in library_data.get("channels", [])]
        artists_list = [_to_proto_followed_artist(artist) for artist in library_data.get("artists", [])]
        podcasts_list = [_to_proto_podcast(podcast) for podcast in library_data.get("podcasts", [])]
        library = music_pb2.GetLibraryResponse(
            albums=albums_list,
            playlists=playlists_list,
            channels=channels_list,
            artists=artists_list,
            podcasts=podcasts_list,
        )
        return library

    @override
    def GetUserSavedTracks(self, request: music_pb2.GetUserSavedTracksRequest, context: grpc.ServicerContext) -> music_pb2.GetUserSavedTracksResponse:
        limit = request.limit if request.limit > 0 else 100
        songs_data = self.client.get_user_saved_tracks(limit=limit)
        songs_list = [_to_proto_song(song) for song in songs_data]
        return music_pb2.GetUserSavedTracksResponse(tracks=songs_list, total=len(songs_list))

    @override
    def GetUserSavedAlbums(self, request: music_pb2.GetUserSavedAlbumsRequest, context: grpc.ServicerContext) -> music_pb2.GetUserSavedAlbumsResponse:
        limit = request.limit if request.limit > 0 else 25
        albums_data = self.client.get_user_saved_albums(limit=limit)
        albums_list = [_to_proto_album(album) for album in albums_data]
        return music_pb2.GetUserSavedAlbumsResponse(albums=albums_list, total=len(albums_list))

    @override
    def GetUserPlaylists(self, request: music_pb2.GetUserPlaylistsRequest, context: grpc.ServicerContext) -> music_pb2.GetUserPlaylistsResponse:
        limit = request.limit if request.limit > 0 else 25
        playlists_data = self.client.get_user_playlists(limit=limit)
        playlists_list = [_to_proto_playlist(playlist) for playlist in playlists_data]
        return music_pb2.GetUserPlaylistsResponse(playlists=playlists_list, total=len(playlists_list))

    @override
    def GetTrack(self, request: music_pb2.GetTrackRequest, context: grpc.ServicerContext) -> music_pb2.GetTrackResponse:
        track_details = self.client.get_track(video_id=request.video_id)
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
                song_msg.thumbnails.append(_to_proto_thumbnail(thumb))

        len_str: str | None = track_details.get("lengthSeconds")
        if len_str:
            try:
                song_msg.duration_seconds = int(len_str)
            except ValueError:
                pass

        return music_pb2.GetTrackResponse(track=song_msg)

    @override
    def GetAlbumTracks(self, request: music_pb2.GetAlbumTracksRequest, context: grpc.ServicerContext) -> music_pb2.GetAlbumTracksResponse:
        album_data = self.client.get_album_tracks(browse_id=request.browse_id) or {}
        
        response = music_pb2.GetAlbumTracksResponse(
            title=album_data.get("title") or "",
            year=album_data.get("year") or "",
            total=album_data.get("trackCount") or 0,
            description=album_data.get("description") or "",
        )
        for artist in (album_data.get("artists") or []):
            response.artists.append(_to_proto_artist(artist))
        for thumbnail in (album_data.get("thumbnails") or []):
            response.thumbnails.append(_to_proto_thumbnail(thumbnail))
        for track in (album_data.get("tracks") or []):
            response.tracks.append(_to_proto_song(track))
        
        return response

    @override
    def GetPlaylistItems(self, request: music_pb2.GetPlaylistItemsRequest, context: grpc.ServicerContext) -> music_pb2.GetPlaylistItemsResponse:
        limit = request.limit if request.limit > 0 else 100
        playlist_data = self.client.get_playlist_items(playlist_id=request.playlist_id, limit=limit) or {}
        
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
            track_count=playlist_data.get("trackCount") or 0,
        )
        for thumbnail in (playlist_data.get("thumbnails") or []):
            response.thumbnails.append(_to_proto_thumbnail(thumbnail))
        for track in (playlist_data.get("tracks") or []):
            response.tracks.append(_to_proto_song(track))
            
        return response

    @override
    def GetSearchResults(self, request: music_pb2.GetSearchResultsRequest, context: grpc.ServicerContext) -> music_pb2.GetSearchResultsResponse:
        limit = request.limit if request.limit > 0 else 50
        filter_val: YTSearchFilter | None = None
        if request.filter in ("songs", "videos", "albums", "artists", "playlists", "podcasts", "episodes"):
            filter_val = request.filter
        
        raw_results: list[YTSearchResult] = self.client.get_search_results(query=request.query, filter_type=filter_val, limit=limit)
        response: music_pb2.GetSearchResultsResponse = music_pb2.GetSearchResultsResponse()
        
        for result in raw_results:
            result_type = result.get("resultType")
            if result_type == "song":
                song_item = music_pb2.SearchResultSong(
                    video_id=result.get("videoId") or "",
                    title=result.get("title") or "",
                    album="",
                    album_id="",
                    duration_seconds=result.get("duration_seconds") or 0,
                    is_explicit=bool(result.get("isExplicit")),
                )
                album_info = result.get("album")
                if isinstance(album_info, dict):
                    song_item.album = album_info.get("name") or ""
                    song_item.album_id = album_info.get("id") or ""
                elif isinstance(album_info, str):
                    song_item.album = album_info

                for artist in (result.get("artists") or []):
                    song_item.artists.append(_to_proto_artist(artist))
                for thumbnail in (result.get("thumbnails") or []):
                    song_item.thumbnails.append(_to_proto_thumbnail(thumbnail))
                response.songs.append(song_item)
                
            elif result_type == "album":
                album_item = music_pb2.SearchResultAlbum(
                    browse_id=result.get("browseId") or "",
                    title=result.get("title") or "",
                    year=result.get("year") or "",
                    type=result.get("type") or "",
                    is_explicit=bool(result.get("isExplicit")),
                )
                for artist in (result.get("artists") or []):
                    album_item.artists.append(_to_proto_artist(artist))
                for thumbnail in (result.get("thumbnails") or []):
                    album_item.thumbnails.append(_to_proto_thumbnail(thumbnail))
                response.albums.append(album_item)
                
            elif result_type == "artist":
                artist_item = music_pb2.SearchResultArtist(
                    browse_id=result.get("browseId") or "",
                    name=result.get("artist") or "",
                    subscribers=result.get("subscribers") or "",
                )
                for thumbnail in (result.get("thumbnails") or []):
                    artist_item.thumbnails.append(_to_proto_thumbnail(thumbnail))
                response.artists.append(artist_item)
                
            elif result_type == "playlist":
                playlist_item = music_pb2.SearchResultPlaylist(
                    browse_id=result.get("browseId") or "",
                    title=result.get("title") or "",
                    author=result.get("author") or "",
                    item_count=result.get("itemCount") or "",
                )
                for thumbnail in (result.get("thumbnails") or []):
                    playlist_item.thumbnails.append(_to_proto_thumbnail(thumbnail))
                response.playlists.append(playlist_item)

            elif result_type == "podcast":
                author_val = result.get("author")
                author_str = ""
                if isinstance(author_val, dict):
                    author_str = author_val.get("name") or ""
                elif isinstance(author_val, str):
                    author_str = author_val

                podcast_item = music_pb2.SearchResultPodcast(
                    browse_id=result.get("browseId") or "",
                    title=result.get("title") or "",
                    author=author_str,
                )
                for thumbnail in (result.get("thumbnails") or []):
                    podcast_item.thumbnails.append(_to_proto_thumbnail(thumbnail))
                response.podcasts.append(podcast_item)

            elif result_type == "episode":
                podcast_info = result.get("podcast")
                podcast_name = ""
                podcast_id = ""
                if isinstance(podcast_info, dict):
                    podcast_name = podcast_info.get("name") or ""
                    podcast_id = podcast_info.get("id") or ""
                elif isinstance(podcast_info, str):
                    podcast_name = podcast_info

                episode_item = music_pb2.SearchResultEpisode(
                    video_id=result.get("videoId") or "",
                    title=result.get("title") or "",
                    podcast_name=podcast_name,
                    podcast_id=podcast_id,
                    date=result.get("date") or "",
                )
                for thumbnail in (result.get("thumbnails") or []):
                    episode_item.thumbnails.append(_to_proto_thumbnail(thumbnail))
                response.episodes.append(episode_item)

            elif result_type == "video":
                song_item = music_pb2.SearchResultSong(
                    video_id=result.get("videoId") or "",
                    title=result.get("title") or "",
                    album="",
                    album_id="",
                    duration_seconds=result.get("duration_seconds") or 0,
                    is_explicit=bool(result.get("isExplicit")),
                )
                album_info = result.get("album")
                if isinstance(album_info, dict):
                    song_item.album = album_info.get("name") or ""
                    song_item.album_id = album_info.get("id") or ""
                elif isinstance(album_info, str):
                    song_item.album = album_info

                for artist in (result.get("artists") or []):
                    song_item.artists.append(_to_proto_artist(artist))
                for thumbnail in (result.get("thumbnails") or []):
                    song_item.thumbnails.append(_to_proto_thumbnail(thumbnail))
                response.songs.append(song_item)

        return response

    @override
    def GetArtistTopTracks(self, request: music_pb2.GetArtistTopTracksRequest, context: grpc.ServicerContext) -> music_pb2.GetArtistTopTracksResponse:
        artist_data = self.client.get_artist_top_tracks(channel_id=request.channel_id) or {}
        
        response = music_pb2.GetArtistTopTracksResponse(
            name=artist_data.get("name") or "",
            subscribers=artist_data.get("subscribers") or "",
        )
        for thumbnail in (artist_data.get("thumbnails") or []):
            response.thumbnails.append(_to_proto_thumbnail(thumbnail))
            
        songs_sec = artist_data.get("songs") or {}
        if songs_sec:
            for song in (songs_sec.get("results") or []):
                response.tracks.append(_to_proto_song(song))
                
        return response

    @override
    def GetFollowedArtists(self, request: music_pb2.GetFollowedArtistsRequest, context: grpc.ServicerContext) -> music_pb2.GetFollowedArtistsResponse:
        limit = request.limit if request.limit > 0 else 25
        artists_data = self.client.get_followed_artists(limit=limit) or []
        
        response = music_pb2.GetFollowedArtistsResponse(total=len(artists_data))
        for artist in artists_data:
            artist_msg = music_pb2.FollowedArtist(
                channel_id=artist.get("browseId") or "",
                name=artist.get("artist") or "",
                subscribers=artist.get("subscribers") or "",
            )
            for thumbnail in (artist.get("thumbnails") or []):
                artist_msg.thumbnails.append(_to_proto_thumbnail(thumbnail))
            response.artists.append(artist_msg)
            
        return response

    @override
    def GetUserProfile(self, request: music_pb2.GetUserProfileRequest, context: grpc.ServicerContext) -> music_pb2.GetUserProfileResponse:
        user_info = self.client.get_user_profile()
        response = music_pb2.GetUserProfileResponse(
            name=user_info.get("accountName") or "",
            channel_id=user_info.get("channelHandle") or "",
        )
        photo_url = user_info.get("accountPhotoUrl") or ""
        if photo_url:
            response.thumbnails.append(music_pb2.Thumbnail(url=photo_url, width=0, height=0))
        return response

    @override
    def GetUserTopItems(self, request: music_pb2.GetUserTopItemsRequest, context: grpc.ServicerContext) -> music_pb2.GetUserTopItemsResponse:
        limit = request.limit if request.limit > 0 else 25
        songs_data = self.client.get_user_top_items()
        songs_list = [_to_proto_song(song) for song in songs_data[:limit]]
        return music_pb2.GetUserTopItemsResponse(tracks=songs_list, total=len(songs_list))

    @override
    def CheckUserSavedTrack(self, request: music_pb2.CheckUserSavedTrackRequest, context: grpc.ServicerContext) -> music_pb2.CheckUserSavedTrackResponse:
        is_saved = self.client.check_user_saved_track(video_id=request.video_id)
        return music_pb2.CheckUserSavedTrackResponse(is_saved=is_saved)

    @override
    def SaveRemoveTrack(self, request: music_pb2.SaveRemoveTrackRequest, context: grpc.ServicerContext) -> music_pb2.SaveRemoveTrackResponse:
        self.client.save_remove_track(video_ids=list(request.video_ids), is_remove=request.is_remove)
        return music_pb2.SaveRemoveTrackResponse()

    @override
    def SearchSongs(self, request: music_pb2.SearchSongsRequest, context: grpc.ServicerContext) -> music_pb2.SearchSongsResponse:
        songs_data = self.client.search(query=request.query)
        songs_list = [_to_proto_song(song) for song in songs_data]
        return music_pb2.SearchSongsResponse(songs=songs_list)

    @override
    def LikeSong(self, request: music_pb2.LikeSongRequest, context: grpc.ServicerContext) -> music_pb2.LikeSongResponse:
        _ = self.client.like_song(request.video_id)
        return music_pb2.LikeSongResponse()

    @override
    def UnlikeSong(self, request: music_pb2.UnlikeSongRequest, context: grpc.ServicerContext) -> music_pb2.UnlikeSongResponse:
        _ = self.client.unlike_song(request.video_id)
        return music_pb2.UnlikeSongResponse()
    @override
    def GetVideoStreamURLAndDuration(self, request: music_pb2.GetVideoStreamURLAndDurationRequest, context:grpc.ServicerContext) -> music_pb2.GetVideoStreamURLAndDurationResponse:
        stream_url_and_duration= self.client.get_stream_url_and_duration(request.videoId)
        duration = stream_url_and_duration.get('duration')
        return  music_pb2.GetVideoStreamURLAndDurationResponse(
            url=stream_url_and_duration.get('url'),
            duration= "" if duration is None else str(duration)
        )
    @override
    def GetHomePage(self, request: music_pb2.GetHomePageRequest, context: grpc.ServicerContext) -> music_pb2.GetHomePageResponse:
        home_sections: list[YTHomeSection] = self.client.get_home()
        
        response = music_pb2.GetHomePageResponse()
        
        for section in home_sections:
            section_msg = music_pb2.HomePageSection(
                title=section.get("title") or ""
            )
            
            for content in section.get("contents", []):
                playlist_id = content.get("playlistId") or ""
                video_id = content.get("videoId") or ""
                browse_id = content.get("browseId") or ""
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
                    title=content.get("title") or "",
                    playlist_id=playlist_id,
                    video_id=video_id,
                    browse_id=browse_id,
                    content_type=content_type,
                    description=content.get("description") or ""
                )
                
                for thumbnail in content.get("thumbnails", []):
                    content_msg.thumbnails.append(_to_proto_thumbnail(thumbnail))
                
                section_msg.contents.append(content_msg)
            
            response.sections.append(section_msg)
        
        return response



def make_shutdown_handler(server: grpc.Server) -> Callable[..., None]:
    def shutdown(signum: int, frame: FrameType | None) -> None:
        print(f"Received {signal.Signals(signum).name}")
        _ = server.stop(grace=5)
    return shutdown



def serve() -> None:
    port = "50051"
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    auth_file = str(get_browser_json_path())
    servicer = MusicService(auth_file)
    music_pb2_grpc.add_MusicServiceServicer_to_server(servicer, server) # type: ignore  # pyright: ignore[reportUnknownMemberType]

    _ = server.add_insecure_port("[::]:" + port)
    server.start()
    print("Server started, listening on " + port)
    _ = signal.signal(signal.SIGTERM, handler=make_shutdown_handler(server=server))
    _ = signal.signal(signal.SIGINT, make_shutdown_handler(server=server))     
    _ = server.wait_for_termination()




if __name__ == "__main__":  
    if len(sys.argv) > 1 and "--login" in sys.argv:
        file_arg = None
        args = sys.argv[1:]
        login_idx = args.index("--login")
        if login_idx + 1 < len(args) and not args[login_idx + 1].startswith("-"):
            file_arg = args[login_idx + 1]
        run_login_flow(file_path=file_arg)
    else:
        serve()