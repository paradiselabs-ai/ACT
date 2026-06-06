package app

import (
	"testing"
	"time"
)

// agedPendingTask builds a pending task created `min` minutes ago.
func agedPendingTask(id string, caps []string, deps []string, min int) TaskSummary {
	return TaskSummary{
		ID:                   id,
		Title:                id,
		Status:               "pending",
		RequiredCapabilities: caps,
		Dependencies:         deps,
		CreatedAt:            time.Now().Add(-time.Duration(min) * time.Minute).UTC().Format(time.RFC3339),
	}
}

func countCategory(anoms []Anomaly, cat AnomalyCategory) int {
	n := 0
	for _, a := range anoms {
		if a.Category == cat {
			n++
		}
	}
	return n
}

func TestDetectAnomalies_OrphanedPending(t *testing.T) {
	freshAgent := func(id string, status string, caps []string) AgentSummary {
		return AgentSummary{ID: id, Status: status, Capabilities: caps,
			LastSeen: time.Now().UTC().Format(time.RFC3339)}
	}

	cases := []struct {
		name     string
		snap     *StatusSnapshot
		wantCat  AnomalyCategory
		wantSome bool // expect at least one of wantCat
	}{
		{
			name: "no capable agent at all → unservable",
			snap: &StatusSnapshot{
				Tasks:  []TaskSummary{agedPendingTask("t1", []string{"go"}, nil, 5)},
				Agents: []AgentSummary{freshAgent("fe-1", "online", []string{"typescript"})},
			},
			wantCat: CategoryUnservableTask, wantSome: true,
		},
		{
			name: "only capable agent offline → unresponsive",
			snap: &StatusSnapshot{
				Tasks:  []TaskSummary{agedPendingTask("t1", []string{"go"}, nil, 5)},
				Agents: []AgentSummary{freshAgent("dev-1", "offline", []string{"go"})},
			},
			wantCat: CategoryUnresponsiveAgent, wantSome: true,
		},
		{
			name: "capable agent online → assignment wedged",
			snap: &StatusSnapshot{
				Tasks:  []TaskSummary{agedPendingTask("t1", []string{"go"}, nil, 5)},
				Agents: []AgentSummary{freshAgent("dev-1", "online", []string{"go"})},
			},
			wantCat: CategoryAssignmentWedged, wantSome: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detectAnomalies(c.snap)
			if c.wantSome && countCategory(got, c.wantCat) == 0 {
				t.Errorf("expected a %s anomaly; got %+v", c.wantCat, got)
			}
		})
	}
}

func TestDetectAnomalies_PendingWithUnsatisfiedDep_NotFlagged(t *testing.T) {
	// t2 depends on t1 (still pending) → t2 is correctly waiting, must NOT be flagged.
	snap := &StatusSnapshot{
		Tasks: []TaskSummary{
			agedPendingTask("t1", []string{"go"}, nil, 5),
			agedPendingTask("t2", []string{"go"}, []string{"t1"}, 5),
		},
		Agents: []AgentSummary{{ID: "dev-1", Status: "online", Capabilities: []string{"go"},
			LastSeen: time.Now().UTC().Format(time.RFC3339)}},
	}
	got := detectAnomalies(snap)
	for _, a := range got {
		if a.TaskID == "t2" {
			t.Errorf("t2 has an unsatisfied dependency and must not be flagged; got %+v", a)
		}
	}
}

func TestDetectAnomalies_FreshPending_NoNewAnomaly(t *testing.T) {
	// pending for <2min should not trip the NEW pending-pipeline checks. (The
	// older idle-agent rule may still fire — that's pre-existing behavior.)
	snap := &StatusSnapshot{
		Tasks:  []TaskSummary{agedPendingTask("t1", []string{"go"}, nil, 1)},
		Agents: []AgentSummary{{ID: "fe-1", Status: "online", Capabilities: []string{"typescript"}}},
	}
	got := detectAnomalies(snap)
	for _, cat := range []AnomalyCategory{CategoryUnservableTask, CategoryUnresponsiveAgent, CategoryAssignmentWedged} {
		if countCategory(got, cat) != 0 {
			t.Errorf("fresh pending task should not trip %s; got %+v", cat, got)
		}
	}
}

func TestDetectAnomalies_HealthyInProgress_NoNewAnomaly(t *testing.T) {
	// A recently-created in_progress task must not trip any rule (no regression).
	snap := &StatusSnapshot{
		Tasks: []TaskSummary{{
			ID: "t1", Title: "t1", Status: "in_progress", AssignedAgent: "dev-1",
			CreatedAt: time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339),
		}},
		Agents: []AgentSummary{{ID: "dev-1", Status: "busy", CurrentTask: "t1"}},
	}
	if got := detectAnomalies(snap); len(got) != 0 {
		t.Errorf("healthy in-progress task should produce no anomaly; got %+v", got)
	}
}
