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

// --- M-0259/AC-1: LocalRefHits/RemoteRefHits widen the view to carry
// per-hit (kind, id, path, ref) instead of collapsing to bare ids ---

func TestLocalRefHits_CarriesKindPathAndRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-0001-foo.md", "# foo\n")
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "sibling")
	commitFile(t, ctx, dir, "work/gaps/G-0005-bar.md", "# bar\n")
	mustRun(t, ctx, dir, "checkout", "-q", "main")

	got := LocalRefHits(ctx, dir)
	var siblingHit *RefHit
	for i := range got {
		if got[i].ID == "G-0005" {
			siblingHit = &got[i]
		}
	}
	if siblingHit == nil {
		t.Fatalf("LocalRefHits = %v, want a hit for sibling-only id G-0005", got)
	}
	if siblingHit.Kind != entity.KindGap {
		t.Errorf("siblingHit.Kind = %q, want %q", siblingHit.Kind, entity.KindGap)
	}
	if siblingHit.Path != "work/gaps/G-0005-bar.md" {
		t.Errorf("siblingHit.Path = %q, want work/gaps/G-0005-bar.md", siblingHit.Path)
	}
	if siblingHit.Ref != "refs/heads/sibling" {
		t.Errorf("siblingHit.Ref = %q, want refs/heads/sibling", siblingHit.Ref)
	}
}

func TestRemoteRefHits_CarriesKindPathAndRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	up := initRepo(t)
	commitFile(t, ctx, up, "work/gaps/G-0001-foo.md", "# foo\n")
	mustRun(t, ctx, up, "checkout", "-q", "-b", "feature")
	commitFile(t, ctx, up, "work/gaps/G-0005-bar.md", "# bar\n")
	mustRun(t, ctx, up, "checkout", "-q", "main")
	clone := cloneRepo(t, up)

	got := RemoteRefHits(ctx, clone)
	var featureHit *RefHit
	for i := range got {
		if got[i].ID == "G-0005" {
			featureHit = &got[i]
		}
	}
	if featureHit == nil {
		t.Fatalf("RemoteRefHits = %v, want a hit for feature-branch id G-0005", got)
	}
	if featureHit.Ref != "refs/remotes/origin/feature" {
		t.Errorf("featureHit.Ref = %q, want refs/remotes/origin/feature", featureHit.Ref)
	}
	if featureHit.Path != "work/gaps/G-0005-bar.md" {
		t.Errorf("featureHit.Path = %q, want work/gaps/G-0005-bar.md", featureHit.Path)
	}
}

func TestLocalRefIDs_DerivedFromHits_Unaffected(t *testing.T) {
	// AC-1: the widening is additive — LocalRefIDs' existing []string
	// consumption (the allocator) must not change shape. Pin that the
	// plain id-string view stays index-aligned with the hit view.
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-0001-foo.md", "# foo\n")
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "sibling")
	commitFile(t, ctx, dir, "work/gaps/G-0005-bar.md", "# bar\n")
	mustRun(t, ctx, dir, "checkout", "-q", "main")

	ids := LocalRefIDs(ctx, dir)
	hits := LocalRefHits(ctx, dir)
	if len(ids) != len(hits) {
		t.Fatalf("LocalRefIDs = %v (%d), LocalRefHits = %v (%d), want same length", ids, len(ids), hits, len(hits))
	}
	for i, h := range hits {
		if ids[i] != h.ID {
			t.Errorf("LocalRefIDs[%d] = %q, want %q (from LocalRefHits[%d])", i, ids[i], h.ID, i)
		}
	}
}

// --- M-0260/AC-3: DistinctRefs supplies the candidate-ref list a
// caller (aiwf show/list) surfaces when it declines to arbitrate a
// cross-branch-collision. ---

