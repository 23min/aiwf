package stresstest

import (
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// TestScenarioSetup_SeedsItsOwnFixture pins that every scenario whose
// driver runs only in the `stress`-tagged lane can still build the
// fixture it races against.
//
// Setup calls real `aiwf` verbs, so a kernel rule that refuses one of
// them makes the scenario fail before its oracle ever runs — and
// because the drivers are tagged, nothing on the every-push lane
// notices. The seeding is cheap where the racing is not: each Setup
// here is a handful of subprocesses, and the scale parameters are
// passed at their smallest.
//
// The rows are the tagged scenarios whose Setup calls an aiwf verb.
// lock-kill's only git-inits a directory, and the untagged scenarios'
// own drivers run on every push and call Setup first.
func TestScenarioSetup_SeedsItsOwnFixture(t *testing.T) {
	t.Parallel()
	bin := sharedTestBinary(t)
	cases := []struct {
		name     string
		scenario Scenario
	}{
		{"concurrent-milestone-race", NewConcurrentMilestoneRaceScenario(bin, 1, 0)},
		{"concurrent-writer-at-scale", NewConcurrentWriterAtScaleScenario(bin, 1, 0)},
		{"cross-worktree-id-race", NewCrossWorktreeIDRaceScenario(bin, entity.KindGap, 0)},
		{"mid-write-kill", NewMidWriteKillScenario(bin)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.scenario.Setup(t.TempDir()); err != nil {
				t.Fatalf("Setup: %v", err)
			}
		})
	}
}

// TestLaunchAddIn_LandsAnEntity pins the same property one seam later,
// for the scenario whose actors build their `aiwf add` outside Setup:
// cross-worktree-id-race constructs its own command line, so a body or
// flag the kernel refuses fails every actor rather than the seeding,
// and the race reports two refusals instead of two ids.
//
// One actor, run alone: the id it allocates is beside the point here —
// what is under test is that the command a racing actor issues is one
// the kernel accepts.
func TestLaunchAddIn_LandsAnEntity(t *testing.T) {
	t.Parallel()
	bin := sharedTestBinary(t)
	dir := t.TempDir()
	if err := newSiblingWorktreesFixture(dir); err != nil {
		t.Fatalf("seeding sibling worktrees: %v", err)
	}
	got := launchAddIn(bin, filepath.Join(dir, "wt-a"), entity.KindGap, "solo add from one race actor")
	if got.execErr != nil {
		t.Fatalf("launchAddIn: %v\n%s", got.execErr, got.out)
	}
	env, err := parseVerbEnvelope([]string{"add", string(entity.KindGap)}, got.out)
	if err != nil {
		t.Fatalf("parsing the add envelope: %v\n%s", err, got.out)
	}
	if env.Status != "ok" {
		t.Errorf("aiwf add reported status=%s, error=%+v — a racing actor's own command is refused by the kernel", env.Status, env.Error)
	}
}
