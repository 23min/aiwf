package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// G37 — real-scenario integration tests using a local bare repo as
// the shared "origin." Two clones simulate two operators on two
// laptops; pushes and fetches flow through the bare repo exactly as
// they would across a network. No mocks, no synthetic update-refs.
//
// The setup pattern in every test:
//
//   1. makeBareOrigin() creates an empty bare repo.
//   2. makeClone(bare, "A") clones it into a working copy and sets
//      a per-clone git identity so commits are attributable.
//   3. The first clone runs `aiwf init` and pushes; subsequent clones
//      pull that initial state.
//   4. Each scenario then drives the clones through the conflict
//      pathway under test.
//
// These tests depend on `git`, `sh`, and a writable temp dir. They
// cost a few seconds each because they exec a real binary against
// real git plumbing; that cost is the point — anything cheaper would
// be testing the wrong thing.

// makeBareOrigin creates an empty bare repository in a temp dir and
// returns its absolute path. It is the "server" the clones talk to.
func makeBareOrigin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bare := filepath.Join(dir, "origin.git")
	cmd := exec.Command("git", "init", "--bare", "-q", "-b", "main", bare)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	return bare
}

// makeClone clones bare into a fresh working dir named clone-<name>
// and configures a unique git identity so the per-clone commits are
// attributable. Returns the clone's absolute path.
func makeClone(t *testing.T, bare, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "clone-"+name)
	cmd := exec.Command("git", "clone", "-q", bare, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone %s: %v\n%s", bare, err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", strings.ToLower(name) + "@example.com"},
		{"config", "user.name", "Clone " + name},
	} {
		if out, err := runGit(dir, args...); err != nil {
			t.Fatalf("git config in %s: %v\n%s", dir, err, out)
		}
	}
	return dir
}

// aiwfInitClone runs `aiwf init` in clone, commits any resulting
// changes (so the clone's main has the framework files), and pushes
// to origin so siblings can pull the same starting state. The
// initial push uses --set-upstream so subsequent pulls/fetches don't
// need explicit refs.
func aiwfInitClone(t *testing.T, clone, binDir string) {
	t.Helper()
	if out, err := runBin(t, clone, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init in %s: %v\n%s", clone, err, out)
	}
	// aiwf init does not create a commit, so commit the framework
	// files and push them to origin.
	if out, err := runGit(clone, "add", "-A"); err != nil {
		t.Fatalf("git add -A: %v\n%s", err, out)
	}
	if out, err := runGit(clone, "commit", "-q", "-m", "aiwf init"); err != nil {
		t.Fatalf("git commit init: %v\n%s", err, out)
	}
	if out, err := runGit(clone, "push", "-q", "-u", "origin", "main"); err != nil {
		t.Fatalf("git push -u origin main: %v\n%s", err, out)
	}
}

// makeSiblingClone is makeClone after another clone has already
// pushed `aiwf init` to bare. The new clone inherits the framework
// files via the clone itself.
func makeSiblingClone(t *testing.T, bare, name string) string {
	t.Helper()
	return makeClone(t, bare, name)
}

// aiwfAddGap runs `aiwf add gap --title <title>` in clone and returns
// the combined output. Fails the test on a non-zero exit so each
// caller sees the failure point clearly.
func aiwfAddGap(t *testing.T, clone, binDir, title string) string {
	t.Helper()
	out, err := runBin(t, clone, binDir, nil, "add", "gap", "--title", title)
	if err != nil {
		t.Fatalf("aiwf add gap %q in %s: %v\n%s", title, clone, err, out)
	}
	return out
}

// aiwfAddADR is aiwfAddGap for ADRs. Its own helper so callers don't
// have to remember which kind needs which flags.
func aiwfAddADR(t *testing.T, clone, binDir, title string) string {
	t.Helper()
	out, err := runBin(t, clone, binDir, nil, "add", "adr", "--title", title)
	if err != nil {
		t.Fatalf("aiwf add adr %q in %s: %v\n%s", title, clone, err, out)
	}
	return out
}

// aiwfAddMilestone adds a milestone under the given epic id.
func aiwfAddMilestone(t *testing.T, clone, binDir, title, epicID string) string {
	t.Helper()
	out, err := runBin(t, clone, binDir, nil, "add", "milestone", "--title", title, "--epic", epicID)
	if err != nil {
		t.Fatalf("aiwf add milestone %q in %s: %v\n%s", title, clone, err, out)
	}
	return out
}

