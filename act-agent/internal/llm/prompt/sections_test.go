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
