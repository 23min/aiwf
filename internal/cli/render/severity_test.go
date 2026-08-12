package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// renderSeverityFixture builds a git repo carrying one epic with an
// empty `## Goal` (its other two sections carry prose, so exactly one
// entity-body-empty finding fires), under the aiwf.yaml the caller
// supplies, and renders the HTML site into it. Returns the site
// directory.
func renderSeverityFixture(t *testing.T, config string) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(rel, content string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	write("aiwf.yaml", config)
	write("work/epics/E-0001-probe-epic/epic.md",
		"---\nid: E-0001\ntitle: Probe epic\nstatus: proposed\n---\n\n"+
			"## Goal\n\n## Scope\n\nReal prose.\n\n## Out of scope\n\nReal prose.\n")
	if out, err := testutil.RunGit(root, "add", "-A"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := testutil.RunGit(root, "commit", "-m", "chore: fixture"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	if code := RunSite(root, "html", "site", "", true, false); code != cliutil.ExitOK {
		t.Fatalf("RunSite = %d, want %d", code, cliutil.ExitOK)
	}
	return filepath.Join(root, "site")
}

// TestRunSite_RollupCountsHonorTheConsumersSeverityPolicy pins the
// rendered site's agreement with the gate, in both directions.
//
// A published page outlives the run that made it, so reporting a
// finding below the severity `aiwf check` applies to it is an overclaim
// with a long half-life — and unlike a terminal surface, the reader
// cannot re-run the check to find out.
//
// The rollup line is the assertion because it is what RunSite's own
// findings slice feeds; the status page's tables are built from a
// separately-escalated report, so an assertion there would pass even
// with this surface's policy dropped entirely.
func TestRunSite_RollupCountsHonorTheConsumersSeverityPolicy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		config string
		want   string
	}{
		{
			name:   "tdd.strict raises the finding to an error",
			config: "tdd:\n  strict: true\n",
			want:   "Findings: 1 error(s), 0 warning(s).",
		},
		{
			// The negative control. Without it a renderer that escalated
			// unconditionally would satisfy the case above, and every
			// consumer that never set the knob would get a page calling
			// its warnings errors.
			name:   "without the knob it stays a warning",
			config: "tdd:\n  strict: false\n",
			want:   "Findings: 0 error(s), 1 warning(s).",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			site := renderSeverityFixture(t, tc.config)
			index, err := os.ReadFile(filepath.Join(site, "index.html"))
			if err != nil {
				t.Fatalf("reading index.html: %v", err)
			}
			if !strings.Contains(string(index), tc.want) {
				t.Errorf("index.html rollup does not read %q — the page counted the finding at a severity `aiwf check` does not agree with", tc.want)
			}
		})
	}
}
