package verb_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
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
	// GIT_{AUTHOR,COMMITTER}_{NAME,EMAIL} are seeded once in TestMain
	// (setup_test.go) — using t.Setenv here would panic under t.Parallel.
	root := t.TempDir()
	ctx := context.Background()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	tracked := filepath.Join("work", "epics", "E-0001-foo", "epic.md")
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
	t.Parallel()
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
	t.Parallel()
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

// TestApply_RollsBackOnCaptureWriteFailure: captureWrite's own
// generic-error branch fires when an OpWrite target exists but isn't
// readable (distinct from "doesn't exist yet", which is the common
// case and returns cleanly). Phase 2 must surface this before ever
// attempting the write.
func TestApply_RollsBackOnCaptureWriteFailure(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	r := newApplyTestRepo(t)

	blocked := filepath.Join(r.root, "work", "epics", "E-0002-bar")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(blocked, "epic.md")
	if err := os.WriteFile(target, []byte("pre-existing"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(target, 0o644) })

	plan := &verb.Plan{
		Subject:  "test captureWrite failure",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: "work/epics/E-0002-bar/epic.md", Content: []byte("new content")},
		},
	}
	err := verb.Apply(r.ctx, r.root, plan)
	if err == nil {
		t.Fatal("expected Apply to fail capturing the unreadable pre-existing file")
	}
	if !strings.Contains(err.Error(), "capturing pre-write state") {
		t.Errorf("error %q should mention capturing pre-write state", err.Error())
	}
	if headSHA(t, r.root) != r.preCommit {
		t.Error("HEAD must not advance")
	}
}

// TestApply_RollsBackOnMoveFailure: when the OpMove's source doesn't
// exist on disk, the filesystem rename fails; no commit and no
// leftover state.
func TestApply_RollsBackOnMoveFailure(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)
	plan := &verb.Plan{
		Subject:  "test mv-fail",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			// Source doesn't exist → the rename fails.
			{Type: verb.OpMove, Path: "does/not/exist.md", NewPath: "work/x/y.md"},
		},
	}
	err := verb.Apply(r.ctx, r.root, plan)
	if err == nil {
		t.Fatal("expected Apply to fail on missing move source")
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
	t.Parallel()
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

	newFilePath := filepath.Join("work", "milestones", "M-0001-new", "milestone.md")
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
	t.Parallel()
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
//
// This test stays SERIAL (no t.Parallel) per M-0091: it deliberately
// clears the GIT identity env vars TestMain seeded, plus
// GIT_CONFIG_GLOBAL/SYSTEM, to provoke a real commit failure.
// t.Setenv is fundamental to the test's premise; t.Parallel would
// panic, and even if it didn't, parallel tests sharing the process's
// env would see the cleared values transiently.
func TestApply_RollsBackOnCommitFailure(t *testing.T) {
	r := newApplyTestRepo(t)
	// Override author/committer with empty values so git refuses to
	// commit. (TestMain seeds these; t.Setenv here overrides just
	// for this test.)
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

// TestApply_RollbackPreservesPreExistingDirtyContent: when a touched
// path carries uncommitted worktree edits BEFORE Apply runs, a failed
// commit must leave those edits intact — not silently revert the path
// to HEAD. The old rollback ran `git restore --worktree` which
// discarded the pre-Apply bytes; this test pins the fix that captures
// pre-Apply state and restores from the capture.
//
// Closes G-0170.
func TestApply_RollbackPreservesPreExistingDirtyContent(t *testing.T) {
	r := newApplyTestRepo(t)
	// Trigger commit failure deterministically via empty git identity
	// (same shape as TestApply_RollsBackOnCommitFailure).
	t.Setenv("GIT_AUTHOR_NAME", "")
	t.Setenv("GIT_AUTHOR_EMAIL", "")
	t.Setenv("GIT_COMMITTER_NAME", "")
	t.Setenv("GIT_COMMITTER_EMAIL", "")
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	// The tracked path has HEAD content "original body" (seeded by
	// newApplyTestRepo). Modify the worktree to a pre-Apply
	// uncommitted edit — simulating the bless-mode case where the
	// operator's hand-authored prose lives only in the worktree.
	const dirty = "---\nid: E-01\n---\noriginal body\n\nOperator hand-edit — unsaved.\n"
	full := filepath.Join(r.root, r.trackedPath)
	if err := os.WriteFile(full, []byte(dirty), 0o644); err != nil {
		t.Fatal(err)
	}

	// Apply writes different content to the same path; commit will fail.
	plan := &verb.Plan{
		Subject:  "test pre-apply preservation",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: r.trackedPath, Content: []byte("verb-computed content\n")},
		},
	}
	if err := verb.Apply(r.ctx, r.root, plan); err == nil {
		t.Fatal("expected commit failure")
	}

	// The load-bearing assertion: the worktree must hold the
	// pre-Apply dirty bytes, not HEAD and not the verb's content.
	got, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("read tracked: %v", err)
	}
	if string(got) != dirty {
		t.Errorf("worktree after rollback = %q\n want %q (pre-Apply dirty bytes — rollback must not revert to HEAD)", got, dirty)
	}
	if headSHA(t, r.root) != r.preCommit {
		t.Error("HEAD must not advance on commit failure")
	}
}

