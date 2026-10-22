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
	root, _ := initRepoWithMain(t, "github.com/example/downstream")
	ctx := context.Background()
	pseudoVersion := "v0.0.0-20260503120000-abcdef123456"
	info := version.Info{Version: pseudoVersion}
	got := binaryStaleness(ctx, root, config.DefaultAllocateTrunk, info, "github.com/23min/aiwf")
	if got != "" {
		t.Errorf("binaryStaleness out-of-tree = %q, want \"\"", got)
	}
}

// TestBinaryStaleness_SkipsWhenOriginMainAbsent confirms the
// degrade-silently behavior when no refs/remotes/origin/main exists
// (fresh clone, detached state, no remote fetched).
func TestBinaryStaleness_SkipsWhenOriginMainAbsent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := gitops.Init(ctx, root); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module github.com/23min/aiwf\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	mustGit(t, ctx, root, "add", "--", "go.mod")
	mustGit(t, ctx, root, "commit", "-q", "-m", "initial")
	// No update-ref — origin/main does not exist.
	info := version.Info{Version: "v0.0.0-20260503120000-abcdef123456"}
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

// TestBinaryStaleness_StaleSuffix confirms the suffix names both the
// binary SHA and the origin/main SHA when they differ, plus the
// remediation hint.
func TestBinaryStaleness_StaleSuffix(t *testing.T) {
	t.Parallel()
	root, mainSHA := initRepoWithMain(t, "github.com/23min/aiwf")
	ctx := context.Background()
	binarySHA := "abcdef123456"
	// Sanity: binarySHA must not equal mainSHA, else the test asserts
	// the wrong branch. (Fixture authoring discipline — if this trips,
	// pick a different fixture string.)
	if binarySHA == mainSHA {
		t.Fatalf("fixture collision: binarySHA == mainSHA %q", mainSHA)
	}
	info := version.Info{Version: "v0.0.0-20260503120000-" + binarySHA}
	got := binaryStaleness(ctx, root, config.DefaultAllocateTrunk, info, "github.com/23min/aiwf")
	if got == "" {
		t.Fatal("binaryStaleness stale = \"\", want suffix")
	}
	// Structural assertion: both SHAs must appear in the suffix, plus
	// the remediation hint. Substring is OK here because each value is
	// distinct enough that an unrelated location would be a real bug.
	if !strings.Contains(got, binarySHA) {
		t.Errorf("suffix %q missing binary SHA %q", got, binarySHA)
	}
	if !strings.Contains(got, mainSHA) {
		t.Errorf("suffix %q missing main SHA %q", got, mainSHA)
	}
	if !strings.Contains(got, "make install") {
		t.Errorf("suffix %q missing remediation `make install`", got)
	}
	if row := staleRowSuffix(got); !strings.HasPrefix(row, " (stale: ") {
		t.Errorf("row suffix %q must open with \" (stale: \" to append to the binary row", row)
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
	root, _ := initRepoWithMain(t, "github.com/23min/aiwf")
	ctx := context.Background()
	info := version.Info{Version: "v0.0.0-20260503120000-abcdef123456"}
	got := binaryStaleness(ctx, root, config.DefaultAllocateTrunk, info, "")
	if got != "" {
		t.Errorf("binaryStaleness empty-expected-module = %q, want \"\"", got)
	}
}

// TestBinaryStaleness_ComparesConfiguredTrunk confirms the check reads
// the trunk it is handed rather than assuming origin/main: with only a
// non-default trunk ref present, a mismatch is reported against it.
func TestBinaryStaleness_ComparesConfiguredTrunk(t *testing.T) {
	t.Parallel()
	root, trunkSHA := initRepoWithMain(t, "github.com/23min/aiwf")
	ctx := context.Background()
	const trunkRef = "refs/remotes/origin/trunk"
	mustGit(t, ctx, root, "update-ref", trunkRef, "HEAD")
	mustGit(t, ctx, root, "update-ref", "-d", "refs/remotes/origin/main")
	info := version.Info{Version: "v0.0.0-20260503120000-abcdef123456"}
	got := binaryStaleness(ctx, root, trunkRef, info, "github.com/23min/aiwf")
	if !strings.Contains(got, trunkRef+" "+trunkSHA) {
		t.Errorf("suffix %q does not report the configured trunk %s at %s", got, trunkRef, trunkSHA)
	}
}

// TestBinaryStaleness_JudgesEveryBuildThatNamesACommit pins M-0363/AC-1:
// a Makefile stamp, a dirty build and a tagged release are each judged by
// the commit they name — stale when it differs from the ref, silent when
// it is the ref — and a tag the repository lacks is silent.
func TestBinaryStaleness_JudgesEveryBuildThatNamesACommit(t *testing.T) {
	t.Parallel()
	root, refSHA := initRepoWithMain(t, "github.com/23min/aiwf")
	ctx := context.Background()
	mustGit(t, ctx, root, "tag", "-a", "v0.9.0", "-m", "at the ref")
	mustGit(t, ctx, root, "commit", "-q", "--allow-empty", "-m", "past the ref")
	mustGit(t, ctx, root, "tag", "v0.9.1")
	cases := []struct {
		name      string
		version   string
		wantStale bool
	}{
		{"stamp at the ref", "main@v0.8.0-3-g" + refSHA[:9], false},
		{"stamp elsewhere", "main@v0.8.0-3-gabcdef123", true},
		{"dirty stamp at the ref", "main@v0.8.0-3-g" + refSHA[:9] + "-dirty", false},
		{"dirty pseudo-version elsewhere", "v0.0.0-20260503120000-abcdef123456+dirty", true},
		{"tag at the ref", "v0.9.0", false},
		{"tag elsewhere", "v0.9.1", true},
		{"stamp at a tag elsewhere", "main@v0.9.1", true},
		{"tag the repository lacks", "v0.1.0", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := binaryStaleness(ctx, root, config.DefaultAllocateTrunk, version.Parse(tc.version), "github.com/23min/aiwf")
			if stale := got != ""; stale != tc.wantStale {
				t.Errorf("binaryStaleness(%q) = %q, want stale=%v", tc.version, got, tc.wantStale)
			}
			if tc.wantStale && !strings.Contains(got, refSHA) {
				t.Errorf("suffix %q does not name the ref's commit %s", got, refSHA)
			}
		})
	}
}

