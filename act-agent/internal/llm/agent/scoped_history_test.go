package agent

import (
	"testing"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
)

func TestScopedHistory(t *testing.T) {
	msgs := []message.Message{
		{ID: "a", Role: message.User, ThreadID: "planner"},
		{ID: "b", Role: message.Assistant, ThreadID: "observer"},
		{ID: "c", Role: message.User, ThreadID: "planner"},
		{ID: "d", Role: message.Assistant, ThreadID: ""},
	}

	t.Run("HistoryNone drops everything", func(t *testing.T) {
		if got := scopedHistory(msgs, HistoryNone, "planner"); got != nil {
			t.Fatalf("HistoryNone: want nil, got %d messages", len(got))
		}
	})

	t.Run("HistoryFull keeps everything in order", func(t *testing.T) {
		got := scopedHistory(msgs, HistoryFull, "ignored")
		if len(got) != len(msgs) {
			t.Fatalf("HistoryFull: want %d, got %d", len(msgs), len(got))
		}
		for i := range msgs {
			if got[i].ID != msgs[i].ID {
				t.Fatalf("HistoryFull: message %d changed: want %q got %q", i, msgs[i].ID, got[i].ID)
			}
		}
	})

	t.Run("HistoryThread keeps only the matching threadID", func(t *testing.T) {
		got := scopedHistory(msgs, HistoryThread, "planner")
		if len(got) != 2 {
			t.Fatalf("HistoryThread(planner): want 2 messages, got %d", len(got))
		}
		if got[0].ID != "a" || got[1].ID != "c" {
			t.Fatalf("HistoryThread(planner): want [a c], got [%s %s]", got[0].ID, got[1].ID)
		}
	})

	t.Run("HistoryThread excludes other agents' traffic", func(t *testing.T) {
		got := scopedHistory(msgs, HistoryThread, "observer")
		if len(got) != 1 || got[0].ID != "b" {
			t.Fatalf("HistoryThread(observer): want [b], got %d messages", len(got))
		}
	})

	t.Run("empty input is safe under every mode", func(t *testing.T) {
		for _, mode := range []HistoryMode{HistoryNone, HistoryThread, HistoryFull} {
			if got := scopedHistory(nil, mode, "planner"); len(got) != 0 {
				t.Fatalf("nil/%q: want 0, got %d", mode, len(got))
			}
		}
	})
}
