package check

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// CodeEntityIDNarrowWidth is the finding code emitted by entityIDNarrowWidth.
// Typed per G-0129.
const CodeEntityIDNarrowWidth = "entity-id-narrow-width"

// entityIDNarrowWidth reports a narrow-width id in an active entity's
// on-disk filename — entities outside any `<kind>/archive/` subtree —
// as an error. Canonical width is the only legal width for an active
// entity, so a narrow one is a defect regardless of what its
// neighbours look like.
//
// Scope is the *filename*, not the frontmatter. A frontmatter `id:`
// narrower than its own canonical-width filename is not reported here
// and is not reported by idPathConsistent either, which compares the
// two sides canonicalized and so treats a width-only divergence as a
// match. Widening this rule to read both axes is a live question
// tracked in G-0532.
//
// Archive entries are excluded per ADR-0008's "Drift control"
// subsection, and the exclusion is permanent: no verb widens an id
// in place, so a repo that archived before adopting canonical width
// holds narrow ids under `<kind>/archive/` forever. That is why
// narrow read tolerance (entity.Canonicalize, the grep alternation,
// prior_ids resolution) is load-bearing for live cross-references
// into archived entities rather than a legacy concession. Stubs are
// excluded too — their ids are path-derived and a parse failure is
// already its own finding.
//
// ADR needs no exemption: entity.IDFromPath rejects any ADR path below
// canonical width, so a narrow ADR id never reaches the width test.
//
// Fixture tests live in entity_id_narrow_width_test.go; the
// active-tree-clean assertion against this repo lives in
// internal/policies/this_repo_drift_check_clean_test.go.
func entityIDNarrowWidth(t *tree.Tree) []Finding {
	var findings []Finding
	for _, e := range t.Entities {
		if isArchivePath(e.Path) {
			continue
		}
		// The loader admits filenames this call rejects: PathKind
		// classifies on `^<prefix>-\d+`, while IDFromPath additionally
		// applies the kind's grammar floor (two digits for epic, three
		// for gap, milestone, decision and contract, four for ADR). An id below
		// that floor loads, lands here, and is reported by
		// frontmatter-shape instead.
		pathID, ok := entity.IDFromPath(e.Path, e.Kind)
		if !ok {
			continue
		}
		if !isNarrowID(pathID) {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodeEntityIDNarrowWidth,
			Severity: SeverityError,
			// The path-derived id, not e.ID: the width tested is the
			// filename's, so quoting frontmatter would print a
			// canonical id and call it narrow whenever the two diverge
			// — which is exactly the file move the remediation names.
			Message: fmt.Sprintf(
				"narrow-width id %q in the active tree (canonical width is %d digits per ADR-0008); undo the hand-edit or file move that produced it",
				pathID, entity.CanonicalPad),
			Path:     e.Path,
			EntityID: e.ID,
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
