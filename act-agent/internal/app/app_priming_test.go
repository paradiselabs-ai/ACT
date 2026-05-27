package app

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/paradiselabs-ai/ACT/act-agent/internal/llm/tools"
)

// TestACPPrimingMatchesAllowlist locks the bullet list inside the ACP
// priming text (renderShimNote) to tools.AllowedFor(role). Adding a new
// entry to the whitelist OR forgetting to update the priming-generation
// code would surface here, not in the next ACP-backed session where the
// Planner would silently not know about the new affordance.
func TestACPPrimingMatchesAllowlist(t *testing.T) {
	for _, role := range []string{"planner", "observer", "assurance", "qa_synthesizer"} {
		t.Run(role, func(t *testing.T) {
			note := renderShimNote(role)
			advertised := extractAllowlistBullets(t, note)
			want := tools.AllowedFor(role)

			// Both slices have a deterministic order (AllowedFor preserves
			// definition order; the renderer iterates the same slice). Compare
			// as sorted sets to guard against future renderer changes that
			// might re-order the bullets.
			sortedAdv := append([]string(nil), advertised...)
			sortedWant := append([]string(nil), want...)
			sort.Strings(sortedAdv)
			sort.Strings(sortedWant)

			if !equalStringSlices(sortedAdv, sortedWant) {
				t.Errorf("ACP priming shim note for role %q drift:\n  advertised: %v\n  allowlist:  %v\n\nFull note:\n%s",
					role, sortedAdv, sortedWant, note)
			}
		})
	}
}

// TestACPPrimingShimNote_NoParentheticalPlaceholders guards against the
// regression that prompted audit Fix 7 — the old shim note had
// "(status, log, etc.)" as a sample list. Any future revision that
// re-introduces an "etc." or other open-ended placeholder gets caught.
func TestACPPrimingShimNote_NoParentheticalPlaceholders(t *testing.T) {
	note := renderShimNote("planner")
	for _, banned := range []string{"etc.", "(status,", "such as"} {
		if strings.Contains(note, banned) {
			t.Errorf("shim note must enumerate the FULL allowed list, not a placeholder %q; got:\n%s", banned, note)
		}
	}
}

// TestACPPrimingComposition_MarkerAndHeader (Fix 8) — asserts the full
// composed priming string carries the InternalPromptMarker (so log
// scrapers can distinguish primer noise from real Planner output) and
// the leading "do not respond" header (so the LLM has an explicit
// handle on "this is config, not chat" since ACP has no system
// channel). Uses wrapPriming directly to avoid the prompt.GetAgentPrompt
// → config.WorkingDirectory panic that fires when no config is loaded
// in test contexts.
func TestACPPrimingComposition_MarkerAndHeader(t *testing.T) {
	const sampleBase = "You are the Planner.\n\n# act_cli — your ONLY shell-style tool\n..."
	got := wrapPriming("planner", sampleBase)

	if !strings.HasPrefix(got, InternalPromptMarker) {
		t.Errorf("priming must start with InternalPromptMarker for log-scraper filtering; got first 64 bytes:\n%q", got[:safeMin(64, len(got))])
	}
	afterMarker := strings.TrimPrefix(got, InternalPromptMarker)
	if !strings.HasPrefix(afterMarker, "[ACT priming — do not respond") {
		t.Errorf("expected do-not-respond header immediately after marker; got first 128 bytes after marker:\n%q",
			afterMarker[:safeMin(128, len(afterMarker))])
	}
	if !strings.Contains(got, sampleBase) {
		t.Errorf("expected role prompt base to remain present in composed priming")
	}
	if !strings.Contains(got, "act-tier1-planner") {
		t.Errorf("expected shim discoverability note to remain present; got tail:\n%q",
			got[safeMax(0, len(got)-200):])
	}
}

func safeMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func safeMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// extractAllowlistBullets parses lines of the shape `  - <entry>` from the
// shim note and returns the entries in order of appearance.
func extractAllowlistBullets(t *testing.T, note string) []string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^  - (.+)$`)
	matches := re.FindAllStringSubmatch(note, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, strings.TrimSpace(m[1]))
	}
	return out
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
