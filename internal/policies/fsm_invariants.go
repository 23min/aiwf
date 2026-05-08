package policies

import (
	"fmt"
	"sort"

	"github.com/23min/ai-workflow-v2/internal/entity"
)

// PolicyFSMInvariants asserts kernel-level invariants over the entity-
// kind status FSMs. Filed under G44 item 2's drift-prevention follow-up.
//
// Why this is a policy and not a test in internal/entity:
//
// The invariants below are *kernel commitments*, not entity-package
// implementation details. Co-located tests in entity/ work for state-
// set drift (G44 item 2), but two drift modes weren't caught there:
//
//  1. Iteration source = test target. The G44 item 2 tests iterate
//     `transitions` (the unexported FSM map). A new entity Kind added
//     without an entry in `transitions` is *invisible* to that loop:
//     the kind has no FSM at all and no test fails. This policy
//     iterates `entity.AllKinds()` — the canonical Kind enum — and
//     asserts wiring exists for each. New Kind ⇒ new violation.
//
//  2. The kernel commitment "FSM is one-directional — no demote"
//     (kernel commitment 1, design-decisions.md) lives in prose only.
//     A future contributor adding a transition that closes a cycle
//     (e.g., `cancelled → active` to resurrect a cancelled epic) would
//     not trip any G44 item 2 test: the state set is unchanged, the
//     transition is a regular entry. This policy detects cycles via
//     DFS on the FSM graph and reports any back-edge.
//
// The policy uses only entity's exported API (AllKinds, AllowedStatuses,
// AllowedTransitions, CancelTarget) so the dependency direction stays
// clean. The rootDir argument is unused — this policy is a runtime-
// introspection policy, not a source-scan policy. The framework's
// signature is preserved for consistency with policies_test.go.
func PolicyFSMInvariants(_ string) ([]Violation, error) {
	var out []Violation

	for _, kind := range entity.AllKinds() {
		statuses := entity.AllowedStatuses(kind)

		// Drift mode 1: kind has no AllowedStatuses entry.
		if len(statuses) == 0 {
			out = append(out, Violation{
				Policy: "fsm-invariants",
				Detail: fmt.Sprintf("kind %q is in AllKinds() but has no AllowedStatuses; FSM unwired", kind),
			})
			continue
		}

		// Drift mode 2: kind is unwired (no transitions at all). Every
		// kind needs at least one non-terminal status with outgoing
		// transitions, otherwise the entity has no lifecycle. This
		// catches "Kind constant added to AllKinds() without an entry
		// in the FSM data" (AllowedTransitions returns nil for an
		// unknown kind, indistinguishable from "all states terminal"
		// at the public API — but both are bugs).
		anyWired := false
		for _, from := range statuses {
			if len(entity.AllowedTransitions(kind, from)) > 0 {
				anyWired = true
				break
			}
		}
		if !anyWired {
			out = append(out, Violation{
				Policy: "fsm-invariants",
				Detail: fmt.Sprintf("kind %q has no non-terminal statuses; FSM is unwired or every state is terminal", kind),
			})
		}

		// Drift mode 2b: every transition target must itself be in
		// the kind's AllowedStatuses. Catches "FSM transitions to a
		// status the schemas table doesn't know about."
		declared := make(map[string]struct{}, len(statuses))
		for _, s := range statuses {
			declared[s] = struct{}{}
		}
		for _, from := range statuses {
			for _, to := range entity.AllowedTransitions(kind, from) {
				if _, ok := declared[to]; !ok {
					out = append(out, Violation{
						Policy: "fsm-invariants",
						Detail: fmt.Sprintf("kind %q: transition %q → %q targets a status not in AllowedStatuses", kind, from, to),
					})
				}
			}
		}

		// Drift mode 3: CancelTarget returns a status that is not in
		// the closed set or is non-terminal. This pins the cancel
		// verb's commitment that "any non-terminal entity can be
		// cancelled to a terminal state in one step."
		target := entity.CancelTarget(kind)
		if target == "" {
			out = append(out, Violation{
				Policy: "fsm-invariants",
				Detail: fmt.Sprintf("kind %q: CancelTarget returns empty; cancel verb has no terminal to project to", kind),
			})
		} else if !entity.IsAllowedStatus(kind, target) {
			out = append(out, Violation{
				Policy: "fsm-invariants",
				Detail: fmt.Sprintf("kind %q: CancelTarget %q not in AllowedStatuses", kind, target),
			})
		} else if outs := entity.AllowedTransitions(kind, target); len(outs) != 0 {
			out = append(out, Violation{
				Policy: "fsm-invariants",
				Detail: fmt.Sprintf("kind %q: CancelTarget %q has outgoing transitions %v; must be terminal", kind, target, outs),
			})
		}

		// Drift mode 4: FSM contains a cycle. Kernel commitment 1
		// declares the FSM one-directional ("there is no demote").
		// Any transition that closes a cycle (e.g., cancelled → active)
		// silently violates that. DFS with three-color marking
		// detects back-edges in O(V+E). Because the FSMs are tiny
		// (≤5 states per kind), the deterministic ordering helps
		// reproducibility — sort sources before walking.
		sortedSources := make([]string, len(statuses))
		copy(sortedSources, statuses)
		sort.Strings(sortedSources)

		// One cycle is enough to violate the commitment. Stop at the
		// first one found per kind to keep the violation message
		// readable; finding all cycles is not the point.
		if cycle := findCycle(sortedSources, func(from string) []string {
			return entity.AllowedTransitions(kind, from)
		}); cycle != nil {
			out = append(out, Violation{
				Policy: "fsm-invariants",
				Detail: fmt.Sprintf("kind %q: FSM contains a cycle: %v", kind, cycle),
			})
		}
	}

	// AC and TDD-phase composite FSMs: same DAG check, exposed via
	// IsLegalACTransition / IsLegalTDDPhaseTransition. We probe the
	// FSM by querying every (from, to) pair against the closed sets.
	for _, fsm := range []struct {
		name        string
		statuses    []string
		entryStates []string
		isLegal     func(from, to string) bool
	}{
		{"ac-status", entity.AllowedACStatuses(), []string{"open"}, entity.IsLegalACTransition},
		{"tdd-phase", entity.AllowedTDDPhases(), []string{"", "red"}, entity.IsLegalTDDPhaseTransition},
	} {
		out = append(out, fsmDAGViolations(fsm.name, fsm.statuses, fsm.entryStates, fsm.isLegal)...)
	}

	return out, nil
}

