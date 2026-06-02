package cellcoverage

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/scope"
	"github.com/23min/aiwf/internal/verb"
)

// AuthorizeScope opens an `aiwf authorize` scope on entityID for the
// named agent (e.g. "ai/claude"), commits it in-process, and returns
// the resulting active scope — loaded back through the same git-log
// reader (cliutil.LoadEntityScopes) the cmd layer uses, so a driver
// consuming the scope sees exactly what a production verb invocation
// would.
//
// The scope-entity must already be non-terminal (the authorize verb
// refuses otherwise); the caller brings it up first. Fails the test
// if the authorize commit or the round-trip load does not yield an
// active scope.
//
// M-0159/AC-7 / G-0213: a real branch ref is created in the fixture's
// tmp git repo BEFORE the verb stamps the aiwf-branch: trailer, so
// the trailer value resolves end-to-end. Before this fix the fixture
// stamped a fictional `epic/E-NNNN-cellcoverage-fixture` value
// purely to satisfy verb.Authorize's M-0103 AI-target preflight;
// today's rules don't validate branch resolvability, but the moment
// any future milestone (M-0159 finishing or M-0161) lands a rule
// that does, every M-0125 positive cell test would silently break
// at once. Creating the branch in fixture setup is G-0213's Option
// 1 — keeps rule semantics strict, no production-code coupling to
// fixture markers, ~few-ms overhead per cell. See the test pin at
// authorized_scope_branch_resolves_test.go for the invariant.
func (f *CellFixture) AuthorizeScope(t *testing.T, entityID, agent string) *scope.Scope {
	t.Helper()
	branchName := "epic/" + entityID + "-cellcoverage-fixture"

	// Create the branch ref so a downstream branch-resolution rule
	// can verify the trailer value points at a real ref. `git
	// branch <name>` defaults to HEAD — at this point the fixture
	// has at least one commit (`aiwf init` produced one), so HEAD
	// is a real commit and the new branch resolves immediately.
	cmd := exec.CommandContext(f.ctx, "git", "branch", branchName)
	cmd.Dir = f.Root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create cellcoverage fixture branch %q: %v\n%s",
			branchName, err, strings.TrimSpace(string(out)))
	}

	f.Must(verb.Authorize(f.ctx, f.Tree(), entityID, testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeOpen,
		Agent:  agent,
		Reason: "cellcoverage authorized-scope fixture",
		// M-0103: satisfy the AI-target preflight; the branch
		// named here is the one the helper created above, so the
		// trailer value resolves end-to-end (G-0213 / M-0159/AC-7).
		CurrentBranch: branchName,
	}))

	scopes, err := cliutil.LoadEntityScopes(f.ctx, f.Root, entityID)
	if err != nil { //coverage:ignore LoadEntityScopes only errors on git failure; a healthy fixture after authorize+apply never hits it
		t.Fatalf("LoadEntityScopes(%s): %v", entityID, err)
	}
	for _, s := range scopes {
		if s.State == scope.StateActive {
			return s
		}
	}
	//coverage:ignore unreachable in a well-formed fixture: AuthorizeOpen+Apply always yields one active scope on entityID
	t.Fatalf("no active scope on %s after authorize --to %s", entityID, agent)
	return nil
}
