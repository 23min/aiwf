package policies

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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

	bin, err := exec.LookPath("golangci-lint")
	if err != nil {
		if os.Getenv("AIWF_REQUIRE_GOLANGCI") != "" {
			t.Fatalf("AIWF_REQUIRE_GOLANGCI is set but golangci-lint is not on PATH: %v", err)
		}
		t.Skip("golangci-lint not on PATH; set AIWF_REQUIRE_GOLANGCI=1 to require it (the CI lint job does)")
	}

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
			want: []string{"filepathJoin"},
		},
	}

	// Subtests run serially, not t.Parallel: each shells a heavyweight
	// golangci-lint process, and concurrent runs contend on the shared
	// build/lint cache (an early version flaked here). A unique module
	// path per row removes any cross-row cache-key aliasing on top.
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			dir := t.TempDir()
			mod := "module fixture_" + strings.ReplaceAll(row.name, "-", "_") + "\n\ngo 1.24\n"
			mustWrite(t, filepath.Join(dir, "go.mod"), mod)
			mustWrite(t, filepath.Join(dir, "bad.go"), row.code)

			cmd := exec.Command(bin, "run", "--config", cfg, "./...")
			cmd.Dir = dir
			// Non-zero exit is expected: findings are present. We assert
			// on the output, not the exit code.
			out, _ := cmd.CombinedOutput()

			for _, w := range row.want {
				if !strings.Contains(string(out), w) {
					t.Errorf("rule %s did not fire: golangci-lint output lacked %q — the config rule is dormant, disabled, or dropped from the enable list (a G-0264-class vacuous chokepoint).\n--- golangci-lint output ---\n%s", row.name, w, out)
				}
			}
		})
	}
}
