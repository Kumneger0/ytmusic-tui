package headless

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"
	musicpb "github.com/kumneger0/ytmusic-tui/gen"
	"github.com/kumneger0/ytmusic-tui/internal/types"
	"github.com/kumneger0/ytmusic-tui/internal/ui"
)

type UserLibrary struct {
	Playlist           []*musicpb.Playlist       `json:"playlist"`
	UserFollowedArtist []*musicpb.FollowedArtist `json:"artist"`
	Album              []*musicpb.Album          `json:"album"`
}

type TracksType string

const (
	PlaylistType   TracksType = "playlist"
	FollowedArtist TracksType = "followed_artist"
	LikedSongs     TracksType = "saved_tracks"
	AlbumTracks    TracksType = "album_tracks"
)

type TracksResponse struct {
	Tracks []*types.PlaylistTrackObject `json:"tracks"`
}

type SSEMessage struct {
	Player *struct {
		IsPlaying     bool    `json:"isPlaying"`
		CurrentIndex  int     `json:"currentIndex"`
		SecondsPlayed float64 `json:"secondsPlayed"`
	} `json:"player,omitempty"`
}

type PlayRequestBodyType struct {
	TrackID string `json:"trackID"`
	//this is a flag that whether the user skips the track or not
	// b/c during cache mode if the skip we need to remove the track from the cache to prevent saving it b/c it may not be fully downloaded
	IsSkip bool   `json:"isSkip"`
	Queue  *Queue `json:"queue"`
}

type AddTrackToQueue struct {
	Track types.PlaylistTrackObject `json:"track"`
	Index int                       `json:"index"`
}

type RemoveTrackFromQueue struct {
	Track types.PlaylistTrackObject `json:"track"`
}

type Queue struct {
	Tracks       []*types.PlaylistTrackObject `json:"tracks"`
	CurrentIndex int                          `json:"currentIndex"`
}

func (h *Queue) AddTrack(track *types.PlaylistTrackObject, index int) {
	if index < 0 || index > len(h.Tracks) {
		index = len(h.Tracks)
	}
	h.Tracks = append(h.Tracks[:index], append([]*types.PlaylistTrackObject{track}, h.Tracks[index:]...)...)
}

func (h *Queue) RemoveTrack(index int) {
	if index < 0 || index >= len(h.Tracks) {
		return
	}
	h.Tracks = append(h.Tracks[:index], h.Tracks[index+1:]...)
}

func NewMusicQueue() *Queue {
	return &Queue{
		Tracks:       []*types.PlaylistTrackObject{},
		CurrentIndex: 0,
	}
}

