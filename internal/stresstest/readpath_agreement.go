package stresstest

import (
	"fmt"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
)

// readpath_agreement.go — M-0300/AC-1: after each step of a composed
// sequence, run every verdict-rendering read path over the repository
// that step produced and report when two of them contradict each other
// about the same subject.
//
// The property holds no model of the reference-resolution tier rules.
// It compares two observations of the same bytes and never states which
// surface is right, which is what keeps it correct across a change to
// those rules and what keeps it cheap once the walker's mutation space
// widens.

// readPathAgreementInvariant is M-0300/AC-1's property: no two read
// paths contradict each other on the same bytes.
type readPathAgreementInvariant struct{}

// Name identifies the property in a violation message.
func (readPathAgreementInvariant) Name() string { return "read-path agreement" }

// Evaluate runs every verdict-rendering read path over dir and reports
// each subject two of them contradict each other about.
func (readPathAgreementInvariant) Evaluate(aiwfBin, dir, label string) ([]Violation, error) {
	gate, others, err := observeReadPaths(aiwfBin, dir)
	if err != nil {
		return nil, err
	}
	return classifyReadPathAgreement(label, gate, others), nil
}

// readPathGateSurface is the authoritative surface every other read path
// is measured against: the full `aiwf check` the pre-push hook runs.
var readPathGateSurface = []string{"check"}

// cheaperCheckSurfaces are the reduced `aiwf check` modes an operator
// reaches for between commits. Each emits the same findings envelope as
// the gate, so each is compared claim-for-claim.
var cheaperCheckSurfaces = [][]string{
	{"check", "--fast"},
	{"check", "--shape-only"},
}

// observeReadPaths runs every verdict-rendering read path over dir and
// returns what each stated.
//
// `aiwf status` is observed too, but through its own decoder: it renders
// the same in-memory rule pass into a report carrying a blocking count
// and warning rows without subcode or severity. It is recorded as a
// non-itemized observation rather than having a subcode inferred for it,
// because inferring one would be the harness deciding what a surface
// meant instead of reading what it said.
func observeReadPaths(aiwfBin, dir string) (gate readPathObservation, others []readPathObservation, err error) {
	gateEnv, err := runAiwfJSON(aiwfBin, dir, readPathGateSurface...)
	if err != nil { //coverage:ignore defensive: same launch-failure class pinned at its source by TestVerbSequenceScenario_RealBinary_RunErrorsWhenBinaryMissing
		return readPathObservation{}, nil, fmt.Errorf("running the authoritative %s: %w", surfaceName(readPathGateSurface), err)
	}
	gate = readPathObservationFrom(surfaceName(readPathGateSurface), gateEnv.Findings)

	for _, args := range cheaperCheckSurfaces {
		env, runErr := runAiwfJSON(aiwfBin, dir, args...)
		if runErr != nil { //coverage:ignore defensive: same launch-failure class as the gate call above
			return readPathObservation{}, nil, fmt.Errorf("running %s: %w", surfaceName(args), runErr)
		}
		others = append(others, readPathObservationFrom(surfaceName(args), env.Findings))
	}

	statusEnv, err := runAiwfStatusJSON(aiwfBin, dir)
	if err != nil { //coverage:ignore defensive: same launch-failure class as the gate call above
		return readPathObservation{}, nil, fmt.Errorf("running aiwf status: %w", err)
	}
	others = append(others, readPathObservation{
		Surface:  "aiwf status",
		Blocking: statusEnv.Result.Health.Errors,
	})

	return gate, others, nil
}

// surfaceName renders a surface's argv the way an operator would type it.
func surfaceName(args []string) string {
	return "aiwf " + strings.Join(args, " ")
}

// readPathSubject identifies what a claim is about: one rule's verdict
// on one entity. EntityID is canonicalized so a narrow legacy width and
// its canonical form name one subject (ADR-0008), and is empty for a
// tree-wide finding.
type readPathSubject struct {
	EntityID string
	Code     string
}

func (s readPathSubject) String() string {
	if s.EntityID == "" {
		return "(tree-wide)/" + s.Code
	}
	return s.EntityID + "/" + s.Code
}

// readPathClaim is one substantive classification a surface gave a
// subject: how it classified it, and whether that classification blocks.
type readPathClaim struct {
	Subcode  string
	Severity string
}

func (c readPathClaim) String() string {
	if c.Subcode == "" {
		return c.Severity
	}
	return c.Severity + "/" + c.Subcode
}

// readPathClaimSet is every substantive classification one surface gave
// one subject. A rule fires once per offending token or reference, so a
// subject carries a set rather than a single verdict.
type readPathClaimSet map[readPathClaim]bool

// readPathObservation is one surface's verdict on one repository, stated
// in the terms that surface actually speaks.
type readPathObservation struct {
	// Surface is the read path as an operator would invoke it.
	Surface string
	// Claims is what this surface substantively classified. A subject
	// absent from the map is one the surface made no claim about — it
	// either ran no rule that could produce one, or declined to judge.
	// Absence is never a claim, which is what lets surfaces running
	// different rule sets be compared at all.
	Claims map[readPathSubject]readPathClaimSet
	// Itemized reports whether Claims is this surface's full
	// itemization. A surface that states only aggregate counts leaves it
	// false and is compared on Blocking alone.
	Itemized bool
	// Blocking is how many blocking verdicts the surface stated.
	Blocking int
}

