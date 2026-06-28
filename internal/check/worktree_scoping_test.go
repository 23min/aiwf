package check

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// M-0188 — pin that the aiwf loader / aiwf check does not descend into
// in-repo worktrees under .claude/worktrees/.
//
// Per ADR-0023, the default placement for ritual worktrees is an in-repo
// worktree under .claude/worktrees/<branch>/. Such a worktree is a full
// second checkout of the repo *inside the tree*, including its own
// work/... If the loader walked from the repo root into .claude/worktrees/,
// it would load duplicate entity files and report phantom id collisions —
// which would break aiwf check the moment the in-repo default took effect.
//
// tree.Load walks only the entity-bearing subdirectories
// (work/{epics,gaps,decisions,contracts}, docs/adr) relative to the root —
// never the repo root — so a nested checkout's work/ lives under .claude/,
// outside every walk root, and is never loaded. These tests pin that
// behavior at the aiwf check seam (tree.Load + check.Run), with a
// non-vacuity guard (AC-2) proving the assertion can catch a regression.

// worktreeScopingEpic is the shared entity body for the worktree-scoping tests:
// one epic id reused at multiple paths so a descending loader would
// produce an ids-unique collision.
const worktreeScopingEpic = `---
id: E-0001
title: A real epic in the active tree
status: proposed
---
`

// TestLoaderIgnoresNestedWorktreeCheckout — M-0188 AC-1.
//
// A fixture with a real epic under an in-scope walk root AND a
// byte-identical copy under .claude/worktrees/<branch>/work/... (same id
// E-0001) must yield zero ids-unique findings: the nested copy is never
// loaded, so it can never collide with the real entity.
func TestLoaderIgnoresNestedWorktreeCheckout(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// The real entity, under an in-scope walk root.
	mustWrite(t, root, "work/epics/E-0001-real/epic.md", worktreeScopingEpic)
	// A full second checkout under .claude/worktrees/<branch>/ — the
	// in-repo-worktree default placement (ADR-0023). Same id, same shape.
	// If the loader descended into .claude/worktrees/ it would load this
	// as a duplicate E-0001 and idsUnique would fire.
	mustWrite(t, root, ".claude/worktrees/epic-E-0046/work/epics/E-0001-real/epic.md", worktreeScopingEpic)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Non-vacuity guard: the real, in-scope entity must actually be
	// loaded. Without this, "no .claude/ entity" and "no collision" would
	// both pass trivially on an empty tree — a totally broken loader would
	// look like correct scoping.
	inScope := tr.ByID("E-0001")
	if inScope == nil {
		t.Fatal("real in-scope E-0001 was not loaded; loader produced an empty/degenerate tree")
	}
	if inScope.Path != "work/epics/E-0001-real/epic.md" {
		t.Errorf("E-0001 loaded from %q, want the in-scope work/epics path (not the .claude/worktrees copy)", inScope.Path)
	}

	// Direct loader assertion: nothing was loaded from under .claude/.
	for _, e := range tr.Entities {
		if strings.HasPrefix(e.Path, ".claude/") {
			t.Errorf("loader descended into .claude/: loaded %s (id %s)", e.Path, e.ID)
		}
	}

	// Check-seam assertion: no ids-unique collision, because the nested
	// copy never entered the tree.
	for _, f := range Run(tr, loadErrs) {
		if f.Code == CodeIDsUnique {
			t.Errorf("unexpected ids-unique finding (loader must ignore .claude/worktrees): %+v", f)
		}
	}
}

// TestInScopeDuplicateIDStillFires — M-0188 AC-2 (non-vacuity guard).
//
// Proves the ids-unique detector is live and AC-1's clean result is not
// vacuous: the SAME duplicate id, placed at an *in-scope* path (under
// work/epics/ rather than .claude/worktrees/), IS reported as a collision.
// So AC-1 passes specifically because .claude/worktrees/ is outside the
// loader's walk scope — if the loader were ever changed to descend, the
// nested copy would behave like this in-scope duplicate and AC-1 would go
// red.
func TestInScopeDuplicateIDStillFires(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	mustWrite(t, root, "work/epics/E-0001-real/epic.md", worktreeScopingEpic)
	// Same id, but in-scope (under work/epics/, not .claude/worktrees/).
	mustWrite(t, root, "work/epics/E-0001-dup/epic.md", worktreeScopingEpic)

	tr, loadErrs, err := tree.Load(t.Context(), root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	var found bool
	for _, f := range Run(tr, loadErrs) {
		if f.Code == CodeIDsUnique && f.EntityID == "E-0001" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected an ids-unique finding for in-scope duplicate E-0001; got none — the AC-1 guard would be vacuous")
	}
}
