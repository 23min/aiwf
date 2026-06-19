package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// firingFixtureModule is the synthetic module path for the
// firingFixtureCore fixtures; parseCoverProfile strips it to recover
// repo-relative paths.
const firingFixtureModule = "example.com/cov"

// firingFixture builds a throwaway tree at a t.TempDir: a go.mod, a
// synthetic internal/policies/fake.go holding policySrc, and a coverage
// profile that marks each `Policy: "<id>"` construction line covered or
// uncovered per dark[id] (default covered). It returns the root and the
// profile path for firingFixtureCore / firingFixtureViolations.
func firingFixture(t *testing.T, policySrc string, dark map[string]bool) (root, profilePath string) {
	t.Helper()
	root = t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module "+firingFixtureModule+"\n\ngo 1.24\n")
	write("internal/policies/fake.go", policySrc)

	// Build one coverage block per construction line, count 0 (dark) or
	// 1 (covered), so the line's enclosing block is exactly that line.
	var b strings.Builder
	b.WriteString("mode: set\n")
	for i, line := range strings.Split(policySrc, "\n") {
		m := constructionLinePat.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		count := 1
		if dark[m[1]] {
			count = 0
		}
		ln := i + 1
		fmt.Fprintf(&b, "%s/internal/policies/fake.go:%d.2,%d.40 1 %d\n", firingFixtureModule, ln, ln, count)
	}
	profilePath = filepath.Join(root, "coverage.out")
	write("coverage.out", b.String())
	return root, profilePath
}

// twoPolicySrc is a synthetic policies file: one dark policy and one
// lit policy, each on a single-line Violation construction.
const twoPolicySrc = `package policies

func PolicyAlpha(root string) ([]Violation, error) {
	return []Violation{{Policy: "alpha-dark"}}, nil
}

func PolicyBeta(root string) ([]Violation, error) {
	return []Violation{{Policy: "beta-lit"}}, nil
}
`

func siteIDs(sites []constructionSite) []string {
	out := make([]string, 0, len(sites))
	for _, s := range sites {
		out = append(out, s.id)
	}
	sort.Strings(out)
	return out
}

// TestFiringFixtureCore_FlagsOnlyUncoveredSites proves the dark
// detection: the policy whose construction line ran zero times is
// reported, the covered one is not.
func TestFiringFixtureCore_FlagsOnlyUncoveredSites(t *testing.T) {
	t.Parallel()
	root, profile := firingFixture(t, twoPolicySrc, map[string]bool{"alpha-dark": true})

	dark, err := firingFixtureCore(root, profile)
	if err != nil {
		t.Fatalf("firingFixtureCore: %v", err)
	}
	if got := siteIDs(dark); len(got) != 1 || got[0] != "alpha-dark" {
		t.Fatalf("dark sites = %v, want [alpha-dark]", got)
	}
	if dark[0].file != "internal/policies/fake.go" {
		t.Errorf("file = %q", dark[0].file)
	}
	if dark[0].line != 4 { // the `Policy: "alpha-dark"` line
		t.Errorf("line = %d, want 4", dark[0].line)
	}
}

// TestFiringFixtureCore_AllCovered proves a tree where every
// construction line ran is reported fully lit (no dark sites).
func TestFiringFixtureCore_AllCovered(t *testing.T) {
	t.Parallel()
	root, profile := firingFixture(t, twoPolicySrc, nil)
	dark, err := firingFixtureCore(root, profile)
	if err != nil {
		t.Fatalf("firingFixtureCore: %v", err)
	}
	if len(dark) != 0 {
		t.Fatalf("dark sites = %v, want none", siteIDs(dark))
	}
}

