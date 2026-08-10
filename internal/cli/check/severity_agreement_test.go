package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// severityOf returns the severity `aiwf check` reported for code in a
// captured JSON envelope, and whether the code was reported at all.
func severityOf(t *testing.T, raw []byte, code string) (check.Severity, bool) {
	t.Helper()
	var env struct {
		Findings []check.Finding `json:"findings"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("check output %q is not a findings envelope: %v", raw, err)
	}
	for i := range env.Findings {
		if env.Findings[i].Code == code {
			return env.Findings[i].Severity, true
		}
	}
	return "", false
}

// writeEpicFixture plants one epic carrying the given area tag, with
// every load-bearing body section filled so entity-body-empty stays
// silent and the area axis is the only thing under test.
func writeEpicFixture(t *testing.T, root, area string) {
	t.Helper()
	dir := filepath.Join(root, "work", "epics", "E-0001-probe-epic")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir epic dir: %v", err)
	}
	body := "---\nid: E-0001\ntitle: Probe epic\nstatus: proposed\narea: " + area + "\n---\n\n" +
		"## Goal\n\nReal prose.\n\n## Scope\n\nReal prose.\n\n## Out of scope\n\nReal prose.\n"
	if err := os.WriteFile(filepath.Join(dir, "epic.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write epic.md: %v", err)
	}
}

// TestCheckAndFastAgreeOnAreaUnknownSeverity is G-0567's measured case,
// stated as the property it violates: `aiwf check --fast` and
// `aiwf check` must report one finding at one severity.
//
// Measured before the shared seam, on a tree with `areas.required: true`
// and an entity tagged with an undeclared area:
//
//	aiwf check          error   area-unknown   exit 1
//	aiwf check --fast   warning area-unknown   exit 0
//
// --fast ran the rule and simply never applied the escalation, so a
// surface an operator is invited to reach for was more permissive than
// the gate it approximates.
func TestCheckAndFastAgreeOnAreaUnknownSeverity(t *testing.T) {
	root := t.TempDir()
	writeAiwfYAML(t, root, "areas:\n  members:\n    - name: platform\n  required: true\n")
	writeEpicFixture(t, root, "kernel")

	var fastCode, fullCode int
	fastOut := testutil.CaptureStdout(t, func() {
		fastCode = Run(root, "json", false, "", false, true, false, nil, "")
	})
	fullOut := testutil.CaptureStdout(t, func() {
		fullCode = Run(root, "json", false, "", false, false, false, nil, "")
	})

	fastSeverity, fastFound := severityOf(t, fastOut, check.CodeAreaUnknown)
	fullSeverity, fullFound := severityOf(t, fullOut, check.CodeAreaUnknown)
	if !fastFound || !fullFound {
		t.Fatalf("area-unknown reported by --fast: %v, by the full check: %v — the fixture no longer triggers the rule on both surfaces",
			fastFound, fullFound)
	}
	if fastSeverity != fullSeverity {
		t.Errorf("area-unknown severity: --fast says %q, the full check says %q — the two surfaces disagree about the same finding on the same tree",
			fastSeverity, fullSeverity)
	}
	if fastSeverity != check.SeverityError {
		t.Errorf("area-unknown severity under `areas.required: true` = %q, want %q", fastSeverity, check.SeverityError)
	}
	if fastCode != fullCode {
		t.Errorf("exit code: --fast returned %d, the full check returned %d — a cheaper approximation of the gate must not be more permissive than it",
			fastCode, fullCode)
	}
	if fastCode != cliutil.ExitFindings {
		t.Errorf("--fast exit code = %d, want %d (errors present)", fastCode, cliutil.ExitFindings)
	}
}

// TestDocsStrictIsARuleOmissionNotASeverityDivergence pins the other
// half of G-0567's analysis, which is *not* a divergence to close:
// `--fast` never runs the doc rules at all, so the two surfaces share
// no doc finding to disagree about. `docs.strict` therefore has nothing
// to escalate on the fast path, and that inertness is what lets one
// uniform seam serve a surface running a subset of the rules.
//
// The fixture has to make the claim falsifiable: a docs corpus that
// really does trip a doc rule, so the full check reports it at error
// while `--fast` reports it not at all. Against a root with no docs at
// all — the shape this test first had — neither surface could produce a
// doc finding, and the assertion would hold against any implementation
// whatsoever.
func TestDocsStrictIsARuleOmissionNotASeverityDivergence(t *testing.T) {
	root := t.TempDir()
	writeAiwfYAML(t, root, "docs:\n  paths:\n    - README.md\n  strict: true\n")
	writeEpicFixture(t, root, "")
	// A narrow-width reference to the epic above: a real id below
	// canonical width is exactly what doc-id-width fires on.
	if err := os.WriteFile(filepath.Join(root, "README.md"),
		[]byte("# Probe\n\nSee E-01 for the plan.\n"), 0o644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}

	var fastCode int
	fastOut := testutil.CaptureStdout(t, func() {
		fastCode = Run(root, "json", false, "", false, true, false, nil, "")
	})
	fullOut := testutil.CaptureStdout(t, func() {
		Run(root, "json", false, "", false, false, false, nil, "")
	})

	fullSeverity, fullFound := severityOf(t, fullOut, check.CodeDocIDWidth)
	if !fullFound {
		t.Fatal("the full check reported no doc-id-width finding — the fixture no longer trips the doc rules, so this test could not tell an inert pass from an omitted one")
	}
	if fullSeverity != check.SeverityError {
		t.Errorf("full check doc-id-width severity = %q, want %q under `docs.strict: true`", fullSeverity, check.SeverityError)
	}
	if _, fastFound := severityOf(t, fastOut, check.CodeDocIDWidth); fastFound {
		t.Error("--fast reported doc-id-width; the fast path is an in-memory pass over the loaded tree and deliberately reads no docs off disk")
	}
	if fastCode != cliutil.ExitOK {
		t.Errorf("--fast exit code = %d, want %d — a rule it does not run must not change its verdict", fastCode, cliutil.ExitOK)
	}
}
