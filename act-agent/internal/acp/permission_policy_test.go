package acp

import (
	"encoding/json"
	"testing"
)

func TestPermissionAllowed(t *testing.T) {
	cases := []struct {
		role, kind, desc string
		want             bool
	}{
		// Verification reads are fine for every role.
		{"planner", "read", "view spend.py", true},
		{"assurance", "search", "grep success_criteria", true},
		{"qa_synthesizer", "think", "", true},
		// The role shim is the only sanctioned execute.
		{"planner", "execute", "act-tier1-planner status", true},
		{"observer", "execute", "act-tier1-observer log --tail 20", true},
		// Cross-role shim use is denied.
		{"planner", "execute", "act-tier1-assurance validation queue", false},
		// Arbitrary commands are denied — the antigravity failure mode.
		{"planner", "execute", "python3 -m pytest test_spend.py", false},
		{"planner", "execute", "mkdir -p scratch && touch spend.py", false},
		// Writes are denied for all Tier 1 roles.
		{"planner", "edit", "write spend.py", false},
		{"assurance", "delete", "rm -rf build", false},
		{"observer", "move", "mv a b", false},
		// Fetch/other/unknown fail closed.
		{"planner", "fetch", "https://example.com", false},
		{"planner", "other", "", false},
		{"planner", "", "mystery tool", false},
	}
	for _, c := range cases {
		if got := permissionAllowed(c.role, c.kind, c.desc); got != c.want {
			t.Errorf("permissionAllowed(%q,%q,%q) = %v, want %v", c.role, c.kind, c.desc, got, c.want)
		}
	}
}

func TestAnswerPermissionRequest(t *testing.T) {
	mk := func(kind, title string, opts ...PermissionOption) json.RawMessage {
		b, _ := json.Marshal(RequestPermissionParams{
			SessionID: "s1",
			ToolCall:  PermissionToolCall{Kind: kind, Title: title},
			Options:   opts,
		})
		return b
	}
	allowOnce := PermissionOption{OptionID: "a1", Kind: "allow_once"}
	allowAlways := PermissionOption{OptionID: "a2", Kind: "allow_always"}
	rejectOnce := PermissionOption{OptionID: "r1", Kind: "reject_once"}

	// Allowed read → allow_once picked.
	res, rpcErr := answerPermissionRequest("planner", mk("read", "view file", allowOnce, allowAlways, rejectOnce))
	if rpcErr != nil {
		t.Fatalf("unexpected rpc error: %v", rpcErr)
	}
	if out := res.(RequestPermissionResult).Outcome; out.Outcome != "selected" || out.OptionID != "a1" {
		t.Fatalf("read: want selected/a1, got %+v", out)
	}

	// Denied edit → reject_once picked.
	res, _ = answerPermissionRequest("planner", mk("edit", "write file", allowOnce, rejectOnce))
	if out := res.(RequestPermissionResult).Outcome; out.OptionID != "r1" {
		t.Fatalf("edit: want r1, got %+v", out)
	}

	// Denied with no reject option offered → cancelled.
	res, _ = answerPermissionRequest("planner", mk("edit", "write file", allowOnce))
	if out := res.(RequestPermissionResult).Outcome; out.Outcome != "cancelled" {
		t.Fatalf("edit w/o reject option: want cancelled, got %+v", out)
	}

	// Family fallback: only allow_always offered for an allowed call.
	res, _ = answerPermissionRequest("planner", mk("read", "view", allowAlways, rejectOnce))
	if out := res.(RequestPermissionResult).Outcome; out.OptionID != "a2" {
		t.Fatalf("family fallback: want a2, got %+v", out)
	}

	// Malformed params → RPC error, not a silent allow.
	if _, rpcErr := answerPermissionRequest("planner", json.RawMessage(`{"options": "nope"}`)); rpcErr == nil {
		t.Fatalf("malformed params should return an RPC error")
	}
}
