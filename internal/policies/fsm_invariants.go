package policies

import (
	"fmt"
	"slices"

	"github.com/23min/aiwf/internal/entity"
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
//  2. The commitment "FSM is one-directional — no demote" lives in
//     prose only: transition.go's own doc comment, and R-AUDIT-0028 /
//     R-RULE-019 in docs/design/legal-workflows-audit.md.
//     A future contributor adding a transition that closes a cycle
//     (e.g., `cancelled → active` to resurrect a cancelled epic) would
//     not trip any G44 item 2 test: the state set is unchanged, the
//     transition is a regular entry. This policy detects cycles via
//     DFS on the FSM graph and reports any back-edge.
//
// The policy uses only entity's exported API, through kernelFSMView and
// the composite-FSM predicates, so the dependency direction stays clean. The rootDir argument is unused — this policy is a runtime-
// introspection policy, not a source-scan policy. The framework's
// signature is preserved for consistency with policies_test.go.
func PolicyFSMInvariants(_ string) ([]Violation, error) {
	var out []Violation

	for _, kind := range entity.AllKinds() {
		out = append(out, kindFSMViolations(kernelFSMView(kind))...)
	}

	// AC and TDD-phase composite FSMs: same DAG check, exposed via
	// IsLegalACTransition / IsLegalTDDPhaseTransition. We probe the
	// FSM by querying every (from, to) pair against the closed sets.
	// ac-status is keyed by entity.Status and tdd-phase by plain string,
	// so each instantiates fsmDAGViolations at its own element type.
	out = append(out, fsmDAGViolations("ac-status", entity.AllowedACStatuses(), []entity.Status{entity.StatusOpen}, entity.IsLegalACTransition)...)
	out = append(out, fsmDAGViolations("tdd-phase", entity.AllowedTDDPhases(), []string{"", entity.TDDPhaseRed}, entity.IsLegalTDDPhaseTransition)...)

	return out, nil
}

// fsmView is one kind's status machine as PolicyFSMInvariants reads it
// through entity's exported API, so the checks run the same over the
// kernel's tables and over a test's synthetic machine.
type fsmView struct {
	kind         entity.Kind
	statuses     []entity.Status
	transitions  func(from entity.Status) []entity.Status
	isTerminal   func(s entity.Status) bool
	isAllowed    func(s entity.Status) bool
	cancelTarget func(from entity.Status) entity.Status
}

// kernelFSMView reads one kind's status machine from entity's tables.
func kernelFSMView(kind entity.Kind) fsmView {
	return fsmView{
		kind:         kind,
		statuses:     entity.AllowedStatuses(kind),
		transitions:  func(from entity.Status) []entity.Status { return entity.AllowedTransitions(kind, from) },
		isTerminal:   func(s entity.Status) bool { return entity.IsTerminal(kind, s) },
		isAllowed:    func(s entity.Status) bool { return entity.IsAllowedStatus(kind, s) },
		cancelTarget: func(from entity.Status) entity.Status { return entity.CancelTarget(kind, from) },
	}
}

// kindFSMViolations checks one kind's status machine for the invariants
// PolicyFSMInvariants documents.
func kindFSMViolations(v fsmView) []Violation {
	var out []Violation

	// Drift mode 1: kind has no AllowedStatuses entry.
	if len(v.statuses) == 0 {
		out = append(out, Violation{
			Policy: "fsm-invariants",
			Detail: fmt.Sprintf("kind %q is in AllKinds() but has no AllowedStatuses; FSM unwired — add the kind's statuses to the schemas table in internal/entity/entity.go", v.kind),
		})
		return out
	}

	// Drift mode 2: kind is unwired (no transitions at all). Every
	// kind needs at least one non-terminal status with outgoing
	// transitions, otherwise the entity has no lifecycle. This
	// catches "Kind constant added to AllKinds() without an entry
	// in the FSM data" (AllowedTransitions returns nil for an
	// unknown kind, indistinguishable from "all states terminal"
	// at the public API — but both are bugs).
	anyWired := false
	for _, from := range v.statuses {
		if len(v.transitions(from)) > 0 {
			anyWired = true
			break
		}
	}
	if !anyWired {
		out = append(out, Violation{
			Policy: "fsm-invariants",
			Detail: fmt.Sprintf("kind %q has no non-terminal statuses; FSM is unwired or every state is terminal — give the kind's statuses outgoing transitions in the transitions table in internal/entity/transition.go", v.kind),
		})
	}

	// Drift mode 2b: every transition target must itself be in
	// the kind's AllowedStatuses. Catches "FSM transitions to a
	// status the schemas table doesn't know about."
	declared := make(map[entity.Status]struct{}, len(v.statuses))
	for _, s := range v.statuses {
		declared[s] = struct{}{}
	}
	for _, from := range v.statuses {
		for _, to := range v.transitions(from) {
			if _, ok := declared[to]; !ok {
				out = append(out, Violation{
					Policy: "fsm-invariants",
					Detail: fmt.Sprintf("kind %q: transition %q → %q targets a status not in AllowedStatuses — add %q to the kind's statuses in internal/entity/entity.go, or retarget the transition in internal/entity/transition.go", v.kind, from, to, to),
				})
			}
		}
	}

	// Drift mode 3: CancelTarget pins the cancel verb's commitment
	// that any cancel projection routed through the verb lands on a
	// legal terminal state. The target depends on the current status
	// (`CancelTarget(kind, currentStatus) Status`), so walk every
	// non-terminal status of the kind and, when the returned target
	// is non-empty, assert it is (a) in AllowedStatuses and (b)
	// itself terminal. An empty return is permitted — the FSM may
	// have no cancel target from a particular non-terminal state
	// (ADR.accepted and Decision.accepted exit only via promote →
	// superseded; G-0163). The verb surfaces the empty case to the
	// operator as "no cancel target." Terminal current-states are
	// skipped — the verb's IsTerminal pre-flight guard handles them.
	for _, from := range v.statuses {
		if v.isTerminal(from) {
			continue
		}
		target := v.cancelTarget(from)
		if target == "" {
			continue
		}
		if !v.isAllowed(target) {
			out = append(out, Violation{
				Policy: "fsm-invariants",
				Detail: fmt.Sprintf("kind %q at %q: CancelTarget %q not in AllowedStatuses — return one of the kind's statuses from CancelTarget in internal/entity/transition.go", v.kind, from, target),
			})
			continue
		}
		if outs := v.transitions(target); len(outs) != 0 {
			out = append(out, Violation{
				Policy: "fsm-invariants",
				Detail: fmt.Sprintf("kind %q at %q: CancelTarget %q has outgoing transitions %v; must be terminal — return a terminal status from CancelTarget in internal/entity/transition.go, or remove %q's outgoing transitions", v.kind, from, target, outs, target),
			})
		}
	}

	// Drift mode 4: FSM contains a cycle. Kernel commitment 1
	// declares the FSM one-directional ("there is no demote").
	// Any transition that closes a cycle (e.g., cancelled → active)
	// silently violates that. DFS with three-color marking
	// detects back-edges in O(V+E). Because the FSMs are tiny
	// (≤5 states per kind), the deterministic ordering helps
	// reproducibility — sort sources before walking.
	sortedSources := make([]entity.Status, len(v.statuses))
	copy(sortedSources, v.statuses)
	slices.Sort(sortedSources)

	// One cycle is enough to violate the commitment. Stop at the
	// first one found per kind to keep the violation message
	// readable; finding all cycles is not the point.
	if cycle := findCycle(sortedSources, v.transitions); cycle != nil {
		out = append(out, Violation{
			Policy: "fsm-invariants",
			Detail: fmt.Sprintf("kind %q: FSM contains a cycle: %v — remove the transition that closes it in internal/entity/transition.go, since a status only moves forward", v.kind, cycle),
		})
	}
	return out
}

// fsmDAGViolations runs cycle detection on a composite FSM (one whose
// transitions are exposed via a single isLegal predicate rather than
// an AllowedTransitions(kind, from) probe). Returns one Violation if
// a cycle is found, none otherwise.
func fsmDAGViolations[T ~string](name string, statuses, entryStates []T, isLegal func(from, to T) bool) []Violation {
	allFroms := append([]T{}, entryStates...)
	allFroms = append(allFroms, statuses...)
	slices.Sort(allFroms)

	successors := func(from T) []T {
		var out []T
		for _, to := range statuses {
			if isLegal(from, to) {
				out = append(out, to)
			}
		}
		slices.Sort(out)
		return out
	}

	if cycle := findCycle(allFroms, successors); cycle != nil {
		return []Violation{{
			Policy: "fsm-invariants",
			Detail: fmt.Sprintf("composite FSM %q contains a cycle: %v — remove the transition that closes it in internal/entity/transition.go, since a state only moves forward", name, cycle),
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
func findCycle[T ~string](vertices []T, successors func(T) []T) []T {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[T]int, len(vertices))
	var found []T

	var visit func(s T, path []T) bool
	visit = func(s T, path []T) bool {
		color[s] = gray
		path = append(path, s)
		for _, to := range successors(s) {
			switch color[to] {
			case gray:
				found = append(append([]T{}, path...), to)
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
