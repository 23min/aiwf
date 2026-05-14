package entity

import (
	"testing"
)

// Tests in this file are property tests over the entity-kind status FSMs
// and the two composite FSMs (AC status, TDD phase). They land under
// G44 item 2.
//
// The gap text proposed pgregory.net/rapid for state-machine generation;
// in practice the FSMs are small enough (3–5 states per kind, ≤15
// transitions per kind) that exhaustive enumeration over the input space
// is feasible and strictly stronger than randomized walks. No new
// dependency is added.
//
// The load-bearing property the kernel relies on (commitment 1: closed
// status sets per kind) is enforced *jointly* by two parallel sources of
// truth: AllowedStatuses(kind) declares the closed set, and
// transitions[kind] declares the legal moves. These tests assert the two
// agree, plus the standard FSM hygiene rules (no self-transitions,
// every state reachable from the initial state, terminality is honored).

// kindInitialStatus is the entry state for each kind's FSM. Every
// non-initial state must be reachable from this one via some sequence
// of legal transitions.
var kindInitialStatus = map[Kind]string{
	KindEpic:      "proposed",
	KindMilestone: "draft",
	KindADR:       "proposed",
	KindGap:       "open",
	KindDecision:  "proposed",
	KindContract:  "proposed",
}

// TestKindFSM_StateSetAgreement: the set of states declared by the
// schemas table (AllowedStatuses) must equal the set of states the
// transitions map mentions (as keys *or* as transition targets).
// Drift between these two sources of truth means either a state is
// declared legal but unreachable in the FSM, or the FSM admits
// transitions to states the schema considers illegal.
func TestKindFSM_StateSetAgreement(t *testing.T) {
	t.Parallel()
	for kind := range transitions {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			declared := stringSet(AllowedStatuses(kind))
			fsm := stringSet(nil)
			for from, tos := range transitions[kind] {
				fsm[from] = struct{}{}
				for _, to := range tos {
					fsm[to] = struct{}{}
				}
			}
			for s := range declared {
				if _, ok := fsm[s]; !ok {
					t.Errorf("status %q in AllowedStatuses(%s) but never appears in FSM", s, kind)
				}
			}
			for s := range fsm {
				if _, ok := declared[s]; !ok {
					t.Errorf("status %q in FSM but not in AllowedStatuses(%s)", s, kind)
				}
			}
		})
	}
}

// TestKindFSM_EveryDeclaredStatusIsAFSMSource: every state in the
// declared closed set must appear as a key in the transitions map (i.e.
// have an explicit outgoing-set, even if empty). A state present as a
// transition target but missing as a source is a silent dead-state
// caused by typo / forgotten entry — the entity can land there but
// nothing knows what to do with it.
func TestKindFSM_EveryDeclaredStatusIsAFSMSource(t *testing.T) {
	t.Parallel()
	for kind, kindFSM := range transitions {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			for _, s := range AllowedStatuses(kind) {
				if _, ok := kindFSM[s]; !ok {
					t.Errorf("status %q is declared allowed for %s but has no entry as an FSM source state", s, kind)
				}
			}
		})
	}
}

// TestKindFSM_TerminalityHonored: each kind has at least one terminal
// state (empty outgoing list). Without one, the FSM has no completion
// path and the kind never reaches a "done"-like resting state. This is
// kernel commitment 1's "closed status set with terminal members" rule
// stated mechanically.
func TestKindFSM_TerminalityHonored(t *testing.T) {
	t.Parallel()
	for kind, kindFSM := range transitions {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			anyTerminal := false
			for _, tos := range kindFSM {
				if len(tos) == 0 {
					anyTerminal = true
					break
				}
			}
			if !anyTerminal {
				t.Errorf("%s FSM has no terminal state", kind)
			}
		})
	}
}

// TestKindFSM_NoSelfTransitions: a transition from X to X serves no
// purpose under the kernel's "every mutating verb produces exactly one
// commit" rule (the commit would be a no-op promote). Forbidden.
func TestKindFSM_NoSelfTransitions(t *testing.T) {
	t.Parallel()
	for kind, kindFSM := range transitions {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			for from, tos := range kindFSM {
				for _, to := range tos {
					if from == to {
						t.Errorf("%s: self-transition %q → %q", kind, from, to)
					}
				}
			}
		})
	}
}