// TestFiringFixtureViolations_AllowlistGate proves the allowlist filter:
// an unlisted dark policy fires a violation; the same policy listed is
// tolerated. This also covers the gate's own Violation construction
// line, keeping firing-fixture-presence lit on a clean live tree.
func TestFiringFixtureViolations_AllowlistGate(t *testing.T) {
	t.Parallel()
	root, profile := firingFixture(t, twoPolicySrc, map[string]bool{"alpha-dark": true})

	vs, err := firingFixtureViolations(root, profile, nil)
	if err != nil {
		t.Fatalf("firingFixtureViolations: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("violations = %+v, want one for alpha-dark", vs)
	}
	if vs[0].Policy != "firing-fixture-presence" {
		t.Errorf("policy id = %q", vs[0].Policy)
	}
	if !strings.Contains(vs[0].Detail, "alpha-dark") {
		t.Errorf("detail does not name the dark policy: %q", vs[0].Detail)
	}

	allowed, err := firingFixtureViolations(root, profile, map[string]bool{"alpha-dark": true})
	if err != nil {
		t.Fatalf("firingFixtureViolations (allowed): %v", err)
	}
	if len(allowed) != 0 {
		t.Fatalf("violations with alpha-dark allowed = %+v, want none", allowed)
	}
}

// TestFiringFixtureCore_FailsClosedWithoutPoliciesCoverage proves the
// fail-closed guard: a profile that instruments no internal/policies
// file errors loudly rather than silently passing every policy.
func TestFiringFixtureCore_FailsClosedWithoutPoliciesCoverage(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module "+firingFixtureModule+"\n\ngo 1.24\n")
	mustWrite(t, filepath.Join(root, "coverage.out"),
		"mode: set\n"+firingFixtureModule+"/internal/foo/bar.go:1.1,1.10 1 1\n")

	_, err := firingFixtureCore(root, filepath.Join(root, "coverage.out"))
	if err == nil {
		t.Fatal("expected an error when the profile carries no internal/policies blocks")
	}
	if !strings.Contains(err.Error(), "no internal/policies blocks") {
		t.Errorf("error does not explain the fail-closed cause: %v", err)
	}
}

// TestFiringFixtureCore_Errors covers the input-error branches: a
// missing go.mod (modulePath), an unreadable profile (parseCoverProfile),
// and an absent internal/policies directory (constructionSites) after a
// profile that does carry policies blocks.
func TestFiringFixtureCore_Errors(t *testing.T) {
	t.Parallel()

	t.Run("modulePath error when go.mod absent", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mustWrite(t, filepath.Join(root, "coverage.out"), "mode: set\n")
		if _, err := firingFixtureCore(root, filepath.Join(root, "coverage.out")); err == nil {
			t.Fatal("want error for missing go.mod, got nil")
		}
	})

	t.Run("parseCoverProfile error when profile missing", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mustWrite(t, filepath.Join(root, "go.mod"), "module "+firingFixtureModule+"\n\ngo 1.24\n")
		if _, err := firingFixtureCore(root, filepath.Join(root, "nope.out")); err == nil {
			t.Fatal("want error for missing profile, got nil")
		}
	})

	t.Run("constructionSites error when internal/policies absent", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		mustWrite(t, filepath.Join(root, "go.mod"), "module "+firingFixtureModule+"\n\ngo 1.24\n")
		// A profile that carries an internal/policies block (passes the
		// fail-closed guard) but no such directory on disk → ReadDir fails.
		mustWrite(t, filepath.Join(root, "coverage.out"),
			"mode: set\n"+firingFixtureModule+"/internal/policies/x.go:1.1,1.10 1 1\n")
		if _, err := firingFixtureCore(root, filepath.Join(root, "coverage.out")); err == nil {
			t.Fatal("want error for missing internal/policies dir, got nil")
		}
	})
}

// TestFiringFixtureViolations_PropagatesCoreError proves the wrapper
// surfaces a firingFixtureCore error rather than swallowing it.
func TestFiringFixtureViolations_PropagatesCoreError(t *testing.T) {
	t.Parallel()
	root := t.TempDir() // no go.mod → firingFixtureCore errors at modulePath
	if _, err := firingFixtureViolations(root, filepath.Join(root, "coverage.out"), nil); err == nil {
		t.Fatal("want propagated error, got nil")
	}
}

