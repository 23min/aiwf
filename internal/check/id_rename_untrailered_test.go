package check

import (
	"strings"
	"testing"

	codespkg "github.com/23min/aiwf/internal/codes"
)

// id_rename_untrailered_test.go — M-0160/AC-4: unit-level
// coverage of the new id-rename-untrailered kernel chokepoint
// rule per CLAUDE.md §"Id-collision resolution at merge time":
//
//	"a kernel-side check that flags 'id-renames missing
//	 reallocate trailers' would be the chokepoint to add"
//
// The rule fires when a commit between merge-base(HEAD, trunk)
// and HEAD renames an id-bearing entity file AND lacks a
// rename-class aiwf-verb trailer (retitle / rename / reallocate
// / archive / move per the closed set at
// internal/gitops/refs.go::renameVerbs).
//
// Catches the operator-discipline gap CLAUDE.md documents: an
// operator resolves a trunk-collision by inline `git mv`
// instead of `aiwf reallocate <id-or-path>`. The immediate
// trunk-collision finding clears (git's rename detection
// paired the move via G-0167's trailer-driven path or G-0109's
// cumulative-similarity fallback), but the kernel trailer
// history misses the renumber event: `aiwf history G-old`
// doesn't bridge to the new id, cross-references in body
// prose aren't rewritten, and any future check rule keyed on
// `aiwf-verb: reallocate` doesn't see the rename.
//
// The unit tests below pin the rule's input/output contract
// against in-memory fixtures (no git, no fixture trees). The
// real-git seam is pinned at the integration level in
// internal/cli/integration/id_rename_untrailered_scenarios_test.go.
//
// API shape pinned by these tests (locks GREEN's choices):
//
//   - CodeIDRenameUntrailered is the typed
//     codespkg.Code{ID: "id-rename-untrailered", Class: ...}
//     shape (mirrors CodeIsolationEscape at
//     internal/check/isolation_escape.go:35), NOT the bare-string
//     shape (CodeTrailerVerbUnknown at trailer_verb_unknown.go:32).
//     The dot-access `CodeIDRenameUntrailered.ID` at line :55-56
//     below requires the struct shape.
//
//   - The expected Class is codespkg.ClassBranchChoreography
//     (this rule polices the same trunk-collision-resolution
//     discipline the isolation-escape rule polices for AI-actor
//     commits — both belong to the ADR-0011 layer-4 carve-out).
//     The class is asserted explicitly in
//     TestIDRenameUntrailered_TypedCodeClassIsBranchChoreography.

// TestIDRenameUntrailered_TypedCodeClassIsBranchChoreography
// pins the typed Code shape AND the Class assignment, locking
// GREEN's choice so a quietly-flipped bare-string code or
// wrong-class assignment fails this test (reviewer S1 follow-up
// pre-GREEN).
func TestIDRenameUntrailered_TypedCodeClassIsBranchChoreography(t *testing.T) {
	t.Parallel()
	if CodeIDRenameUntrailered.ID != "id-rename-untrailered" {
		t.Errorf("CodeIDRenameUntrailered.ID = %q; want %q", CodeIDRenameUntrailered.ID, "id-rename-untrailered")
	}
	if CodeIDRenameUntrailered.Class != codespkg.ClassBranchChoreography {
		t.Errorf("CodeIDRenameUntrailered.Class = %q; want %q (ADR-0011 layer-4 carve-out; rule polices the trunk-collision-resolution discipline alongside isolation-escape)",
			CodeIDRenameUntrailered.Class, codespkg.ClassBranchChoreography)
	}
}

