package gitops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHasRemotes_NoRemotes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	got, err := HasRemotes(ctx, dir)
	if err != nil {
		t.Fatalf("HasRemotes: %v", err)
	}
	if got {
		t.Error("HasRemotes = true, want false (fresh repo has no remotes)")
	}
}

func TestHasRemotes_WithRemote(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	mustRun(t, ctx, dir, "remote", "add", "origin", "https://example.invalid/x.git")
	got, err := HasRemotes(ctx, dir)
	if err != nil {
		t.Fatalf("HasRemotes: %v", err)
	}
	if !got {
		t.Error("HasRemotes = false, want true after remote add")
	}
}

func TestHasRef_Present(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "a.txt", "hello")
	got, err := HasRef(ctx, dir, "HEAD")
	if err != nil {
		t.Fatalf("HasRef: %v", err)
	}
	if !got {
		t.Error("HasRef(HEAD) = false, want true after commit")
	}
}

func TestHasRef_Missing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "a.txt", "hello")
	got, err := HasRef(ctx, dir, "refs/remotes/origin/main")
	if err != nil {
		t.Fatalf("HasRef: %v", err)
	}
	if got {
		t.Error("HasRef(refs/remotes/origin/main) = true, want false (no remote configured)")
	}
}

func TestLsTreePaths_FullTree(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	writeFile(t, dir, "work/gaps/G-001-foo.md", "# foo\n")
	writeFile(t, dir, "work/gaps/G-002-bar.md", "# bar\n")
	writeFile(t, dir, "docs/adr/ADR-0001-baz.md", "# baz\n")
	writeFile(t, dir, "README.md", "readme\n")
	mustRun(t, ctx, dir, "add", "-A")
	mustRun(t, ctx, dir, "commit", "-q", "-m", "seed")

	got, err := LsTreePaths(ctx, dir, "HEAD")
	if err != nil {
		t.Fatalf("LsTreePaths: %v", err)
	}
	sort.Strings(got)
	want := []string{
		"README.md",
		"docs/adr/ADR-0001-baz.md",
		"work/gaps/G-001-foo.md",
		"work/gaps/G-002-bar.md",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("paths mismatch (-want +got):\n%s", diff)
	}
}

func TestLsTreePaths_Prefixes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	writeFile(t, dir, "work/gaps/G-001-foo.md", "# foo\n")
	writeFile(t, dir, "docs/adr/ADR-0001-baz.md", "# baz\n")
	writeFile(t, dir, "README.md", "readme\n")
	mustRun(t, ctx, dir, "add", "-A")
	mustRun(t, ctx, dir, "commit", "-q", "-m", "seed")

	got, err := LsTreePaths(ctx, dir, "HEAD", "work/", "docs/adr/")
	if err != nil {
		t.Fatalf("LsTreePaths: %v", err)
	}
	sort.Strings(got)
	want := []string{
		"docs/adr/ADR-0001-baz.md",
		"work/gaps/G-001-foo.md",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("paths mismatch (-want +got):\n%s", diff)
	}
}

func TestLsTreePaths_RefNotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "a.txt", "hello")
	_, err := LsTreePaths(ctx, dir, "refs/remotes/origin/main")
	if !errors.Is(err, ErrRefNotFound) {
		t.Fatalf("expected ErrRefNotFound, got %v", err)
	}
}

func TestLsTreePaths_EmptyTree(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	// Create an empty tree object and a commit pointing at it on a branch.
	emptyTree := mustOutput(t, ctx, dir, "mktree")
	commit := mustOutput(t, ctx, dir, "commit-tree", emptyTree, "-m", "empty")
	mustRun(t, ctx, dir, "update-ref", "refs/heads/empty", commit)

	got, err := LsTreePaths(ctx, dir, "refs/heads/empty")
	if err != nil {
		t.Fatalf("LsTreePaths: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("LsTreePaths on empty ref = %v, want []", got)
	}
}

func TestAddCommitSHA_ReturnsBirthCommitForExactPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)

	commitFile(t, ctx, dir, "work/gaps/G-001-foo.md", "# foo\n")
	birthSHA := mustOutput(t, ctx, dir, "rev-parse", "HEAD")

	// Unrelated commit so HEAD isn't the birth.
	commitFile(t, ctx, dir, "README.md", "readme\n")

	got, err := AddCommitSHA(ctx, dir, "work/gaps/G-001-foo.md")
	if err != nil {
		t.Fatalf("AddCommitSHA: %v", err)
	}
	if got != birthSHA {
		t.Errorf("AddCommitSHA = %q, want birth %q", got, birthSHA)
	}
}

