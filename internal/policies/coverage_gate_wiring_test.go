package policies

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// coverage_gate_wiring_test.go — G-0469 chokepoint pins.
//
// `make ci` is the gate every wrap ritual mandates before a merge or a
// push, and it is where the report of an untested changed statement has
// to arrive — anywhere later and the change has already crossed the
// trunk boundary.
//
// These tests pin the properties that keep it there: `make ci` resolves
// to a recipe running the diff-scoped gate against the profile test-cov
// builds and builds that profile once; `.NOTPARALLEL:` keeps that
// ordering real under -j; and coverage-gate-only honors a
// caller-supplied base, reports each base it cannot use, and refuses
// outright rather than gating on a profile that is not there.
//
// They resolve the recipes through make itself rather than grepping the
// Makefile, so a rename of the intermediate target keeps them green and
// only the gate genuinely dropping out turns them red.

// makeDryRun returns the fully-expanded recipe make would run for target,
// without running it. `make -n` substitutes every variable, so the result
// is executable shell — which is what lets a test run the recipe rather
// than pattern-match the Makefile.
//
// The output has to be exactly the recipe and nothing else. These tests
// themselves run under `make ci`, so MAKELEVEL and MAKEFLAGS arrive in the
// environment and turn this into a sub-make: GNU make then wraps the
// recipe in `make[1]: Entering directory` / `Leaving directory` lines,
// and a caller feeding that to a shell executes them. Dropping the two
// variables and passing --no-print-directory makes the result identical
// however the suite was invoked.
func makeDryRun(t *testing.T, root, target string) string {
	t.Helper()
	cmd := exec.Command("make", "--no-print-directory", "-n", target)
	cmd.Dir = root
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "MAKELEVEL=") || strings.HasPrefix(kv, "MAKEFLAGS=") {
			continue
		}
		cmd.Env = append(cmd.Env, kv)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n %s: %v\n%s", target, err, out)
	}
	return string(out)
}

// TestCoverageGateWiring_CIRunsTheDiffScopedGate pins the composition
// claim every wrap ritual makes on `make ci`'s behalf — "the same checks
// CI runs on push, not a subset". The gate is the check that was missing
// from that subset.
func TestCoverageGateWiring_CIRunsTheDiffScopedGate(t *testing.T) {
	t.Parallel()
	recipe := makeDryRun(t, repoRoot(t), "ci")

	profileAt := strings.Index(recipe, "-coverprofile=coverage.out")
	// The gate runs as a -run alternation, so BranchCoverageAudit is the
	// contiguous literal; AIWF_COVERAGE_PROFILE is what points it at the
	// profile built above.
	gateAt := strings.Index(recipe, "BranchCoverageAudit")
	switch {
	case profileAt < 0:
		t.Fatalf("`make ci` must build a coverage profile:\n%s", recipe)
	case gateAt < 0:
		t.Fatalf("`make ci` must run the diff-scoped coverage gate:\n%s", recipe)
	case !strings.Contains(recipe, "AIWF_COVERAGE_PROFILE="):
		t.Fatalf("`make ci` must hand the gate a coverage profile:\n%s", recipe)
	case profileAt > gateAt:
		t.Errorf("`make ci` must build the profile before gating on it; gate at %d precedes profile at %d:\n%s",
			gateAt, profileAt, recipe)
	}
}

// TestCoverageGateWiring_CIBuildsTheProfileOnlyOnce pins the property
// that makes the gate affordable at this tier. A second instrumented run
// costs the full suite and is never served from the test cache, because
// every `go test` in the Makefile passes -exec, which is outside the
// cacheable flag set; gating off the profile test-cov already wrote is
// what keeps the marginal cost at seconds.
func TestCoverageGateWiring_CIBuildsTheProfileOnlyOnce(t *testing.T) {
	t.Parallel()
	recipe := makeDryRun(t, repoRoot(t), "ci")

	if n := strings.Count(recipe, "-coverprofile=coverage.out"); n != 1 {
		t.Errorf("`make ci` must build the coverage profile exactly once, got %d instrumented runs:\n%s", n, recipe)
	}
}

// baseResolutionScript extracts the half of coverage-gate-only that
// resolves the base ref and invokes the gate, so a test can run it
// directly against a stubbed `go`.
func baseResolutionScript(t *testing.T, root string) string {
	t.Helper()
	recipe := makeDryRun(t, root, "coverage-gate-only")
	// The extract is fed to a shell, so anything make added around the
	// recipe would be executed. Catch it here with a legible message
	// rather than as a bare exit 127 from sh.
	if strings.Contains(recipe, "make[") {
		t.Fatalf("dry run carries sub-make decoration; the extract is not executable:\n%s", recipe)
	}
	idx := strings.Index(recipe, "base=")
	if idx < 0 {
		t.Fatalf("coverage-gate-only must resolve a base ref:\n%s", recipe)
	}
	return recipe[idx:]
}