// aiwfAddEpic adds an epic; useful to seed a parent for milestones.
func aiwfAddEpic(t *testing.T, clone, binDir, title string) string {
	t.Helper()
	out, err := runBin(t, clone, binDir, nil, "add", "epic", "--title", title)
	if err != nil {
		t.Fatalf("aiwf add epic %q in %s: %v\n%s", title, clone, err, out)
	}
	return out
}

// aiwfCheck runs `aiwf check` in clone and returns the combined
// output plus the exec error (which is non-nil whenever findings or
// errors are present, since the binary signals findings via exit code).
func aiwfCheck(t *testing.T, clone, binDir string) (string, error) {
	t.Helper()
	return runBin(t, clone, binDir, nil, "check")
}

// pushAll pushes the current branch in clone to origin. Fails the
// test on non-fast-forward unless the caller expects the rejection
// (in which case the caller invokes the underlying runGit directly).
func pushAll(t *testing.T, clone string) {
	t.Helper()
	if out, err := runGit(clone, "push", "-q", "origin", "main"); err != nil {
		t.Fatalf("git push origin main from %s: %v\n%s", clone, err, out)
	}
}

// fetchOrigin runs `git fetch origin` in clone — updates
// refs/remotes/origin/* without touching the working tree.
func fetchOrigin(t *testing.T, clone string) {
	t.Helper()
	if out, err := runGit(clone, "fetch", "-q", "origin"); err != nil {
		t.Fatalf("git fetch in %s: %v\n%s", clone, err, out)
	}
}

// TestIntegrationG37_AllocatorSkipsTrunkAfterFetch is the canonical
// "forgot to pull" scenario the trunk-aware allocator solves:
//
//  1. Clone A adds G-001 and pushes. Origin now has G-001.
//  2. Clone B has not done any aiwf work yet. It runs `git fetch`
//     so refs/remotes/origin/main on B sees A's G-001, but does
//     NOT merge — B's working branch is still at the pre-G-001
//     commit.
//  3. Clone B runs `aiwf add gap`. Without trunk awareness the
//     allocator would scan B's working tree only (no gaps yet) and
//     pick G-001, setting up a future collision. With trunk
//     awareness the allocator sees G-001 in origin/main and skips
//     to G-002.
//
// This test fails only if the cmd dispatcher's trunk wiring is
// genuinely absent — exactly what the unit test of AllocateID could
// not catch on its own.
func TestIntegrationG37_AllocatorSkipsTrunkAfterFetch(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	bare := makeBareOrigin(t)
	cloneA := makeClone(t, bare, "A")
	aiwfInitClone(t, cloneA, binDir)

	cloneB := makeSiblingClone(t, bare, "B")

	aiwfAddGap(t, cloneA, binDir, "Cache eviction is broken")
	if out, err := runGit(cloneA, "add", "-A"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	pushAll(t, cloneA)

	fetchOrigin(t, cloneB)

	out := aiwfAddGap(t, cloneB, binDir, "B's first gap")
	if !strings.Contains(out, "G-002") {
		t.Errorf("expected B to allocate G-002 after fetching A's G-001 from trunk; got:\n%s", out)
	}
	if !strings.Contains(out, "G-002") || strings.Contains(out, "added gap G-001") {
		t.Errorf("allocator did not consult trunk; output:\n%s", out)
	}
	gapDir := filepath.Join(cloneB, "work", "gaps")
	entries, err := os.ReadDir(gapDir)
	if err != nil {
		t.Fatalf("reading %s: %v", gapDir, err)
	}
	var gotPaths []string
	for _, e := range entries {
		gotPaths = append(gotPaths, e.Name())
	}
	hasG002 := false
	for _, p := range gotPaths {
		if strings.HasPrefix(p, "G-002-") {
			hasG002 = true
			break
		}
	}
	if !hasG002 {
		t.Errorf("expected G-002-*.md in %s; found %v", gapDir, gotPaths)
	}
}

// TestIntegrationG37_DivergedBranchesCaughtByCheckPostFetch covers
// the residual case the design names: two clones diverged from the
// same trunk SHA, both allocate the same id locally, the first to
// push wins. The second must catch the collision *before* attempting
// to merge — i.e., right after `git fetch` brings A's G-001 into B's
// refs/remotes/origin/main while B's working tree still has its own
// G-001 at a different path.
//
// This scenario fails the regular intra-tree ids-unique check on its
// own (B's working tree alone has only one G-001 at one path —
// nothing to flag yet). The trunk-aware variant of ids-unique is
// what makes the collision visible at this stage. The pre-push hook
// on B's branch then fails before the colliding push ships.
func TestIntegrationG37_DivergedBranchesCaughtByCheckPostFetch(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	bare := makeBareOrigin(t)
	cloneA := makeClone(t, bare, "A")
	aiwfInitClone(t, cloneA, binDir)

	cloneB := makeSiblingClone(t, bare, "B")

	// Both clones are at the same trunk SHA. Each independently
	// allocates G-001 locally. No fetch between the two adds.
	aiwfAddGap(t, cloneA, binDir, "A's gap")
	aiwfAddGap(t, cloneB, binDir, "B's gap")

	// A pushes first; origin/main now carries A's G-001.
	pushAll(t, cloneA)

	// B fetches A's update but does not merge. B's working tree
	// still has B's G-001 at its own slug-derived path.
	fetchOrigin(t, cloneB)

	// B's aiwf check should now flag the cross-tree collision.
	out, err := aiwfCheck(t, cloneB, binDir)
	if err == nil {
		t.Fatalf("aiwf check on B should have reported findings (cross-tree G-001 collision); output was clean:\n%s", out)
	}
	if !strings.Contains(out, "ids-unique") {
		t.Errorf("expected ids-unique finding; got:\n%s", out)
	}
	if !strings.Contains(out, "trunk") {
		t.Errorf("finding should mention trunk; got:\n%s", out)
	}
	if !strings.Contains(out, "G-001") {
		t.Errorf("finding should name G-001; got:\n%s", out)
	}
}

// TestIntegrationG37_NoRemoteSkipsSilently confirms the opposite
// edge: a sandbox repo with no remote configured at all (a fresh
// `git init` somebody is exploring with) gets working-tree-only
// allocation and no errors. This is the case where the allocator is
// deliberately *not* doing extra work, because the repo has no
// cross-branch coordination surface.
func TestIntegrationG37_NoRemoteSkipsSilently(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	root := t.TempDir()
	if out, err := runGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "solo@example.com"},
		{"config", "user.name", "Solo"},
	} {
		if out, err := runGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := runBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}

	out := aiwfAddGap(t, root, binDir, "First")
	if !strings.Contains(out, "G-001") {
		t.Errorf("expected G-001 in no-remote sandbox; got:\n%s", out)
	}
	out2 := aiwfAddGap(t, root, binDir, "Second")
	if !strings.Contains(out2, "G-002") {
		t.Errorf("expected G-002 in no-remote sandbox; got:\n%s", out2)
	}
}

