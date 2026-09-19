package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

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

// TestRunCommand_DefaultSeeds cannot use t.Parallel(): runRun unconditionally
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
func TestRunCommand_DefaultSeeds(t *testing.T) {
	outDir := t.TempDir()
	var out bytes.Buffer

	seed := int64(40)
	seedFn := func() int64 { seed++; return seed }
	cmd := newRunCmd(seedFn)
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--module-root", repoRootRelative, "--out", outDir, "--scenario", "disk-fault", "--repeat", "2"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	reportPath := filepath.Join(outDir, "report.jsonl")
	composed, err := stresstest.Compose(reportPath)
	if err != nil {
		t.Fatalf("Compose(%q): %v", reportPath, err)
	}
	if len(composed.Events) != 2 {
		t.Fatalf("expected 2 logged events (one per repeat attempt), got %d", len(composed.Events))
	}
	for i, raw := range composed.Events {
		var event stresstest.RepeatEvent
		if decodeErr := json.Unmarshal(raw, &event); decodeErr != nil {
			t.Fatal(decodeErr)
		}
		if want := int64(41 + i); event.Seed != want {
			t.Errorf("event %d seed = %d; want %d", i, event.Seed, want)
		}
	}
	if !strings.Contains(out.String(), "disk-fault: 2/2 attempts passed") {
		t.Fatalf("unexpected summary output: %q", out.String())
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
	if err := runRun(context.Background(), repoRootRelative, outDir, 0, "disk-fault", io.Discard, nextSeed); err == nil {
		t.Fatal("expected runRun to reject a non-positive repeat count before doing any work")
	}
}

// TestRunRun_ErrorsWhenScenarioIsUnknown pins that an unregistered
// --scenario name refuses before any I/O (repeat<=0's sibling
// fail-fast check) — no build, no report file, just the refusal.
func TestRunRun_ErrorsWhenScenarioIsUnknown(t *testing.T) {
	t.Parallel()
	outDir := t.TempDir()
	err := runRun(context.Background(), repoRootRelative, outDir, 1, "does-not-exist", io.Discard, nextSeed)
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

	if err := runRun(context.Background(), repoRootRelative, bad, 1, "disk-fault", io.Discard, nextSeed); err == nil {
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
	err := runRun(context.Background(), repoRootRelative, outDir, 1, "disk-fault", io.Discard, nextSeed)
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

	if err := runRun(context.Background(), bogusRoot, outDir, 1, "disk-fault", io.Discard, nextSeed); err == nil {
		t.Fatal("expected runRun to propagate a BuildBinary failure")
	}
}

// TestResolveScenarios_All_NamesEveryCatalogEntryInOrder pins the
// selection half of `--scenario all`: it resolves to every registered
// entry, in catalog order. The execution half — actually running the
// whole catalog — asserts timing properties of the machine it runs on
// (the catalog carries the concurrency and fault-injection scenarios),
// so it lives behind the `stress` build tag in
// run_scenario_all_test.go. This assertion is what keeps the selection
// claim mechanically checked on every push.
func TestResolveScenarios_All_NamesEveryCatalogEntryInOrder(t *testing.T) {
	t.Parallel()

	got, err := resolveScenarios(scenarioAll)
	if err != nil {
		t.Fatalf("resolveScenarios(%q): %v", scenarioAll, err)
	}
	gotNames := make([]string, len(got))
	for i, e := range got {
		gotNames[i] = e.Name
	}
	if diff := cmp.Diff(scenarioNames(), gotNames); diff != "" {
		t.Errorf("resolveScenarios(%q) (-want +got):\n%s", scenarioAll, diff)
	}
}

// TestResolveScenarios_NamedEntry_ResolvesToThatEntryAlone pins the
// other selection arm: a registered name resolves to exactly one entry.
func TestResolveScenarios_NamedEntry_ResolvesToThatEntryAlone(t *testing.T) {
	t.Parallel()

	got, err := resolveScenarios(lockKillName)
	if err != nil {
		t.Fatalf("resolveScenarios(%q): %v", lockKillName, err)
	}
	if len(got) != 1 {
		t.Fatalf("resolveScenarios(%q) resolved to %d entries; want exactly one", lockKillName, len(got))
	}
	if got[0].Name != lockKillName {
		t.Fatalf("resolveScenarios(%q) resolved to %q; want %q", lockKillName, got[0].Name, lockKillName)
	}
}

func TestPrintScenarioSummary_Violations(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		results []stresstest.RunResult
		want    string
	}{
		{"multiple failures", []stresstest.RunResult{
			{Passed: true},
			{Dir: "/preserved", Violations: []stresstest.Violation{{Message: "first breach"}, {Message: "second breach"}}},
			{Violations: []stresstest.Violation{{Message: "third breach"}}},
		}, "stresstest run: sample: attempt failed, repo preserved at /preserved\n  violation: first breach\n  violation: second breach\n  violation: third breach\nstresstest run: sample: 1/3 attempts passed\n"},
		{"pass", []stresstest.RunResult{{Passed: true}}, "stresstest run: sample: 1/1 attempts passed\n"},
		{"failure without violations", []stresstest.RunResult{{Dir: "/preserved"}}, "stresstest run: sample: attempt failed, repo preserved at /preserved\nstresstest run: sample: 0/1 attempts passed\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			printScenarioSummary(&out, "sample", tc.results)
			if got := out.String(); got != tc.want {
				t.Errorf("summary = %q; want %q", got, tc.want)
			}
		})
	}
}

// Serial: successful commands set the process-wide diagnostic environment.
func TestRunCommand_ExplicitSeed(t *testing.T) {
	for _, seed := range []int64{0, -42, 12345} {
		t.Run(strconv.FormatInt(seed, 10), func(t *testing.T) {
			outDir := t.TempDir()
			cmd := newRunCmd(nextSeed)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"--module-root", repoRootRelative, "--out", outDir, "--scenario", "disk-fault", "--repeat", "2", "--seed", strconv.FormatInt(seed, 10)})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			report, err := stresstest.Compose(filepath.Join(outDir, "report.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Events) != 2 {
				t.Fatalf("events = %d; want 2", len(report.Events))
			}
			for i, raw := range report.Events {
				var event stresstest.RepeatEvent
				if decodeErr := json.Unmarshal(raw, &event); decodeErr != nil {
					t.Fatal(decodeErr)
				}
				if event.Seed != seed || event.Attempt != i || !event.Passed {
					t.Errorf("event %d = %+v; want seed %d, matching attempt, and pass", i, event, seed)
				}
			}
		})
	}
}
