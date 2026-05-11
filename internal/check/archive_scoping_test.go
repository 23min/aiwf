package check

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// M-0086 AC-4: existing shape and health rules skip archive per
// ADR-0004 §"`aiwf check` shape rules":
//
//	Shape and health rules skip archive entirely: acs-shape,
//	entity-body-empty-ac, acs-tdd-audit, acs-body-coherence,
//	milestone-done-incomplete-acs, unexpected-tree-file, etc.
//
// frontmatter-shape is included on the strength of the ADR's "etc."
// (the shape-and-health group is open) and the M-0084 discovery: the
// narrow-id archive fixture triggered frontmatter-shape, which the
// ADR's intent forbids.
//
// The tests below pair each named rule with a fixture that fires the
// rule on an active entity (positive control) and then asserts the
// same rule does not fire on the same authoring shape under archive/
// (the scoping invariant). Per CLAUDE.md "test the seam, not just the
// layer," the tests drive through tree.Load + check.Run end-to-end.

// TestArchiveScoping_FrontmatterShape — narrow-width id under
// archive must not trigger frontmatter-shape. Mirrors the M-0084
// rewidth-archive seam discovery.
func TestArchiveScoping_FrontmatterShape(t *testing.T) {
	root := t.TempDir()

	// Active gap with a missing-required-field shape — empty
	// status fires frontmatter-shape on active, must not on
	// archive per ADR-0004 §"Check shape rules".
	mustWrite(t, root, "work/gaps/G-0050-active.md", `---
id: G-0050
title: Active gap with empty status
status:
---
`)
	// Archived gap missing required `status` field — this would
	// normally fire frontmatter-shape on an active entity, and
	// must NOT fire on archive.
	mustWrite(t, root, "work/gaps/archive/G-0099-old.md", `---
id: G-0099
title: Archived with missing status
status:
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)

	// Active entity must fire frontmatter-shape (positive control).
	activeFired := false
	for _, f := range got {
		if f.Code == "frontmatter-shape" && f.EntityID == "G-0050" {
			activeFired = true
		}
	}
	if !activeFired {
		t.Errorf("expected frontmatter-shape on active malformed entity (positive control); got: %+v", got)
	}

	// No frontmatter-shape finding may target an archive path.
	for _, f := range got {
		if f.Code == "frontmatter-shape" && strings.Contains(f.Path, "archive/") {
			t.Errorf("frontmatter-shape fired on archive path %q (must skip per ADR-0004 §Check shape rules): %+v", f.Path, f)
		}
	}
}

// TestArchiveScoping_AcsShape — an archived milestone with a
// malformed AC must not trigger acs-shape findings. The ADR's
// shape-and-health group includes acs-shape explicitly.
func TestArchiveScoping_AcsShape(t *testing.T) {
	root := t.TempDir()

	// Active epic for the milestone parent ref.
	mustWrite(t, root, "work/epics/E-0001-active/epic.md", `---
id: E-0001
title: Active epic
status: active
---
`)
	// Archived epic + milestone with malformed AC ids (positions
	// don't match the AC-N convention). Active counterpart fires;
	// archive must not.
	mustWrite(t, root, "work/epics/archive/E-0099-old/epic.md", `---
id: E-0099
title: Old epic
status: done
---
`)
	mustWrite(t, root, "work/epics/archive/E-0099-old/M-0099-old.md", `---
id: M-0099
title: Archived milestone with malformed acs
status: done
parent: E-0099
acs:
  - id: AC-9
    title: Out-of-order ac id
    status: met
---
`)
	// Active milestone with the same malformed-acs shape — the
	// positive control.
	mustWrite(t, root, "work/epics/E-0001-active/M-0001-active.md", `---
id: M-0001
title: Active milestone with malformed acs
status: in_progress
parent: E-0001
acs:
  - id: AC-9
    title: Out-of-order ac id
    status: open
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)

	activeFired := false
	for _, f := range got {
		// acs-shape uses composite ids (M-NNN/AC-N).
		if f.Code == "acs-shape" && strings.HasPrefix(f.EntityID, "M-0001") {
			activeFired = true
		}
		if f.Code == "acs-shape" && strings.HasPrefix(f.EntityID, "M-0099") {
			t.Errorf("acs-shape fired on archived milestone (must skip): %+v", f)
		}
	}
	if !activeFired {
		t.Errorf("expected acs-shape on active malformed milestone (positive control); got: %+v", got)
	}
}

