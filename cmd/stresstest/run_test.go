package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/23min/aiwf/internal/stresstest"
)

// repoRootRelative is the module root relative to this test binary's
// working directory. This file always lives at cmd/stresstest/, a
// fixed two levels below the repo root — mirrors
// internal/stresstest/binary_test.go's own repoRootRelative constant.
const repoRootRelative = "../.."

func TestResolveOutDir_EmptyCreatesFreshTempDir(t *testing.T) {
	t.Parallel()
	dir, err := resolveOutDir("")
	if err != nil {
		t.Fatalf("resolveOutDir(\"\"): %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	if !filepath.IsAbs(dir) {
		t.Fatalf("resolveOutDir(\"\") = %q, want an absolute path", dir)
	}
	if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
		t.Fatalf("resolveOutDir did not create a directory: stat err=%v", statErr)
	}
}

func TestResolveOutDir_NonEmptyCreatesGivenDir(t *testing.T) {
	t.Parallel()
	want := filepath.Join(t.TempDir(), "run-out")
	dir, err := resolveOutDir(want)
	if err != nil {
		t.Fatalf("resolveOutDir(%q): %v", want, err)
	}
	if dir != want {
		t.Fatalf("resolveOutDir(%q) = %q, want %q", want, dir, want)
	}
	if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
		t.Fatalf("resolveOutDir did not create the directory: %v", statErr)
	}
}

func TestResolveOutDir_ErrorsWhenMkdirAllFails(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed blocker file: %v", err)
	}
	// blocker exists as a regular file; asking to create a directory
	// under it must fail — a path component can't be both a file and
	// a directory.
	bad := filepath.Join(blocker, "child")

	if _, err := resolveOutDir(bad); err == nil {
		t.Fatal("expected resolveOutDir to fail when a path component is a plain file")
	}
}

// TestResolveOutDir_ErrorsWhenMkdirTempFails cannot use t.Parallel():
// t.Setenv panics if the test (or an ancestor) is parallel, and
// os.MkdirTemp("", ...) resolves its base directory from $TMPDIR.
// Pointing TMPDIR at a path with no such directory forces a
// deterministic, portable MkdirTemp failure without touching the
// process's working directory (which would be unsafe to mutate under
// parallel tests).
func TestResolveOutDir_ErrorsWhenMkdirTempFails(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "does-not-exist"))
	if _, err := resolveOutDir(""); err == nil {
		t.Fatal("expected resolveOutDir(\"\") to fail when the OS temp dir doesn't exist")
	}
}

// TestRunRun_Succeeds cannot use t.Parallel(): runRun unconditionally
// enables diagnostic logging via os.Setenv(AIWF_LOG*) before running
// any scenario (M-0249/AC-2) — a process-wide mutation. Go's env
// functions are memory-safe to call concurrently (internally
// mutex-guarded since Go 1.9), but two overlapping runRun calls could
// still logically race: a later AIWF_LOG_FILE Setenv from a different
// test could land while this test's own RunRepeated loop is still
// mid-flight, misdirecting a later attempt's subprocess output into
// the wrong test's diagnostic log. Every runRun-driving test in this
// file that reaches the env-setting code (past scenario/out-dir
// resolution and the binary build) stays serial for the same reason.
func TestRunRun_Succeeds(t *testing.T) {
	outDir := t.TempDir()
	var out bytes.Buffer

	if err := runRun(context.Background(), repoRootRelative, outDir, 2, "disk-fault", &out); err != nil {
		t.Fatalf("runRun: %v", err)
	}

	reportPath := filepath.Join(outDir, "report.jsonl")
	composed, err := stresstest.Compose(reportPath)
	if err != nil {
		t.Fatalf("Compose(%q): %v", reportPath, err)
	}
	if len(composed.Events) != 2 {
		t.Fatalf("expected 2 logged events (one per repeat attempt), got %d", len(composed.Events))
	}
	if !strings.Contains(out.String(), "disk-fault: 2/2 attempts passed") {
		t.Fatalf("unexpected summary output: %q", out.String())
	}
}