// TestIntegrationG37_MixedKindsAcrossTrunk confirms that the
// allocator's kind filter applies to trunk ids the same way it
// applies to working-tree ids. Trunk has G-001 and ADR-0001; B's
// working tree is empty. B's next allocations must be G-002,
// ADR-0002, M-001 (since trunk has no milestones).
//
// Without correct kind filtering, a trunk ADR could bleed into the
// gap-allocator's max and produce surprising leaps in id numbers.
func TestIntegrationG37_MixedKindsAcrossTrunk(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	bare := makeBareOrigin(t)
	cloneA := makeClone(t, bare, "A")
	aiwfInitClone(t, cloneA, binDir)

	cloneB := makeSiblingClone(t, bare, "B")

	// A seeds two different kinds onto trunk.
	aiwfAddGap(t, cloneA, binDir, "Trunk's only gap")
	aiwfAddADR(t, cloneA, binDir, "Trunk's only ADR")
	aiwfAddEpic(t, cloneA, binDir, "Foundations")
	pushAll(t, cloneA)

	// B fetches; working tree is still untouched.
	fetchOrigin(t, cloneB)

	// B's allocations must reflect kind-filtered trunk awareness.
	gapOut := aiwfAddGap(t, cloneB, binDir, "B's gap")
	if !strings.Contains(gapOut, "G-002") {
		t.Errorf("gap should be G-002 (skipping trunk's G-001); got:\n%s", gapOut)
	}
	adrOut := aiwfAddADR(t, cloneB, binDir, "B's ADR")
	if !strings.Contains(adrOut, "ADR-0002") {
		t.Errorf("ADR should be ADR-0002 (skipping trunk's ADR-0001); got:\n%s", adrOut)
	}
	// Trunk has E-01 (the epic A added). To add a milestone under it,
	// B needs E-01 in its own working tree, which means rebasing onto
	// origin/main so trunk's entities land on B's branch. Real-world
	// equivalent: `git pull --rebase`.
	if out, err := runGit(cloneB, "rebase", "-q", "origin/main"); err != nil {
		t.Fatalf("rebase in B: %v\n%s", err, out)
	}
	mOut := aiwfAddMilestone(t, cloneB, binDir, "B's milestone", "E-01")
	if !strings.Contains(mOut, "M-001") {
		t.Errorf("milestone should be M-001 (no milestones on trunk); got:\n%s", mOut)
	}
}

