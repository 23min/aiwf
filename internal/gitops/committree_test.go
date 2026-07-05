package gitops

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestCommitTree_DoesNotTouchLiveIndexOrWorktree pins M-0186/AC-1: the
// primitive builds its commit against a throwaway index and never reads
// or writes the live index file or the worktree. The live index file's
// raw bytes must be byte-for-byte identical before and after the call
// (proving CommitTree never opened it for writing), an unrelated
// unstaged worktree edit must survive untouched, and the new commit's
// content must never be materialized into the worktree — it lives only
// in the object database until something else reconciles it (that's
// AC-2, not this primitive). Deliberately does NOT assert `git diff
// --cached` is unchanged: moving HEAD forward changes what diff --cached
// reports for a path new-in-HEAD-but-absent-from-the-live-index — that
// side effect is exactly the gap AC-2's reconciliation step closes, not
// a violation of this primitive's contract.
func TestCommitTree_DoesNotTouchLiveIndexOrWorktree(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()

	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "base.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "base.md"); err != nil {
		t.Fatalf("add base.md: %v", err)
	}
	if err := Commit(ctx, root, "initial commit", "", nil); err != nil {
		t.Fatalf("initial commit: %v", err)
	}
	headBefore, err := output(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}

	// Unrelated staged change.
	err = os.WriteFile(filepath.Join(root, "staged.md"), []byte("staged\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	err = Add(ctx, root, "staged.md")
	if err != nil {
		t.Fatalf("add staged.md: %v", err)
	}

	gitDir, err := GitDir(ctx, root)
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	indexPath := filepath.Join(gitDir, "index")
	indexBefore, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("reading live index before: %v", err)
	}

	// Unrelated unstaged worktree edit.
	err = os.WriteFile(filepath.Join(root, "base.md"), []byte("base\nedited\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	worktreeBefore, err := os.ReadFile(filepath.Join(root, "base.md"))
	if err != nil {
		t.Fatal(err)
	}

	sha, err := CommitTree(ctx, root, nil, []PathWrite{
		{Path: "new.md", Content: []byte("new content\n")},
	}, "verb commit via temp index", "", []Trailer{
		{Key: "aiwf-verb", Value: "add"},
	})
	if err != nil {
		t.Fatalf("CommitTree: %v", err)
	}
	if sha == "" {
		t.Fatal("CommitTree returned empty SHA")
	}

	// HEAD advanced to the new commit, with the expected subject/trailers.
	subj, err := HeadSubject(ctx, root)
	if err != nil {
		t.Fatalf("HeadSubject: %v", err)
	}
	if subj != "verb commit via temp index" {
		t.Errorf("HEAD subject = %q, want %q", subj, "verb commit via temp index")
	}
	trailers, err := HeadTrailers(ctx, root)
	if err != nil {
		t.Fatalf("HeadTrailers: %v", err)
	}
	wantTrailers := []Trailer{{Key: "aiwf-verb", Value: "add"}}
	if diff := cmp.Diff(wantTrailers, trailers); diff != "" {
		t.Errorf("trailers mismatch (-want +got):\n%s", diff)
	}
	headAfter, err := output(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD (after): %v", err)
	}
	if headAfter == headBefore {
		t.Fatal("HEAD did not advance")
	}
	if got := strings.TrimSpace(headAfter); got != sha {
		t.Errorf("HEAD = %q, want returned SHA %q", got, sha)
	}

	// HEAD must stay attached to its branch, not detach into a bare SHA —
	// `update-ref HEAD` derefs symbolic refs by default, but a future
	// change (e.g. adding --no-deref) would silently detach HEAD here.
	branch, err := output(ctx, root, "symbolic-ref", "HEAD")
	if err != nil {
		t.Fatalf("symbolic-ref HEAD: %v", err)
	}
	if strings.TrimSpace(branch) != "refs/heads/main" {
		t.Errorf("HEAD detached; symbolic-ref HEAD = %q, want refs/heads/main", strings.TrimSpace(branch))
	}

	// The live index file itself was never opened for writing.
	indexAfter, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("reading live index after: %v", err)
	}
	if !bytes.Equal(indexBefore, indexAfter) {
		t.Error("live index file changed; CommitTree must build against a throwaway index only")
	}

	// The user's pre-existing unstaged worktree edit is byte-for-byte unchanged.
	worktreeAfter, err := os.ReadFile(filepath.Join(root, "base.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(worktreeBefore, worktreeAfter) {
		t.Errorf("worktree file changed:\nbefore: %q\nafter:  %q", worktreeBefore, worktreeAfter)
	}

	// The new commit's content is never materialized into the worktree —
	// it lives only in the object database until something reconciles it.
	if _, statErr := os.Stat(filepath.Join(root, "new.md")); !os.IsNotExist(statErr) {
		t.Errorf("new.md was materialized into the worktree; CommitTree must not touch it (stat err: %v)", statErr)
	}

	// The new commit's full tree contains both new.md (the write really
	// landed in the object database, just not in the live index/worktree)
	// AND base.md (read-tree correctly seeded the parent's existing
	// content — a commit that dropped everything except the new write
	// would still pass a diff-only check against its parent, so this
	// must walk the full tree, not `git show --name-only`).
	treeOut, err := output(ctx, root, "ls-tree", "-r", "--name-only", sha)
	if err != nil {
		t.Fatalf("ls-tree %s: %v", sha, err)
	}
	entries := strings.Fields(treeOut)
	if !slices.Contains(entries, "new.md") {
		t.Errorf("commit %s tree does not contain new.md; content did not land: %q", sha, treeOut)
	}
	if !slices.Contains(entries, "base.md") {
		t.Errorf("commit %s tree does not contain base.md; read-tree did not seed the parent's content: %q", sha, treeOut)
	}

	// new.md landed as a regular, non-executable file — every current
	// PathWrite caller (verb.Apply's OpWrite, via pathutil.AtomicWriteFile)
	// writes plain 0o644 content; a wrong mode here would silently ship
	// every future verb-written file as executable.
	modeOut, err := output(ctx, root, "ls-tree", sha, "--", "new.md")
	if err != nil {
		t.Fatalf("ls-tree (mode) %s: %v", sha, err)
	}
	if !strings.HasPrefix(modeOut, "100644 ") {
		t.Errorf("new.md mode = %q, want 100644 blob entry", modeOut)
	}
}

// TestCommitTree_HEADResolutionFails_NotARepo exercises CommitTree's own
// error branch: workdir isn't a git repository at all, so its rev-parse
// HEAD fails before commitTreeFromParent (and its own GitDir call) is
// ever reached.
func TestCommitTree_HEADResolutionFails_NotARepo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()

	_, err := CommitTree(ctx, root, nil, []PathWrite{{Path: "a.md", Content: []byte("a\n")}}, "subject", "", nil)
	if err == nil {
		t.Fatal("want error in a non-repo directory, got nil")
	}
	if !strings.Contains(err.Error(), "resolving HEAD") {
		t.Errorf("error %q should mention resolving HEAD", err.Error())
	}
}

// TestCommitTreeFromParent_GitDirFails_NotARepo exercises
// commitTreeFromParent's own GitDir branch directly. CommitTree can never
// reach it in production (its rev-parse HEAD would already have failed
// on a non-repo workdir, per the test above) — this drives the unexported
// helper directly, the friend-assembly technique for a branch a public
// caller can't reach but a direct call still can.
func TestCommitTreeFromParent_GitDirFails_NotARepo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()

	_, err := commitTreeFromParent(ctx, root, "0000000000000000000000000000000000000000", nil, nil, "subject", "", nil)
	if err == nil {
		t.Fatal("want error in a non-repo directory, got nil")
	}
	if !strings.Contains(err.Error(), "resolving git dir") {
		t.Errorf("error %q should mention resolving git dir", err.Error())
	}
}