// TestKindFSM_AllStatesReachableFromInitial: every state in the closed
// set is reachable from the kind's initial state via some sequence of
// legal transitions. An unreachable state means the FSM declares the
// status legal but no entity can ever land there through verb-driven
// transitions — silent dead code in the closed set.
func TestKindFSM_AllStatesReachableFromInitial(t *testing.T) {
	t.Parallel()
	for kind, kindFSM := range transitions {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			initial, ok := kindInitialStatus[kind]
			if !ok {
				t.Fatalf("test bug: no initial status declared for %s", kind)
			}
			reachable := map[string]struct{}{initial: {}}
			frontier := []string{initial}
			for len(frontier) > 0 {
				next := frontier[0]
				frontier = frontier[1:]
				for _, to := range kindFSM[next] {
					if _, seen := reachable[to]; !seen {
						reachable[to] = struct{}{}
						frontier = append(frontier, to)
					}
				}
			}
			for s := range kindFSM {
				if _, ok := reachable[s]; !ok {
					t.Errorf("%s: status %q is declared but not reachable from initial state %q", kind, s, initial)
				}
			}
		})
	}
}

// TestKindFSM_ValidateTransition_TotalOverClosedSet: ValidateTransition
// returns nil iff (from, to) is in the declared transition set, and
// returns a non-nil error otherwise. Exhausts the cross-product of
// declared statuses for each kind — if the function panics on any pair,
// the test fails with a recover()'d error.
func TestKindFSM_ValidateTransition_TotalOverClosedSet(t *testing.T) {
	t.Parallel()
	for kind := range transitions {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			statuses := AllowedStatuses(kind)
			for _, from := range statuses {
				for _, to := range statuses {
					expectLegal := false
					for _, allowed := range transitions[kind][from] {
						if allowed == to {
							expectLegal = true
							break
						}
					}
					func() {
						defer func() {
							if r := recover(); r != nil {
								t.Errorf("%s: ValidateTransition(%q, %q) panicked: %v", kind, from, to, r)
							}
						}()
						err := ValidateTransition(kind, from, to)
						if expectLegal && err != nil {
							t.Errorf("%s: ValidateTransition(%q, %q) returned error %v; expected nil", kind, from, to, err)
						}
						if !expectLegal && err == nil {
							t.Errorf("%s: ValidateTransition(%q, %q) returned nil; expected error", kind, from, to)
						}
					}()
				}
			}
		})
	}
}

// TestACFSM_StateSetAgreement: same closed-set property, applied to
// the AC status FSM. acAllowedStatuses (declared) must equal the set
// of states the acTransitions map mentions.
func TestACFSM_StateSetAgreement(t *testing.T) {
	t.Parallel()
	declared := stringSet(acAllowedStatuses)
	fsm := stringSet(nil)
	for from, tos := range acTransitions {
		fsm[from] = struct{}{}
		for _, to := range tos {
			fsm[to] = struct{}{}
		}
	}
	for s := range declared {
		if _, ok := fsm[s]; !ok {
			t.Errorf("AC status %q declared allowed but never appears in FSM", s)
		}
	}
	for s := range fsm {
		if _, ok := declared[s]; !ok {
			t.Errorf("AC status %q in FSM but not in declared set", s)
		}
	}
}

// TestACFSM_AllStatesReachableFromOpen: every AC state is reachable
// from the initial state "open" via some sequence of legal transitions.
func TestACFSM_AllStatesReachableFromOpen(t *testing.T) {
	t.Parallel()
	reachable := map[string]struct{}{"open": {}}
	frontier := []string{"open"}
	for len(frontier) > 0 {
		next := frontier[0]
		frontier = frontier[1:]
		for _, to := range acTransitions[next] {
			if _, seen := reachable[to]; !seen {
				reachable[to] = struct{}{}
				frontier = append(frontier, to)
			}
		}
	}
	for _, s := range acAllowedStatuses {
		if _, ok := reachable[s]; !ok {
			t.Errorf("AC status %q is declared but not reachable from initial state \"open\"", s)
		}
	}
}

// TestACFSM_IsLegalACTransition_TotalOverClosedSet: IsLegalACTransition
// returns true iff (from, to) is in acTransitions[from], false otherwise.
// Exhaustive over the closed-set cross-product. Includes the
// IsLegalACTransition(_, _) === false rule for unknown statuses by
// passing values not in the closed set.
func TestACFSM_IsLegalACTransition_TotalOverClosedSet(t *testing.T) {
	t.Parallel()
	for _, from := range acAllowedStatuses {
		for _, to := range acAllowedStatuses {
			expectLegal := false
			for _, allowed := range acTransitions[from] {
				if allowed == to {
					expectLegal = true
					break
				}
			}
			got := IsLegalACTransition(from, to)
			if got != expectLegal {
				t.Errorf("IsLegalACTransition(%q, %q) = %v; want %v", from, to, got, expectLegal)
			}
		}
	}
	// Unknown statuses always return false, both directions.
	for _, from := range acAllowedStatuses {
		if IsLegalACTransition(from, "unknown_state") {
			t.Errorf("IsLegalACTransition(%q, \"unknown_state\") = true; want false", from)
		}
		if IsLegalACTransition("unknown_state", from) {
			t.Errorf("IsLegalACTransition(\"unknown_state\", %q) = true; want false", from)
		}
	}
}

