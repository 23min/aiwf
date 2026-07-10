package main

import (
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/stresstest"
)

// registry.go — M-0249/AC-1: the name -> constructor catalog backing
// `--scenario <name>`, `--scenario all`, and `list`. Adapts each of
// the 12 scenarios built across M-0241 through M-0244 into
// RunRepeated's newScenario(seed int64) Scenario shape; every
// scenario's own constructor signature stays exactly as its own
// package defines it (G-0397) — the adaptation (fixed kind/scale
// choices, which of aiwfBin/lockHolderBin a builder actually uses)
// happens here, once per entry, not by forcing a uniform shape onto
// the constructors themselves.

// scenarioRuntime carries the resources a registered builder may
// need: the aiwf binary under test (every scenario but lock-kill), and
// the separately built lockholder binary (lock-kill only).
type scenarioRuntime struct {
	aiwfBin       string
	lockHolderBin string
}

// scenarioBuilder produces one Scenario per --repeat attempt from
// that attempt's seed, closing over rt.
type scenarioBuilder func(rt scenarioRuntime) func(seed int64) stresstest.Scenario

// scenarioEntry is one catalog row: a selectable name plus its builder.
type scenarioEntry struct {
	Name  string
	Build scenarioBuilder
}

// defaultScenarioKind is the entity kind every kind-parameterized
// scenario in the registry drives. Gap is the uniform choice already
// made across every scenario's own tests in internal/stresstest — it
// needs no parent epic/milestone scaffolding, so it exercises each
// scenario's own race/isolation property without entangling it with a
// second entity's lifecycle.
const defaultScenarioKind = entity.KindGap

// defaultScale is the concurrent-actor count for the two
// scale-parameterized scenarios (concurrent-id-allocation,
// concurrent-writer-at-scale) — large enough to make the race window
// each exercises real, small enough that `--scenario all` stays fast
// enough for interactive, on-demand use.
const defaultScale = 8

// defaultVerbSequenceSteps is the promote-attempt count per entity
// kind the verb-sequence walker runs (M-0250/AC-1). Six kinds *
// this many steps must be large enough that every operation in the
// walker's weighted transition table (M-0250/AC-2) fires at least
// once with high probability, small enough that `--scenario all`
// stays fast enough for interactive, on-demand use.
const defaultVerbSequenceSteps = 12

// scenarioAll is the pseudo-name selecting the whole catalog rather
// than one entry — never itself a scenarioCatalog row. Named once so
// resolveScenarios (run.go) and needsLockHolder below can't drift on
// what "all" means.
const scenarioAll = "all"

// lockKillName is lock-kill's own registered name, named once so
// needsLockHolder's string comparison can't silently drift from the
// catalog entry it means to match.
const lockKillName = "lock-kill"

// scenarioCatalog is the ordered registry. Order is display order for
// `list`, run order for `--scenario all`, and the order named in the
// refusal error when --scenario names an unregistered value.
var scenarioCatalog = []scenarioEntry{
	{"concurrent-id-allocation", func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(seed int64) stresstest.Scenario {
			return stresstest.NewConcurrentIDAllocationScenario(rt.aiwfBin, defaultScenarioKind, defaultScale, seed)
		}
	}},
	{"cross-worktree-id-race", func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(seed int64) stresstest.Scenario {
			return stresstest.NewCrossWorktreeIDRaceScenario(rt.aiwfBin, defaultScenarioKind, seed)
		}
	}},
	{"reachability-isolation", func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(seed int64) stresstest.Scenario {
			return stresstest.NewReachabilityIsolationScenario(rt.aiwfBin, defaultScenarioKind, seed)
		}
	}},
	{lockKillName, func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(_ int64) stresstest.Scenario {
			return stresstest.NewLockKillScenario(rt.lockHolderBin)
		}
	}},
	{"mid-write-kill", func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(_ int64) stresstest.Scenario {
			return stresstest.NewMidWriteKillScenario(rt.aiwfBin)
		}
	}},
	{"disk-fault", func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(_ int64) stresstest.Scenario {
			return stresstest.NewDiskFaultScenario(rt.aiwfBin)
		}
	}},
	{"parallel-branch-reallocate", func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(_ int64) stresstest.Scenario {
			return stresstest.NewParallelBranchReallocateScenario(rt.aiwfBin, defaultScenarioKind)
		}
	}},
	{"cross-worktree-edit-body-race", func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(_ int64) stresstest.Scenario {
			return stresstest.NewCrossWorktreeEditBodyRaceScenario(rt.aiwfBin)
		}
	}},
	{"archive-during-active-scope", func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(_ int64) stresstest.Scenario {
			return stresstest.NewArchiveDuringActiveScopeScenario(rt.aiwfBin)
		}
	}},
	{"force-override-durability", func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(_ int64) stresstest.Scenario {
			return stresstest.NewForceOverrideDurabilityScenario(rt.aiwfBin)
		}
	}},
	{"head-drift", func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(_ int64) stresstest.Scenario {
			return stresstest.NewHeadDriftScenario(rt.aiwfBin)
		}
	}},
	{"concurrent-writer-at-scale", func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(seed int64) stresstest.Scenario {
			return stresstest.NewConcurrentWriterAtScaleScenario(rt.aiwfBin, defaultScale, seed)
		}
	}},
	{"verb-sequence", func(rt scenarioRuntime) func(int64) stresstest.Scenario {
		return func(seed int64) stresstest.Scenario {
			return stresstest.NewVerbSequenceScenario(rt.aiwfBin, seed, defaultVerbSequenceSteps)
		}
	}},
}

// expectedRedScenario names the one catalog entry that is deliberately
// expected to report a violation until G-0269's own guard ships
// (head_drift.go's doc comment). `--scenario all`'s combined summary
// reports it distinctly rather than folding it into the same pass/fail
// signal as every other scenario.
const expectedRedScenario = "head-drift"

// scenarioNames returns every registered name in catalog order.
func scenarioNames() []string {
	names := make([]string, len(scenarioCatalog))
	for i, e := range scenarioCatalog {
		names[i] = e.Name
	}
	return names
}

// lookupScenario returns the catalog entry named name, or ok=false if
// no such entry is registered.
func lookupScenario(name string) (scenarioEntry, bool) {
	for _, e := range scenarioCatalog {
		if e.Name == name {
			return e, true
		}
	}
	return scenarioEntry{}, false
}

// needsLockHolder reports whether name requires the separately built
// lockholder binary — true for lock-kill itself, or for scenarioAll
// (which runs lock-kill as part of the catalog).
func needsLockHolder(name string) bool {
	return name == scenarioAll || name == lockKillName
}