// TestArchiveScoping_AcsBodyCoherence — orphan/missing AC body
// headings on archived milestones must not fire.
func TestArchiveScoping_AcsBodyCoherence(t *testing.T) {
	root := t.TempDir()

	// Active epic.
	mustWrite(t, root, "work/epics/E-0001-active/epic.md", `---
id: E-0001
title: Active epic
status: active
---
`)
	mustWrite(t, root, "work/epics/archive/E-0099-old/epic.md", `---
id: E-0099
title: Old epic
status: done
---
`)
	// Active milestone with an AC declared in frontmatter but no
	// `### AC-1` heading in body — fires acs-body-coherence on
	// active, must not on archive.
	mustWrite(t, root, "work/epics/E-0001-active/M-0001-active.md", `---
id: M-0001
title: Active milestone
status: in_progress
parent: E-0001
acs:
  - id: AC-1
    title: Some ac
    status: open
---

## Acceptance criteria

(no AC-1 heading)
`)
	mustWrite(t, root, "work/epics/archive/E-0099-old/M-0099-old.md", `---
id: M-0099
title: Archived milestone with same shape
status: done
parent: E-0099
acs:
  - id: AC-1
    title: Some ac
    status: met
---

## Acceptance criteria

(no AC-1 heading)
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)

	activeFired := false
	for _, f := range got {
		if f.Code == "acs-body-coherence" && strings.HasPrefix(f.EntityID, "M-0001") {
			activeFired = true
		}
		if f.Code == "acs-body-coherence" && strings.HasPrefix(f.EntityID, "M-0099") {
			t.Errorf("acs-body-coherence fired on archived milestone (must skip): %+v", f)
		}
	}
	if !activeFired {
		t.Errorf("expected acs-body-coherence on active milestone (positive control); got: %+v", got)
	}
}

// TestArchiveScoping_AcsTDDAudit — an archived tdd: required
// milestone with an AC in `met` status but tdd_phase != done would
// fire acs-tdd-audit on active. On archive it must skip.
func TestArchiveScoping_AcsTDDAudit(t *testing.T) {
	root := t.TempDir()

	mustWrite(t, root, "work/epics/E-0001-active/epic.md", `---
id: E-0001
title: Active epic
status: active
---
`)
	mustWrite(t, root, "work/epics/archive/E-0099-old/epic.md", `---
id: E-0099
title: Old epic
status: done
---
`)
	// Active: tdd: required, AC met, tdd_phase missing — fires.
	mustWrite(t, root, "work/epics/E-0001-active/M-0001-active.md", `---
id: M-0001
title: Active milestone
status: in_progress
parent: E-0001
tdd: required
acs:
  - id: AC-1
    title: Some ac
    status: met
---

## Acceptance criteria

### AC-1 — Some ac

Body text.
`)
	// Archive: same shape, must not fire.
	mustWrite(t, root, "work/epics/archive/E-0099-old/M-0099-old.md", `---
id: M-0099
title: Archived milestone with same shape
status: done
parent: E-0099
tdd: required
acs:
  - id: AC-1
    title: Some ac
    status: met
---

## Acceptance criteria

### AC-1 — Some ac

Body text.
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)

	activeFired := false
	for _, f := range got {
		if f.Code == "acs-tdd-audit" && strings.HasPrefix(f.EntityID, "M-0001") {
			activeFired = true
		}
		if f.Code == "acs-tdd-audit" && strings.HasPrefix(f.EntityID, "M-0099") {
			t.Errorf("acs-tdd-audit fired on archived milestone (must skip): %+v", f)
		}
	}
	if !activeFired {
		t.Errorf("expected acs-tdd-audit on active milestone (positive control); got: %+v", got)
	}
}

