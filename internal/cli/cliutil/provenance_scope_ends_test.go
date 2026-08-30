package cliutil

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

// TestLoadEndableScopeAuthSHAsForEntity_SelectsEveryNonEndedScope is
// M-0325/AC-4 at the package layer.
//
// The seam-level proof — that an entity reaching a terminal status
// leaves no scope stranded — lives in internal/cli/integration, but it
// drives the binary as a subprocess and so contributes nothing to this
// package's coverage; this is what exercises the selection itself.
//
// One scope of each state sits on the entity at once, so the assertion
// distinguishes the three answers a predicate could give rather than
// confirming a single one: active-only strands the paused scope,
// paused-only strands the active scope, and everything-including-ended
// re-ends a scope already terminal, which would put a second
// aiwf-scope-ends for the same SHA on the record.
func TestLoadEndableScopeAuthSHAsForEntity_SelectsEveryNonEndedScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// runGit seeds the GIT identity vars itself, so no config step here.
	runGit(t, root, "init", "-q", "-b", "main")

	active := openScopeCommit(t, root, "E-0001", "ai/active")
	paused := openScopeCommit(t, root, "E-0001", "ai/paused")
	runGit(t, root, "commit", "--allow-empty", "-m", "aiwf authorize E-0001 --pause",
		"--trailer", "aiwf-verb: authorize", "--trailer", "aiwf-entity: E-0001",
		"--trailer", "aiwf-actor: human/test", "--trailer", "aiwf-scope: paused",
		"--trailer", "aiwf-reason: holding")
	ended := openScopeCommit(t, root, "E-0001", "ai/ended")
	runGit(t, root, "commit", "--allow-empty", "-m", "aiwf authorize E-0001 --end",
		"--trailer", "aiwf-verb: authorize", "--trailer", "aiwf-entity: E-0001",
		"--trailer", "aiwf-actor: human/test", "--trailer", "aiwf-scope-ends: "+ended,
		"--trailer", "aiwf-reason: withdrawn")
	// A scope on a different entity: the selection is per entity, and
	// without one here a predicate ignoring entityID would still pass.
	elsewhere := openScopeCommit(t, root, "E-0002", "ai/other")

	got, err := loadEndableScopeAuthSHAsForEntity(context.Background(), root, "E-0001")
	if err != nil {
		t.Fatalf("loadEndableScopeAuthSHAsForEntity: %v", err)
	}

	// The pause commit lands on the most-recently-opened active scope,
	// which is `paused` — the replay's own rule, not an assumption here.
	want := map[string]string{active: "the active scope", paused: "the paused scope"}
	if len(got) != len(want) {
		t.Fatalf("selected %d scopes, want %d: got %v", len(got), len(want), got)
	}
	for _, sha := range got {
		if _, ok := want[sha]; !ok {
			switch sha {
			case ended:
				t.Errorf("selected the already-ended scope; a second aiwf-scope-ends for one SHA "+
					"records a termination that did not happen: %s", sha[:8])
			case elsewhere:
				t.Errorf("selected a scope opened on E-0002; the side effect ends the closing "+
					"entity's delegations, not every delegation in the repo: %s", sha[:8])
			default:
				t.Errorf("selected an unrecognized SHA %s", sha[:8])
			}
			continue
		}
		delete(want, sha)
	}
	for sha, what := range want {
		t.Errorf("%s (%s) was not selected; its entity is closing, so nothing will ever resume it",
			what, sha[:8])
	}
}

// openScopeCommit writes an authorize-opener commit and returns its SHA,
// which is the AuthSHA the replay assigns the resulting scope.
func openScopeCommit(t *testing.T, root, entityID, agent string) string {
	t.Helper()
	runGit(t, root, "commit", "--allow-empty", "-m", "aiwf authorize "+entityID+" --to "+agent,
		"--trailer", "aiwf-verb: authorize",
		"--trailer", "aiwf-entity: "+entityID,
		"--trailer", "aiwf-actor: human/test",
		"--trailer", "aiwf-to: "+agent,
		"--trailer", "aiwf-scope: opened")
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}