// TestCommitTree_CreatesRootCommitWhenNoHEAD exercises the case where
// the repo is real but has no commits yet (unborn HEAD): CommitTree
// must build a root commit — the same as `git commit` does on a fresh
// repository — rather than erroring, since verb.Apply's very first
// commit against a brand-new consumer repo hits exactly this path.
func TestCommitTree_CreatesRootCommitWhenNoHEAD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}

	sha, err := CommitTree(ctx, root, nil, []PathWrite{{Path: "a.md", Content: []byte("a\n")}}, "root commit", "", nil)
	if err != nil {
		t.Fatalf("CommitTree: %v", err)
	}

	parents, err := output(ctx, root, "log", "-1", "--pretty=%P", sha)
	if err != nil {
		t.Fatalf("log -1 --pretty=%%P: %v", err)
	}
	if strings.TrimSpace(parents) != "" {
		t.Errorf("root commit has parents %q, want none", parents)
	}

	entries, err := output(ctx, root, "ls-tree", "-r", "--name-only", sha)
	if err != nil {
		t.Fatalf("ls-tree %s: %v", sha, err)
	}
	if !slices.Contains(strings.Fields(entries), "a.md") {
		t.Errorf("root commit tree missing a.md: %q", entries)
	}

	headSHA, err := output(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	if strings.TrimSpace(headSHA) != sha {
		t.Errorf("HEAD = %q, want %q", strings.TrimSpace(headSHA), sha)
	}
}

