package stresstest

import "fmt"

// invariant.go — M-0300: the oracle seam. A scenario that composes verb
// sequences judges each state it reaches against a set of registered
// properties rather than against an expected end state it computed
// itself (D-0063).
//
// The distinction is what makes widening the mutation space affordable.
// An oracle that knows the right answer needs a second implementation of
// the kernel — one that drifts against the first and is wrong in a way
// no test catches, because it *is* the test — and needs extending for
// every new axis. A property that compares two observations of the same
// bytes needs neither.

// Invariant is one property the harness requires to hold in every state
// a scenario reaches.
type Invariant interface {
	// Name identifies the property in a violation message.
	Name() string
	// Evaluate judges the repository at dir. label identifies the step
	// that produced that state, so a failure is reproducible without
	// replaying the whole sequence.
	//
	// A returned error means the property could not be evaluated — a
	// subprocess failed to launch, a repository could not be read — which
	// is a harness fault, distinct from the property being violated.
	Evaluate(aiwfBin, dir, label string) ([]Violation, error)
}

// walkInvariants are the properties every composed sequence is judged
// against after each step.
//
// The list-vs-ground-truth invariant (M-0250/AC-3) is registered here
// alongside the agreement properties rather than called inline, so the
// walker has one way of asking "does everything that must hold, hold?"
// instead of one per property.
func walkInvariants() []Invariant {
	return []Invariant{
		listGroundTruthInvariant{},
		readPathAgreementInvariant{},
		refLessStabilityInvariant{},
	}
}

// evaluateInvariants runs every invariant against the repository at dir
// and returns their violations in registration order.
func evaluateInvariants(invariants []Invariant, aiwfBin, dir, label string) ([]Violation, error) {
	var violations []Violation
	for _, inv := range invariants {
		found, err := inv.Evaluate(aiwfBin, dir, label)
		if err != nil {
			return nil, fmt.Errorf("evaluating the %s invariant after %s: %w", inv.Name(), label, err)
		}
		violations = append(violations, found...)
	}
	return violations, nil
}

// listGroundTruthInvariant is M-0250/AC-3's property behind the seam:
// `aiwf list --archived` reports exactly the entities on disk.
type listGroundTruthInvariant struct{}

// Name identifies the property in a violation message.
func (listGroundTruthInvariant) Name() string { return "list-vs-ground-truth" }

// Evaluate compares `aiwf list --archived`'s output against the entities
// on disk at dir.
func (listGroundTruthInvariant) Evaluate(aiwfBin, dir, label string) ([]Violation, error) {
	return checkListInvariant(aiwfBin, dir, label)
}