// TestConstructionSites_SkipsNonGoTestAndDirs pins the scan's skip
// behavior: only non-test .go files are read. A _test.go file, a non-.go
// file, and a subdirectory in internal/policies are all skipped — even
// when they textually contain a `Policy: "<id>"` literal — so their ids
// never enter the construction-site set.
func TestConstructionSites_SkipsNonGoTestAndDirs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	base := filepath.Join(root, "internal", "policies")
	mustWrite(t, filepath.Join(base, "real.go"),
		"package policies\n\nfunc PolicyReal() ([]Violation, error) {\n\treturn []Violation{{Policy: \"real-one\"}}, nil\n}\n")
	// _test.go → skipped despite carrying a Policy literal.
	mustWrite(t, filepath.Join(base, "real_test.go"),
		"package policies\n\nvar _ = Violation{Policy: \"should-be-skipped\"}\n")
	// non-.go → skipped.
	mustWrite(t, filepath.Join(base, "notes.md"), "Policy: \"md-noise\"\n")
	// subdir → skipped (ReadDir is non-recursive); its .go is never read.
	mustWrite(t, filepath.Join(base, "sub", "nested.go"),
		"package sub\n\nvar _ = struct{}{ /* Policy: \"nested\" */ }\n")

	sites, err := constructionSites(root)
	if err != nil {
		t.Fatalf("constructionSites: %v", err)
	}
	got := siteIDs(sites)
	if len(got) != 1 || got[0] != "real-one" {
		t.Fatalf("construction sites = %v, want exactly [real-one] (others must be skipped)", got)
	}
}

// TestPolicyFiringFixturePresence_Env drives the env-fed entry point
// through both branches so the wrapper's own lines are covered (it is
// never reached in the broad test run, where the live gate skips on an
// unset profile). Serial (t.Setenv panics under t.Parallel) and
// documented in setup_test.go's skip-list.
func TestPolicyFiringFixturePresence_Env(t *testing.T) {
	// Unset profile → no-op.
	t.Setenv("AIWF_COVERAGE_PROFILE", "")
	vs, err := PolicyFiringFixturePresence(t.TempDir())
	if err != nil {
		t.Fatalf("unset profile: unexpected error: %v", err)
	}
	if vs != nil {
		t.Fatalf("unset profile: want nil violations, got %+v", vs)
	}

	// Set profile → delegates to firingFixtureViolations and surfaces the
	// dark, non-grandfathered policy in the fixture tree.
	root, profilePath := firingFixture(t, twoPolicySrc, map[string]bool{"alpha-dark": true})
	t.Setenv("AIWF_COVERAGE_PROFILE", profilePath)
	vs, err = PolicyFiringFixturePresence(root)
	if err != nil {
		t.Fatalf("set profile: unexpected error: %v", err)
	}
	if len(vs) != 1 || vs[0].Policy != "firing-fixture-presence" {
		t.Fatalf("set profile: want one firing-fixture-presence violation, got %+v", vs)
	}
}

// TestPolicy_FiringFixturePresence is the CI gate entry point. It audits
// the live policy corpus against the coverage profile supplied via
// AIWF_COVERAGE_PROFILE, failing for any non-grandfathered policy whose
// firing branch no test covers. Without a profile (the default in the
// broad `go test ./...` job) it skips — the authoritative invocation is
// the CI coverage-gate step and `make coverage-gate`, both of which set
// the env var.
func TestPolicy_FiringFixturePresence(t *testing.T) {
	t.Parallel()
	if os.Getenv("AIWF_COVERAGE_PROFILE") == "" {
		t.Skip("AIWF_COVERAGE_PROFILE unset; run via `make coverage-gate` or the CI coverage-gate step")
	}
	runPolicy(t, PolicyFiringFixturePresence)
}

// TestPolicy_FiringFixtureNoStaleAllowlist keeps the grandfatherDark
// ledger honest: every id it lists must still be dark in the live
// profile. When a firing fixture lands (G-0262 burn-down) the policy's
// construction line becomes covered, this test fails naming the id, and
// the fix is to delete the now-stale grandfatherDark entry — so the
// ledger shrinks monotonically and cannot rot. Env-gated like the gate.
func TestPolicy_FiringFixtureNoStaleAllowlist(t *testing.T) {
	t.Parallel()
	profile := os.Getenv("AIWF_COVERAGE_PROFILE")
	if profile == "" {
		t.Skip("AIWF_COVERAGE_PROFILE unset; run via `make coverage-gate` or the CI coverage-gate step")
	}
	dark, err := firingFixtureCore(repoRoot(t), profile)
	if err != nil {
		t.Fatalf("firingFixtureCore: %v", err)
	}
	stillDark := map[string]bool{}
	for _, s := range dark {
		stillDark[s.id] = true
	}
	for id := range grandfatherDark {
		if !stillDark[id] {
			t.Errorf("grandfatherDark lists %q but it is no longer dark (a test now covers its construction line, or the policy was renamed/removed). Delete the entry — its firing fixture has landed (G-0262 burn-down).", id)
		}
	}
}