// TestApply_RollbackUnaffectedByIndexLockContention generalizes the
// G-0170 regression pin for the M-0186 retrofit: rollback is now pure
// filesystem (no git call at all — see applyTx.rollback), so a held
// `.git/index.lock` cannot interfere with it even in principle. This
// test proves that directly: hold the lock throughout, fail the commit
// via a distinct, lock-independent trigger (empty git identity, the
// same technique TestApply_RollbackPreservesPreExistingDirtyContent
// uses), and confirm the pre-Apply dirty content survives — the lock
// is irrelevant to the outcome. Pre-M-0186, a held index.lock would
// have made the old rollback's own `git restore` step fail too; this
// test's predecessor pinned that recovery path. Now there is no such
// step to fail.
//
// Closes G-0170.
func TestApply_RollbackUnaffectedByIndexLockContention(t *testing.T) {
	r := newApplyTestRepo(t)
	// Trigger a genuine pre-commit failure via empty git identity —
	// gitops.CommitTree's underlying `git commit-tree` refuses these
	// exactly like `git commit` does (unlike `.git/index.lock`, which
	// CommitTree never touches at all, per TestApply_LockContentionDiagnostic).
	t.Setenv("GIT_AUTHOR_NAME", "")
	t.Setenv("GIT_AUTHOR_EMAIL", "")
	t.Setenv("GIT_COMMITTER_NAME", "")
	t.Setenv("GIT_COMMITTER_EMAIL", "")
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")

	// Synthetically hold .git/index.lock for the duration of the test —
	// the load-bearing setup: if rollback depended on git in any way,
	// this would make it fail too.
	lockPath := filepath.Join(r.root, ".git", "index.lock")
	if err := os.WriteFile(lockPath, []byte("hold"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(lockPath) })

	plan := &verb.Plan{
		Subject:  "test rollback unaffected by index lock contention",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: r.trackedPath, Content: []byte("verb-intended content\n")},
		},
	}
	if err := verb.Apply(r.ctx, r.root, plan); err == nil {
		t.Fatal("expected commit-tree failure from empty identity")
	}

	full := filepath.Join(r.root, r.trackedPath)
	got, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("read tracked: %v", err)
	}
	if strings.Contains(string(got), "verb-intended content") {
		t.Errorf("rollback did not run despite the held index.lock: worktree still holds the verb's content\n got:\n%s", got)
	}
	if !strings.Contains(string(got), "original body") {
		t.Errorf("expected rollback to restore pre-Apply content; got:\n%s", got)
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
	t.Parallel()
	t.Skip("git-add failure is defensive; not reachable from a clean unit test")
}

