package spil

import (
	"strings"
	"testing"
)

func TestLoadAllRoles(t *testing.T) {
	required := []string{
		// Tier 1
		"planner", "observer", "assurance", "qa_synthesizer",
		// Tier 2
		"developer", "frontend_dev", "backend_dev", "qa_engineer", "researcher",
	}
	for _, name := range required {
		t.Run(name, func(t *testing.T) {
			f, ok := Get(name)
			if !ok {
				t.Fatalf("spil file %q not found in embedded prompts", name)
			}
			if f.Body == "" {
				t.Fatalf("spil file %q has empty body", name)
			}
			if len(f.Order) == 0 {
				t.Fatalf("spil file %q has no @sections", name)
			}
			if !strings.Contains(f.Body, "@") {
				t.Fatalf("spil file %q has no @keyword markers in body", name)
			}
			// Sanity: identity-like section in every role
			hasIdent := false
			for _, s := range f.Order {
				if s == "identity" || s == "specialization" || s == "scope_check" || s == "how_you_receive_work" {
					hasIdent = true
					break
				}
			}
			if !hasIdent {
				t.Errorf("spil file %q missing identity/specialization/scope_check/how_you_receive_work section; got %v", name, f.Order)
			}
		})
	}
}

func TestManifestFormat(t *testing.T) {
	f, ok := Get("planner")
	if !ok {
		t.Fatal("planner not loaded")
	}
	m := f.Manifest()
	if m == "" {
		t.Fatal("planner manifest empty")
	}
	for _, section := range f.Order {
		if !strings.Contains(m, "@"+section) {
			t.Errorf("manifest missing @%s line: %s", section, m)
		}
	}
}

func TestSectionFetch(t *testing.T) {
	f := MustGet("assurance")
	body, ok := f.Section("two_layer_validation")
	if !ok {
		t.Fatalf("assurance.two_layer_validation not found; available: %v", f.Order)
	}
	if !strings.Contains(body, "Ralph Wiggum") {
		t.Errorf("two_layer_validation should mention Ralph Wiggum (Layer 1); got: %s", body)
	}
}
