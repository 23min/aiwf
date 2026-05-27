package policies

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cellcoverage"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestM0146_ScopeReachMachinery is M-0146/AC-2: the authorized-scope
// fixture (AC-1) plus the driver subprocess path exercise a scope-gated
// cell both ways. An agent authorized on E-0001 may act on an in-scope
// target (a milestone under E-0001) but is refused on an out-of-scope
// target (a milestone under E-0002), via the existing runtime
// provenance-authorization-out-of-scope gate (M-0141). The global spec
// rule lands in M-0147; this proves the machinery can drive a
// scope-gated cell positive and negative.
func TestM0146_ScopeReachMachinery(t *testing.T) {
	t.Parallel()
	f := cellcoverage.NewCellFixture(t)
	ctx := context.Background()
	const human = "human/test"

	// Two active epics, each with a draft milestone (a legal promote
	// target). The fixture allocates ids sequentially: E-0001/M-0001
	// under the first epic, E-0002/M-0002 under the second.
	f.Must(verb.Add(ctx, f.Tree(), entity.KindEpic, "In-scope Epic", human, verb.AddOptions{}))
	f.Must(verb.Promote(ctx, f.Tree(), "E-0001", entity.StatusActive, human, "", false, verb.PromoteOptions{}))
	f.Must(verb.Add(ctx, f.Tree(), entity.KindMilestone, "In-scope Milestone", human, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	f.Must(verb.Add(ctx, f.Tree(), entity.KindEpic, "Out-of-scope Epic", human, verb.AddOptions{}))
	f.Must(verb.Promote(ctx, f.Tree(), "E-0002", entity.StatusActive, human, "", false, verb.PromoteOptions{}))
	f.Must(verb.Add(ctx, f.Tree(), entity.KindMilestone, "Out-of-scope Milestone", human, verb.AddOptions{EpicID: "E-0002", TDD: "none"}))

	// Authorize the agent on E-0001 only.
	f.AuthorizeScope(t, "E-0001", "ai/claude")

	const inScope = "M-0001"  // parent E-0001 → reachable
	const outScope = "M-0002" // parent E-0002 → not reachable

	agentArgs := func(target string) []string {
		return []string{"promote", target, "in_progress", "--actor", "ai/claude", "--principal", human}
	}

	// Positive arm: the in-scope agent promote succeeds.
	if out, err := testutil.RunBin(t, f.Root, "", nil, agentArgs(inScope)...); err != nil {
		t.Fatalf("in-scope agent promote should succeed, got error %v:\n%s", err, out)
	}

	// Negative arm: the out-of-scope agent promote is refused and lands
	// no commit.
	headBefore, err := testutil.RunGit(f.Root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	out, runErr := testutil.RunBin(t, f.Root, "", nil, agentArgs(outScope)...)
	if runErr == nil {
		t.Fatalf("out-of-scope agent promote should be refused, but succeeded:\n%s", out)
	}
	if !strings.Contains(out, "provenance-authorization-out-of-scope") {
		t.Errorf("refusal should cite provenance-authorization-out-of-scope; got:\n%s", out)
	}
	headAfter, err := testutil.RunGit(f.Root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	if headBefore != headAfter {
		t.Errorf("refused verb must not commit: HEAD moved %s → %s", headBefore, headAfter)
	}
}