// TestArchiveScoping_MilestoneDoneIncompleteACs — an archived done
// milestone whose ACs are not all met would fire on active. Must
// skip on archive.
func TestArchiveScoping_MilestoneDoneIncompleteACs(t *testing.T) {
	root := t.TempDir()

	mustWrite(t, root, "work/epics/E-0001-active/epic.md", `---
id: E-0001
title: Active epic
status: active
---
`)
	mustWrite(t, root, "work/epics/archive/E-0099-old/epic.md", `---
id: E-0099
title: Old epic
status: done
---
`)
	// Active milestone status: done with one AC still open.
	mustWrite(t, root, "work/epics/E-0001-active/M-0001-active.md", `---
id: M-0001
title: Active done milestone with open AC
status: done
parent: E-0001
acs:
  - id: AC-1
    title: Open ac
    status: open
---

## Acceptance criteria

### AC-1 — Open ac

Body text.
`)
	// Archive: same shape, must not fire.
	mustWrite(t, root, "work/epics/archive/E-0099-old/M-0099-old.md", `---
id: M-0099
title: Archived done milestone with open AC
status: done
parent: E-0099
acs:
  - id: AC-1
    title: Open ac
    status: open
---

## Acceptance criteria

### AC-1 — Open ac

Body text.
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)

	activeFired := false
	for _, f := range got {
		if f.Code == "milestone-done-incomplete-acs" && f.EntityID == "M-0001" {
			activeFired = true
		}
		if f.Code == "milestone-done-incomplete-acs" && f.EntityID == "M-0099" {
			t.Errorf("milestone-done-incomplete-acs fired on archived milestone (must skip): %+v", f)
		}
	}
	if !activeFired {
		t.Errorf("expected milestone-done-incomplete-acs on active milestone (positive control); got: %+v", got)
	}
}

// TestArchiveScoping_EntityBodyEmpty — an archived entity with empty
// load-bearing body sections would fire on active. Must skip on
// archive.
func TestArchiveScoping_EntityBodyEmpty(t *testing.T) {
	root := t.TempDir()

	// Active gap with empty body section — fires.
	mustWrite(t, root, "work/gaps/G-0050-active.md", `---
id: G-0050
title: Active gap with empty body
status: open
---

## What's missing

`)
	// Archive: same shape, must not fire.
	mustWrite(t, root, "work/gaps/archive/G-0099-old.md", `---
id: G-0099
title: Archived gap with empty body
status: addressed
---

## What's missing

`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)

	activeFired := false
	for _, f := range got {
		if f.Code == "entity-body-empty" && f.EntityID == "G-0050" {
			activeFired = true
		}
		if f.Code == "entity-body-empty" && f.EntityID == "G-0099" {
			t.Errorf("entity-body-empty fired on archived gap (must skip): %+v", f)
		}
	}
	if !activeFired {
		t.Errorf("expected entity-body-empty on active gap (positive control); got: %+v", got)
	}
}

// M-0086 AC-6: refsResolve seam. Per ADR-0004 §"Check shape rules":
//
//	Reference-validity (refs-resolve in internal/check/check.go):
//	id-form references in frontmatter resolve across both active
//	and archive directories. References from active → archived
//	ids are legal and unflagged. References from archive → active
//	ids are not linted (the active side is fine; the archive side
//	is out of scope for health rules).
//
// The active→archive direction is already covered by M-0084's
// TestRefsResolve_ResolvesArchivedTargets in check_test.go. This
// pin is the inverse: an archived entity whose own references
// don't resolve must NOT fire — refs-resolve skips archive
// entities as the source of references.

// TestArchiveScoping_RefsResolve_ArchiveSideNotLinted — an archived
// entity whose ref points at a non-existent id must not surface a
// refs-resolve finding. The archive-side references are out of
// scope for active-set health linting.
func TestArchiveScoping_RefsResolve_ArchiveSideNotLinted(t *testing.T) {
	root := t.TempDir()

	// Archived gap with addressed_by pointing at a non-existent
	// milestone (legal historical state — the milestone was
	// renamed/cancelled long ago and the archive entry was never
	// rewritten, per ADR-0004's forget-by-default).
	mustWrite(t, root, "work/gaps/archive/G-0099-old.md", `---
id: G-0099
title: Old gap
status: addressed
addressed_by:
  - M-9999
---
`)
	// Active gap with a similar broken reference — positive
	// control. refs-resolve must fire here.
	mustWrite(t, root, "work/gaps/G-0050-active.md", `---
id: G-0050
title: Active gap
status: addressed
addressed_by:
  - M-9999
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)

	activeFired := false
	for _, f := range got {
		if f.Code == "refs-resolve" && f.EntityID == "G-0050" {
			activeFired = true
		}
		if f.Code == "refs-resolve" && f.EntityID == "G-0099" {
			t.Errorf("refs-resolve fired on archived entity (must skip per ADR-0004): %+v", f)
		}
	}
	if !activeFired {
		t.Errorf("expected refs-resolve on active gap with broken reference (positive control); got: %+v", got)
	}
}