func TestAddCommitSHA_DoesNotFollowAcrossRename(t *testing.T) {
	t.Parallel()
	// AddCommitSHA deliberately omits `git log --follow` because
	// `--follow` is a content-similarity heuristic that mis-attributes
	// one entity's add commit to a similar entity in the
	// duplicate-id reallocate scenario. Document the trade-off here:
	// after a path rename the function returns the rename commit,
	// not the original birth. Callers (the reallocate tiebreaker)
	// rely on this — they care about "when did this exact path get
	// these bytes," not "what's the genealogy of this file."
	ctx := context.Background()
	dir := initTestRepo(t)

	commitFile(t, ctx, dir, "work/gaps/G-001-foo.md", "# foo\n")
	birthSHA := mustOutput(t, ctx, dir, "rev-parse", "HEAD")
	commitFile(t, ctx, dir, "README.md", "readme\n")

	mustRun(t, ctx, dir, "mv", "work/gaps/G-001-foo.md", "work/gaps/G-002-foo.md")
	mustRun(t, ctx, dir, "add", "-A")
	mustRun(t, ctx, dir, "commit", "-q", "-m", "rename")
	renameSHA := mustOutput(t, ctx, dir, "rev-parse", "HEAD")

	got, err := AddCommitSHA(ctx, dir, "work/gaps/G-002-foo.md")
	if err != nil {
		t.Fatalf("AddCommitSHA: %v", err)
	}
	if got == birthSHA {
		t.Errorf("AddCommitSHA after rename = birth %q; expected the rename commit, not the birth", birthSHA)
	}
	if got != renameSHA {
		t.Errorf("AddCommitSHA after rename = %q, want rename commit %q", got, renameSHA)
	}
}

func TestAddCommitSHA_PathWithNoHistory(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "README.md", "readme\n")

	got, err := AddCommitSHA(ctx, dir, "work/gaps/G-001-never-committed.md")
	if err != nil {
		t.Fatalf("AddCommitSHA: %v", err)
	}
	if got != "" {
		t.Errorf("AddCommitSHA on uncommitted path = %q, want \"\"", got)
	}
}

func TestIsAncestor_TrueAndFalse(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "a.txt", "1")
	first := mustOutput(t, ctx, dir, "rev-parse", "HEAD")
	commitFile(t, ctx, dir, "b.txt", "2")

	// first is an ancestor of HEAD.
	got, err := IsAncestor(ctx, dir, first, "HEAD")
	if err != nil {
		t.Fatalf("IsAncestor: %v", err)
	}
	if !got {
		t.Errorf("IsAncestor(first, HEAD) = false, want true")
	}

	// HEAD is not an ancestor of first.
	head := mustOutput(t, ctx, dir, "rev-parse", "HEAD")
	got, err = IsAncestor(ctx, dir, head, first)
	if err != nil {
		t.Fatalf("IsAncestor: %v", err)
	}
	if got {
		t.Errorf("IsAncestor(HEAD, first) = true, want false")
	}
}

func TestIsAncestor_BadRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "a.txt", "1")

	_, err := IsAncestor(ctx, dir, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "HEAD")
	if err == nil {
		t.Error("IsAncestor on a SHA that doesn't exist should error, got nil")
	}
}

