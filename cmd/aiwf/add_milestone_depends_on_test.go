package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// M-076/AC-1: `aiwf add milestone --depends-on M-PPP[,M-QQQ]` allocates
// the milestone and atomically writes the depends_on frontmatter array
// in the same commit. Closes G-072 — the kernel asymmetry where
// depends_on had six read sites and zero writers.
//
// This file pins the flag's contract end-to-end through the in-process
// dispatcher: presence is optional, accepts comma-separated lists,
// validates referent existence at allocation time (AC-4), and refuses
// non-milestone referents.

// addMilestoneDependsOnSetup gives every test in this file a freshly-
// init'd repo with one epic and two milestones (M-001, M-002) the
// subsequent --depends-on tests can reference.
func addMilestoneDependsOnSetup(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "First", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add M-001: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Second", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add M-002: %d", rc)
	}
	return root
}

// TestAddMilestone_DependsOnSingle pins AC-1's basic contract: a single
// id passed to --depends-on lands as a one-element depends_on list in
// the new milestone's frontmatter, in the same atomic create commit.
func TestAddMilestone_DependsOnSingle(t *testing.T) {
	t.Parallel()
	root := addMilestoneDependsOnSetup(t)

	rc := run([]string{
		"add", "milestone",
		"--epic", "E-0001",
		"--tdd", "none",
		"--title", "Third",
		"--depends-on", "M-0001",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("add milestone --depends-on M-001 = %d, want %d", rc, cliutil.ExitOK)
	}

	mPath := filepath.Join(root, "work", "epics", "E-0001-foundations", "M-0003-third.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	if !strings.Contains(string(body), "depends_on:") {
		t.Errorf("milestone frontmatter missing `depends_on:` block:\n%s", body)
	}
	if !strings.Contains(string(body), "- M-0001") {
		t.Errorf("milestone frontmatter missing `- M-0001` entry:\n%s", body)
	}
}

// TestAddMilestone_DependsOnMultiple pins the comma-separated list
// shape: --depends-on M-001,M-002 lands as a two-element list.
func TestAddMilestone_DependsOnMultiple(t *testing.T) {
	t.Parallel()
	root := addMilestoneDependsOnSetup(t)

	rc := run([]string{
		"add", "milestone",
		"--epic", "E-0001",
		"--tdd", "none",
		"--title", "Third",
		"--depends-on", "M-0001,M-0002",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("add milestone --depends-on M-001,M-002 = %d, want %d", rc, cliutil.ExitOK)
	}

	mPath := filepath.Join(root, "work", "epics", "E-0001-foundations", "M-0003-third.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	content := string(body)
	if !strings.Contains(content, "- M-0001") {
		t.Errorf("milestone frontmatter missing `- M-0001`:\n%s", content)
	}
	if !strings.Contains(content, "- M-0002") {
		t.Errorf("milestone frontmatter missing `- M-0002`:\n%s", content)
	}
}

// TestAddMilestone_DependsOnAbsent locks in that absence of the flag
// produces no depends_on block — the field is optional, the YAML
// `depends_on,omitempty` tag must hold.
func TestAddMilestone_DependsOnAbsent(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := run([]string{"add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Solo", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	mPath := filepath.Join(root, "work", "epics", "E-0001-foundations", "M-0001-solo.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	if strings.Contains(string(body), "depends_on:") {
		t.Errorf("milestone without --depends-on should not emit a depends_on block:\n%s", body)
	}
}

// TestAddMilestone_DependsOnRejectedOnNonMilestone (AC-1 negative):
// --depends-on is milestone-only. Passing it on `aiwf add gap` (or any
// non-milestone kind) is a usage error.
func TestAddMilestone_DependsOnRejectedOnNonMilestone(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	rc := run([]string{
		"add", "gap",
		"--title", "stray",
		"--depends-on", "M-0001",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("add gap --depends-on = %d, want %d (usage error)", rc, cliutil.ExitUsage)
	}
}

// TestAddMilestone_DependsOnUnknownReferent (AC-4): every id passed to
// --depends-on must resolve to an existing milestone. An unknown id
// is refused before the create commit lands; the error names the
// specific unresolvable id.
func TestAddMilestone_DependsOnUnknownReferent(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add epic: %d", rc)
	}

	rc := run([]string{
		"add", "milestone",
		"--epic", "E-0001",
		"--tdd", "none",
		"--title", "Bogus",
		"--depends-on", "M-0999",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("add milestone --depends-on M-999 = %d, want %d (M-999 doesn't exist)", rc, cliutil.ExitUsage)
	}

	// The would-be milestone file must NOT have been written.
	mPath := filepath.Join(root, "work", "epics", "E-0001-foundations", "M-0001-bogus.md")
	if _, err := os.Stat(mPath); !os.IsNotExist(err) {
		t.Errorf("milestone file should not exist after referent rejection: stat err = %v", err)
	}
}

// TestAddMilestone_DependsOnNonMilestoneReferent (AC-4): --depends-on
// is restricted to milestone→milestone edges per the schema's
// AllowedKinds. Passing an epic id (or any non-milestone) is refused.
func TestAddMilestone_DependsOnNonMilestoneReferent(t *testing.T) {
	t.Parallel()
	root := addMilestoneDependsOnSetup(t)

	rc := run([]string{
		"add", "milestone",
		"--epic", "E-0001",
		"--tdd", "none",
		"--title", "WrongKind",
		"--depends-on", "E-0001",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("add milestone --depends-on E-01 = %d, want %d (E-01 is not a milestone)", rc, cliutil.ExitUsage)
	}
}

// TestAddMilestone_DependsOnPartialUnknown (AC-4): when the list has
// some valid and some invalid ids, the verb refuses the whole call
// (no partial writes).
func TestAddMilestone_DependsOnPartialUnknown(t *testing.T) {
	t.Parallel()
	root := addMilestoneDependsOnSetup(t)

	rc := run([]string{
		"add", "milestone",
		"--epic", "E-0001",
		"--tdd", "none",
		"--title", "Mixed",
		"--depends-on", "M-001,M-999",
		"--actor", "human/test",
		"--root", root,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("add milestone --depends-on M-001,M-999 = %d, want %d", rc, cliutil.ExitUsage)
	}
}