func TestDistinctRefs_DedupesPreservingFirstSeenOrder(t *testing.T) {
	t.Parallel()
	hits := []RefHit{
		{ID: "G-0001", Ref: "refs/heads/b"},
		{ID: "G-0001", Ref: "refs/heads/a"},
		{ID: "G-0001", Ref: "refs/heads/b"},
	}
	got := DistinctRefs(hits)
	want := []string{"refs/heads/b", "refs/heads/a"}
	if len(got) != len(want) {
		t.Fatalf("DistinctRefs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("DistinctRefs[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDistinctRefs_Empty(t *testing.T) {
	t.Parallel()
	if got := DistinctRefs(nil); got != nil {
		t.Errorf("DistinctRefs(nil) = %v, want nil", got)
	}
}

// --- M-0259/AC-3: DetectCollisions compares blob content across every
// ref holding the same id, escalating genuine divergence. ---

func TestDetectCollisions_IdenticalContentAcrossRefs_NoCollision(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-0001-foo.md", "same content\n")
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "sibling")
	// Sibling touches an unrelated file so the id's own file is
	// byte-identical on both refs (same content, genuinely not merged
	// yet — the pending case, not a collision).
	commitFile(t, ctx, dir, "work/gaps/G-0002-unrelated.md", "unrelated\n")
	mustRun(t, ctx, dir, "checkout", "-q", "main")

	hits := LocalRefHits(ctx, dir)
	got := DetectCollisions(ctx, dir, hits)
	if got["G-0001"] {
		t.Errorf("DetectCollisions = %v, want no collision for G-0001 (identical content on every ref)", got)
	}
}

func TestDetectCollisions_DivergentContentAcrossRefs_Collision(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-0001-foo.md", "main version\n")
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "sibling")
	// Sibling independently holds the SAME id at the SAME path with
	// DIFFERENT content — a genuine collision neither side has merged
	// yet (G-0415's motivating class).
	writeFile(t, dir, "work/gaps/G-0001-foo.md", "sibling version — diverged\n")
	mustRun(t, ctx, dir, "add", "work/gaps/G-0001-foo.md")
	mustRun(t, ctx, dir, "commit", "-q", "-m", "sibling diverges G-0001")
	mustRun(t, ctx, dir, "checkout", "-q", "main")

	hits := LocalRefHits(ctx, dir)
	got := DetectCollisions(ctx, dir, hits)
	if !got["G-0001"] {
		t.Errorf("DetectCollisions = %v, want collision=true for G-0001 (divergent content across refs)", got)
	}
}

func TestDetectCollisions_SingleHit_NeverCollision(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-0001-foo.md", "only ever on main\n")

	hits := LocalRefHits(ctx, dir)
	got := DetectCollisions(ctx, dir, hits)
	if got["G-0001"] {
		t.Errorf("DetectCollisions = %v, want no collision for a single-ref id", got)
	}
}

func TestDetectCollisions_NoHits_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	got := DetectCollisions(ctx, dir, nil)
	if len(got) != 0 {
		t.Errorf("DetectCollisions(nil) = %v, want empty", got)
	}
}

func TestDetectCollisions_MultipleMultiHitIDs_ReusesSubprocess(t *testing.T) {
	// Covers the br == nil check's reuse (false) branch: a second
	// multi-hit id in the same call must not spawn a second
	// BlobReader subprocess.
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-0001-foo.md", "main G1\n")
	commitFile(t, ctx, dir, "work/gaps/G-0002-bar.md", "main G2\n")
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "sibling")
	writeFile(t, dir, "work/gaps/G-0001-foo.md", "sibling G1 diverged\n")
	writeFile(t, dir, "work/gaps/G-0002-bar.md", "sibling G2 diverged\n")
	mustRun(t, ctx, dir, "add", "work/gaps/G-0001-foo.md", "work/gaps/G-0002-bar.md")
	mustRun(t, ctx, dir, "commit", "-q", "-m", "sibling diverges both")
	mustRun(t, ctx, dir, "checkout", "-q", "main")

	hits := LocalRefHits(ctx, dir)
	got := DetectCollisions(ctx, dir, hits)
	if !got["G-0001"] || !got["G-0002"] {
		t.Errorf("DetectCollisions = %v, want collisions for both G-0001 and G-0002", got)
	}
}

