package policies

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// golangciLockPath is the start-up lock golangci-lint acquires unless
// --allow-parallel-runners is passed: os.TempDir()/golangci-lint.lock.
// The path is keyed to the temp dir, not to the lint cache — scoping
// GOLANGCI_LINT_CACHE alone leaves two instances contending here.
func golangciLockPath() string {
	return filepath.Join(os.TempDir(), "golangci-lint.lock")
}

// requireGolangci resolves the golangci-lint binary for a test that
// needs to run one, applying the fail-closed contract: the CI lint job
// sets AIWF_REQUIRE_GOLANGCI=1, which turns absence into a failure so a
// silently-skipped chokepoint is caught. Everywhere else — the test job,
// a local `go test` without golangci-lint — it skips.
func requireGolangci(t *testing.T) string {
	t.Helper()

	bin, err := exec.LookPath("golangci-lint")
	if err != nil {
		if os.Getenv("AIWF_REQUIRE_GOLANGCI") != "" {
			t.Fatalf("AIWF_REQUIRE_GOLANGCI is set but golangci-lint is not on PATH: %v", err)
		}
		t.Skip("golangci-lint not on PATH; set AIWF_REQUIRE_GOLANGCI=1 to require it (the CI lint job does)")
	}
	return bin
}

// golangciFixtureCmd builds the golangci-lint invocation the firing
// harness runs against a single-file fixture module.
//
// Two isolation measures, both load-bearing for a suite that shares a
// machine with other golangci-lint processes — this repo routinely
// carries several worktrees, and `make lint`, the pre-push hook and an
// editor integration each spawn one:
//
//   - --allow-parallel-runners. Otherwise golangci-lint takes an
//     exclusive lock on golangciLockPath() at start-up, waits briefly for
//     it, and then exits with "parallel golangci-lint is running" —
//     carrying no findings, which this harness would read as every rule
//     being dormant at once. The flag is the documented way to decline
//     that lock; where the lock file lives is not documented at all,
//     which is why the flag is preferred over scoping the child's TMPDIR
//     to move it. --allow-serial-runners queues on the lock instead of
//     refusing, which trades a spurious failure for an unbounded wait on
//     however many runners the machine has.
//   - a private GOLANGCI_LINT_CACHE, so a run is hermetic and cannot be
//     perturbed by another instance's cache state. An inherited
//     environment points every instance at the machine-global default.
//     The measured cost is a few seconds per run against a one-file
//     fixture, once per lint job.
//
// This deliberately diverges from the Makefile's per-worktree cache,
// which respects a pre-set GOLANGCI_LINT_CACHE: honoring one here would
// reintroduce the shared cache the private dir exists to avoid.
func golangciFixtureCmd(bin, cfg, workDir, cacheDir string) *exec.Cmd {
	cmd := exec.Command(bin, "run", "--config", cfg, "--allow-parallel-runners", "./...")
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "GOLANGCI_LINT_CACHE="+cacheDir)
	return cmd
}

// golangciRefusedForConcurrency reports whether out is golangci-lint
// declining to start because another instance holds the start-up lock.
//
// The literal is upstream's, so it is pinned against the real binary
// rather than against a second copy of itself:
// TestGolangciFixtureCmd_RefusesWithoutTheFlag in
// golangci_firing_isolation_test.go provokes a genuine refusal and feeds
// the bytes here. If golangci-lint rewords the message this predicate
// goes permanently false, and that test is what says so.
func golangciRefusedForConcurrency(out string) bool {
	return strings.Contains(out, "parallel golangci-lint is running")
}

// findingLinePrefix matches the `path:line:col: ` that opens each
// golangci-lint finding line.
var findingLinePrefix = regexp.MustCompile(`^\S+\.go:\d+:\d+: `)

