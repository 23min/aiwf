package trunk

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

func TestRead_NoRemotes_Skips(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-001-foo.md", "# foo\n")

	res, err := Read(ctx, dir, nil)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !res.Skipped {
		t.Error("Skipped = false, want true (no remotes configured → no tracking refs → skip)")
	}
	if len(res.IDs) != 0 {
		t.Errorf("IDs = %v, want empty when skipped", res.IDs)
	}
}

func TestRead_RemoteAddedButNeverFetched_Skips(t *testing.T) {
	t.Parallel()
	// `git remote add` without `git fetch` leaves no refs/remotes/*
	// tracking refs. There's nothing on this remote we know about
	// yet, so trunk-awareness has nothing to do. This also covers
	// the "freshly cloned an empty bare" case at the moment of
	// first-push, where the bare has no branches and the clone has
	// no tracking refs.
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "README.md", "readme\n")
	mustRun(t, ctx, dir, "remote", "add", "origin", "https://example.invalid/x.git")

	res, err := Read(ctx, dir, nil)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !res.Skipped {
		t.Error("Skipped = false, want true (remote configured but no tracking refs)")
	}
}

func TestRead_RemoteAndDefaultTrunk_ReturnsIDs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-001-foo.md", "# foo\n")
	commitFile(t, ctx, dir, "docs/adr/ADR-0001-baz.md", "# baz\n")
	commitFile(t, ctx, dir, "README.md", "readme\n")
	mustRun(t, ctx, dir, "remote", "add", "origin", "https://example.invalid/x.git")
	// Mirror HEAD as the default trunk ref so Read finds it.
	mustRun(t, ctx, dir, "update-ref", config.DefaultAllocateTrunk, "HEAD")

	res, err := Read(ctx, dir, nil)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if res.Skipped {
		t.Error("Skipped = true, want false")
	}
	// trunk.Read extracts ids from the on-disk filename verbatim;
	// width canonicalization happens at the consumer layer (allocator,
	// ids-unique check via tree.ByID). The narrow id is intentional
	// — this is parser-tolerance test data per AC-2 in M-081.
	want := []ID{
		{Kind: entity.KindADR, ID: "ADR-0001", Path: "docs/adr/ADR-0001-baz.md"},
		{Kind: entity.KindGap, ID: "G-001", Path: "work/gaps/G-001-foo.md"},
	}
	if diff := cmp.Diff(want, res.IDs); diff != "" {
		t.Errorf("IDs mismatch (-want +got):\n%s", diff)
	}
}

func TestRead_TrackingRefsExistButTrunkMissing_HardError(t *testing.T) {
	t.Parallel()
	// The repo has fetched at least one branch from origin (so
	// refs/remotes/origin/* is populated) but the configured trunk
	// is not one of them. That is real misconfiguration: the user
	// either named the wrong branch in allocate.trunk or hasn't
	// fetched the right one. We must surface the error so they fix it.
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "README.md", "readme\n")
	mustRun(t, ctx, dir, "remote", "add", "origin", "https://example.invalid/x.git")
	// Simulate having fetched origin/develop so a tracking ref
	// exists. The configured trunk (default refs/remotes/origin/main)
	// still doesn't resolve.
	mustRun(t, ctx, dir, "update-ref", "refs/remotes/origin/develop", "HEAD")

	_, err := Read(ctx, dir, nil)
	if err == nil {
		t.Fatal("Read: expected error for missing default trunk with tracking refs present, got nil")
	}
	if !strings.Contains(err.Error(), config.DefaultAllocateTrunk) {
		t.Errorf("error %q should mention the missing ref %q", err, config.DefaultAllocateTrunk)
	}
	if !strings.Contains(err.Error(), "allocate.trunk") {
		t.Errorf("error %q should hint at allocate.trunk config", err)
	}
}