// TestIntegrationG37_ReallocateUsesTrunkView confirms that
// `aiwf reallocate` participates in the trunk-aware allocation: the
// renumbered entity gets a fresh id that doesn't collide with trunk.
//
// Setup: trunk has G-001 and G-002 (both pushed by A). B fetches but
// doesn't merge, then locally adds two gaps — the allocator picks
// G-003 and G-004 (skipping trunk). B then runs reallocate on G-004.
// The new id must be G-005 (max(working ∪ trunk) + 1), not G-003 or
// anything that already exists anywhere.
func TestIntegrationG37_ReallocateUsesTrunkView(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	bare := makeBareOrigin(t)
	cloneA := makeClone(t, bare, "A")
	aiwfInitClone(t, cloneA, binDir)

	cloneB := makeSiblingClone(t, bare, "B")

	aiwfAddGap(t, cloneA, binDir, "Trunk gap one")
	aiwfAddGap(t, cloneA, binDir, "Trunk gap two")
	pushAll(t, cloneA)

	fetchOrigin(t, cloneB)

	// B's allocations must skip trunk's G-001 and G-002.
	if out := aiwfAddGap(t, cloneB, binDir, "B local one"); !strings.Contains(out, "G-003") {
		t.Fatalf("expected B's first add to allocate G-003; got:\n%s", out)
	}
	if out := aiwfAddGap(t, cloneB, binDir, "B local two"); !strings.Contains(out, "G-004") {
		t.Fatalf("expected B's second add to allocate G-004; got:\n%s", out)
	}

	// Locate the G-004 entity path so we can pass it to reallocate.
	gapDir := filepath.Join(cloneB, "work", "gaps")
	entries, err := os.ReadDir(gapDir)
	if err != nil {
		t.Fatal(err)
	}
	var g004Path string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "G-004-") {
			g004Path = filepath.Join("work", "gaps", e.Name())
			break
		}
	}
	if g004Path == "" {
		t.Fatalf("could not find G-004-*.md in %s", gapDir)
	}

	out, err := runBin(t, cloneB, binDir, nil, "reallocate", g004Path)
	if err != nil {
		t.Fatalf("aiwf reallocate %s: %v\n%s", g004Path, err, out)
	}
	if !strings.Contains(out, "G-005") {
		t.Errorf("reallocate of G-004 should produce G-005 (max of trunk ∪ working + 1); got:\n%s", out)
	}
	// G-004 must no longer exist in the working tree; G-005 must.
	post, err := os.ReadDir(gapDir)
	if err != nil {
		t.Fatal(err)
	}
	var hasG004, hasG005 bool
	for _, e := range post {
		if strings.HasPrefix(e.Name(), "G-004-") {
			hasG004 = true
		}
		if strings.HasPrefix(e.Name(), "G-005-") {
			hasG005 = true
		}
	}
	if hasG004 {
		t.Error("G-004-*.md should be gone after reallocate")
	}
	if !hasG005 {
		t.Error("G-005-*.md should exist after reallocate")
	}

	// Audit-trail check: the reallocate commit must carry the standard
	// aiwf-verb / aiwf-entity / aiwf-actor trailers, plus the
	// aiwf-prior-entity trailer that bridges G-004 to G-005 in
	// `aiwf history` queries. Without this assertion the test would
	// only prove the side effects on disk; the kernel's "git log is
	// the audit log" guarantee needs to be exercised end-to-end too.
	trailers, gitErr := runGit(cloneB, "log", "-1", "--format=%(trailers)", "HEAD")
	if gitErr != nil {
		t.Fatalf("reading HEAD trailers: %v\n%s", gitErr, trailers)
	}
	for _, want := range []string{
		"aiwf-verb: reallocate",
		"aiwf-entity: G-005",
		"aiwf-prior-entity: G-004",
		"aiwf-actor: ",
	} {
		if !strings.Contains(trailers, want) {
			t.Errorf("HEAD trailers missing %q; got:\n%s", want, trailers)
		}
	}
}