// TestIDRenameUntrailered_FiresOnUntrailedRecord pins the
// primary fire path: a single record produces a single finding.
func TestIDRenameUntrailered_FiresOnUntrailedRecord(t *testing.T) {
	t.Parallel()
	renames := []UntrailedIDRename{
		{
			SHA:     "abc1234",
			OldPath: "work/gaps/G-0001-original-slug.md",
			NewPath: "work/gaps/G-0001-new-slug.md",
			OldID:   "G-0001",
			NewID:   "G-0001",
		},
	}
	findings := RunIDRenameUntrailered(renames, nil)
	if len(findings) != 1 {
		t.Fatalf("RunIDRenameUntrailered: got %d findings; want 1\nfindings: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Code != CodeIDRenameUntrailered.ID {
		t.Errorf("Code = %q; want %q", f.Code, CodeIDRenameUntrailered.ID)
	}
	if f.Severity != SeverityWarning {
		t.Errorf("Severity = %q; want %q (M-0160/AC-4 lands as warning; future D-NNN may tighten)", f.Severity, SeverityWarning)
	}
	if !strings.Contains(f.Message, "G-0001") {
		t.Errorf("Message does not name the entity id; got %q", f.Message)
	}
	if !strings.Contains(f.Message, "G-0001-new-slug.md") {
		t.Errorf("Message does not name the new path; got %q", f.Message)
	}
}

// TestIDRenameUntrailered_SilentOnNilOrEmpty pins the no-input
// path: no records → no findings.
func TestIDRenameUntrailered_SilentOnNilOrEmpty(t *testing.T) {
	t.Parallel()
	if findings := RunIDRenameUntrailered(nil, nil); len(findings) != 0 {
		t.Errorf("RunIDRenameUntrailered(nil): got %d findings; want 0", len(findings))
	}
	if findings := RunIDRenameUntrailered([]UntrailedIDRename{}, nil); len(findings) != 0 {
		t.Errorf("RunIDRenameUntrailered(empty): got %d findings; want 0", len(findings))
	}
}

// TestIDRenameUntrailered_AckedSHAExempted pins the
// acknowledge-illegal silencing path (M-0160/AC-4 wires the
// same ackedSHAs map M-0159/AC-3 lifted for the other three
// ack-consuming rules). A record whose SHA appears in
// ackedSHAs is suppressed; same-shape per-SHA closed-set
// semantics.
func TestIDRenameUntrailered_AckedSHAExempted(t *testing.T) {
	t.Parallel()
	renames := []UntrailedIDRename{
		{
			SHA:     "abc1234",
			OldPath: "work/gaps/G-0001-original-slug.md",
			NewPath: "work/gaps/G-0001-new-slug.md",
			OldID:   "G-0001",
			NewID:   "G-0001",
		},
		{
			SHA:     "def5678",
			OldPath: "work/gaps/G-0002-other-slug.md",
			NewPath: "work/gaps/G-0002-other-new-slug.md",
			OldID:   "G-0002",
			NewID:   "G-0002",
		},
	}
	acked := map[string]bool{"abc1234": true}
	findings := RunIDRenameUntrailered(renames, acked)
	if len(findings) != 1 {
		t.Fatalf("RunIDRenameUntrailered with one acked SHA: got %d findings; want 1\nfindings: %+v", len(findings), findings)
	}
	// The surviving finding must be for the un-acked record.
	if !strings.Contains(findings[0].Message, "G-0002") {
		t.Errorf("surviving finding is not the un-acked one; message: %q", findings[0].Message)
	}
}

// TestIDRenameUntrailered_PerRecordFiring pins the
// no-aggregation contract (mirrors M-0106/AC-10): multiple
// records produce multiple findings, one per rename, never
// rolled up.
func TestIDRenameUntrailered_PerRecordFiring(t *testing.T) {
	t.Parallel()
	renames := []UntrailedIDRename{
		{SHA: "sha1", OldPath: "work/gaps/G-0001-a.md", NewPath: "work/gaps/G-0001-b.md", OldID: "G-0001", NewID: "G-0001"},
		{SHA: "sha2", OldPath: "work/gaps/G-0002-c.md", NewPath: "work/gaps/G-0002-d.md", OldID: "G-0002", NewID: "G-0002"},
		{SHA: "sha3", OldPath: "work/gaps/G-0003-e.md", NewPath: "work/gaps/G-0003-f.md", OldID: "G-0003", NewID: "G-0003"},
	}
	findings := RunIDRenameUntrailered(renames, nil)
	if len(findings) != 3 {
		t.Errorf("expected 3 findings (one per record); got %d", len(findings))
	}
}
