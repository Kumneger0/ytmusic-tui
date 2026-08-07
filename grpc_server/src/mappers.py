import base64
from typing import cast

from grpc_server.gen import music_pb2
from grpc_server.src.client.types import (
    YTLibraryAlbum,
    YTLibraryArtist,
    YTLibraryChannel,
    YTLibraryPlaylist,
    YTSong,
)


def parse_auth_metadata(auth_data: str) -> str:
    auth_data = auth_data.strip()
    if not auth_data:
        return ""
    if auth_data.startswith("{"):
        return auth_data
    try:
        decoded = base64.b64decode(auth_data).decode("utf-8")
        if decoded.startswith("{"):
            return decoded
    except Exception:
        pass
    return auth_data


def coerce_str(value: object) -> str:
    return str(value) if value is not None else ""


def coerce_int(value: object, default: int = 0) -> int:
    if value is None:
        return default
    if isinstance(value, int):
        return value
    if isinstance(value, float):
        return int(value)
    try:
        return int(str(value))
    except (ValueError, TypeError):
        return default


def to_proto_thumbnail(thumb: object | str) -> music_pb2.Thumbnail:
    if isinstance(thumb, str):
        return music_pb2.Thumbnail(url=thumb, width=0, height=0)
    if isinstance(thumb, dict):
        thumb_map = cast(dict[str, object], thumb)
        return music_pb2.Thumbnail(
            url=coerce_str(thumb_map.get("url")),
            width=coerce_int(thumb_map.get("width")),
            height=coerce_int(thumb_map.get("height")),
        )
    return music_pb2.Thumbnail(url="", width=0, height=0)


def to_proto_artist(artist: object | str) -> music_pb2.Artist:
    if isinstance(artist, str):
        return music_pb2.Artist(id="", name=artist)
    if isinstance(artist, dict):
        artist_map = cast(dict[str, object], artist)
        return music_pb2.Artist(
            id=coerce_str(artist_map.get("id")),
            name=coerce_str(artist_map.get("name")),
        )
    return music_pb2.Artist(id="", name=coerce_str(artist))


def to_proto_song(song: YTSong) -> music_pb2.Song:
    album_name = ""
    album_id = ""
    album_data = song.get("album")
    if isinstance(album_data, dict):
        name_val = album_data.get("name")
        id_val = album_data.get("id")
        if isinstance(name_val, str):
            album_name = name_val
        if isinstance(id_val, str):
            album_id = id_val
    elif isinstance(album_data, str):
        album_name = album_data

    song_msg = music_pb2.Song(
        video_id=song.get("videoId") or "",
        title=song.get("title") or "",
        album=album_name,
        album_id=album_id,
        duration_seconds=coerce_int(song.get("duration_seconds")),
        liked=(song.get("likeStatus") == "LIKE"),
        is_explicit=bool(song.get("isExplicit")),
        set_video_id=song.get("setVideoId") or "",
    )

    for artist in (song.get("artists") or []):
        song_msg.artists.append(to_proto_artist(artist))

    for thumbnail in (song.get("thumbnails") or []):
        song_msg.thumbnails.append(to_proto_thumbnail(thumbnail))

    return song_msg


def to_proto_album(album: YTLibraryAlbum) -> music_pb2.Album:
    album_msg = music_pb2.Album(
        browse_id=album.get("browseId") or "",
        title=album.get("title") or "",
        year=album.get("year") or "",
        is_explicit=bool(album.get("isExplicit")),
        type=album.get("type") or "",
    )
    for artist in (album.get("artists") or []):
        album_msg.artists.append(to_proto_artist(artist))
    for thumbnail in (album.get("thumbnails") or []):
        album_msg.thumbnails.append(to_proto_thumbnail(thumbnail))
    return album_msg


def to_proto_playlist(playlist: YTLibraryPlaylist) -> music_pb2.Playlist:
    count_int = coerce_int(playlist.get("count"))

    author_name = ""
    author_val = playlist.get("author")
    if isinstance(author_val, list) and len(author_val) > 0:
        first_author = author_val[0]
        if isinstance(first_author, dict):
            name_val = str(first_author.get("name") or "")
            if name_val:
                author_name = name_val
        elif isinstance(first_author, str):
            author_name = first_author
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
        playlist_msg.thumbnails.append(to_proto_thumbnail(thumbnail))
    return playlist_msg