// TestApply_DedupesTouchedPaths: when two ops touch the same path
// (e.g. a move whose dest is also written to), the rollback must
// not pass the same path to git restore twice — that would error or
// emit duplicate warnings.
func TestApply_DedupesTouchedPaths(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestApply_G0275ToxicShapeNoLongerObstructs pins the M-0186 retrofit's
// closure of G-0275/G-0276: the exact toxic shape that used to abort
// `git stash push --staged` mid-flight — a staged rename whose old path
// is squatted by an untracked file — no longer causes ANY friction,
// because Apply never touches the live index for anything except the
// verb's own written paths. There is nothing left to stash, so there is
// nothing left to fail. This supersedes the old "fails gracefully with
// no dangling stash" regression test with a strictly stronger
// assertion: the verb's commit lands cleanly, and the user's unrelated
// staged rename survives completely untouched.
func TestApply_G0275ToxicShapeNoLongerObstructs(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)

	// Toxic shape: stage a rename of the seed file (via a plain
	// worktree rename + git add, mirroring what a user's own staged
	// rename looks like), then drop an untracked file at the old path —
	// exactly the shape that used to abort the stash push.
	moved := "work/epics/E-0001-foo/moved.md"
	full := filepath.Join(r.root, r.trackedPath)
	movedFull := filepath.Join(r.root, moved)
	if err := os.MkdirAll(filepath.Dir(movedFull), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(full, movedFull); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(r.ctx, r.root, r.trackedPath, moved); err != nil {
		t.Fatalf("git add: %v", err)
	}
	shim := filepath.Join(r.root, r.trackedPath)
	if err := os.WriteFile(shim, []byte("untracked shim\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verb writes an unrelated path.
	verbPath := "work/epics/E-0002-bar/epic.md"
	plan := &verb.Plan{
		Subject:  "test toxic shape no longer obstructs",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: verbPath, Content: []byte("---\nid: E-0002\n---\n")},
		},
	}

	if err := verb.Apply(r.ctx, r.root, plan); err != nil {
		t.Fatalf("apply: %v (toxic shape must no longer obstruct unrelated verb commits)", err)
	}

	// HEAD advanced with the verb's commit.
	if headSHA(t, r.root) == r.preCommit {
		t.Error("HEAD did not advance")
	}

	// The user's staged rename survives exactly as they left it —
	// still staged, not committed, not disturbed by the verb's commit.
	statusCmd := exec.Command("git", "status", "--porcelain", "--", r.trackedPath, moved)
	statusCmd.Dir = r.root
	statusOut, err := statusCmd.Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	got := strings.TrimSpace(string(statusOut))
	// The old path shows both the staged removal (rename source) and the
	// untracked shim; the moved path shows the staged addition.
	if !strings.Contains(got, "moved.md") {
		t.Errorf("staged rename destination missing from status: %q", got)
	}
	if !strings.Contains(got, r.trackedPath) {
		t.Errorf("staged rename source / untracked shim missing from status: %q", got)
	}

	// The untracked shim survives untouched on disk.
	shimContent, err := os.ReadFile(shim)
	if err != nil {
		t.Fatalf("reading shim: %v", err)
	}
	if string(shimContent) != "untracked shim\n" {
		t.Errorf("untracked shim content changed: %q", shimContent)
	}
}

// TestApply_RefusesNothingToCommit: git commit-tree (unlike git commit)
// has no built-in refusal for a same-tree commit, so Apply must guard
// this itself — a plan with no Ops and AllowEmpty unset (a verb bug)
// must fail loudly rather than silently create an empty commit.
func TestApply_RefusesNothingToCommit(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)
	plan := &verb.Plan{
		Subject:  "test nothing to commit",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
	}
	err := verb.Apply(r.ctx, r.root, plan)
	if err == nil {
		t.Fatal("expected Apply to refuse a plan with no Ops and AllowEmpty unset")
	}
	if !strings.Contains(err.Error(), "nothing to commit") {
		t.Errorf("error %q should mention nothing to commit", err.Error())
	}
	if headSHA(t, r.root) != r.preCommit {
		t.Error("HEAD must not advance when there is nothing to commit")
	}
}