// readPathObservationFrom builds an itemized observation from one
// surface's findings envelope.
//
// A finding carrying check.SubcodeUnresolvedUnverified is dropped rather
// than recorded: that subcode is a surface saying it did not build the
// tier the question needs, which is a declined judgment and contradicts
// nothing (G-0558). Recording it as a claim is what would make this
// property fire on a correct tree.
func readPathObservationFrom(surface string, findings []verbEnvelopeFinding) readPathObservation {
	obs := readPathObservation{
		Surface:  surface,
		Claims:   make(map[readPathSubject]readPathClaimSet, len(findings)),
		Itemized: true,
	}
	for _, f := range findings {
		if f.Subcode == check.SubcodeUnresolvedUnverified {
			continue
		}
		subj := readPathSubject{EntityID: entity.Canonicalize(f.EntityID), Code: f.Code}
		if obs.Claims[subj] == nil {
			obs.Claims[subj] = readPathClaimSet{}
		}
		obs.Claims[subj][readPathClaim{Subcode: f.Subcode, Severity: f.Severity}] = true
		if f.Severity == severityError {
			obs.Blocking++
		}
	}
	return obs
}

// classifyReadPathAgreement judges one repository's read paths against
// each other. gate is the authoritative surface — the full `aiwf check`
// the pre-push hook runs; others are the cheaper surfaces an operator
// reaches for between commits.
//
// Itemized observations are compared pairwise, not merely against the
// gate: two cheaper surfaces can contradict each other on a subject the
// gate is silent about.
//
// A surface stating only aggregate counts is compared on the one thing
// it does state. Its rule set is a subset of the gate's with declined
// judgments already downgraded, so it can never legitimately block more
// than the gate does; blocking where the authoritative surface would not
// sends a reader to the one place that will tell them nothing is wrong.
func classifyReadPathAgreement(label string, gate readPathObservation, others []readPathObservation) []Violation {
	itemized := []readPathObservation{gate}
	var counted []readPathObservation
	for _, o := range others {
		if o.Itemized {
			itemized = append(itemized, o)
			continue
		}
		counted = append(counted, o)
	}

	var violations []Violation
	for i := range itemized {
		for j := i + 1; j < len(itemized); j++ {
			violations = append(violations, classifyReadPathPair(label, itemized[i], itemized[j])...)
		}
	}
	for _, o := range counted {
		if o.Blocking > gate.Blocking {
			violations = append(violations, Violation{Message: fmt.Sprintf(
				"%s: %s states %d blocking verdicts where the authoritative %s states %d",
				label, o.Surface, o.Blocking, gate.Surface, gate.Blocking,
			)})
		}
	}
	return violations
}

// classifyReadPathPair reports one violation per subject a and b
// contradict each other about, subject-sorted so a failing run is
// reproducible.
func classifyReadPathPair(label string, a, b readPathObservation) []Violation {
	subjects := make([]readPathSubject, 0, len(a.Claims))
	for subj := range a.Claims {
		if _, ok := b.Claims[subj]; ok {
			subjects = append(subjects, subj)
		}
	}
	sort.Slice(subjects, func(i, j int) bool { return subjects[i].String() < subjects[j].String() })

	var violations []Violation
	for _, subj := range subjects {
		if !claimsContradict(a.Claims[subj], b.Claims[subj]) {
			continue
		}
		violations = append(violations, Violation{Message: fmt.Sprintf(
			"%s: %s and %s classify %s differently: %s says %s, %s says %s",
			label, a.Surface, b.Surface, subj,
			a.Surface, renderClaimSet(a.Claims[subj]),
			b.Surface, renderClaimSet(b.Claims[subj]),
		)})
	}
	return violations
}

// claimsContradict reports whether two surfaces' claim sets about one
// subject cannot both hold.
//
// A set contained in the other is not a contradiction: the containing
// surface simply classified more of the same subject — one more bad
// reference, one more prose token — which is the absence rule applied at
// per-classification granularity rather than per-subject. Silence needs
// no separate arm, because an empty set is contained in everything.
func claimsContradict(a, b readPathClaimSet) bool {
	return !claimSubset(a, b) && !claimSubset(b, a)
}

// claimSubset reports whether every claim in a is also in b.
func claimSubset(a, b readPathClaimSet) bool {
	for c := range a {
		if !b[c] {
			return false
		}
	}
	return true
}

// renderClaimSet formats a claim set for a violation message, sorted so
// the same divergence reads identically on every run.
func renderClaimSet(set readPathClaimSet) string {
	rendered := make([]string, 0, len(set))
	for c := range set {
		rendered = append(rendered, c.String())
	}
	sort.Strings(rendered)
	return "[" + strings.Join(rendered, " ") + "]"
}
