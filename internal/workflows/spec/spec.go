// Package spec is the canonical Go encoding of aiwf's legal-workflow surface.
//
// The package is built and maintained per ADR-0011's three-pass methodology
// (Pass A audit, Pass B first-principles, Pass C reconcile). M-0123 is the
// reconciliation milestone; this package is its load-bearing deliverable.
//
// The cell-by-cell table lives in Rules() and AntiRules(); per-cell positive
// and negative coverage lives under internal/policies/ (M-0124, M-0125). The
// bidirectional drift policy under internal/policies/ closes the spec
// against the impl in both directions.
//
// Anti-rules. The kernel deliberately excludes patterns that contributors
// might mis-assume are policed. AntiRules() catalogs them. The meta-policy
// is: a candidate becomes an anti-rule only when (a) reconciliation surfaces
// a plausibly mis-assumed pattern and (b) Pass C judges the spec better
// served by an explicit non-rule entry than by silence. The list grows as
// future PRs surface new near-rules; small follow-up acts (a gap, a one-row
// spec amendment) are the expected cadence.
//
// Schema invariants enforced by drift policies in internal/policies/:
//   - Outcome != OutcomeUnspecified for every cell.
//   - Outcome == OutcomeIllegal implies RejectionLayer != RejectionLayerNone.
//   - RejectionLayer == RejectionLayerVerbTime implies BlockingStrict == true.
//   - Outcome == OutcomeLegal implies ExpectedErrorCode == "".
//   - Sources.Decision (when non-empty) resolves to a planning-tree entity.
//   - The (Kind, FromState, Verb) triple uniquely keys each Rule.
package spec

import (
	"github.com/23min/aiwf/internal/entity"
)

// Kind extensions for sub-FSM cells. ACs and TDD-phase are not first-class
// entity kinds (entity.Kind covers the six top-level kinds), but they have
// their own FSMs (entity.IsLegalACTransition, entity.IsLegalTDDPhaseTransition)
// that the spec must encode. These constants extend the Kind value space
// for spec-table purposes only — they are NOT added to entity.transitions.
const (
	KindAC       entity.Kind = "ac"
	KindTDDPhase entity.Kind = "tdd-phase"
)

// Outcome is the legal/illegal axis of a Rule cell.
//
// The zero value OutcomeUnspecified is a sentinel that surfaces "forgot to
// set Outcome on a Rule literal" bugs at drift-policy time (the policy test
// asserts no cell carries OutcomeUnspecified).
type Outcome int

// Outcome values.
const (
	OutcomeUnspecified Outcome = iota
	OutcomeLegal
	OutcomeIllegal
)

// RejectionLayer names where in the kernel pipeline an illegal cell is
// rejected. Verb-time rejections are returned by the verb itself (non-zero
// exit, no commit, no side effect); check-time rejections are surfaced as
// findings by aiwf check (the verb may have succeeded structurally).
//
// The zero value RejectionLayerNone is meaningful — it applies to legal
// cells (where the field is irrelevant). The drift policy asserts that
// illegal cells carry a non-zero RejectionLayer.
type RejectionLayer int

// RejectionLayer values.
const (
	RejectionLayerNone RejectionLayer = iota
	RejectionLayerVerbTime
	RejectionLayerCheckTime
)

// Predicate is a precondition expressed against the planning tree at
// verb-time. Subject vocabulary is closed to five forms:
//
//	self.<field>         — the entity the verb operates on
//	parent.<field>       — the entity's parent (e.g., milestone's epic)
//	all-children.<field> — every child satisfies the predicate
//	any-child.<field>    — at least one child satisfies
//	scope-reach          — actor's active-scope-entity reaches target via
//	                       the scope-tree edges (D-0006); not a field
//	                       comparison
//
// Op vocabulary is closed to six forms: ==, !=, ∈, ∉, non-empty, exists.
//
// Widening either vocabulary requires a decision entity per the M-0123
// body's predicate-vocabulary constraint.
type Predicate struct {
	Subject string
	Op      string
	Value   string
}

// RuleSource records the catalog citations that motivated this cell. The
// population shape encodes the reconciliation class per M-0123 body
// §"Rule sources":
//
//	Agreement   → Audit non-empty, FP non-empty, Decision empty
//	Audit-only  → Audit non-empty, FP empty, Decision empty
//	FP-only     → Audit empty, FP non-empty, Decision non-empty (D-NNNN)
//	Conflict    → Audit non-empty, FP non-empty, Decision non-empty (D-NNNN)
//
// AC-6's drift test asserts every cell's Sources.Decision (when non-empty)
// resolves to an existing D-NNNN entity via tree.Load + Tree.ByID.
type RuleSource struct {
	Audit    []string
	FP       []string
	Decision string
}

// Rule is one legality cell in the spec table.
//
// Keyed by (Kind, FromState, Verb). Outcome carries Legal/Illegal; for
// Illegal cells, RejectionLayer + BlockingStrict + ExpectedErrorCode pin
// the rejection mode. Preconditions narrow when the cell applies (e.g., a
// legal-only-if-children-all-terminal precondition pairs with a companion
// illegal-when-any-child-non-terminal cell).
//
// Sources records the catalogs that motivated the cell — Audit (R-AUDIT-NNNN
// ids), FP (R-FP-NNNN ids), Decision (D-NNNN, populated only for FP-only and
// Conflict classes).
//
// Cross-cutting precondition rules that are NOT (Kind, FromState, Verb)
// cells (ADR-0013, e.g. the scope-reach rule) live in [GlobalRules], a
// separate accessor — they are deliberately absent from [Rules] so every
// per-cell consumer iterates cells only, with no per-rule exclusion. Only
// the code-oriented AC-5 drift arms union the two.
//
// ID (optional, added by M-0158) carries an explicit string identifier for
// cells that live outside the (Kind, FromState, Verb) keyspace — layer-4
// branch-choreography cells (e.g. `branch-cell-1`, `branch-cell-override-
// preflight`). Layers 1–3 leave ID empty and continue to be keyed by the
// natural tuple; their tests and meta-policies are unaffected. The ID is
// the consumer-facing name when an explicit cell-id-to-test-name
// convention is required (M-0158/AC-2, AC-3, AC-5).
type Rule struct {
	ID                string
	Kind              entity.Kind
	FromState         string
	Verb              string
	Preconditions     []Predicate
	Outcome           Outcome
	ExpectedErrorCode string
	RejectionLayer    RejectionLayer
	BlockingStrict    bool
	Sources           RuleSource
}

// AntiRule catalogs a pattern that the kernel deliberately does NOT police.
// Anti-rules clarify scope by negation; they are not cells in the (Kind,
// FromState, Verb) keyed Rule table.
//
// Pass C's anti-rule meta-policy: a candidate becomes an anti-rule only
// when reconciliation surfaces a plausibly-mis-assumed pattern. Examples
// from Pass B §10 and Q10:
//
//   - A milestone is NOT required to have ≥1 AC.
//   - An epic MAY transition proposed → active with zero milestones.
//   - There is no kernel rule about which branch a verb is legal on.
//
// AntiRules() returns the closed-set list; the order is the listing order
// in the spec body (loosely thematic, per-source).
type AntiRule struct {
	ID        string
	Statement string
	Reasoning string
	Sources   RuleSource
}
