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
