package policies_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// mutate_diff_test.go — G-0267 chokepoint pins.
//
// Behavioral tests for scripts/mutate-diff.sh (the `make mutate-diff`
// advisory diff-scoped mutation runner): the wf-vacuity / mutate-hunt
// companion that mutates only the internal/ packages changed since the
// merge-base with origin/main.
//
// TestMutateDiff_Wiring is the CI-runnable pin (script present +
// executable, Makefile target wired). TestMutateDiff_SurfacesPlanted-
// Survivor is the end-to-end smoke; it skips when gremlins is absent,
// because mutation tooling is deliberately not installed in routine CI
// (mutate-hunt is workflow_dispatch-only) — so it runs wherever a
// developer has gremlins on PATH, the same posture as the tool it
// tests.

func mutateDiffScriptPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRootForHook(t), "scripts", "mutate-diff.sh")
}

// TestMutateDiff_Wiring pins the install surface independently of
// gremlins so the gate has CI-level teeth: drop the script, its exec
// bit, or the Makefile target and the advisory tool silently stops
// being invokable — the rot mode this pin catches.
func TestMutateDiff_Wiring(t *testing.T) {
	t.Parallel()
	root := repoRootForHook(t)

	info, err := os.Stat(mutateDiffScriptPath(t))
	if err != nil {
		t.Fatalf("tracked mutate-diff script missing: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("scripts/mutate-diff.sh must be executable; mode = %v", info.Mode())
	}

	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("reading Makefile: %v", err)
	}
	for _, want := range []string{"mutate-diff:", "scripts/mutate-diff.sh"} {
		if !strings.Contains(string(makefile), want) {
			t.Errorf("Makefile must wire the mutate-diff target; missing %q", want)
		}
	}
}

// TestMutateDiff_SurfacesPlantedSurvivor plants an internal/ package
// whose `<` boundary the weak test never probes, points the script's
// base ref at the pre-package commit so the package counts as
// "changed", runs the script, and asserts it surfaces the survivor via
// its own stable SURVIVOR marker (not gremlins' stdout wording — the
// JSON-derived marker is this wrapper's contract, gremlins' human
// output is upstream-defined and may drift).
func TestMutateDiff_SurfacesPlantedSurvivor(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("gremlins"); err != nil {
		t.Skip("gremlins not on PATH; mutate-diff is advisory dev tooling (mutate-hunt is workflow_dispatch-only)")
	}

	dir := t.TempDir()
	git := func(args ...string) string { return gitInFixture(t, dir, args...) }
	git("init", "-q", "-b", "main")

	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	// Base commit: module skeleton, no internal/ package yet.
	write("go.mod", "module example.test/mutdemo\n\ngo 1.24\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	base := git("rev-parse", "HEAD")

	// Change: an internal/ package whose `n < 10` boundary the weak
	// test never probes (no n==10 / n==9 case), so the
	// CONDITIONALS_BOUNDARY mutant (`<` -> `<=`) survives.
	write("internal/mutdemo/calc.go", strings.Join([]string{
		"package mutdemo",
		"",
		"// AtLeast reports whether n meets the threshold.",
		"func AtLeast(n int) bool {",
		"\tif n < 10 {",
		"\t\treturn false",
		"\t}",
		"\treturn true",
		"}",
		"",
	}, "\n"))
	write("internal/mutdemo/calc_test.go", strings.Join([]string{
		"package mutdemo",
		"",
		"import \"testing\"",
		"",
		"func TestAtLeast(t *testing.T) {",
		"\tif !AtLeast(100) {",
		"\t\tt.Fatal(\"want true for 100\")",
		"\t}",
		"\tif AtLeast(0) {",
		"\t\tt.Fatal(\"want false for 0\")",
		"\t}",
		"}",
		"",
	}, "\n"))
	git("add", "-A")
	git("commit", "-q", "-m", "add mutdemo package")

	cmd := exec.Command("bash", mutateDiffScriptPath(t))
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "MUTATE_DIFF_BASE="+base)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mutate-diff.sh must exit 0 (advisory); got %v\n--- output ---\n%s", err, out)
	}
	got := string(out)

	// Find the wrapper's own survivor marker LINE and assert its parts
	// appear together on it. "SURVIVOR LIVED" is the only token unique to
	// the wrapper — calc.go / CONDITIONALS_BOUNDARY / the package path all
	// also appear in gremlins' raw stdout, so four independent Contains
	// checks would pass even if the wrapper emitted no marker of its own.
	// Asserting the same-line shape pins the JSON-derived marker, not
	// gremlins' wording (per CLAUDE.md "substring assertions are not
	// structural assertions"). The volatile line:col is deliberately not
	// pinned.
	var marker string
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "SURVIVOR LIVED") {
			marker = line
			break
		}
	}
	if marker == "" {
		t.Fatalf("no SURVIVOR marker line in mutate-diff output\n--- output ---\n%s", got)
	}
	for _, want := range []string{"calc.go", "(CONDITIONALS_BOUNDARY)", "in ./internal/mutdemo"} {
		if !strings.Contains(marker, want) {
			t.Errorf("survivor marker line missing %q\nmarker: %q\n--- full output ---\n%s", want, marker, got)
		}
	}
}
