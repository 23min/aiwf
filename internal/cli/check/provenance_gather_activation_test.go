package check

import (
	"context"
	"testing"
)

// provenance_gather_activation_test.go — G-0270 (round-2 independent
// review finding): gatherActivationCommitsLocalBranches must be scoped
// to LOCAL branches only (`git log --branches`), symmetric with
// branchTips (built from listRitualBranches, which also reads
// refs/heads/ only). An earlier version of this gather used `--all`,
// which also pulls in remote-tracking refs — surfacing a commit that
// exists only on `refs/remotes/origin/*` as a candidate with no local
// tip to judge it against, so RunPromoteOnWrongBranch fired on
// activations that were actually correct, just not yet fetched,
// merged, or pulled locally. Reproduced empirically against a real
// bare-origin + two-clone setup before this test was written.

// TestGatherActivationCommitsLocalBranches_ExcludesRemoteOnlyRef pins
// the fix: a promote commit pushed to origin by a DIFFERENT clone,
// never fetched/merged into this clone's own local main, must not be
// returned as a candidate.
func TestGatherActivationCommitsLocalBranches_ExcludesRemoteOnlyRef(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	origin := t.TempDir()
	gitRun(t, origin, "init", "-q", "--bare", "--initial-branch=main")

	root := t.TempDir()
	gitRun(t, root, "clone", "-q", origin, ".")
	gitRun(t, root, "config", "user.email", "test@example.com")
	gitRun(t, root, "config", "user.name", "test")
	gitRun(t, root, "commit", "-q", "--allow-empty", "-m", "seed")
	gitRun(t, root, "push", "-q", "origin", "main")

	// A second clone activates an epic and pushes it straight to
	// origin/main — this commit is never fetched into root's own
	// local main.
	other := t.TempDir()
	gitRun(t, other, "clone", "-q", origin, ".")
	gitRun(t, other, "config", "user.email", "test@example.com")
	gitRun(t, other, "config", "user.name", "test")
	gitRun(t, other, "commit", "-q", "--allow-empty",
		"-m", "aiwf promote E-0001 proposed -> active",
		"--trailer", "aiwf-verb: promote",
		"--trailer", "aiwf-entity: E-0001",
		"--trailer", "aiwf-to: active")
	remoteOnlySHA := gitOutput(t, other, "rev-parse", "HEAD")
	gitRun(t, other, "push", "-q", "origin", "main")

	// root must actually see the pushed commit via its remote-tracking
	// ref for this test to distinguish anything — without this fetch,
	// root's object store never even has remoteOnlySHA, and the test
	// would pass vacuously regardless of --branches vs --all.
	gitRun(t, root, "fetch", "-q", "origin")
	if got := gitOutput(t, root, "rev-parse", "refs/remotes/origin/main"); got != remoteOnlySHA {
		t.Fatalf("setup: refs/remotes/origin/main = %s, want %s (the fetch didn't pick up the pushed commit)", got, remoteOnlySHA)
	}

	commits, err := gatherActivationCommitsLocalBranches(ctx, root)
	if err != nil {
		t.Fatalf("gatherActivationCommitsLocalBranches: %v", err)
	}
	for _, c := range commits {
		if c.SHA == remoteOnlySHA {
			t.Fatalf("gatherActivationCommitsLocalBranches returned commit %s, reachable only via a remote-tracking ref (origin/main), not any local branch", remoteOnlySHA)
		}
	}
}

// TestGatherActivationCommitsLocalBranches_IncludesLocalNonHEADBranch
// pins the positive counterpart and the original G-0270 incident
// shape: a promote commit on a genuinely LOCAL branch other than the
// one currently checked out must still be returned as a candidate.
func TestGatherActivationCommitsLocalBranches_IncludesLocalNonHEADBranch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	root := t.TempDir()
	gitRun(t, root, "init", "-q", "--initial-branch=main")
	gitRun(t, root, "config", "user.email", "test@example.com")
	gitRun(t, root, "config", "user.name", "test")
	gitRun(t, root, "commit", "-q", "--allow-empty", "-m", "seed")

	gitRun(t, root, "checkout", "-q", "-b", "interloper-branch")
	gitRun(t, root, "commit", "-q", "--allow-empty",
		"-m", "aiwf promote E-0002 proposed -> active",
		"--trailer", "aiwf-verb: promote",
		"--trailer", "aiwf-entity: E-0002",
		"--trailer", "aiwf-to: active")
	localSHA := gitOutput(t, root, "rev-parse", "HEAD")
	gitRun(t, root, "checkout", "-q", "main")

	commits, err := gatherActivationCommitsLocalBranches(ctx, root)
	if err != nil {
		t.Fatalf("gatherActivationCommitsLocalBranches: %v", err)
	}
	found := false
	for _, c := range commits {
		if c.SHA == localSHA {
			found = true
		}
	}
	if !found {
		t.Fatalf("gatherActivationCommitsLocalBranches did not include commit %s on a local branch other than the one currently checked out", localSHA)
	}
}

// TestParseActivationCommitsLocalBranches_DropsMalformedRecord pins
// parseActivationCommitsLocalBranches's malformed-record guard: a
// record with fewer than the two expected fields (SHA + trailers) is
// dropped rather than producing a zero-value scope.Commit.
func TestParseActivationCommitsLocalBranches_DropsMalformedRecord(t *testing.T) {
	t.Parallel()
	const fieldSep = "\x1f"
	const recSep = "\x1e"
	// One well-formed record, one malformed (no field separator at
	// all — a single field, missing the trailers portion entirely).
	in := recSep + "aaa1111" + fieldSep + "aiwf-verb: promote\n" +
		recSep + "malformed-no-separator"
	got := parseActivationCommitsLocalBranches(in)
	if len(got) != 1 {
		t.Fatalf("parseActivationCommitsLocalBranches(%q) = %+v, want exactly 1 well-formed record (the malformed one dropped)", in, got)
	}
	if got[0].SHA != "aaa1111" {
		t.Errorf("SHA = %q, want %q", got[0].SHA, "aaa1111")
	}
}