// TestCommitTree_MkdirTempFails_GitDirReadOnly makes the repo's .git dir
// read-only so creating the temp index directory fails. Mirrors the
// chmod-based fault injection in internal/verb/apply_internal_test.go.
func TestCommitTree_MkdirTempFails_GitDirReadOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)

	gitDir, err := GitDir(ctx, root)
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	err = os.Chmod(gitDir, 0o500)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(gitDir, 0o755) })

	_, err = CommitTree(ctx, root, nil, []PathWrite{{Path: "a.md", Content: []byte("a\n")}}, "subject", "", nil)
	if err == nil {
		t.Fatal("want error with a read-only git dir, got nil")
	}
	if !strings.Contains(err.Error(), "creating temp index dir") {
		t.Errorf("error %q should mention creating temp index dir", err.Error())
	}
}

// TestCommitTree_ReadTreeFails_CorruptedTreeObject deletes HEAD's tree
// object from the object database before calling CommitTree, simulating
// local repository corruption (an incomplete transfer, a truncated
// object). `git read-tree HEAD` then fails because the object it needs
// to seed the temp index doesn't exist.
func TestCommitTree_ReadTreeFails_CorruptedTreeObject(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)

	treeSHA, err := output(ctx, root, "rev-parse", "HEAD^{tree}")
	if err != nil {
		t.Fatalf("rev-parse HEAD^{tree}: %v", err)
	}
	treeSHA = strings.TrimSpace(treeSHA)
	gitDir, err := GitDir(ctx, root)
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	objectPath := filepath.Join(gitDir, "objects", treeSHA[:2], treeSHA[2:])
	err = os.Remove(objectPath)
	if err != nil {
		t.Fatalf("removing tree object %s: %v", objectPath, err)
	}

	_, err = CommitTree(ctx, root, nil, []PathWrite{{Path: "a.md", Content: []byte("a\n")}}, "subject", "", nil)
	if err == nil {
		t.Fatal("want error with a missing tree object, got nil")
	}
	if !strings.Contains(err.Error(), "read-tree") {
		t.Errorf("error %q should mention read-tree", err.Error())
	}
}

// TestCommitTree_HashObjectFails_ObjectsDirReadOnly makes the object
// database read-only so hash-object can't write the new blob. read-tree
// still succeeds (it only reads); the write is what fails.
func TestCommitTree_HashObjectFails_ObjectsDirReadOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)

	gitDir, err := GitDir(ctx, root)
	if err != nil {
		t.Fatalf("GitDir: %v", err)
	}
	objectsDir := filepath.Join(gitDir, "objects")
	err = os.Chmod(objectsDir, 0o500)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(objectsDir, 0o755) })

	_, err = CommitTree(ctx, root, nil, []PathWrite{{Path: "a.md", Content: []byte("a\n")}}, "subject", "", nil)
	if err == nil {
		t.Fatal("want error with a read-only objects dir, got nil")
	}
	if !strings.Contains(err.Error(), "hashing blob") {
		t.Errorf("error %q should mention hashing blob", err.Error())
	}
}

