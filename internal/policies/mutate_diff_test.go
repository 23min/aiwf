package policies_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

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
// TestMutateDiff_SurfacesPlantedSurvivor and
// TestMutateDiff_MutatesOnlyChangedLines drive the real tool and skip
// when gremlins is absent — they run wherever a developer has gremlins
// on PATH, the same posture as the tool they test.

func mutateDiffScriptPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRootForHook(t), "scripts", "mutate-diff.sh")
}

// writeFixtureFile writes content to rel under dir, creating parents.
func writeFixtureFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
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
// directory it ran in; the path (the repository root written as
// <top>), --diff ref, --workers and --timeout-coefficient it was
// handed; whether GIT_INDEX_FILE reached
// it; the names two other `git diff` calls return, which the shim must
// pass through untouched (so the caller's own diff config applies to
// them); whether git still answers, both through and around the shim's
// diff branch, when a test runs it under a PATH holding neither bash
// nor env; and each file:hunk its own `git diff --merge-base <ref>`
// returns — the call gremlins makes to build its filter. It then
// writes $STUB_DIR/<run dir name>.json as its report, if present, and
// exits with the status in $STUB_DIR/<run dir name>.exit, default 0.
// The pass-through calls carry diff.autoRefreshIndex=false in the
// environment, so the stand-in itself never writes the index.
const stubGremlinsScript = `#!/usr/bin/env bash
base= out= workers= coeff=
path="${*: -1}"
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
top="$(git rev-parse --show-toplevel)"
names() { GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=diff.autoRefreshIndex GIT_CONFIG_VALUE_0=false git "$@" 2>&1 | paste -sd' ' -; }
passes="$(names diff --name-only "$base")|$(names diff --merge-base "$base" --name-only)"
g="$(command -v git)"
bare="$(PATH=/nonexistent "$g" --version >/dev/null 2>&1 && echo ok || echo fail),$(PATH=/nonexistent "$g" diff --merge-base "$base" >/dev/null 2>&1 && echo ok || echo fail)"
sees="$(git diff --merge-base "$base" | awk '/^\+\+\+ b\// { f = substr($0, 7); sub(/\t$/, "", f) } /^@@ / { printf "%s%s:%s", sep, f, $3; sep = " " }')"
printf 'cwd=%s path=%s diff=%s workers=%s coefficient=%s index=%s bare=%s passes=%s sees=%s\n' \
	"$(git rev-parse --show-prefix)" "${path/#"$top"/<top>}" "$base" "$workers" "$coeff" "${GIT_INDEX_FILE:-unset}" "$bare" "$passes" "$sees" >>"$STUB_DIR/log"
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
	data, err := json.Marshal(map[string][]stubFile{"files": files})
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, stubDir, run+".json", string(data))
}

// gitDirEntries lists the names directly under dir's .git.
func gitDirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// numberedLines renders n lines "// line 1" .. "// line n".
func numberedLines(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("// line %d", i+1)
	}
	return lines
}

// TestMutateDiff_Wiring pins the install surface independently of
// gremlins so the gate has CI-level teeth: drop the script, its exec
// bit, or the Makefile recipe that runs it and the advisory tool
// silently stops being invokable — the rot mode this pin catches.
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
// fixture's changes cover the scoping rules: a same-size edit made in
// the index's own second (which git re-reads only while the index
// keeps its timestamp); an untracked file in a changed package and one
// in a new package; a changed subpackage, committed after the fork,
// under a changed parent; a test edit beside a code edit; a staged
// edit in a sibling directory sharing the first's name prefix; a
// test-only change under a changed parent and in a package of its own;
// a deleted package, a package that lost one file, and a file that
// lost only lines; an edit outside internal/; and a trunk that moved on
// after the fork. A gitignored Go file sits under internal/, the index
// is split, and the caller's config sets color, both prefix options,
// an external diff and diff.relative against the script's fixed diff;
// the coefficient is overridden, and the script runs from a
// subdirectory.
func TestMutateDiff_ScopesEachGremlinsRun(t *testing.T) {
	t.Parallel()
	requireJQ(t)

	dir := t.TempDir()
	git := func(args ...string) string { return gitInFixture(t, dir, args...) }
	write := func(rel, content string) { writeFixtureFile(t, dir, rel, content) }
	git("init", "-q", "-b", "main")
	git("config", "core.trustctime", "false")

	stamp := time.Unix(1_000_000_000, 0)
	touch := func(rel string) {
		t.Helper()
		if err := os.Chtimes(filepath.Join(dir, rel), stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}

	write("internal/a/a.go", "package a\n\nvar A = 10\n")
	touch("internal/a/a.go")
	write("internal/a/a_test.go", "package a\n")
	write("internal/a/sub/s.go", "package sub\n\nvar S = 40\n")
	write("internal/a/sub/s_test.go", "package sub\n")
	write("internal/a/sub2/x.go", "package sub2\n\nvar X = 50\n")
	write("internal/ab/ab.go", "package ab\n\nvar AB = 20\n")
	write("internal/testonly/o.go", "package testonly\n")
	write("internal/testonly/o_test.go", "package testonly\n")
	write("internal/gone/g.go", "package gone\n\nvar G = 1\n")
	write("internal/partgone/keep.go", "package partgone\n")
	write("internal/partgone/drop.go", "package partgone\n\nvar D = 1\n")
	write("internal/shrunk/s.go", "package shrunk\n\nvar X = 1\nvar Y = 2\n")
	write("cmd/x/x.go", "package main\n\nvar X = 60\n")
	write(".gitignore", "internal/ignored/\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	base := git("rev-parse", "HEAD")

	git("checkout", "-q", "-b", "trunk")
	write("internal/trunkonly/t.go", "package trunkonly\n\nvar T = 1\n")
	git("add", "-A")
	git("commit", "-q", "-m", "trunk moves on")
	git("checkout", "-q", "main")
	write("internal/a/sub2/x.go", "package sub2\n\nvar X = 51\n")
	git("add", "-A")
	git("commit", "-q", "-m", "main moves on")

	write("internal/a/a.go", "package a\n\nvar A = 11\n")
	touch("internal/a/a.go")
	write("internal/a/u.go", "package a\n\nvar U = 70\n")
	write("internal/a/a_test.go", "package a\n\n// widened\n")
	write("internal/a/sub/s_test.go", "package sub\n\n// widened\n")
	write("internal/fresh/f.go", "package fresh\n\nvar F = 80\n")
	write("internal/ab/ab.go", "package ab\n\nvar AB = 21\n")
	git("add", "internal/ab/ab.go")
	write("internal/testonly/o_test.go", "package testonly\n\n// widened\n")
	for _, rel := range []string{"internal/gone/g.go", "internal/partgone/drop.go"} {
		if err := os.Remove(filepath.Join(dir, rel)); err != nil {
			t.Fatal(err)
		}
	}
	write("internal/shrunk/s.go", "package shrunk\n\nvar X = 1\n")
	write("cmd/x/x.go", "package main\n\nvar X = 61\n")
	write("internal/ignored/i.go", "package ignored\n\nvar I = 1\n")
	git("config", "core.splitIndex", "true")
	git("config", "splitIndex.maxPercentChange", "0") // every index write re-splits
	git("update-index", "--split-index")

	for key, value := range map[string]string{
		"color.ui":            "always",
		"diff.noprefix":       "true",
		"diff.mnemonicPrefix": "true",
		"diff.external":       "false",
		"diff.relative":       "true",
	} {
		git("config", key, value)
	}
	// The last index write is back-dated to the edit's second, the
	// state a fast editor-then-script sequence leaves.
	touch(".git/index")

	indexPath := filepath.Join(dir, ".git", "index")
	indexBefore, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("reading fixture index: %v", err)
	}
	gitDirBefore := gitDirEntries(t, dir)

	stubDir, stubEnv := stubGremlins(t)
	for _, run := range []string{"a", "ab", "fresh"} {
		stubReply(t, stubDir, run)
	}
	env := append([]string{"MUTATE_DIFF_BASE=trunk", "MUTATE_DIFF_COEFFICIENT=7"}, stubEnv...)
	got := runMutateDiff(t, filepath.Join(dir, "internal"), env...)

	// passes: the caller's diff.relative applies to the pass-through
	// calls, which read the operator's index, where the untracked
	// files are absent.
	invocation := "diff=" + base + " workers=1 coefficient=7 index=unset bare=ok,ok"
	wantRuns := []string{
		"cwd=internal/a/ path=<top>/internal/a " + invocation + " passes=a.go a_test.go sub/s_test.go sub2/x.go|a.go a_test.go sub/s_test.go sub2/x.go sees=a.go:+3 a_test.go:+2,2 sub/s_test.go:+2,2 sub2/x.go:+3 u.go:+1,3",
		"cwd=internal/ab/ path=<top>/internal/ab " + invocation + " passes=ab.go|ab.go sees=ab.go:+3",
		"cwd=internal/fresh/ path=<top>/internal/fresh " + invocation + " passes=| sees=f.go:+1,3",
	}
	if diff := cmp.Diff(wantRuns, stubLog(t, stubDir)); diff != "" {
		t.Errorf("gremlins runs (-want +got):\n%s\n--- output ---\n%s", diff, got)
	}
	wantTestOnly := []string{
		"TEST-ONLY ./internal/a/sub",
		"gremlins unleash --workers 1 --timeout-coefficient 7 ./internal/a/sub",
		"TEST-ONLY ./internal/testonly",
		"gremlins unleash --workers 1 --timeout-coefficient 7 ./internal/testonly",
	}
	var testOnly []string
	for _, line := range strings.Split(got, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TEST-ONLY") || strings.HasPrefix(line, "gremlins unleash") {
			testOnly = append(testOnly, line)
		}
	}
	if diff := cmp.Diff(wantTestOnly, testOnly); diff != "" {
		t.Errorf("test-only packages (-want +got):\n%s\n--- output ---\n%s", diff, got)
	}
	if diff := cmp.Diff([]string{mutateDiffPassLine}, markerLines(got, "mutate-diff: no surviving")); diff != "" {
		t.Errorf("a clean report must end in the advisory pass (-want +got):\n%s\n--- output ---\n%s", diff, got)
	}

	indexAfter, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("reading fixture index after the run: %v", err)
	}
	if !bytes.Equal(indexBefore, indexAfter) {
		t.Error("mutate-diff.sh rewrote the operator's index; untracked files must join the diff through a copy")
	}
	if diff := cmp.Diff(gitDirBefore, gitDirEntries(t, dir)); diff != "" {
		t.Errorf("mutate-diff.sh changed the operator's git directory (-before +after):\n%s", diff)
	}
	if status := git("status", "--porcelain", "--", "internal/a/u.go"); status != "?? internal/a/u.go" {
		t.Errorf("the untracked file must stay untracked; git status reports %q", status)
	}
}

// TestMutateDiff_ChecksGremlinsAgainstGitsDiff pins how reports are
// read and totalled across runs. For internal/c the stand-in reports a
// SKIPPED mutant on every line, so exactly the changed lines surface as
// SKIPPED-CHANGED; mutants it ran on unchanged lines — killed, lived
// or not covered — surface as MUTATED-UNCHANGED; LIVED mutants surface
// as survivors; and a mutant on a changed line that gremlins did not
// skip raises nothing. c's changes are a two-line hunk, a pure
// deletion that shifts every later line, and two single-line hunks,
// in a file whose name git would quote or pad, under a caller config
// that would merge nearby hunks. internal/d adds a second run to the
// totals, and a file moved into it from another directory, which
// gremlins can only see as new, counts as wholly changed. The caller's
// config is otherwise git's default, so what gremlins' own diff call
// returns is the shim's doing alone, and GIT_DIFF_OPTS asks for context
// lines both readers must refuse. No file is untracked, and a tracked
// file whose timestamp moved without its content is left for git to
// refresh: the script must read the diff without writing the index.
func TestMutateDiff_ChecksGremlinsAgainstGitsDiff(t *testing.T) {
	t.Parallel()
	requireJQ(t)

	dir := t.TempDir()
	git := func(args ...string) string { return gitInFixture(t, dir, args...) }
	git("init", "-q", "-b", "main")
	const cFile = "c ü.go"
	before := numberedLines(40)
	writeFixtureFile(t, dir, "internal/c/"+cFile, strings.Join(before, "\n")+"\n")
	writeFixtureFile(t, dir, "internal/c/keep.go", "package c\n")
	writeFixtureFile(t, dir, "internal/d/d.go", "package d\n\nvar D = 1\n")
	writeFixtureFile(t, dir, "internal/e/moved.go", "package d\n\nvar M = 1\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	base := git("rev-parse", "HEAD")
	git("config", "diff.interHunkContext", "10")
	git("mv", "internal/e/moved.go", "internal/d/moved.go")

	after := append([]string(nil), before...)
	after[2], after[3] = "// changed 3", "// changed 4"
	after[20], after[31] = "// changed old 21", "// changed old 32"
	after = append(after[:9], after[10:]...) // drop old line 10
	writeFixtureFile(t, dir, "internal/c/"+cFile, strings.Join(after, "\n")+"\n")
	writeFixtureFile(t, dir, "internal/d/d.go", "package d\n\nvar D = 2\n")
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "internal", "c", "keep.go"), later, later); err != nil {
		t.Fatal(err)
	}

	stubDir, stubEnv := stubGremlins(t)
	var cMutations []stubMutation
	for line := 1; line <= len(after); line++ {
		cMutations = append(cMutations, stubMutation{Type: "CONDITIONALS_BOUNDARY", Status: "SKIPPED", Line: line, Column: 1})
	}
	cMutations = append(cMutations,
		stubMutation{Type: "CONDITIONALS_NEGATION", Status: "LIVED", Line: 3, Column: 9},
		stubMutation{Type: "CONDITIONALS_NEGATION", Status: "KILLED", Line: 10, Column: 9},
		stubMutation{Type: "CONDITIONALS_NEGATION", Status: "LIVED", Line: 11, Column: 9},
		stubMutation{Type: "CONDITIONALS_NEGATION", Status: "NOT COVERED", Line: 12, Column: 9},
		stubMutation{Type: "CONDITIONALS_NEGATION", Status: "NOT COVERED", Line: 20, Column: 9},
	)
	stubReply(t, stubDir, "c", stubFile{cFile, cMutations})
	stubReply(t, stubDir, "d",
		stubFile{"d.go", []stubMutation{
			{Type: "ARITHMETIC_BASE", Status: "LIVED", Line: 3, Column: 5},
			{Type: "ARITHMETIC_BASE", Status: "SKIPPED", Line: 3, Column: 7},
		}},
		stubFile{"moved.go", []stubMutation{{Type: "ARITHMETIC_BASE", Status: "KILLED", Line: 3, Column: 9}}},
	)

	indexPath := filepath.Join(dir, ".git", "index")
	indexBefore, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("reading fixture index: %v", err)
	}

	got := runMutateDiff(t, dir, append([]string{"MUTATE_DIFF_BASE=" + base, "GIT_DIFF_OPTS=--unified=3"}, stubEnv...)...)

	var sees []string
	for _, line := range stubLog(t, stubDir) {
		cwd, _, _ := strings.Cut(strings.TrimPrefix(line, "cwd="), " ")
		_, seen, _ := strings.Cut(line, " sees=")
		sees = append(sees, cwd+" "+seen)
	}
	wantSees := []string{
		"internal/c/ " + cFile + ":+3,2 " + cFile + ":+9,0 " + cFile + ":+20 " + cFile + ":+31",
		"internal/d/ d.go:+3 moved.go:+1,3",
	}
	if diff := cmp.Diff(wantSees, sees); diff != "" {
		t.Errorf("what gremlins' diff call returned (-want +got):\n%s\n--- output ---\n%s", diff, got)
	}

	checks := []struct {
		marker string
		want   []string
	}{
		{"SKIPPED-CHANGED", []string{
			"SKIPPED-CHANGED " + cFile + ":3:1 (CONDITIONALS_BOUNDARY) in ./internal/c",
			"SKIPPED-CHANGED " + cFile + ":4:1 (CONDITIONALS_BOUNDARY) in ./internal/c",
			"SKIPPED-CHANGED " + cFile + ":20:1 (CONDITIONALS_BOUNDARY) in ./internal/c",
			"SKIPPED-CHANGED " + cFile + ":31:1 (CONDITIONALS_BOUNDARY) in ./internal/c",
			"SKIPPED-CHANGED d.go:3:7 (ARITHMETIC_BASE) in ./internal/d",
		}},
		{"MUTATED-UNCHANGED", []string{
			"MUTATED-UNCHANGED " + cFile + ":10:9 (CONDITIONALS_NEGATION) in ./internal/c",
			"MUTATED-UNCHANGED " + cFile + ":11:9 (CONDITIONALS_NEGATION) in ./internal/c",
			"MUTATED-UNCHANGED " + cFile + ":12:9 (CONDITIONALS_NEGATION) in ./internal/c",
		}},
		{"SURVIVOR LIVED", []string{
			"SURVIVOR LIVED " + cFile + ":3:9 (CONDITIONALS_NEGATION) in ./internal/c",
			"SURVIVOR LIVED " + cFile + ":11:9 (CONDITIONALS_NEGATION) in ./internal/c",
			"SURVIVOR LIVED d.go:3:5 (ARITHMETIC_BASE) in ./internal/d",
		}},
		{"mutate-diff: ", []string{
			"mutate-diff: changed internal/ Go lines vs " + base + " (base " + base[:12] + "), in 2 run(s):",
			"mutate-diff: WARNING gremlins' diff filter disagreed with git's on 8 mutant(s) (SKIPPED-CHANGED / MUTATED-UNCHANGED above).",
			"mutate-diff: 3 surviving mutant(s) (LIVED) — ADVISORY.",
		}},
	}
	for _, c := range checks {
		if diff := cmp.Diff(c.want, markerLines(got, c.marker)); diff != "" {
			t.Errorf("%s lines (-want +got):\n%s\n--- output ---\n%s", c.marker, diff, got)
		}
	}

	indexAfter, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("reading fixture index after the run: %v", err)
	}
	if !bytes.Equal(indexBefore, indexAfter) {
		t.Error("mutate-diff.sh rewrote the operator's index while reading the diff")
	}
}

// TestMutateDiff_ReportsEachRunsOutcome pins how a run's outcome
// reaches the summary. A run gremlins exits non-zero from, or that
// leaves no readable report, is named FAILED; one that skipped a
// mutant on a changed line is named for it; either withholds the
// advisory pass, even with no survivor. A readable report is still
// read when gremlins exits non-zero, so its survivors are not lost. A
// run with no report at exit 0 found nothing to mutate and costs the
// pass nothing. A run of internal/d follows internal/c each time and
// must stay unmarked unless the case says otherwise. The script exits
// 0 throughout.
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
			name: "gremlins exits non-zero with a readable report",
			setup: func(t *testing.T, stubDir string) {
				t.Helper()
				stubReply(t, stubDir, "c", stubFile{"c.go", []stubMutation{{Type: "CONDITIONALS_BOUNDARY", Status: "LIVED", Line: 3, Column: 5}}})
				writeFixtureFile(t, stubDir, "c.exit", "3")
			},
			wantMarked: []string{
				"FAILED ./internal/c — gremlins exited 3; its report is read below.",
				"SURVIVOR LIVED c.go:3:5 (CONDITIONALS_BOUNDARY) in ./internal/c",
			},
			wantSummary: []string{failedSummary, "mutate-diff: 1 surviving mutant(s) (LIVED) — ADVISORY."},
		},
		{
			name: "gremlins exits non-zero without a report",
			setup: func(t *testing.T, stubDir string) {
				t.Helper()
				writeFixtureFile(t, stubDir, "c.exit", "3")
			},
			wantMarked:  []string{"FAILED ./internal/c — gremlins exited 3 and left no readable report; nothing in this run was tested."},
			wantSummary: []string{failedSummary, incomplete},
		},
		{
			name: "the report is not a JSON object",
			setup: func(t *testing.T, stubDir string) {
				t.Helper()
				writeFixtureFile(t, stubDir, "c.json", "null")
			},
			wantMarked:  []string{"FAILED ./internal/c — gremlins exited 0 and left no readable report; nothing in this run was tested."},
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
		{
			name: "a changed line was skipped and nothing survived",
			setup: func(t *testing.T, stubDir string) {
				t.Helper()
				stubReply(t, stubDir, "c", stubFile{"c.go", []stubMutation{{Type: "CONDITIONALS_BOUNDARY", Status: "SKIPPED", Line: 3, Column: 1}}})
			},
			wantMarked:  []string{"SKIPPED-CHANGED c.go:3:1 (CONDITIONALS_BOUNDARY) in ./internal/c"},
			wantSummary: []string{"mutate-diff: WARNING gremlins' diff filter disagreed with git's on 1 mutant(s) (SKIPPED-CHANGED / MUTATED-UNCHANGED above).", incomplete},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			git := func(args ...string) string { return gitInFixture(t, dir, args...) }
			git("init", "-q", "-b", "main")
			writeFixtureFile(t, dir, "internal/c/c.go", "package c\n\nvar C = 1\n")
			writeFixtureFile(t, dir, "internal/d/d.go", "package d\n\nvar D = 1\n")
			git("add", "-A")
			git("commit", "-q", "-m", "base")
			base := git("rev-parse", "HEAD")
			writeFixtureFile(t, dir, "internal/c/c.go", "package c\n\nvar C = 2\n")
			writeFixtureFile(t, dir, "internal/d/d.go", "package d\n\nvar D = 2\n")

			stubDir, stubEnv := stubGremlins(t)
			stubReply(t, stubDir, "d")
			tc.setup(t, stubDir)
			got := runMutateDiff(t, dir, append([]string{"MUTATE_DIFF_BASE=" + base}, stubEnv...)...)

			var marked []string
			for _, marker := range []string{"FAILED ./", "NO MUTANTS ./", "SKIPPED-CHANGED", "MUTATED-UNCHANGED", "SURVIVOR LIVED"} {
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
func TestMutateDiff_TestOnlyChangeMutatesNothing(t *testing.T) {
	t.Parallel()
	requireJQ(t)

	dir := t.TempDir()
	git := func(args ...string) string { return gitInFixture(t, dir, args...) }
	git("init", "-q", "-b", "main")
	writeFixtureFile(t, dir, "internal/t/t.go", "package t\n")
	writeFixtureFile(t, dir, "internal/t/t_test.go", "package t\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	base := git("rev-parse", "HEAD")
	writeFixtureFile(t, dir, "internal/t/t_test.go", "package t\n\n// widened\n")

	stubDir, stubEnv := stubGremlins(t)
	got := runMutateDiff(t, dir, append([]string{"MUTATE_DIFF_BASE=" + base}, stubEnv...)...)

	if runs := stubLog(t, stubDir); len(runs) != 0 {
		t.Errorf("a test-only diff must start no gremlins run; got %q\n--- output ---\n%s", runs, got)
	}
	want := []string{
		"mutate-diff: no changed internal/ Go code lines vs " + base + " — nothing to mutate.",
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
			git := func(args ...string) string { return gitInFixture(t, dir, args...) }
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

	write := func(rel, content string) { writeFixtureFile(t, dir, rel, content) }

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

	got := runMutateDiff(t, dir, "MUTATE_DIFF_BASE="+base)

	// Find the wrapper's own survivor marker LINE and assert its parts
	// appear together on it. "SURVIVOR LIVED" is the only token unique to
	// the wrapper — calc.go / CONDITIONALS_BOUNDARY / the package path all
	// also appear in gremlins' raw stdout, so four independent Contains
	// checks would pass even if the wrapper emitted no marker of its own.
	// Asserting the same-line shape pins the JSON-derived marker, not
	// gremlins' wording (per CLAUDE.md "substring assertions are not
	// structural assertions"). The volatile line:col is deliberately not
	// pinned.
	markers := markerLines(got, "SURVIVOR LIVED")
	if len(markers) == 0 {
		t.Fatalf("no SURVIVOR marker line in mutate-diff output\n--- output ---\n%s", got)
	}
	marker := markers[0]
	for _, want := range []string{"calc.go", "(CONDITIONALS_BOUNDARY)", "in ./internal/mutdemo"} {
		if !strings.Contains(marker, want) {
			t.Errorf("survivor marker line missing %q\nmarker: %q\n--- full output ---\n%s", want, marker, got)
		}
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
// cmd/. A boundary survivor must surface for each edited line and the
// untracked file, and for nothing else; the subpackage rides
// internal/a's run; the test-only package is named rather than
// mutated; gremlins' filter agrees with git's diff; and the operator's
// index and status are unchanged by the run.
func TestMutateDiff_MutatesOnlyChangedLines(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("gremlins"); err != nil {
		t.Skip("gremlins not on PATH; mutate-diff is advisory dev tooling (mutate-hunt is workflow_dispatch-only)")
	}
	requireJQ(t)

	dir := t.TempDir()
	git := func(args ...string) string { return gitInFixture(t, dir, args...) }
	write := func(rel, content string) { writeFixtureFile(t, dir, rel, content) }
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

	// git status refreshes the index's stat data, so it runs before the
	// index is read.
	statusBefore := git("status", "--porcelain")
	indexPath := filepath.Join(dir, ".git", "index")
	indexBefore, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("reading fixture index: %v", err)
	}

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
		"u.go:3 (CONDITIONALS_BOUNDARY) in ./internal/a",
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
	for _, marker := range []string{"SKIPPED-CHANGED", "MUTATED-UNCHANGED", "FAILED ./"} {
		if lines := markerLines(got, marker); len(lines) != 0 {
			t.Errorf("a correctly scoped run reported %s: %q\n--- output ---\n%s", marker, lines, got)
		}
	}

	indexAfter, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("reading fixture index after the run: %v", err)
	}
	if !bytes.Equal(indexBefore, indexAfter) {
		t.Error("mutate-diff.sh rewrote the operator's index; untracked files must join the diff through a copy")
	}
	if statusAfter := git("status", "--porcelain"); statusAfter != statusBefore {
		t.Errorf("git status changed across the run:\nbefore:\n%s\nafter:\n%s", statusBefore, statusAfter)
	}
}