// TestApply_StagedPathsCheckFails: outside a git repository entirely,
// the pre-flight gitops.StagedPaths call itself fails before any disk
// mutation is attempted.
func TestApply_StagedPathsCheckFails(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	plan := &verb.Plan{
		Subject:  "test staged-paths check failure",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpWrite, Path: "new.md", Content: []byte("hi")},
		},
	}
	err := verb.Apply(context.Background(), root, plan)
	if err == nil {
		t.Fatal("expected Apply to fail outside a git repository")
	}
	if !strings.Contains(err.Error(), "checking pre-staged changes") {
		t.Errorf("error %q should mention checking pre-staged changes", err.Error())
	}
}

// TestApply_RollsBackOnMoveMkdirFailure: Phase 1's os.MkdirAll for the
// move destination's parent fails when that parent's own parent
// directory denies write access.
func TestApply_RollsBackOnMoveMkdirFailure(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	r := newApplyTestRepo(t)

	noWrite := filepath.Join(r.root, "noWrite")
	if err := os.Mkdir(noWrite, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(noWrite, 0o755) })

	plan := &verb.Plan{
		Subject:  "test move-mkdir failure",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpMove, Path: r.trackedPath, NewPath: filepath.Join("noWrite", "newsub", "dest.md")},
		},
	}
	err := verb.Apply(r.ctx, r.root, plan)
	if err == nil {
		t.Fatal("expected Apply to fail creating the move destination's parent")
	}
	if headSHA(t, r.root) != r.preCommit {
		t.Error("HEAD must not advance on move-mkdir failure")
	}
	if got := porcelain(t, r.root); got != "" {
		t.Errorf("tree must stay clean on move-mkdir failure: %q", got)
	}
	// Original file must survive at its original path.
	got := readFile(t, filepath.Join(r.root, r.trackedPath))
	if !bytes.Contains(got, []byte("original body")) {
		t.Errorf("original file content lost: %q", got)
	}
}

// TestApply_DirectoryMoveWithNestedFile_CommitTreeIsCorrect pins the
// commit CONTENT for a directory move containing a nested file (the
// archive/reallocate shape: an epic dir carrying a milestone inside
// it) — not just the worktree, which Phase 1's os.Rename already gets
// right regardless of how gatherCommitOps computes the commit's
// removes/writes sets. Asserts via `git ls-tree -r` directly: the
// nested file's OLD path must be absent from the landed commit's tree
// and its NEW path must carry the original content — the worktree-only
// os.Stat checks the other directory-move tests use cannot distinguish
// "moved correctly" from "duplicated in the commit, moved on disk."
func TestApply_DirectoryMoveWithNestedFile_CommitTreeIsCorrect(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)

	srcDir := filepath.Join(r.root, "work", "epics", "E-9999-src")
	nestedRel := filepath.Join("work", "epics", "E-9999-src", "M-0001-nested.md")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.root, nestedRel), []byte("nested milestone\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", r.root, "add", nestedRel).Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", r.root, "commit", "-m", "seed nested file").Run(); err != nil {
		t.Fatal(err)
	}

	plan := &verb.Plan{
		Subject:  "test directory move with nested file",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpMove, Path: "work/epics/E-9999-src", NewPath: "work/epics/E-9998-dst"},
		},
	}
	if err := verb.Apply(r.ctx, r.root, plan); err != nil {
		t.Fatalf("apply: %v", err)
	}

	out, err := exec.Command("git", "-C", r.root, "ls-tree", "-r", "--name-only", "HEAD").Output()
	if err != nil {
		t.Fatalf("ls-tree: %v", err)
	}
	entries := strings.Fields(string(out))
	oldPath := "work/epics/E-9999-src/M-0001-nested.md"
	newPath := "work/epics/E-9998-dst/M-0001-nested.md"
	if slices.Contains(entries, oldPath) {
		t.Errorf("commit tree still contains the nested file's OLD path %q: %v", oldPath, entries)
	}
	if !slices.Contains(entries, newPath) {
		t.Errorf("commit tree missing the nested file's NEW path %q: %v", newPath, entries)
	}

	content, err := exec.Command("git", "-C", r.root, "show", "HEAD:"+newPath).Output()
	if err != nil {
		t.Fatalf("show HEAD:%s: %v", newPath, err)
	}
	if string(content) != "nested milestone\n" {
		t.Errorf("nested file content = %q, want %q", content, "nested milestone\n")
	}
}