// TestRenamesFromRef_DetectsCommittedRename pins G-0109's primary path.
// Set up a "trunk" ref pointing at the original slug, commit a rename
// on top, and assert that RenamesFromRef reports the old → new pair.
func TestRenamesFromRef_DetectsCommittedRename(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	const oldPath = "work/gaps/G-0035-very-long-historical-slug.md"
	const newPath = "work/gaps/G-0035-short.md"
	body := "---\nid: G-0035\nkind: gap\ntitle: example\nstatus: open\n---\nbody text\n"
	commitFile(t, ctx, dir, oldPath, body)
	mustRun(t, ctx, dir, "branch", "trunk")

	// Rename in a fresh commit on top of trunk.
	mustRun(t, ctx, dir, "mv", oldPath, newPath)
	mustRun(t, ctx, dir, "commit", "-q", "-m", "rename G-0035 to short slug")

	got, err := RenamesFromRef(ctx, dir, "trunk")
	if err != nil {
		t.Fatalf("RenamesFromRef: %v", err)
	}
	want := map[string]string{oldPath: newPath}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("RenamesFromRef mismatch (-want +got):\n%s", diff)
	}
}

// TestRenamesFromRef_DetectsTrailerDrivenRenameAcrossBodyEdits pins
// the G-0167 fix: when an entity is retitled (file rename + slug
// change) in one commit AND its body is substantially enriched in
// separate commits within the same branch, the cumulative
// merge-base..HEAD diff falls below the -M50 similarity threshold
// and git's default rename detection misses it. The kernel falls
// back to operator intent via the `aiwf-verb: retitle` commit
// trailer — the retitle commit's per-commit diff (which has high
// similarity since the body is unchanged in that commit) tells the
// rule definitively "this was a rename of entity X."
//
// Scenario authored to mirror the M-0125 G-0139 case: rename + 3×
// body growth. Without the trailer-driven detection, git's default
// 50% similarity heuristic misses the rename and the
// ids-unique/trunk-collision rule fires a false positive. With it,
// the rename is captured from the retitle commit's trailer.
func TestRenamesFromRef_DetectsTrailerDrivenRenameAcrossBodyEdits(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	const oldPath = "work/gaps/G-0050-original-short-slug.md"
	const newPath = "work/gaps/G-0050-new-much-longer-slug-after-retitle-and-enrichment.md"

	originalBody := `---
id: G-0050
title: original title
status: open
---

## Problem

Short original body, a few lines of prose to establish the gap's
diagnostic surface. Nothing elaborate; just the seed.
`
	commitFile(t, ctx, dir, oldPath, originalBody)
	mustRun(t, ctx, dir, "branch", "trunk")

	// Commit 1: retitle (slug change only; frontmatter title also
	// changed in real retitle, but for the rename map only the path
	// change matters here). Stamp the kernel trailer.
	mustRun(t, ctx, dir, "mv", oldPath, newPath)
	// Update frontmatter title in the file at newPath (real retitle
	// touches both slug and frontmatter title).
	retitleBody := strings.Replace(originalBody, "title: original title", "title: new much longer title with substantially more detail", 1)
	if err := os.WriteFile(filepath.Join(dir, newPath), []byte(retitleBody), 0o644); err != nil {
		t.Fatalf("writing retitle body: %v", err)
	}
	mustRun(t, ctx, dir, "add", newPath)
	mustRun(t, ctx, dir, "commit", "-q", "-m", "aiwf retitle G-0050 -> new title\n\naiwf-verb: retitle\naiwf-entity: G-0050\naiwf-actor: human/test")

	// Commit 2: substantial body enrichment in a separate commit.
	// This is the change that pushes cumulative similarity below
	// -M50 in the M-0125 G-0139 scenario.
	enrichedBody := retitleBody + `

## Why it matters

The first enrichment section adds rationale — why the gap matters,
who's affected, what the failure mode looks like in practice. This
is content that the original gap didn't carry but reviewers and
downstream consumers want to see.

## Proposed fix shape

A second enrichment section sketches the impl path. Multiple
sub-points, code references, file pointers, and the closing
checklist. Substantial content, mostly net-new relative to the
original body.

  - Sub-point one: identify the chokepoint.
  - Sub-point two: design the fix.
  - Sub-point three: test the fix.
  - Sub-point four: land the fix.
  - Sub-point five: close the gap.

## History

A third enrichment section captures the consolidation history —
prior duplicates, retitle rationale, related decisions.

## Closing this gap

Final section enumerates the cleanup steps when the impl lands.
`
	if err := os.WriteFile(filepath.Join(dir, newPath), []byte(enrichedBody), 0o644); err != nil {
		t.Fatalf("writing enriched body: %v", err)
	}
	mustRun(t, ctx, dir, "add", newPath)
	mustRun(t, ctx, dir, "commit", "-q", "-m", "aiwf edit-body G-0050\n\naiwf-verb: edit-body\naiwf-entity: G-0050\naiwf-actor: human/test")

	got, err := RenamesFromRef(ctx, dir, "trunk")
	if err != nil {
		t.Fatalf("RenamesFromRef: %v", err)
	}
	want := map[string]string{oldPath: newPath}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("expected trailer-driven rename detection (G-0167 fix); mismatch (-want +got):\n%s", diff)
	}
}