// stubbedGoEnv returns an environment whose `go` reports the coverage
// variables it was handed instead of running any test, and which carries
// no ambient AIWF_COVERAGE_* values. Scrubbing them matters because this
// package's own tests run under `make ci`, which sets AIWF_COVERAGE_BASE
// for the gate — inheriting it would mask the unset case entirely.
func stubbedGoEnv(t *testing.T, extra ...string) []string {
	t.Helper()
	stub := t.TempDir()
	script := "#!/bin/sh\necho \"BASE=$AIWF_COVERAGE_BASE\"\necho \"PROFILE=$AIWF_COVERAGE_PROFILE\"\n"
	if err := testsupport.WriteExecutable(filepath.Join(stub, "go"), []byte(script)); err != nil {
		t.Fatalf("writing go stub: %v", err)
	}

	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "AIWF_COVERAGE_") || strings.HasPrefix(kv, "PATH=") {
			continue
		}
		env = append(env, kv)
	}
	env = append(env, "PATH="+stub+string(os.PathListSeparator)+os.Getenv("PATH"))
	return append(env, extra...)
}

// baseFixture builds a throwaway repo the base-resolution recipe can be
// run against, so the assertions do not depend on the live repo's refs.
// origin/main is planted at the first commit; a second commit moves HEAD
// past it unless behindHEAD is false, which pins the degenerate case
// where a merge has already landed.
func baseFixture(t *testing.T, withOrigin, behindHEAD bool) (dir, originSHA string) {
	t.Helper()
	dir = t.TempDir()
	runGit := repoGitRunner(t, dir)
	writeFile := repoFileWriter(t, dir)

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "aiwf-test")
	writeFile("a.go", "package a\n")
	runGit("add", "-A")
	runGit("commit", "-m", "first")
	originSHA = trimLine(runGit("rev-parse", "HEAD"))

	if behindHEAD {
		writeFile("a.go", "package a\n\n// second\n")
		runGit("commit", "-am", "second")
	}
	if withOrigin {
		runGit("update-ref", "refs/remotes/origin/main", originSHA)
	} else {
		originSHA = ""
	}
	return dir, originSHA
}

// runBaseResolution executes the recipe's base-resolution half in dir
// with a stubbed `go`, returning its combined output.
func runBaseResolution(t *testing.T, dir string, env ...string) string {
	t.Helper()
	script := baseResolutionScript(t, repoRoot(t))
	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = dir
	cmd.Env = stubbedGoEnv(t, env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running the base-resolution recipe: %v\n%s", err, out)
	}
	return string(out)
}

// TestCoverageGateOnly_BaseResolution pins every arm of the recipe's base
// resolution against a synthetic repo, so the expected values are
// constructed rather than read off whatever state the live checkout
// happens to be in.
//
// A caller-supplied AIWF_COVERAGE_BASE must reach the gate unchanged: it
// is the only way to audit a range once the merge carrying it has landed
// and the default base has collapsed onto HEAD. The remaining arms each
// name a distinct outcome, because the gates no-op on some unusable bases
// and fail outright on others.
func TestCoverageGateOnly_BaseResolution(t *testing.T) {
	t.Parallel()

	t.Run("a caller-supplied base wins over the default", func(t *testing.T) {
		t.Parallel()
		dir, _ := baseFixture(t, true, true)
		out := runBaseResolution(t, dir, "AIWF_COVERAGE_BASE=stand-in-base-ref")
		if !strings.Contains(out, "BASE=stand-in-base-ref") {
			t.Errorf("a caller-set AIWF_COVERAGE_BASE must reach the gate unchanged; got:\n%s", out)
		}
	})

	t.Run("no caller value defaults to the merge-base with origin/main", func(t *testing.T) {
		t.Parallel()
		dir, originSHA := baseFixture(t, true, true)
		out := runBaseResolution(t, dir)
		if !strings.Contains(out, "BASE="+originSHA) {
			t.Errorf("default base must be the merge-base with origin/main (%s); got:\n%s", originSHA, out)
		}
	})

	t.Run("a base that collapsed onto HEAD says so", func(t *testing.T) {
		t.Parallel()
		dir, _ := baseFixture(t, true, false)
		out := runBaseResolution(t, dir)
		if !strings.Contains(out, "base resolves to HEAD") {
			t.Errorf("a landed range leaves nothing to diff; the recipe must say so rather than report green:\n%s", out)
		}
	})

	// A symbolic base has to be resolved before it is compared, or the
	// warning is skipped for the spelling an operator is most likely to
	// reach for by hand.
	t.Run("a symbolic base naming HEAD says so", func(t *testing.T) {
		t.Parallel()
		dir, _ := baseFixture(t, true, true)
		out := runBaseResolution(t, dir, "AIWF_COVERAGE_BASE=HEAD")
		if !strings.Contains(out, "base resolves to HEAD") {
			t.Errorf("AIWF_COVERAGE_BASE=HEAD leaves only uncommitted changes in scope; the recipe must say so:\n%s", out)
		}
	})

	t.Run("no origin/main says so", func(t *testing.T) {
		t.Parallel()
		dir, _ := baseFixture(t, false, true)
		out := runBaseResolution(t, dir)
		if !strings.Contains(out, "no comparison point") {
			t.Errorf("with no comparison point the recipe must report the no-op; got:\n%s", out)
		}
	})

	// Every diff-scoped policy treats the all-zero sha as "no base" and
	// no-ops. CI hands it exactly that value for a brand-new branch's
	// github.event.before, so it must not read as a real comparison point.
	t.Run("an all-zero base says so", func(t *testing.T) {
		t.Parallel()
		dir, _ := baseFixture(t, true, true)
		out := runBaseResolution(t, dir, "AIWF_COVERAGE_BASE="+strings.Repeat("0", 40))
		if !strings.Contains(out, "no comparison point") {
			t.Errorf("an all-zero base makes the gates no-op; the recipe must say so rather than report green:\n%s", out)
		}
	})

	// An unresolvable ref is the same class: nothing to diff against, so
	// it must not be passed through as though it named a commit.
	// A base naming no commit is a different outcome from the two above:
	// the gates do not no-op on it, they fail on git's bad-revision
	// error. Promising a no-op here would be a lie the operator finds out
	// about seconds later.
	t.Run("a base naming no commit is reported as a failure, not a no-op", func(t *testing.T) {
		t.Parallel()
		dir, _ := baseFixture(t, true, true)
		out := runBaseResolution(t, dir, "AIWF_COVERAGE_BASE=no-such-ref-anywhere")
		if !strings.Contains(out, "does not name a commit") {
			t.Errorf("an unresolvable base must be reported as such; got:\n%s", out)
		}
		if strings.Contains(out, "will no-op") {
			t.Errorf("an unresolvable base makes the gates fail, not no-op; the message must not promise silence:\n%s", out)
		}
	})
}

