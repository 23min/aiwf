package cellcoverage

import (
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
func (f *CellFixture) AuthorizeScope(t *testing.T, entityID, agent string) *scope.Scope {
	t.Helper()
	f.Must(verb.Authorize(f.ctx, f.Tree(), entityID, testActor, verb.AuthorizeOptions{
		Mode:   verb.AuthorizeOpen,
		Agent:  agent,
		Reason: "cellcoverage authorized-scope fixture",
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
