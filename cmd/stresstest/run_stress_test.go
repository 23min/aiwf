//go:build stress

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/stresstest"
)

// run_stress_test.go — run.go's `stress`-lane tests. The
// whole-catalog tests execute every registered scenario, including
// the ones that drive real concurrent subprocesses, and the lock-kill
// test drives a real lock-holding subprocess of its own. Running that
// many processes alongside `go test ./...` roughly doubles their wall
// time, and `concurrent-milestone-race` still judges the machine, so
// they own the runner here rather than sharing it.
// `resolveScenarios`'s own selection claim — that "all" names every
// registered entry, in catalog order — is pinned untagged in
// run_test.go and runs on every push.

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
// diagnostic-log file across every registered scenario, so a bug that reset
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