// TestRenamesFromRef_ChainsForwardThroughMultipleRetitles pins the
// chain-forward semantics of the trailer-driven detection. When an
// entity is retitled multiple times in a single branch (A→B in
// commit X, B→C in commit Y), the cumulative rename map records
// A→C, not {A:B, B:C}. Without chaining, the trunk-collision rule
// would look up A (trunk-side path) and find B (which doesn't
// exist on the branch), failing the rename match.
func TestRenamesFromRef_ChainsForwardThroughMultipleRetitles(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	const pathA = "work/gaps/G-0060-first-slug.md"
	const pathB = "work/gaps/G-0060-second-slug.md"
	const pathC = "work/gaps/G-0060-third-slug.md"
	body := "---\nid: G-0060\ntitle: a\nstatus: open\n---\nbody\n"
	commitFile(t, ctx, dir, pathA, body)
	mustRun(t, ctx, dir, "branch", "trunk")

	// Retitle 1: A → B.
	mustRun(t, ctx, dir, "mv", pathA, pathB)
	mustRun(t, ctx, dir, "commit", "-q", "-m", "aiwf retitle G-0060 -> b\n\naiwf-verb: retitle\naiwf-entity: G-0060\naiwf-actor: human/test")

	// Retitle 2: B → C.
	mustRun(t, ctx, dir, "mv", pathB, pathC)
	mustRun(t, ctx, dir, "commit", "-q", "-m", "aiwf retitle G-0060 -> c\n\naiwf-verb: retitle\naiwf-entity: G-0060\naiwf-actor: human/test")

	got, err := RenamesFromRef(ctx, dir, "trunk")
	if err != nil {
		t.Fatalf("RenamesFromRef: %v", err)
	}
	want := map[string]string{pathA: pathC}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("expected chained rename A→C, not {A:B, B:C}; mismatch (-want +got):\n%s", diff)
	}
}

