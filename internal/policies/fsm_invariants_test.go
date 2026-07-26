package policies

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestFindCycle_DetectsBackEdge is the direct unit test for the
// generic DFS cycle detector PolicyFSMInvariants/fsmDAGViolations rely
// on to catch "FSM demotes into a cycle" (kernel commitment: statuses
// only move forward). No real entity Kind's FSM is actually cyclic —
// that is the whole point of the policy passing — so the back-edge
// branch (`case gray` when a successor is already on the current DFS
// path) never fires against production data. A synthetic graph is the
// only way to exercise it.
func TestFindCycle_DetectsBackEdge(t *testing.T) {
	t.Parallel()
	graph := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	}
	successors := func(v string) []string { return graph[v] }

	got := findCycle([]string{"a", "b", "c"}, successors)
	if got == nil {
		t.Fatal("findCycle returned nil, want a detected cycle")
	}
	// DFS starts at "a", walks a -> b -> c, then finds "a" already gray
	// (still on the current path) while probing c's successors — the
	// back-edge that proves a cycle exists.
	want := []string{"a", "b", "c", "a"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("findCycle cycle mismatch (-want +got):\n%s", diff)
	}
}

// TestFindCycle_AcyclicReturnsNil is the negative counterpart: a DAG
// with no back-edges returns nil, the case every real entity-kind FSM
// hits today.
func TestFindCycle_AcyclicReturnsNil(t *testing.T) {
	t.Parallel()
	graph := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": nil,
	}
	successors := func(v string) []string { return graph[v] }

	if got := findCycle([]string{"a", "b", "c"}, successors); got != nil {
		t.Errorf("findCycle = %v, want nil for an acyclic graph", got)
	}
}