// TestRunRun_LockKillScenario_BuildsLockHolderAndRuns pins runRun's
// needsLockHolder branch: selecting "lock-kill" builds the separate
// lockholder binary (BuildLockHolder) alongside the aiwf binary under
// test, and the scenario runs to a real pass. Serial — see
// TestRunRun_Succeeds's doc comment.
func TestRunRun_LockKillScenario_BuildsLockHolderAndRuns(t *testing.T) {
	outDir := t.TempDir()
	var out bytes.Buffer

	if err := runRun(context.Background(), repoRootRelative, outDir, 1, "lock-kill", &out); err != nil {
		t.Fatalf("runRun: %v", err)
	}
	if !strings.Contains(out.String(), "lock-kill: 1/1 attempts passed") {
		t.Fatalf("unexpected summary output: %q", out.String())
	}
}

// TestRunRun_ScenarioAll_RunsWholeCatalogIntoOneReport pins AC-2's own
// acceptance text: --scenario all runs every registered scenario, all
// logged into the same raw-report file. Serial — see
// TestRunRun_Succeeds's doc comment.
func TestRunRun_ScenarioAll_RunsWholeCatalogIntoOneReport(t *testing.T) {
	outDir := t.TempDir()
	var out bytes.Buffer

	if err := runRun(context.Background(), repoRootRelative, outDir, 1, "all", &out); err != nil {
		t.Fatalf("runRun: %v", err)
	}

	reportPath := filepath.Join(outDir, "report.jsonl")
	composed, err := stresstest.Compose(reportPath)
	if err != nil {
		t.Fatalf("Compose(%q): %v", reportPath, err)
	}
	if len(composed.Events) != len(scenarioNames()) {
		t.Fatalf("expected 1 logged event per catalog scenario (%d), got %d", len(scenarioNames()), len(composed.Events))
	}

	for _, name := range scenarioNames() {
		if !strings.Contains(out.String(), name) {
			t.Errorf("summary output does not mention scenario %q:\n%s", name, out.String())
		}
		if !strings.Contains(out.String(), name+": 1/1 attempts passed") {
			t.Errorf("expected scenario %q to report a clean pass, got:\n%s", name, out.String())
		}
	}
}

// TestRunRun_ScenarioAll_CorrelationIDsDoNotBleedAcrossScenarios pins
// the cross-scenario diagnostic-log cursor: --scenario all shares one
// diagnostic-log file across all 12 scenarios, so a bug that reset
// the read cursor per scenario (rather than carrying it forward)
// would re-scan from byte 0 each time and re-attribute every earlier
// scenario's own correlation ids to each later scenario's first
// event. With --repeat 1, report.jsonl has exactly one event per
// scenario in registry order, so no id may appear in more than one
// event. Serial — see TestRunRun_Succeeds's doc comment.
func TestRunRun_ScenarioAll_CorrelationIDsDoNotBleedAcrossScenarios(t *testing.T) {
	outDir := t.TempDir()
	var out bytes.Buffer

	if err := runRun(context.Background(), repoRootRelative, outDir, 1, "all", &out); err != nil {
		t.Fatalf("runRun: %v", err)
	}

	reportPath := filepath.Join(outDir, "report.jsonl")
	composed, err := stresstest.Compose(reportPath)
	if err != nil {
		t.Fatalf("Compose(%q): %v", reportPath, err)
	}

	type event struct {
		CorrelationIDs []string `json:"correlation_ids"`
	}
	seen := make(map[string]int) // id -> index of the event that first carried it
	totalIDs := 0
	for i, raw := range composed.Events {
		var ev event
		if err := json.Unmarshal(raw, &ev); err != nil {
			t.Fatalf("event %d not valid JSON: %v", i, err)
		}
		totalIDs += len(ev.CorrelationIDs)
		for _, id := range ev.CorrelationIDs {
			if firstIdx, ok := seen[id]; ok {
				t.Fatalf("correlation id %q appears in both event %d and event %d — the diagnostic-log cursor re-attributed an earlier scenario's id to a later one", id, firstIdx, i)
			}
			seen[id] = i
		}
	}
	if totalIDs == 0 {
		t.Fatal("no correlation ids observed across the whole run; diagnostic logging did not attach to any scenario")
	}
}

