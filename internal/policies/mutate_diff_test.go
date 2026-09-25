package policies

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/testsupport"
)

// mutate_diff_test.go — G-0267 / G-0110 chokepoint pins.
//
// Behavioral tests for scripts/mutate-diff.sh (the `make mutate-diff`
// advisory diff-scoped mutation runner): the wf-vacuity / mutate-hunt
// companion that mutates only the internal/ Go lines changed since the
// merge-base with origin/main.
//
// Mutation tooling is deliberately not installed in routine CI
// (mutate-hunt is workflow_dispatch-only), so the pins split in two.
// The CI-runnable ones drive the script against a stand-in gremlins
// that records how it was run and what its own `git diff` returns, and
// that hands back a scripted report: they pin which directories run,
// what gremlins is shown, and how the report is read.
// TestMutateDiff_MutatesOnlyChangedLines drives the real tool and skips
// when gremlins is absent — it runs wherever a developer has gremlins
// on PATH, the same posture as the tool it tests.

func mutateDiffScriptPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "scripts", "mutate-diff.sh")
}

// runMutateDiff runs the script from dir with env appended to the
// process environment, less any MUTATE_DIFF_* or GREMLINS_* setting
// the developer's shell carries; fails the test on a non-zero exit
// (the script is advisory and always exits 0); and returns its
// combined output.
func runMutateDiff(t *testing.T, dir string, env ...string) string {
	t.Helper()
	cmd := exec.Command("bash", mutateDiffScriptPath(t))
	cmd.Dir = dir
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "MUTATE_DIFF_") && !strings.HasPrefix(kv, "GREMLINS_") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mutate-diff.sh must exit 0 (advisory); got %v\n--- output ---\n%s", err, out)
	}
	return string(out)
}

// markerLines returns the output lines that begin, once trimmed, with
// marker.
func markerLines(out, marker string) []string {
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); strings.HasPrefix(line, marker) {
			lines = append(lines, line)
		}
	}
	return lines
}

const mutateDiffPassLine = "mutate-diff: no surviving mutants (LIVED) on changed internal/ lines — advisory pass."

// requireJQ skips a test that drives the script past its jq check.
func requireJQ(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not on PATH; mutate-diff needs it to read gremlins' report")
	}
}

// stubGremlinsScript stands in for gremlins, driven by files in
// $STUB_DIR. For each run it appends one line to $STUB_DIR/log: the
// directory it ran in; the --diff ref, --workers and
// --timeout-coefficient it was handed; whether git still answers,
// both through and around the shim's diff branch, when a test runs it
// under a PATH holding neither bash nor env; the names that
// `git diff --name-only <ref>` and `git diff --merge-base <ref>
// --name-only` return, which the shim must pass through untouched; and
// each file:hunk its own `git diff --merge-base <ref>` returns — the
// call gremlins makes to build its filter — leaving out a deleted
// file's hunks. It then writes $STUB_DIR/<run dir name>.json as its
// report, if present, and exits with the status in
// $STUB_DIR/<run dir name>.exit, default 0.
const stubGremlinsScript = `#!/usr/bin/env bash
base= out= workers= coeff=
while [ $# -gt 0 ]; do
	case "$1" in
	--diff) base="$2"; shift ;;
	--output) out="$2"; shift ;;
	--workers) workers="$2"; shift ;;
	--timeout-coefficient) coeff="$2"; shift ;;
	esac
	shift
done
name="$(basename "$PWD")"
g="$(command -v git)"
bare="$(PATH=/nonexistent "$g" --version >/dev/null 2>&1 && echo ok || echo fail),$(PATH=/nonexistent "$g" diff --merge-base "$base" >/dev/null 2>&1 && echo ok || echo fail)"
passes="$( { git diff --name-only "$base"; git diff --merge-base "$base" --name-only; } | LC_ALL=C sort -u | paste -sd' ' -)"
sees="$(git diff --merge-base "$base" | awk '/^\+\+\+ / { f = ($2 == "/dev/null") ? "" : substr($0, 7); sub(/\t$/, "", f) } /^@@ / && f != "" { printf "%s%s:%s", sep, f, $3; sep = " " }')"
printf 'cwd=%s diff=%s workers=%s coefficient=%s bare=%s passes=%s sees=%s\n' \
	"$(git rev-parse --show-prefix)" "$base" "$workers" "$coeff" "$bare" "$passes" "$sees" >>"$STUB_DIR/log"
if [ -f "$STUB_DIR/$name.json" ]; then
	cat "$STUB_DIR/$name.json" >"$out"
fi
exit "$(cat "$STUB_DIR/$name.exit" 2>/dev/null || echo 0)"
`

