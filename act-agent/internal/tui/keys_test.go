package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/components/chat"
	"github.com/paradiselabs-ai/ACT/act-agent/internal/tui/layout"
)

// layoutKeyMapToSlice is a thin typed wrapper so the test can enumerate the
// appModel key map without reflection at every call site.
func layoutKeyMapToSlice(m any) []key.Binding { return layout.KeyMapToSlice(m) }

// chatPageBindingKeys constructs the same binding set chatPage.BindingKeys()
// aggregates (messages + editor) without needing a full App.
func chatPageBindingKeys() []key.Binding {
	editor := chat.NewEditorCmp(nil)
	msgs := chat.NewMessagesCmp(nil)
	var out []key.Binding
	out = append(out, editor.(interface{ BindingKeys() []key.Binding }).BindingKeys()...)
	out = append(out, msgs.(interface{ BindingKeys() []key.Binding }).BindingKeys()...)
	return out
}

// TestNoDuplicateGlobalBindings asserts that the appModel-level key map and
// the page/component key maps never claim the same key sequence. A duplicate
// is not a compile error: the first switch case silently wins, the second
// binding becomes unreachable, and the help overlay then shows one arbitrary
// description for the contested key (audit finding H1: ctrl+e was bound to
// both "rename session" and "open editor").
//
// Known intentional overlap inside one component is exempt: editorMaps.Send
// and DeleteKeyMaps live in different scopes and are handled by guards, and
// viewport page keys are re-exported verbatim by messagesCmp.BindingKeys —
// those appear once per source map, so exact-sequence dedup within a single
// owner is filtered before comparison.
func TestNoDuplicateGlobalBindings(t *testing.T) {
	claim := func(owner string, bs []key.Binding, seen map[string]string) {
		perOwner := map[string]bool{}
		for _, b := range bs {
			seq := strings.Join(b.Keys(), " ")
			if seq == "" || perOwner[seq] {
				continue // empty or same-owner duplicate (re-export)
			}
			perOwner[seq] = true
			if prev, dup := seen[seq]; dup {
				t.Errorf("key %q claimed by %q and %q", seq, prev, owner)
			}
			seen[seq] = owner
		}
	}

	seen := map[string]string{}

	// appModel-level bindings (the first-match layer).
	claim("tui.keys", layoutKeyMapToSlice(keys), seen)

	// Page-level bindings: chat aggregates editor + message list.
	claim("chat.page", chatPageBindingKeys(), seen)

	// The rebind that motivated this test must hold specifically:
	// alt+e belongs to the editor; ctrl+e belongs to rename-session only.
	foundAltE, foundCtrlEinEditor := false, false
	for _, b := range chatPageBindingKeys() {
		seq := strings.Join(b.Keys(), " ")
		if strings.Contains(seq, "alt+e") {
			foundAltE = true
		}
		if strings.Contains(seq, "ctrl+e") {
			foundCtrlEinEditor = true
		}
	}
	if !foundAltE {
		t.Error("editor open-in-$EDITOR is not bound to alt+e (H1 regression)")
	}
	if foundCtrlEinEditor {
		t.Error("editor still claims ctrl+e; it belongs to appModel rename-session")
	}
}
