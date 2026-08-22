package logging

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logfmt/logfmt"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/pubsub"
)

const (
	persistKeyArg  = "$_persist"
	PersistTimeArg = "$_persist_time"

	// maxLogMessages bounds the in-memory log ring. Without it every
	// InfoPersist across a hours-long swarm run grows the slice forever
	// (memory) while logs/table.go re-sorts the whole thing on every event
	// (CPU). Oldest entries fall off the front.
	maxLogMessages = 2000
)

type LogData struct {
	messages []LogMessage
	*pubsub.Broker[LogMessage]
	lock sync.Mutex
}

func (l *LogData) Add(msg LogMessage) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.messages = append(l.messages, msg)
	if len(l.messages) > maxLogMessages {
		// Ring trim: drop the oldest block, keep append amortized O(1).
		// Copying into a fresh slice would re-grow from 0; reslicing keeps
		// capacity and lets the next appends reuse the dropped prefix.
		drop := len(l.messages) - maxLogMessages
		l.messages = append(l.messages[:0], l.messages[drop:]...)
	}
	l.Publish(pubsub.CreatedEvent, msg)
}

func (l *LogData) List() []LogMessage {
	l.lock.Lock()
	defer l.lock.Unlock()
	// Defensive copy: callers sort and index the result (logs/table.go
	// slices.SortFunc) while concurrent Add() calls append under this same
	// lock. Returning the backing array was a data race.
	out := make([]LogMessage, len(l.messages))
	copy(out, l.messages)
	return out
}

var defaultLogData = &LogData{
	messages: make([]LogMessage, 0),
	Broker:   pubsub.NewBroker[LogMessage](),
}

type writer struct{}

func (w *writer) Write(p []byte) (int, error) {
	d := logfmt.NewDecoder(bytes.NewReader(p))

	for d.ScanRecord() {
		msg := LogMessage{
			ID:   fmt.Sprintf("%d", time.Now().UnixNano()),
			Time: time.Now(),
		}
		for d.ScanKeyval() {
			switch string(d.Key()) {
			case "time":
				parsed, err := time.Parse(time.RFC3339, string(d.Value()))
				if err != nil {
					return 0, fmt.Errorf("parsing time: %w", err)
				}
				msg.Time = parsed
			case "level":
				msg.Level = strings.ToLower(string(d.Value()))
			case "msg":
				msg.Message = string(d.Value())
			default:
				if string(d.Key()) == persistKeyArg {
					msg.Persist = true
				} else if string(d.Key()) == PersistTimeArg {
					parsed, err := time.ParseDuration(string(d.Value()))
					if err != nil {
						continue
					}
					msg.PersistTime = parsed
				} else {
					msg.Attributes = append(msg.Attributes, Attr{
						Key:   string(d.Key()),
						Value: string(d.Value()),
					})
				}
			}
		}
		defaultLogData.Add(msg)
	}
	if d.Err() != nil {
		return 0, d.Err()
	}
	return len(p), nil
}

func NewWriter() *writer {
	w := &writer{}
	return w
}

func Subscribe(ctx context.Context) <-chan pubsub.Event[LogMessage] {
	return defaultLogData.Subscribe(ctx)
}

// SubscribeBroker exposes the logging event broker so hosts can install
// slow-consumer drop telemetry (OnDrop) or read Dropped() counters.
func SubscribeBroker() *pubsub.Broker[LogMessage] {
	return defaultLogData.Broker
}

func List() []LogMessage {
	return defaultLogData.List()
}

// Count returns the number of buffered log messages without copying them.
// Use instead of len(List()) — the defensive copy in List exists for callers
// that mutate the result, and a header count doesn't need to pay for it.
func Count() int {
	defaultLogData.lock.Lock()
	defer defaultLogData.lock.Unlock()
	return len(defaultLogData.messages)
}
