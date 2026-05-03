package verb_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/verb"
)

// applyTestRepo bootstraps a git repo with one tracked file under
// work/epics/E-01-foo/epic.md so the rollback tests have a path that
// exists at HEAD to mv from.
type applyTestRepo struct {
	root        string
	ctx         context.Context
	preCommit   string // SHA before the verb under test runs
	trackedPath string
}

func newApplyTestRepo(t *testing.T) *applyTestRepo {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	root := t.TempDir()
	ctx := context.Background()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	tracked := filepath.Join("work", "epics", "E-01-foo", "epic.md")
	full := filepath.Join(root, tracked)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("---\nid: E-01\n---\noriginal body\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(ctx, root, tracked); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(ctx, root, "seed", "", nil); err != nil {
		t.Fatal(err)
	}
	preCommit := headSHA(t, root)
	return &applyTestRepo{root: root, ctx: ctx, preCommit: preCommit, trackedPath: tracked}
}

func headSHA(t *testing.T, root string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func porcelain(t *testing.T, root string) string {
	t.Helper()
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

// --- happy path regression ---

// TestApply_HappyPath_OneCommitNoExtraIndexChurn proves the
// non-failure path: after Apply succeeds, exactly one new commit
// exists and the working tree is clean.
func TestApply_HappyPath_OneCommitNoExtraIndexChurn(t *testing.T) {
	r := newApplyTestRepo(t)
	plan := &verb.Plan{
		Subject: "test write",
		Trailers: []gitops.Trailer{
			{Key: "aiwf-verb", Value: "test"},
		},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: "work/epics/E-02-bar/epic.md", Content: []byte("---\nid: E-02\n---\n")},
		},
	}
	if err := verb.Apply(r.ctx, r.root, plan); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if got := porcelain(t, r.root); got != "" {
		t.Errorf("dirty after happy-path Apply: %q", got)
	}
	if headSHA(t, r.root) == r.preCommit {
		t.Error("HEAD did not advance; expected one new commit")
	}
}

// --- rollback on write failure ---

// TestApply_RollsBackOnWriteFailure is the load-bearing test for G2:
// when a write fails after a successful git mv, the staged mv must
// be rolled back so the working tree is exactly as it was before.
//
// Setup: source and destination share a parent dir, so git mv
// succeeds without pre-mkdir. Then a write to a blocked path fails,
// triggering the rollback path that keeps the staged mv around.
func TestApply_RollsBackOnWriteFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	r := newApplyTestRepo(t)

	// Create a directory with no write permission. The plan will try
	// to write into it, which must fail.
	noWrite := filepath.Join(r.root, "noWrite")
	if err := os.Mkdir(noWrite, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(noWrite, 0o755) })

	// Same-dir mv so it succeeds without parent prep.
	dest := filepath.Join(filepath.Dir(r.trackedPath), "epic-renamed.md")

	plan := &verb.Plan{
		Subject: "test rollback",
		Trailers: []gitops.Trailer{
			{Key: "aiwf-verb", Value: "test"},
		},
		Ops: []verb.FileOp{
			{Type: verb.OpMove, Path: r.trackedPath, NewPath: dest},
			{Type: verb.OpWrite, Path: filepath.Join("noWrite", "child", "blocked.md"), Content: []byte("nope")},
		},
	}

	err := verb.Apply(r.ctx, r.root, plan)
	if err == nil {
		t.Fatal("expected Apply to fail on unwritable target; got nil")
	}

	// HEAD must not have advanced.
	if got := headSHA(t, r.root); got != r.preCommit {
		t.Errorf("HEAD advanced from %s to %s; rollback should keep it at preCommit", r.preCommit, got)
	}

	// Working tree must be clean (mv staged + worktree change reverted,
	// any partial new file removed).
	if got := porcelain(t, r.root); got != "" {
		t.Errorf("dirty tree after rollback: %q", got)
	}

	// Original file must exist at its original path with original content.
	got := readFile(t, filepath.Join(r.root, r.trackedPath))
	if !bytes.Contains(got, []byte("original body")) {
		t.Errorf("original file content lost after rollback: %q", got)
	}
}

