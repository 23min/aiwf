package cliutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// treeload_test.go — in-process coverage for LoadTreeWithTrunk's
// G-0378/ADR-0031 wiring: the disputed-id-gated pair of rename-
// detection git calls and the trunk-side merge/inversion. The
// existing binary-level integration tests (internal/cli/integration)
// exercise this same wiring end-to-end, but as a subprocess — Go's
// coverage instrumentation cannot see across a subprocess boundary,
// so the diff-scoped coverage gate needs an in-process caller of
// LoadTreeWithTrunk itself.

// TestLoadTreeWithTrunk_MergesBranchAndTrunkSideRenames drives the
// full disputed-id path: a working tree with an entity id disputed
// against trunk (different path), where the exemption comes from the
// NEW trunk-side detector (a real trailer-stamped rename committed on
// trunk after the branch forked, never merged back). Covers the
// disputed>0 branch, both gitops calls, and the merge/inversion loop.
func TestLoadTreeWithTrunk_MergesBranchAndTrunkSideRenames(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "aiwf-test")

	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("aiwf_version: 0.1.0\nallocate:\n  trunk: refs/heads/trunk\n"), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	oldRel := "work/gaps/G-0035-original-slug.md"
	newRel := "work/gaps/G-0035-retitled-slug.md"
	if err := os.MkdirAll(filepath.Join(root, "work", "gaps"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Body sized so the retitle commit's per-commit `git show -M`
	// similarity clears the default 50% threshold — a tiny body makes
	// the title-line rewrite dominate the byte diff and drops
	// similarity below the threshold (see
	// trunk_side_rename_g0378_test.go's identical fixture note).
	body := `---
id: G-0035
title: original title
status: open
---

## Problem

Original body text describing the gap in enough detail that a
reviewer understands the diagnostic surface. Several lines of
prose here establish a moderate body size, so a single
title-line rewrite stays a small fraction of the total file
content when git computes rename similarity.

## Why it matters

A short section explaining impact, so the fixture body is
comfortably larger than a couple of lines.
`
	if err := os.WriteFile(filepath.Join(root, oldRel), []byte(body), 0o644); err != nil {
		t.Fatalf("write gap: %v", err)
	}
	runGit(t, root, "add", "aiwf.yaml", oldRel)
	runGit(t, root, "commit", "-q", "-m", "seed: gap on trunk")
	runGit(t, root, "branch", "trunk")

	// The feature branch (the current working tree LoadTreeWithTrunk
	// reads) forks here and never touches the gap again.
	runGit(t, root, "checkout", "-q", "-b", "feature")

	// Trunk retitles: a real trailer-stamped rename commit, landed
	// directly on "trunk", never merged back to "feature".
	runGit(t, root, "checkout", "-q", "trunk")
	runGit(t, root, "mv", oldRel, newRel)
	runGit(t, root, "commit", "-q", "-m",
		"aiwf retitle G-0035 -> retitled\n\naiwf-verb: retitle\naiwf-entity: G-0035\naiwf-actor: human/test")
	runGit(t, root, "checkout", "-q", "feature")

	tr, loadErrs, err := LoadTreeWithTrunk(ctx, root)
	if err != nil {
		t.Fatalf("LoadTreeWithTrunk: %v", err)
	}
	if len(loadErrs) != 0 {
		t.Fatalf("LoadTreeWithTrunk load errors: %+v", loadErrs)
	}
	got, ok := tr.TrunkCollisionRenames[newRel]
	if !ok || got != oldRel {
		t.Errorf("TrunkCollisionRenames[%q] = (%q, %v), want (%q, true) — the trunk-side detector's rename was not merged in with the expected key/value orientation", newRel, got, ok, oldRel)
	}
}

// TestLoadTreeWithTrunk_NoDisputeSkipsRenameDetection covers the
// common-case gate: when no working-tree id is disputed against
// trunk, TrunkCollisionRenames stays nil/empty — the git rename-
// detection calls are never reached (their absence isn't directly
// observable here, but a regression that made the gate fire on every
// push regardless of dispute would still leave this assertion
// trivially satisfied by design; the DisputedTrunkIDs unit tests own
// the "is this actually a dispute" predicate itself).
func TestLoadTreeWithTrunk_NoDisputeSkipsRenameDetection(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "aiwf-test")

	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("aiwf_version: 0.1.0\nallocate:\n  trunk: refs/heads/trunk\n"), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	rel := "work/gaps/G-0001-foo.md"
	if err := os.MkdirAll(filepath.Join(root, "work", "gaps"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, rel), []byte("---\nid: G-0001\ntitle: foo\nstatus: open\n---\nbody\n"), 0o644); err != nil {
		t.Fatalf("write gap: %v", err)
	}
	runGit(t, root, "add", "aiwf.yaml", rel)
	runGit(t, root, "commit", "-q", "-m", "seed")
	runGit(t, root, "branch", "trunk")

	tr, _, err := LoadTreeWithTrunk(ctx, root)
	if err != nil {
		t.Fatalf("LoadTreeWithTrunk: %v", err)
	}
	if len(tr.TrunkCollisionRenames) != 0 {
		t.Errorf("TrunkCollisionRenames = %+v, want empty (no id is disputed against trunk)", tr.TrunkCollisionRenames)
	}
}