func StartServer(m *ui.SafeModel, dbusMessageChan *chan types.DBusMessage) {
	musicQueue := NewMusicQueue()
	var mqMu sync.Mutex

	go func() {
		if dbusMessageChan == nil {
			return
		}
		for msg := range *dbusMessageChan {
			mqMu.Lock()
			switch msg.MessageType {
			case types.NextTrack:
				if len(musicQueue.Tracks) == 0 {
					continue
				}
				nextTrackIndex := musicQueue.CurrentIndex + 1
				if nextTrackIndex >= len(musicQueue.Tracks) {
					nextTrackIndex = 0
				}
				musicQueue.CurrentIndex = nextTrackIndex
				nextTrack := musicQueue.Tracks[musicQueue.CurrentIndex]
				if nextTrack != nil {
					//the code this in this function is only executed when user clicks on
					// control button on his/her desktop environment
					//which means it is skip
					model, _ := m.PlaySelectedMusic(*nextTrack)
					m.Model = &model
				}
			case types.PlayPause:
				model, _ := m.HandleMusicPausePlay()
				m.Model = &model
			case types.PreviousTrack:
				if len(musicQueue.Tracks) == 0 {
					continue
				}
				prevTrackIndex := musicQueue.CurrentIndex - 1
				if prevTrackIndex < 0 {
					prevTrackIndex = len(musicQueue.Tracks) - 1
				}
				musicQueue.CurrentIndex = prevTrackIndex
				prevTrack := musicQueue.Tracks[musicQueue.CurrentIndex]
				if prevTrack != nil {
					model, _ := m.PlaySelectedMusic(*prevTrack)
					m.Model = &model
				}
			}
			mqMu.Unlock()
		}
	}()

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8282",
		Handler: mux,
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "Application/json")
		fmt.Fprintln(w, `{"message":"server is up"}`)
	})

	mux.HandleFunc("/library", func(w http.ResponseWriter, r *http.Request) {
		m.Mu.RLock()
		defer m.Mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		userPlaylist, err := m.YtMusicClient.GetUserPlaylists(ctx, connect.NewRequest(&musicpb.GetUserPlaylistsRequest{}))
		if err != nil {
			slog.Error("GetUserPlaylists failed: " + err.Error())
			http.Error(w, `{"error":"failed to fetch user playlists"}`, http.StatusInternalServerError)
			return
		}
		if userPlaylist == nil {
			slog.Error("GetUserPlaylists returned nil playlist")
			http.Error(w, `{"error":"no playlist data returned"}`, http.StatusInternalServerError)
			return
		}

		followedArtists, err := m.YtMusicClient.GetFollowedArtists(ctx, connect.NewRequest(&musicpb.GetFollowedArtistsRequest{}))
		if err != nil {
			slog.Error("GetFollowedArtists failed: " + err.Error())
			http.Error(w, `{"error":"failed to fetch followed artists"}`, http.StatusInternalServerError)
			return
		}
		if followedArtists == nil {
			slog.Error("GetFollowedArtists returned nil")
			http.Error(w, `{"error":"no artist data returned"}`, http.StatusInternalServerError)
			return
		}

		albums, err := m.YtMusicClient.GetUserSavedAlbums(ctx, connect.NewRequest(&musicpb.GetUserSavedAlbumsRequest{}))
		if err != nil {
			slog.Error("GetUserSavedAlbums failed: " + err.Error())
			http.Error(w, `{"error":"failed to fetch albums"}`, http.StatusInternalServerError)
			return
		}
		if albums == nil {
			slog.Error("GetUserSavedAlbums returned nil")
			http.Error(w, `{"error":"no album data returned"}`, http.StatusInternalServerError)
			return
		}

		userLibrary := &UserLibrary{
			Playlist:           userPlaylist.Msg.Playlists,
			UserFollowedArtist: followedArtists.Msg.Artists,
			Album:              albums.Msg.Albums,
		}

		data, err := json.Marshal(userLibrary)
		if err != nil {
			slog.Error("json.Marshal failed: " + err.Error())
			http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(data)
		if err != nil {
			slog.Error(err.Error())
		}
	})

	mux.HandleFunc("/tracks", func(w http.ResponseWriter, r *http.Request) {
		m.Mu.RLock()
		defer m.Mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")

		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, `{"error":"missing required query param: id"}`, http.StatusBadRequest)
			return
		}

		queryType := r.URL.Query().Get("type")
		if queryType == "" {
			http.Error(w, `{"error":"missing required query param: type"}`, http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if TracksType(queryType) == FollowedArtist {
			artistSongs, err := m.YtMusicClient.GetArtistTopTracks(ctx, connect.NewRequest(&musicpb.GetArtistTopTracksRequest{ChannelId: id}))
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, `{"error":"failed to fetch artist tracks"}`, http.StatusInternalServerError)
				return
			}

			var tracks []*types.PlaylistTrackObject
			for _, track := range artistSongs.Msg.Tracks {
				tracks = append(tracks, &types.PlaylistTrackObject{
					Track: track,
				})
			}

			resp := &TracksResponse{Tracks: tracks}
			data, err := json.Marshal(resp)
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, err = w.Write(data)
			if err != nil {
				slog.Error(err.Error())
			}
			return
		}

		if TracksType(queryType) == PlaylistType {
			playlistItems, err := m.YtMusicClient.GetPlaylistItems(ctx, connect.NewRequest(&musicpb.GetPlaylistItemsRequest{PlaylistId: id}))
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, `{"error":"failed to fetch playlist items"}`, http.StatusInternalServerError)
				return
			}

			var tracks []*types.PlaylistTrackObject
			for _, track := range playlistItems.Msg.Tracks {
				tracks = append(tracks, &types.PlaylistTrackObject{
					Track: track,
				})
			}

			resp := &TracksResponse{Tracks: tracks}
			data, err := json.Marshal(resp)
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, err = w.Write(data)
			if err != nil {
				slog.Error(err.Error())
			}
			return
		}

		if TracksType(queryType) == LikedSongs {
			savedTracks, err := m.YtMusicClient.GetUserSavedTracks(ctx, connect.NewRequest(&musicpb.GetUserSavedTracksRequest{}))
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, `{"error":"failed to fetch saved tracks"}`, http.StatusInternalServerError)
				return
			}

			var tracks []*types.PlaylistTrackObject
			for _, item := range savedTracks.Msg.Tracks {
				tracks = append(tracks, &types.PlaylistTrackObject{
					Track: item,
				})
			}

			resp := &TracksResponse{Tracks: tracks}
			data, err := json.Marshal(resp)
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, err = w.Write(data)
			if err != nil {
				slog.Error(err.Error())
			}
			return
		}

		if TracksType(queryType) == AlbumTracks {
			albumTracks, err := m.YtMusicClient.GetAlbumTracks(ctx, connect.NewRequest(&musicpb.GetAlbumTracksRequest{BrowseId: id}))
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, `{"error":"failed to fetch album tracks"}`, http.StatusInternalServerError)
				return
			}

			var trackObject []*types.PlaylistTrackObject
			for _, item := range albumTracks.Msg.Tracks {
				trackObject = append(trackObject, &types.PlaylistTrackObject{
					Track: item,
				})
			}

			resp := &TracksResponse{Tracks: trackObject}
			data, err := json.Marshal(resp)
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			_, err = w.Write(data)
			if err != nil {
				slog.Error(err.Error())
			}
			return
		}

		http.Error(w, `{"error":"unknown type"}`, http.StatusBadRequest)
	})

	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		m.Mu.RLock()
		defer m.Mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")

		query := r.URL.Query().Get("q")

		if query == "" {
			slog.Error("query is empty")
			http.Error(w, `{"error":"please provide a search query"}`, http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		searchResults, err := m.YtMusicClient.GetSearchResults(ctx, connect.NewRequest(&musicpb.GetSearchResultsRequest{Query: query}))
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, `{"error":"failed to search"}`, http.StatusInternalServerError)
			return
		}

		data, err := json.Marshal(searchResults.Msg)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(data)
		if err != nil {
			slog.Error(err.Error())
		}
	})

	mux.HandleFunc("/player", func(w http.ResponseWriter, r *http.Request) {
		m.Mu.Lock()
		defer m.Mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		var action string
		if m.PlayerProcess != nil && m.PlayerProcess.OtoPlayer.IsPlaying() {
			action = "paused"
		} else {
			action = "play"
		}
		m.HandleMusicPausePlay()
		resp := map[string]any{
			"status": "ok",
			"action": action,
		}
		data, _ := json.Marshal(resp)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(data)
		if err != nil {
			slog.Error(err.Error())
		}
	})

	mux.HandleFunc("POST /player/play", func(w http.ResponseWriter, r *http.Request) {
		m.Mu.Lock()
		defer m.Mu.Unlock()
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		if r.Body == nil {
			http.Error(w, `{"error":"request body required"}`, http.StatusBadRequest)
			return
		}

		defer r.Body.Close()

		var reqBody PlayRequestBodyType
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			slog.Error("failed to decode body: " + err.Error())
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}

		if reqBody.TrackID == "" {
			http.Error(w, `{"error":"trackID is required"}`, http.StatusBadRequest)
			return
		}

		model, _ := m.HandleMusicPausePlay()
		m.Model = &model

		if reqBody.Queue != nil {
			musicQueue = reqBody.Queue
		}

		var trackObject *types.PlaylistTrackObject

		if reqBody.Queue != nil {
			if reqBody.Queue.CurrentIndex >= 0 && reqBody.Queue.CurrentIndex < len(reqBody.Queue.Tracks) {
				trackObject = reqBody.Queue.Tracks[reqBody.Queue.CurrentIndex]
			}
		}

		if trackObject == nil {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			track, err := m.YtMusicClient.GetTrack(ctx, connect.NewRequest(&musicpb.GetTrackRequest{VideoId: reqBody.TrackID}))
			if err != nil {
				slog.Error(err.Error())
				http.Error(w, `{"error":"failed to get track"}`, http.StatusInternalServerError)
				return
			}

			if track == nil {
				slog.Error("track is nil")
				http.Error(w, `{"error":"failed to get track"}`, http.StatusInternalServerError)
				return
			}

			trackObject = &types.PlaylistTrackObject{
				Track: track.Msg.Track,
			}
		}

		model, _ = m.PlaySelectedMusic(*trackObject)
		m.Model = &model
		resp := map[string]any{
			"status":  "ok",
			"message": "track is now playing",
			"track": map[string]any{
				"id":      reqBody.TrackID,
				"name":    trackObject.Track.Title,
				"album":   trackObject.Track.Album,
				"artists": trackObject.Track.Artists,
			},
		}

		data, err := json.Marshal(resp)
		if err != nil {
			slog.Error("failed to encode response: " + err.Error())
			http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write(data)
		if err != nil {
			slog.Error(err.Error())
		}
	})

	mux.HandleFunc("GET /player/queue", func(w http.ResponseWriter, r *http.Request) {
		mqMu.Lock()
		defer mqMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(musicQueue)
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, `{"message":"failed to encode response", "status":"error"}`, http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(data)
		if err != nil {
			slog.Error(err.Error())
		}
	})

	mux.HandleFunc("POST /player/queue/add", func(w http.ResponseWriter, r *http.Request) {
		mqMu.Lock()
		defer mqMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		var reqBody AddTrackToQueue
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil {
			slog.Error("failed to decode request body: " + err.Error())
			fmt.Println("failed to add to queue", reqBody)
			http.Error(w, `{"message":"failed to decode request body", "status":"error"}`, http.StatusBadRequest)
			return
		}

		musicQueue.AddTrack(&reqBody.Track, reqBody.Index)

		w.WriteHeader(http.StatusOK)
		data, err := json.Marshal(map[string]any{
			"status":  "success",
			"message": "track added to queue",
		})
		if err != nil {
			slog.Error(err.Error())
			http.Error(w, `{"message":"failed to encode response", "status":"error"}`, http.StatusBadRequest)
			return
		}
		_, err = w.Write(data)
		if err != nil {
			slog.Error(err.Error())
		}
	})

	mux.HandleFunc("DELETE /player/queue/remove", func(w http.ResponseWriter, r *http.Request) {
		mqMu.Lock()
		defer mqMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		var reqBody RemoveTrackFromQueue
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil {
			slog.Error("failed to decode request body: " + err.Error())
			http.Error(w, `{"message":"failed to decode request body", "status":"error"}`, http.StatusBadRequest)
			return
		}
		var index int = -1
		for i, track := range musicQueue.Tracks {
			if track.Track != nil && reqBody.Track.Track != nil && track.Track.VideoId == reqBody.Track.Track.VideoId {
				index = i
				break
			}
		}
		if index == -1 {
			http.Error(w, `{"message":"track not found", "status":"error"}`, http.StatusNotFound)
			return
		}

		musicQueue.RemoveTrack(index)
		// Adjust CurrentIndex if necessary
		if index < musicQueue.CurrentIndex {
			musicQueue.CurrentIndex--
		} else if index == musicQueue.CurrentIndex && musicQueue.CurrentIndex >= len(musicQueue.Tracks) {
			// If we removed the current track and it was the last one, adjust to the new last track
			if len(musicQueue.Tracks) > 0 {
				musicQueue.CurrentIndex = len(musicQueue.Tracks) - 1
			} else {
				musicQueue.CurrentIndex = 0
			}
		}

		w.WriteHeader(http.StatusOK)
		data, err := json.Marshal(map[string]any{
			"status":  "success",
			"message": "track removed from queue",
		})
		if err != nil {
			slog.Error(err.Error())
			return
		}

		_, err = w.Write(data)
		if err != nil {
			slog.Error(err.Error())
		}
	})

	mux.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ctx := r.Context()

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
		flusher.Flush()

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.Mu.RLock()
				if m.PlayerProcess != nil && m.PlayerProcess.ByteCounterReader != nil {
					currentIndex := musicQueue.CurrentIndex
					isPlaying := m.PlayerProcess != nil && m.PlayerProcess.OtoPlayer.IsPlaying()
					seconds := m.PlayerProcess.ByteCounterReader.CurrentSeconds()

					message := SSEMessage{
						Player: &struct {
							IsPlaying     bool    `json:"isPlaying"`
							CurrentIndex  int     `json:"currentIndex"`
							SecondsPlayed float64 `json:"secondsPlayed"`
						}{
							IsPlaying:     isPlaying,
							CurrentIndex:  currentIndex,
							SecondsPlayed: seconds,
						},
					}

					msg, _ := json.Marshal(message)
					fmt.Fprintf(w, "data: %s\n\n", msg)
				}
				m.Mu.RUnlock()
				flusher.Flush()
			}
		}
	})

	fmt.Println("Server started at http://localhost:8282")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error(err.Error())
		fmt.Printf("❌ Server error: %v\n", err)
	}
}
