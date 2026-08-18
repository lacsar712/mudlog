package dlq

import (
	"sync"
	"time"

	"github.com/lacsar712/mudlog/internal/job"
	"github.com/lacsar712/mudlog/internal/redact"
)

type Item struct {
	DeadAt       time.Time `json:"dead_at"`
	Reason       string    `json:"reason"`
	FrameID      string    `json:"frame_id"`
	RelayID      string    `json:"relay_id"`
	StoreID      string    `json:"store_id"`
	Type         string    `json:"type"`
	Attempt      int       `json:"attempt"`
	Body         []byte    `json:"body"`
	BodyRedacted []byte    `json:"body_redacted,omitempty"`
	ReplayOf     string    `json:"replay_of,omitempty"`
}

func FromJob(j job.Job, reason string, now time.Time) Item {
	return Item{
		DeadAt:       now,
		Reason:       reason,
		FrameID:      j.FrameID,
		RelayID:      j.RelayID,
		StoreID:      j.StoreID,
		Type:         j.Type,
		Attempt:      j.Attempt,
		Body:         append([]byte(nil), j.Body...),
		BodyRedacted: redact.JSON(j.Body),
		ReplayOf:     j.ReplayOf,
	}
}

type Queue struct {
	mu    sync.Mutex
	max   int
	items []Item
}

func New(max int) *Queue {
	if max < 20 {
		max = 100
	}
	return &Queue{max: max}
}

func (q *Queue) Push(it Item) {
	if len(it.Body) > 0 && len(it.BodyRedacted) == 0 {
		it.BodyRedacted = redact.JSON(it.Body)
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, it)
	if len(q.items) > q.max {
		q.items = append([]Item(nil), q.items[len(q.items)-q.max:]...)
	}
}

func (q *Queue) List() []Item {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]Item, len(q.items))
	for i := range q.items {
		cp := q.items[len(q.items)-1-i]
		cp.Body = nil
		out[i] = cp
	}
	return out
}

func (q *Queue) Get(relayID string) (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := len(q.items) - 1; i >= 0; i-- {
		if q.items[i].RelayID == relayID {
			return q.items[i], true
		}
	}
	return Item{}, false
}

func (q *Queue) Remove(relayID string) (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i, it := range q.items {
		if it.RelayID == relayID {
			q.items = append(q.items[:i], q.items[i+1:]...)
			return it, true
		}
	}
	return Item{}, false
}

func (q *Queue) Snapshot() []Item {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]Item, len(q.items))
	copy(out, q.items)
	return out
}

func (q *Queue) Restore(items []Item) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append([]Item(nil), items...)
}

func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}
