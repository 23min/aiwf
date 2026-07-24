package gitops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCommitMessage(t *testing.T) {
	t.Parallel()
	got := CommitMessage("add milestone M-007", "", []Trailer{
		{Key: "aiwf-verb", Value: "add"},
		{Key: "aiwf-entity", Value: "M-007"},
		{Key: "aiwf-actor", Value: "human/peter"},
	})
	want := "add milestone M-007\n\naiwf-verb: add\naiwf-entity: M-007\naiwf-actor: human/peter\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestCommitMessage_NoTrailers(t *testing.T) {
	t.Parallel()
	got := CommitMessage("subject only", "", nil)
	if got != "subject only\n" {
		t.Errorf("got %q, want %q", got, "subject only\n")
	}
}

func TestCommitMessage_WithBody(t *testing.T) {
	t.Parallel()
	got := CommitMessage("aiwf cancel M-002 -> cancelled", "scope folded into M-001", []Trailer{
		{Key: "aiwf-verb", Value: "cancel"},
		{Key: "aiwf-entity", Value: "M-002"},
	})
	want := "aiwf cancel M-002 -> cancelled\n\nscope folded into M-001\n\naiwf-verb: cancel\naiwf-entity: M-002\n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestCommitMessage_BodyTrimmed(t *testing.T) {
	t.Parallel()
	// Whitespace at body edges is trimmed; an all-whitespace body produces no body section.
	got := CommitMessage("subject", "   ", nil)
	if got != "subject\n" {
		t.Errorf("got %q, want %q", got, "subject\n")
	}
}