// TestLoadTreeWithTrunk_PopulatesCrossBranchHits — M-0259/AC-2: a
// sibling local branch's entity, invisible to the working tree, must
// surface in tr.CrossBranchHits (and its bare id in tr.LocalRefIDs) so
// refs-resolve/body-prose-id can classify a reference to it as
// cross-branch-pending rather than unresolved.
func TestLoadTreeWithTrunk_PopulatesCrossBranchHits(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "aiwf-test")

	rel := "work/gaps/G-0001-foo.md"
	if err := os.MkdirAll(filepath.Join(root, "work", "gaps"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, rel), []byte("---\nid: G-0001\ntitle: foo\nstatus: open\n---\nbody\n"), 0o644); err != nil {
		t.Fatalf("write gap: %v", err)
	}
	runGit(t, root, "add", rel)
	runGit(t, root, "commit", "-q", "-m", "seed")

	// A sibling branch carries an id absent from the working tree
	// (main) entirely — visible only via LocalRefHits.
	runGit(t, root, "checkout", "-q", "-b", "sibling")
	siblingRel := "work/gaps/G-0005-bar.md"
	if err := os.WriteFile(filepath.Join(root, siblingRel), []byte("---\nid: G-0005\ntitle: bar\nstatus: open\n---\nbody\n"), 0o644); err != nil {
		t.Fatalf("write sibling gap: %v", err)
	}
	runGit(t, root, "add", siblingRel)
	runGit(t, root, "commit", "-q", "-m", "sibling: add G-0005")
	runGit(t, root, "checkout", "-q", "-") // back to whatever the init default branch was

	tr, _, err := LoadTreeWithTrunk(ctx, root)
	if err != nil {
		t.Fatalf("LoadTreeWithTrunk: %v", err)
	}
	var found bool
	for _, h := range tr.CrossBranchHits {
		if h.ID == "G-0005" {
			found = true
			if h.Ref != "refs/heads/sibling" {
				t.Errorf("hit.Ref = %q, want refs/heads/sibling", h.Ref)
			}
			if h.Path != siblingRel {
				t.Errorf("hit.Path = %q, want %q", h.Path, siblingRel)
			}
		}
	}
	if !found {
		t.Fatalf("CrossBranchHits = %+v, want a hit for sibling-only id G-0005", tr.CrossBranchHits)
	}
}

