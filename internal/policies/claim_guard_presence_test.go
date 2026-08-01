package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// M-0285. PolicyNoOpClaimScope records what each converging verb's claim
// is about; this policy asserts the verbs whose recorded scope names a
// file actually consult it before converging.

// TestPolicy_ClaimGuardPresence is the live assertion over this repo.
func TestPolicy_ClaimGuardPresence(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyClaimGuardPresence)
}

// calls is the map literal shorthand the fixtures below read better with.
func calls(names ...string) map[string]bool {
	out := map[string]bool{}
	for _, n := range names {
		out[n] = true
	}
	return out
}

// scopedFuncs is the fixture shape the helper tests share: the two guard
// wrappers over the shared comparison, exactly as internal/verb declares
// them, plus one verb whose call set the case under test supplies.
func scopedFuncs(verbCalls ...string) map[string]verbFunc {
	return map[string]verbFunc{
		"SomeVerb": {File: "internal/verb/someverb.go", Line: 42, Calls: calls(verbCalls...)},
		"guardClaim": {
			File: "internal/verb/claimguard.go", Line: 100,
			Calls: calls("guardClaimPaths"),
		},
		"guardClaimConfig": {
			File: "internal/verb/claimguard.go", Line: 110,
			Calls: calls("guardClaimPaths"),
		},
	}
}

// TestCheckClaimGuardPresence_ScopedRowThatNeverReachesTheGuardFires is
// AC-1's firing case, and the hole this policy exists to close: the
// sibling ledger records names, so a row can claim `target-entity` while
// the verb consults nothing. Both surfaces compile and both read as
// covered.
func TestCheckClaimGuardPresence_ScopedRowThatNeverReachesTheGuardFires(t *testing.T) {
	t.Parallel()
	vs := checkClaimGuardPresence(
		[]claimScope{{Func: "SomeVerb", Scope: claimScopeTargetEntity, Reason: "converges on the target's stored status"}},
		scopedFuncs(),
	)
	assertDetail(t, vs, "SomeVerb records claim scope")
	assertDetail(t, vs, "never reaches")
}

// TestCheckClaimGuardPresence_ConfigScopedRowThatNeverReachesTheGuardFires
// covers the other guarded scope. The three aiwf.yaml claims read a file
// no entity owns, which is a different guard call and the same
// requirement.
func TestCheckClaimGuardPresence_ConfigScopedRowThatNeverReachesTheGuardFires(t *testing.T) {
	t.Parallel()
	vs := checkClaimGuardPresence(
		[]claimScope{{Func: "SomeVerb", Scope: claimScopeConfigFile, Reason: "converges on the binding recorded in aiwf.yaml"}},
		scopedFuncs(),
	)
	assertDetail(t, vs, "SomeVerb records claim scope")
}

// TestCheckClaimGuardPresence_ReachingTheGuardThroughAHelperPasses pins
// what the check is about: reaching the comparison, not calling it from
// the entry point. A verb that factors its prelude into a helper has not
// bypassed the seam, and a check that said otherwise would push back on
// ordinary refactoring rather than on the defect.
func TestCheckClaimGuardPresence_ReachingTheGuardThroughAHelperPasses(t *testing.T) {
	t.Parallel()
	funcs := scopedFuncs("prelude")
	funcs["prelude"] = verbFunc{
		File: "internal/verb/someverb.go", Line: 80,
		Calls: calls("guardClaim"),
	}
	vs := checkClaimGuardPresence(
		[]claimScope{{Func: "SomeVerb", Scope: claimScopeTargetEntity, Reason: "converges on the target's stored status"}},
		funcs,
	)
	if len(vs) != 0 {
		t.Errorf("a verb reaching the guard through a helper should pass; got:\n%s", joinDetails(vs))
	}
}

// TestCheckClaimGuardPresence_ExemptRowsCarryNoPresenceRequirement is the
// negative control that keeps the presence check from collapsing into
// "every row calls the guard" — which would refuse the two scopes whose
// whole content is that no per-verb comparison is wired.
func TestCheckClaimGuardPresence_ExemptRowsCarryNoPresenceRequirement(t *testing.T) {
	t.Parallel()
	vs := checkClaimGuardPresence(
		[]claimScope{
			{Func: "SomeVerb", Scope: claimScopeNone, Reason: "the converging path writes nothing"},
			{Func: "OtherVerb", Scope: claimScopeSweepDeciders, Reason: "compares per candidate inside its own planner"},
		},
		scopedFuncs(),
	)
	if len(vs) != 0 {
		t.Errorf("an exempt row should carry no presence requirement; got:\n%s", joinDetails(vs))
	}
}

