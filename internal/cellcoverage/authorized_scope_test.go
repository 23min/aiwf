package cellcoverage

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/scope"
	"github.com/23min/aiwf/internal/verb"
)

// TestCellFixture_AuthorizeScope is M-0146/AC-1: the fixture can build
// an authorized-scope context — an open `aiwf authorize` scope on an
// entity with an agent actor — and the resulting scope is active and
// loadable (round-tripped through the same git-log loader the cmd
// layer uses).
func TestCellFixture_AuthorizeScope(t *testing.T) {
	t.Parallel()
	f := NewCellFixture(t)

	// An active epic to authorize an agent on.
	f.Must(verb.Add(f.ctx, f.Tree(), entity.KindEpic, "Scope Epic", testActor, verb.AddOptions{}))
	f.Must(verb.Promote(f.ctx, f.Tree(), "E-0001", entity.StatusActive, testActor, "", false, verb.PromoteOptions{}))

	s := f.AuthorizeScope(t, "E-0001", "ai/claude")

	if s == nil {
		t.Fatal("AuthorizeScope returned nil scope")
	}
	if s.State != scope.StateActive {
		t.Errorf("scope state = %q, want %q", s.State, scope.StateActive)
	}
	if s.Entity != "E-0001" {
		t.Errorf("scope entity = %q, want E-0001", s.Entity)
	}
	if s.Agent != "ai/claude" {
		t.Errorf("scope agent = %q, want ai/claude", s.Agent)
	}
}
