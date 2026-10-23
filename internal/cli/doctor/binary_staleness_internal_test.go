package doctor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/version"
)

// initRepoWithMain initializes a temp git repo, commits one file, and
// points refs/remotes/origin/main at HEAD. Returns the repo path and
// the 12-char short SHA of the commit. Identity vars are seeded by
// setup_test.go so the helper is t.Parallel-compatible.
func initRepoWithMain(t *testing.T, modulePath string) (root, shortSHA string) {
	t.Helper()
	ctx := context.Background()
	root = t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	// Seed a go.mod so binaryStaleness can read the module path.
	if modulePath != "" {
		if err := os.WriteFile(filepath.Join(root, "go.mod"),
			[]byte("module "+modulePath+"\n\ngo 1.24\n"), 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
	}
	mustGit(t, ctx, root, "add", "--", ".")
	mustGit(t, ctx, root, "commit", "-q", "-m", "initial")
	// Synthesize a refs/remotes/origin/main ref pointing at HEAD —
	// matches the shape a real clone caches after `git fetch origin`.
	mustGit(t, ctx, root, "update-ref", "refs/remotes/origin/main", "HEAD")
	sha, err := gitops.ShortSHA(ctx, root, "refs/remotes/origin/main", 12)
	if err != nil {
		t.Fatalf("ShortSHA: %v", err)
	}
	return root, sha
}

func mustGit(t *testing.T, ctx context.Context, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, string(out))
	}
}

// TestBinaryStaleness_SkipsByShape covers the early returns that
// silence the check before any git work: a version that names no commit
// returns "".
func TestBinaryStaleness_SkipsByShape(t *testing.T) {
	t.Parallel()
	root, _ := initRepoWithMain(t, "github.com/23min/aiwf")
	ctx := context.Background()
	cases := []struct {
		name string
		info version.Info
	}{
		{"devel", version.Info{Version: version.DevelVersion}},
		{"empty", version.Info{}},
		{"non-version garbage", version.Info{Version: "main"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := binaryStaleness(ctx, root, config.DefaultAllocateTrunk, tc.info, "github.com/23min/aiwf")
			if got != "" {
				t.Errorf("binaryStaleness(%s) = %q, want \"\"", tc.name, got)
			}
		})
	}
}

// TestBinaryStaleness_SkipsOutOfTree confirms the kernel-developer
// Lane B constraint: when the repo's go.mod module path doesn't match
// the binary's module path (downstream consumer case), the check
// returns "" silently regardless of binary version.
func TestBinaryStaleness_SkipsOutOfTree(t *testing.T) {
	t.Parallel()
	root, info := staleFixture(t, "github.com/example/downstream")
	got := binaryStaleness(context.Background(), root, config.DefaultAllocateTrunk, info, "github.com/23min/aiwf")
	if got != "" {
		t.Errorf("binaryStaleness out-of-tree = %q, want \"\"", got)
	}
}

// TestBinaryStaleness_SkipsWhenOriginMainAbsent confirms the
// degrade-silently behavior when no refs/remotes/origin/main exists
// (fresh clone, detached state, no remote fetched).
func TestBinaryStaleness_SkipsWhenOriginMainAbsent(t *testing.T) {
	t.Parallel()
	root, info := staleFixture(t, "github.com/23min/aiwf")
	ctx := context.Background()
	mustGit(t, ctx, root, "update-ref", "-d", config.DefaultAllocateTrunk)
	got := binaryStaleness(ctx, root, config.DefaultAllocateTrunk, info, "github.com/23min/aiwf")
	if got != "" {
		t.Errorf("binaryStaleness no-origin-main = %q, want \"\"", got)
	}
}

// TestBinaryStaleness_OkWhenSHAsMatch confirms no suffix is rendered
// when the binary's pseudo SHA equals origin/main's short SHA.
func TestBinaryStaleness_OkWhenSHAsMatch(t *testing.T) {
	t.Parallel()
	root, sha := initRepoWithMain(t, "github.com/23min/aiwf")
	ctx := context.Background()
	// Construct a pseudo-version that ends in the real origin/main SHA.
	info := version.Info{Version: "v0.0.0-20260503120000-" + sha}
	got := binaryStaleness(ctx, root, config.DefaultAllocateTrunk, info, "github.com/23min/aiwf")
	if got != "" {
		t.Errorf("binaryStaleness sha-match = %q, want \"\"", got)
	}
}