// TestCheckClaimGuardPresence_RowTheScanCannotSeeFires closes the silent
// arm. A guarded row whose function the scan does not find is either
// stale or a shape outside the scan — a converging method is the live
// example — and skipping it would let the policy report green over a row
// it never checked. The bound costs a finding rather than a gap.
func TestCheckClaimGuardPresence_RowTheScanCannotSeeFires(t *testing.T) {
	t.Parallel()
	vs := checkClaimGuardPresence(
		[]claimScope{{Func: "LongGone", Scope: claimScopeTargetEntity, Reason: "converges on the target's stored status"}},
		scopedFuncs(),
	)
	assertDetail(t, vs, "no package-level function by that name")
}

// TestCheckDormantClaimExemptions_ExemptRowThatReachesTheGuardFires is
// AC-2's dormancy case. An exemption whose premise has stopped holding
// still reads as a reviewed decision, which is the shape that outlives
// the condition it exempts.
func TestCheckDormantClaimExemptions_ExemptRowThatReachesTheGuardFires(t *testing.T) {
	t.Parallel()
	vs := checkDormantClaimExemptions(
		[]claimScope{{Func: "SomeVerb", Scope: claimScopeNone, Reason: "the converging path writes nothing"}},
		scopedFuncs("guardClaim"),
	)
	assertDetail(t, vs, "SomeVerb is recorded exempt")
}

// TestCheckDormantClaimExemptions_LiveExemptionPasses is the state the
// four recorded exemptions are in, held as a spec rather than inferred
// from the live run: an exempt route that wires nothing draws no finding,
// whether the scan finds its function or not.
func TestCheckDormantClaimExemptions_LiveExemptionPasses(t *testing.T) {
	t.Parallel()
	vs := checkDormantClaimExemptions(
		[]claimScope{
			{Func: "SomeVerb", Scope: claimScopeNone, Reason: "the converging path writes nothing"},
			{Func: "LongGone", Scope: claimScopeSweepDeciders, Reason: "compares per candidate inside its own planner"},
		},
		scopedFuncs(),
	)
	if len(vs) != 0 {
		t.Errorf("an exemption with nothing wired is not dormant; got:\n%s", joinDetails(vs))
	}
}

// TestCheckDormantClaimExemptions_GuardedRowsAreNotDormant is the control
// that stops the dormancy check from firing on every guarded verb — they
// all reach the guard, which is the point.
func TestCheckDormantClaimExemptions_GuardedRowsAreNotDormant(t *testing.T) {
	t.Parallel()
	vs := checkDormantClaimExemptions(
		[]claimScope{{Func: "SomeVerb", Scope: claimScopeTargetEntity, Reason: "converges on the target's stored status"}},
		scopedFuncs("guardClaim"),
	)
	if len(vs) != 0 {
		t.Errorf("a guarded row is not a dormant exemption; got:\n%s", joinDetails(vs))
	}
}

// TestValidateExemptClaimReasons_SharedReasonFires is AC-2's
// one-entry-per-route half. Two exempt routes carrying the same sentence
// is a category label wearing two names, and it hides that only one of
// the two was ever reasoned about.
func TestValidateExemptClaimReasons_SharedReasonFires(t *testing.T) {
	t.Parallel()
	vs := validateExemptClaimReasons([]claimScope{
		{Func: "SomeVerb", Scope: claimScopeNone, Reason: "nothing can contradict the claim"},
		{Func: "OtherVerb", Scope: claimScopeNone, Reason: "nothing can contradict the claim"},
	})
	assertDetail(t, vs, "record the same exemption reason")
}

// TestValidateExemptClaimReasons_ReasonThatIsJustTheRouteNameFires covers
// the other half of AC-2's "not a bare route name". An exemption with no
// guard to inspect is only as good as its sentence.
func TestValidateExemptClaimReasons_ReasonThatIsJustTheRouteNameFires(t *testing.T) {
	t.Parallel()
	vs := validateExemptClaimReasons([]claimScope{
		{Func: "SomeVerb", Scope: claimScopeNone, Reason: "  SomeVerb "},
		{Func: "OtherVerb", Scope: claimScopeNone, Reason: "otherverb"},
	})
	assertDetail(t, vs, "restates its own name")
	// Case is not the distinction — a reason differing from the route
	// name only in capitalization restates it just as completely.
	assertDetail(t, vs, "OtherVerb records an exemption reason")
}