// stubGremlins installs the stand-in in a fresh directory and returns
// that directory, which also holds its log and per-run replies, and
// the environment that puts it ahead of any real gremlins.
func stubGremlins(t *testing.T) (dir string, env []string) {
	t.Helper()
	dir = t.TempDir()
	if err := testsupport.WriteExecutable(filepath.Join(dir, "gremlins"), []byte(stubGremlinsScript)); err != nil {
		t.Fatal(err)
	}
	return dir, []string{"STUB_DIR=" + dir, "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")}
}

// stubLog returns the stand-in's log lines, sorted.
func stubLog(t *testing.T, stubDir string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(stubDir, "log"))
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	sort.Strings(lines)
	return lines
}

// stubMutation is one mutant in a scripted gremlins report.
type stubMutation struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// stubFile is one file's mutations in a scripted gremlins report,
// named as gremlins names it: relative to the run directory.
type stubFile struct {
	FileName  string         `json:"file_name"`
	Mutations []stubMutation `json:"mutations"`
}

// stubReply writes the stand-in's report for the run directory named
// run.
func stubReply(t *testing.T, stubDir, run string, files ...stubFile) {
	t.Helper()
	data, err := json.Marshal(map[string][]stubFile{"files": append([]stubFile{}, files...)})
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(stubDir, run+".json"), string(data))
}