func TestRead_ExplicitTrunk_UsedInsteadOfDefault(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-007-explicit.md", "# explicit\n")
	mustRun(t, ctx, dir, "remote", "add", "origin", "https://example.invalid/x.git")
	mustRun(t, ctx, dir, "update-ref", "refs/remotes/origin/develop", "HEAD")

	cfg := &config.Config{Allocate: config.Allocate{Trunk: "refs/remotes/origin/develop"}}
	res, err := Read(ctx, dir, cfg)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if res.Skipped {
		t.Error("Skipped = true, want false")
	}
	want := []ID{{Kind: entity.KindGap, ID: "G-007", Path: "work/gaps/G-007-explicit.md"}}
	if diff := cmp.Diff(want, res.IDs); diff != "" {
		t.Errorf("IDs mismatch (-want +got):\n%s", diff)
	}
}

func TestRead_ExplicitTrunkMissing_HardError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "README.md", "readme\n")
	mustRun(t, ctx, dir, "remote", "add", "origin", "https://example.invalid/x.git")

	cfg := &config.Config{Allocate: config.Allocate{Trunk: "refs/remotes/origin/typo"}}
	_, err := Read(ctx, dir, cfg)
	if err == nil {
		t.Fatal("Read: expected error for missing explicit trunk, got nil")
	}
	if !strings.Contains(err.Error(), "refs/remotes/origin/typo") {
		t.Errorf("error %q should mention the missing ref", err)
	}
}

func TestResult_IDStrings(t *testing.T) {
	t.Parallel()
	r := Result{IDs: []ID{
		{Kind: entity.KindGap, ID: "G-0001", Path: "work/gaps/G-001-foo.md"},
		{Kind: entity.KindADR, ID: "ADR-0001", Path: "docs/adr/ADR-0001-baz.md"},
	}}
	got := r.IDStrings()
	want := []string{"G-0001", "ADR-0001"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("IDStrings mismatch (-want +got):\n%s", diff)
	}

	if (Result{}).IDStrings() != nil {
		t.Error("empty Result.IDStrings should be nil")
	}
}

// --- M-0212: LocalRefIDs (the allocator's broadened cross-branch view) ---

func TestLocalRefIDs_UnionsSiblingBranchIDs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	// main carries G-0001.
	commitFile(t, ctx, dir, "work/gaps/G-0001-foo.md", "# foo\n")
	// A sibling branch carries a higher id that exists ONLY on that
	// ref — the dominant solo+worktree collision class. No remote, so
	// no trunk ref is in play; the sibling id is visible solely via
	// refs/heads/*.
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "sibling")
	commitFile(t, ctx, dir, "work/gaps/G-0005-bar.md", "# bar\n")
	mustRun(t, ctx, dir, "checkout", "-q", "main")

	got := LocalRefIDs(ctx, dir)
	if !slices.Contains(got, "G-0005") {
		t.Errorf("LocalRefIDs = %v, want it to include sibling-only id G-0005", got)
	}
	if !slices.Contains(got, "G-0001") {
		t.Errorf("LocalRefIDs = %v, want it to include main id G-0001", got)
	}
}

func TestLocalRefIDs_ScansAcrossEntityKinds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-0001-foo.md", "# foo\n")
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "sibling")
	commitFile(t, ctx, dir, "docs/adr/ADR-0009-baz.md", "# baz\n")
	mustRun(t, ctx, dir, "checkout", "-q", "main")

	got := LocalRefIDs(ctx, dir)
	if !slices.Contains(got, "ADR-0009") {
		t.Errorf("LocalRefIDs = %v, want it to include sibling-branch ADR-0009", got)
	}
}

func TestIDsFromPaths_SkipsNonEntityPaths(t *testing.T) {
	t.Parallel()
	got := idsFromPaths([]string{
		"work/notes.md",           // under work/ but not a kind subdir → PathKind false
		"work/gaps/G-1-narrow.md", // gap-shaped filename, but G-1 is too narrow to ValidateID → IDFromPath false
		"work/gaps/G-0007-ok.md",  // a well-formed gap → kept
	})
	want := []ID{{Kind: entity.KindGap, ID: "G-0007", Path: "work/gaps/G-0007-ok.md"}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("idsFromPaths mismatch (-want +got):\n%s", diff)
	}
}

