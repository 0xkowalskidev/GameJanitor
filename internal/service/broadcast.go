package service

import (
	"sync"
	"time"
)

type StatusEvent struct {
	GameserverID string    `json:"gameserver_id"`
	OldStatus    string    `json:"old_status"`
	NewStatus    string    `json:"new_status"`
	ErrorReason  string    `json:"error_reason,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

type EventBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[uint64]chan StatusEvent
	nextID      uint64
}

func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{
		subscribers: make(map[uint64]chan StatusEvent),
	}
}

// Subscribe returns a channel that receives status events and an unsubscribe function.
func (b *EventBroadcaster) Subscribe() (<-chan StatusEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.nextID
	b.nextID++
	ch := make(chan StatusEvent, 64)
	b.subscribers[id] = ch

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.subscribers, id)
		close(ch)
	}

	return ch, unsubscribe
}

// Publish sends an event to all subscribers. Non-blocking: slow clients miss events.
func (b *EventBroadcaster) Publish(event StatusEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}
