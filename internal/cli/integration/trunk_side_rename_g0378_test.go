package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// trunk_side_rename_g0378_test.go — the binary-level seam tests for
// G-0378 / ADR-0031: idsUnique's trunk-collision rule gains a
// trailer-only, trunk-side rename detector so a rename landed *on
// trunk* (e.g. `aiwf retitle`) after a feature branch already forked
// away from it is recognized as the same entity moved, not a
// duplicate id allocation.
//
// TestBinary_Check_TrunkRenameNotCollision (check_trunk_rename_seam_test.go)
// and TestTrunkRenameScenarios_AC2_G0167TrailerDrivenRescue
// (trunk_rename_g0167_test.go) already pin the BRANCH-side rescue
// (G-0109/G-0167: a rename the branch itself committed). Neither
// covers the reverse direction this gap fixes, nor the negative
// control confirming a genuine collision still fires through the new
// wiring. ADR-0031's Validation section requires both as fixture
// tests; this file provides them.

// TestBinary_Check_TrunkSideRetitleAfterForkNotCollision is
// ADR-0031's primary validation fixture: trunk retitles an entity
// (a real `aiwf retitle`-shaped commit, trailer and all) after a
// branch has already forked away from it, and the branch never
// merges the retitle back. `aiwf check` on the branch must not fire
// trunk-collision — RenamesFromRef's branch-scoped walk can't see
// this rename (it never happened on the branch's own history); only
// the new trunk-scoped detector (gitops.TrunkRenamesFromRef, wired
// through cliutil.LoadTreeWithTrunk's DisputedTrunkIDs-gated call)
// can.
func TestBinary_Check_TrunkSideRetitleAfterForkNotCollision(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	bin := testutil.AiwfBinary(t)

	repo := t.TempDir()
	testutil.MustExec(t, repo, "git", "init", "-q", "-b", "main")
	testutil.MustExec(t, repo, "git", "config", "user.email", "test@example.com")
	testutil.MustExec(t, repo, "git", "config", "user.name", "aiwf-test")

	// aiwf.yaml points the trunk-allocator at a local "trunk" branch,
	// same as the G-0109 seam test — no real remote needed.
	aiwfCfg := []byte("aiwf_version: 0.1.0\nallocate:\n  trunk: refs/heads/trunk\n")
	if err := os.WriteFile(filepath.Join(repo, "aiwf.yaml"), aiwfCfg, 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	// Seed the gap at its original slug and commit on what becomes
	// both "main" (the eventual feature branch) and "trunk".
	oldRel := "work/gaps/G-0035-very-long-historical-slug-that-was-the-original-shape.md"
	// The body must be large enough that the retitle commit's
	// per-commit `git show -M` similarity clears the default -M50
	// threshold — renamesInCommit (shared by both the branch-side and
	// trunk-side trailer walks) still relies on git's own per-commit
	// rename heuristic to extract the old/new path pair once the
	// trailer has identified WHICH commit to inspect. A tiny body
	// makes the title-line rewrite dominate the file's byte diff and
	// drops per-commit similarity below 50%, exactly the pitfall
	// trunk_rename_g0167_test.go's seedBodyAC2 fixture documents.
	gapBody := `---
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
	if err := os.MkdirAll(filepath.Join(repo, "work", "gaps"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, oldRel), []byte(gapBody), 0o644); err != nil {
		t.Fatalf("write gap: %v", err)
	}
	testutil.MustExec(t, repo, "git", "add", "aiwf.yaml", oldRel)
	testutil.MustExec(t, repo, "git", "commit", "-q", "-m", "seed: gap on trunk")
	testutil.MustExec(t, repo, "git", "branch", "trunk")

	// The feature branch forks here and never touches the gap again.
	testutil.MustExec(t, repo, "git", "checkout", "-q", "-b", "feature/g0378-trunk-retitle")

	// Trunk retitles: check out "trunk", run the REAL `aiwf retitle`
	// verb (not a fabricated trailer) so the fixture proves the full
	// verb -> gitops -> check wiring, then return to the feature
	// branch without merging the retitle back.
	testutil.MustExec(t, repo, "git", "checkout", "-q", "trunk")
	out, err := testutil.RunBinaryAt(repo, bin, "retitle", "G-0035",
		"Trunk gap retitled to a longer title after the branch already forked",
		"--reason", "G-0378 fixture: retitle on trunk after a branch already forked")
	if err != nil {
		t.Fatalf("aiwf retitle failed: %v\n%s", err, out)
	}
	testutil.MustExec(t, repo, "git", "checkout", "-q", "feature/g0378-trunk-retitle")

	out, err = testutil.RunBinaryAt(repo, bin, "check", "--root", repo)
	if err != nil {
		t.Fatalf("aiwf check failed unexpectedly: %v\n%s", err, out)
	}
	if strings.Contains(out, "trunk-collision") {
		t.Errorf("aiwf check reported trunk-collision for a trunk-side retitle the branch never merged back (G-0378 regression):\n%s", out)
	}
}

// TestBinary_Check_GenuineCollisionHighSimilarityStillFires is
// ADR-0031's negative control: two entities independently allocated
// the same id at different paths (the G37 shape) — trunk gets one,
// the branch gets the other, from a shared ancestor, with
// near-identical template bodies. Neither side ever renamed
// anything, so no trailer connects them. The rule must still fire;
// proving this through the real binary confirms the new trunk-side
// detector's dispatcher wiring doesn't widen the rule's exemption and
// mask a genuine collision it has no rename evidence for.
func TestBinary_Check_GenuineCollisionHighSimilarityStillFires(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	bin := testutil.AiwfBinary(t)

	repo := t.TempDir()
	testutil.MustExec(t, repo, "git", "init", "-q", "-b", "main")
	testutil.MustExec(t, repo, "git", "config", "user.email", "test@example.com")
	testutil.MustExec(t, repo, "git", "config", "user.name", "aiwf-test")

	aiwfCfg := []byte("aiwf_version: 0.1.0\nallocate:\n  trunk: refs/heads/trunk\n")
	if err := os.WriteFile(filepath.Join(repo, "aiwf.yaml"), aiwfCfg, 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("shared\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	testutil.MustExec(t, repo, "git", "add", "aiwf.yaml", "README.md")
	testutil.MustExec(t, repo, "git", "commit", "-q", "-m", "seed: shared ancestor")

	// "trunk" is created here without switching to it — the implicit
	// default branch from `git init` is never referenced by name
	// (it varies by host git config: "main" or "master"), mirroring
	// check_trunk_rename_seam_test.go's convention.
	testutil.MustExec(t, repo, "git", "branch", "trunk")

	// A feature branch forks from the same shared commit and
	// independently allocates G-0001 at one slug, with a
	// near-identical template body. Both sides only ADD their file
	// (no matching delete anywhere in range), so this isn't actually
	// a shape `-M` could ever pair as a rename regardless of body
	// similarity — that specific "-M would falsely pair similar
	// bodies" risk is what
	// TestTrunkRenamesFromRef_NoDashMFallback_ManualGitMvNotDetected
	// pins instead (a real add+delete pair, identical content, that
	// -M would trivially catch). This test's claim is narrower and
	// still worth pinning end-to-end: a genuine independent id
	// collision, with no rename relationship at all, must still fire
	// through the real binary once the trunk-side detector is wired
	// in.
	testutil.MustExec(t, repo, "git", "checkout", "-q", "-b", "feature/g0378-collision")
	if err := os.MkdirAll(filepath.Join(repo, "work", "gaps"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	branchRel := "work/gaps/G-0001-branch-side.md"
	if err := os.WriteFile(filepath.Join(repo, branchRel), []byte("---\nid: G-0001\ntitle: branch side\nstatus: open\n---\n\n## Problem\n\n"), 0o644); err != nil {
		t.Fatalf("write branch gap: %v", err)
	}
	testutil.MustExec(t, repo, "git", "add", branchRel)
	testutil.MustExec(t, repo, "git", "commit", "-q", "-m", "branch: add G-0001")

	// Trunk independently allocates the SAME id at a different slug.
	// Checking out "trunk" resets the working tree to its own commit
	// history, which doesn't yet have work/gaps/ (only the feature
	// branch committed that directory) — recreate it here.
	testutil.MustExec(t, repo, "git", "checkout", "-q", "trunk")
	if err := os.MkdirAll(filepath.Join(repo, "work", "gaps"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	trunkRel := "work/gaps/G-0001-trunk-side.md"
	if err := os.WriteFile(filepath.Join(repo, trunkRel), []byte("---\nid: G-0001\ntitle: trunk side\nstatus: open\n---\n\n## Problem\n\n"), 0o644); err != nil {
		t.Fatalf("write trunk gap: %v", err)
	}
	testutil.MustExec(t, repo, "git", "add", trunkRel)
	testutil.MustExec(t, repo, "git", "commit", "-q", "-m", "trunk: add G-0001")
	testutil.MustExec(t, repo, "git", "checkout", "-q", "feature/g0378-collision")

	out, err := testutil.RunBinaryAt(repo, bin, "check", "--root", repo)
	if err == nil {
		t.Fatalf("aiwf check succeeded, want trunk-collision finding:\n%s", out)
	}
	if !strings.Contains(out, "trunk-collision") {
		t.Errorf("aiwf check did not report trunk-collision for a genuine independent id collision (G-0378 false-negative risk):\n%s", out)
	}
}
