package integration

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestAdd_FetchReflectsUpstreamID pins M-0213/AC-1: `aiwf add --fetch`
// refreshes the configured trunk ref before computing max, so an id that
// landed on trunk since the last local fetch is seen and skipped. The
// same allocation without --fetch allocates against the stale tracking
// ref and does not.
//
// Setup: an upstream repo carrying G-0001 is cloned twice; upstream then
// advances to G-0002 out-of-band (the clones' refs/remotes/origin/main go
// stale). The --fetch clone refreshes and skips to G-0003; the no-fetch
// clone re-allocates the upstream id (G-0002) — the collision --fetch
// prevents. All local, offline (filesystem clone). Driven in-process so
// the dispatcher's --fetch path is instrumented (the success arm:
// FetchTrunkBestEffort returns nil → no warning).
func TestAdd_FetchReflectsUpstreamID(t *testing.T) {
	t.Parallel()
	up := newUpstreamWithGap(t) // upstream on main carrying G-0001

	cloneFetch := cloneAt(t, up)
	cloneNoFetch := cloneAt(t, up)

	// Advance upstream out-of-band; the clones' tracking refs are now stale.
	mustRun(t, "add", "gap", "--title", "upstream two", "--root", up, "--actor", "human/test")
	if got := gapIDs(t, up); !slices.Contains(got, "G-0002") {
		t.Fatalf("precondition: upstream should carry G-0002, got %v", got)
	}

	// --fetch refreshes origin/main → sees G-0002 → skips to G-0003.
	mustRun(t, "add", "gap", "--fetch", "--title", "fetch three", "--root", cloneFetch, "--actor", "human/test")
	gotFetch := gapIDs(t, cloneFetch)
	if !slices.Contains(gotFetch, "G-0003") || slices.Contains(gotFetch, "G-0002") {
		t.Errorf("--fetch clone gaps = %v, want G-0003 (skipped past upstream G-0002), not G-0002", gotFetch)
	}

	// Without --fetch, the stale trunk ref hides G-0002, so it is re-allocated.
	mustRun(t, "add", "gap", "--title", "nofetch two", "--root", cloneNoFetch, "--actor", "human/test")
	gotNoFetch := gapIDs(t, cloneNoFetch)
	if !slices.Contains(gotNoFetch, "G-0002") {
		t.Errorf("no-fetch clone gaps = %v, want G-0002 (stale trunk re-allocates upstream id)", gotNoFetch)
	}
}

// TestAdd_FetchBestEffort_NoRemote pins M-0213/AC-2: with no remote, the
// fetch fails but degrades to local-only allocation with a warning and a
// success exit — never blocking the add.
//
// SERIAL (no t.Parallel): captures the process-global os.Stderr to assert
// the operator warning. Listed in setup_test.go's serial block.
func TestAdd_FetchBestEffort_NoRemote(t *testing.T) {
	repo := newRepoNoRemote(t)

	var rc int
	stderr := captureStderr(t, func() {
		rc = cli.Execute([]string{"add", "gap", "--fetch", "--root", repo, "--title", "local only", "--actor", "human/test"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf add --fetch (no remote) rc = %d, want OK (best-effort never blocks)\nstderr: %s", rc, stderr)
	}
	if !strings.Contains(stderr, "--fetch") || !strings.Contains(stderr, "allocating against the local view") {
		t.Errorf("stderr should warn about the degraded fetch, got: %q", stderr)
	}
	// The add still succeeded against the local view: G-0001 allocated.
	if got := gapIDs(t, repo); !slices.Contains(got, "G-0001") {
		t.Errorf("gap not created on degraded fetch; gaps = %v", got)
	}
}

// --- helpers ---

// newRepoNoRemote builds a committed git repo with one base commit and
// no remote.
func newRepoNoRemote(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, dir, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	mustGit(t, dir, "add", "-A")
	mustGit(t, dir, "commit", "-q", "-m", "base")
	return dir
}

// newUpstreamWithGap builds a committed upstream repo on main carrying a
// single gap (G-0001), suitable for cloning.
func newUpstreamWithGap(t *testing.T) string {
	t.Helper()
	dir := newRepoNoRemote(t)
	mustRun(t, "add", "gap", "--title", "upstream one", "--root", dir, "--actor", "human/test")
	if got := gapIDs(t, dir); !slices.Contains(got, "G-0001") {
		t.Fatalf("upstream setup: expected G-0001, got %v", got)
	}
	return dir
}

// cloneAt clones src into a fresh temp dir (origin → src) and returns it.
func cloneAt(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	cmd := exec.Command("git", "clone", "-q", src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone: %v\n%s", err, out)
	}
	return dst
}

// captureStderr swaps os.Stderr for a pipe, runs fn, and returns what fn
// wrote to stderr. Mutates a process global, so callers must be serial
// (no t.Parallel) — see setup_test.go.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	_ = w.Close()
	os.Stderr = old
	return <-done
}

// gapIDs returns the sorted gap ids present in root's work/gaps tree.
func gapIDs(t *testing.T, root string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, "work", "gaps", "G-*.md"))
	if err != nil {
		t.Fatalf("glob gaps: %v", err)
	}
	ids := make([]string, 0, len(matches))
	for _, m := range matches {
		parts := strings.SplitN(filepath.Base(m), "-", 3)
		if len(parts) >= 2 {
			ids = append(ids, parts[0]+"-"+parts[1])
		}
	}
	sort.Strings(ids)
	return ids
}