// TestMutateDiff_Wiring pins the install surface independently of
// gremlins so the gate has CI-level teeth: drop the script, its exec
// bit, or the Makefile recipe that runs it and the advisory tool
// silently stops being invokable — the rot mode this pin catches.
func TestMutateDiff_Wiring(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	info, err := os.Stat(mutateDiffScriptPath(t))
	if err != nil {
		t.Fatalf("tracked mutate-diff script missing: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("scripts/mutate-diff.sh must be executable; mode = %v", info.Mode())
	}

	if _, lookErr := exec.LookPath("make"); lookErr != nil {
		t.Skip("make not on PATH")
	}
	cmd := exec.Command("make", "-n", "--no-print-directory", "mutate-diff")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("make -n mutate-diff: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "scripts/mutate-diff.sh" {
		t.Errorf("make mutate-diff must run scripts/mutate-diff.sh; its recipe is %q", got)
	}
}

// TestMutateDiff_ScopesEachGremlinsRun pins which directories run, how
// gremlins is invoked in each, and what its diff call shows it. The
// fixture's changes cover the scoping rules: two unstaged edits in one
// file around an untouched line; a changed subpackage, committed after
// the fork, under a changed parent; a test edit beside a code edit; a
// staged edit in a sibling directory sharing the first's name prefix,
// with one file moved into it from another directory and one moved
// within it, each edited, so each reads as wholly new;
// an untracked file in a changed package and one in a new package; a
// test-only change under a changed parent and in a package of its own;
// a deleted package; an edit outside internal/; and a file trunk
// deleted after the fork. The caller's git config asks for color, no
// path prefixes, wider hunk context and an external diff tool, and its
// environment for three context lines, all of which the diff gremlins
// parses must refuse; a gitignored file sits beside the untracked ones.
// The coefficient is overridden, and the script runs from a
// subdirectory.
func TestMutateDiff_ScopesEachGremlinsRun(t *testing.T) {
	t.Parallel()
	requireJQ(t)

	dir := t.TempDir()
	git := func(args ...string) string { return gitIn(t, dir, args...) }
	write := func(rel, content string) { mustWrite(t, filepath.Join(dir, rel), content) }
	git("init", "-q", "-b", "main")

	write("internal/a/z.go", "package a\n\nvar A = 10\nvar B = 20\nvar C = 30\n")
	write("internal/a/a_test.go", "package a\n")
	write("internal/a/sub/s_test.go", "package sub\n")
	write("internal/a/sub2/x.go", "package sub2\n\nvar X = 50\n")
	write("internal/ab/ab.go", "package ab\n\nvar AB = 20\n")
	write("internal/e/m.go", "package ab\n\nvar M = 1\n")
	write("internal/ab/n.go", "package ab\n\nvar N = 1\n")
	write("internal/testonly/o_test.go", "package testonly\n")
	write("internal/testonly/p_test.go", "package testonly\n")
	write("internal/gone/g.go", "package gone\n\nvar G = 1\n")
	write("internal/trunkdel/t.go", "package trunkdel\n\nvar T = 1\n")
	write("cmd/x/x.go", "package main\n\nvar X = 60\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	base := git("rev-parse", "HEAD")

	git("checkout", "-q", "-b", "trunk")
	git("rm", "-q", "internal/trunkdel/t.go")
	git("commit", "-q", "-m", "trunk moves on")
	git("checkout", "-q", "main")
	write("internal/a/sub2/x.go", "package sub2\n\nvar X = 51\n")
	git("add", "-A")
	git("commit", "-q", "-m", "main moves on")

	write("internal/a/z.go", "package a\n\nvar A = 11\nvar B = 20\nvar C = 31\n")
	write("internal/a/a_test.go", "package a\n\n// widened\n")
	write("internal/a/u.go", "package a\n\nvar U = 70\n")
	write("internal/a/sub/s_test.go", "package sub\n\n// widened\n")
	write("internal/fresh/f.go", "package fresh\n\nvar F = 80\n")
	write("internal/ab/ab.go", "package ab\n\nvar AB = 21\n")
	git("mv", "internal/e/m.go", "internal/ab/m.go")
	write("internal/ab/m.go", "package ab\n\nvar M = 2\n")
	git("mv", "internal/ab/n.go", "internal/ab/n2.go")
	write("internal/ab/n2.go", "package ab\n\nvar N = 2\n")
	git("add", "internal/ab")
	write("internal/testonly/o_test.go", "package testonly\n\n// widened\n")
	write("internal/testonly/p_test.go", "package testonly\n\n// widened\n")
	if err := os.Remove(filepath.Join(dir, "internal", "gone", "g.go")); err != nil {
		t.Fatal(err)
	}
	write("cmd/x/x.go", "package main\n\nvar X = 61\n")
	git("config", "color.ui", "always")
	git("config", "diff.noprefix", "true")
	git("config", "diff.interHunkContext", "3")
	git("config", "diff.external", "true")
	write(".gitignore", "/internal/a/ignored.go\n")
	write("internal/a/ignored.go", "package a\n")

	stubDir, stubEnv := stubGremlins(t)
	for _, run := range []string{"a", "ab"} {
		stubReply(t, stubDir, run)
	}
	env := append([]string{"MUTATE_DIFF_BASE=trunk", "MUTATE_DIFF_COEFFICIENT=7", "GIT_DIFF_OPTS=--unified=3"}, stubEnv...)
	got := runMutateDiff(t, filepath.Join(dir, "internal"), env...)

	invocation := "diff=" + base + " workers=1 coefficient=7 bare=ok,ok"
	wantRuns := []string{
		"cwd=internal/a/ " + invocation + " passes=cmd/x/x.go internal/a/a_test.go internal/a/sub/s_test.go internal/a/sub2/x.go internal/a/z.go internal/ab/ab.go internal/ab/m.go internal/ab/n2.go internal/gone/g.go internal/testonly/o_test.go internal/testonly/p_test.go sees=a_test.go:+2,2 sub/s_test.go:+2,2 sub2/x.go:+3 z.go:+3 z.go:+5",
		"cwd=internal/ab/ " + invocation + " passes=cmd/x/x.go internal/a/a_test.go internal/a/sub/s_test.go internal/a/sub2/x.go internal/a/z.go internal/ab/ab.go internal/ab/m.go internal/ab/n2.go internal/gone/g.go internal/testonly/o_test.go internal/testonly/p_test.go sees=ab.go:+3 m.go:+1,3 n2.go:+1,3",
	}
	if diff := cmp.Diff(wantRuns, stubLog(t, stubDir)); diff != "" {
		t.Errorf("gremlins runs (-want +got):\n%s\n--- output ---\n%s", diff, got)
	}
	checks := []struct {
		marker string
		want   []string
	}{
		{"UNTRACKED", []string{"UNTRACKED internal/a/u.go", "UNTRACKED internal/fresh/f.go"}},
		{"TEST-ONLY", []string{"TEST-ONLY ./internal/a/sub", "TEST-ONLY ./internal/testonly"}},
		{"gremlins unleash", []string{
			"gremlins unleash --workers 1 --timeout-coefficient 7 ./internal/a/sub",
			"gremlins unleash --workers 1 --timeout-coefficient 7 ./internal/testonly",
		}},
		{"mutate-diff: no surviving", []string{mutateDiffPassLine}},
		{"./", []string{"./internal/a", "./internal/ab"}},
	}
	for _, c := range checks {
		if diff := cmp.Diff(c.want, markerLines(got, c.marker)); diff != "" {
			t.Errorf("%s lines (-want +got):\n%s\n--- output ---\n%s", c.marker, diff, got)
		}
	}
}

// TestMutateDiff_ReportsEachRunsOutcome pins how a run's outcome
// reaches the summary. LIVED mutants are listed from every run's report
// and totalled. A run gremlins exits non-zero from, or whose report jq
// cannot read, is named FAILED and withholds the advisory pass. A run
// with no report at exit 0 found nothing to mutate and costs the pass
// nothing. A run of internal/d follows internal/c each time and must
// stay unmarked unless the case says otherwise. The script exits 0
// throughout.
func TestMutateDiff_ReportsEachRunsOutcome(t *testing.T) {
	t.Parallel()
	requireJQ(t)

	failedSummary := "mutate-diff: WARNING 1 gremlins run(s) FAILED (above) — those packages were not tested."
	incomplete := "mutate-diff: no surviving mutants (LIVED) reported, but the run is incomplete — see the WARNING above."
	cases := []struct {
		name        string
		setup       func(t *testing.T, stubDir string)
		wantMarked  []string
		wantSummary []string
	}{
		{
			name: "survivors are listed and totalled across runs",
			setup: func(t *testing.T, stubDir string) {
				t.Helper()
				stubReply(t, stubDir, "c", stubFile{"c.go", []stubMutation{
					{Type: "CONDITIONALS_BOUNDARY", Status: "LIVED", Line: 3, Column: 5},
					{Type: "CONDITIONALS_NEGATION", Status: "KILLED", Line: 3, Column: 5},
					{Type: "INVERT_NEGATIVES", Status: "LIVED", Line: 3, Column: 7},
				}})
				stubReply(t, stubDir, "d", stubFile{"d.go", []stubMutation{{Type: "ARITHMETIC_BASE", Status: "LIVED", Line: 3, Column: 9}}})
			},
			wantMarked: []string{
				"SURVIVOR LIVED c.go:3:5 (CONDITIONALS_BOUNDARY) in ./internal/c",
				"SURVIVOR LIVED c.go:3:7 (INVERT_NEGATIVES) in ./internal/c",
				"SURVIVOR LIVED d.go:3:9 (ARITHMETIC_BASE) in ./internal/d",
			},
			wantSummary: []string{"mutate-diff: 3 surviving mutant(s) (LIVED) — ADVISORY."},
		},
		{
			name: "gremlins exits non-zero",
			setup: func(t *testing.T, stubDir string) {
				t.Helper()
				mustWrite(t, filepath.Join(stubDir, "c.exit"), "3")
				stubReply(t, stubDir, "d", stubFile{"d.go", []stubMutation{{Type: "ARITHMETIC_BASE", Status: "LIVED", Line: 3, Column: 9}}})
			},
			wantMarked:  []string{"FAILED ./internal/c — gremlins exited 3.", "SURVIVOR LIVED d.go:3:9 (ARITHMETIC_BASE) in ./internal/d"},
			wantSummary: []string{failedSummary, "mutate-diff: 1 surviving mutant(s) (LIVED) — ADVISORY."},
		},
		{
			name: "the report is unreadable",
			setup: func(t *testing.T, stubDir string) {
				t.Helper()
				mustWrite(t, filepath.Join(stubDir, "c.json"), `{"files": [`)
			},
			wantMarked:  []string{"FAILED ./internal/c — jq cannot read gremlins' report."},
			wantSummary: []string{failedSummary, incomplete},
		},
		{
			// The empty run is internal/d, after a run that did write a
			// report, so no earlier report can stand in for its own.
			name: "gremlins finds nothing to mutate",
			setup: func(t *testing.T, stubDir string) {
				t.Helper()
				stubReply(t, stubDir, "c")
				if err := os.Remove(filepath.Join(stubDir, "d.json")); err != nil {
					t.Fatal(err)
				}
			},
			wantMarked:  []string{"NO MUTANTS ./internal/d — gremlins found nothing to mutate."},
			wantSummary: []string{mutateDiffPassLine},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			git := func(args ...string) string { return gitIn(t, dir, args...) }
			write := func(rel, content string) { mustWrite(t, filepath.Join(dir, rel), content) }
			git("init", "-q", "-b", "main")
			write("internal/c/c.go", "package c\n\nvar C = 1\n")
			write("internal/d/d.go", "package d\n\nvar D = 1\n")
			git("add", "-A")
			git("commit", "-q", "-m", "base")
			base := git("rev-parse", "HEAD")
			write("internal/c/c.go", "package c\n\nvar C = 2\n")
			write("internal/d/d.go", "package d\n\nvar D = 2\n")

			stubDir, stubEnv := stubGremlins(t)
			stubReply(t, stubDir, "d")
			tc.setup(t, stubDir)
			got := runMutateDiff(t, dir, append([]string{"MUTATE_DIFF_BASE=" + base}, stubEnv...)...)

			var marked []string
			for _, marker := range []string{"FAILED ./", "NO MUTANTS ./", "SURVIVOR LIVED"} {
				marked = append(marked, markerLines(got, marker)...)
			}
			if diff := cmp.Diff(tc.wantMarked, marked); diff != "" {
				t.Errorf("marked runs and mutants (-want +got):\n%s\n--- output ---\n%s", diff, got)
			}
			want := append([]string{"mutate-diff: changed internal/ Go lines vs " + base + " (base " + base[:12] + "), in 2 run(s):"}, tc.wantSummary...)
			if diff := cmp.Diff(want, markerLines(got, "mutate-diff: ")); diff != "" {
				t.Errorf("summary lines (-want +got):\n%s\n--- output ---\n%s", diff, got)
			}
		})
	}
}

// TestMutateDiff_TestOnlyChangeMutatesNothing pins the test-only path:
// a diff whose only changed lines are in tests starts no gremlins run,
// names the package with the command that mutates it whole, and exits 0.
// The line above the change holds tabs, which a patch's hunk header
// repeats; the changed-file listing must not read them as a path.
func TestMutateDiff_TestOnlyChangeMutatesNothing(t *testing.T) {
	t.Parallel()
	requireJQ(t)

	dir := t.TempDir()
	git := func(args ...string) string { return gitIn(t, dir, args...) }
	write := func(rel, content string) { mustWrite(t, filepath.Join(dir, rel), content) }
	git("init", "-q", "-b", "main")
	write("internal/t/t.go", "package t\n")
	write("internal/t/t_test.go", "package t\n\nvar cols = \"name\tkind\tstatus\"\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	base := git("rev-parse", "HEAD")
	write("internal/t/t_test.go", "package t\n\nvar cols = \"name\tkind\tstatus\"\n\n// widened\n")
	git("update-ref", "refs/remotes/origin/main", base)

	stubDir, stubEnv := stubGremlins(t)
	got := runMutateDiff(t, dir, stubEnv...)

	if runs := stubLog(t, stubDir); len(runs) != 0 {
		t.Errorf("a test-only diff must start no gremlins run; got %q\n--- output ---\n%s", runs, got)
	}
	want := []string{
		"mutate-diff: no changed internal/ Go code lines vs origin/main — nothing to mutate.",
		"mutate-diff: only test lines changed in these packages — no code line to mutate.",
	}
	if diff := cmp.Diff(want, markerLines(got, "mutate-diff: ")); diff != "" {
		t.Errorf("summary lines (-want +got):\n%s\n--- output ---\n%s", diff, got)
	}
	if diff := cmp.Diff([]string{"TEST-ONLY ./internal/t"}, markerLines(got, "TEST-ONLY")); diff != "" {
		t.Errorf("test-only packages (-want +got):\n%s\n--- output ---\n%s", diff, got)
	}
	if diff := cmp.Diff([]string{"gremlins unleash --workers 1 --timeout-coefficient 15 ./internal/t"}, markerLines(got, "gremlins unleash ")); diff != "" {
		t.Errorf("whole-package command (-want +got):\n%s\n--- output ---\n%s", diff, got)
	}
}

// TestMutateDiff_ExitsZeroWhenItCannotRun pins the advisory contract on
// the paths that stop before any run: no gremlins, no jq, or a base ref
// that does not resolve each print why and exit 0. PATH holds only the
// tools each case allows.
func TestMutateDiff_ExitsZeroWhenItCannotRun(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		tools []string
		base  string
		want  string
	}{
		{"gremlins is missing", []string{"git", "jq"}, "HEAD", "mutate-diff: gremlins not installed — advisory tool unavailable. Install with:"},
		{"jq is missing", []string{"git", "gremlins"}, "HEAD", "mutate-diff: jq not installed — required to summarize gremlins' JSON report."},
		{"the base ref does not resolve", []string{"git", "gremlins", "jq"}, "no-such-ref", "mutate-diff: cannot resolve base ref 'no-such-ref' — nothing to mutate."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			git := func(args ...string) string { return gitIn(t, dir, args...) }
			git("init", "-q", "-b", "main")
			git("commit", "-q", "--allow-empty", "-m", "base")

			bin := t.TempDir()
			for _, tool := range tc.tools {
				if tool == "gremlins" {
					if err := testsupport.WriteExecutable(filepath.Join(bin, tool), []byte(stubGremlinsScript)); err != nil {
						t.Fatal(err)
					}
					continue
				}
				found, err := exec.LookPath(tool)
				if err != nil {
					t.Skipf("%s not on PATH", tool)
				}
				if err := os.Symlink(found, filepath.Join(bin, tool)); err != nil {
					t.Fatal(err)
				}
			}
			got := runMutateDiff(t, dir, "PATH="+bin, "MUTATE_DIFF_BASE="+tc.base)
			if diff := cmp.Diff([]string{tc.want}, markerLines(got, "mutate-diff: ")); diff != "" {
				t.Errorf("output (-want +got):\n%s\n--- output ---\n%s", diff, got)
			}
		})
	}
}

