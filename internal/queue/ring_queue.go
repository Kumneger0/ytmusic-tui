package queue

import (
	"container/ring"
	"sync"

	"github.com/kumneger0/ytmusic-tui/internal/types"
)

type RingQueue struct {
	mu      sync.RWMutex
	current *ring.Ring
}

func NewRingQueue() *RingQueue {
	return &RingQueue{}
}

func (rq *RingQueue) Len() int {
	rq.mu.RLock()
	defer rq.mu.RUnlock()
	if rq.current == nil {
		return 0
	}
	return rq.current.Len()
}

func (rq *RingQueue) Current() *types.PlaylistTrackObject {
	rq.mu.RLock()
	defer rq.mu.RUnlock()
	if rq.current == nil || rq.current.Value == nil {
		return nil
	}
	track, ok := rq.current.Value.(*types.PlaylistTrackObject)
	if !ok {
		return nil
	}
	return track
}

func (rq *RingQueue) AddTrack(track *types.PlaylistTrackObject) {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	if track == nil {
		return
	}

	newNode := ring.New(1)
	newNode.Value = track

	if rq.current == nil {
		rq.current = newNode
	} else {
		rq.current.Prev().Link(newNode)
	}
}

func (rq *RingQueue) PlayNextTrack(track *types.PlaylistTrackObject) {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	if track == nil {
		return
	}

	newNode := ring.New(1)
	newNode.Value = track

	if rq.current == nil {
		rq.current = newNode
	} else {
		rq.current.Link(newNode)
	}
}

func (rq *RingQueue) Next() *types.PlaylistTrackObject {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	if rq.current == nil {
		return nil
	}
	rq.current = rq.current.Next()
	track, _ := rq.current.Value.(*types.PlaylistTrackObject)
	return track
}

func (rq *RingQueue) Prev() *types.PlaylistTrackObject {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	if rq.current == nil {
		return nil
	}
	rq.current = rq.current.Prev()
	track, _ := rq.current.Value.(*types.PlaylistTrackObject)
	return track
}

func (rq *RingQueue) SetTracks(tracks []*types.PlaylistTrackObject) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	if len(tracks) == 0 {
		rq.current = nil
		return
	}

	r := ring.New(len(tracks))
	for _, track := range tracks {
		r.Value = track
		r = r.Next()
	}
	rq.current = r
}

func (rq *RingQueue) PopFirst() *types.PlaylistTrackObject {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	if rq.current == nil {
		return nil
	}

	track, _ := rq.current.Value.(*types.PlaylistTrackObject)

	if rq.current.Len() == 1 {
		rq.current = nil
	} else {
		prev := rq.current.Prev()
		prev.Unlink(1)
		rq.current = prev.Next()
	}

	return track
}

func (rq *RingQueue) AllTracks() []*types.PlaylistTrackObject {
	rq.mu.RLock()
	defer rq.mu.RUnlock()

	if rq.current == nil {
		return nil
	}

	n := rq.current.Len()
	tracks := make([]*types.PlaylistTrackObject, 0, n)
	curr := rq.current
	for i := 0; i < n; i++ {
		if track, ok := curr.Value.(*types.PlaylistTrackObject); ok && track != nil {
			tracks = append(tracks, track)
		}
		curr = curr.Next()
	}
	return tracks
}

func (rq *RingQueue) RemoveTrackAtIndex(index int) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	if rq.current == nil || index < 0 || index >= rq.current.Len() {
		return
	}

	if rq.current.Len() == 1 {
		rq.current = nil
		return
	}

	target := rq.current
	for i := 0; i < index; i++ {
		target = target.Next()
	}

	if target == rq.current {
		prev := rq.current.Prev()
		prev.Unlink(1)
		rq.current = prev.Next()
	} else {
		prev := target.Prev()
		prev.Unlink(1)
	}
}

func (rq *RingQueue) RemoveCurrent() {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	if rq.current == nil {
		return
	}
	if rq.current.Len() == 1 {
		rq.current = nil
		return
	}
	prev := rq.current.Prev()
	prev.Unlink(1)
	rq.current = prev.Next()
}

func (rq *RingQueue) Clear() {
	rq.mu.Lock()
	defer rq.mu.Unlock()
	rq.current = nil
}