// TestCoverageGateOnly_RefusesAnEmptyProfile pins that the guard tests for
// content, not existence. An empty coverage.out satisfies `test -f` while
// carrying no blocks at all, which is a profile the gate cannot audit.
func TestCoverageGateOnly_RefusesAnEmptyProfile(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	recipe := makeDryRun(t, root, "coverage-gate-only")
	guardAt := strings.Index(recipe, "base=")
	if guardAt < 0 {
		t.Fatalf("coverage-gate-only must resolve a base ref:\n%s", recipe)
	}

	empty := filepath.Join(t.TempDir(), "coverage.out")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatalf("writing empty profile: %v", err)
	}
	guard := strings.ReplaceAll(recipe[:guardAt], filepath.Join(root, "coverage.out"), empty)

	cmd := exec.Command("sh", "-c", guard)
	cmd.Dir = root
	cmd.Env = stubbedGoEnv(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("a zero-byte coverage.out must abort the target; it exited 0:\n%s", out)
	}
	if !strings.Contains(string(out), "nothing to audit") {
		t.Errorf("the refusal must name the unusable profile; got:\n%s", out)
	}
}

// TestCoverageGateWiring_MakefileForbidsParallel pins the guard that keeps
// `ci`'s prerequisite chain honest. coverage-gate-only reads the profile
// test-cov writes, and `coverage`/`test-cov`/`coverage-gate` all write the
// same coverage.out; under `make -j` those interleave, and the gate reads
// whatever profile happens to be on disk — reporting green off a stale one.
func TestCoverageGateWiring_MakefileForbidsParallel(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "Makefile"))
	if err != nil {
		t.Fatalf("reading Makefile: %v", err)
	}
	// The bare form only. `.NOTPARALLEL: <targets>` is a different
	// directive — GNU make 4.4 honors the prerequisite list and leaves
	// every other target, `ci` included, free to run in parallel.
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == ".NOTPARALLEL:" {
			return
		}
	}
	t.Error("Makefile must declare a bare `.NOTPARALLEL:` — without it `make -j ci` can gate against a stale coverage.out and exit 0")
}

// TestCoverageGateOnly_RefusesWithoutAProfile pins the fail-closed guard.
// The target reads a profile it does not build, so a missing coverage.out
// must abort rather than let the gate report green on a profile that was
// never generated.
func TestCoverageGateOnly_RefusesWithoutAProfile(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	recipe := makeDryRun(t, root, "coverage-gate-only")

	guardAt := strings.Index(recipe, "base=")
	if guardAt < 0 {
		t.Fatalf("coverage-gate-only must resolve a base ref:\n%s", recipe)
	}
	guard := recipe[:guardAt]

	// Run the guard alone, in a directory holding no coverage.out.
	cmd := exec.Command("sh", "-c", strings.ReplaceAll(guard, filepath.Join(root, "coverage.out"), filepath.Join(t.TempDir(), "coverage.out")))
	cmd.Dir = root
	cmd.Env = stubbedGoEnv(t)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("a missing coverage.out must abort the target; it exited 0:\n%s", out)
	}
	if !strings.Contains(string(out), "nothing to audit") {
		t.Errorf("the refusal must name the missing profile; got:\n%s", out)
	}
}