func TestDetectCollisions_UnreadableHitExcluded_NoFalseCollision(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-0001-foo.md", "same content\n")
	mustRun(t, ctx, dir, "checkout", "-q", "-b", "sibling")
	mustRun(t, ctx, dir, "checkout", "-q", "main")

	// main and sibling both carry G-0001 with identical content (sibling
	// never touched it). Inject a bogus third hit for the same id at a
	// path that does not exist on its ref — Stat fails for it, and
	// DetectCollisions must exclude it rather than treating the read
	// failure as a spurious divergence.
	hits := LocalRefHits(ctx, dir)
	hits = append(hits, RefHit{Kind: entity.KindGap, ID: "G-0001", Path: "work/gaps/does-not-exist.md", Ref: "refs/heads/sibling"})

	got := DetectCollisions(ctx, dir, hits)
	if got["G-0001"] {
		t.Errorf("DetectCollisions = %v, want no collision — the unreadable hit must be excluded, not treated as divergent", got)
	}
}

func TestDetectCollisions_BlobReaderUnavailable_DegradesToNoCollision(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	notARepo := t.TempDir()
	hits := []RefHit{
		{Kind: entity.KindGap, ID: "G-0001", Path: "work/gaps/G-0001-foo.md", Ref: "refs/heads/main"},
		{Kind: entity.KindGap, ID: "G-0001", Path: "work/gaps/G-0001-foo.md", Ref: "refs/heads/sibling"},
	}
	got := DetectCollisions(ctx, notARepo, hits)
	if len(got) != 0 {
		t.Errorf("DetectCollisions = %v, want empty when BlobReader can't be constructed (not a repo)", got)
	}
}

// writeFile overwrites path's content without committing — the
// caller commits separately. Distinct from commitFile, which does
// both in one step.
func writeFile(t *testing.T, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// --- M-0214: RemoteRefIDs (the allocator's remote-side cross-branch view) ---

func TestRemoteRefIDs_UnionsRemoteBranchIDs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Upstream with two branches: main carries G-0001, a non-trunk
	// `feature` branch carries G-0005.
	up := initRepo(t)
	commitFile(t, ctx, up, "work/gaps/G-0001-foo.md", "# foo\n")
	mustRun(t, ctx, up, "checkout", "-q", "-b", "feature")
	commitFile(t, ctx, up, "work/gaps/G-0005-bar.md", "# bar\n")
	mustRun(t, ctx, up, "checkout", "-q", "main")
	// Clone: refs/remotes/origin/{main,feature} are populated.
	clone := cloneRepo(t, up)

	got := RemoteRefIDs(ctx, clone)
	if !slices.Contains(got, "G-0005") {
		t.Errorf("RemoteRefIDs = %v, want non-trunk remote-branch id G-0005", got)
	}
	if !slices.Contains(got, "G-0001") {
		t.Errorf("RemoteRefIDs = %v, want main remote-branch id G-0001", got)
	}
}

func TestRemoteRefIDs_NoRemotes_ReturnsNil(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := initRepo(t)
	commitFile(t, ctx, dir, "work/gaps/G-0001-foo.md", "# foo\n")
	if got := RemoteRefIDs(ctx, dir); got != nil {
		t.Errorf("RemoteRefIDs = %v, want nil for a repo with no remotes", got)
	}
}

func TestRemoteRefIDs_NotARepo_ReturnsNil(t *testing.T) {
	t.Parallel()
	if got := RemoteRefIDs(context.Background(), t.TempDir()); got != nil {
		t.Errorf("RemoteRefIDs = %v, want nil for a non-repo dir", got)
	}
}

// cloneRepo clones src into a fresh temp dir (origin → src) and returns it.
func cloneRepo(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	cmd := exec.CommandContext(context.Background(), "git", "clone", "-q", src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone: %v\n%s", err, out)
	}
	return dst
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