// TestBinaryStaleness_StaleSuffix pins the reason binaryStaleness gives
// for a binary whose source differs from the ref, and the row suffix doctor
// appends it with.
func TestBinaryStaleness_StaleSuffix(t *testing.T) {
	t.Parallel()
	root, _ := initRepoWithMain(t, "github.com/23min/aiwf")
	ctx := context.Background()
	built := headSHA(t, ctx, root)
	commitFile(t, ctx, root, "internal/x.go", "package x\n")
	mustGit(t, ctx, root, "update-ref", config.DefaultAllocateTrunk, "HEAD")
	info := version.Info{Version: "v0.0.0-20260503120000-" + built[:12]}
	got := binaryStaleness(ctx, root, config.DefaultAllocateTrunk, info, "github.com/23min/aiwf")
	want := "built from " + built[:12] + ", whose source differs from " + config.DefaultAllocateTrunk
	if got != want {
		t.Errorf("binaryStaleness = %q, want %q", got, want)
	}
	if row := staleRowSuffix(got); row != " (stale: "+want+")" {
		t.Errorf("row suffix = %q, want it to wrap the reason", row)
	}
	if row := staleRowSuffix(""); row != "" {
		t.Errorf("row suffix for a current binary = %q, want none", row)
	}
}

// TestBinaryStaleness_SkipsOnMissingGoMod covers readModulePath's
// os.ReadFile error path: when rootDir has no go.mod, the check
// degrades silently rather than erroring or claiming staleness.
func TestBinaryStaleness_SkipsOnMissingGoMod(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir() // bare tempdir — no git init, no go.mod
	info := version.Info{Version: "v0.0.0-20260503120000-abcdef123456"}
	got := binaryStaleness(ctx, root, config.DefaultAllocateTrunk, info, "github.com/23min/aiwf")
	if got != "" {
		t.Errorf("binaryStaleness no-go.mod = %q, want \"\"", got)
	}
}

// TestBinaryStaleness_SkipsOnMalformedGoMod covers readModulePath's
// no-module-line return: when go.mod parses but lacks a `module`
// declaration, the check degrades silently (matches the
// "consumer-side" branch via the rootModule != expectedModule guard).
func TestBinaryStaleness_SkipsOnMalformedGoMod(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("go 1.24\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	info := version.Info{Version: "v0.0.0-20260503120000-abcdef123456"}
	got := binaryStaleness(ctx, root, config.DefaultAllocateTrunk, info, "github.com/23min/aiwf")
	if got != "" {
		t.Errorf("binaryStaleness malformed-go.mod = %q, want \"\"", got)
	}
}

// TestBinaryStaleness_SkipsOnEmptyExpectedModule covers the
// version.ModulePath() == "" guard at the call site equivalent —
// when the binary lacks build info, the helper degrades silently.
func TestBinaryStaleness_SkipsOnEmptyExpectedModule(t *testing.T) {
	t.Parallel()
	root, info := staleFixture(t, "github.com/23min/aiwf")
	got := binaryStaleness(context.Background(), root, config.DefaultAllocateTrunk, info, "")
	if got != "" {
		t.Errorf("binaryStaleness empty-expected-module = %q, want \"\"", got)
	}
}

// TestBinaryStaleness_ComparesConfiguredTrunk confirms the check reads
// the trunk it is handed rather than assuming origin/main: with only a
// non-default trunk ref present, a difference is reported against it.
func TestBinaryStaleness_ComparesConfiguredTrunk(t *testing.T) {
	t.Parallel()
	root, _ := initRepoWithMain(t, "github.com/23min/aiwf")
	ctx := context.Background()
	built := headSHA(t, ctx, root)
	commitFile(t, ctx, root, "internal/x.go", "package x\n")
	const trunkRef = "refs/remotes/origin/trunk"
	mustGit(t, ctx, root, "update-ref", trunkRef, "HEAD")
	mustGit(t, ctx, root, "update-ref", "-d", "refs/remotes/origin/main")
	info := version.Info{Version: "v0.0.0-20260503120000-" + built[:12]}
	got := binaryStaleness(ctx, root, trunkRef, info, "github.com/23min/aiwf")
	if !strings.Contains(got, "differs from "+trunkRef) {
		t.Errorf("suffix %q does not report the configured trunk %s", got, trunkRef)
	}
}

// TestBinaryStaleness_JudgesTheSourceABuildNames pins M-0363/AC-1: every
// build that names a revision — a Makefile stamp, a dirty build, a tagged
// release — is stale exactly when the binary's source differs from the
// ref's; a commit that changes only planning files or tests leaves it
// current, and a revision the repository lacks is silent.
func TestBinaryStaleness_JudgesTheSourceABuildNames(t *testing.T) {
	t.Parallel()
	root, _ := initRepoWithMain(t, "github.com/23min/aiwf")
	ctx := context.Background()
	mustGit(t, ctx, root, "tag", "-a", "v0.9.0", "-m", "old source")
	old := headSHA(t, ctx, root)
	current := commitFile(t, ctx, root, "internal/x.go", "package x\n")
	mustGit(t, ctx, root, "tag", "v0.9.1")
	commitFile(t, ctx, root, "work/plan.md", "planning\n")
	commitFile(t, ctx, root, "internal/x_test.go", "package x\n")
	mustGit(t, ctx, root, "update-ref", config.DefaultAllocateTrunk, "HEAD")
	cases := []struct {
		name      string
		version   string
		wantStale bool
	}{
		{"stamp at a planning- and test-only distance", "main@v0.9.1-1-g" + current[:9], false},
		{"stamp at older source", "main@v0.9.0-1-g" + old[:9], true},
		{"dirty stamp at current source", "main@v0.9.1-1-g" + current[:9] + "-dirty", false},
		{"dirty pseudo-version at older source", "v0.0.0-20260503120000-" + old[:12] + "+dirty", true},
		{"tag at current source", "v0.9.1", false},
		{"tag at older source", "v0.9.0", true},
		{"stamp at a tag at older source", "main@v0.9.0", true},
		{"tag the repository lacks", "v0.1.0", false},
		{"commit the repository lacks", "main@v0.9.1-1-gabcdef123", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := binaryStaleness(ctx, root, config.DefaultAllocateTrunk, version.Parse(tc.version), "github.com/23min/aiwf")
			if stale := got != ""; stale != tc.wantStale {
				t.Errorf("binaryStaleness(%q) = %q, want stale=%v", tc.version, got, tc.wantStale)
			}
		})
	}
}

