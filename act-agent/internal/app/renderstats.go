package app

import (
	"sync/atomic"
	"time"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
)

// Render-timing counters shared between the chat TUI component (writer) and
// the orchestrator's Observer tick (reader). Lives in the app package to
// break an otherwise-circular import — chat already imports app, and app's
// orchestrator wants to read these stats without importing chat.
//
// All counters are atomics; no locks needed. Bump functions do the slow-path
// logging themselves so callers stay one-liner.

var (
	renderCount     int64
	slowRenderCount int64
	lastRenderMsgs  int64
)

// SlowRenderThreshold is the render duration above which a call is considered
// slow. Tuned for cached-render baseline: uncached renders on a healthy
// session finish in single-digit ms; anything over this is doing real work
// that should be visible in logs.
const SlowRenderThreshold = 50 * time.Millisecond

// BumpRender records a single renderView invocation. Called from the TUI
// message-list component's renderView. Elapsed is the wall-clock duration of
// the call; msgCount is the number of messages the render walked.
func BumpRender(elapsed time.Duration, msgCount int) {
	atomic.AddInt64(&renderCount, 1)
	atomic.StoreInt64(&lastRenderMsgs, int64(msgCount))
	if elapsed >= SlowRenderThreshold {
		atomic.AddInt64(&slowRenderCount, 1)
		logging.Info("render_slow",
			"ms", elapsed.Milliseconds(),
			"msgs", msgCount,
		)
	}
}

// RenderStatsSnapshot returns a point-in-time view of the render counters.
// Called by the orchestrator's Observer tick to emit periodic render_stats
// health lines.
func RenderStatsSnapshot() (total, slow, lastMsgs int64) {
	return atomic.LoadInt64(&renderCount), atomic.LoadInt64(&slowRenderCount), atomic.LoadInt64(&lastRenderMsgs)
}
