package check

import (
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// CodeAreaUnknown is the finding code emitted by AreaUnknown. Typed per
// G-0129 so the compiler closes on rename / retire across the emit site
// and tests.
const CodeAreaUnknown = "area-unknown"

// ApplyAreaRequiredStrict bumps the severity of the area-axis findings from
// warning to error when required=true. Mutates the findings slice in place.
// The escalation mirrors ApplyTDDStrict: the rules stay config-agnostic
// (always emit at warning), and the strictness bump is a separate, testable
// post-pass composed at the CLI layer where `areas.required` is in scope.
//
// Two axes escalate together under `areas.required: true`:
//   - the entity-tag axis (M-0178/AC-7): present-but-undeclared `area`
//     fires area-unknown — escalated here so the pre-push hook blocks it
//     too. (Empty area is the separate area-required error.)
//   - the path-claim axis (M-0180): a dead path glob fires area-dead-glob
//     and two areas claiming one directory fire area-overlap — both
//     escalated here so a monorepo that opted into strictness cannot push an
//     area pointing at nothing or an ambiguous path oracle.
//
// With required off, all stay warnings (byte-for-byte the pre-knob
// behavior). The bumper is intentionally scoped: codes outside the
// escalated area set (area-unknown, area-dead-glob, area-overlap) pass
// through unchanged regardless of the flag.
func ApplyAreaRequiredStrict(findings []Finding, required bool) {
	if !required {
		return
	}
	for i := range findings {
		switch findings[i].Code {
		case CodeAreaUnknown, CodeAreaDeadGlob, CodeAreaOverlap:
			findings[i].Severity = SeverityError
		}
	}
}

// AreaUnknown (warning) reports any non-archived entity whose `area`
// frontmatter value is present and non-empty but not a member of the
// declared set (`aiwf.yaml: areas.members`). It is the present-⇒-declared
// chokepoint for E-0043 — typo protection for the optional grouping tag,
// the authoritative surface a creation-time flag alone can't cover (a
// hand-edit or an `aiwf import` can introduce an undeclared area without
// passing through `aiwf add --area`).
//
// Three behaviors fall out of the guards, in order:
//   - Inert when declared is empty (no `areas` block): present `area`
//     values parse but nothing validates, per M-0171's "the field is
//     inert until a block is declared" contract.
//   - Absence (empty `area`) is never evaluated: absent / explicit-null /
//     empty all deserialize to "" and only a present, non-empty value can
//     be "unknown".
//   - Archive scoping (ADR-0004 §"check shape rules"): archived entities
//     are out of scope for active linting, matching the other
//     shape-and-health rules.
//
// The rule reads the *stored* area, so only root kinds that carry their
// own `area` can fire; a milestone (area blanked at load, derived from
// its parent epic) never double-reports under a bad-area epic.
//
// Composed at the CLI layer (internal/cli/check) with the declared set
// sourced from config — the same seam TreeDiscipline, the contract
// checks, and the tests-metrics check use — so the pure check.Run stays
// config-agnostic (the boundary M-0171/AC-4's metamorphic guard pins).
// Severity is warning with no strictness knob (E-0043 / M-0172 decision).
func AreaUnknown(t *tree.Tree, declared []string) []Finding {
	if len(declared) == 0 {
		return nil
	}
	var findings []Finding
	for _, e := range t.Entities {
		if e.Area == "" {
			continue
		}
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		// The reserved `global` sentinel and any declared member are both
		// valid (M-0184); membership routes through the SSOT predicate so
		// there is no parallel `== global` check here.
		if entity.IsValidAreaValue(e.Area, declared) {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodeAreaUnknown,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"%s declares area %q which is not in the declared set (aiwf.yaml: areas.members: %s)",
				e.ID, e.Area, strings.Join(declared, ", ")),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "area",
		})
	}
	return findings
}