func TestParseTrailers(t *testing.T) {
	t.Parallel()
	out := "aiwf-verb: add\naiwf-entity: M-007\n\naiwf-actor: human/peter\n"
	got := ParseTrailers(out)
	want := []Trailer{
		{Key: "aiwf-verb", Value: "add"},
		{Key: "aiwf-entity", Value: "M-007"},
		{Key: "aiwf-actor", Value: "human/peter"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestEndToEnd_InitAddMvCommit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()

	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	if !IsRepo(ctx, root) {
		t.Fatal("IsRepo false after Init")
	}

	if err := os.WriteFile(filepath.Join(root, "alpha.md"), []byte("alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "alpha.md"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := Commit(ctx, root, "first commit", "", []Trailer{
		{Key: "aiwf-verb", Value: "add"},
		{Key: "aiwf-entity", Value: "E-01"},
		{Key: "aiwf-actor", Value: "human/peter"},
	}); err != nil {
		t.Fatalf("commit: %v", err)
	}

	subj, err := HeadSubject(ctx, root)
	if err != nil {
		t.Fatalf("subject: %v", err)
	}
	if subj != "first commit" {
		t.Errorf("subject = %q, want first commit", subj)
	}

	tr, err := HeadTrailers(ctx, root)
	if err != nil {
		t.Fatalf("trailers: %v", err)
	}
	wantTrailers := []Trailer{
		{Key: "aiwf-verb", Value: "add"},
		{Key: "aiwf-entity", Value: "E-01"},
		{Key: "aiwf-actor", Value: "human/peter"},
	}
	if diff := cmp.Diff(wantTrailers, tr); diff != "" {
		t.Errorf("trailer mismatch (-want +got):\n%s", diff)
	}

	// Now Mv.
	if err := Mv(ctx, root, "alpha.md", "beta.md"); err != nil {
		t.Fatalf("mv: %v", err)
	}
	if err := Commit(ctx, root, "rename to beta", "", []Trailer{
		{Key: "aiwf-verb", Value: "rename"},
	}); err != nil {
		t.Fatalf("second commit: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "alpha.md")); err == nil {
		t.Error("alpha.md still present after mv")
	}
	if _, err := os.Stat(filepath.Join(root, "beta.md")); err != nil {
		t.Errorf("beta.md missing after mv: %v", err)
	}
}

// TestReadFromHEAD covers the branches the helper distinguishes:
// (1) path exists at HEAD → returns the bytes; (2) path does not
// exist at HEAD (either because HEAD has no such path, or because
// the repo has no HEAD yet) → returns (nil, nil). The two
// "no version" cases collapse to the same caller-facing semantic
// because `git rev-parse --verify --quiet HEAD:<path>` returns
// exit 1 silently in both — and a caller in bless mode treats both
// as "use aiwf add to create this entity for the first time."
func TestReadFromHEAD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := Init(ctx, root); err != nil {
		t.Fatal(err)
	}

	// Empty repo (no HEAD): treated as "no version" (nil, nil).
	got, err := ReadFromHEAD(ctx, root, "alpha.md")
	if err != nil {
		t.Errorf("expected nil error in empty repo; got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil bytes in empty repo; got %q", got)
	}

	// Commit a file.
	if writeErr := os.WriteFile(filepath.Join(root, "alpha.md"), []byte("alpha v1\n"), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	if addErr := Add(ctx, root, "alpha.md"); addErr != nil {
		t.Fatal(addErr)
	}
	if commitErr := Commit(ctx, root, "add alpha", "", []Trailer{{Key: "aiwf-verb", Value: "add"}}); commitErr != nil {
		t.Fatal(commitErr)
	}

	// (1) path exists at HEAD → bytes match the committed content.
	got, err = ReadFromHEAD(ctx, root, "alpha.md")
	if err != nil {
		t.Fatalf("ReadFromHEAD: %v", err)
	}
	if string(got) != "alpha v1\n" {
		t.Errorf("HEAD content = %q, want %q", got, "alpha v1\n")
	}

	// Modify the working copy — ReadFromHEAD still returns the v1 bytes.
	if writeErr := os.WriteFile(filepath.Join(root, "alpha.md"), []byte("alpha v2 (uncommitted)\n"), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	got, err = ReadFromHEAD(ctx, root, "alpha.md")
	if err != nil {
		t.Fatalf("ReadFromHEAD: %v", err)
	}
	if string(got) != "alpha v1\n" {
		t.Errorf("HEAD content after working-copy edit = %q, want unchanged %q", got, "alpha v1\n")
	}

	// (2) path does not exist at HEAD → (nil, nil).
	got, err = ReadFromHEAD(ctx, root, "never-committed.md")
	if err != nil {
		t.Errorf("expected nil error for path-not-in-HEAD; got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil bytes for path-not-in-HEAD; got %q", got)
	}
}

// TestCommitAllowEmpty: a commit with no staged changes lands when
// allow-empty is in effect, with the trailer set intact. Used by
// `aiwf authorize` and the `--audit-only` recovery mode (plan step 5b).
func TestCommitAllowEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()

	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	// Seed the repo with one tracked commit so HEAD exists and the
	// allow-empty commit is the second one (matching real-world use).
	if err := os.WriteFile(filepath.Join(root, "seed.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "seed.md"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatalf("seed commit: %v", err)
	}

	// Plain Commit with nothing staged would fail; CommitAllowEmpty
	// must succeed.
	if err := CommitAllowEmpty(ctx, root, "aiwf authorize E-01 --to ai/claude", "implement E-01", []Trailer{
		{Key: "aiwf-verb", Value: "authorize"},
		{Key: "aiwf-entity", Value: "E-01"},
		{Key: "aiwf-actor", Value: "human/peter"},
		{Key: "aiwf-to", Value: "ai/claude"},
		{Key: "aiwf-scope", Value: "opened"},
	}); err != nil {
		t.Fatalf("CommitAllowEmpty: %v", err)
	}

	subj, err := HeadSubject(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if subj != "aiwf authorize E-01 --to ai/claude" {
		t.Errorf("HEAD subject = %q", subj)
	}
	tr, err := HeadTrailers(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	want := []Trailer{
		{Key: "aiwf-verb", Value: "authorize"},
		{Key: "aiwf-entity", Value: "E-01"},
		{Key: "aiwf-actor", Value: "human/peter"},
		{Key: "aiwf-to", Value: "ai/claude"},
		{Key: "aiwf-scope", Value: "opened"},
	}
	if diff := cmp.Diff(want, tr); diff != "" {
		t.Errorf("trailers mismatch (-want +got):\n%s", diff)
	}
}

func TestIsRepo_FalseInPlainDir(t *testing.T) {
	t.Parallel()
	if IsRepo(context.Background(), t.TempDir()) {
		t.Error("IsRepo true in non-repo tmpdir")
	}
}

func TestRun_ErrorIncludesStderr(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Trying to commit in a non-repo directory should fail with
	// stderr embedded in the error message.
	root := t.TempDir()
	err := Commit(ctx, root, "wat", "", nil)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(err.Error(), "git commit") {
		t.Errorf("error %q should mention git commit", err.Error())
	}
}

// TestStagedPaths reports paths in the index that differ from HEAD.
// Used by Apply's conflict guard; the load-bearing assertions are
// "clean index → nil/empty" and "staged file → file's path returned."
func TestStagedPaths(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()

	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "seed.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "seed.md"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Clean index → nil.
	clean, cleanErr := StagedPaths(ctx, root)
	if cleanErr != nil {
		t.Fatalf("StagedPaths clean: %v", cleanErr)
	}
	if len(clean) != 0 {
		t.Errorf("clean index returned %v, want empty", clean)
	}

	// Stage two files; both must appear.
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.md"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "a.md", "b.md"); err != nil {
		t.Fatalf("add: %v", err)
	}
	staged, stagedErr := StagedPaths(ctx, root)
	if stagedErr != nil {
		t.Fatalf("StagedPaths after stage: %v", stagedErr)
	}
	want := []string{"a.md", "b.md"}
	if diff := cmp.Diff(want, staged); diff != "" {
		t.Errorf("StagedPaths mismatch (-want +got):\n%s", diff)
	}
}

// TestDirtyPaths pins AC-2 of M-0276: DirtyPaths returns every repo-relative
// path that differs from HEAD in the working tree — unstaged modifications,
// staged additions, and untracked (non-ignored) files alike — while excluding
// gitignored paths and returning empty on a clean tree. It is the raw material
// the red/green diff-shape gate classifies.
func TestDirtyPaths(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()

	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	// Seed commit: a tracked file to modify, a tracked file to leave alone,
	// and a .gitignore so *.log is excluded from the dirty set.
	for name, content := range map[string]string{
		"mod.md":     "original\n",
		"tracked.md": "tracked\n",
		".gitignore": "*.log\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Add(ctx, root, "mod.md", "tracked.md", ".gitignore"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Clean tree → empty.
	clean, cleanErr := DirtyPaths(ctx, root)
	if cleanErr != nil {
		t.Fatalf("DirtyPaths clean: %v", cleanErr)
	}
	if len(clean) != 0 {
		t.Errorf("clean tree returned %v, want empty", clean)
	}

	// Dirty the tree four ways:
	//   - unstaged modification of a tracked file
	//   - staged new file
	//   - untracked new file
	//   - ignored new file (must be excluded)
	if err := os.WriteFile(filepath.Join(root, "mod.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "staged_new.md"), []byte("s\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "staged_new.md"); err != nil {
		t.Fatalf("add staged_new: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "untracked_new.md"), []byte("u\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored.log"), []byte("noise\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := DirtyPaths(ctx, root)
	if err != nil {
		t.Fatalf("DirtyPaths dirty: %v", err)
	}
	// Sorted, deduped; ignored.log and the unchanged tracked.md are absent.
	want := []string{"mod.md", "staged_new.md", "untracked_new.md"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DirtyPaths mismatch (-want +got):\n%s", diff)
	}
}

// TestDirtyPaths_NonRepoErrors covers DirtyPaths' error path: a workdir that is
// not a git repository makes the first git listing fail, and the error
// propagates rather than being swallowed as a clean tree.
func TestDirtyPaths_NonRepoErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	if _, err := DirtyPaths(ctx, t.TempDir()); err == nil {
		t.Fatal("DirtyPaths on a non-repo dir: want error, got nil")
	}
}

// TestDirtyPaths_Rename covers the rename edge the M-0276 spec flagged as a
// potential false-positive: a working-tree rename of a tracked file surfaces as
// dirty — the old path as a tracked deletion (git diff HEAD) and the new path as
// an untracked addition (ls-files) — so a renamed implementation file still
// counts as dirty at the red/green gate rather than slipping through.
func TestDirtyPaths_Rename(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "old.md"), []byte("body\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "old.md"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatalf("commit: %v", err)
	}
	// Rename on disk without staging: old.md becomes a tracked deletion and
	// new.md an untracked addition — both must surface.
	if err := os.Rename(filepath.Join(root, "old.md"), filepath.Join(root, "new.md")); err != nil {
		t.Fatal(err)
	}
	got, err := DirtyPaths(ctx, root)
	if err != nil {
		t.Fatalf("DirtyPaths: %v", err)
	}
	want := []string{"new.md", "old.md"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DirtyPaths after rename mismatch (-want +got):\n%s", diff)
	}
}

// TestHooksDir covers the three states HooksDir distinguishes:
// `core.hooksPath` unset (fall back to <gitDir>/hooks), set to an
// absolute path (returned verbatim), set to a relative path
// (resolved against workdir). G-048.
func TestHooksDir(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("unset falls back to gitDir/hooks", func(t *testing.T) {
		root := t.TempDir()
		if err := Init(ctx, root); err != nil {
			t.Fatalf("init: %v", err)
		}
		got, err := HooksDir(ctx, root)
		if err != nil {
			t.Fatalf("HooksDir: %v", err)
		}
		gitDir, err := GitDir(ctx, root)
		if err != nil {
			t.Fatalf("GitDir: %v", err)
		}
		want := filepath.Join(gitDir, "hooks")
		if got != want {
			t.Errorf("HooksDir = %q, want %q", got, want)
		}
	})

	t.Run("absolute path returned verbatim", func(t *testing.T) {
		root := t.TempDir()
		if err := Init(ctx, root); err != nil {
			t.Fatalf("init: %v", err)
		}
		want := filepath.Join(t.TempDir(), "absolute-hooks")
		if err := run(ctx, root, "config", "core.hooksPath", want); err != nil {
			t.Fatalf("git config: %v", err)
		}
		got, err := HooksDir(ctx, root)
		if err != nil {
			t.Fatalf("HooksDir: %v", err)
		}
		if got != want {
			t.Errorf("HooksDir = %q, want %q", got, want)
		}
	})

	t.Run("relative path resolves against workdir", func(t *testing.T) {
		root := t.TempDir()
		if err := Init(ctx, root); err != nil {
			t.Fatalf("init: %v", err)
		}
		if err := run(ctx, root, "config", "core.hooksPath", "scripts/git-hooks"); err != nil {
			t.Fatalf("git config: %v", err)
		}
		got, err := HooksDir(ctx, root)
		if err != nil {
			t.Fatalf("HooksDir: %v", err)
		}
		// HooksDir canonicalizes workdir via EvalSymlinks so the
		// result matches git's own form (e.g. /private/var/... on
		// macOS rather than the symlink alias /var/...).
		canonicalRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatalf("EvalSymlinks: %v", err)
		}
		want := filepath.Join(canonicalRoot, "scripts", "git-hooks")
		if got != want {
			t.Errorf("HooksDir = %q, want %q", got, want)
		}
	})

	t.Run("non-git workdir surfaces error (not silent)", func(t *testing.T) {
		// commonGitDir and InWorktree are reached only via HooksDir
		// (and update.Run for InWorktree). When git can't resolve the
		// workdir, the error propagates rather than silently falling
		// back. Covers commonGitDir's error return and InWorktree's
		// GitDir-error branch.
		notARepo := t.TempDir()
		if _, err := HooksDir(ctx, notARepo); err == nil {
			t.Errorf("HooksDir against non-git workdir should error, got nil")
		}
		if _, err := InWorktree(ctx, notARepo); err == nil {
			t.Errorf("InWorktree against non-git workdir should error, got nil")
		}
	})

	t.Run("worktree falls back to common-dir hooks (G-0136)", func(t *testing.T) {
		main := t.TempDir()
		if err := Init(ctx, main); err != nil {
			t.Fatalf("init main: %v", err)
		}
		// Worktree creation needs HEAD; seed an initial commit.
		if err := os.WriteFile(filepath.Join(main, "README.md"), []byte("seed\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, args := range [][]string{
			{"config", "user.email", "test@example.com"},
			{"config", "user.name", "test"},
			{"add", "README.md"},
			{"commit", "-m", "seed"},
		} {
			if err := run(ctx, main, args...); err != nil {
				t.Fatalf("git %v: %v", args, err)
			}
		}
		worktreePath := filepath.Join(t.TempDir(), "wt")
		if err := run(ctx, main, "worktree", "add", "-b", "feat", worktreePath); err != nil {
			t.Fatalf("worktree add: %v", err)
		}
		got, err := HooksDir(ctx, worktreePath)
		if err != nil {
			t.Fatalf("HooksDir from worktree: %v", err)
		}
		mainGitDir, err := GitDir(ctx, main)
		if err != nil {
			t.Fatalf("GitDir main: %v", err)
		}
		want := filepath.Join(mainGitDir, "hooks")
		if got != want {
			t.Errorf("HooksDir from worktree = %q, want %q (shared hooks dir per G-0136; the per-worktree path is inert)", got, want)
		}
	})

	t.Run("empty value falls back to gitDir/hooks", func(t *testing.T) {
		root := t.TempDir()
		if err := Init(ctx, root); err != nil {
			t.Fatalf("init: %v", err)
		}
		// `git config core.hooksPath ""` sets the key to empty;
		// HooksDir should treat empty the same as unset.
		if err := run(ctx, root, "config", "core.hooksPath", ""); err != nil {
			t.Fatalf("git config: %v", err)
		}
		got, err := HooksDir(ctx, root)
		if err != nil {
			t.Fatalf("HooksDir: %v", err)
		}
		gitDir, err := GitDir(ctx, root)
		if err != nil {
			t.Fatalf("GitDir: %v", err)
		}
		want := filepath.Join(gitDir, "hooks")
		if got != want {
			t.Errorf("HooksDir = %q, want %q", got, want)
		}
	})
}

// TestRunPostCommitHook_HooksDirFails exercises RunPostCommitHook's own
// error branch directly: a non-repo workdir makes HooksDir fail before
// any hook-file lookup happens.
func TestRunPostCommitHook_HooksDirFails(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()

	err := RunPostCommitHook(ctx, root)
	if err == nil {
		t.Fatal("want error outside a git repository, got nil")
	}
	if !strings.Contains(err.Error(), "resolving hooks dir") {
		t.Errorf("error %q should mention resolving hooks dir", err.Error())
	}
}

// TestRunPostCommitHook_NoHookInstalled is a no-op: no post-commit hook
// file at all is not an error, mirroring git's own silent skip.
func TestRunPostCommitHook_NoHookInstalled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}

	if err := RunPostCommitHook(ctx, root); err != nil {
		t.Errorf("RunPostCommitHook with no hook installed = %v, want nil", err)
	}
}

// TestRunPostCommitHook_NonExecutableHookIsSkipped: a post-commit file
// exists but lacks the executable bit — git itself silently ignores a
// non-executable hook file, and so does this.
func TestRunPostCommitHook_NonExecutableHookIsSkipped(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	gitDir, err := GitDir(ctx, root)
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	hookPath := filepath.Join(gitDir, "hooks", "post-commit")
	marker := filepath.Join(root, "hook-ran.marker")
	script := "#!/bin/sh\ntouch '" + marker + "'\n"
	if err := os.WriteFile(hookPath, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RunPostCommitHook(ctx, root); err != nil {
		t.Errorf("RunPostCommitHook with a non-executable hook = %v, want nil", err)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Errorf("non-executable hook ran (marker present); stat err = %v", statErr)
	}
}

// TestRunPostCommitHook_ExecutesInstalledHook is the load-bearing
// positive case: an installed, executable post-commit hook actually
// runs, with workdir as its cwd. Pins M-0186's fix for the STATUS.md
// regeneration hook (G-0112) losing its trigger once verb.Apply moved
// to gitops.CommitTree, which fires no git hooks at all.
func TestRunPostCommitHook_ExecutesInstalledHook(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	gitDir, err := GitDir(ctx, root)
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	hookPath := filepath.Join(gitDir, "hooks", "post-commit")
	marker := filepath.Join(root, "hook-ran.marker")
	script := "#!/bin/sh\npwd > '" + marker + "'\n"
	err = os.WriteFile(hookPath, []byte(script), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	err = RunPostCommitHook(ctx, root)
	if err != nil {
		t.Fatalf("RunPostCommitHook: %v", err)
	}
	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("hook did not run (marker missing): %v", err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	canonicalGot, err := filepath.EvalSymlinks(strings.TrimSpace(string(got)))
	if err != nil {
		t.Fatal(err)
	}
	if canonicalGot != canonicalRoot {
		t.Errorf("hook ran with cwd %q, want %q", canonicalGot, canonicalRoot)
	}
}

// TestRunPostCommitHook_ToleratesHookFailure: per githooks(5), a
// post-commit hook's exit status is informational only. A hook that
// exits non-zero must not surface as an error.
func TestRunPostCommitHook_ToleratesHookFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	gitDir, err := GitDir(ctx, root)
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	hookPath := filepath.Join(gitDir, "hooks", "post-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := RunPostCommitHook(ctx, root); err != nil {
		t.Errorf("RunPostCommitHook with a failing hook = %v, want nil (exit status is informational only)", err)
	}
}
