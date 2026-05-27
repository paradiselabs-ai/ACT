package prompt

import "sort"

// sectionRegistry maps Planner-on-demand section names to their content
// providers. Sourced once here so both the in-process
// expand_prompt_section tool AND the act-agent CLI's `prompt-section`
// subcommand (used by ACP-backed Planners via the shim binary) read from
// the same source — no drift possible between backends.
//
// Adding a new section is a one-line entry. Names must also be reflected
// in basePlannerPrompt's "Available sections" enumeration; the test in
// sections_test.go locks the prompt list to this registry.
var sectionRegistry = map[string]func() string{
	"evidence_routing": PlannerSectionEvidenceRouting,
	"success_criteria": PlannerSectionSuccessCriteria,
	"nomik":            PlannerSectionNomik,
	"validation":       PlannerSectionValidation,
	"examples":         PlannerSectionExamples,
}

// SectionRegistry returns a fresh copy of the section→provider map.
// Callers may iterate the copy without holding a lock and without
// mutating shared state.
func SectionRegistry() map[string]func() string {
	out := make(map[string]func() string, len(sectionRegistry))
	for k, v := range sectionRegistry {
		out[k] = v
	}
	return out
}

// SectionNames returns the sorted list of registered section names.
// Used for the LLM-facing enum and for drift-prevention tests.
func SectionNames() []string {
	out := make([]string, 0, len(sectionRegistry))
	for k := range sectionRegistry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// GetSection returns the content of the named section. Second return is
// false if the name is not registered. Callers must NOT mutate the
// returned string (it may share backing storage with subsequent calls).
func GetSection(name string) (string, bool) {
	provider, ok := sectionRegistry[name]
	if !ok {
		return "", false
	}
	return provider(), true
}
