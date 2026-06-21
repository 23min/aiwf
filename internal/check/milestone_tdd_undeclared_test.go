package check

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// findingsWithCode returns the subset of findings carrying the given
// bare code (subcode ignored). Test-local helper so the assertions
// below read as "did <code> fire for <entity>".
func findingsWithCode(got []Finding, code string) []Finding {
	var out []Finding
	for i := range got {
		if got[i].Code == code {
			out = append(out, got[i])
		}
	}
	return out
}

// TestMilestoneTDDUndeclared_FiresOnActiveMissingTDD pins G-0268: an
// active milestone whose frontmatter lacks `tdd:` produces a
// `milestone-tdd-undeclared` warning, and a milestone that declares
// any closed-set policy clears it. The hard `--tdd` refusal at
// creation is the chokepoint; this rule is the defense-in-depth
// backstop for the paths the verb can't see (hand-edit that strips
// the field, import bypass).
func TestMilestoneTDDUndeclared_FiresOnActiveMissingTDD(t *testing.T) {
	t.Parallel()

	t.Run("missing tdd fires warning", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mustWrite(t, root, "work/epics/E-0001-active/epic.md", `---
id: E-0001
title: Active epic
status: active
---

## Goal

Parent epic for the fixture.
`)
		mustWrite(t, root, "work/epics/E-0001-active/M-0001-no-tdd.md", `---
id: M-0001
title: Milestone with no tdd policy
status: in_progress
parent: E-0001
---

## Goal

A milestone whose frontmatter omits the tdd field.
`)
		tr, loadErrs, err := tree.Load(t.Context(), root)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		got := Run(tr, loadErrs)
		fired := findingsWithCode(got, "milestone-tdd-undeclared")
		if len(fired) != 1 {
			t.Fatalf("expected exactly one milestone-tdd-undeclared finding, got %d: %+v", len(fired), fired)
		}
		f := fired[0]
		if f.EntityID != "M-0001" {
			t.Errorf("finding EntityID = %q, want M-0001", f.EntityID)
		}
		if f.Severity != SeverityWarning {
			t.Errorf("finding Severity = %q, want warning", f.Severity)
		}
		if !strings.Contains(f.Path, "M-0001") {
			t.Errorf("finding Path = %q, want it to locate the milestone file", f.Path)
		}
	})

	for _, policy := range []string{"required", "advisory", "none"} {
		t.Run("tdd "+policy+" clears the finding", func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			mustWrite(t, root, "work/epics/E-0001-active/epic.md", `---
id: E-0001
title: Active epic
status: active
---

## Goal

Parent epic for the fixture.
`)
			mustWrite(t, root, "work/epics/E-0001-active/M-0001-with-tdd.md", `---
id: M-0001
title: Milestone with a tdd policy
status: in_progress
parent: E-0001
tdd: `+policy+`
---

## Goal

A milestone that declares its tdd policy explicitly.
`)
			tr, loadErrs, err := tree.Load(t.Context(), root)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			got := Run(tr, loadErrs)
			if fired := findingsWithCode(got, "milestone-tdd-undeclared"); len(fired) != 0 {
				t.Errorf("tdd: %s must clear milestone-tdd-undeclared, got: %+v", policy, fired)
			}
		})
	}
}

// TestMilestoneTDDUndeclared_ArchiveScoped pins the ADR-0004
// shape-and-health archive-scoping invariant (the M-0086 pattern):
// an archived milestone lacking `tdd:` must NOT fire — the active
// counterpart is the positive control. The 61 grandfathered done
// milestones in the real tree are all archived, so this is what keeps
// the rule silent on them.
func TestMilestoneTDDUndeclared_ArchiveScoped(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, root, "work/epics/E-0001-active/epic.md", `---
id: E-0001
title: Active epic
status: active
---

## Goal

Parent epic for the active control.
`)
	// Active milestone lacking tdd — positive control, must fire.
	mustWrite(t, root, "work/epics/E-0001-active/M-0001-active.md", `---
id: M-0001
title: Active milestone with no tdd
status: in_progress
parent: E-0001
---

## Goal

Active control.
`)
	// Archived epic + done milestone lacking tdd — must NOT fire.
	mustWrite(t, root, "work/epics/archive/E-0099-old/epic.md", `---
id: E-0099
title: Old epic
status: done
---

## Goal

Archived parent.
`)
	mustWrite(t, root, "work/epics/archive/E-0099-old/M-0099-old.md", `---
id: M-0099
title: Archived milestone with no tdd
status: done
parent: E-0099
---

## Goal

Grandfathered, archived; must stay silent.
`)
	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := Run(tr, loadErrs)

	fired := findingsWithCode(got, "milestone-tdd-undeclared")
	activeFired := false
	for _, f := range fired {
		if strings.Contains(f.Path, "archive/") {
			t.Errorf("milestone-tdd-undeclared fired on archive path %q (must skip per ADR-0004 §Check shape rules): %+v", f.Path, f)
		}
		if f.EntityID == "M-0001" {
			activeFired = true
		}
	}
	if !activeFired {
		t.Errorf("expected milestone-tdd-undeclared on active M-0001 (positive control); got: %+v", fired)
	}
}

// TestMilestoneTDDUndeclared_NoRetroactiveACsTDDAudit pins the
// grandfather property from G-0055/G-0268: a milestone surfacing
// milestone-tdd-undeclared must NOT have its already-met ACs
// retroactively re-audited by acs-tdd-audit. Mirrors the historical
// E-0014 shape — every AC `met`, no `tdd_phase`, no `tdd:` field. The
// two rules are independent: acs-tdd-audit only runs under tdd:
// required|advisory, so an absent tdd: produces zero acs-tdd-audit
// findings while still surfacing the undeclared warning.
func TestMilestoneTDDUndeclared_NoRetroactiveACsTDDAudit(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, root, "work/epics/E-0001-active/epic.md", `---
id: E-0001
title: Active epic
status: active
---

## Goal

Parent epic.
`)
	mustWrite(t, root, "work/epics/E-0001-active/M-0001-grandfathered.md", `---
id: M-0001
title: Grandfathered milestone shape
status: done
parent: E-0001
acs:
  - id: AC-1
    title: First criterion met without a recorded phase
    status: met
  - id: AC-2
    title: Second criterion met without a recorded phase
    status: met
---

## Goal

Mirrors the historical E-0014 shape: met ACs, no tdd_phase, no tdd.

## Acceptance criteria

### AC-1 — First criterion met without a recorded phase

Body.

### AC-2 — Second criterion met without a recorded phase

Body.
`)
	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := Run(tr, loadErrs)

	if fired := findingsWithCode(got, "milestone-tdd-undeclared"); len(fired) != 1 {
		t.Errorf("expected one milestone-tdd-undeclared on the grandfathered milestone, got %d: %+v", len(fired), fired)
	}
	if audit := findingsWithCode(got, CodeACsTDDAudit); len(audit) != 0 {
		t.Errorf("acs-tdd-audit must not retroactively engage on a tdd-absent milestone, got: %+v", audit)
	}
}
