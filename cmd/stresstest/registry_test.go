package main

import (
	"fmt"
	"testing"
)

// registry_test.go — M-0249/AC-1: pins the name -> constructor
// registry's own contract, independent of any real scenario behavior
// (each scenario's own package already covers that): every one of the
// 12 real scenarios built across M-0241 through M-0244 is reachable by
// a stable name, an unknown name is reported as such, and each name
// resolves to the correct Go scenario type — not a copy-paste entry
// pointing at the wrong constructor.

// wantScenarioNames is the exact, ordered set G-0397 and M-0249/AC-1's
// own acceptance text enumerate.
var wantScenarioNames = []string{
	"concurrent-id-allocation",
	"cross-worktree-id-race",
	"reachability-isolation",
	"lock-kill",
	"mid-write-kill",
	"disk-fault",
	"parallel-branch-reallocate",
	"cross-worktree-edit-body-race",
	"archive-during-active-scope",
	"force-override-durability",
	"head-drift",
	"promote-on-wrong-branch-detection",
	"concurrent-writer-at-scale",
	"verb-sequence",
	"concurrent-move",
	"concurrent-milestone-race",
}

func TestScenarioNames_ListsAllTwelveInCatalogOrder(t *testing.T) {
	t.Parallel()
	got := scenarioNames()
	if len(got) != len(wantScenarioNames) {
		t.Fatalf("scenarioNames() = %v (len %d); want len %d", got, len(got), len(wantScenarioNames))
	}
	for i, name := range wantScenarioNames {
		if got[i] != name {
			t.Errorf("scenarioNames()[%d] = %q, want %q", i, got[i], name)
		}
	}
}

// wantScenarioType is the Go type each registered name must resolve
// to, keyed by name — pins the registry against a copy-paste entry
// silently pointing two names at the same constructor.
var wantScenarioType = map[string]string{
	"concurrent-id-allocation":          "*stresstest.ConcurrentIDAllocationScenario",
	"cross-worktree-id-race":            "*stresstest.CrossWorktreeIDRaceScenario",
	"reachability-isolation":            "*stresstest.ReachabilityIsolationScenario",
	"lock-kill":                         "*stresstest.LockKillScenario",
	"mid-write-kill":                    "*stresstest.MidWriteKillScenario",
	"disk-fault":                        "*stresstest.DiskFaultScenario",
	"parallel-branch-reallocate":        "*stresstest.ParallelBranchReallocateScenario",
	"cross-worktree-edit-body-race":     "*stresstest.CrossWorktreeEditBodyRaceScenario",
	"archive-during-active-scope":       "*stresstest.ArchiveDuringActiveScopeScenario",
	"force-override-durability":         "*stresstest.ForceOverrideDurabilityScenario",
	"head-drift":                        "*stresstest.HeadDriftScenario",
	"promote-on-wrong-branch-detection": "*stresstest.PromoteOnWrongBranchDetectionScenario",
	"concurrent-writer-at-scale":        "*stresstest.ConcurrentWriterAtScaleScenario",
	"verb-sequence":                     "*stresstest.VerbSequenceScenario",
	"concurrent-move":                   "*stresstest.ConcurrentMoveScenario",
	"concurrent-milestone-race":         "*stresstest.ConcurrentMilestoneRaceScenario",
}

func TestLookupScenario_KnownNameBuildsTheMatchingScenarioType(t *testing.T) {
	t.Parallel()
	rt := scenarioRuntime{aiwfBin: "unused-aiwf-bin", lockHolderBin: "unused-lockholder-bin"}
	for _, name := range wantScenarioNames {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			entry, ok := lookupScenario(name)
			if !ok {
				t.Fatalf("lookupScenario(%q) ok = false, want true", name)
			}
			if entry.Name != name {
				t.Errorf("entry.Name = %q, want %q", entry.Name, name)
			}
			scenario := entry.Build(rt)(1)
			gotType := fmt.Sprintf("%T", scenario)
			if gotType != wantScenarioType[name] {
				t.Errorf("Build(rt)(1) type = %s, want %s", gotType, wantScenarioType[name])
			}
		})
	}
}

func TestLookupScenario_UnknownNameReturnsFalse(t *testing.T) {
	t.Parallel()
	if _, ok := lookupScenario("does-not-exist"); ok {
		t.Fatal("lookupScenario(\"does-not-exist\") ok = true, want false")
	}
}

func TestNeedsLockHolder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want bool
	}{
		{"lock-kill", true},
		{"all", true},
		{"disk-fault", false},
		{"does-not-exist", false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := needsLockHolder(tt.name); got != tt.want {
				t.Errorf("needsLockHolder(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
