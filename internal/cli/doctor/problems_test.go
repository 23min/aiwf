package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// errorCount returns the number of SeverityError problems — the
// exit-relevant subset doctor's exit code weighs (warnings never gate
// the exit).
func errorCount(ps []Problem) int {
	n := 0
	for i := range ps {
		if ps[i].Severity == SeverityError {
			n++
		}
	}
	return n
}

// TestProblems_MissingConfig_Error pins that a missing aiwf.yaml surfaces
// as an error-severity problem whose message names it — the concrete
// evidence for AC-1.
func TestProblems_MissingConfig_Error(t *testing.T) {
	t.Parallel()
	dir := t.TempDir() // bare dir: no aiwf.yaml
	ps := Problems(dir, DoctorOptions{})
	var cfg *Problem
	for i := range ps {
		if strings.Contains(ps[i].Message, "aiwf.yaml") {
			cfg = &ps[i]
		}
	}
	if cfg == nil {
		t.Fatalf("no problem mentioning aiwf.yaml among %d problems", len(ps))
	}
	if cfg.Severity != SeverityError {
		t.Errorf("aiwf.yaml problem severity = %q, want %q", cfg.Severity, SeverityError)
	}
}

// TestProblems_PrePushHookNoMarker_Warn pins the WARN branch: a
// `.git/hooks/pre-push` file present but WITHOUT the `# aiwf:pre-push`
// marker is an actionable advisory, so it surfaces as a SeverityWarn —
// never an error (an unmarked hook is the consumer's own; doctor reports
// it but leaves it alone, so it must not gate the exit).
func TestProblems_PrePushHookNoMarker_Warn(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	hooks := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "pre-push"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, problems := appendHookReport(nil, nil, root)
	var warn *Problem
	for i := range problems {
		if problems[i].Severity == SeverityWarn && strings.Contains(problems[i].Message, "# aiwf:pre-push") {
			warn = &problems[i]
		}
	}
	if warn == nil {
		t.Fatalf("no SeverityWarn problem naming the missing `# aiwf:pre-push` marker among %d problems", len(problems))
	}
	if got := errorCount(problems); got != 0 {
		t.Errorf("an unmarked (alien) hook must not raise an error; got %d errors", got)
	}
}

// TestProblems_HealthySection_NoProblem confirms a healthy section
// yields no problem at all: a guidance fragment present and imported by
// CLAUDE.md reports the ok line and contributes nothing to the problem
// list.
func TestProblems_HealthySection_NoProblem(t *testing.T) {
	t.Parallel()
	root := guidanceFixture(t, true, true) // fragment present + imported
	lines, problems := appendGuidanceImportReport(nil, nil, root)
	if len(problems) != 0 {
		t.Errorf("wired guidance is healthy; want 0 problems, got %d", len(problems))
	}
	if !strings.Contains(strings.Join(lines, "\n"), "ok (CLAUDE.md imports") {
		t.Errorf("expected the guidance ok line; got:\n%s", strings.Join(lines, "\n"))
	}
}