// TestApply_RollsBackOnGitMvFailure: when `git mv` fails (e.g. source
// not tracked), no commit and no leftover state.
func TestApply_RollsBackOnGitMvFailure(t *testing.T) {
	r := newApplyTestRepo(t)
	plan := &verb.Plan{
		Subject:  "test mv-fail",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			// Source doesn't exist → git mv fails.
			{Type: verb.OpMove, Path: "does/not/exist.md", NewPath: "work/x/y.md"},
		},
	}
	err := verb.Apply(r.ctx, r.root, plan)
	if err == nil {
		t.Fatal("expected Apply to fail on missing mv source")
	}
	if headSHA(t, r.root) != r.preCommit {
		t.Error("HEAD must not advance on mv failure")
	}
	if got := porcelain(t, r.root); got != "" {
		t.Errorf("tree must stay clean on mv failure: %q", got)
	}
}

// TestApply_RollsBackUntrackedNewFiles: a brand-new file written by
// the verb must be removed (not just unstaged) on rollback, otherwise
// the next aiwf invocation sees stale state.
func TestApply_RollsBackUntrackedNewFiles(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	r := newApplyTestRepo(t)

	// Write op #1 succeeds (creates a brand-new file). Write op #2
	// fails because its parent has no write permission.
	noWrite := filepath.Join(r.root, "blocked")
	if err := os.Mkdir(noWrite, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(noWrite, 0o755) })

	newFilePath := filepath.Join("work", "milestones", "M-001-new", "milestone.md")
	plan := &verb.Plan{
		Subject:  "test new-file rollback",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: newFilePath, Content: []byte("---\nid: M-001\n---\n")},
			{Type: verb.OpWrite, Path: filepath.Join("blocked", "x.md"), Content: []byte("blocked")},
		},
	}
	err := verb.Apply(r.ctx, r.root, plan)
	if err == nil {
		t.Fatal("expected failure")
	}
	if headSHA(t, r.root) != r.preCommit {
		t.Error("HEAD must not advance")
	}
	if got := porcelain(t, r.root); got != "" {
		t.Errorf("dirty tree after rollback: %q", got)
	}
	// The brand-new file must have been removed entirely.
	if _, err := os.Stat(filepath.Join(r.root, newFilePath)); !os.IsNotExist(err) {
		t.Errorf("new file should be removed on rollback; stat err = %v", err)
	}
}

// TestApply_PanicTriggersRollback: a panic mid-Apply must trigger
// the deferred rollback before the panic propagates. Provoke the
// panic by passing a nil Plan, which derefs in the for-range loop.
func TestApply_PanicTriggersRollback(t *testing.T) {
	r := newApplyTestRepo(t)
	defer func() {
		if got := recover(); got == nil {
			t.Error("expected panic to propagate")
		}
		if got := porcelain(t, r.root); got != "" {
			t.Errorf("dirty tree after panic-rollback: %q", got)
		}
		if headSHA(t, r.root) != r.preCommit {
			t.Error("HEAD must not advance on panic")
		}
	}()
	_ = verb.Apply(r.ctx, r.root, nil)
}

// TestApply_RollsBackOnCommitFailure: missing committer identity
// makes `git commit` fail; the rollback must still leave a clean
// tree.
func TestApply_RollsBackOnCommitFailure(t *testing.T) {
	r := newApplyTestRepo(t)
	// Override author/committer with empty values so git refuses to
	// commit. (newApplyTestRepo sets these; t.Setenv here overrides
	// just for this test.)
	t.Setenv("GIT_AUTHOR_NAME", "")
	t.Setenv("GIT_AUTHOR_EMAIL", "")
	t.Setenv("GIT_COMMITTER_NAME", "")
	t.Setenv("GIT_COMMITTER_EMAIL", "")
	// Block global config from supplying defaults.
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	dest := filepath.Join(filepath.Dir(r.trackedPath), "epic-renamed.md")
	plan := &verb.Plan{
		Subject:  "test commit-fail",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpMove, Path: r.trackedPath, NewPath: dest},
		},
	}
	if err := verb.Apply(r.ctx, r.root, plan); err == nil {
		t.Fatal("expected commit failure")
	}
	if got := porcelain(t, r.root); got != "" {
		t.Errorf("dirty tree after commit-fail rollback: %q", got)
	}
	if headSHA(t, r.root) != r.preCommit {
		t.Error("HEAD must not advance on commit failure")
	}
}