// --- M-0212/AC-2: LocalRefIDs degrades cleanly on odd repo states ---

func TestLocalRefIDs_NotARepo_ReturnsNil(t *testing.T) {
	t.Parallel()
	// A plain directory that was never `git init`'d. LocalRefIDs must
	// not error or panic — it degrades to "no cross-branch view".
	got := LocalRefIDs(context.Background(), t.TempDir())
	if got != nil {
		t.Errorf("LocalRefIDs = %v, want nil for a non-repo dir", got)
	}
}

func TestLocalRefIDs_NoBranches_ReturnsNil(t *testing.T) {
	t.Parallel()
	// A freshly-init'd repo has no commit, so refs/heads/main does not
	// yet exist — zero local branches.
	got := LocalRefIDs(context.Background(), initRepo(t))
	if got != nil {
		t.Errorf("LocalRefIDs = %v, want nil for a repo with no branches", got)
	}
}

func TestLocalRefIDs_UnreadableRefSkipped(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	// main carries a readable id.
	commitFile(t, ctx, dir, "work/gaps/G-0001-foo.md", "# foo\n")
	// A sibling branch carries an id, then we corrupt its tip by
	// deleting the commit's tree object. `git for-each-ref` still
	// LISTS the branch (the commit object resolves), but `git ls-tree`
	// on it fails — the "lists but won't read" unreadable-ref case
	// (a real shape under a concurrent worktree branch-deletion race,
	// the very scenario M-0212 targets). LocalRefIDs must skip it and
	// still return main's readable id, with no error and no panic.
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "sibling")
	commitFile(t, ctx, dir, "work/gaps/G-0009-bar.md", "# bar\n")
	treeSHA := strings.TrimSpace(mustOutput(t, ctx, dir, "rev-parse", "sibling^{tree}"))
	mustRun(t, ctx, dir, "checkout", "-q", "main")
	deleteLooseObject(t, dir, treeSHA)

	got := LocalRefIDs(ctx, dir)
	if slices.Contains(got, "G-0009") {
		t.Errorf("LocalRefIDs = %v, want the unreadable sibling id G-0009 skipped", got)
	}
	if !slices.Contains(got, "G-0001") {
		t.Errorf("LocalRefIDs = %v, want main's readable id G-0001 retained", got)
	}
}

// deleteLooseObject removes the loose object file for sha from dir's
// object store. Auto-gc is disabled in tests (testsupport.HardenGitTestEnv),
// so objects in a fresh fixture repo are always loose — no pack to chase.
func deleteLooseObject(t *testing.T, dir, sha string) {
	t.Helper()
	obj := filepath.Join(dir, ".git", "objects", sha[:2], sha[2:])
	if err := os.Remove(obj); err != nil {
		t.Fatalf("removing loose object %s: %v", sha, err)
	}
}

// mustOutput runs a git command and returns its stdout, failing the
// test on error.
func mustOutput(t *testing.T, ctx context.Context, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

// initRepo / commitFile / mustRun mirror the helpers in
// gitops/refs_test.go; duplicated here so this package's tests don't
// depend on internal-test-helper exports from gitops.
// GIT_{AUTHOR,COMMITTER}_{NAME,EMAIL} are seeded once in TestMain
// (setup_test.go) — using t.Setenv here would panic under t.Parallel.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := gitops.Init(context.Background(), dir); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return dir
}

func commitFile(t *testing.T, ctx context.Context, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	mustRun(t, ctx, dir, "add", "--", path)
	mustRun(t, ctx, dir, "commit", "-q", "-m", "add "+path)
}

func mustRun(t *testing.T, ctx context.Context, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