// TestStaleBinaryWarning pins the line every verb prints before it runs in
// the aiwf repository (M-0363/AC-2): a binary whose commit differs from HEAD
// warns, one at HEAD is silent, and inside a linked worktree a Go
// pseudo-version is not judged, because Go stamps a build there with the
// main checkout's commit rather than the worktree's.
func TestStaleBinaryWarning(t *testing.T) {
	t.Parallel()
	root, headSHA := initRepoWithMain(t, "github.com/23min/aiwf")
	ctx := context.Background()
	linked := filepath.Join(t.TempDir(), "linked")
	mustGit(t, ctx, root, "worktree", "add", "-q", "-b", "side", linked)
	mustGit(t, ctx, linked, "commit", "-q", "--allow-empty", "-m", "past the main checkout")
	cases := []struct {
		name     string
		dir      string
		version  string
		wantWarn bool
	}{
		{"stamp at HEAD", root, "main@v0.8.0-3-g" + headSHA[:9], false},
		{"stamp elsewhere", root, "main@v0.8.0-3-gabcdef123", true},
		{"pseudo-version elsewhere in the main checkout", root, "v0.0.0-20260503120000-abcdef123456", true},
		{"pseudo-version in a linked worktree", linked, "v0.0.0-20260503120000-" + headSHA, false},
		{"stamp elsewhere in a linked worktree", linked, "side@v0.8.0-3-g" + headSHA[:9], true},
		{"pseudo-version outside any repository", t.TempDir(), "v0.0.0-20260503120000-abcdef123456", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := StaleBinaryWarning(ctx, tc.dir, version.Parse(tc.version), "github.com/23min/aiwf")
			if warned := got != ""; warned != tc.wantWarn {
				t.Errorf("StaleBinaryWarning(%q in %s) = %q, want warn=%v", tc.version, tc.name, got, tc.wantWarn)
			}
			if tc.wantWarn && !strings.HasPrefix(got, "aiwf: warning: ") {
				t.Errorf("warning %q does not open with the aiwf warning prefix", got)
			}
		})
	}
}
