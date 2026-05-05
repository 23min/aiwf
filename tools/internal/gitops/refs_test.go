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
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "a.txt", "hello")
	_, err := LsTreePaths(ctx, dir, "refs/remotes/origin/main")
	if !errors.Is(err, ErrRefNotFound) {
		t.Fatalf("expected ErrRefNotFound, got %v", err)
	}
}

func TestLsTreePaths_EmptyTree(t *testing.T) {
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
	ctx := context.Background()
	dir := initTestRepo(t)
	commitFile(t, ctx, dir, "a.txt", "1")

	_, err := IsAncestor(ctx, dir, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "HEAD")
	if err == nil {
		t.Error("IsAncestor on a SHA that doesn't exist should error, got nil")
	}
}

// initTestRepo creates a fresh git repo in a temp dir and returns its
// path. It also sets a deterministic commit identity via t.Setenv so
// tests don't depend on the host's git config.
func initTestRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.invalid")
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
