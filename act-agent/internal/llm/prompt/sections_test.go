package prompt

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestPromptSectionAdvertisementMatchesRegistry locks the "Available
// sections:" list in basePlannerPrompt to the section registry. If
// someone adds a section to one but not the other, this test fails and
// catches the drift before the lie ships to the Planner.
//
// The advertisement format is: lines under the "Available sections:"
// header of the form `- "<name>" — <description>`. Anything quoted on
// such a line is treated as an advertised section.
func TestPromptSectionAdvertisementMatchesRegistry(t *testing.T) {
	advertised := extractAdvertisedSectionNames(t, basePlannerPrompt)
	registered := SectionNames()

	if len(advertised) == 0 {
		t.Fatalf("no advertised sections found — did the 'Available sections:' block move or change format?")
	}

	sort.Strings(advertised)
	// SectionNames() already returns sorted.

	if !equalStrings(advertised, registered) {
		t.Errorf("prompt-advertised sections (%v) drift from sectionRegistry (%v). "+
			"Either add the missing entry to sectionRegistry (sections.go) or remove it from "+
			"basePlannerPrompt's 'Available sections' list (planner.go).",
			advertised, registered)
	}
}

// TestGetSection_KnownAndUnknown sanity-checks the lookup behavior the
// ACP CLI subcommand and the in-process tool both rely on.
func TestGetSection_KnownAndUnknown(t *testing.T) {
	for _, name := range SectionNames() {
		content, ok := GetSection(name)
		if !ok {
			t.Errorf("GetSection(%q) returned ok=false for a registered name", name)
		}
		if content == "" {
			t.Errorf("GetSection(%q) returned empty content — section providers must return non-empty text", name)
		}
	}

	if _, ok := GetSection("nonexistent-section-XYZ"); ok {
		t.Errorf("GetSection on unknown name returned ok=true; expected false")
	}
}

// extractAdvertisedSectionNames pulls the quoted names from the
// "Available sections:" block of the prompt. Stops at the first blank
// line after the block begins (matches the prompt's actual layout).
func extractAdvertisedSectionNames(t *testing.T, src string) []string {
	t.Helper()
	headerIdx := strings.Index(src, "Available sections:")
	if headerIdx < 0 {
		t.Fatalf("could not find 'Available sections:' header in basePlannerPrompt")
	}
	body := src[headerIdx:]

	// Match `- "<name>"` (the section enumeration shape).
	re := regexp.MustCompile(`(?m)^- "([a-z_]+)"`)
	matches := re.FindAllStringSubmatch(body, -1)

	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

// TestNoSectionEmitsForbiddenDependenciesShape asserts that no registered
// section body contains the forbidden inline @dependencies SPIL section form.
// The base planner prompt (planner.go:78) explicitly prohibits putting
// @dependencies inside the description string — dependencies belong in the
// top-level JSON "dependencies" array. This test walks every section in
// sectionRegistry so the lint catches future drift in any section, not just
// "examples".
//
// The forbidden pattern is `@dependencies\n` (the SPIL section header followed
// by a newline). A bare occurrence of the word "dependencies" or the JSON key
// "dependencies" is fine — only the @-section header form is prohibited.
func TestNoSectionEmitsForbiddenDependenciesShape(t *testing.T) {
	for name, provider := range SectionRegistry() {
		content := provider()
		if strings.Contains(content, "@dependencies\n") {
			t.Errorf("section %q contains the forbidden @dependencies inline-section form "+
				"(the literal string '@dependencies\\n'). "+
				"The Planner's base prompt forbids @dependencies inside the description string — "+
				"dependencies belong in the top-level JSON \"dependencies\" array. "+
				"This example is actively teaching the wrong shape and contradicts planner.go:78.",
				name)
		}
	}
}

func equalStrings(a, b []string) bool {
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

// TestActCLICommandsFragmentMatchesAllowlist asserts the hand-written Planner
// fragment in actCLICommands("planner") (common.go) enumerates every
// subcommand that RoleSubcommands["planner"] in tools/act_cli_whitelist.go
// allows. The drift risk: AllowedFor("planner") is the enforcement gate —
// if the fragment silently omits an allowed subcommand, in-process Planners
// never learn the capability exists, even though ACP Planners see it via the
// renderShimNote auto-generation.
//
// Note on circular import: tools/expand_prompt_section.go imports this prompt
// package, so prompt cannot import tools without a cycle. wantBareHeads
// hard-codes the AllowedFor("planner") bare subcommands with a pointer to the
// authoritative source. Update both when adding new Planner subcommands.
//
// "message" is in AllowedFor("planner") but intentionally omitted from the
// fragment because the in-process Planner speaks via reply text and message
// is only relevant for ACP-backed Planners (who see it via renderShimNote).
// It is excluded from wantBareHeads to avoid a false failure on that
// intentional omission.
func TestActCLICommandsFragmentMatchesAllowlist(t *testing.T) {
	// Mirrors RoleSubcommands["planner"] bare entries (tools/act_cli_whitelist.go).
	wantBareHeads := []string{
		"status", "context", "log", "graph", "pvm", "prompt-section",
	}
	wantCompounds := []string{"task retry", "task abandon"}

	fragment := actCLICommands("planner")
	for _, head := range wantBareHeads {
		if !strings.Contains(fragment, "act-agent "+head) {
			t.Errorf("actCLICommands(\"planner\") missing subcommand head %q — "+
				"add it to the fragment in common.go and verify it is in "+
				"RoleSubcommands[\"planner\"] in tools/act_cli_whitelist.go", head)
		}
	}
	for _, compound := range wantCompounds {
		if !strings.Contains(fragment, "act-agent "+compound) {
			t.Errorf("actCLICommands(\"planner\") missing compound form %q — "+
				"add it to the fragment in common.go and verify the compound entry "+
				"exists in RoleSubcommands[\"planner\"] in tools/act_cli_whitelist.go", compound)
		}
	}
}

// TestBasePlannerPromptNoFragmentDuplication locks the trim that audit
// Fix 9 (entries 2.2 + 2.3) performed: basePlannerPrompt must not
// re-enumerate the act_cli allowed subcommands (that lives in the
// shared actCLICommands fragment) and must not restate the
// "Reacting to other roles" decision matrix (that lives in the shared
// coordinationConstraints fragment). The fragments are shared across
// 9 roles; the duplications were Planner-only and inflated the
// per-turn token cost while giving smaller models two slightly-
// different framings to anchor to.
func TestBasePlannerPromptNoFragmentDuplication(t *testing.T) {
	bannedSubstrings := []string{
		// The literal enumeration that used to live at planner.go:103.
		// The fragment side renders the same content with descriptions.
		"Allowed subcommands: status, context, log, graph, pvm, message, codebase, task",
		// Two arrow-headed lines from the deleted "Reacting to other
		// roles" block. The fragment's "Constraints" NEVER-list covers
		// the same ground more crisply.
		"Observer reports →",
		"QA reports SYNTHESIS_COMPLETE →",
	}
	for _, banned := range bannedSubstrings {
		if strings.Contains(basePlannerPrompt, banned) {
			t.Errorf("basePlannerPrompt must not contain %q — duplicates a shared fragment. "+
				"Re-introducing this content reverts audit Fix 9 (entries 2.2 + 2.3) and "+
				"costs ~50-200 tokens per Planner turn.", banned)
		}
	}
}