// M-0086 AC-5: tree-integrity rules traverse archive in full. Per
// ADR-0004 §"`aiwf check` shape rules":
//
//	Tree-integrity rules traverse archive in full: ids-unique
//	(id collision matters across active+archive), parse-level
//	errors (a malformed frontmatter is still a problem in
//	archive), and the new convergence findings introduced below.

// TestArchiveTreeIntegrity_IdsUniqueSpansActiveAndArchive — the same
// id existing in active and archive directories is still a
// collision. ids-unique must surface it.
func TestArchiveTreeIntegrity_IdsUniqueSpansActiveAndArchive(t *testing.T) {
	root := t.TempDir()

	// Active and archived gap with the same id — collision.
	mustWrite(t, root, "work/gaps/G-0050-active.md", `---
id: G-0050
title: Active gap
status: open
---
`)
	mustWrite(t, root, "work/gaps/archive/G-0050-old.md", `---
id: G-0050
title: Archived gap with same id
status: addressed
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)
	collisions := 0
	for _, f := range got {
		if f.Code == "ids-unique" && f.EntityID == "G-0050" {
			collisions++
		}
	}
	if collisions == 0 {
		t.Errorf("ids-unique must traverse archive (G-0050 exists active+archive); got: %+v", got)
	}
}

// TestArchiveTreeIntegrity_ParseErrorsTraverseArchive — a malformed
// frontmatter under archive/ must still surface as a load-error
// finding. Parse-level errors do not skip archive.
func TestArchiveTreeIntegrity_ParseErrorsTraverseArchive(t *testing.T) {
	root := t.TempDir()

	// Malformed YAML frontmatter on an archived gap — must surface.
	mustWrite(t, root, "work/gaps/archive/G-0099-old.md", `---
id: G-0099
title: "Unterminated string
status: addressed
---
`)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Run(tr, loadErrs)
	loadErrorFired := false
	for _, f := range got {
		if f.Code == "load-error" && strings.Contains(f.Path, "archive/") {
			loadErrorFired = true
		}
	}
	if !loadErrorFired {
		t.Errorf("load-error must traverse archive (malformed YAML); got: %+v", got)
	}
}

// TestArchiveScoping_UnexpectedTreeFile — a stray .md file under
// archive/ must not trigger unexpected-tree-file. The active-tree
// equivalent (in the same kind dir but not under archive) must
// still fire. The rule reads tree.Strays directly; M-0084's loader
// classifies recognized archive shapes as kind-bearing, but a true
// stray under archive (not matching any pattern) becomes a stray
// path with the archive prefix.
func TestArchiveScoping_UnexpectedTreeFile(t *testing.T) {
	root := t.TempDir()

	// Active stray (positive control).
	mustWrite(t, root, "work/gaps/notes.md", `Just a note.`)
	// Archive stray — should not fire under M-0086 scoping.
	mustWrite(t, root, "work/gaps/archive/historical-notes.md", `Old notes.`)

	tr, _, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// TreeDiscipline lives outside Run; call it directly.
	got := TreeDiscipline(tr, nil, false)

	activeFired := false
	for _, f := range got {
		if f.Code == "unexpected-tree-file" && strings.HasSuffix(f.Path, "notes.md") && !strings.Contains(f.Path, "archive/") {
			activeFired = true
		}
		if f.Code == "unexpected-tree-file" && strings.Contains(f.Path, "archive/") {
			t.Errorf("unexpected-tree-file fired on archive stray (must skip per ADR-0004): %+v", f)
		}
	}
	if !activeFired {
		t.Errorf("expected unexpected-tree-file on active stray (positive control); got: %+v", got)
	}
}