// TestStaleBinaryWarning pins the line every verb prints before it runs in
// the aiwf repository (M-0363/AC-2): a binary whose source differs from HEAD
// warns, one at HEAD's source is silent, and inside a linked worktree a Go
// pseudo-version is not judged, dirty or not, because the go1.25 toolchain
// stamps a build there with the main checkout's commit rather than the
// worktree's.
func TestStaleBinaryWarning(t *testing.T) {
	t.Parallel()
	root, _ := initRepoWithMain(t, "github.com/23min/aiwf")
	ctx := context.Background()
	old := headSHA(t, ctx, root)
	current := commitFile(t, ctx, root, "internal/x.go", "package x\n")
	linked := filepath.Join(t.TempDir(), "linked")
	mustGit(t, ctx, root, "worktree", "add", "-q", "-b", "side", linked)
	cases := []struct {
		name     string
		dir      string
		version  string
		wantWarn bool
	}{
		{"stamp at HEAD", root, "main@v0.8.0-3-g" + current[:9], false},
		{"stamp at older source", root, "main@v0.8.0-3-g" + old[:9], true},
		{"pseudo-version at older source in the main checkout", root, "v0.0.0-20260503120000-" + old[:12], true},
		{"pseudo-version in a linked worktree", linked, "v0.0.0-20260503120000-" + old[:12], false},
		{"dirty pseudo-version in a linked worktree", linked, "v0.0.0-20260503120000-" + old[:12] + "+dirty", false},
		{"stamp at older source in a linked worktree", linked, "side@v0.8.0-3-g" + old[:9], true},
		{"pseudo-version outside any repository", t.TempDir(), "v0.0.0-20260503120000-" + old[:12], false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := StaleBinaryWarning(ctx, tc.dir, version.Parse(tc.version), "github.com/23min/aiwf")
			if warned := got != ""; warned != tc.wantWarn {
				t.Errorf("StaleBinaryWarning(%q in %s) = %q, want warn=%v", tc.version, tc.name, got, tc.wantWarn)
			}
		})
	}
	want := "aiwf: warning: stale binary — built from " + old[:9] + ", whose source differs from HEAD"
	if got := StaleBinaryWarning(ctx, root, version.Parse("main@v0.8.0-3-g"+old[:9]), "github.com/23min/aiwf"); got != want {
		t.Errorf("StaleBinaryWarning = %q, want %q", got, want)
	}
}

// headSHA returns the full commit hash of dir's HEAD.
func headSHA(t *testing.T, ctx context.Context, dir string) string {
	t.Helper()
	sha, err := gitops.ResolveCommitSHA(ctx, dir, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return sha
}

// commitFile writes content to rel under dir, commits it, and returns the
// new commit's full hash.
func commitFile(t *testing.T, ctx context.Context, dir, rel, content string) string {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, ctx, dir, "add", "--", rel)
	mustGit(t, ctx, dir, "commit", "-q", "-m", "change "+rel)
	return headSHA(t, ctx, dir)
}

// staleFixture builds a repository with module modulePath whose trunk ref
// is one source change past the commit the returned binary version names,
// so binaryStaleness reports it stale unless the guard under test silences
// it.
func staleFixture(t *testing.T, modulePath string) (root string, info version.Info) {
	t.Helper()
	root, _ = initRepoWithMain(t, modulePath)
	ctx := context.Background()
	built := headSHA(t, ctx, root)
	commitFile(t, ctx, root, "internal/x.go", "package x\n")
	mustGit(t, ctx, root, "update-ref", config.DefaultAllocateTrunk, "HEAD")
	return root, version.Info{Version: "v0.0.0-20260503120000-" + built[:12]}
}
