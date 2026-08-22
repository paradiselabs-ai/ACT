package pubsub

import (
	"context"
	"sync"
	"sync/atomic"
)

const bufferSize = 64

type Broker[T any] struct {
	subs      map[chan Event[T]]struct{}
	mu        sync.RWMutex
	done      chan struct{}
	subCount  int
	maxEvents int

	// dropped counts events discarded because a subscriber's buffer was
	// full at Publish time. Read via Dropped() for health/telemetry; a
	// host can install OnDrop to surface slow-consumer conditions (the
	// TUI wires this to a warn log) without every Publish paying for it.
	dropped atomic.Int64
	onDrop  func(eventType EventType)
}

func NewBroker[T any]() *Broker[T] {
	return NewBrokerWithOptions[T](bufferSize, 1000)
}

func NewBrokerWithOptions[T any](channelBufferSize, maxEvents int) *Broker[T] {
	b := &Broker[T]{
		subs:      make(map[chan Event[T]]struct{}),
		done:      make(chan struct{}),
		subCount:  0,
		maxEvents: maxEvents,
	}
	return b
}

// Dropped reports how many events have been silently discarded due to
// full subscriber buffers across the broker's lifetime.
func (b *Broker[T]) Dropped() int64 { return b.dropped.Load() }

// OnDrop installs a hook invoked (from the publishing goroutine) whenever
// an event is discarded. Keep the handler cheap and lock-free — it runs on
// every drop. Passing nil disables the hook.
func (b *Broker[T]) OnDrop(fn func(EventType)) { b.onDrop = fn }

func (b *Broker[T]) Shutdown() {
	select {
	case <-b.done: // Already closed
		return
	default:
		close(b.done)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for ch := range b.subs {
		delete(b.subs, ch)
		close(ch)
	}

	b.subCount = 0
}

func (b *Broker[T]) Subscribe(ctx context.Context) <-chan Event[T] {
	b.mu.Lock()
	defer b.mu.Unlock()

	select {
	case <-b.done:
		ch := make(chan Event[T])
		close(ch)
		return ch
	default:
	}

	sub := make(chan Event[T], bufferSize)
	b.subs[sub] = struct{}{}
	b.subCount++

	go func() {
		<-ctx.Done()

		b.mu.Lock()
		defer b.mu.Unlock()

		select {
		case <-b.done:
			return
		default:
		}

		delete(b.subs, sub)
		close(sub)
		b.subCount--
	}()

	return sub
}

func (b *Broker[T]) GetSubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.subCount
}

func (b *Broker[T]) Publish(t EventType, payload T) {
	b.mu.RLock()
	select {
	case <-b.done:
		b.mu.RUnlock()
		return
	default:
	}

	subscribers := make([]chan Event[T], 0, len(b.subs))
	for sub := range b.subs {
		subscribers = append(subscribers, sub)
	}
	b.mu.RUnlock()

	event := Event[T]{Type: t, Payload: payload}

	for _, sub := range subscribers {
		select {
		case sub <- event:
		default:
			// Subscriber buffer full. Never block the publisher, but the
			// loss must not be silent — coordination events ARE the
			// product here. Counted for Dropped(); OnDrop (if installed)
			// surfaces it to logging.
			b.dropped.Add(1)
			if b.onDrop != nil {
				b.onDrop(t)
			}
		}
	}
}
