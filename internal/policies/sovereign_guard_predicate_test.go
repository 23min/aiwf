package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-0293/AC-3. The sovereign-dispatcher policy asserts a live code
// reference.
//
// G-0534 measured the old predicate as satisfied by any occurrence of
// the actor-shape prefix in a function body, including one inside a
// flag-help string — and all four dispatchers passed on help text and
// nothing else. M-0293/AC-1 then made that strictly worse by requiring
// `human/` in the sovereign --force help, so the old predicate could no
// longer fail for any dispatcher that satisfied a different test.
//
// The predicate now looks for a call to the shared guard, found by
// walking call expressions rather than searching text, so no string can
// satisfy it.

// writePolicyTree materializes a synthetic source tree under a temp dir
// and returns its root.
func writePolicyTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	return root
}

// TestSovereignGuardPredicate_HelpStringDoesNotSatisfyIt is the
// regression pin for G-0534. This exact source shape — a dispatcher
// whose only mention of the actor prefix is a flag-help string — passed
// the old predicate, and is what made the policy unable to fail.
func TestSovereignGuardPredicate_HelpStringDoesNotSatisfyIt(t *testing.T) {
	t.Parallel()

	root := writePolicyTree(t, map[string]string{
		"internal/cli/sv/sv.go": `package sv

func NewCmd() {
	flags().StringVar(&principal, "principal", "", "the human/<id> the actor acts for")
	flags().BoolVar(&force, "force", false, "sovereign, so the actor must be human/...")
	flags().StringVar(&reason, "reason", "", "why")
}
`,
	})

	vs, err := PolicySovereignDispatchersGuardHumanActor(root)
	if err != nil {
		t.Fatalf("policy errored: %v", err)
	}
	if len(vs) == 0 {
		t.Fatal("a dispatcher whose only `human/` is flag-help text was reported as guarded. " +
			"That is G-0534 exactly: the policy reads as coverage while asserting nothing, and " +
			"every sovereign dispatcher passes it by documenting its flags")
	}
}