// TestApply_RollsBackOnDirectoryMoveThenNestedRewrite_BothSymptoms pins
// D-0029: a plan that moves a directory AND rewrites a file nested
// inside it, then fails on a later step, must fully reverse BOTH
// mutations — not just the move. This is the real op shape `reallocate`
// and `rewidth` use on epic entities (move the epic dir, then rewrite
// the epic.md or a nested milestone inside it), so a Phase-2 failure or
// a commit-tree failure after a successful directory move is a real,
// reachable scenario, not a contrived one.
//
// Two independent corruption symptoms must both be checked: (a) the
// nested file's content must be restored at its ORIGINAL (moved-back)
// location, and (b) the vacated new-location path must not exist at
// all — a test that checks only one of these would pass even if the
// other half of the corruption were still present.
func TestApply_RollsBackOnDirectoryMoveThenNestedRewrite_BothSymptoms(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	r := newApplyTestRepo(t)

	srcDir := filepath.Join(r.root, "work", "epics", "E-9999-src")
	nestedRel := filepath.Join("work", "epics", "E-9999-src", "M-0001-nested.md")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte("original nested content\n")
	if err := os.WriteFile(filepath.Join(r.root, nestedRel), original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", r.root, "add", nestedRel).Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", r.root, "commit", "-m", "seed nested file").Run(); err != nil {
		t.Fatal(err)
	}
	preCommit := headSHA(t, r.root)

	// A directory blocking a later, unrelated write forces Apply to
	// fail AFTER the move and the nested rewrite have both already
	// landed on disk — exactly the ordering that exposes the bug.
	noWrite := filepath.Join(r.root, "noWrite")
	if err := os.Mkdir(noWrite, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(noWrite, 0o755) })

	plan := &verb.Plan{
		Subject:  "test directory move + nested rewrite rollback",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpMove, Path: "work/epics/E-9999-src", NewPath: "work/epics/E-9998-dst"},
			{Type: verb.OpWrite, Path: "work/epics/E-9998-dst/M-0001-nested.md", Content: []byte("rewritten content\n")},
			{Type: verb.OpWrite, Path: filepath.Join("noWrite", "child", "blocked.md"), Content: []byte("nope")},
		},
	}
	err := verb.Apply(r.ctx, r.root, plan)
	if err == nil {
		t.Fatal("expected Apply to fail on the unwritable third op")
	}
	if headSHA(t, r.root) != preCommit {
		t.Error("HEAD must not advance")
	}

	// Symptom (a): the original location must hold the ORIGINAL content
	// — not the rewritten content the move+write sequence produced.
	restored, readErr := os.ReadFile(filepath.Join(r.root, nestedRel))
	if readErr != nil {
		t.Fatalf("reading restored nested file: %v", readErr)
	}
	if !bytes.Equal(restored, original) {
		t.Errorf("nested file at original location = %q, want %q (rollback must undo the rewrite, not just the move)", restored, original)
	}

	// Symptom (b): the vacated new-location directory must not exist —
	// no stray duplicate left behind.
	if _, statErr := os.Stat(filepath.Join(r.root, "work", "epics", "E-9998-dst")); !os.IsNotExist(statErr) {
		t.Errorf("stray directory left at the vacated new location (stat err: %v)", statErr)
	}

	if got := porcelain(t, r.root); got != "" {
		t.Errorf("dirty tree after rollback: %q", got)
	}
}

