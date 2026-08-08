package acp

import "strings"

// The priming payload the injector closure hands us (app.go::wrapPriming) is
// shaped for a USER message: InternalPromptMarker + do-not-respond header +
// role prompt + shim note. That framing exists only because ACP's ContentBlock
// has no system channel.
//
// The Claude Code ACP bridge does expose one: session/new accepts
// `_meta.systemPrompt`, and an object there is spread over the claude_code
// preset with type/preset locked, so `{"append": "<text>"}` appends our role
// text to Claude Code's REAL system prompt (verified in
// @agentclientprotocol/claude-agent-acp 0.37 dist/acp-agent.js — the
// `params._meta?.systemPrompt` branch). A system prompt outranks a user
// message, which is the whole point: claude-code Tier 1 roles were ignoring
// user-message priming.
//
// So for claude-code we strip the user-message framing and ship the role text
// as a system append instead — and skip the priming turn entirely. Every other
// backend keeps the user-message path byte-for-byte; their bridges may ignore
// or choke on unknown _meta, so they never see it.

// internalPromptMarker mirrors app.InternalPromptMarker. Duplicated rather
// than imported because app imports acp — importing back would be a cycle.
// It is a sentinel for TUI/log filtering of injected prompts and must never
// reach a system prompt.
const internalPromptMarker = "\x00ACT_INTERNAL\x00"

// doNotRespondHeaderPrefix is the leading run of app.doNotRespondHeader. We
// match on the prefix and cut through the blank line that terminates the
// header rather than requiring the full string, so wording tweaks upstream
// don't silently leak the header into the system prompt.
const doNotRespondHeaderPrefix = "[ACT priming — do not respond"

// stripPrimingWrappers removes the user-message framing from a priming
// payload, leaving the role prompt (+ shim note) alone. Returns "" when
// nothing survives.
func stripPrimingWrappers(payload string) string {
	// ReplaceAll rather than TrimPrefix: the marker must not survive anywhere
	// in a system prompt, not merely at position 0.
	rest := strings.ReplaceAll(payload, internalPromptMarker, "")
	rest = strings.TrimLeft(rest, " \t\r\n")

	if strings.HasPrefix(rest, doNotRespondHeaderPrefix) {
		switch {
		case strings.Contains(rest, "\n\n"):
			rest = rest[strings.Index(rest, "\n\n")+2:]
		case strings.Contains(rest, "\n"):
			rest = rest[strings.Index(rest, "\n")+1:]
		default:
			// Header with no body — nothing to append.
			rest = ""
		}
	}
	return strings.TrimSpace(rest)
}

// primingPlan is the per-backend transport decision for a role payload.
// Exactly one field is populated in the normal case; both are empty when
// there is no priming to deliver.
type primingPlan struct {
	// SystemAppend goes into session/new `_meta.systemPrompt.append`.
	SystemAppend string
	// PromptText is sent as the first user message of the ACP session.
	PromptText string
}

// planPriming decides how host receives payload.
//
// claude-code (and the empty host, which spawns the claude-code defaults —
// see buildCommand) gets a real system-prompt append and no priming turn.
// Everything else keeps the user-message priming turn with no _meta.
//
// Fail-safe: if stripping leaves nothing (payload was only framing, or the
// framing shape changed such that we'd send an empty append), fall back to
// the unmodified user-message priming rather than shipping a session with no
// role prompt at all.
func planPriming(host, payload string) primingPlan {
	if payload == "" {
		return primingPlan{}
	}
	if !isClaudeACPHost(host) {
		return primingPlan{PromptText: payload}
	}
	stripped := stripPrimingWrappers(payload)
	if stripped == "" {
		return primingPlan{PromptText: payload}
	}
	return primingPlan{SystemAppend: stripped}
}

// isClaudeACPHost reports whether host is served by the Claude Code ACP
// bridge. Mirrors the "claude-code", "" case in buildCommand — keep the two
// in sync if a new claude-flavoured alias is added there.
func isClaudeACPHost(host string) bool {
	return host == "claude-code" || host == ""
}
