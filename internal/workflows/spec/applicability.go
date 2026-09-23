package spec

import "github.com/23min/aiwf/internal/entity"

// Applicability declares a verb's domain independently of entity state.
// Reason explains an inapplicable pair; refusal at one state does not make
// an otherwise meaningful kind/verb pair inapplicable.
type Applicability struct {
	Kind    entity.Kind
	Verb    string
	Applies bool
	Reason  string
}

// Applicabilities declares every kind/transition-verb pair (D-0077).
// It is separate from Rules so an absent transition cannot imply inapplicability.
func Applicabilities() []Applicability {
	return []Applicability{
		{Kind: entity.KindEpic, Verb: "promote", Applies: true},
		{Kind: entity.KindEpic, Verb: "cancel", Applies: true},
		{Kind: entity.KindMilestone, Verb: "promote", Applies: true},
		{Kind: entity.KindMilestone, Verb: "cancel", Applies: true},
		{Kind: entity.KindADR, Verb: "promote", Applies: true},
		{Kind: entity.KindADR, Verb: "cancel", Applies: true},
		{Kind: entity.KindGap, Verb: "promote", Applies: true},
		{Kind: entity.KindGap, Verb: "cancel", Applies: true},
		{Kind: entity.KindDecision, Verb: "promote", Applies: true},
		{Kind: entity.KindDecision, Verb: "cancel", Applies: true},
		{Kind: entity.KindContract, Verb: "promote", Applies: true},
		{Kind: entity.KindContract, Verb: "cancel", Applies: true},
		{Kind: KindAC, Verb: "promote", Applies: true},
		{Kind: KindAC, Verb: "cancel", Applies: true},
		{Kind: KindTDDPhase, Verb: "promote", Applies: true},
		{Kind: KindTDDPhase, Verb: "cancel", Reason: "Cancellation acts on AC status; the TDD-phase ladder has no cancellation operation."},
	}
}
