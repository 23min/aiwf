package verb

import (
	"testing"

	"github.com/23min/aiwf/internal/gitops"
)

// TestCheckTrailerCoherence_Rules table-drives every required-together
// and mutually-exclusive rule from provenance-model.md §"Required-
// together and mutually-exclusive rules". Each case names the rule it
// expects to fire (or "" for happy-path combinations).
func TestCheckTrailerCoherence_Rules(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		trailers []gitops.Trailer
		wantRule string // "" = expect nil (coherent)
	}{
		// --- Happy-path combinations from §"Worked examples". ---
		{
			name: "Example 1: solo human, direct verb",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "add"},
				{Key: gitops.TrailerEntity, Value: "G-0001"},
				{Key: gitops.TrailerActor, Value: "human/peter"},
			},
		},
		{
			name: "Example 2: human directs LLM, no scope",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "add"},
				{Key: gitops.TrailerEntity, Value: "G-0001"},
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
			},
		},
		{
			name: "Example 3: authorized autonomous work (full trailer set)",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "promote"},
				{Key: gitops.TrailerEntity, Value: "M-0007"},
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
				{Key: gitops.TrailerAuthorizedBy, Value: "4b13a0f"},
			},
		},
		{
			name: "Example 6: forced override by human",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "cancel"},
				{Key: gitops.TrailerEntity, Value: "M-0007"},
				{Key: gitops.TrailerActor, Value: "human/peter"},
				{Key: gitops.TrailerForce, Value: "scope was wrong from the start"},
			},
		},
		{
			name: "audit-only by human (G24 recovery)",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "cancel"},
				{Key: gitops.TrailerEntity, Value: "G-0021"},
				{Key: gitops.TrailerActor, Value: "human/peter"},
				{Key: gitops.TrailerAuditOnly, Value: "manual commit recovery"},
			},
		},
		{
			name: "authorize commit (open scope) by human",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "authorize"},
				{Key: gitops.TrailerEntity, Value: "E-0003"},
				{Key: gitops.TrailerActor, Value: "human/peter"},
				{Key: gitops.TrailerTo, Value: "ai/claude"},
				{Key: gitops.TrailerScope, Value: "opened"},
			},
		},

		// --- Required-together violations. ---
		{
			name: "on-behalf-of without authorized-by",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
			},
			wantRule: CoherenceRuleOnBehalfOfMissingAuthorizedBy,
		},
		{
			name: "authorized-by without on-behalf-of",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerAuthorizedBy, Value: "4b13a0f"},
			},
			wantRule: CoherenceRuleAuthorizedByMissingOnBehalfOf,
		},
		{
			name: "non-human actor without principal",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerActor, Value: "ai/claude"},
			},
			wantRule: CoherenceRulePrincipalMissingForNonHumanActor,
		},
		{
			name: "bot actor without principal",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerActor, Value: "bot/ci"},
			},
			wantRule: CoherenceRulePrincipalMissingForNonHumanActor,
		},

		// --- Mutually-exclusive violations. ---
		{
			name: "principal with human actor",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerActor, Value: "human/peter"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
			},
			wantRule: CoherenceRulePrincipalForbiddenForHumanActor,
		},
		{
			name: "on-behalf-of with human actor",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerActor, Value: "human/peter"},
				{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
				{Key: gitops.TrailerAuthorizedBy, Value: "4b13a0f"},
			},
			wantRule: CoherenceRuleOnBehalfOfForbiddenForHumanActor,
		},
		{
			name: "force with on-behalf-of",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
				{Key: gitops.TrailerAuthorizedBy, Value: "4b13a0f"},
				{Key: gitops.TrailerForce, Value: "override"},
			},
			wantRule: CoherenceRuleForceWithOnBehalfOf,
		},
		{
			name: "force with non-human actor",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerForce, Value: "override"},
			},
			wantRule: CoherenceRuleForceNonHuman,
		},
		{
			name: "audit-only with force",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerActor, Value: "human/peter"},
				{Key: gitops.TrailerAuditOnly, Value: "recovery"},
				{Key: gitops.TrailerForce, Value: "skip-fsm"},
			},
			wantRule: CoherenceRuleAuditOnlyWithForce,
		},
		{
			name: "audit-only with non-human actor",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerAuditOnly, Value: "recovery"},
			},
			wantRule: CoherenceRuleAuditOnlyNonHuman,
		},

		// --- Sub-agent delegation pair: NOT enforced (deferred to G22). ---
		{
			name: "authorize commit inside an existing scope: tolerated (G22-deferred)",
			trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "authorize"},
				{Key: gitops.TrailerEntity, Value: "M-0007"},
				{Key: gitops.TrailerActor, Value: "ai/claude"},
				{Key: gitops.TrailerPrincipal, Value: "human/peter"},
				{Key: gitops.TrailerOnBehalfOf, Value: "human/peter"},
				{Key: gitops.TrailerAuthorizedBy, Value: "4b13a0f"},
				{Key: gitops.TrailerTo, Value: "ai/claude-sub"},
				{Key: gitops.TrailerScope, Value: "opened"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTrailerCoherence(tt.trailers)
			if tt.wantRule == "" {
				if err != nil {
					t.Errorf("expected nil error (coherent), got %v", err)
				}
				return
			}
			ce, gotRule := AsCoherenceError(err)
			if ce == nil {
				t.Fatalf("expected *CoherenceError, got %T: %v", err, err)
			}
			if gotRule != tt.wantRule {
				t.Errorf("rule = %q, want %q (msg: %s)", gotRule, tt.wantRule, ce.Message)
			}
		})
	}
}

// TestCheckTrailerCoherence_EmptyTrailerSet covers the degenerate
// case: no trailers at all. Coherent by construction (all rules
// trigger only on presence).
func TestCheckTrailerCoherence_EmptyTrailerSet(t *testing.T) {
	t.Parallel()
	if err := CheckTrailerCoherence(nil); err != nil {
		t.Errorf("empty trailer set should be coherent, got %v", err)
	}
	if err := CheckTrailerCoherence([]gitops.Trailer{}); err != nil {
		t.Errorf("empty slice should be coherent, got %v", err)
	}
}

// TestAsCoherenceError_NonCoherenceErrorPassesThrough confirms the
// helper returns nil/"" for any other error type — callers can safely
// dispatch on the result.
func TestAsCoherenceError_NonCoherenceErrorPassesThrough(t *testing.T) {
	t.Parallel()
	plain := &someOtherError{msg: "not a coherence error"}
	ce, rule := AsCoherenceError(plain)
	if ce != nil {
		t.Errorf("AsCoherenceError on non-coherence: ce = %v, want nil", ce)
	}
	if rule != "" {
		t.Errorf("AsCoherenceError on non-coherence: rule = %q, want empty", rule)
	}
}

type someOtherError struct{ msg string }

func (e *someOtherError) Error() string { return e.msg }
