package tools

import (
	"sort"
	"testing"
)

func TestIsAllowed_Bare(t *testing.T) {
	cases := []struct {
		name       string
		role       string
		subcommand string
		args       []string
		want       bool
	}{
		{"planner status", "planner", "status", nil, true},
		{"planner ls", "planner", "ls", nil, false},
		{"observer pvm", "observer", "pvm", nil, false}, // pvm not on observer list
		{"observer status", "observer", "status", nil, true},
		{"unknown role", "captain", "status", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsAllowed(tc.role, tc.subcommand, tc.args...)
			if got != tc.want {
				t.Fatalf("IsAllowed(%q,%q,%v) = %v, want %v",
					tc.role, tc.subcommand, tc.args, got, tc.want)
			}
		})
	}
}

func TestIsAllowed_Compound(t *testing.T) {
	cases := []struct {
		name       string
		role       string
		subcommand string
		args       []string
		want       bool
	}{
		{"planner task retry", "planner", "task", []string{"retry", "id-1"}, true},
		{"planner task abandon", "planner", "task", []string{"abandon", "id-1", "--reason", "dup"}, true},
		// task complete/submit-for-validation/progress are swarm-only.
		{"planner task complete forbidden", "planner", "task", []string{"complete", "id-1"}, false},
		{"planner task submit forbidden", "planner", "task", []string{"submit-for-validation", "id-1"}, false},
		{"planner task progress forbidden", "planner", "task", []string{"progress", "id-1"}, false},
		// bare "task" with no args must NOT match a compound entry.
		{"planner bare task no args", "planner", "task", nil, false},
		// "task retry" cannot be reached without args[0]=="retry"
		{"planner task wrong arg0", "planner", "task", []string{"--reason", "x"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsAllowed(tc.role, tc.subcommand, tc.args...)
			if got != tc.want {
				t.Fatalf("IsAllowed(%q,%q,%v) = %v, want %v",
					tc.role, tc.subcommand, tc.args, got, tc.want)
			}
		})
	}
}

func TestAllowedSubcommandHeads_DedupesCompoundHeads(t *testing.T) {
	heads := AllowedSubcommandHeads("planner")
	got := append([]string(nil), heads...)
	sort.Strings(got)
	want := []string{
		"context", "graph", "log", "message", "prompt-section", "pvm", "status", "task",
	}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("heads length mismatch: got %v want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("heads[%d] = %q, want %q (full got=%v)", i, got[i], want[i], got)
		}
	}
}
