package gitops

import (
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