// fsmDAGViolations runs cycle detection on a composite FSM (one whose
// transitions are exposed via a single isLegal predicate rather than
// an AllowedTransitions(kind, from) probe). Returns one Violation if
// a cycle is found, none otherwise.
func fsmDAGViolations(name string, statuses, entryStates []string, isLegal func(from, to string) bool) []Violation {
	allFroms := append([]string{}, entryStates...)
	allFroms = append(allFroms, statuses...)
	sort.Strings(allFroms)

	successors := func(from string) []string {
		var out []string
		for _, to := range statuses {
			if isLegal(from, to) {
				out = append(out, to)
			}
		}
		sort.Strings(out)
		return out
	}

	if cycle := findCycle(allFroms, successors); cycle != nil {
		return []Violation{{
			Policy: "fsm-invariants",
			Detail: fmt.Sprintf("composite FSM %q contains a cycle: %v", name, cycle),
		}}
	}
	return nil
}

// findCycle runs three-color DFS over a directed graph and returns the
// first detected cycle's vertex sequence (with the closing vertex
// repeated at the end), or nil if the graph is acyclic. Vertices are
// visited in the order given. successors(v) returns v's outgoing
// neighbors.
//
// Single-cycle reporting is deliberate: the FSM-invariants policy
// commits to "FSM is a DAG"; one back-edge is enough to violate the
// commitment, and finding every cycle would just clutter the
// violation list. Once the policy fires the contributor fixes the
// FSM and reruns; the next cycle (if any) appears on the next pass.
func findCycle(vertices []string, successors func(string) []string) []string {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int, len(vertices))
	var found []string

	var visit func(s string, path []string) bool
	visit = func(s string, path []string) bool {
		color[s] = gray
		path = append(path, s)
		for _, to := range successors(s) {
			switch color[to] {
			case gray:
				found = append(append([]string{}, path...), to)
				return true
			case white:
				if visit(to, path) {
					return true
				}
			}
		}
		color[s] = black
		return false
	}

	for _, s := range vertices {
		if color[s] != white {
			continue
		}
		if visit(s, nil) {
			return found
		}
	}
	return nil
}