def to_proto_channel(channel: YTLibraryChannel) -> music_pb2.LibraryChannel:
    channel_msg = music_pb2.LibraryChannel(
        browse_id=channel.get("browseId") or "",
        name=channel.get("artist") or "",
        subscribers=channel.get("subscribers") or "",
    )
    for thumbnail in (channel.get("thumbnails") or []):
        channel_msg.thumbnails.append(to_proto_thumbnail(thumbnail))
    return channel_msg


def to_proto_followed_artist(artist: YTLibraryArtist) -> music_pb2.FollowedArtist:
    artist_msg = music_pb2.FollowedArtist(
        channel_id=artist.get("browseId") or "",
        name=artist.get("artist") or "",
        subscribers=artist.get("subscribers") or "",
    )
    for thumbnail in (artist.get("thumbnails") or []):
        artist_msg.thumbnails.append(to_proto_thumbnail(thumbnail))
    return artist_msg


def to_proto_podcast(podcast: YTLibraryPlaylist) -> music_pb2.Podcast:
    author_name = ""
    channel: object = podcast.get("channel")
    if isinstance(channel, dict):
        channel_map = cast(dict[str, object], channel)
        val = str(channel_map.get("name") or "")
        if val:
            author_name = val

    if not author_name:
        author_val = podcast.get("author")
        if isinstance(author_val, list) and len(author_val) > 0:
            first_author = author_val[0]
            if isinstance(first_author, dict):
                val = str(first_author.get("name") or "")
                if val:
                    author_name = val
            elif isinstance(first_author, str):
                author_name = first_author
        elif isinstance(author_val, str):
            author_name = author_val

    podcast_id_val = (
        podcast.get("podcastId")
        or podcast.get("playlistId")
        or podcast.get("browseId")
        or ""
    )
    podcast_id = str(podcast_id_val)

    podcast_msg = music_pb2.Podcast(
        podcast_id=podcast_id,
        title=podcast.get("title") or "",
        author=author_name,
    )
    for thumbnail in (podcast.get("thumbnails") or []):
        podcast_msg.thumbnails.append(to_proto_thumbnail(thumbnail))
    return podcast_msg


def to_proto_song_related_content(content: dict[str, object]) -> music_pb2.SongRelatedContent:
    album_name = ""
    album_id = ""
    album_data = content.get("album")
    if isinstance(album_data, dict):
        album_map = cast(dict[str, object], album_data)
        name_val = album_map.get("name")
        if isinstance(name_val, str):
            album_name = name_val
        id_val = album_map.get("id")
        if isinstance(id_val, str):
            album_id = id_val
    elif isinstance(album_data, str):
        album_name = album_data

    content_type = ""
    video_id = content.get("videoId")
    browse_id = content.get("browseId")
    playlist_id = content.get("playlistId")
    subscribers = content.get("subscribers")
    if video_id:
        content_type = "song"
    elif subscribers or (browse_id and str(browse_id).startswith("UC")):
        content_type = "artist"
    elif playlist_id:
        content_type = "playlist"
    elif browse_id:
        browse_id_str = str(browse_id)
        if browse_id_str.startswith("MPRE") or browse_id_str.startswith("FEmusic_album"):
            content_type = "album"
        else:
            content_type = "artist"

    content_msg = music_pb2.SongRelatedContent(
        title=coerce_str(content.get("title")),
        video_id=coerce_str(video_id),
        playlist_id=coerce_str(playlist_id),
        browse_id=coerce_str(browse_id),
        is_explicit=bool(content.get("isExplicit") or content.get("is_explicit")),
        album=album_name,
        album_id=album_id,
        description=coerce_str(content.get("description")),
        subscribers=coerce_str(subscribers),
        year=coerce_str(content.get("year")) if content.get("year") is not None else "",
        content_type=content_type,
    )

    raw_artists = content.get("artists")
    if isinstance(raw_artists, list):
        for artist in cast(list[object], raw_artists):
            content_msg.artists.append(to_proto_artist(artist))

    raw_thumbnails = content.get("thumbnails")
    if isinstance(raw_thumbnails, list):
        for thumbnail in cast(list[object], raw_thumbnails):
            content_msg.thumbnails.append(to_proto_thumbnail(thumbnail))

    return content_msg
