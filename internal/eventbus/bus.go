package eventbus

import (
	"sync"
	"sync/atomic"

	"github.com/shiyisir/claude-proxy/internal/logging"
)

// Event is a normalized event ready for broadcast.
type Event struct {
	SessionID string
	Data      []byte // pre-serialized JSON
}

// Subscriber receives events on its channel. Non-blocking on send.
type Subscriber struct {
	Ch   chan *Event
	done chan struct{}
}

// Bus is a non-blocking event broadcaster.
type Bus struct {
	mu          sync.RWMutex
	subs        map[*Subscriber]struct{}
	maxQueue    int
	dropped     atomic.Int64
}

// New creates a Bus with the given per-subscriber queue size.
func New(maxQueue int) *Bus {
	return &Bus{
		subs:     make(map[*Subscriber]struct{}),
		maxQueue: maxQueue,
	}
}

// Subscribe returns a new Subscriber that receives events.
func (b *Bus) Subscribe() *Subscriber {
	s := &Subscriber{
		Ch:   make(chan *Event, b.maxQueue),
		done: make(chan struct{}),
	}
	b.mu.Lock()
	b.subs[s] = struct{}{}
	b.mu.Unlock()
	return s
}

// Unsubscribe removes a subscriber.
func (b *Bus) Unsubscribe(s *Subscriber) {
	b.mu.Lock()
	delete(b.subs, s)
	b.mu.Unlock()
	close(s.done)
}

// Publish sends an event to all subscribers. Never blocks.
func (b *Bus) Publish(ev *Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for s := range b.subs {
		select {
		case s.Ch <- ev:
		default:
			b.dropped.Add(1)
		}
	}
}

// SubscriberCount returns the number of active subscribers.
func (b *Bus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}

// Dropped returns total dropped events across all subscribers.
func (b *Bus) Dropped() int64 {
	return b.dropped.Load()
}

// Stats returns subscriber count and dropped count.
func (b *Bus) Stats() (subs int, dropped int64) {
	b.mu.RLock()
	subs = len(b.subs)
	b.mu.RUnlock()
	dropped = b.dropped.Load()
	return
}

// Ensure logging is imported
var _ = logging.Info