// golangciFindingMessages returns the message half of each finding line
// in out, dropping the path that opens it.
//
// Assertions run against these rather than the raw output because
// golangci-lint echoes the fixture's path, and t.TempDir() derives that
// path from the subtest name. A row asserting on "filepathJoin" while
// running under a directory named for the gocritic-filepathJoin subtest
// otherwise matches its own working directory, and passes with the
// linter disabled — the vacuous chokepoint this harness exists to be.
func golangciFindingMessages(out string) string {
	var b strings.Builder
	for _, line := range strings.Split(out, "\n") {
		if loc := findingLinePrefix.FindStringIndex(line); loc != nil {
			b.WriteString(line[loc[1]:])
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// judgeGolangciOutput decides what one run says about one config rule,
// returning "" when the rule fired and a failure message otherwise.
//
// The two ways a run yields no finding are different claims, and the
// harness's value depends on telling them apart: a refused run ended
// before the config was applied and is evidence about the machine, while
// a completed run that found nothing is evidence about the rule. This is
// a pure function so both verdicts are reachable from a test — the
// refusal arm is unreachable through the harness itself once the
// isolation measures work, and Go does not instrument _test.go files, so
// the coverage gate cannot see it either.
func judgeGolangciOutput(rule string, want []string, out string) string {
	if golangciRefusedForConcurrency(out) {
		return fmt.Sprintf(
			"golangci-lint declined to start: another instance holds %s. This is evidence about the machine, not about rule %s — the run ended before the config was applied, so it says nothing either way about whether that rule fires. The harness passes --allow-parallel-runners and a private cache to avoid exactly this, so a refusal here means that isolation regressed.\n--- golangci-lint output ---\n%s",
			golangciLockPath(), rule, out)
	}

	messages := golangciFindingMessages(out)
	for _, w := range want {
		if !strings.Contains(messages, w) {
			return fmt.Sprintf(
				"rule %s did not fire: golangci-lint reported no finding containing %q — the config rule is dormant, disabled, or dropped from the enable list (a G-0264-class vacuous chokepoint).\n--- golangci-lint output ---\n%s",
				rule, w, out)
		}
	}
	return ""
}

// TestGolangciConfigRulesFire is the execution firing harness for the
// golangci-lint config rules (M-0170/AC-2). For each guarded rule it
// builds a self-contained temp module that violates exactly that rule,
// runs golangci-lint against the repo's .golangci.yml, and asserts the
// rule actually fires. It is the firing-evidence mechanism for the
// golangci-config surface — the analog of firing_fixture_presence for
// the internal/policies Go policies (G-0264 / G-0259).
//
// Why execution, not a config-structural check: a rule can be present in
// config yet match nothing — the dormant `^panic\(` / `^os\.Exit\(`
// forbidigo patterns that motivated G-0264 matched zero sites under
// forbidigo v2. Only running golangci-lint proves the rule fires. This
// generalizes M-0167/AC-2's structural gocritic guard (which stays as
// the cheap always-on test in the test job) to real execution across
// forbidigo and gocritic.
//
// CI wiring & fail-closed: golangci-lint lives only in the lint job, not
// the test job. The lint-job step sets AIWF_REQUIRE_GOLANGCI=1, which
// turns "golangci-lint not on PATH" into a hard failure there — catching
// a silently-skipped chokepoint. Everywhere else (the test job, local
// `go test` without golangci-lint) it skips gracefully.
//
// Each fixture is a non-test `bad.go`, so forbidigo's `_test.go`
// exclusion (AC-1) does not suppress it — the harness exercises the
// production-code path the rule actually guards.
func TestGolangciConfigRulesFire(t *testing.T) {
	t.Parallel()

	bin := requireGolangci(t)

	cfg := filepath.Join(repoRoot(t), ".golangci.yml")

	rows := []struct {
		name string
		code string   // fixture body, written to bad.go
		want []string // substrings that together prove THIS rule fired
	}{
		{
			name: "forbidigo-panic",
			code: "package fixture\n\n" +
				"// Boom is library code that must not panic.\n" +
				"func Boom() {\n\tpanic(\"library code must not panic\")\n}\n",
			want: []string{"(forbidigo)", "panic"},
		},
		{
			name: "forbidigo-os-exit",
			code: "package fixture\n\n" +
				"import \"os\"\n\n" +
				"// Die is library code that must not call os.Exit.\n" +
				"func Die() {\n\tos.Exit(1)\n}\n",
			want: []string{"(forbidigo)", "os.Exit"},
		},
		{
			name: "gocritic-filepathJoin",
			code: "package fixture\n\n" +
				"import \"path/filepath\"\n\n" +
				"// Bad embeds a separator in a filepath.Join arg.\n" +
				"func Bad(root string) string {\n\treturn filepath.Join(root, \"work/epics\")\n}\n",
			want: []string{"filepathJoin", "(gocritic)"},
		},
	}

	// Subtests run serially, not t.Parallel: each shells a heavyweight
	// golangci-lint process, and three at once buys little on fixtures
	// this small. Each row gets its own cache and its own module path, so
	// neither the lint cache nor a cache key is shared between them.
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			dir := t.TempDir()
			mod := "module fixture_" + strings.ReplaceAll(row.name, "-", "_") + "\n\ngo 1.24\n"
			mustWrite(t, filepath.Join(dir, "go.mod"), mod)
			mustWrite(t, filepath.Join(dir, "bad.go"), row.code)

			cmd := golangciFixtureCmd(bin, cfg, dir, t.TempDir())
			// Non-zero exit is expected: findings are present. We assert
			// on the output, not the exit code.
			out, _ := cmd.CombinedOutput()

			if verdict := judgeGolangciOutput(row.name, row.want, string(out)); verdict != "" {
				t.Error(verdict)
			}
		})
	}
}