// TestValidateExemptClaimReasons_DistinctReasonsPass is the negative
// control, and it also pins that guarded rows are outside this check —
// they may share wording, because the guard call is what a reader checks
// them against.
func TestValidateExemptClaimReasons_DistinctReasonsPass(t *testing.T) {
	t.Parallel()
	vs := validateExemptClaimReasons([]claimScope{
		{Func: "SomeVerb", Scope: claimScopeNone, Reason: "the converging path writes nothing"},
		{Func: "OtherVerb", Scope: claimScopeSweepDeciders, Reason: "compares per candidate inside its own planner"},
		{Func: "Guarded", Scope: claimScopeTargetEntity, Reason: "converges on the target's stored status"},
		{Func: "AlsoGuarded", Scope: claimScopeTargetEntity, Reason: "converges on the target's stored status"},
	})
	if len(vs) != 0 {
		t.Errorf("distinct exempt reasons should pass; got:\n%s", joinDetails(vs))
	}
}

// TestPolicyClaimGuardPresence_EmptyScanFailsClosed pins the fail-closed
// arm. A tree with no verb package must not read as "every scoped row
// consults its guard" — it means the policy is scanning nothing.
func TestPolicyClaimGuardPresence_EmptyScanFailsClosed(t *testing.T) {
	t.Parallel()
	vs, err := PolicyClaimGuardPresence(t.TempDir())
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 1 || !strings.Contains(vs[0].Detail, "scanning nothing") {
		t.Errorf("expected a single fail-closed violation, got:\n%s", joinDetails(vs))
	}
}

// assertDetail fails unless some violation's Detail contains want.
func assertDetail(t *testing.T, vs []Violation, want string) {
	t.Helper()
	for _, v := range vs {
		if strings.Contains(v.Detail, want) {
			return
		}
	}
	t.Errorf("no violation contains %q; got:\n%s", want, joinDetails(vs))
}

// --- Seam tests -------------------------------------------------------
//
// The helper tests above hand-build their inputs, so every one of them
// passes against a policy whose scan output goes nowhere. These drive the
// exported entry point against a real tree, which is the only thing that
// distinguishes a wired policy from a disconnected one.

// guardChainSrc is the fixture preamble: the two wrappers over the shared
// comparison, declared as internal/verb declares them, so a fixture verb
// can reach a guard and the anchor check finds both names.
const guardChainSrc = `package verb

type Result struct{ NoOp bool }

func guardClaimPaths(root, subject string, exempt bool, paths []string) error { return nil }
func guardClaim(root, subject string, paths ...string) error {
	return guardClaimPaths(root, subject, false, paths)
}
func guardClaimConfig(root, subject string, paths ...string) error {
	return guardClaimPaths(root, subject, true, paths)
}
`

// TestPolicyClaimGuardPresence_RealBypassFiresThroughTheExportedPolicy is
// the regression AC-1 exists to fail on, driven end to end: SetPriority
// is a live target-entity row, and here it converges with no guard.
func TestPolicyClaimGuardPresence_RealBypassFiresThroughTheExportedPolicy(t *testing.T) {
	t.Parallel()
	src := guardChainSrc + `
func SetPriority(id string) (*Result, error) {
	return &Result{NoOp: true}, nil
}
`
	vs, err := PolicyClaimGuardPresence(claimScopeFixture(t, src))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	assertDetail(t, vs, "SetPriority records claim scope")
	assertDetail(t, vs, "never reaches guardClaim")
}

// TestPolicyClaimGuardPresence_GuardBelowTheConvergenceFires pins the
// placement half. A guard positioned after the converging return is
// present on every call-graph edge and still never runs on the input it
// exists to refuse.
func TestPolicyClaimGuardPresence_GuardBelowTheConvergenceFires(t *testing.T) {
	t.Parallel()
	src := guardChainSrc + `
func SetPriority(id string) (*Result, error) {
	if id == "" {
		return &Result{NoOp: true}, nil
	}
	if err := guardClaim("root", id, "path"); err != nil {
		return nil, err
	}
	return nil, nil
}
`
	vs, err := PolicyClaimGuardPresence(claimScopeFixture(t, src))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	assertDetail(t, vs, "below the Result{NoOp: true} it protects")
}

// TestPolicyClaimGuardPresence_GuardAboveTheConvergencePasses is the
// control that keeps the placement check from firing on the arrangement
// every guarded verb actually uses.
func TestPolicyClaimGuardPresence_GuardAboveTheConvergencePasses(t *testing.T) {
	t.Parallel()
	src := guardChainSrc + `
func SetPriority(id string) (*Result, error) {
	if err := guardClaim("root", id, "path"); err != nil {
		return nil, err
	}
	if id == "" {
		return &Result{NoOp: true}, nil
	}
	return nil, nil
}
`
	vs, err := PolicyClaimGuardPresence(claimScopeFixture(t, src))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	for _, v := range vs {
		if strings.Contains(v.Detail, "SetPriority") {
			t.Errorf("a prelude guard drew a finding: %s", v.Detail)
		}
	}
}

