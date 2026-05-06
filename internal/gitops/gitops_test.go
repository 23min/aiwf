package gitops

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// gitTestEnv sets the env vars git needs to author commits without
// reading a user-level config (which CI and t.TempDir won't have).
func gitTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
}

func TestCommitMessage(t *testing.T) {
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
	got := CommitMessage("subject only", "", nil)
	if got != "subject only\n" {
		t.Errorf("got %q, want %q", got, "subject only\n")
	}
}

func TestCommitMessage_WithBody(t *testing.T) {
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
	// Whitespace at body edges is trimmed; an all-whitespace body produces no body section.
	got := CommitMessage("subject", "   ", nil)
	if got != "subject\n" {
		t.Errorf("got %q, want %q", got, "subject\n")
	}
}

func TestParseTrailers(t *testing.T) {
	out := "aiwf-verb: add\naiwf-entity: M-007\n\naiwf-actor: human/peter\n"
	got := parseTrailers(out)
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
	gitTestEnv(t)
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

// TestCommitAllowEmpty: a commit with no staged changes lands when
// allow-empty is in effect, with the trailer set intact. Used by
// `aiwf authorize` and the `--audit-only` recovery mode (plan step 5b).
func TestCommitAllowEmpty(t *testing.T) {
	gitTestEnv(t)
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
	if IsRepo(context.Background(), t.TempDir()) {
		t.Error("IsRepo true in non-repo tmpdir")
	}
}

func TestRun_ErrorIncludesStderr(t *testing.T) {
	gitTestEnv(t)
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

// TestStashStaged_PushPopRoundTrip is the gitops-level pin for the
// G34 stash-isolation primitive: StashStaged pushes only the staged
// part of the index, leaving the worktree alone; StashPop restores
// the staging exactly. This is the contract verb.Apply relies on to
// isolate the user's pre-staged work from a verb's commit while
// letting the verb's normal `git commit` flow (and any pre-commit
// hooks that auto-add files) run unchanged.
func TestStashStaged_PushPopRoundTrip(t *testing.T) {
	gitTestEnv(t)
	ctx := context.Background()
	root := t.TempDir()

	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "seed.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "seed.md"); err != nil {
		t.Fatalf("add seed: %v", err)
	}
	if err := Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatalf("seed commit: %v", err)
	}

	// Stage a new file as if the user had work in flight.
	userPath := filepath.Join(root, "user.md")
	userContent := []byte("user wip\n")
	if err := os.WriteFile(userPath, userContent, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "user.md"); err != nil {
		t.Fatalf("add user: %v", err)
	}

	// Push the stage onto the stash.
	if err := StashStaged(ctx, root, "test push"); err != nil {
		t.Fatalf("StashStaged: %v", err)
	}

	// After push, the index must be clean of the user's stage.
	postPush, postPushErr := StagedPaths(ctx, root)
	if postPushErr != nil {
		t.Fatalf("StagedPaths after push: %v", postPushErr)
	}
	if len(postPush) != 0 {
		t.Errorf("StashStaged left paths staged: %v", postPush)
	}

	// Simulate the verb's commit landing — write and commit an
	// unrelated file. This is the production scenario the stash is
	// meant to support.
	if err := os.WriteFile(filepath.Join(root, "verb.md"), []byte("verb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "verb.md"); err != nil {
		t.Fatalf("add verb: %v", err)
	}
	if err := Commit(ctx, root, "verb commit", "", nil); err != nil {
		t.Fatalf("verb commit: %v", err)
	}

	// HEAD captured only verb.md — user's stash content is not in HEAD.
	headFiles, headErr := output(ctx, root, "show", "--name-only", "--format=", "HEAD")
	if headErr != nil {
		t.Fatal(headErr)
	}
	files := strings.Fields(strings.TrimSpace(headFiles))
	if len(files) != 1 || files[0] != "verb.md" {
		t.Errorf("HEAD captured wrong paths: %v, want only [verb.md]", files)
	}

	// Pop the stash; user's stage must be back.
	if err := StashPop(ctx, root); err != nil {
		t.Fatalf("StashPop: %v", err)
	}
	postPop, postPopErr := StagedPaths(ctx, root)
	if postPopErr != nil {
		t.Fatalf("StagedPaths after pop: %v", postPopErr)
	}
	want := []string{"user.md"}
	if diff := cmp.Diff(want, postPop); diff != "" {
		t.Errorf("StashPop did not restore user's stage (-want +got):\n%s", diff)
	}

	// Worktree content of the popped file matches what was staged.
	gotContent, readErr := os.ReadFile(userPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(gotContent, userContent) {
		t.Errorf("worktree content after pop: %q, want %q", gotContent, userContent)
	}
}

// TestStagedPaths reports paths in the index that differ from HEAD.
// Used by Apply's conflict guard; the load-bearing assertions are
// "clean index → nil/empty" and "staged file → file's path returned."
func TestStagedPaths(t *testing.T) {
	gitTestEnv(t)
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

// TestHooksDir covers the three states HooksDir distinguishes:
// `core.hooksPath` unset (fall back to <gitDir>/hooks), set to an
// absolute path (returned verbatim), set to a relative path
// (resolved against workdir). G-048.
func TestHooksDir(t *testing.T) {
	gitTestEnv(t)
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
