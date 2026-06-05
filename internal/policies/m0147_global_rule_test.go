package policies

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cellcoverage"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// globalScopeReachRule returns the scope-reach global rule from
// spec.GlobalRules() (ADR-0013: global rules live in their own accessor,
// not in Rules()), failing if it is absent or duplicated. As of E-0030
// the accessor holds more than one global rule (the branch-context
// preflight cells joined the catalog), so this helper selects by the
// rule's ExpectedErrorCode rather than by slice index.
func globalScopeReachRule(t *testing.T) spec.Rule {
	t.Helper()
	const code = "provenance-authorization-out-of-scope"
	globals := spec.GlobalRules()
	var matchIdxs []int
	for i := range globals {
		if globals[i].ExpectedErrorCode == code {
			matchIdxs = append(matchIdxs, i)
		}
	}
	if len(matchIdxs) != 1 {
		t.Fatalf("expected exactly 1 rule in spec.GlobalRules() with ExpectedErrorCode=%q, got %d", code, len(matchIdxs))
	}
	return globals[matchIdxs[0]]
}

// TestM0147_AC1_GlobalRulePresent asserts the marked global scope-reach
// rule exists with M-0144 / ADR-0013's shape: Global, empty cell
// coordinate, Illegal+VerbTime+BlockingStrict, the scope-reach == false
// precondition, the code, and the D-0006 source.
func TestM0147_AC1_GlobalRulePresent(t *testing.T) {
	t.Parallel()
	r := globalScopeReachRule(t)

	if r.Kind != "" || r.FromState != "" || r.Verb != "" {
		t.Errorf("global rule should have empty cell coordinate, got Kind=%q FromState=%q Verb=%q", r.Kind, r.FromState, r.Verb)
	}
	if r.Outcome != spec.OutcomeIllegal {
		t.Errorf("global rule Outcome = %v, want OutcomeIllegal", r.Outcome)
	}
	if r.RejectionLayer != spec.RejectionLayerVerbTime {
		t.Errorf("global rule RejectionLayer = %v, want RejectionLayerVerbTime", r.RejectionLayer)
	}
	if !r.BlockingStrict {
		t.Error("global rule should be BlockingStrict (verb-time refusal)")
	}
	if r.ExpectedErrorCode != "provenance-authorization-out-of-scope" {
		t.Errorf("global rule ExpectedErrorCode = %q, want provenance-authorization-out-of-scope", r.ExpectedErrorCode)
	}
	if r.Sources.Decision != "D-0006" {
		t.Errorf("global rule Sources.Decision = %q, want D-0006", r.Sources.Decision)
	}
	var sawScopeReach bool
	for _, p := range r.Preconditions {
		if p.Subject == "scope-reach" {
			sawScopeReach = true
			if p.Op != "==" || p.Value != "false" {
				t.Errorf("scope-reach precondition = {Op:%q Value:%q}, want {== false} (the out-of-scope violation)", p.Op, p.Value)
			}
		}
	}
	if !sawScopeReach {
		t.Error("global rule missing the scope-reach precondition")
	}
}

// TestM0147_AC2_CodeIsLegality asserts provenance-authorization-out-of-scope
// is classified ClassLegality by the same scanner the AC-5 fourth arm
// uses, and that the fourth-arm invariant holds for it (a legality code
// must be named by ≥1 illegal spec rule).
func TestM0147_AC2_CodeIsLegality(t *testing.T) {
	t.Parallel()
	implCodes, err := collectImplFindingCodes(repoRoot(t))
	if err != nil {
		t.Fatalf("collectImplFindingCodes: %v", err)
	}
	const code = "provenance-authorization-out-of-scope"
	class, ok := implCodes[code]
	if !ok {
		t.Fatalf("%s not discovered by collectImplFindingCodes", code)
	}
	if class != codes.ClassLegality {
		t.Errorf("%s class = %v, want ClassLegality", code, class)
	}
	if !specIllegalErrorCodes()[code] {
		t.Errorf("%s is ClassLegality but named by no illegal spec rule — AC-5 fourth arm would fail", code)
	}
}

// TestM0147_AC3_GlobalRuleExercised is the spec↔runtime tie: the global
// rule's ExpectedErrorCode (read from spec.Rules()) is exactly the code
// the runtime refuses an out-of-scope agent with, and an in-scope agent
// succeeds — exercised through the M-0146 authorized-scope machinery
// (full integration; the per-cell m0124/m0125 drivers skip the global
// rule, which has no cell coordinate).
func TestM0147_AC3_GlobalRuleExercised(t *testing.T) {
	t.Parallel()
	code := globalScopeReachRule(t).ExpectedErrorCode

	f := cellcoverage.NewCellFixture(t)
	ctx := context.Background()
	const human = "human/test"
	f.Must(verb.Add(ctx, f.Tree(), entity.KindEpic, "In-scope Epic", human, verb.AddOptions{}))
	f.Must(verb.Promote(ctx, f.Tree(), "E-0001", entity.StatusActive, human, "", false, verb.PromoteOptions{}))
	f.Must(verb.Add(ctx, f.Tree(), entity.KindMilestone, "In-scope MS", human, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	f.Must(verb.Add(ctx, f.Tree(), entity.KindEpic, "Out-of-scope Epic", human, verb.AddOptions{}))
	f.Must(verb.Promote(ctx, f.Tree(), "E-0002", entity.StatusActive, human, "", false, verb.PromoteOptions{}))
	f.Must(verb.Add(ctx, f.Tree(), entity.KindMilestone, "Out-of-scope MS", human, verb.AddOptions{EpicID: "E-0002", TDD: "none"}))
	f.AuthorizeScope(t, "E-0001", "ai/claude")

	agentArgs := func(target string) []string {
		return []string{"promote", target, "in_progress", "--actor", "ai/claude", "--principal", human}
	}

	// Positive: in-scope agent promote succeeds.
	if out, err := testutil.RunBin(t, f.Root, "", nil, agentArgs("M-0001")...); err != nil {
		t.Fatalf("in-scope agent promote should succeed: %v\n%s", err, out)
	}

	// Negative: out-of-scope agent promote refused with the global rule's code.
	headBefore, err := testutil.RunGit(f.Root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	out, runErr := testutil.RunBin(t, f.Root, "", nil, agentArgs("M-0002")...)
	if runErr == nil {
		t.Fatalf("out-of-scope agent promote should be refused:\n%s", out)
	}
	if !strings.Contains(out, code) {
		t.Errorf("refusal should cite the global rule's ExpectedErrorCode %q; got:\n%s", code, out)
	}
	headAfter, err := testutil.RunGit(f.Root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	if headBefore != headAfter {
		t.Errorf("refused verb must not commit: HEAD moved %s → %s", headBefore, headAfter)
	}
}