// probeTest renders a test in package pkg that calls each named
// threshold function (`n >= limit`) at 0 and 100 only, so the
// function's CONDITIONALS_NEGATION mutant dies and its
// CONDITIONALS_BOUNDARY mutant survives: a boundary survivor marks a
// line gremlins actually mutated.
func probeTest(pkg, name string, funcs ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\nimport \"testing\"\n\nfunc Test%s(t *testing.T) {\n", pkg, name)
	for _, f := range funcs {
		fmt.Fprintf(&b, "\tif !%s(100) || %s(0) {\n\t\tt.Fatal(%q)\n\t}\n", f, f, f)
	}
	b.WriteString("}\n")
	return b.String()
}

var survivorMarker = regexp.MustCompile(`^SURVIVOR LIVED (\S+):(\d+):\d+ (\(\S+\)) in (\S+)$`)

// TestMutateDiff_MutatesOnlyChangedLines pins, against real gremlins,
// what the script mutates: the changed internal/ code lines, and
// nothing else. After the base commit the fixture edits two lines of
// internal/a that share one diff hunk around an untouched line, edits
// the subpackage internal/a/sub, adds an untracked file to internal/a,
// changes only the test of internal/testonly, and edits a file under
// cmd/. A boundary survivor must surface for each edited line, and for
// nothing else; the subpackage rides internal/a's run; and the
// test-only package is named rather than mutated.
func TestMutateDiff_MutatesOnlyChangedLines(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("gremlins"); err != nil {
		t.Skip("gremlins not on PATH; mutate-diff is advisory dev tooling (mutate-hunt is workflow_dispatch-only)")
	}
	requireJQ(t)

	dir := t.TempDir()
	git := func(args ...string) string { return gitIn(t, dir, args...) }
	write := func(rel, content string) { mustWrite(t, filepath.Join(dir, rel), content) }
	git("init", "-q", "-b", "main")

	write("go.mod", "module example.test/mutscope\n\ngo 1.24\n")
	write("internal/a/a.go", "package a\n\nfunc A(n int) bool { return n >= 10 }\nfunc B(n int) bool { return n >= 20 }\nfunc C(n int) bool { return n >= 30 }\n")
	write("internal/a/a_test.go", probeTest("a", "ABC", "A", "B", "C"))
	write("internal/a/sub/s.go", "package sub\n\nfunc S(n int) bool { return n >= 40 }\n")
	write("internal/a/sub/s_test.go", probeTest("sub", "S", "S"))
	write("internal/testonly/o.go", "package testonly\n\nfunc O(n int) bool { return n >= 50 }\n")
	write("internal/testonly/o_test.go", probeTest("testonly", "O", "O"))
	write("cmd/x/x.go", "package main\n\nfunc X(n int) bool { return n >= 60 }\n\nfunc main() { _ = X(0) }\n")
	write("cmd/x/x_test.go", probeTest("main", "X", "X"))
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	base := git("rev-parse", "HEAD")

	write("internal/a/a.go", "package a\n\nfunc A(n int) bool { return n >= 11 }\nfunc B(n int) bool { return n >= 20 }\nfunc C(n int) bool { return n >= 31 }\n")
	write("internal/a/sub/s.go", "package sub\n\nfunc S(n int) bool { return n >= 41 }\n")
	write("internal/a/u.go", "package a\n\nfunc U(n int) bool { return n >= 70 }\n")
	write("internal/a/u_test.go", probeTest("a", "U", "U"))
	write("internal/testonly/o_test.go", probeTest("testonly", "O", "O")+"\nfunc TestAgain(t *testing.T) { _ = O(5) }\n")
	write("cmd/x/x.go", "package main\n\nfunc X(n int) bool { return n >= 61 }\n\nfunc main() { _ = X(0) }\n")

	got := runMutateDiff(t, dir, "MUTATE_DIFF_BASE="+base)

	var survivors []string
	for _, line := range markerLines(got, "SURVIVOR LIVED") {
		m := survivorMarker.FindStringSubmatch(line)
		if m == nil {
			t.Fatalf("survivor marker %q does not have the SURVIVOR LIVED <file>:<line>:<col> (<type>) in <dir> shape\n--- output ---\n%s", line, got)
		}
		survivors = append(survivors, fmt.Sprintf("%s:%s %s in %s", m[1], m[2], m[3], m[4]))
	}
	wantSurvivors := []string{
		"a.go:3 (CONDITIONALS_BOUNDARY) in ./internal/a",
		"a.go:5 (CONDITIONALS_BOUNDARY) in ./internal/a",
		"sub/s.go:3 (CONDITIONALS_BOUNDARY) in ./internal/a",
	}
	sort.Strings(survivors)
	if diff := cmp.Diff(wantSurvivors, survivors); diff != "" {
		t.Errorf("mutated lines (-want +got):\n%s\n--- output ---\n%s", diff, got)
	}
	if diff := cmp.Diff([]string{"── mutate-diff: gremlins ./internal/a ──"}, markerLines(got, "── mutate-diff: gremlins ")); diff != "" {
		t.Errorf("gremlins runs (-want +got):\n%s\n--- output ---\n%s", diff, got)
	}
	if diff := cmp.Diff([]string{"TEST-ONLY ./internal/testonly"}, markerLines(got, "TEST-ONLY")); diff != "" {
		t.Errorf("test-only packages (-want +got):\n%s\n--- output ---\n%s", diff, got)
	}
}
