package policies_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// growth_report_test.go — wiring pins for `make growth-report`.
//
// scripts/growth-report.py is the re-runnable measure behind
// docs/design/growth.md, whose whole method is measure -> ship a lever ->
// measure again. Drop the script, its exec bit, or the Makefile target and
// the doc's re-measure instructions become false while every gate stays
// green: the tool is advisory, so nothing else fails when it stops being
// invokable.
//
// Deliberately wiring-and-parse only. The report's *numbers* are not pinned
// here — a metric is a judgment about what to watch, and a snapshot asserted
// against a fixture would need editing on every commit that moves the tree.

func growthReportScriptPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRootForHook(t), "scripts", "growth-report.py")
}

// TestGrowthReport_Wiring pins the install surface: the script is tracked and
// executable, and the Makefile target invokes it.
func TestGrowthReport_Wiring(t *testing.T) {
	t.Parallel()
	root := repoRootForHook(t)

	info, err := os.Stat(growthReportScriptPath(t))
	if err != nil {
		t.Fatalf("tracked growth-report script missing: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("scripts/growth-report.py must be executable; mode = %v", info.Mode())
	}

	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("reading Makefile: %v", err)
	}
	if !strings.Contains(string(makefile), "scripts/growth-report.py") {
		t.Error("Makefile has no target invoking scripts/growth-report.py; `make growth-report` is the documented entry point in docs/design/growth.md")
	}
}

// TestGrowthReport_Parses runs the script's --help path, which exercises
// import and argument wiring without touching git. A syntax error or a broken
// import surfaces here rather than the next time someone re-measures.
func TestGrowthReport_Parses(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("python3", growthReportScriptPath(t), "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("growth-report.py --help failed: %v\n%s", err, out)
	}
	for _, flag := range []string{"--at", "--baseline", "--tsv"} {
		if !strings.Contains(string(out), flag) {
			t.Errorf("growth-report.py --help does not document %s; docs/design/growth.md tells readers to use it", flag)
		}
	}
}
