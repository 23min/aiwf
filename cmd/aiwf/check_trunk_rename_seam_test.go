package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBinary_Check_TrunkRenameNotCollision is the end-to-end seam test
// for G-0109. It builds the real binary, sets up a consumer repo with
// a "trunk" ref carrying an entity at one slug, renames the entity
// on a feature branch, and asserts that `aiwf check` reports no
// trunk-collision finding for the renamed entity.
//
// The unit test (TestIDsUnique_GitRenameNotCollision) exercises the
// check rule alone with a hand-set TrunkRenames map; the gitops test
// (TestRenamesFromRef_DetectsCommittedRename) exercises the git
// helper alone. Neither catches the case where the dispatcher fails
// to wire the two together. This test does.
//
// Without this exception, the catch-22 the gap documents fires:
// pre-push runs `aiwf check`, the rename is misread as a duplicate id
// allocation, push is blocked, and the only escape is `--no-verify`.
func TestBinary_Check_TrunkRenameNotCollision(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp)

	repo := t.TempDir()
	mustExec(t, repo, "git", "init", "-q")
	mustExec(t, repo, "git", "config", "user.email", "test@example.com")
	mustExec(t, repo, "git", "config", "user.name", "aiwf-test")

	// aiwf.yaml points the trunk-allocator at a local "trunk" branch so
	// the cross-tree machinery activates without needing an actual
	// remote. The behavior under test is identical to a real
	// refs/remotes/origin/main setup; we just avoid the network.
	aiwfCfg := []byte("aiwf_version: 0.1.0\nallocate:\n  trunk: refs/heads/trunk\n")
	if err := os.WriteFile(filepath.Join(repo, "aiwf.yaml"), aiwfCfg, 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	// Commit a gap entity at its original long slug on the soon-to-be
	// trunk branch.
	oldRel := "work/gaps/G-0035-very-long-historical-slug-that-was-the-original-shape.md"
	gapBody := "---\nid: G-0035\ntitle: example gap for rename detection\nstatus: open\n---\n# Body\n\nSample.\n"
	if err := os.MkdirAll(filepath.Join(repo, "work", "gaps"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, oldRel), []byte(gapBody), 0o644); err != nil {
		t.Fatalf("write gap: %v", err)
	}
	mustExec(t, repo, "git", "add", "aiwf.yaml", oldRel)
	mustExec(t, repo, "git", "commit", "-q", "-m", "seed: gap on trunk")
	mustExec(t, repo, "git", "branch", "trunk")

	// On a feature branch, rename the gap to a short slug — the same
	// shape `aiwf rename` produces (one git mv + one commit). Use git
	// directly so the test stays focused on the check rule rather than
	// on aiwf rename's verb mechanics.
	mustExec(t, repo, "git", "checkout", "-q", "-b", "fix/slug-cleanup")
	newRel := "work/gaps/G-0035-short.md"
	mustExec(t, repo, "git", "mv", oldRel, newRel)
	mustExec(t, repo, "git", "commit", "-q", "-m", "rename G-0035 slug")

	// `aiwf check` against this branch must not report a trunk-
	// collision finding, because the branch-side path is a git-
	// detected rename of the trunk-side path. Pre-G-0109 the rule
	// fired here and blocked any rename-heavy cleanup at push time.
	out, err := runBinaryAt(repo, bin, "check", "--root", repo)
	if err != nil {
		t.Fatalf("aiwf check failed unexpectedly: %v\n%s", err, out)
	}
	if strings.Contains(out, "trunk-collision") {
		t.Errorf("aiwf check reported trunk-collision for a renamed entity (G-0109 regression):\n%s", out)
	}
}

// TestBinary_Check_NonRenameSameIDStillCollides pins the negative
// side of the G-0109 fix: when the trunk-side and branch-side paths
// are NOT a git-detected rename pair (e.g., the branch deleted the
// trunk file and created a fresh different entity at the same id),
// the trunk-collision finding must still fire. Otherwise the rename
// exception masks the genuine duplicate-id case the rule exists to
// catch.
func TestBinary_Check_NonRenameSameIDStillCollides(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp)

	repo := t.TempDir()
	mustExec(t, repo, "git", "init", "-q")
	mustExec(t, repo, "git", "config", "user.email", "test@example.com")
	mustExec(t, repo, "git", "config", "user.name", "aiwf-test")

	aiwfCfg := []byte("aiwf_version: 0.1.0\nallocate:\n  trunk: refs/heads/trunk\n")
	if err := os.WriteFile(filepath.Join(repo, "aiwf.yaml"), aiwfCfg, 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	// Trunk-side: a fully developed gap entity with body text.
	oldRel := "work/gaps/G-0050-trunk-side-entity.md"
	trunkBody := "---\nid: G-0050\ntitle: trunk-side\nstatus: open\n---\n# Body\n\nTrunk-side prose, distinctive content that git's similarity heuristic won't pair with a brand-new differently-worded file.\n"
	if err := os.MkdirAll(filepath.Join(repo, "work", "gaps"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, oldRel), []byte(trunkBody), 0o644); err != nil {
		t.Fatalf("write trunk gap: %v", err)
	}
	mustExec(t, repo, "git", "add", "aiwf.yaml", oldRel)
	mustExec(t, repo, "git", "commit", "-q", "-m", "seed: trunk gap")
	mustExec(t, repo, "git", "branch", "trunk")

	// Branch-side: delete trunk's file, create a wholly different gap
	// claiming the same id at an unrelated path. This is the case the
	// rule was designed to catch — git's rename detection should NOT
	// match these because the body content is too different.
	mustExec(t, repo, "git", "checkout", "-q", "-b", "feature/parallel-collision")
	mustExec(t, repo, "git", "rm", "-q", oldRel)
	newRel := "work/gaps/G-0050-branch-collision-entity.md"
	branchBody := "---\nid: G-0050\ntitle: branch-side\nstatus: open\n---\n# Body\n\nCompletely unrelated entity that happens to reuse the same id; this is exactly the duplicate-allocation case the trunk-collision rule exists to catch.\n"
	if err := os.MkdirAll(filepath.Join(repo, "work", "gaps"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, newRel), []byte(branchBody), 0o644); err != nil {
		t.Fatalf("write branch gap: %v", err)
	}
	mustExec(t, repo, "git", "add", newRel)
	mustExec(t, repo, "git", "commit", "-q", "-m", "feature: different G-0050")

	out, err := runBinaryAt(repo, bin, "check", "--root", repo)
	// Findings exit code is 1; usage error is 2; internal error is 3.
	// We expect findings here.
	if err == nil {
		t.Errorf("aiwf check should have reported findings for a true duplicate-id collision; got exit 0\n%s", out)
	}
	if !strings.Contains(out, "trunk-collision") {
		t.Errorf("expected trunk-collision finding for a non-rename same-id pair, got:\n%s", out)
	}
}
