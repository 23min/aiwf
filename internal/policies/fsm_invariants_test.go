package policies

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/entity"
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

// TestKindFSMViolations_MessagesStateTheFix pins M-0337 AC-2: each way a
// status machine can be miswired is reported with the change that fixes
// it. The kernel's own tables are wired correctly, so the fixtures are
// synthetic machines.
func TestKindFSMViolations_MessagesStateTheFix(t *testing.T) {
	t.Parallel()
	const a, b, c, z = entity.Status("a"), entity.Status("b"), entity.Status("c"), entity.Status("z")
	view := func(statuses []entity.Status, edges map[entity.Status][]entity.Status, cancel map[entity.Status]entity.Status) fsmView {
		return fsmView{
			kind:         "widget",
			statuses:     statuses,
			transitions:  func(from entity.Status) []entity.Status { return edges[from] },
			isTerminal:   func(s entity.Status) bool { return len(edges[s]) == 0 },
			isAllowed:    func(s entity.Status) bool { return slices.Contains(statuses, s) },
			cancelTarget: func(from entity.Status) entity.Status { return cancel[from] },
		}
	}
	cases := []struct {
		name string
		v    fsmView
		want string
	}{
		{"no statuses", view(nil, nil, nil), "add the kind's statuses to the schemas table"},
		{"no transitions", view([]entity.Status{a}, nil, nil), "give the kind's statuses outgoing transitions"},
		{"undeclared target", view([]entity.Status{a}, map[entity.Status][]entity.Status{a: {b}}, nil), "or retarget the transition"},
		{"undeclared cancel target", view([]entity.Status{a, b}, map[entity.Status][]entity.Status{a: {b}}, map[entity.Status]entity.Status{a: z}), "return one of the kind's statuses from CancelTarget"},
		{"non-terminal cancel target", view([]entity.Status{a, b, c}, map[entity.Status][]entity.Status{a: {b}, b: {c}}, map[entity.Status]entity.Status{a: b}), "return a terminal status from CancelTarget"},
		{"cycle", view([]entity.Status{a, b}, map[entity.Status][]entity.Status{a: {b}, b: {a}}, nil), "remove the transition that closes it"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mustHaveViolation(t, kindFSMViolations(tc.v), tc.want)
		})
	}
	t.Run("a terminal status's cancel target is not judged", func(t *testing.T) {
		t.Parallel()
		vs := kindFSMViolations(view([]entity.Status{a, b}, map[entity.Status][]entity.Status{a: {b}}, map[entity.Status]entity.Status{b: z}))
		for _, v := range vs {
			if strings.Contains(v.Detail, "CancelTarget") {
				t.Errorf("violation %q judges the cancel target of terminal status %q", v.Detail, b)
			}
		}
	})
	t.Run("no statuses ends the kind's checks", func(t *testing.T) {
		t.Parallel()
		if vs := kindFSMViolations(view(nil, nil, nil)); len(vs) != 1 {
			t.Errorf("violations = %+v, want only the missing-statuses one", vs)
		}
	})
	t.Run("composite cycle", func(t *testing.T) {
		t.Parallel()
		vs := fsmDAGViolations("widget-phase", []string{"x", "y"}, nil, func(from, to string) bool { return from != to })
		mustHaveViolation(t, vs, "remove the transition that closes it")
	})
}

// TestKernelFSMView_ReadsTheEntityTables pins the seam between
// PolicyFSMInvariants and the kernel (M-0337 AC-2): the view it checks for
// each kind answers every lookup as entity's exported API does, so the
// checks TestKindFSMViolations_MessagesStateTheFix drives over synthetic
// machines run over the real tables.
func TestKernelFSMView_ReadsTheEntityTables(t *testing.T) {
	t.Parallel()
	for _, kind := range entity.AllKinds() {
		v := kernelFSMView(kind)
		if v.kind != kind || !slices.Equal(v.statuses, entity.AllowedStatuses(kind)) {
			t.Errorf("%s: view kind %q statuses %v, want %q %v", kind, v.kind, v.statuses, kind, entity.AllowedStatuses(kind))
		}
		for _, s := range append(slices.Clone(v.statuses), "not-a-status") {
			if got, want := v.transitions(s), entity.AllowedTransitions(kind, s); !slices.Equal(got, want) {
				t.Errorf("%s at %q: transitions %v, want %v", kind, s, got, want)
			}
			if got, want := v.isTerminal(s), entity.IsTerminal(kind, s); got != want {
				t.Errorf("%s at %q: isTerminal %v, want %v", kind, s, got, want)
			}
			if got, want := v.isAllowed(s), entity.IsAllowedStatus(kind, s); got != want {
				t.Errorf("%s at %q: isAllowed %v, want %v", kind, s, got, want)
			}
			if got, want := v.cancelTarget(s), entity.CancelTarget(kind, s); got != want {
				t.Errorf("%s at %q: cancelTarget %q, want %q", kind, s, got, want)
			}
		}
	}
}
