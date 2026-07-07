package acp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/logging"
)

// Per-role tool-permission policy for ACP-backed Tier 1 agents.
//
// Mirrors the in-process rule (KI-02: no Tier 1 role gets raw bash or file
// writes): Tier 1 roles observe, decide, and validate — they never edit the
// project. The single sanctioned side door is the role's act-tier1-<role>
// shim, which enforces its own subcommand allowlist. So:
//
//	read / search / think  → allowed (Assurance and QA verify with these)
//	execute                → allowed ONLY when the command is the role shim
//	edit / delete / move / fetch / other / unknown → denied
//
// This is hard enforcement at the protocol boundary — it binds any ACP
// backend that routes permission requests, regardless of what the role
// prompt says or how the model drifts.

// permissionAllowed decides one tool-permission request for a Tier 1 role.
// desc is the human-readable context (title + raw input) used to recognize
// shim invocations.
func permissionAllowed(role, kind, desc string) bool {
	switch kind {
	case "read", "search", "think":
		return true
	case "execute":
		return strings.Contains(desc, "act-tier1-"+role)
	default:
		return false
	}
}

// answerPermissionRequest implements the session/request_permission handler
// for one role. Picks the agent-offered option matching the policy decision:
// allow → allow_once, deny → reject_once (with prefix fallbacks so
// allow_always/reject_always variants still resolve). No matching option →
// outcome "cancelled", which agents treat as a denial.
func answerPermissionRequest(role string, params json.RawMessage) (any, *RPCError) {
	var req RequestPermissionParams
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &RPCError{Code: -32602, Message: fmt.Sprintf("bad request_permission params: %v", err)}
	}

	desc := req.ToolCall.Title + " " + string(req.ToolCall.RawInput)
	allowed := permissionAllowed(role, req.ToolCall.Kind, desc)
	logging.Info("acp_permission_decision",
		"role", role,
		"tool_kind", req.ToolCall.Kind,
		"title", req.ToolCall.Title,
		"allowed", allowed,
	)

	want := "reject"
	if allowed {
		want = "allow"
	}
	// Exact once-variant first, then any option of the right family.
	for _, opt := range req.Options {
		if opt.Kind == want+"_once" {
			return RequestPermissionResult{Outcome: PermissionOutcome{Outcome: "selected", OptionID: opt.OptionID}}, nil
		}
	}
	for _, opt := range req.Options {
		if strings.HasPrefix(opt.Kind, want) {
			return RequestPermissionResult{Outcome: PermissionOutcome{Outcome: "selected", OptionID: opt.OptionID}}, nil
		}
	}
	return RequestPermissionResult{Outcome: PermissionOutcome{Outcome: "cancelled"}}, nil
}