// TestApply_RollsBackOnDirectoryMoveMkdirFailure: Phase 1's directory-
// move branch fails creating the destination's parent when that
// parent's own parent denies write access.
func TestApply_RollsBackOnDirectoryMoveMkdirFailure(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	r := newApplyTestRepo(t)

	srcDir := filepath.Join(r.root, "work", "epics", "E-9999-src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	noWrite := filepath.Join(r.root, "noWrite")
	if err := os.Mkdir(noWrite, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(noWrite, 0o755) })

	plan := &verb.Plan{
		Subject:  "test directory-move mkdir failure",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpMove, Path: "work/epics/E-9999-src", NewPath: filepath.Join("noWrite", "newsub", "E-9999-dst")},
		},
	}
	err := verb.Apply(r.ctx, r.root, plan)
	if err == nil {
		t.Fatal("expected Apply to fail creating the directory move's destination parent")
	}
	if headSHA(t, r.root) != r.preCommit {
		t.Error("HEAD must not advance on directory-move mkdir failure")
	}
	if _, statErr := os.Stat(srcDir); statErr != nil {
		t.Errorf("source directory should be untouched: %v", statErr)
	}
}

// TestApply_RollsBackOnDirectoryMoveRenameFailure: Phase 1's directory
// move fails when the destination already exists as a non-empty
// directory — os.Rename refuses to replace it (ENOTEMPTY).
func TestApply_RollsBackOnDirectoryMoveRenameFailure(t *testing.T) {
	t.Parallel()
	r := newApplyTestRepo(t)

	srcDir := filepath.Join(r.root, "work", "epics", "E-9999-src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	destDir := filepath.Join(r.root, "work", "epics", "E-9998-dst")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "occupied.md"), []byte("occupied"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan := &verb.Plan{
		Subject:  "test directory-move rename failure",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpMove, Path: "work/epics/E-9999-src", NewPath: "work/epics/E-9998-dst"},
		},
	}
	err := verb.Apply(r.ctx, r.root, plan)
	if err == nil {
		t.Fatal("expected Apply to fail moving onto a non-empty destination directory")
	}
	if !strings.Contains(err.Error(), "moving") {
		t.Errorf("error %q should mention moving", err.Error())
	}
	if headSHA(t, r.root) != r.preCommit {
		t.Error("HEAD must not advance on directory-move rename failure")
	}
	if _, statErr := os.Stat(srcDir); statErr != nil {
		t.Errorf("source directory should be untouched: %v", statErr)
	}
}

// TestApply_RollsBackOnGatherCommitOpsFailure drives a real end-to-end
// Apply() failure in gatherCommitOps: an OpMove of a directory whose
// destination (after the real os.Rename) contains a permission-denied
// subdirectory fails the post-move recursive walk. Phase 1 itself
// succeeds (the rename doesn't need to read the subdirectory's
// contents); the failure surfaces only when gatherCommitOps walks the
// moved tree to build the commit's write set.
func TestApply_RollsBackOnGatherCommitOpsFailure(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	r := newApplyTestRepo(t)

	srcDir := filepath.Join(r.root, "work", "epics", "E-9999-blocked")
	blockedDir := filepath.Join(srcDir, "blocked")
	if err := os.MkdirAll(blockedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blockedDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blockedDir, 0o755) })

	plan := &verb.Plan{
		Subject:  "test gatherCommitOps walk failure",
		Trailers: []gitops.Trailer{{Key: "aiwf-verb", Value: "test"}},
		Ops: []verb.FileOp{
			{Type: verb.OpMove, Path: "work/epics/E-9999-blocked", NewPath: "work/epics/E-9998-blocked-moved"},
		},
	}
	err := verb.Apply(r.ctx, r.root, plan)
	if err == nil {
		t.Fatal("expected Apply to fail walking the moved directory")
	}
	if !strings.Contains(err.Error(), "walking") {
		t.Errorf("error %q should mention walking", err.Error())
	}
	if headSHA(t, r.root) != r.preCommit {
		t.Error("HEAD must not advance on gatherCommitOps failure")
	}
	// Rollback restores the moved directory to its original location —
	// the move itself (Phase 1) succeeded and must be undone.
	if _, statErr := os.Stat(filepath.Join(r.root, "work", "epics", "E-9999-blocked")); statErr != nil {
		t.Errorf("original directory not restored after rollback: %v", statErr)
	}
}