// TestApply_RollsBackOnGitAddFailure: a path that doesn't exist on
// disk (e.g. write was somehow skipped) makes `git add` fail. We
// can simulate this by writing a content of length 0 to a path then
// removing it before Apply gets to git add — but Apply doesn't
// expose hooks. Instead, exploit the fact that `git add --` with a
// path containing a NUL byte fails. NUL bytes in paths are rejected
// by the OS layer, so we can't actually create such a file. Use
// the simpler trigger: an OpWrite whose path is "." (the repo
// root) — os.WriteFile fails with "is a directory", but that's a
// write error, not git-add. This branch is therefore exercised
// only by deliberate corruption; we skip it here and rely on
// inspection.
func TestApply_RollsBackOnGitAddFailure(t *testing.T) {
	t.Skip("git-add failure is defensive; not reachable from a clean unit test")
}

// TestApply_DedupesTouchedPaths: when two ops touch the same path
// (e.g. a move whose dest is also written to), the rollback must
// not pass the same path to git restore twice — that would error or
// emit duplicate warnings.
func TestApply_DedupesTouchedPaths(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	r := newApplyTestRepo(t)

	// move + write to the same dest, then a failing write to force
	// rollback. The rollback must dedupe the touched paths.
	dest := filepath.Join(filepath.Dir(r.trackedPath), "epic-renamed.md")
	noWrite := filepath.Join(r.root, "noWrite")
	if err := os.Mkdir(noWrite, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(noWrite, 0o755) })

	plan := &verb.Plan{
		Subject:  "test dedupe",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpMove, Path: r.trackedPath, NewPath: dest},
			{Type: verb.OpWrite, Path: dest, Content: []byte("rewritten")},
			{Type: verb.OpWrite, Path: filepath.Join("noWrite", "fail.md"), Content: []byte("nope")},
		},
	}
	if err := verb.Apply(r.ctx, r.root, plan); err == nil {
		t.Fatal("expected failure")
	}
	if got := porcelain(t, r.root); got != "" {
		t.Errorf("dirty tree after rollback: %q", got)
	}
}

// --- G34: pre-staged isolation + conflict guard ---

// TestApply_PreservesUnrelatedStagedChanges is the load-bearing test
// for G34: when the user has unrelated staged changes (e.g. an
// in-progress patch from another tool), Apply's commit must capture
// only the verb's paths, leaving the user's staged work intact in
// the index.
//
// Reproducer: a user stages `unrelated.go`, then runs a verb that
// writes `work/epics/E-02-bar/epic.md`. After Apply, exactly one
// commit lands carrying only the verb's path; `unrelated.go` is
// still staged for the user's next commit.
func TestApply_PreservesUnrelatedStagedChanges(t *testing.T) {
	r := newApplyTestRepo(t)

	// User pre-stages an unrelated file.
	unrelated := filepath.Join(r.root, "unrelated.go")
	if err := os.WriteFile(unrelated, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(r.ctx, r.root, "unrelated.go"); err != nil {
		t.Fatal(err)
	}

	// Verb writes a different path.
	plan := &verb.Plan{
		Subject:  "test isolated commit",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: "work/epics/E-02-bar/epic.md", Content: []byte("---\nid: E-02\n---\n")},
		},
	}
	if err := verb.Apply(r.ctx, r.root, plan); err != nil {
		t.Fatalf("apply: %v", err)
	}

	// HEAD's commit must contain *only* the verb's path.
	cmd := exec.Command("git", "show", "--name-only", "--format=", "HEAD")
	cmd.Dir = r.root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git show: %v", err)
	}
	files := strings.Fields(strings.TrimSpace(string(out)))
	if len(files) != 1 || files[0] != "work/epics/E-02-bar/epic.md" {
		t.Errorf("commit captured wrong path set: got %v, want only [work/epics/E-02-bar/epic.md]", files)
	}

	// `unrelated.go` must still be staged (not yet committed).
	statusCmd := exec.Command("git", "status", "--porcelain", "--", "unrelated.go")
	statusCmd.Dir = r.root
	statusOut, err := statusCmd.Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	if got := strings.TrimSpace(string(statusOut)); got != "A  unrelated.go" {
		t.Errorf("unrelated staged file lost or modified: got porcelain %q, want %q", got, "A  unrelated.go")
	}
}