// TestCommitTreeFromParent_RefusesStaleParent_ConcurrentHEADMove pins the
// safety property CommitTree's doc comment claims: update-ref's
// compare-and-swap detects a HEAD that moved since the parent was
// captured, rather than silently overwriting it. Reproducing the actual
// race through CommitTree's public entry point would require timing two
// goroutines around its internal git subprocess calls — inherently
// flaky. commitTreeFromParent's parent parameter is the seam: capture a
// real stale parent, let a concurrent commit land for real, then drive
// the exact same construction-and-update-ref code CommitTree uses
// against that stale parent, deterministically reproducing the race's
// end state without timing games.
func TestCommitTreeFromParent_RefusesStaleParent_ConcurrentHEADMove(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)

	staleParent, err := output(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	staleParent = strings.TrimSpace(staleParent)

	// A concurrent commit lands for real, moving HEAD past staleParent.
	err = os.WriteFile(filepath.Join(root, "concurrent.md"), []byte("concurrent\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	err = Add(ctx, root, "concurrent.md")
	if err != nil {
		t.Fatalf("add concurrent.md: %v", err)
	}
	err = Commit(ctx, root, "concurrent commit", "", nil)
	if err != nil {
		t.Fatalf("concurrent commit: %v", err)
	}

	_, err = commitTreeFromParent(ctx, root, staleParent, nil, []PathWrite{
		{Path: "should-not-land.md", Content: []byte("nope\n")},
	}, "should be refused", "", nil)
	if err == nil {
		t.Fatal("want error when parent is stale, got nil")
	}
	if !strings.Contains(err.Error(), "update-ref") {
		t.Errorf("error %q should mention update-ref", err.Error())
	}

	// The concurrent commit's content is what HEAD still has — the
	// refused attempt did not clobber it.
	subj, err := HeadSubject(ctx, root)
	if err != nil {
		t.Fatalf("HeadSubject: %v", err)
	}
	if subj != "concurrent commit" {
		t.Errorf("HEAD subject = %q, want %q (refused commit must not have landed)", subj, "concurrent commit")
	}
}

// seedRepo initializes a repo at a fresh t.TempDir with one commit, and
// returns its root. Shared setup for the CommitTree failure-path tests.
func seedRepo(t *testing.T, ctx context.Context) string {
	t.Helper()
	root := t.TempDir()
	if err := Init(ctx, root); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "base.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Add(ctx, root, "base.md"); err != nil {
		t.Fatalf("add base.md: %v", err)
	}
	if err := Commit(ctx, root, "initial commit", "", nil); err != nil {
		t.Fatalf("initial commit: %v", err)
	}
	return root
}

// TestOutputIndexed_ErrorIncludesStderr pins outputIndexed's own
// error-wrapping branch directly. In production it's only reachable via
// `write-tree` failing (coverage:ignore'd at that call site as requiring
// object-database corruption); a deliberately invalid subcommand
// exercises the same generic wrap without needing that corruption.
func TestOutputIndexed_ErrorIncludesStderr(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)
	indexPath := filepath.Join(t.TempDir(), "index")

	_, err := outputIndexed(ctx, root, indexPath, "not-a-real-git-command")
	if err == nil {
		t.Fatal("want error for an invalid git subcommand, got nil")
	}
	if !strings.Contains(err.Error(), "git not-a-real-git-command") {
		t.Errorf("error %q should mention the failing command", err.Error())
	}
}

// TestCommitTree_OverwritesExistingTrackedFile pins the primary
// real-world case: most aiwf verbs (promote, edit-body, cancel) rewrite
// an EXISTING entity file, not add a new one. update-index --add
// --cacheinfo must replace the existing index entry rather than
// duplicate it or leave the stale blob behind.
func TestCommitTree_OverwritesExistingTrackedFile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx) // base.md = "base\n", tracked at HEAD

	sha, err := CommitTree(ctx, root, nil, []PathWrite{
		{Path: "base.md", Content: []byte("overwritten\n")},
	}, "overwrite base.md", "", []Trailer{{Key: "aiwf-verb", Value: "edit-body"}})
	if err != nil {
		t.Fatalf("CommitTree: %v", err)
	}

	content, err := output(ctx, root, "show", sha+":base.md")
	if err != nil {
		t.Fatalf("show %s:base.md: %v", sha, err)
	}
	if content != "overwritten\n" {
		t.Errorf("base.md content = %q, want %q", content, "overwritten\n")
	}

	entries, err := output(ctx, root, "ls-tree", "-r", "--name-only", sha)
	if err != nil {
		t.Fatalf("ls-tree %s: %v", sha, err)
	}
	if got := strings.Count(entries, "base.md"); got != 1 {
		t.Errorf("base.md appears %d times in the tree, want exactly 1: %q", got, entries)
	}
}

