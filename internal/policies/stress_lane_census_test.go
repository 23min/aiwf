package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPolicy_StressLaneCensus is the live-repo chokepoint: no test in
// internal/stresstest may carry the `stress` build tag without doing
// work that needs the on-demand lane. Zero violations expected. See
// G-0468.
func TestPolicy_StressLaneCensus(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyStressLaneCensus)
}

// TestPolicyStressLaneCensus_Branches drives the policy over synthetic
// trees so every arm is exercised: the tagged/untagged split, each
// kind of slow-work marker, the declarations that are skipped rather
// than judged, and the two error paths. The negative cases matter most
// — a marker list that quietly matches everything would leave the
// policy green and useless, so each marker gets a case proving it is
// the thing that excused the test.
func TestPolicyStressLaneCensus_Branches(t *testing.T) {
	t.Parallel()

	const hermetic = `//go:build stress

package stresstest

func TestDecision(t *testing.T) {
	if got := patchExactlyOnce("a", "b", "c"); got != "" {
		t.Fatal("nope")
	}
}
`
	const untagged = `package stresstest

func TestDecision(t *testing.T) {
	_ = patchExactlyOnce("a", "b", "c")
}
`
	const viaIdent = `//go:build stress

package stresstest

func TestDrivesAScenario(t *testing.T) {
	bin := sharedTestBinary(t)
	_ = bin
}
`
	const viaSetup = `//go:build stress

package stresstest

func TestDrivesSetup(t *testing.T) {
	s := NewSomeScenario("bin")
	if err := s.Setup(t.TempDir()); err != nil {
		t.Fatal(err)
	}
}
`
	const viaExec = `//go:build stress

package stresstest

func TestExecs(t *testing.T) {
	_ = exec.Command("git", "status")
}
`
	const viaSleep = `//go:build stress

package stresstest

func TestWaits(t *testing.T) {
	time.Sleep(time.Second)
}
`
	const viaGoroutine = `//go:build stress

package stresstest

func TestRaces(t *testing.T) {
	go func() { close(done) }()
	<-done
}
`
	// The harness's own runner, driven with a fake scenario: hermetic,
	// and the shape most of this package's untagged tests take. The
	// runner name alone must not excuse it.
	const fakeScenarioRunner = `//go:build stress

package stresstest

func TestRunScenarioCleansUp(t *testing.T) {
	res, err := RunScenario(&fakeScenario{}, t.TempDir())
	if err != nil || !res.Passed {
		t.Fatal("nope")
	}
}
`
	// The same runner handed a real scenario: that constructor is what
	// turns the call into real work.
	const realScenarioRunner = `//go:build stress

package stresstest

func TestRunScenarioForReal(t *testing.T) {
	res, err := RunScenario(NewLockKillScenario("bin"), t.TempDir())
	if err != nil || !res.Passed {
		t.Fatal("nope")
	}
}
`
	// Setup on a fake, with no constructor in sight: the same
	// conditional shape as the runners.
	const fakeSetup = `//go:build stress

package stresstest

func TestFakeSetupErrorIsPropagated(t *testing.T) {
	f := &fakeScenario{setupErr: errors.New("boom")}
	if err := f.Setup(t.TempDir()); err == nil {
		t.Fatal("want error")
	}
}
`
	// A file pinned to the default lane is the opposite of this
	// policy's subject and must not be judged as if it were tagged.
	const negatedConstraint = `//go:build !stress

package stresstest

func TestHermeticOnlyInTheDefaultLane(t *testing.T) {
	_ = 1
}
`
	// A build-constraint-looking comment inside a function body does
	// not tag the file.
	const constraintInBody = `package stresstest

func TestHermetic(t *testing.T) {
	//go:build stress
	_ = 1
}
`
	// TestMain drives the binary rather than asserting anything, and
	// takes a *testing.M, so it is not a test to judge.
	const testMainOnly = `//go:build stress

package stresstest

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
`
	// Test-prefixed declarations that the runner would not call: one
	// taking no parameter, one taking a value rather than a pointer.
	// Neither is a test, so neither is judged.
	const testShapedNonTests = `//go:build stress

package stresstest

func TestNoParams() {
	_ = 1
}

func TestValueParam(t testing.T) {
	_ = 1
}
`
	// Prose ahead of the package clause: not a build constraint, so
	// the constraint parser rejects it and the file reads as untagged.
	const proseBeforePackage = `// Package stresstest drives the harness.

package stresstest

func TestHermetic(t *testing.T) {
	_ = 1
}
`
	// A nested selector (pkg.thing.Method) whose receiver is itself a
	// selector rather than a plain identifier: the walk must step past
	// it rather than treat it as a package-qualified call.
	const nestedSelector = `//go:build stress

package stresstest

func TestNestedSelector(t *testing.T) {
	_ = outer.inner.Method()
}
`
	// A table-driven hermetic test: t.Run must not excuse it, which is
	// why "Run" is absent from the marker list even though "Setup" is
	// present. Also carries a non-matching selector (strings.Count) and
	// a bare call so the walk traverses its non-matching arms.
	const subtestsDoNotExcuse = `//go:build stress

package stresstest

func TestTableDriven(t *testing.T) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_ = strings.Count(tc.in, "x")
			doThing()
		})
	}
}
`
	// Only non-Test declarations and a method: nothing to judge.
	const noTests = `//go:build stress

package stresstest

func helperOnly(t *testing.T) {
	_ = 1
}

func (s *SomeScenario) Verify(dir string) []Violation {
	return nil
}
`
	const twoHermetic = `//go:build stress

package stresstest

func TestOne(t *testing.T) {
	_ = 1
}

func TestTwo(t *testing.T) {
	_ = 2
}
`
	const unparseable = `//go:build stress

package stresstest

func TestBroken(t *testing.T) {
`

	cases := []struct {
		name      string
		files     map[string]string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "a tagged hermetic test is flagged",
			files:     map[string]string{"internal/stresstest/decision_test.go": hermetic},
			wantCount: 1,
		},
		{
			name:      "the same test untagged is not the policy's business",
			files:     map[string]string{"internal/stresstest/decision_test.go": untagged},
			wantCount: 0,
		},
		{
			name:      "a package-local binary helper excuses the tag",
			files:     map[string]string{"internal/stresstest/driver_test.go": viaIdent},
			wantCount: 0,
		},
		{
			name:      "driving a scenario's Setup excuses the tag",
			files:     map[string]string{"internal/stresstest/driver_test.go": viaSetup},
			wantCount: 0,
		},
		{
			name:      "exec.Command excuses the tag",
			files:     map[string]string{"internal/stresstest/driver_test.go": viaExec},
			wantCount: 0,
		},
		{
			name:      "time.Sleep excuses the tag",
			files:     map[string]string{"internal/stresstest/driver_test.go": viaSleep},
			wantCount: 0,
		},
		{
			name:      "a goroutine excuses the tag",
			files:     map[string]string{"internal/stresstest/driver_test.go": viaGoroutine},
			wantCount: 0,
		},
		{
			name:      "t.Run does not excuse a table-driven decision test",
			files:     map[string]string{"internal/stresstest/table_test.go": subtestsDoNotExcuse},
			wantCount: 1,
		},
		{
			name:      "helpers and methods in a tagged file are not judged",
			files:     map[string]string{"internal/stresstest/helpers_test.go": noTests},
			wantCount: 0,
		},
		{
			name:      "every stranded test in a file is reported, not just the first",
			files:     map[string]string{"internal/stresstest/two_test.go": twoHermetic},
			wantCount: 2,
		},
		{
			name: "non-test sources and other packages are ignored",
			files: map[string]string{
				"internal/stresstest/scenario.go": "package stresstest\n\nfunc Thing() {}\n",
				"internal/other/other_test.go":    hermetic,
			},
			wantCount: 0,
		},
		{
			name:      "a receiver that is itself a selector is stepped past, not matched",
			files:     map[string]string{"internal/stresstest/nested_test.go": nestedSelector},
			wantCount: 1,
		},
		{
			name:      "the harness runner driven with a fake scenario does not excuse the tag",
			files:     map[string]string{"internal/stresstest/runner_test.go": fakeScenarioRunner},
			wantCount: 1,
		},
		{
			name:      "the same runner handed a real scenario does excuse it",
			files:     map[string]string{"internal/stresstest/runner_test.go": realScenarioRunner},
			wantCount: 0,
		},
		{
			name:      "Setup on a fake scenario does not excuse the tag",
			files:     map[string]string{"internal/stresstest/fake_test.go": fakeSetup},
			wantCount: 1,
		},
		{
			name:      "a file pinned out of the stress lane is not judged",
			files:     map[string]string{"internal/stresstest/negated_test.go": negatedConstraint},
			wantCount: 0,
		},
		{
			name:      "a constraint-shaped comment inside a body does not tag the file",
			files:     map[string]string{"internal/stresstest/inbody_test.go": constraintInBody},
			wantCount: 0,
		},
		{
			name:      "TestMain is not a test to judge",
			files:     map[string]string{"internal/stresstest/setup_test.go": testMainOnly},
			wantCount: 0,
		},
		{
			name:      "cmd/stresstest is in scope too",
			files:     map[string]string{"cmd/stresstest/decision_test.go": hermetic},
			wantCount: 1,
		},
		{
			name:      "Test-prefixed declarations the runner would not call are not judged",
			files:     map[string]string{"internal/stresstest/shapes_test.go": testShapedNonTests},
			wantCount: 0,
		},
		{
			name:      "prose ahead of the package clause is not a build constraint",
			files:     map[string]string{"internal/stresstest/prose_test.go": proseBeforePackage},
			wantCount: 0,
		},
		{
			name:    "unparseable source surfaces as an error",
			files:   map[string]string{"internal/stresstest/broken_test.go": unparseable},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for rel, content := range tc.files {
				writeGoFixture(t, root, rel, content)
			}
			vs, err := PolicyStressLaneCensus(root)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got nil (violations: %+v)", vs)
				}
				return
			}
			if err != nil {
				t.Fatalf("PolicyStressLaneCensus: %v", err)
			}
			if len(vs) != tc.wantCount {
				t.Fatalf("got %d violations, want %d: %+v", len(vs), tc.wantCount, vs)
			}
			for _, v := range vs {
				if v.Policy != "stress-lane-census" {
					t.Errorf("violation Policy = %q, want %q", v.Policy, "stress-lane-census")
				}
				if v.Line == 0 {
					t.Errorf("violation carries no line number: %+v", v)
				}
			}
		})
	}
}

// TestPolicyStressLaneCensus_UnreadablePackageSurfacesAsAnError pins
// the read-failure arm, distinct from the absent-package arm below: a
// path that exists but cannot be listed is a broken tree rather than a
// consumer repo without the package, and must not be reported as zero
// violations.
func TestPolicyStressLaneCensus_UnreadablePackageSurfacesAsAnError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// A regular file where the package directory belongs: readable,
	// present, and not a directory.
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatalf("mkdir internal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "stresstest"), []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("write stresstest as a file: %v", err)
	}

	if _, err := PolicyStressLaneCensus(root); err == nil {
		t.Fatal("expected an error when internal/stresstest cannot be listed")
	}
}

// TestPolicyStressLaneCensus_AbsentPackageIsNotAViolation pins the
// consumer case: a tree with no internal/stresstest at all reports
// nothing rather than erroring, since the policy travels with the repo
// that has the package.
func TestPolicyStressLaneCensus_AbsentPackageIsNotAViolation(t *testing.T) {
	t.Parallel()
	vs, err := PolicyStressLaneCensus(t.TempDir())
	if err != nil {
		t.Fatalf("PolicyStressLaneCensus on a tree with no stresstest package: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("got %d violations, want 0: %+v", len(vs), vs)
	}
}