// TestApply_RefusesConflictingPreStagedPath: when the user has
// already staged content for a path the verb is about to write,
// Apply must refuse before any disk mutation. The two intents (the
// user's staged content, the verb's computed content) cannot both
// land in the verb's commit; silently picking the verb's content
// would destroy the user's work.
//
// Setup: user pre-stages a file at the same path the verb will
// write. Apply must error, must not advance HEAD, and must leave the
// user's staged content untouched.
func TestApply_RefusesConflictingPreStagedPath(t *testing.T) {
	r := newApplyTestRepo(t)

	// Pre-stage content at the path the verb will write.
	conflictPath := "work/epics/E-02-bar/epic.md"
	full := filepath.Join(r.root, conflictPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	userContent := []byte("user's hand-edited content\n")
	if err := os.WriteFile(full, userContent, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(r.ctx, r.root, conflictPath); err != nil {
		t.Fatal(err)
	}

	plan := &verb.Plan{
		Subject:  "test conflict refusal",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: conflictPath, Content: []byte("verb's computed content\n")},
		},
	}

	err := verb.Apply(r.ctx, r.root, plan)
	if err == nil {
		t.Fatal("expected Apply to refuse on conflicting pre-staged path; got nil")
	}
	if !strings.Contains(err.Error(), conflictPath) {
		t.Errorf("error message must name the conflicting path: %v", err)
	}
	if !strings.Contains(err.Error(), "pre-staged") {
		t.Errorf("error message must explain the conflict: %v", err)
	}

	// HEAD must not have advanced.
	if got := headSHA(t, r.root); got != r.preCommit {
		t.Errorf("HEAD advanced; conflict guard must fire before any commit")
	}

	// The user's staged content must survive untouched on disk.
	got := readFile(t, full)
	if !bytes.Equal(got, userContent) {
		t.Errorf("verb wrote over user's content despite the guard: got %q, want %q", got, userContent)
	}
}

// TestApply_AllowEmptyPreservesUnrelatedStaged: an authorize /
// audit-only plan (AllowEmpty=true, no Ops) records an event with
// no file diff. With the stash-isolation fix, pre-existing staged
// changes are pushed before the commit, the empty commit lands with
// trailers only, then the stash is popped — the user's staged work
// is back in the index for their next commit.
func TestApply_AllowEmptyPreservesUnrelatedStaged(t *testing.T) {
	r := newApplyTestRepo(t)

	// User stages an unrelated file.
	unrelated := filepath.Join(r.root, "unrelated.go")
	if err := os.WriteFile(unrelated, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(r.ctx, r.root, "unrelated.go"); err != nil {
		t.Fatal(err)
	}

	plan := &verb.Plan{
		Subject:    "aiwf authorize E-01 [test]",
		Trailers:   []gitops.Trailer{{Key: "aiwf-verb", Value: "authorize"}},
		AllowEmpty: true,
	}
	if err := verb.Apply(r.ctx, r.root, plan); err != nil {
		t.Fatalf("apply allow-empty: %v", err)
	}

	// HEAD's commit must have no path changes.
	cmd := exec.Command("git", "show", "--name-only", "--format=", "HEAD")
	cmd.Dir = r.root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git show: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "" {
		t.Errorf("allow-empty commit captured paths: %q", got)
	}

	// User's staged file must be back in the index after the pop.
	statusCmd := exec.Command("git", "status", "--porcelain", "--", "unrelated.go")
	statusCmd.Dir = r.root
	statusOut, _ := statusCmd.Output()
	if got := strings.TrimSpace(string(statusOut)); got != "A  unrelated.go" {
		t.Errorf("user's staged file lost across allow-empty verb: porcelain %q", got)
	}
}

// TestApply_AllowEmptyOnCleanIndex: the positive path — when the
// index is clean, an allow-empty plan still commits trailers-only
// the way authorize / --audit-only have always done.
func TestApply_AllowEmptyOnCleanIndex(t *testing.T) {
	r := newApplyTestRepo(t)
	plan := &verb.Plan{
		Subject:    "aiwf authorize E-01 [test]",
		Trailers:   []gitops.Trailer{{Key: "aiwf-verb", Value: "authorize"}},
		AllowEmpty: true,
	}
	if err := verb.Apply(r.ctx, r.root, plan); err != nil {
		t.Fatalf("apply allow-empty on clean index: %v", err)
	}
	cmd := exec.Command("git", "show", "--name-only", "--format=", "HEAD")
	cmd.Dir = r.root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git show: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "" {
		t.Errorf("allow-empty commit captured paths: %q", got)
	}
}