// TestCrossBranchEscalation_PendingThenUnresolved_M0259AC4 is the
// ADR-0030 escalation fixture: proof, not prose, that a
// cross-branch-pending classification is never a permanent softening
// of what is actually a dangling reference. A reference validated
// cross-branch-pending while its source branch exists must
// re-escalate to a hard unresolved once that branch disappears — and
// since nothing here is cached, that falls out of a plain re-run
// against a mutated repo, with no separate escalation-tracking
// mechanism to drift.
func TestCrossBranchEscalation_PendingThenUnresolved_M0259AC4(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "aiwf-test")

	// main carries a milestone whose parent epic (E-0099) does not
	// exist on this branch at all.
	mRel := "work/epics/E-0001-foo/M-0001-bar.md"
	if err := os.MkdirAll(filepath.Join(root, "work", "epics", "E-0001-foo"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	mBody := "---\nid: M-0001\ntitle: bar\nstatus: draft\nparent: E-0099\ntdd: none\n---\n\n## Goal\n\ngoal\n"
	if err := os.WriteFile(filepath.Join(root, mRel), []byte(mBody), 0o644); err != nil {
		t.Fatalf("write milestone: %v", err)
	}
	runGit(t, root, "add", mRel)
	runGit(t, root, "commit", "-q", "-m", "seed: milestone referencing E-0099")

	// A sibling branch mints E-0099 — the reference is real, just not
	// merged into main yet.
	runGit(t, root, "checkout", "-q", "-b", "sibling")
	eRel := "work/epics/E-0099-baz/epic.md"
	if err := os.MkdirAll(filepath.Join(root, "work", "epics", "E-0099-baz"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	eBody := "---\nid: E-0099\ntitle: baz\nstatus: proposed\n---\n\n## Goal\n\ngoal\n"
	if err := os.WriteFile(filepath.Join(root, eRel), []byte(eBody), 0o644); err != nil {
		t.Fatalf("write epic: %v", err)
	}
	runGit(t, root, "add", eRel)
	runGit(t, root, "commit", "-q", "-m", "sibling: mint E-0099")
	runGit(t, root, "checkout", "-q", "-") // back to whatever the init default branch was

	// Phase 1: sibling exists — the reference classifies
	// cross-branch-pending, not unresolved.
	tr, loadErrs, err := LoadTreeWithTrunk(ctx, root)
	if err != nil {
		t.Fatalf("LoadTreeWithTrunk (phase 1): %v", err)
	}
	findings := check.Run(tr, loadErrs)
	f := findRefsResolveFinding(findings, "M-0001")
	if f == nil {
		t.Fatalf("phase 1: no refs-resolve finding for M-0001; findings: %+v", findings)
	}
	if f.Subcode != "cross-branch-pending" {
		t.Errorf("phase 1: Subcode = %q, want cross-branch-pending (sibling branch still exists)", f.Subcode)
	}

	// Phase 2: sibling branch disappears (deleted, never merged) —
	// re-run against the mutated repo. Nothing here is cached, so the
	// same reference must now hard-fail unresolved.
	runGit(t, root, "branch", "-D", "sibling")

	tr2, loadErrs2, err := LoadTreeWithTrunk(ctx, root)
	if err != nil {
		t.Fatalf("LoadTreeWithTrunk (phase 2): %v", err)
	}
	findings2 := check.Run(tr2, loadErrs2)
	f2 := findRefsResolveFinding(findings2, "M-0001")
	if f2 == nil {
		t.Fatalf("phase 2: no refs-resolve finding for M-0001; findings: %+v", findings2)
	}
	if f2.Subcode != "unresolved" {
		t.Errorf("phase 2: Subcode = %q, want unresolved — the source branch is gone, this must re-escalate", f2.Subcode)
	}
	if f2.Severity != check.SeverityError {
		t.Errorf("phase 2: Severity = %q, want error (blocking)", f2.Severity)
	}
}

// findRefsResolveFinding returns the refs-resolve finding for
// entityID, or nil if none exists.
func findRefsResolveFinding(findings []check.Finding, entityID string) *check.Finding {
	for i := range findings {
		if findings[i].Code == check.CodeRefsResolve && findings[i].EntityID == entityID {
			return &findings[i]
		}
	}
	return nil
}

// runGit invokes git in dir with a fixed deterministic identity,
// fatal'ing the test on failure.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=aiwf-test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=aiwf-test", "GIT_COMMITTER_EMAIL=test@example.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