// TestGolangciFixtureCmd_CarriesBothIsolationMeasures pins the two
// measures at the seam where they are declared, so the pin holds in the
// test job too — golangci-lint lives only in the lint job, so the
// behavioral test in golangci_firing_isolation_test.go cannot run there.
func TestGolangciFixtureCmd_CarriesBothIsolationMeasures(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	cmd := golangciFixtureCmd("golangci-lint", "/cfg/.golangci.yml", t.TempDir(), cacheDir)

	if !slices.Contains(cmd.Args, "--allow-parallel-runners") {
		t.Errorf("the fixture command must pass --allow-parallel-runners, or a concurrent golangci-lint anywhere on the machine refuses this run at the start-up lock (%s); args: %q",
			golangciLockPath(), cmd.Args)
	}

	// Resolved the way the child resolves it, not by presence: os/exec
	// uses the last value for a duplicated key, so a later entry pointing
	// at a shared cache would win while a presence check stayed green.
	if got, ok := resolvedEnv(cmd.Env, "GOLANGCI_LINT_CACHE"); !ok || got != cacheDir {
		t.Errorf("the child must resolve GOLANGCI_LINT_CACHE to the caller's dir %q, or the run is not hermetic; resolved to %q (present=%v)", cacheDir, got, ok)
	}
}

// resolvedEnv returns the value a child would see for key, applying
// os/exec's documented rule that the last duplicate wins.
func resolvedEnv(env []string, key string) (string, bool) {
	prefix := key + "="
	value, found := "", false
	for _, entry := range env {
		if after, ok := strings.CutPrefix(entry, prefix); ok {
			value, found = after, true
		}
	}
	return value, found
}

// TestJudgeGolangciOutput pins the harness's verdict on each shape of
// output a run can produce. This is the seam the fix turns on: deleting
// the refusal arm leaves every other test green, because the harness's
// own path never reaches it once the isolation measures work.
func TestJudgeGolangciOutput(t *testing.T) {
	t.Parallel()

	const (
		refusal   = "Error: parallel golangci-lint is running\nThe command is terminated due to an error: parallel golangci-lint is running\n"
		fired     = "/tmp/x/bad.go:5:2: use of `panic` forbidden because \"library code must not panic\" (forbidigo)\n"
		otherRule = "/tmp/x/bad.go:1:1: package-comments: should have a package comment (revive)\n"
		// golangci-lint echoes the fixture path, and t.TempDir() derives
		// it from the subtest name — so the wanted token can appear in a
		// finding line while no rule reported it.
		tokenOnlyInPath = "/tmp/TestGolangciConfigRulesFiregocritic-filepathJoin99/001/bad.go:1:1: package-comments: should have a package comment (revive)\n"
	)

	tests := []struct {
		name string
		want []string
		out  string
		// substrings the verdict must contain; empty means "rule fired,
		// no verdict at all"
		wantVerdict []string
	}{
		{
			name: "rule fired",
			want: []string{"(forbidigo)", "panic"},
			out:  fired,
		},
		{
			name:        "refusal is judged as a refusal, not as a dormant rule",
			want:        []string{"(forbidigo)", "panic"},
			out:         refusal,
			wantVerdict: []string{"declined to start", golangciLockPath(), "says nothing either way"},
		},
		{
			name:        "a completed run that found nothing is judged as a dormant rule",
			want:        []string{"(forbidigo)"},
			out:         otherRule,
			wantVerdict: []string{"did not fire", "dormant, disabled, or dropped from the enable list"},
		},
		{
			name:        "a token present only in the echoed path does not count as a finding",
			want:        []string{"filepathJoin", "(gocritic)"},
			out:         tokenOnlyInPath,
			wantVerdict: []string{"did not fire", "filepathJoin"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := judgeGolangciOutput("rule-under-test", tc.want, tc.out)

			if len(tc.wantVerdict) == 0 {
				if got != "" {
					t.Errorf("want no verdict (the rule fired); got:\n%s", got)
				}
				return
			}
			if got == "" {
				t.Fatalf("want a verdict containing %q; got none", tc.wantVerdict)
			}
			for _, w := range tc.wantVerdict {
				if !strings.Contains(got, w) {
					t.Errorf("verdict must contain %q; got:\n%s", w, got)
				}
			}
		})
	}
}

// TestJudgeGolangciOutput_RefusalIsNeverBlamedOnTheConfig states the
// property the two verdicts must never confuse, independently of how
// either is worded: a refused run must not be reported with the message
// reserved for a rule that genuinely stopped firing.
func TestJudgeGolangciOutput_RefusalIsNeverBlamedOnTheConfig(t *testing.T) {
	t.Parallel()

	refusal := judgeGolangciOutput("rule-under-test", []string{"(forbidigo)"}, "Error: parallel golangci-lint is running\n")
	dormant := judgeGolangciOutput("rule-under-test", []string{"(forbidigo)"}, "0 issues.\n")

	if refusal == "" || dormant == "" {
		t.Fatalf("both shapes must produce a verdict; refusal=%q dormant=%q", refusal, dormant)
	}
	if refusal == dormant {
		t.Errorf("a refused run and a dormant rule must not receive the same verdict; both were:\n%s", refusal)
	}
	if strings.Contains(refusal, "did not fire") {
		t.Errorf("the refusal verdict must not carry the dormant-rule phrasing — the run ended before the config was applied, so it is no evidence about the rule; got:\n%s", refusal)
	}
}