// TestRunRun_PrintsPreservedDirOnAFailingAttempt pins that
// printScenarioSummary surfaces a failing attempt's preserved repo
// dir to the operator — previously RunResult.Dir was populated in
// memory but never printed. Exercised directly against a fabricated
// failing RunResult rather than a real catalog scenario: every
// registered scenario is expected to pass cleanly (there is no longer
// a deterministically-failing one in the catalog to piggyback on
// since G-0269's guard shipped).
func TestRunRun_PrintsPreservedDirOnAFailingAttempt(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer

	printScenarioSummary(&out, "fabricated-scenario", []stresstest.RunResult{
		{Passed: false, Dir: "/tmp/fabricated-preserved-dir"},
	})
	if !strings.Contains(out.String(), "attempt failed, repo preserved at /tmp/fabricated-preserved-dir") {
		t.Fatalf("expected the failing attempt's preserved dir to be printed, got:\n%s", out.String())
	}
}

func TestRunRun_ErrorsWhenRepeatIsNonPositive(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	if err := runRun(context.Background(), repoRootRelative, outDir, 0, "disk-fault", io.Discard); err == nil {
		t.Fatal("expected runRun to reject a non-positive repeat count before doing any work")
	}
}

// TestRunRun_ErrorsWhenScenarioIsUnknown pins that an unregistered
// --scenario name refuses before any I/O (repeat<=0's sibling
// fail-fast check) — no build, no report file, just the refusal.
func TestRunRun_ErrorsWhenScenarioIsUnknown(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	err := runRun(context.Background(), repoRootRelative, outDir, 1, "does-not-exist", io.Discard)
	if err == nil {
		t.Fatal("expected runRun to reject an unregistered --scenario name")
	}
	if !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("expected the error to name the bad value, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "report.jsonl")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no report.jsonl to be created for a rejected scenario name, stat err: %v", statErr)
	}
}

func TestRunRun_ErrorsWhenOutDirResolutionFails(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed blocker file: %v", err)
	}
	bad := filepath.Join(blocker, "child")

	if err := runRun(context.Background(), repoRootRelative, bad, 1, "disk-fault", io.Discard); err == nil {
		t.Fatal("expected runRun to propagate a resolveOutDir failure")
	}
}

// TestRunRun_ErrorsWhenReportPathIsADirectory pins that report-opening
// happens BEFORE the (expensive) binary build, not just that runRun
// eventually fails somehow. Using a real, buildable moduleRoot is
// deliberate: an invalid moduleRoot would make BuildBinary itself the
// one that fails, which would let this test pass even if the report
// were opened last — the failure would just come from a different
// step. A real moduleRoot plus a timing bound closes that gap: this
// call must fail fast, well under the real build's ~1.4s, or the
// implementation regressed to building first.
func TestRunRun_ErrorsWhenReportPathIsADirectory(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	// Pre-create the report path as a directory so OpenReportWriter's
	// os.OpenFile fails with EISDIR — a fast, deterministic way to
	// exercise runRun's report-open error branch without needing
	// BuildBinary to run at all (report opening happens first).
	if err := os.Mkdir(filepath.Join(outDir, "report.jsonl"), 0o755); err != nil {
		t.Fatalf("seed report.jsonl as a directory: %v", err)
	}

	start := time.Now()
	err := runRun(context.Background(), repoRootRelative, outDir, 1, "disk-fault", io.Discard)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected runRun to fail opening the raw-report file")
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("runRun took %s to fail; expected a fast failure before any build was attempted (report must open before build)", elapsed)
	}
}

func TestRunRun_ErrorsWhenBuildFails(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	bogusRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(bogusRoot, "go.mod"), []byte("module bogus\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	if err := runRun(context.Background(), bogusRoot, outDir, 1, "disk-fault", io.Discard); err == nil {
		t.Fatal("expected runRun to propagate a BuildBinary failure")
	}
}