// TestCommitTree_HandlesRename pins M-0186/AC-3's rename requirement:
// read-tree seeds the temp index with the OLD path from the parent
// tree, so a rename must explicitly evict it via `removes` — otherwise
// the resulting tree would carry both the old and new paths. A rename
// is a delete-old + add-new against the same commit tree, in one atomic
// commit-tree call.
func TestCommitTree_HandlesRename(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx) // base.md tracked at HEAD

	sha, err := CommitTree(ctx, root, []string{"base.md"}, []PathWrite{
		{Path: "renamed.md", Content: []byte("base\n")},
	}, "rename base.md", "", nil)
	if err != nil {
		t.Fatalf("CommitTree: %v", err)
	}

	entries, err := output(ctx, root, "ls-tree", "-r", "--name-only", sha)
	if err != nil {
		t.Fatalf("ls-tree %s: %v", sha, err)
	}
	fields := strings.Fields(entries)
	if slices.Contains(fields, "base.md") {
		t.Errorf("base.md still present in renamed commit's tree: %q", entries)
	}
	if !slices.Contains(fields, "renamed.md") {
		t.Errorf("renamed.md missing from commit's tree: %q", entries)
	}

	// The live index and worktree are untouched by the rename either —
	// same AC-1 guarantee, now also proven for the remove side.
	if _, statErr := os.Stat(filepath.Join(root, "renamed.md")); !os.IsNotExist(statErr) {
		t.Errorf("renamed.md materialized into the worktree (stat err: %v)", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(root, "base.md")); statErr != nil {
		t.Errorf("base.md removed from the worktree; CommitTree must not touch it (stat err: %v)", statErr)
	}
}

// TestCommitTree_RemoveOfAbsentPathIsNoOp pins the underlying git
// primitive's own behavior for a defensive edge: removing a path that
// never existed in the parent tree does not error — the tree is built
// as if that remove were never requested.
func TestCommitTree_RemoveOfAbsentPathIsNoOp(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)

	sha, err := CommitTree(ctx, root, []string{"never-existed.md"}, []PathWrite{
		{Path: "new.md", Content: []byte("new\n")},
	}, "remove of absent path is a no-op", "", nil)
	if err != nil {
		t.Fatalf("CommitTree: %v", err)
	}

	entries, err := output(ctx, root, "ls-tree", "-r", "--name-only", sha)
	if err != nil {
		t.Fatalf("ls-tree %s: %v", sha, err)
	}
	fields := strings.Fields(entries)
	if !slices.Contains(fields, "base.md") {
		t.Errorf("base.md missing (unrelated no-op remove should not disturb read-tree's seed): %q", entries)
	}
	if !slices.Contains(fields, "new.md") {
		t.Errorf("new.md missing from commit's tree: %q", entries)
	}
}

// TestCommitTree_WritesNewNestedPath pins the other real-world write
// shape: a brand-new path under directories that don't exist in the
// parent tree yet (e.g. `aiwf add` creating a new entity file). Neither
// read-tree nor update-index needs an explicit mkdir step — the index
// and tree model paths are flat strings — but that's worth pinning
// directly rather than assumed from the flat-path happy-path test.
func TestCommitTree_WritesNewNestedPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := seedRepo(t, ctx)

	sha, err := CommitTree(ctx, root, nil, []PathWrite{
		{Path: "work/gaps/G-0999-example.md", Content: []byte("nested\n")},
	}, "add nested entity", "", nil)
	if err != nil {
		t.Fatalf("CommitTree: %v", err)
	}

	entries, err := output(ctx, root, "ls-tree", "-r", "--name-only", sha)
	if err != nil {
		t.Fatalf("ls-tree %s: %v", sha, err)
	}
	if !slices.Contains(strings.Fields(entries), "work/gaps/G-0999-example.md") {
		t.Errorf("nested path missing from tree: %q", entries)
	}

	content, err := output(ctx, root, "show", sha+":work/gaps/G-0999-example.md")
	if err != nil {
		t.Fatalf("show %s:work/gaps/G-0999-example.md: %v", sha, err)
	}
	if content != "nested\n" {
		t.Errorf("nested file content = %q, want %q", content, "nested\n")
	}
}