// TestTDDPhaseFSM_StateSetAgreement: tddPhases (declared) must equal
// the non-empty-string states in the FSM. The empty string is the
// pre-cycle entry sentinel — present in the FSM as a source but not a
// declared phase value (entities never carry an empty-string phase).
func TestTDDPhaseFSM_StateSetAgreement(t *testing.T) {
	t.Parallel()
	declared := stringSet(tddPhases)
	fsm := stringSet(nil)
	for from, tos := range tddPhaseTransitions {
		if from != "" {
			fsm[from] = struct{}{}
		}
		for _, to := range tos {
			fsm[to] = struct{}{}
		}
	}
	for s := range declared {
		if _, ok := fsm[s]; !ok {
			t.Errorf("TDD phase %q declared allowed but never appears in FSM", s)
		}
	}
	for s := range fsm {
		if _, ok := declared[s]; !ok {
			t.Errorf("TDD phase %q in FSM but not in declared set", s)
		}
	}
}

// TestTDDPhaseFSM_AllStatesReachableFromEmpty: every TDD phase state
// is reachable from the empty-string entry sentinel. red is the entry,
// then linear progression.
func TestTDDPhaseFSM_AllStatesReachableFromEmpty(t *testing.T) {
	t.Parallel()
	reachable := map[string]struct{}{"": {}}
	frontier := []string{""}
	for len(frontier) > 0 {
		next := frontier[0]
		frontier = frontier[1:]
		for _, to := range tddPhaseTransitions[next] {
			if _, seen := reachable[to]; !seen {
				reachable[to] = struct{}{}
				frontier = append(frontier, to)
			}
		}
	}
	for _, s := range tddPhases {
		if _, ok := reachable[s]; !ok {
			t.Errorf("TDD phase %q is declared but not reachable from initial pre-cycle state", s)
		}
	}
}

// TestTDDPhaseFSM_IsLegalTDDPhaseTransition_TotalOverClosedSet:
// IsLegalTDDPhaseTransition returns true iff (from, to) is in
// tddPhaseTransitions[from], false otherwise. Exhaustive over the
// closed-set cross-product, plus the empty-string entry sentinel and
// unknown-status fail-closed rule.
func TestTDDPhaseFSM_IsLegalTDDPhaseTransition_TotalOverClosedSet(t *testing.T) {
	t.Parallel()
	allFroms := append([]string{""}, tddPhases...)
	for _, from := range allFroms {
		for _, to := range tddPhases {
			expectLegal := false
			for _, allowed := range tddPhaseTransitions[from] {
				if allowed == to {
					expectLegal = true
					break
				}
			}
			got := IsLegalTDDPhaseTransition(from, to)
			if got != expectLegal {
				t.Errorf("IsLegalTDDPhaseTransition(%q, %q) = %v; want %v", from, to, got, expectLegal)
			}
		}
		if IsLegalTDDPhaseTransition(from, "unknown_phase") {
			t.Errorf("IsLegalTDDPhaseTransition(%q, \"unknown_phase\") = true; want false", from)
		}
	}
}

// TestCancelTarget_AllKinds: CancelTarget returns a status that is
// (a) in the kind's closed set, and (b) terminal in the kind's FSM.
// This pins the cancel verb's commitment that "any non-terminal
// entity can be cancelled to a terminal state in one step."
func TestCancelTarget_AllKinds(t *testing.T) {
	t.Parallel()
	for kind := range transitions {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			target := CancelTarget(kind)
			if target == "" {
				t.Fatalf("%s: CancelTarget returned empty", kind)
			}
			if !IsAllowedStatus(kind, target) {
				t.Errorf("%s: CancelTarget %q not in AllowedStatuses", kind, target)
			}
			if outs := transitions[kind][target]; len(outs) != 0 {
				t.Errorf("%s: CancelTarget %q is not terminal (outgoing: %v)", kind, target, outs)
			}
		})
	}
}

// stringSet builds a set from a slice of strings. Returns an empty
// (non-nil) set when xs is nil so callers can write into it.
func stringSet(xs []string) map[string]struct{} {
	s := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		s[x] = struct{}{}
	}
	return s
}
