package queue

import (
	"testing"

	musicpb "github.com/kumneger0/ytmusic-tui/gen"
	"github.com/kumneger0/ytmusic-tui/internal/types"
)

func TestRingQueue_Empty(t *testing.T) {
	rq := NewRingQueue()
	if rq.Len() != 0 {
		t.Fatalf("expected len 0, got %d", rq.Len())
	}
	if rq.Current() != nil {
		t.Fatal("expected nil current track")
	}
	if rq.Next() != nil {
		t.Fatal("expected nil next track")
	}
	if rq.Prev() != nil {
		t.Fatal("expected nil prev track")
	}
}

func TestRingQueue_AddAndNavigate(t *testing.T) {
	rq := NewRingQueue()
	t1 := &types.PlaylistTrackObject{Track: &musicpb.Song{VideoId: "v1", Title: "Song 1"}}
	t2 := &types.PlaylistTrackObject{Track: &musicpb.Song{VideoId: "v2", Title: "Song 2"}}

	rq.AddTrack(t1)
	rq.AddTrack(t2)

	if rq.Len() != 2 {
		t.Fatalf("expected len 2, got %d", rq.Len())
	}
	if rq.Current().Track.VideoId != "v1" {
		t.Fatalf("expected current v1, got %s", rq.Current().Track.VideoId)
	}

	next := rq.Next()
	if next.Track.VideoId != "v2" {
		t.Fatalf("expected next v2, got %s", next.Track.VideoId)
	}

	nextWrap := rq.Next()
	if nextWrap.Track.VideoId != "v1" {
		t.Fatalf("expected wrap v1, got %s", nextWrap.Track.VideoId)
	}

	prev := rq.Prev()
	if prev.Track.VideoId != "v2" {
		t.Fatalf("expected prev v2, got %s", prev.Track.VideoId)
	}
}

func TestRingQueue_PlayNextTrack(t *testing.T) {
	rq := NewRingQueue()
	t1 := &types.PlaylistTrackObject{Track: &musicpb.Song{VideoId: "v1"}}
	t2 := &types.PlaylistTrackObject{Track: &musicpb.Song{VideoId: "v2"}}
	t3 := &types.PlaylistTrackObject{Track: &musicpb.Song{VideoId: "v3"}}

	rq.AddTrack(t1)
	rq.AddTrack(t2)
	rq.PlayNextTrack(t3)

	if rq.Len() != 3 {
		t.Fatalf("expected len 3, got %d", rq.Len())
	}

	if rq.Current().Track.VideoId != "v1" {
		t.Fatalf("expected current v1, got %s", rq.Current().Track.VideoId)
	}
	if rq.Next().Track.VideoId != "v3" {
		t.Fatalf("expected next v3, got %s", rq.Current().Track.VideoId)
	}
}

func TestRingQueue_SetAndRemove(t *testing.T) {
	rq := NewRingQueue()
	tracks := []*types.PlaylistTrackObject{
		{Track: &musicpb.Song{VideoId: "v1"}},
		{Track: &musicpb.Song{VideoId: "v2"}},
	}
	rq.SetTracks(tracks)
	if rq.Len() != 2 {
		t.Fatalf("expected len 2, got %d", rq.Len())
	}

	rq.RemoveCurrent()
	if rq.Len() != 1 {
		t.Fatalf("expected len 1 after remove, got %d", rq.Len())
	}
	if rq.Current().Track.VideoId != "v2" {
		t.Fatalf("expected current v2 after remove, got %s", rq.Current().Track.VideoId)
	}
}
