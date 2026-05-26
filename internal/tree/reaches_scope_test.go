package tree

import (
	"context"
	"testing"
)

// scopeFixture builds a tree exercising every included scope edge and
// every excluded governance edge for ReachesScope (M-0141/AC-1, D-0006):
//
//	E-0001 ── M-0001 (AC-1)         included: parent, composite rollup
//	       └─ M-0002 depends_on M-0001    excluded edge: depends_on
//	E-0002 ── M-0003
//	G-0001 discovered_in M-0001, addressed_by M-0001/AC-1   included: discovered_in
//	G-0002 discovered_in M-0001, addressed_by M-0003        excluded edge: addressed_by
//	D-0001 relates_to E-0002                                 excluded edge: relates_to
//	ADR-0001 superseded_by ADR-0002                          excluded edge: superseded_by
//	ADR-0002 supersedes ADR-0001                             excluded edge: supersedes
//	C-0001 linked_adrs ADR-0001                              excluded edge: linked_adrs
//
// The excluded-edge fixtures are shaped so the governance edge is the
// ONLY path from source to target — there is no parent/discovered_in
// alternative — so a `false` result isolates the exclusion.
func scopeFixture(t *testing.T) *Tree {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "work/epics/E-0001-platform/epic.md", "---\nid: E-0001\ntitle: Platform\nstatus: active\n---\n")
	writeFile(t, root, "work/epics/E-0001-platform/M-0001-cache.md", "---\nid: M-0001\ntitle: Cache\nstatus: in_progress\nparent: E-0001\nacs:\n  - id: AC-1\n    title: warm\n    status: open\n---\n")
	writeFile(t, root, "work/epics/E-0001-platform/M-0002-evict.md", "---\nid: M-0002\ntitle: Evict\nstatus: draft\nparent: E-0001\ndepends_on:\n  - M-0001\n---\n")
	writeFile(t, root, "work/epics/E-0002-billing/epic.md", "---\nid: E-0002\ntitle: Billing\nstatus: active\n---\n")
	writeFile(t, root, "work/epics/E-0002-billing/M-0003-invoice.md", "---\nid: M-0003\ntitle: Invoice\nstatus: draft\nparent: E-0002\n---\n")
	writeFile(t, root, "work/gaps/G-0001-thrash.md", "---\nid: G-0001\ntitle: Thrash\nstatus: open\ndiscovered_in: M-0001\naddressed_by:\n  - M-0001/AC-1\n---\n")
	writeFile(t, root, "work/gaps/G-0002-leak.md", "---\nid: G-0002\ntitle: Leak\nstatus: open\ndiscovered_in: M-0001\naddressed_by:\n  - M-0003\n---\n")
	writeFile(t, root, "work/decisions/D-0001-pick.md", "---\nid: D-0001\ntitle: Pick\nstatus: accepted\nrelates_to:\n  - E-0002\n---\n")
	writeFile(t, root, "docs/adr/ADR-0001-old.md", "---\nid: ADR-0001\ntitle: Old\nstatus: superseded\nsuperseded_by: ADR-0002\n---\n")
	writeFile(t, root, "docs/adr/ADR-0002-new.md", "---\nid: ADR-0002\ntitle: New\nstatus: accepted\nsupersedes:\n  - ADR-0001\n---\n")
	writeFile(t, root, "work/contracts/C-0001-render.md", "---\nid: C-0001\ntitle: Render\nstatus: active\nlinked_adrs:\n  - ADR-0001\n---\n")

	tr, _, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return tr
}

// TestReachesScope is M-0141/AC-1: reachability traverses exactly
// D-0006's three edges — parent-forward, composite-id containment,
// discovered_in-reverse — and reaches a target through NO governance
// edge. Each `want true` case names an included edge; each `want false`
// case named "<edge> excluded" isolates one governance edge that must
// not traverse.
func TestReachesScope(t *testing.T) {
	t.Parallel()
	tr := scopeFixture(t)

	cases := []struct {
		name   string
		target string
		scope  string
		want   bool
	}{
		// Included edges (reachable).
		{"self-loop bare", "E-0001", "E-0001", true},
		{"self-loop composite", "M-0001/AC-1", "M-0001/AC-1", true},
		{"composite rollup to parent milestone", "M-0001/AC-1", "M-0001", true},
		{"parent forward: milestone to epic", "M-0001", "E-0001", true},
		{"parent forward + composite: AC to epic", "M-0001/AC-1", "E-0001", true},
		{"discovered_in reverse: gap to milestone", "G-0001", "M-0001", true},
		{"discovered_in then parent climb: gap to epic", "G-0001", "E-0001", true},

		// Excluded governance edges (not reachable).
		{"depends_on excluded", "M-0002", "M-0001", false},
		{"addressed_by excluded", "G-0002", "M-0003", false},
		{"relates_to excluded", "D-0001", "E-0002", false},
		{"supersedes excluded", "ADR-0002", "ADR-0001", false},
		{"superseded_by excluded", "ADR-0001", "ADR-0002", false},
		{"linked_adrs excluded", "C-0001", "ADR-0001", false},

		// Sanity: no edge, wrong direction, absent.
		{"cross-epic no edge", "M-0003", "E-0001", false},
		{"backwards epic to milestone", "E-0001", "M-0001", false},
		{"target absent from tree", "X-0099", "E-0001", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tr.ReachesScope(tc.target, tc.scope); got != tc.want {
				t.Errorf("ReachesScope(%q, %q) = %v, want %v", tc.target, tc.scope, got, tc.want)
			}
		})
	}
}

// TestReachesScopeAny is the multi-target creation-act variant: at least
// one proposed outbound reference must reach the scope-entity.
func TestReachesScopeAny(t *testing.T) {
	t.Parallel()
	tr := scopeFixture(t)

	if !tr.ReachesScopeAny([]string{"M-0001", "X-0099"}, "E-0001") {
		t.Error("ReachesScopeAny([M-0001 X-0099], E-0001) = false; want true (M-0001 reaches E-0001 via parent)")
	}
	if tr.ReachesScopeAny([]string{"M-0002"}, "M-0001") {
		t.Error("ReachesScopeAny([M-0002], M-0001) = true; want false (depends_on excluded)")
	}
	if tr.ReachesScopeAny(nil, "E-0001") {
		t.Error("ReachesScopeAny(nil, E-0001) = true; want false (no targets)")
	}
}

// TestReachesScope_ParentCycleTerminates exercises the parent-climb
// visited guard: a malformed tree with a parent cycle (the loader
// tolerates invalid parent kinds per errors-are-findings) must not hang
// and must return false for an unreachable scope.
func TestReachesScope_ParentCycleTerminates(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, root, "work/epics/E-0009-x/epic.md", "---\nid: E-0009\ntitle: X\nstatus: active\n---\n")
	// M-0901.parent → M-0902, M-0902.parent → M-0901 (invalid kinds,
	// loader-tolerated).
	writeFile(t, root, "work/epics/E-0009-x/M-0901-a.md", "---\nid: M-0901\ntitle: A\nstatus: draft\nparent: M-0902\n---\n")
	writeFile(t, root, "work/epics/E-0009-x/M-0902-b.md", "---\nid: M-0902\ntitle: B\nstatus: draft\nparent: M-0901\n---\n")
	tr, _, err := Load(context.Background(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if tr.ReachesScope("M-0901", "E-0009") {
		t.Error("ReachesScope(M-0901, E-0009) = true; want false (cycle does not reach E-0009)")
	}
}
