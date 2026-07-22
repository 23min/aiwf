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
	// M-0268/AC-1+AC-2: draft -> in_progress now refuses a zero-AC
	// milestone, or one with an empty AC body; seed a real one so the
	// positive arm below exercises the scope machinery, not the
	// AC-completeness guards.
	f.Must(verb.AddACBatch(ctx, f.Tree(), "M-0001", []string{"Does the thing"}, [][]byte{[]byte("Real prose.")}, human))
	f.Must(verb.Add(ctx, f.Tree(), entity.KindEpic, "Out-of-scope Epic", human, verb.AddOptions{}))
	f.Must(verb.Promote(ctx, f.Tree(), "E-0002", entity.StatusActive, human, "", false, verb.PromoteOptions{}))
	f.Must(verb.Add(ctx, f.Tree(), entity.KindMilestone, "Out-of-scope Milestone", human, verb.AddOptions{EpicID: "E-0002", TDD: "none"}))
	// Same seeding on the negative-arm target: the out-of-scope
	// refusal under test must fire before the AC-completeness guards
	// would, not be masked by them.
	f.Must(verb.AddACBatch(ctx, f.Tree(), "M-0002", []string{"Does the other thing"}, [][]byte{[]byte("Real prose.")}, human))

	// Authorize the agent on E-0001 only.
	f.AuthorizeScope(t, "E-0001", "ai/claude")

	const inScope = "M-0001"  // parent E-0001 → reachable
	const outScope = "M-0002" // parent E-0002 → not reachable

	agentArgs := func(target string) []string {
		return []string{"promote", target, "in_progress", "--actor", "ai/claude", "--principal", human}
	}

	// The in-scope promote target (M-0001 -> in_progress) is an
	// activating transition the G-0269 guard polices; check out its
	// parent epic's ritual branch first, matching the real ritual
	// (this fixture never cuts one — AuthorizeScope's own branch ref,
	// above, is a differently-named ref for the authorize verb's own
	// branch-trailer preflight, unrelated to where this promote runs).
	checkoutEpicRitualBranch(t, f, "E-0001")

	// Positive arm: the in-scope agent promote succeeds.
	if out, err := testutil.RunBin(t, f.Root, "", nil, agentArgs(inScope)...); err != nil {
		t.Fatalf("in-scope agent promote should succeed, got error %v:\n%s", err, out)
	}

	// Negative arm: the out-of-scope agent promote is refused and lands
	// no commit. Check out M-0002's own correct ritual branch first —
	// isolating the scope violation under test from the unrelated
	// G-0269 branch guard, which would otherwise ALSO refuse here
	// (HEAD is still on E-0001's branch from the positive arm above)
	// and mask which check actually fired.
	checkoutEpicRitualBranch(t, f, "E-0002")
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
