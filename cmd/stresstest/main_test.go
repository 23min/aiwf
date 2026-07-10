package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun_HelpReturnsZero(t *testing.T) {
	t.Parallel()
	if code := run([]string{"--help"}); code != 0 {
		t.Fatalf("run([--help]) = %d, want 0", code)
	}
}

func TestRun_UnknownCommandReturnsOne(t *testing.T) {
	t.Parallel()
	if code := run([]string{"bogus-verb"}); code != 1 {
		t.Fatalf("run([bogus-verb]) = %d, want 1", code)
	}
}

// TestRun_RunCommand_Succeeds drives the "run" subcommand's own
// RunE closure (newRunCmd's flag-to-runRun wiring) through the same
// entry point a real invocation uses — the seam the run_test.go
// unit tests around runRun itself never exercise, since they call
// runRun directly.
func TestRun_RunCommand_Succeeds(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()

	code := run([]string{
		"run",
		"--module-root", repoRootRelative,
		"--out", outDir,
		"--scenario", "disk-fault",
		"--repeat", "1",
	})
	if code != 0 {
		t.Fatalf("run([run ...]) = %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(outDir, "report.jsonl")); err != nil {
		t.Fatalf("expected report.jsonl to exist after a successful run: %v", err)
	}
}

// TestRun_RunCommand_ErrorsWhenScenarioMissing pins that --scenario is
// a required flag at the Cobra layer (MarkFlagRequired), not just
// something runRun happens to reject.
func TestRun_RunCommand_ErrorsWhenScenarioMissing(t *testing.T) {
	t.Parallel()
	if code := run([]string{"run", "--out", t.TempDir()}); code != 1 {
		t.Fatalf("run([run --out ...]) with no --scenario = %d, want 1", code)
	}
}
