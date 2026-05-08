package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-076/AC-2 + AC-3: `aiwf milestone depends-on M-NNN --on M-PPP[,M-QQQ]`
// sets edges on an already-allocated milestone in a single commit.
// `--clear` empties the list. `--clear` and `--on` are mutually
// exclusive. Closes the verb half of G-072.

// milestoneDependsOnSetup gives every test in this file a freshly-init'd
// repo with one epic and three milestones (M-001, M-002, M-003) so the
// post-allocation depends-on verb has referents to point at.
func milestoneDependsOnSetup(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	for i, slug := range []string{"First", "Second", "Third"} {
		_ = i
		if rc := run([]string{"add", "milestone", "--epic", "E-01", "--tdd", "none", "--title", slug, "--actor", "human/test", "--root", root}); rc != exitOK {
			t.Fatalf("add milestone %s: %d", slug, rc)
		}
	}
	return root
}

// TestMilestoneDependsOn_SetSingle pins AC-2's basic contract: setting
// a single dependency via the dedicated verb writes the depends_on
// frontmatter array on the target milestone.
func TestMilestoneDependsOn_SetSingle(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	rc := run([]string{
		"milestone", "depends-on", "M-003",
		"--on", "M-001",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("milestone depends-on M-003 --on M-001 = %d, want %d", rc, exitOK)
	}

	mPath := filepath.Join(root, "work", "epics", "E-01-foundations", "M-003-third.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	if !strings.Contains(string(body), "depends_on:") {
		t.Errorf("frontmatter missing depends_on block:\n%s", body)
	}
	if !strings.Contains(string(body), "- M-001") {
		t.Errorf("frontmatter missing - M-001:\n%s", body)
	}
}

// TestMilestoneDependsOn_SetMultiple pins AC-2's comma-list contract.
func TestMilestoneDependsOn_SetMultiple(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	rc := run([]string{
		"milestone", "depends-on", "M-003",
		"--on", "M-001,M-002",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("milestone depends-on M-003 --on M-001,M-002 = %d, want %d", rc, exitOK)
	}

	mPath := filepath.Join(root, "work", "epics", "E-01-foundations", "M-003-third.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	content := string(body)
	if !strings.Contains(content, "- M-001") {
		t.Errorf("frontmatter missing - M-001:\n%s", content)
	}
	if !strings.Contains(content, "- M-002") {
		t.Errorf("frontmatter missing - M-002:\n%s", content)
	}
}

// TestMilestoneDependsOn_Replace pins AC-2's replace-not-append
// semantics (per the spec's locked design): a second invocation
// replaces the list rather than appending.
func TestMilestoneDependsOn_Replace(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	if rc := run([]string{"milestone", "depends-on", "M-003", "--on", "M-001", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("first set: %d", rc)
	}
	if rc := run([]string{"milestone", "depends-on", "M-003", "--on", "M-002", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("second set (replace): %d", rc)
	}

	mPath := filepath.Join(root, "work", "epics", "E-01-foundations", "M-003-third.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	content := string(body)
	if strings.Contains(content, "- M-001") {
		t.Errorf("replace semantics broken: M-001 still present after second set:\n%s", content)
	}
	if !strings.Contains(content, "- M-002") {
		t.Errorf("frontmatter missing - M-002 after replace:\n%s", content)
	}
}

// TestMilestoneDependsOn_Clear pins AC-3: --clear empties the
// depends_on list.
func TestMilestoneDependsOn_Clear(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	if rc := run([]string{"milestone", "depends-on", "M-003", "--on", "M-001,M-002", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("set initial: %d", rc)
	}
	if rc := run([]string{"milestone", "depends-on", "M-003", "--clear", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("clear: %d", rc)
	}

	mPath := filepath.Join(root, "work", "epics", "E-01-foundations", "M-003-third.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	if strings.Contains(string(body), "depends_on:") {
		t.Errorf("--clear should remove the depends_on block (omitempty); got:\n%s", body)
	}
}

// TestMilestoneDependsOn_ClearAndOnMutex pins AC-3's mutex: --clear
// and --on cannot be combined; the verb refuses with a usage error.
func TestMilestoneDependsOn_ClearAndOnMutex(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	rc := run([]string{
		"milestone", "depends-on", "M-003",
		"--on", "M-001",
		"--clear",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("milestone depends-on --on --clear = %d, want %d (mutex)", rc, exitUsage)
	}
}

// TestMilestoneDependsOn_NoFlagIsUsage pins the verb's contract: at
// least one of --on or --clear must be passed; bare invocation is a
// usage error.
func TestMilestoneDependsOn_NoFlagIsUsage(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	rc := run([]string{
		"milestone", "depends-on", "M-003",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("milestone depends-on with no --on/--clear = %d, want %d", rc, exitUsage)
	}
}

// TestMilestoneDependsOn_TargetNotMilestone pins the verb-side guard:
// the positional id must resolve to a milestone, not any other kind.
func TestMilestoneDependsOn_TargetNotMilestone(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	rc := run([]string{
		"milestone", "depends-on", "E-01",
		"--on", "M-001",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("milestone depends-on E-01 = %d, want %d (E-01 is not a milestone)", rc, exitUsage)
	}
}

// TestMilestoneDependsOn_TargetUnknown pins the verb-side guard for
// missing target milestone.
func TestMilestoneDependsOn_TargetUnknown(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	rc := run([]string{
		"milestone", "depends-on", "M-999",
		"--on", "M-001",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("milestone depends-on M-999 = %d, want %d (M-999 doesn't exist)", rc, exitUsage)
	}
}

// TestMilestoneDependsOn_OnRefUnknown pins AC-4 on the verb side: the
// --on referent must resolve to an existing milestone.
func TestMilestoneDependsOn_OnRefUnknown(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	rc := run([]string{
		"milestone", "depends-on", "M-003",
		"--on", "M-999",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("milestone depends-on M-003 --on M-999 = %d, want %d", rc, exitUsage)
	}
}

// TestMilestoneDependsOn_OnRefNonMilestone pins AC-4's kind-restriction
// on the verb side.
func TestMilestoneDependsOn_OnRefNonMilestone(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	rc := run([]string{
		"milestone", "depends-on", "M-003",
		"--on", "E-01",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("milestone depends-on M-003 --on E-01 = %d, want %d (E-01 is not a milestone)", rc, exitUsage)
	}
}

// TestMilestoneDependsOn_CompositeIDRejected pins the verb-level guard
// that rejects composite ids (M-NNN/AC-N) — depends_on is a milestone-
// level field, not an AC-level one.
func TestMilestoneDependsOn_CompositeIDRejected(t *testing.T) {
	root := milestoneDependsOnSetup(t)
	// Allocate an AC under M-001 so the composite id resolves.
	if rc := run([]string{"add", "ac", "M-001", "--title", "first ac", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add ac: %d", rc)
	}

	rc := run([]string{
		"milestone", "depends-on", "M-001/AC-1",
		"--on", "M-002",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("milestone depends-on M-001/AC-1 = %d, want %d (composite ids rejected)", rc, exitUsage)
	}
}

// TestMilestoneDependsOn_SelfDependencyRejected pins the self-loop
// guard: a milestone cannot depend on itself.
func TestMilestoneDependsOn_SelfDependencyRejected(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	rc := run([]string{
		"milestone", "depends-on", "M-003",
		"--on", "M-003",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitUsage {
		t.Errorf("milestone depends-on M-003 --on M-003 = %d, want %d (self-loop)", rc, exitUsage)
	}
}

// TestMilestoneDependsOn_DispatcherSeam_AddFlag is the seam test for
// M-076/AC-7 per CLAUDE.md "Test the seam, not just the layer". It
// drives `aiwf add milestone --depends-on …` end-to-end through the
// dispatcher and asserts the on-disk milestone reflects the writer's
// contract: the depends_on array landed atomically with the create
// commit, and `aiwf history M-NNN` finds the trailered create commit.
//
// Why this exists alongside the focused AC-1 tests: AC-1's tests assert
// the file content; this one additionally asserts that the dispatcher
// path (cmd → verb.Add → projectAdd → projection-check → Apply) wires
// the new field through every layer. A regression where, say, the cmd
// flag is read but never copied into AddOptions would slip past
// individual unit tests but trip here.
func TestMilestoneDependsOn_DispatcherSeam_AddFlag(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	rc := run([]string{
		"add", "milestone",
		"--epic", "E-01",
		"--tdd", "required",
		"--title", "Fourth",
		"--depends-on", "M-001,M-002",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("add milestone with --depends-on (seam): %d", rc)
	}

	// On-disk shape: the new milestone carries depends_on with both ids.
	mPath := filepath.Join(root, "work", "epics", "E-01-foundations", "M-004-fourth.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read M-004: %v", err)
	}
	content := string(body)
	for _, want := range []string{"depends_on:", "- M-001", "- M-002"} {
		if !strings.Contains(content, want) {
			t.Errorf("M-004 frontmatter missing %q (seam):\n%s", want, content)
		}
	}

	// Verb trailers: history finds the create commit with the new
	// entity's id, proving the dispatcher's actor / trailer chain ran.
	if rc := run([]string{"history", "M-004", "--root", root}); rc != exitOK {
		t.Errorf("aiwf history M-004 (seam): %d", rc)
	}
}

// TestMilestoneDependsOn_DispatcherSeam_Verb is the seam test for the
// post-allocation verb. Drives `aiwf milestone depends-on …` through
// the dispatcher end-to-end and asserts both the on-disk frontmatter
// shape and the trailered commit are present.
func TestMilestoneDependsOn_DispatcherSeam_Verb(t *testing.T) {
	root := milestoneDependsOnSetup(t)

	rc := run([]string{
		"milestone", "depends-on", "M-003",
		"--on", "M-001,M-002",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != exitOK {
		t.Fatalf("milestone depends-on (seam): %d", rc)
	}

	// On-disk shape.
	mPath := filepath.Join(root, "work", "epics", "E-01-foundations", "M-003-third.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read M-003: %v", err)
	}
	content := string(body)
	for _, want := range []string{"depends_on:", "- M-001", "- M-002"} {
		if !strings.Contains(content, want) {
			t.Errorf("M-003 frontmatter missing %q (seam):\n%s", want, content)
		}
	}

	// `aiwf history M-003` finds the trailered milestone-depends-on
	// commit, proving the verb's trailer chain reached git.
	if rc := run([]string{"history", "M-003", "--root", root}); rc != exitOK {
		t.Errorf("aiwf history M-003 (seam): %d", rc)
	}
}