// TestSovereignGuardPredicate_GuardCallSatisfiesIt is the other
// direction. Without it the predicate could be unsatisfiable, which
// reports every dispatcher forever and is no more useful than reporting
// none.
func TestSovereignGuardPredicate_GuardCallSatisfiesIt(t *testing.T) {
	t.Parallel()

	root := writePolicyTree(t, map[string]string{
		"internal/cli/sv/sv.go": `package sv

func NewCmd() {
	flags().BoolVar(&force, "force", false, "override")
	flags().StringVar(&reason, "reason", "", "why")
}

func Run(opts Options) int {
	if code, ok := cliutil.RefuseNonHumanSovereignForce("aiwf sv", actorStr, force); !ok {
		return code
	}
	return 0
}
`,
	})

	vs, err := PolicySovereignDispatchersGuardHumanActor(root)
	if err != nil {
		t.Fatalf("policy errored: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("a dispatcher package calling the shared guard was still reported: %+v", vs)
	}
}

// TestSovereignGuardPredicate_GuardIsFoundAcrossThePackage pins the
// scope the predicate needs to have.
//
// The flags are declared in the command constructor, where no actor
// exists yet; the guard runs in the verb's Run function, after the
// prelude resolves one. A predicate scoped to the declaring function
// could never be satisfied by a correctly-placed guard, so the unit is
// the package.
func TestSovereignGuardPredicate_GuardIsFoundAcrossThePackage(t *testing.T) {
	t.Parallel()

	root := writePolicyTree(t, map[string]string{
		"internal/cli/sv/sv.go": `package sv

func NewCmd() {
	flags().BoolVar(&force, "force", false, "override")
	flags().StringVar(&reason, "reason", "", "why")
}
`,
		"internal/cli/sv/run.go": `package sv

func Run(opts Options) int {
	if code, ok := cliutil.RefuseNonHumanSovereignForce("aiwf sv", actorStr, force); !ok {
		return code
	}
	return 0
}
`,
	})

	vs, err := PolicySovereignDispatchersGuardHumanActor(root)
	if err != nil {
		t.Fatalf("policy errored: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("the guard was in a sibling file of the same package and the dispatcher was "+
			"still reported: %+v", vs)
	}
}

// TestSovereignGuardPredicate_ExemptDispatchersAreNotReported pins the
// exemptions and the shape of their justification.
//
// Two dispatchers correctly carry no guard at this layer, for different
// measured reasons: `add`'s --force is conditional, so a flag-keyed
// pre-check would refuse invocations the kernel permits; `authorize`
// already refuses every non-human actor, so a pre-check preempts a
// stronger refusal with a weaker message. Each entry names the test
// holding its reason true — an exemption whose premise nothing pins is
// indistinguishable from a dispatcher someone forgot.
func TestSovereignGuardPredicate_ExemptDispatchersAreNotReported(t *testing.T) {
	t.Parallel()

	if len(dispatchersGuardedElsewhere) == 0 {
		t.Fatal("the exemption set is empty, so this test asserts nothing")
	}

	root := writePolicyTree(t, map[string]string{
		"internal/cli/add/add.go": `package add

func NewCmd() {
	flags().BoolVar(&force, "force", false, "bypass the empty-body gate")
	flags().StringVar(&reason, "reason", "", "why")
}
`,
		"internal/cli/authorize/authorize.go": `package authorize

func NewCmd() {
	flags().StringVar(&to, "to", "", "agent")
	flags().BoolVar(&force, "force", false, "reopen a terminal scope-entity")
	flags().StringVar(&reason, "reason", "", "why")
}
`,
	})

	vs, err := PolicySovereignDispatchersGuardHumanActor(root)
	if err != nil {
		t.Fatalf("policy errored: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("an exempt dispatcher was reported: %+v", vs)
	}

	for _, e := range dispatchersGuardedElsewhere {
		// The exemption must stay narrow. A prefix that swallowed the
		// whole dispatcher layer would silence the policy without
		// deleting it.
		if e.Prefix == sovereignDispatcherPrefix || !strings.HasPrefix(e.Prefix, sovereignDispatcherPrefix) {
			t.Errorf("exemption %q is not a proper subdirectory of the scanned prefix %q; it "+
				"exempts more than one dispatcher package", e.Prefix, sovereignDispatcherPrefix)
		}
		if strings.TrimSpace(e.Why) == "" {
			t.Errorf("exemption %q carries no reason", e.Prefix)
		}
		if strings.TrimSpace(e.PinnedBy) == "" {
			t.Errorf("exemption %q names no test holding its reason true; without one it is "+
				"indistinguishable from a dispatcher whose guard was forgotten", e.Prefix)
		}
	}
}

// TestSovereignGuardPredicate_UnqualifiedGuardCallSatisfiesIt covers
// the call shape a guard inside cliutil itself would take.
//
// internal/cli/cliutil is under the scanned prefix, so a dispatcher
// helper living there would call the guard by bare identifier rather
// than through a package selector. Nothing does today; the arm exists
// because the scan reaches the package that owns the guard.
func TestSovereignGuardPredicate_UnqualifiedGuardCallSatisfiesIt(t *testing.T) {
	t.Parallel()

	root := writePolicyTree(t, map[string]string{
		"internal/cli/sv/sv.go": `package sv

func NewCmd() {
	flags().BoolVar(&force, "force", false, "override")
	flags().StringVar(&reason, "reason", "", "why")
}

func Run() int {
	if code, ok := RefuseNonHumanSovereignForce("aiwf sv", actorStr, force); !ok {
		return code
	}
	return 0
}
`,
	})

	vs, err := PolicySovereignDispatchersGuardHumanActor(root)
	if err != nil {
		t.Fatalf("policy errored: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("an unqualified guard call was not recognized: %+v", vs)
	}
}

// TestGoPackageDir covers the helper's two shapes, including the
// slash-less path the scan never produces — WalkGoFiles returns
// repo-relative paths under internal/cli/, so every one has a
// separator. Returning "" rather than the path itself keeps a
// hypothetical root-level file from being read as its own package.
func TestGoPackageDir(t *testing.T) {
	t.Parallel()

	cases := []struct{ in, want string }{
		{"internal/cli/promote/promote.go", "internal/cli/promote/"},
		{"main.go", ""},
	}
	for _, tc := range cases {
		if got := goPackageDir(tc.in); got != tc.want {
			t.Errorf("goPackageDir(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestSovereignDispatchers_LiveTreeIsFullyGuarded is the assertion
// against the real repository. The synthetic cases above prove the
// predicate discriminates; this one proves the tree satisfies it.
func TestSovereignDispatchers_LiveTreeIsFullyGuarded(t *testing.T) {
	t.Parallel()

	vs, err := PolicySovereignDispatchersGuardHumanActor(repoRoot(t))
	if err != nil {
		t.Fatalf("policy errored: %v", err)
	}
	for _, v := range vs {
		t.Errorf("%s:%d — %s", v.File, v.Line, v.Detail)
	}
}
