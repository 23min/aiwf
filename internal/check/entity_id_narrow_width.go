package check

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// entityIDNarrowWidth implements the M-083 AC-1 drift-check rule.
//
// The rule classifies the active tree (entities outside any
// `<kind>/archive/` subtree) as uniform-narrow, uniform-canonical,
// or mixed:
//
//   - Uniform narrow active tree → consumer hasn't run `aiwf rewidth`
//     yet → silent.
//   - Uniform canonical active tree → consumer has migrated cleanly →
//     silent.
//   - Mixed active tree (some canonical alongside some narrow) →
//     warning fires on each narrow file. Effective message: "narrow-
//     width id detected in mixed-state active tree; run `aiwf rewidth`
//     to complete the migration."
//
// Archive entries are excluded from the mixed-state computation
// entirely per ADR-0008's "Drift control" subsection: their width
// never participates in the active-tree state assessment, and the
// rule never fires on them. Stubs are also excluded — their ids are
// path-derived and a parse failure is already its own finding.
//
// The width classification uses the on-disk filename's id segment,
// extracted via entity.IDFromPath. An entity whose path doesn't
// match the kind's expected shape is skipped (defensive — the loader
// already rejects such files; idPathConsistent reports the rest).
//
// The rule is the chokepoint that ADR-0008 §"Drift control" calls
// for. Its fixture-based tests live in entity_id_narrow_width_test.go
// (M-083/AC-1); the active-tree-clean assertion against this repo
// lives in internal/policies/this_repo_drift_check_clean_test.go
// (M-083/AC-5).
func entityIDNarrowWidth(t *tree.Tree) []Finding {
	type cls struct {
		entity *entity.Entity
		narrow bool
	}
	var active []cls
	for _, e := range t.Entities {
		if isArchivePath(e.Path) {
			continue
		}
		// ADR is exempt: its grammar (`ADR-\d{4,}`) was always at
		// canonical width; it has no narrow-legacy form and does not
		// participate in the migration. Including it would taint
		// otherwise uniform-narrow pre-migration trees as "mixed"
		// (e.g., E-01 + ADR-0001) which is not the signal ADR-0008
		// asks for. The rule cares about kinds that *had* a narrow
		// legacy width: E, M, G, D, C (and F when finding lands).
		if e.Kind == entity.KindADR {
			continue
		}
		// Determine width from the on-disk path segment, not from the
		// frontmatter id, so a tree mid-migration where one side leads
		// the other doesn't false-positive: the rule is about
		// what's-on-disk uniformity, which is what `aiwf rewidth`
		// changes.
		pathID, ok := entity.IDFromPath(e.Path, e.Kind)
		if !ok { //coverage:ignore defensive: every entity in t.Entities passed PathKind classification at load time, so IDFromPath matches by construction; the branch exists so a future loader-policy change doesn't silently classify mismatched entries
			continue
		}
		active = append(active, cls{entity: e, narrow: isNarrowID(pathID)})
	}

	// Classify the active set: count narrow vs canonical.
	var nNarrow, nCanonical int
	for _, c := range active {
		if c.narrow {
			nNarrow++
		} else {
			nCanonical++
		}
	}
	// Uniform (or empty) → silent.
	if nNarrow == 0 || nCanonical == 0 {
		return nil
	}
	// Mixed → emit a warning for every narrow active entry.
	var findings []Finding
	for _, c := range active {
		if !c.narrow {
			continue
		}
		findings = append(findings, Finding{
			Code:     "entity-id-narrow-width",
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"narrow-width id %q in mixed-state active tree (canonical width is %d digits per ADR-0008); run `aiwf rewidth` to complete the migration",
				c.entity.ID, entity.CanonicalPad),
			Path:     c.entity.Path,
			EntityID: c.entity.ID,
			Field:    "id",
		})
	}
	return findings
}

// isArchivePath reports whether path lives under any `<kind>/archive/`
// subtree. Per ADR-0008, archive entries never participate in the
// active-tree state assessment.
func isArchivePath(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, p := range parts {
		if p == "archive" {
			return true
		}
	}
	return false
}

// isNarrowID reports whether id's numeric portion is shorter than
// entity.CanonicalPad. ADR was always at canonical width, so its ids
// are never narrow by construction (Atoi over a `\d{4,}` numeric tail
// always yields ≥ pad characters); the predicate is therefore
// kind-agnostic and width-driven.
//
// An id that does not match the recognized prefix-digits shape passes
// through as not-narrow (defensive: an unrecognized id is not the
// rule's concern; frontmatter-shape will surface it).
func isNarrowID(id string) bool {
	for _, prefix := range []string{"ADR-", "E-", "M-", "G-", "D-", "C-", "F-"} {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		num := id[len(prefix):]
		if num == "" {
			return false
		}
		// Confirm the tail is digits-only (defensive — caller passes
		// path-extracted ids that already validate).
		for _, r := range num {
			if r < '0' || r > '9' {
				return false
			}
		}
		return len(num) < entity.CanonicalPad
	}
	return false
}
