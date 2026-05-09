package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
)

// G-055 layer #1: `aiwf add milestone` requires an explicit `--tdd
// <required|advisory|none>` declaration at creation time. Tests in
// this file pin the flag's contract end-to-end through the in-process
// dispatcher: presence required, value validated against the closed
// set, persisted into the milestone's frontmatter, completed as the
// closed set, and rejected on non-milestone kinds.
//
// The flag closes the chokepoint G-055 documented: pre-fix, milestones
// could be created without any tdd: declaration, and the kernel
// silently treated absence as `tdd: none`. Post-fix, the policy
// decision is a single explicit act recorded in the create commit.

// addMilestoneTDDSetup gives every test in this file a freshly-init'd
// repo with one epic in place, returning the repo root. Pre-conditions
// match the rest of the cmd test suite (see setupCLITestRepo).
func addMilestoneTDDSetup(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	if rc := run([]string{"add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add epic: %d", rc)
	}
	return root
}

// TestAddMilestone_TDDFlagRequired pins the chokepoint: `aiwf add
// milestone` without `--tdd` must exit usage-error. Pre-G-055-layer-1
// this invocation succeeded silently with `tdd:` absent (treated as
// none). Post-fix, the operator must state the policy.
func TestAddMilestone_TDDFlagRequired(t *testing.T) {
	root := addMilestoneTDDSetup(t)

	got := run([]string{"add", "milestone", "--epic", "E-0001", "--title", "Bootstrap", "--actor", "human/test", "--root", root})
	if got != exitUsage {
		t.Errorf("add milestone without --tdd = %d, want %d (usage error — G-055 layer 1)", got, exitUsage)
	}
}

// TestAddMilestone_TDDValueValidation rejects values outside the closed
// set {required, advisory, none}.
func TestAddMilestone_TDDValueValidation(t *testing.T) {
	root := addMilestoneTDDSetup(t)

	// A bogus value must produce exitUsage.
	got := run([]string{"add", "milestone", "--epic", "E-0001", "--title", "Bogus", "--tdd", "bogus", "--actor", "human/test", "--root", root})
	if got != exitUsage {
		t.Errorf("add milestone --tdd bogus = %d, want %d", got, exitUsage)
	}

	// Each valid value succeeds. Use a different epic per call so the
	// allocator doesn't collide and so we exercise the three values
	// independently.
	for _, val := range []string{"required", "advisory", "none"} {
		t.Run(val, func(t *testing.T) {
			subRoot := addMilestoneTDDSetup(t)
			rc := run([]string{"add", "milestone", "--epic", "E-0001", "--title", "Bootstrap " + val, "--tdd", val, "--actor", "human/test", "--root", subRoot})
			if rc != exitOK {
				t.Errorf("add milestone --tdd %s = %d, want %d", val, rc, exitOK)
			}
		})
	}
}

// TestAddMilestone_TDDPersisted_Required pins the on-disk shape: the
// created milestone file contains `tdd: required` in its frontmatter.
func TestAddMilestone_TDDPersisted_Required(t *testing.T) {
	root := addMilestoneTDDSetup(t)

	if rc := run([]string{"add", "milestone", "--epic", "E-0001", "--title", "Bootstrap", "--tdd", "required", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	mPath := filepath.Join(root, "work", "epics", "E-0001-foundations", "M-0001-bootstrap.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	if !strings.Contains(string(body), "tdd: required") {
		t.Errorf("milestone frontmatter missing `tdd: required`:\n%s", body)
	}
}

// TestAddMilestone_TDDPersisted_Advisory mirrors the above for advisory.
func TestAddMilestone_TDDPersisted_Advisory(t *testing.T) {
	root := addMilestoneTDDSetup(t)

	if rc := run([]string{"add", "milestone", "--epic", "E-0001", "--title", "Bootstrap", "--tdd", "advisory", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	mPath := filepath.Join(root, "work", "epics", "E-0001-foundations", "M-0001-bootstrap.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	if !strings.Contains(string(body), "tdd: advisory") {
		t.Errorf("milestone frontmatter missing `tdd: advisory`:\n%s", body)
	}
}

// TestAddMilestone_TDDPersisted_None pins that the explicit opt-out
// is recorded — `tdd: none` is *not* the same as the field being
// absent. Pre-fix, absence masqueraded as opt-out; post-fix, opt-out
// is loud.
func TestAddMilestone_TDDPersisted_None(t *testing.T) {
	root := addMilestoneTDDSetup(t)

	if rc := run([]string{"add", "milestone", "--epic", "E-0001", "--title", "Bootstrap", "--tdd", "none", "--actor", "human/test", "--root", root}); rc != exitOK {
		t.Fatalf("add milestone: %d", rc)
	}
	mPath := filepath.Join(root, "work", "epics", "E-0001-foundations", "M-0001-bootstrap.md")
	body, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("read milestone: %v", err)
	}
	if !strings.Contains(string(body), "tdd: none") {
		t.Errorf("milestone frontmatter missing `tdd: none` (explicit opt-out must be loud, not absent):\n%s", body)
	}
}

// TestAddMilestone_TDDFlagCompletes pins shell completion for the
// flag — required by CLAUDE.md's auto-completion principle and the
// drift-prevention test in completion_drift_test.go. The completion
// must return exactly the closed set.
func TestAddMilestone_TDDFlagCompletes(t *testing.T) {
	root := newRootCmd()
	addCmd, _, err := root.Find([]string{"add"})
	if err != nil {
		t.Fatalf("find add: %v", err)
	}
	fn, ok := addCmd.GetFlagCompletionFunc("tdd")
	if !ok {
		t.Fatal("add: --tdd has no completion function (G-055 layer 1 requires shell completion)")
	}
	got, dir := fn(addCmd, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("--tdd completion directive = %v, want NoFileComp", dir)
	}
	gotSorted := append([]string{}, got...)
	sort.Strings(gotSorted)
	want := []string{"advisory", "none", "required"}
	if diff := cmp.Diff(want, gotSorted); diff != "" {
		t.Errorf("--tdd completion values mismatch (-want +got):\n%s", diff)
	}
}

// TestAddMilestone_TDDOnlyForMilestones pins that --tdd is rejected
// on non-milestone kinds. The flag is milestone-policy-shaped and
// has no meaning on epics, ADRs, gaps, decisions, or contracts;
// silently accepting it on those kinds would invite confusion.
func TestAddMilestone_TDDOnlyForMilestones(t *testing.T) {
	root := setupCLITestRepo(t)

	cases := []struct {
		name string
		args []string
	}{
		{"epic", []string{"add", "epic", "--title", "X", "--tdd", "required"}},
		{"gap", []string{"add", "gap", "--title", "X", "--tdd", "required"}},
		{"adr", []string{"add", "adr", "--title", "X", "--tdd", "required"}},
		{"decision", []string{"add", "decision", "--title", "X", "--tdd", "required"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{}, tc.args...)
			args = append(args, "--actor", "human/test", "--root", root)
			rc := run(args)
			if rc != exitUsage {
				t.Errorf("add %s --tdd required = %d, want %d (--tdd is milestone-only)", tc.name, rc, exitUsage)
			}
		})
	}
}