// TestRenamesFromRef_IgnoresUncommittedRename pins the
// merge-base..HEAD scoping (G-0109): an uncommitted `git mv` shows
// in the working tree but not in any commit, so RenamesFromRef does
// not see it. This is the documented contract — pre-push (the
// trunk-collision rule's canonical caller) runs after commit, so the
// working-tree-only state is a transient interactive condition and
// is intentionally out of scope.
func TestRenamesFromRef_IgnoresUncommittedRename(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	const oldPath = "work/gaps/G-0036-original.md"
	const newPath = "work/gaps/G-0036-after.md"
	body := "---\nid: G-0036\ntitle: example\nstatus: open\n---\nbody text\n"
	commitFile(t, ctx, dir, oldPath, body)
	mustRun(t, ctx, dir, "branch", "trunk")
	mustRun(t, ctx, dir, "mv", oldPath, newPath)

	got, err := RenamesFromRef(ctx, dir, "trunk")
	if err != nil {
		t.Fatalf("RenamesFromRef: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("uncommitted rename should not appear in committed-since-merge-base view; got %+v", got)
	}
}

// TestRenamesFromRef_IgnoresParallelClonesG37Case pins the negative
// side of the G-0109 fix against the original G37 scenario: two
// independent commits on different branches each add a file claiming
// the same id at a different slug. The two files are byte-similar
// enough that git's plain rename heuristic would pair them, but
// neither branch's history contains a rename — they're true
// duplicate allocations. The merge-base..HEAD scope guarantees that
// only renames *this branch committed* surface, which excludes the
// parallel-add case the original trunk-collision rule must still
// catch.
func TestRenamesFromRef_IgnoresParallelClonesG37Case(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	// Establish a shared starting commit, then branch.
	commitFile(t, ctx, dir, "README.md", "shared\n")
	// Trunk's branch adds G-0001 at one slug, near-template body.
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "trunk")
	commitFile(t, ctx, dir, "work/gaps/G-0001-trunk-side.md", "---\nid: G-0001\ntitle: t\nstatus: open\n---\n\n## Problem\n\n")
	// HEAD (feature branch) returns to the merge-base and adds G-0001 at
	// a DIFFERENT slug with a near-identical template body — the G37
	// shape git's -M would otherwise pair as a rename.
	mustRun(t, ctx, dir, "checkout", "-q", "main")
	commitFile(t, ctx, dir, "work/gaps/G-0001-branch-side.md", "---\nid: G-0001\ntitle: b\nstatus: open\n---\n\n## Problem\n\n")

	got, err := RenamesFromRef(ctx, dir, "trunk")
	if err != nil {
		t.Fatalf("RenamesFromRef: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("parallel-clone G37 case must not register as a rename; got %+v", got)
	}
}

// TestRenamesFromRef_NoRenamesEmptyMap verifies that an unmodified
// working tree returns an empty (non-nil) map. Empty vs nil matters
// because the caller assigns into Tree.TrunkRenames; nil-vs-empty would
// be observable in tests that lookup keys.
func TestRenamesFromRef_NoRenamesEmptyMap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-0001-a.md", "x\n")
	mustRun(t, ctx, dir, "branch", "trunk")

	got, err := RenamesFromRef(ctx, dir, "trunk")
	if err != nil {
		t.Fatalf("RenamesFromRef: %v", err)
	}
	if got == nil {
		t.Fatal("RenamesFromRef returned nil map, want empty non-nil")
	}
	if len(got) != 0 {
		t.Errorf("RenamesFromRef on clean tree = %+v, want empty", got)
	}
}

// TestRenamesFromRef_AbsentRefReturnsNil verifies the degraded-graceful
// path: when the named ref doesn't resolve, the function returns
// (nil, nil) rather than erroring. This matches the trunk-collision
// rule's behavior — no trunk view means no cross-tree comparison.
func TestRenamesFromRef_AbsentRefReturnsNil(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "a.txt", "x")

	got, err := RenamesFromRef(ctx, dir, "refs/remotes/origin/nope")
	if err != nil {
		t.Fatalf("RenamesFromRef: %v", err)
	}
	if got != nil {
		t.Errorf("RenamesFromRef with absent ref = %+v, want nil", got)
	}
}

// initTestRepo creates a fresh git repo in a temp dir and returns
// its path. Commit identity is seeded by TestMain in setup_test.go
// (via os.Setenv) so this helper is t.Parallel-compatible — t.Setenv
// would panic under parallel execution.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	if err := Init(ctx, dir); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return dir
}

// commitFile writes content to path inside dir, stages, and commits.
func commitFile(t *testing.T, ctx context.Context, dir, path, content string) {
	t.Helper()
	writeFile(t, dir, path, content)
	mustRun(t, ctx, dir, "add", "--", path)
	mustRun(t, ctx, dir, "commit", "-q", "-m", "add "+path)
}

// writeFile materializes content at dir/path, creating parent dirs.
func writeFile(t *testing.T, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustRun(t *testing.T, ctx context.Context, dir string, args ...string) {
	t.Helper()
	if err := run(ctx, dir, args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

func mustOutput(t *testing.T, ctx context.Context, dir string, args ...string) string {
	t.Helper()
	out, err := output(ctx, dir, args...)
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return trimNewline(out)
}

func trimNewline(s string) string {
	if s != "" && s[len(s)-1] == '\n' {
		return s[:len(s)-1]
	}
	return s
}
