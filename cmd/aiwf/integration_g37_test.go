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
	out, err := runBin(t, clone, binDir, nil, "add", "milestone", "--tdd", "none", "--title", title, "--epic", epicID)
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

// aiwfAddDecision adds a decision with optional --relates-to refs.
// The relatesTo slice is comma-joined and passed verbatim.
func aiwfAddDecision(t *testing.T, clone, binDir, title string, relatesTo []string) string {
	t.Helper()
	args := []string{"add", "decision", "--title", title}
	if len(relatesTo) > 0 {
		args = append(args, "--relates-to", strings.Join(relatesTo, ","))
	}
	out, err := runBin(t, clone, binDir, nil, args...)
	if err != nil {
		t.Fatalf("aiwf add decision %q in %s: %v\n%s", title, clone, err, out)
	}
	return out
}

// aiwfReallocateByPath runs `aiwf reallocate <path>` in clone and
// returns the combined output.
func aiwfReallocateByPath(t *testing.T, clone, binDir, path string) string {
	t.Helper()
	out, err := runBin(t, clone, binDir, nil, "reallocate", path)
	if err != nil {
		t.Fatalf("aiwf reallocate %s in %s: %v\n%s", path, clone, err, out)
	}
	return out
}

// findEntityPath returns the repo-relative path under dir whose
// basename starts with prefix, or "" if no entry matches.
func findEntityPath(t *testing.T, root, dir, prefix string) string {
	t.Helper()
	full := filepath.Join(root, dir)
	entries, err := os.ReadDir(full)
	if err != nil {
		t.Fatalf("reading %s: %v", full, err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
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
	if !strings.Contains(out, "G-0002") {
		t.Errorf("expected B to allocate G-002 after fetching A's G-001 from trunk; got:\n%s", out)
	}
	if !strings.Contains(out, "G-0002") || strings.Contains(out, "added gap G-001") {
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
		if strings.HasPrefix(p, "G-0002-") {
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
	if !strings.Contains(out, "G-0001") {
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
	if !strings.Contains(out, "G-0001") {
		t.Errorf("expected G-001 in no-remote sandbox; got:\n%s", out)
	}
	out2 := aiwfAddGap(t, root, binDir, "Second")
	if !strings.Contains(out2, "G-0002") {
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
	if !strings.Contains(gapOut, "G-0002") {
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
	mOut := aiwfAddMilestone(t, cloneB, binDir, "B's milestone", "E-0001")
	if !strings.Contains(mOut, "M-0001") {
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
	if out := aiwfAddGap(t, cloneB, binDir, "B local one"); !strings.Contains(out, "G-0003") {
		t.Fatalf("expected B's first add to allocate G-003; got:\n%s", out)
	}
	if out := aiwfAddGap(t, cloneB, binDir, "B local two"); !strings.Contains(out, "G-0004") {
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
		if strings.HasPrefix(e.Name(), "G-0004-") {
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
	if !strings.Contains(out, "G-0005") {
		t.Errorf("reallocate of G-004 should produce G-005 (max of trunk ∪ working + 1); got:\n%s", out)
	}
	// G-004 must no longer exist in the working tree; G-005 must.
	post, err := os.ReadDir(gapDir)
	if err != nil {
		t.Fatal(err)
	}
	var hasG004, hasG005 bool
	for _, e := range post {
		if strings.HasPrefix(e.Name(), "G-0004-") {
			hasG004 = true
		}
		if strings.HasPrefix(e.Name(), "G-0005-") {
			hasG005 = true
		}
	}
	if hasG004 {
		t.Error("G-0004-*.md should be gone after reallocate")
	}
	if !hasG005 {
		t.Error("G-0005-*.md should exist after reallocate")
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
		"aiwf-entity: G-0005",
		"aiwf-prior-entity: G-0004",
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
	if !strings.Contains(out, "G-0002") {
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

// TestIntegrationG37_ReallocateRewritesRefsAndHistoryThreads is the
// full-flow scenario: a trunk-aware reallocate must (1) pick a new
// id that's free against trunk, (2) rewrite every other entity's
// frontmatter references to the renumbered entity in the same
// commit, and (3) thread `aiwf history` correctly across the rename
// via the aiwf-prior-entity trailer.
//
// Setup:
//
//   - Clone A creates G-001 and a decision D-001 whose `relates_to`
//     names G-001. Pushes both. Trunk now carries the cross-reference.
//   - Clone B fetches and rebases trunk in (so B's working tree has
//     G-001 and D-001 with the live reference).
//   - Clone A then adds G-002 (an unrelated gap) and pushes. Trunk
//     advances; refs/remotes/origin/main on B is stale until the next
//     fetch.
//   - Clone B fetches without merging. B's working tree still has
//     G-001 and D-001 from the earlier rebase; B's trunk view now
//     also includes G-002.
//   - Clone B reallocates G-001.
//
// Asserted:
//
//  1. Trunk-aware allocator skips G-002 (visible only via trunk) and
//     picks G-003 for the new id. Without trunk awareness the
//     allocator would pick G-002 and collide with trunk.
//  2. The decision D-001 in B's working tree has `relates_to: [G-003]`
//     after the reallocate — the reference rewrite rode along in the
//     same commit.
//  3. The reallocate commit carries the standard verb/entity/actor
//     trailers plus aiwf-prior-entity: G-0001, the bridge that lets
//     `aiwf history G-001` continue to find the entity post-rename.
//  4. `aiwf history G-001` returns at least one event referencing
//     the rename (via the prior-entity trailer match).
//  5. `aiwf history G-003` returns the post-rename history (matching
//     aiwf-entity: G-0003 directly).
//
// This is the test that pins the kernel's "git log is the audit log"
// guarantee for the cross-tree reallocate path: id-aware, ref-aware,
// history-aware in one go.
func TestIntegrationG37_ReallocateRewritesRefsAndHistoryThreads(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	bare := makeBareOrigin(t)
	cloneA := makeClone(t, bare, "A")
	aiwfInitClone(t, cloneA, binDir)

	cloneB := makeSiblingClone(t, bare, "B")

	// A creates the cross-referenced pair: a gap and a decision
	// that names the gap in relates_to. Both push to trunk.
	aiwfAddGap(t, cloneA, binDir, "Cache eviction breaks under churn")
	aiwfAddDecision(t, cloneA, binDir, "Bound the cache by entry count", []string{"G-0001"})
	pushAll(t, cloneA)

	// B brings trunk in via rebase so the working tree has both
	// entities at their original ids (G-001 and D-001 with the
	// reference live in D-001's frontmatter).
	fetchOrigin(t, cloneB)
	if out, err := runGit(cloneB, "rebase", "-q", "origin/main"); err != nil {
		t.Fatalf("B rebase: %v\n%s", err, out)
	}

	// Verify the precondition: D-001 references G-001 in B's tree.
	dPath := findEntityPath(t, cloneB, "work/decisions", "D-0001-")
	if dPath == "" {
		t.Fatal("expected D-001-*.md in B's work/decisions/")
	}
	dContent, err := os.ReadFile(filepath.Join(cloneB, dPath))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dContent), "G-0001") {
		t.Fatalf("decision should reference G-001 before reallocate; got:\n%s", dContent)
	}

	// A advances trunk with an unrelated G-002. B fetches but does
	// NOT merge — B's trunk view now includes G-002 even though B's
	// working tree doesn't.
	aiwfAddGap(t, cloneA, binDir, "Unrelated gap")
	pushAll(t, cloneA)
	fetchOrigin(t, cloneB)

	// Reallocate G-001 on B. Trunk-aware allocator must skip
	// G-002 (only on trunk) and pick G-003.
	gPath := findEntityPath(t, cloneB, "work/gaps", "G-0001-")
	if gPath == "" {
		t.Fatal("expected G-001-*.md in B's work/gaps/ before reallocate")
	}
	out := aiwfReallocateByPath(t, cloneB, binDir, gPath)
	if !strings.Contains(out, "G-0003") {
		t.Fatalf("reallocate should produce G-003 (skipping trunk's G-002); got:\n%s", out)
	}

	// Assertion 1: G-001 is gone from disk; G-003 has replaced it.
	if got := findEntityPath(t, cloneB, "work/gaps", "G-0001-"); got != "" {
		t.Errorf("G-0001-*.md should be gone after reallocate; still found %s", got)
	}
	g3Path := findEntityPath(t, cloneB, "work/gaps", "G-0003-")
	if g3Path == "" {
		t.Errorf("expected G-003-*.md to exist after reallocate")
	}

	// Assertion 2: D-001's frontmatter relates_to was rewritten in
	// the same commit. This is the core "references update" check.
	dPathPost := findEntityPath(t, cloneB, "work/decisions", "D-0001-")
	if dPathPost == "" {
		t.Fatal("expected D-001-*.md still present after reallocate")
	}
	dContentPost, err := os.ReadFile(filepath.Join(cloneB, dPathPost))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dContentPost), "G-0001") {
		t.Errorf("D-001 should no longer reference G-001 after reallocate; got:\n%s", dContentPost)
	}
	if !strings.Contains(string(dContentPost), "G-0003") {
		t.Errorf("D-001 should now reference G-003; got:\n%s", dContentPost)
	}

	// Assertion 3: the reallocate commit carries the prior-entity
	// trailer plus the standard set.
	trailers, gitErr := runGit(cloneB, "log", "-1", "--format=%(trailers)", "HEAD")
	if gitErr != nil {
		t.Fatalf("reading HEAD trailers: %v\n%s", gitErr, trailers)
	}
	for _, want := range []string{
		"aiwf-verb: reallocate",
		"aiwf-entity: G-0003",
		"aiwf-prior-entity: G-0001",
		"aiwf-actor: ",
	} {
		if !strings.Contains(trailers, want) {
			t.Errorf("HEAD trailers missing %q; got:\n%s", want, trailers)
		}
	}

	// Assertion 4: aiwf history G-001 (the OLD id) still finds the
	// entity's lifecycle via the aiwf-prior-entity backward grep.
	// At minimum it should return the rename commit itself.
	histOld, err := runBin(t, cloneB, binDir, nil, "history", "G-0001")
	if err != nil {
		t.Fatalf("aiwf history G-001: %v\n%s", err, histOld)
	}
	if !strings.Contains(histOld, "reallocate") {
		t.Errorf("aiwf history G-001 should surface the reallocate event; got:\n%s", histOld)
	}

	// Assertion 5: aiwf history G-003 (the NEW id) finds the post-
	// rename history via the aiwf-entity trailer.
	histNew, err := runBin(t, cloneB, binDir, nil, "history", "G-0003")
	if err != nil {
		t.Fatalf("aiwf history G-003: %v\n%s", err, histNew)
	}
	if !strings.Contains(histNew, "reallocate") {
		t.Errorf("aiwf history G-003 should include the reallocate commit; got:\n%s", histNew)
	}

	// Assertion 6: aiwf check on the post-reallocate working tree
	// is clean — the cross-tree collision (G-001 vs trunk's G-001)
	// is gone and no new findings were introduced.
	if checkOut, err := aiwfCheck(t, cloneB, binDir); err != nil {
		t.Errorf("aiwf check should be clean after reallocate; got %v\n%s", err, checkOut)
	}
}

// TestIntegrationG37_ReallocateTiebreakerPicksLocalSide pins the
// load-bearing tiebreaker behavior (id-allocation.md §"Reallocate
// when both branches did real work"):
//
//   - Two clones independently allocate G-001 (different titles, so
//     different paths).
//   - A pushes first; A's G-001 is now an ancestor of trunk
//     (refs/remotes/origin/main on B after B fetches).
//   - B rebases. B's working tree now carries both files: A's
//     G-001-cache-busts.md AND B's G-001-pre-aiwf-docs.md.
//   - B runs `aiwf reallocate G-001` with the bare id.
//   - Tiebreaker resolves: A's add-commit IS an ancestor of trunk;
//     B's local add-commit IS NOT. B's local side is the loser.
//   - Reallocate renumbers B's local side to G-002 (or higher,
//     skipping any trunk gaps); A's side keeps G-001.
//
// This is exactly the workflow the gap entry describes for the real
// flowtime-vnext incident, executed against real git plumbing.
func TestIntegrationG37_ReallocateTiebreakerPicksLocalSide(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	bare := makeBareOrigin(t)
	cloneA := makeClone(t, bare, "A")
	aiwfInitClone(t, cloneA, binDir)

	cloneB := makeSiblingClone(t, bare, "B")

	// Both clones allocate G-001 independently.
	aiwfAddGap(t, cloneA, binDir, "Cache busts under heavy load")
	aiwfAddGap(t, cloneB, binDir, "Pre-aiwf v1 docs survived migration")

	// A pushes first. Trunk now carries A's G-001.
	pushAll(t, cloneA)

	// B fetches and rebases, bringing A's G-001 into B's working
	// tree. The two G-001 files have different slugs so git merges
	// them in cleanly; B's tree now has duplicate ids.
	fetchOrigin(t, cloneB)
	if out, err := runGit(cloneB, "rebase", "-q", "origin/main"); err != nil {
		t.Fatalf("B rebase: %v\n%s", err, out)
	}

	// Sanity: both G-001 files now exist in B's tree.
	gapDir := filepath.Join(cloneB, "work", "gaps")
	entries, err := os.ReadDir(gapDir)
	if err != nil {
		t.Fatal(err)
	}
	g001Count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "G-0001-") {
			g001Count++
		}
	}
	if g001Count != 2 {
		t.Fatalf("after rebase B should have 2 G-001-* files, found %d: %v", g001Count, entries)
	}

	// `aiwf reallocate G-001` with a bare id. Without the tiebreaker
	// this would fail with "ambiguous, pass a path". With the
	// tiebreaker, ancestry resolves: A's G-001 is on trunk, B's is
	// not, so B's side is the loser and gets renumbered.
	out, err := runBin(t, cloneB, binDir, nil, "reallocate", "G-0001")
	if err != nil {
		t.Fatalf("aiwf reallocate G-001 (bare): %v\n%s", err, out)
	}
	if !strings.Contains(out, "G-0002") {
		t.Errorf("reallocate output should reference G-002 (the renumbered local side); got:\n%s", out)
	}

	// After reallocate, A's G-001 (cache-busts) still exists; B's
	// local pre-aiwf G-001 is gone, replaced by G-002.
	entries, err = os.ReadDir(gapDir)
	if err != nil {
		t.Fatal(err)
	}
	var hasG001CacheBusts, hasG002PreAiwf, lingeringG001Pre bool
	for _, e := range entries {
		switch {
		case strings.HasPrefix(e.Name(), "G-0001-cache-busts"):
			hasG001CacheBusts = true
		case strings.HasPrefix(e.Name(), "G-0002-pre-aiwf"):
			hasG002PreAiwf = true
		case strings.HasPrefix(e.Name(), "G-0001-pre-aiwf"):
			lingeringG001Pre = true
		}
	}
	if !hasG001CacheBusts {
		t.Errorf("A's G-001-cache-busts-* should survive; entries: %v", entries)
	}
	if !hasG002PreAiwf {
		t.Errorf("B's local should be renumbered to G-002-pre-aiwf-*; entries: %v", entries)
	}
	if lingeringG001Pre {
		t.Errorf("B's local G-001-pre-aiwf-* should have been renamed; entries: %v", entries)
	}

	// And aiwf check on the post-reallocate tree must be clean.
	if checkOut, err := aiwfCheck(t, cloneB, binDir); err != nil {
		t.Errorf("aiwf check after tiebreaker reallocate should be clean; got %v\n%s", err, checkOut)
	}

	// Audit-trail check: the reallocate commit's trailers name the
	// renumbered side. aiwf-prior-entity is G-001 (the id B's local
	// side carried before the rename); aiwf-entity is G-002 (the new
	// canonical id for B's local side).
	trailers, gitErr := runGit(cloneB, "log", "-1", "--format=%(trailers)", "HEAD")
	if gitErr != nil {
		t.Fatalf("reading HEAD trailers: %v\n%s", gitErr, trailers)
	}
	for _, want := range []string{
		"aiwf-verb: reallocate",
		"aiwf-entity: G-0002",
		"aiwf-prior-entity: G-0001",
		"aiwf-actor: ",
	} {
		if !strings.Contains(trailers, want) {
			t.Errorf("HEAD trailers missing %q; got:\n%s", want, trailers)
		}
	}
}

// TestIntegrationG37_ReallocateTiebreakerAmbiguousNeitherInTrunk
// pins the negative case: when ancestry can't decide (both sides
// merged to trunk, or neither has, or both are local-only), the
// verb refuses with a clear error rather than silently picking one.
//
// Setup: two clones add G-001 independently; neither pushes. Then
// one clone pulls the other's branch directly so its tree has
// duplicate ids — but neither add commit is an ancestor of
// origin/main (which still only has the aiwf-init commit).
func TestIntegrationG37_ReallocateTiebreakerAmbiguousNeitherInTrunk(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	bare := makeBareOrigin(t)
	cloneA := makeClone(t, bare, "A")
	aiwfInitClone(t, cloneA, binDir)

	cloneB := makeSiblingClone(t, bare, "B")

	aiwfAddGap(t, cloneA, binDir, "A side")
	aiwfAddGap(t, cloneB, binDir, "B side")

	// B adds A's clone as a remote, fetches A's commit, and
	// cherry-picks it. Now B's tree has both G-001 files but
	// neither add-commit is on origin/main (which is still at
	// aiwf-init).
	if out, err := runGit(cloneB, "remote", "add", "peer", cloneA); err != nil {
		t.Fatalf("git remote add peer: %v\n%s", err, out)
	}
	if out, err := runGit(cloneB, "fetch", "-q", "peer"); err != nil {
		t.Fatalf("git fetch peer: %v\n%s", err, out)
	}
	// A's HEAD is the only commit on top of init we want; cherry-pick
	// it into B's main.
	aHead, headErr := runGit(cloneA, "rev-parse", "HEAD")
	if headErr != nil {
		t.Fatalf("rev-parse on A: %v", headErr)
	}
	if out, pickErr := runGit(cloneB, "cherry-pick", strings.TrimSpace(aHead)); pickErr != nil {
		t.Fatalf("git cherry-pick: %v\n%s", pickErr, out)
	}

	// Sanity: B has both G-001 files, neither is on origin/main.
	gapDir := filepath.Join(cloneB, "work", "gaps")
	entries, _ := os.ReadDir(gapDir)
	g001Count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "G-0001-") {
			g001Count++
		}
	}
	if g001Count != 2 {
		t.Fatalf("expected 2 G-001-* files; found %d: %v", g001Count, entries)
	}

	// Bare-id reallocate must refuse with a clear "ambiguous" message
	// naming both candidate paths and the diagnostic.
	out, err := runBin(t, cloneB, binDir, nil, "reallocate", "G-0001")
	if err == nil {
		t.Fatalf("reallocate G-001 (bare) should fail when ancestry can't decide; output:\n%s", out)
	}
	if !strings.Contains(out, "ambiguous") {
		t.Errorf("error message should say ambiguous; got:\n%s", out)
	}
	if !strings.Contains(out, "neither") {
		t.Errorf("diagnostic should mention 'neither' is on trunk; got:\n%s", out)
	}
}

// TestIntegrationG37_HistoryWalksLineageChain pins layer (b)'s
// load-bearing read-side guarantee: after two reallocations
// (G-001 → G-002 → G-003), `aiwf history` returns the same
// chronological chain whether the operator queries by the original
// id, the intermediate id, or the current id.
//
// The chain expansion lives in runHistory: it loads the tree, calls
// ResolveByCurrentOrPriorID(queriedID) to find the canonical entity,
// then walks the entity's PriorIDs slice plus its current ID to
// build the union the git-log grep covers. Without that walk, a
// query for the original id would only ever surface the rename
// commit (matched via aiwf-prior-entity), not the entity's
// post-rename promote/cancel/etc events under the new id.
func TestIntegrationG37_HistoryWalksLineageChain(t *testing.T) {
	bin := aiwfBinary(t)
	binDir := filepath.Dir(bin)

	bare := makeBareOrigin(t)
	clone := makeClone(t, bare, "A")
	aiwfInitClone(t, clone, binDir)

	// Add a gap, then reallocate twice. Because the working tree
	// already carries a G-NNN, each reallocate picks the next free
	// id: G-001 → G-002 → G-003.
	aiwfAddGap(t, clone, binDir, "Original phrasing")

	g1Path := findEntityPath(t, clone, "work/gaps", "G-0001-")
	if g1Path == "" {
		t.Fatal("G-0001-*.md missing after add")
	}
	if out := aiwfReallocateByPath(t, clone, binDir, g1Path); !strings.Contains(out, "G-0002") {
		t.Fatalf("first reallocate should produce G-002; got:\n%s", out)
	}

	g2Path := findEntityPath(t, clone, "work/gaps", "G-0002-")
	if g2Path == "" {
		t.Fatal("G-0002-*.md missing after first reallocate")
	}
	if out := aiwfReallocateByPath(t, clone, binDir, g2Path); !strings.Contains(out, "G-0003") {
		t.Fatalf("second reallocate should produce G-003; got:\n%s", out)
	}

	// Confirm the prior_ids chain landed on disk: the surviving
	// entity must list both prior ids, oldest-first.
	g3Path := findEntityPath(t, clone, "work/gaps", "G-0003-")
	if g3Path == "" {
		t.Fatal("G-0003-*.md missing after second reallocate")
	}
	g3Content, err := os.ReadFile(filepath.Join(clone, g3Path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(g3Content), "prior_ids:") {
		t.Errorf("G-003 frontmatter should carry prior_ids; got:\n%s", g3Content)
	}
	if !strings.Contains(string(g3Content), "G-0001") || !strings.Contains(string(g3Content), "G-0002") {
		t.Errorf("G-003.prior_ids should include both G-001 and G-002; got:\n%s", g3Content)
	}

	// All three queries must surface BOTH reallocate commits — the
	// original add (under G-001), the first rename (G-001 → G-002),
	// and the second rename (G-002 → G-003). The chain expander is
	// what makes the by-old-id queries see the post-rename events.
	for _, q := range []string{"G-0001", "G-0002", "G-0003"} {
		out, err := runBin(t, clone, binDir, nil, "history", q)
		if err != nil {
			t.Fatalf("aiwf history %s: %v\n%s", q, err, out)
		}
		// All three queries must yield three timeline rows: the
		// original add, the first reallocate, the second reallocate.
		// Counting "add\b" or "reallocate\b" occurrences pins
		// ordering and presence without depending on commit SHAs.
		if got := strings.Count(out, "reallocate"); got < 2 {
			t.Errorf("history %s should include both reallocate events (got %d); output:\n%s", q, got, out)
		}
		if !strings.Contains(out, "add") {
			t.Errorf("history %s should include the original add; output:\n%s", q, out)
		}
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
