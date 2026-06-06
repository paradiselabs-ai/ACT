package agent

import (
	"testing"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/message"
)

func TestScopedHistory(t *testing.T) {
	msgs := []message.Message{
		{ID: "a", Role: message.User},
		{ID: "b", Role: message.Assistant},
		{ID: "c", Role: message.User},
	}

	t.Run("scope=true drops the prior transcript", func(t *testing.T) {
		got := scopedHistory(msgs, true)
		if got != nil {
			t.Fatalf("scopeHistory=true: want nil, got %d messages", len(got))
		}
	})

	t.Run("scope=false keeps the full transcript", func(t *testing.T) {
		got := scopedHistory(msgs, false)
		if len(got) != len(msgs) {
			t.Fatalf("scopeHistory=false: want %d messages, got %d", len(msgs), len(got))
		}
		for i := range msgs {
			if got[i].ID != msgs[i].ID {
				t.Fatalf("scopeHistory=false: message %d reordered/changed: want %q, got %q", i, msgs[i].ID, got[i].ID)
			}
		}
	})

	t.Run("empty input is safe under both", func(t *testing.T) {
		if got := scopedHistory(nil, true); got != nil {
			t.Fatalf("nil/scope=true: want nil, got %v", got)
		}
		if got := scopedHistory([]message.Message{}, false); len(got) != 0 {
			t.Fatalf("empty/scope=false: want 0, got %d", len(got))
		}
	})
}