// TestIntegrationG37_CleanFetchAndMergeRoundTrip is the non-conflict
// happy path: two clones use disjoint id spaces by virtue of the
// allocator's trunk awareness, and the merge succeeds. This is the
// shape the framework wants every team's day-to-day to look like.
//
// The test fails only if a regression makes the trunk-aware
// allocator over-eager (e.g., flagging same-path-on-trunk ids as
// collisions, or refusing to allocate when trunk is reachable).
func TestIntegrationG37_CleanFetchAndMergeRoundTrip(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	bare := makeBareOrigin(t)
	cloneA := makeClone(t, bare, "A")
	aiwfInitClone(t, cloneA, binDir)

	cloneB := makeSiblingClone(t, bare, "B")

	// A adds G-001 and pushes.
	aiwfAddGap(t, cloneA, binDir, "A's gap")
	pushAll(t, cloneA)

	// B fetches and adds a gap — must allocate G-002 cleanly.
	fetchOrigin(t, cloneB)
	out := aiwfAddGap(t, cloneB, binDir, "B's gap")
	if !strings.Contains(out, "G-002") {
		t.Errorf("expected G-002; got:\n%s", out)
	}

	// B rebases onto trunk and runs `aiwf check` — must be clean (both
	// gaps coexist at distinct ids and distinct paths). Rebase, not
	// fast-forward merge: B has its own commit for G-002 so origin/main
	// and B's main have diverged. Real-world equivalent: `git pull --rebase`.
	if out, err := runGit(cloneB, "rebase", "-q", "origin/main"); err != nil {
		t.Fatalf("rebase: %v\n%s", err, out)
	}
	if checkOut, err := aiwfCheck(t, cloneB, binDir); err != nil {
		t.Errorf("aiwf check on cleanly-merged B should pass; got %v\n%s", err, checkOut)
	}

	// B pushes back; round-trip completes.
	pushAll(t, cloneB)

	// A pulls B's changes; A's check must also be clean.
	if out, err := runGit(cloneA, "pull", "-q", "--ff-only", "origin", "main"); err != nil {
		t.Fatalf("A pull: %v\n%s", err, out)
	}
	if checkOut, err := aiwfCheck(t, cloneA, binDir); err != nil {
		t.Errorf("aiwf check on A after pulling B's gap should pass; got %v\n%s", err, checkOut)
	}
}

// TestIntegrationG37_PrePushHookCatchesCollision verifies the
// load-bearing claim that the pre-push hook (which runs `aiwf check`
// against the about-to-push state) refuses a colliding push. Same
// setup as DivergedBranchesCaughtByCheckPostFetch, but the assertion
// is on the hook's exit code rather than `aiwf check` directly —
// confirming the wiring from hook → check → trunk-aware ids-unique.
func TestIntegrationG37_PrePushHookCatchesCollision(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	bare := makeBareOrigin(t)
	cloneA := makeClone(t, bare, "A")
	aiwfInitClone(t, cloneA, binDir)

	cloneB := makeSiblingClone(t, bare, "B")
	// B also needs aiwf init's hook installed. The clone inherited
	// .git/config from A's push but not A's local .git/hooks. Run
	// aiwf init's update path on B so the hook lands locally too.
	if out, err := runBin(t, cloneB, binDir, nil, "update"); err != nil {
		t.Fatalf("aiwf update on B: %v\n%s", err, out)
	}

	aiwfAddGap(t, cloneA, binDir, "A's first")
	aiwfAddGap(t, cloneB, binDir, "B's first")

	pushAll(t, cloneA)
	fetchOrigin(t, cloneB)

	// Run the hook directly; the build-in absolute-path invocation
	// was already pinned by TestIntegration_FreshRepoLifecycle.
	hookOut, hookErr := runHook(t, cloneB, "")
	if hookErr == nil {
		t.Fatalf("pre-push hook on B should have failed (cross-tree G-001); output:\n%s", hookOut)
	}
	if !strings.Contains(hookOut, "ids-unique") {
		t.Errorf("hook output should mention ids-unique; got:\n%s", hookOut)
	}
	if !strings.Contains(hookOut, "trunk") {
		t.Errorf("hook output should mention trunk; got:\n%s", hookOut)
	}
}
