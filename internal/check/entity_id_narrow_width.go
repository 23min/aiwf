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

// entityIDNarrowWidth reports a narrow-width id carried by an active
// entity — entities outside any `<kind>/archive/` subtree — as an
// error. Canonical width is the only legal width for an active entity,
// so a narrow one is a defect regardless of what its neighbours look
// like.
//
// An entity carries its id on two axes, the on-disk filename and the
// frontmatter `id:`, and either can be narrow while the other is
// canonical. Both are in scope here, and each is judged independently,
// because this rule is the only one that tests an entity's own id
// width: idPathConsistent compares the two sides canonicalized, so a
// width-only divergence reads as a match to it, and frontmatterShape
// validates against the kind's grammar floor, which admits narrow ids
// permanently. One finding per narrow entity, not per narrow axis —
// one entity is one defect with one fix, and the message names
// whichever axis makes it actionable.
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
		// The loader admits filenames IDFromPath rejects: PathKind
		// classifies on `^<prefix>-\d+`, while IDFromPath additionally
		// applies the kind's grammar floor (two digits for epic, three
		// for gap, milestone, decision and contract, four for ADR).
		// A rejected filename yields "", which is not narrow, so the
		// path axis simply drops out. The two axes never gate each
		// other: an unreadable filename must not take a readable
		// frontmatter id down with it, or a narrow `id:` under a
		// sub-floor filename would go unreported by every rule.
		pathID, _ := entity.IDFromPath(e.Path, e.Kind)
		phrase := narrowIDPhrase(pathID, frontmatterWidthID(e))
		if phrase == "" {
			continue
		}
		findings = append(findings, Finding{
			Code:     CodeEntityIDNarrowWidth,
			Severity: SeverityError,
			Message: fmt.Sprintf(
				"narrow-width %s in the active tree (canonical width is %d digits per ADR-0008); undo the hand-edit or file move that produced it",
				phrase, entity.CanonicalPad),
			Path:     e.Path,
			EntityID: e.ID,
			Field:    "id",
		})
	}
	return findings
}

// frontmatterWidthID returns the entity's frontmatter id when it is
// well-formed for the kind, and "" when it is not.
//
// Width is only a meaningful question about an id the kind's grammar
// admits. Below that floor the id is malformed rather than narrow, and
// frontmatterShape reports it with a message that names the expected
// format — so returning "" here routes the case to the rule that
// explains it instead of stacking a second finding on top. This mirrors
// the filename axis, where entity.IDFromPath rejects a sub-floor path
// before the width test ever sees it.
func frontmatterWidthID(e *entity.Entity) string {
	if entity.ValidateID(e.Kind, e.ID) != nil {
		return ""
	}
	return e.ID
}

// narrowIDPhrase describes the narrow spelling an entity carries, ready
// to slot into the finding message after "narrow-width ". It returns ""
// when neither axis is narrow.
//
// It quotes only spellings that are actually narrow. Printing a
// canonical id and calling it narrow would contradict itself at the one
// seam an operator reads to locate the offending file, so which axis
// diverges decides which id is quoted — and when both are narrow at the
// same spelling, naming an axis is dropped, since that would imply the
// other side is clean.
func narrowIDPhrase(pathID, frontmatterID string) string {
	pathNarrow, fmNarrow := isNarrowID(pathID), isNarrowID(frontmatterID)
	switch {
	case !pathNarrow && !fmNarrow:
		return ""
	case pathNarrow && !fmNarrow:
		return fmt.Sprintf("filename id %q", pathID)
	case !pathNarrow && fmNarrow:
		return fmt.Sprintf("frontmatter id %q", frontmatterID)
	case pathID == frontmatterID:
		return fmt.Sprintf("id %q", pathID)
	default:
		// Both narrow and disagreeing: neither spelling locates the
		// other, so quote both. idPathConsistent reports the
		// disagreement itself — this rule reports only the width.
		return fmt.Sprintf("filename id %q and frontmatter id %q", pathID, frontmatterID)
	}
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
