// Package branchparse holds the canonical ritual-shape branch grammar
// defined by ADR-0010: `epic/E-NNNN-<slug>`, `milestone/M-NNNN-<slug>`,
// `patch/g-NNNN-<slug>` (case-insensitive id segment). Lifted from
// internal/cli/status/worktrees.go in M-0102 so M-0103's preflight,
// M-0102's `aiwf authorize --branch` completion, and the existing
// `aiwf status --worktrees` correlation share one regex set — drift
// between them is structurally impossible.
package branchparse

import (
	"regexp"
	"strings"
)

// branchEntityPattern matches the conventional ritual-branch prefixes,
// enforcing prefix-id coherence (G-0198): each prefix accepts only its
// own id kind.
//
//	epic/E-NNNN-<slug>          → E-NNNN
//	milestone/M-NNNN-<slug>     → M-NNNN
//	patch/g-NNNN-<slug>         → G-NNNN (id segment is case-insensitive)
//
// Incoherent combinations (epic/M-..., milestone/E-..., patch/E-...) and
// other shapes (fix/*, chore/*, patch/<topic-without-id>) yield "" — the
// prefix and the id kind must agree. The three alternation groups are
// mutually exclusive, so exactly one is non-empty on a match.
// Narrow-legacy id widths (E-01, M-007) are preserved as-typed on output;
// canonicalization is a downstream concern handled by entity.Canonicalize
// at the consumer's discretion (e.g. when stamping a trailer).
var branchEntityPattern = regexp.MustCompile(`^(?:epic/([Ee]-\d+)|milestone/([Mm]-\d+)|patch/([Gg]-\d+))(?:-|$)`)

// ParseEntityFromBranch tries to derive an entity id from a ritual-shape
// branch name, requiring the prefix and id kind to agree:
// `epic/E-NNNN-...`, `milestone/M-NNNN-...`, `patch/g-NNNN-...`. Returns
// "" on no match or on a prefix-id mismatch (G-0198) — the caller treats
// that as "not a ritual branch."
func ParseEntityFromBranch(branch string) string {
	m := branchEntityPattern.FindStringSubmatch(branch)
	if m == nil {
		return ""
	}
	// Exactly one of the three alternation groups (epic/E-, milestone/M-,
	// patch/G-) matched; the other two are empty, so concatenation yields
	// the single id segment.
	return strings.ToUpper(m[1] + m[2] + m[3])
}

// branchRungPattern recognizes the kind segment of a ritual-shape
// branch name. Requires an id segment after the kind prefix (matching
// branchEntityPattern's stricter shape), so non-ritual branches under
// the same prefix (e.g., `patch/some-topic` without an id) don't
// falsely classify as a ritual rung.
var branchRungPattern = regexp.MustCompile(`^(epic|milestone|patch)/[EeMmGg]-\d+`)

// RungOf classifies a branch name into its ritual rung:
//
//   - "trunk"     if branch equals trunkShort (config-driven trunk detection).
//   - "epic"      if branch matches `epic/...`.
//   - "milestone" if branch matches `milestone/...`.
//   - "patch"     if branch matches `patch/...`.
//   - ""          on no match (non-ritual branch; detached HEAD; degenerate input).
//
// trunkShort is the consumer's configured trunk short-name as sourced
// from Config.TrunkBranchShortName() (M-0161/AC-1). When trunkShort is
// the empty string, no branch can be classified as "trunk" — the
// empty-guard prevents silent coincidence with an empty CurrentBranch
// (detached-HEAD state).
//
// Trunk detection is config-driven so a repo using `master` (or any
// other operator-chosen trunk) gets the right rung classification
// without regex hardcoding. The ritual-prefix detection (epic/
// milestone/ patch/) is independent of trunkShort — both predicates
// are checked in sequence.
//
// Used by the M-0161/AC-2 verb-layer authorize predicate alongside
// LegalRungPair: the verb computes (RungOf(current, trunk),
// RungOf(target, trunk)) and refuses when the pair is not in the
// legal set.
func RungOf(branch, trunkShort string) string {
	if branch == "" {
		return ""
	}
	if trunkShort != "" && branch == trunkShort {
		return "trunk"
	}
	m := branchRungPattern.FindStringSubmatch(branch)
	if m == nil {
		return ""
	}
	return m[1]
}

// legalRungPairs is the closed set of (currentRung, targetRung) pairs
// the M-0161/AC-2 authorize-predicate accepts. Every other pair refuses.
// Per ADR-0010:
//
//   - trunk → epic       — aiwfx-start-epic (sovereign promote +
//     authorize on trunk; epic branch cut next).
//   - epic → milestone   — aiwfx-start-milestone from parent epic.
//   - milestone → patch  — wf-patch under a milestone.
//   - epic → patch       — wf-patch directly under an epic, skipping
//     an intermediate milestone (operator-intent;
//     not a typo).
//
// All other (rung, rung) combinations are typos (same-rung, cross-rung),
// up-the-tree shapes (milestone → epic, etc.), or trunk-targeting
// shapes (anything → trunk; AI work on trunk is verboten per ADR-0010).
var legalRungPairs = map[[2]string]bool{
	{"trunk", "epic"}:      true,
	{"epic", "milestone"}:  true,
	{"milestone", "patch"}: true,
	{"epic", "patch"}:      true,
}

// LegalRungPair returns true iff the (currentRung, targetRung) pair is
// in the closed legal set. Any pair involving an empty rung (`""` on
// either side) returns false — the rung predicate is only meaningful
// when both sides classify.
//
// Used by the M-0161/AC-2 verb-layer authorize predicate.
func LegalRungPair(currentRung, targetRung string) bool {
	return legalRungPairs[[2]string{currentRung, targetRung}]
}
