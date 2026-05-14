package gitops

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
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
