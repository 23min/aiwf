package integration

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestAdd_AllocatesPastNonTrunkRemoteRef pins M-0214/AC-1: the allocator
// unions ids from every remote-tracking ref (refs/remotes/*), not just
// the trunk ref. An id that lives only on a non-trunk remote branch
// (a teammate's pushed feature branch) raises the allocated max.
//
// Upstream: main carries G-0001; a non-trunk `feature` branch carries
// G-0009. After cloning, the clone's refs/remotes/origin/feature holds
// G-0009 — invisible to {working tree + local heads + trunk ref}, seen
// only by the new remote-refs scan. The next allocation must skip to
// G-0010, not reuse G-0002. No --fetch needed: the clone already
// populated the remote-tracking refs.
func TestAdd_AllocatesPastNonTrunkRemoteRef(t *testing.T) {
	t.Parallel()
	up := newUpstreamWithGap(t) // main + G-0001
	mustGit(t, up, "checkout", "-q", "-b", "feature")
	writeAndCommitGap(t, up, "G-0009")
	mustGit(t, up, "checkout", "-q", "main")

	clone := cloneAt(t, up)

	mustRun(t, "add", "gap", "--title", "next", "--root", clone, "--actor", "human/test")
	got := gapIDs(t, clone)
	if !slices.Contains(got, "G-0010") {
		t.Errorf("clone gaps = %v, want G-0010 (allocated past non-trunk remote-branch G-0009)", got)
	}
	if slices.Contains(got, "G-0002") {
		t.Errorf("clone gaps = %v, allocated G-0002 — the non-trunk remote ref was not scanned", got)
	}
}

// TestAdd_FetchAllReflectsNonTrunkRemoteID pins M-0214/AC-2: `aiwf add
// --fetch` runs `git fetch --all`, so a branch pushed to a NON-TRUNK
// remote ref after the last local fetch is brought in and its id is
// seen by the AC-1 scan. Without --fetch the branch stays unknown.
func TestAdd_FetchAllReflectsNonTrunkRemoteID(t *testing.T) {
	t.Parallel()
	up := newUpstreamWithGap(t) // main + G-0001
	cloneFetch := cloneAt(t, up)
	cloneNoFetch := cloneAt(t, up)

	// AFTER cloning, push a non-trunk branch upstream carrying a high id —
	// neither clone knows `feature` yet.
	mustGit(t, up, "checkout", "-q", "-b", "feature")
	writeAndCommitGap(t, up, "G-0009")
	mustGit(t, up, "checkout", "-q", "main")

	// --fetch (git fetch --all) brings refs/remotes/origin/feature → the
	// scan sees G-0009 → allocate G-0010.
	mustRun(t, "add", "gap", "--fetch", "--title", "fetched", "--root", cloneFetch, "--actor", "human/test")
	if got := gapIDs(t, cloneFetch); !slices.Contains(got, "G-0010") {
		t.Errorf("--fetch clone gaps = %v, want G-0010 (git fetch --all brought non-trunk feature/G-0009)", got)
	}

	// Without --fetch, feature is unknown locally → allocate G-0002.
	mustRun(t, "add", "gap", "--title", "unfetched", "--root", cloneNoFetch, "--actor", "human/test")
	if got := gapIDs(t, cloneNoFetch); !slices.Contains(got, "G-0002") {
		t.Errorf("no-fetch clone gaps = %v, want G-0002 (feature branch unknown without --fetch)", got)
	}
}

// TestAdd_FetchBadRemote_WarnsButSucceeds pins M-0214/AC-3 (degrade arm):
// a `git fetch --all` failure (here an unreachable remote at a nonexistent
// local path — deterministic and offline) warns to stderr but never blocks
// the add, which allocates against the local view.
//
// SERIAL (no t.Parallel): captures the process-global os.Stderr.
func TestAdd_FetchBadRemote_WarnsButSucceeds(t *testing.T) {
	repo := newRepoNoRemote(t)
	mustGit(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "nope.git"))

	var rc int
	stderr := captureStderr(t, func() {
		rc = cli.Execute([]string{"add", "gap", "--fetch", "--root", repo, "--title", "x", "--actor", "human/test"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf add --fetch (bad remote) rc = %d, want OK (best-effort never blocks)\nstderr: %s", rc, stderr)
	}
	if !strings.Contains(stderr, "--fetch:") || !strings.Contains(stderr, "allocating against the local view") {
		t.Errorf("an unreachable remote should warn, got: %q", stderr)
	}
	if got := gapIDs(t, repo); !slices.Contains(got, "G-0001") {
		t.Errorf("gap not created on degraded fetch; gaps = %v", got)
	}
}

// writeAndCommitGap writes a minimal gap file (filename carries the id;
// content is irrelevant to the ref scan, which reads names via ls-tree)
// and commits it on dir's current branch.
func writeAndCommitGap(t *testing.T, dir, id string) {
	t.Helper()
	full := filepath.Join(dir, "work", "gaps", id+"-x.md")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("# "+id+"\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", id, err)
	}
	mustGit(t, dir, "add", "-A")
	mustGit(t, dir, "commit", "-q", "-m", "add "+id)
}