// TestPolicyClaimGuardPresence_DormantExemptionFiresThroughTheExportedPolicy
// drives AC-2's dormancy half end to end. Rewidth is a live `none` row.
func TestPolicyClaimGuardPresence_DormantExemptionFiresThroughTheExportedPolicy(t *testing.T) {
	t.Parallel()
	src := guardChainSrc + `
func Rewidth(id string) (*Result, error) {
	return nil, guardClaim("root", id, "path")
}
`
	vs, err := PolicyClaimGuardPresence(claimScopeFixture(t, src))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	assertDetail(t, vs, "Rewidth is recorded exempt")
}

// TestClaimGuardPresence_ExemptReasonCheckIsWiredIntoTheComposition
// reaches the reason validator through the same composition the exported
// policy runs, which the real ledger cannot do because it is well-formed.
func TestClaimGuardPresence_ExemptReasonCheckIsWiredIntoTheComposition(t *testing.T) {
	t.Parallel()
	vs, err := claimGuardPresence(claimScopeFixture(t, guardChainSrc), []claimScope{
		{Func: "SomeVerb", Scope: claimScopeNone, Reason: "a shared label"},
		{Func: "OtherVerb", Scope: claimScopeNone, Reason: "a shared label"},
	})
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	assertDetail(t, vs, "record the same exemption reason")
}

// TestPolicyClaimGuardPresence_MissingAnchorFiresOnce pins that a renamed
// or inlined wrapper is reported as the anchor it is, rather than as a
// bypass by every verb that referenced it.
func TestPolicyClaimGuardPresence_MissingAnchorFiresOnce(t *testing.T) {
	t.Parallel()
	src := `package verb

type Result struct{ NoOp bool }

func Placeholder() {}
`
	vs, err := PolicyClaimGuardPresence(claimScopeFixture(t, src))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	assertDetail(t, vs, "guardClaim is not declared under internal/verb")
	assertDetail(t, vs, "guardClaimConfig is not declared under internal/verb")
}

// --- The shared call-graph walker -------------------------------------

// TestCalledIdents_RecordsNestedCallTargets pins that the collector
// descends through a call's arguments. Stopping at the outermost call
// would drop every guard passed or wrapped inside another expression, so
// a guarded verb would read as bypassing the seam.
func TestCalledIdents_RecordsNestedCallTargets(t *testing.T) {
	t.Parallel()
	src := `package verb

func outer() { alpha(beta(gamma())) }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fixture.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := calledIdents(f.Decls[0].(*ast.FuncDecl))
	for _, want := range []string{"alpha", "beta", "gamma"} {
		if !got[want] {
			t.Errorf("calledIdents did not record %q; got %v", want, got)
		}
	}
}

// TestCalledIdents_SkipsQualifiedAndMethodCalls pins the boundary that
// keeps the graph same-package. A selector reaches code this walk does
// not model, so counting it would report a reach that is not one.
func TestCalledIdents_SkipsQualifiedAndMethodCalls(t *testing.T) {
	t.Parallel()
	src := `package verb

func outer(e err) { gitops.Rename(); e.Paths() }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fixture.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := calledIdents(f.Decls[0].(*ast.FuncDecl))
	for _, unwanted := range []string{"Rename", "Paths", "gitops"} {
		if got[unwanted] {
			t.Errorf("calledIdents recorded %q from a selector call; got %v", unwanted, got)
		}
	}
}

// TestReachesCall covers the walk's arms, including the cycle. Without
// the visited set a mutual recursion in internal/verb would overflow the
// stack during CI rather than fail a test, and no such pair exists today
// to catch it.
func TestReachesCall(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		graph callGraph
		start string
		want  bool
	}{
		{"direct edge", callGraph{"A": {"target": true}}, "A", true},
		{"transitive chain", callGraph{"A": {"B": true}, "B": {"C": true}, "C": {"target": true}}, "A", true},
		{"no edge", callGraph{"A": {"B": true}, "B": {}}, "A", false},
		{"name absent from the graph", callGraph{"A": {"target": true}}, "Absent", false},
		{"cycle terminates", callGraph{"A": {"B": true}, "B": {"A": true}}, "A", false},
		{"cycle with the target beyond it", callGraph{"A": {"B": true}, "B": {"A": true, "target": true}}, "A", true},
		{"self recursion terminates", callGraph{"A": {"A": true}}, "A", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := reachesCall(tc.start, "target", tc.graph, map[string]bool{}); got != tc.want {
				t.Errorf("reachesCall = %v, want %v", got, tc.want)
			}
		})
	}
}
