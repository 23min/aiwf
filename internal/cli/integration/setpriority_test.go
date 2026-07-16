package integration

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/setpriority"
)

const (
	fixtureGapBody      = "## What's missing\n\nFixture prose for test setup; not the subject under test.\n\n## Why it matters\n\nFixture prose for test setup; not the subject under test.\n"
	fixtureDecisionBody = "## Question\n\nFixture prose for test setup; not the subject under test.\n\n## Decision\n\nFixture prose for test setup; not the subject under test.\n\n## Reasoning\n\nFixture prose for test setup; not the subject under test.\n"
)

// setPriorityRepo builds a repo with one gap G-0001 (unset priority),
// one decision D-0001 (unset priority), and one epic E-0001 (a kind
// that never carries a priority). Returns the repo root.
func setPriorityRepo(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", fixtureGapBody, "--title", "Leak", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "decision", "--body", fixtureDecisionBody, "--title", "Pick one", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Epic one", "--actor", "human/test", "--root", root)
	return root
}

// gapFile returns the raw content of the gap file for id.
func gapFile(t *testing.T, root, id string) string {
	t.Helper()
	return readOne(t, root, filepath.Join("work", "gaps", id+"-*.md"))
}

// decisionFile returns the raw content of the decision file for id.
func decisionFile(t *testing.T, root, id string) string {
	t.Helper()
	return readOne(t, root, filepath.Join("work", "decisions", id+"-*.md"))
}

// TestSetPriority_AC1_SetsAndResets pins AC-1: `set-priority` rewrites
// the target's priority in one commit, in both the unset->set and
// set->reset directions, leaving every other entity byte-identical.
func TestSetPriority_AC1_SetsAndResets(t *testing.T) {
	t.Run("unset to urgent", func(t *testing.T) {
		root := setPriorityRepo(t)
		before := revCount(t, root)
		d1Before := decisionFile(t, root, "D-0001")

		mustRun(t, "set-priority", "G-0001", "urgent", "--actor", "human/test", "--root", root)

		fm := frontmatterOf(gapFile(t, root, "G-0001"))
		if !strings.Contains(fm, "priority: urgent") {
			t.Errorf("G-0001 not set to urgent:\n%s", fm)
		}
		if after := revCount(t, root); after != before+1 {
			t.Errorf("commit count = %d, want %d (+1)", after, before+1)
		}
		if got := decisionFile(t, root, "D-0001"); got != d1Before {
			t.Errorf("D-0001 changed; should be untouched")
		}
	})

	t.Run("set to reset", func(t *testing.T) {
		root := setPriorityRepo(t)
		mustRun(t, "set-priority", "G-0001", "high", "--actor", "human/test", "--root", root)
		before := revCount(t, root)

		mustRun(t, "set-priority", "G-0001", "low", "--actor", "human/test", "--root", root)

		fm := frontmatterOf(gapFile(t, root, "G-0001"))
		if !strings.Contains(fm, "priority: low") {
			t.Errorf("G-0001 not reset to low:\n%s", fm)
		}
		if strings.Contains(fm, "priority: high") {
			t.Errorf("G-0001 still carries the old level:\n%s", fm)
		}
		if after := revCount(t, root); after != before+1 {
			t.Errorf("commit count = %d, want %d (+1)", after, before+1)
		}
	})

	t.Run("decision target", func(t *testing.T) {
		root := setPriorityRepo(t)
		mustRun(t, "set-priority", "D-0001", "medium", "--actor", "human/test", "--root", root)
		fm := frontmatterOf(decisionFile(t, root, "D-0001"))
		if !strings.Contains(fm, "priority: medium") {
			t.Errorf("D-0001 not set to medium:\n%s", fm)
		}
	})
}

// TestSetPriority_AC2_TrailersAndHistory pins AC-2 (write half): the
// commit carries the set-priority trailers, `aiwf history` renders the
// change for both a set and a --clear, and `aiwf check` reports no
// provenance-untrailered finding.
func TestSetPriority_AC2_TrailersAndHistory(t *testing.T) {
	root := setPriorityRepo(t)

	// A set commit.
	mustRun(t, "set-priority", "G-0001", "urgent", "--actor", "human/test", "--root", root)
	msg, err := testutil.RunGit(root, "show", "-s", "--format=%B", "HEAD")
	if err != nil {
		t.Fatalf("git show: %v\n%s", err, msg)
	}
	for _, want := range []string{
		"aiwf-verb: set-priority",
		"aiwf-entity: G-0001",
		"aiwf-actor: human/test",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("set commit missing trailer %q:\n%s", want, msg)
		}
	}

	// A --clear commit carries the same trailer set.
	mustRun(t, "set-priority", "D-0001", "medium", "--actor", "human/test", "--root", root)
	mustRun(t, "set-priority", "D-0001", "--clear", "--actor", "human/test", "--root", root)
	clearMsg, err := testutil.RunGit(root, "show", "-s", "--format=%B", "HEAD")
	if err != nil {
		t.Fatalf("git show: %v\n%s", err, clearMsg)
	}
	for _, want := range []string{
		"aiwf-verb: set-priority",
		"aiwf-entity: D-0001",
		"aiwf-actor: human/test",
	} {
		if !strings.Contains(clearMsg, want) {
			t.Errorf("clear commit missing trailer %q:\n%s", want, clearMsg)
		}
	}

	// aiwf history renders the set-priority row for both entities.
	// Assert structurally on the JSON envelope's per-event `verb` field
	// rather than a substring of the text output (CLAUDE.md
	// §"Substring assertions are not structural assertions").
	for _, id := range []string{"G-0001", "D-0001"} {
		_, stdout, _ := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"history", id, "--format", "json", "--root", root})
		})
		var env struct {
			Result struct {
				Events []struct {
					Verb string `json:"verb"`
				} `json:"events"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(stdout), &env); err != nil {
			t.Fatalf("aiwf history %s --format json: unmarshal: %v\n%s", id, err, stdout)
		}
		found := false
		for _, ev := range env.Result.Events {
			if ev.Verb == "set-priority" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("aiwf history %s has no event with verb=set-priority:\n%s", id, stdout)
		}
	}

	// aiwf check reports no provenance-untrailered-entity-commit — the
	// verb trailer suppresses the audit a hand-edit would trip.
	_, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--root", root})
	})
	if strings.Contains(stdout+stderr, "provenance-untrailered-entity-commit") {
		t.Errorf("set-priority commits should not trip the untrailered-entity audit:\n%s\n%s", stdout, stderr)
	}
}

// TestSetPriority_AC2_Refusals pins AC-2 (refusal half): every refusal
// path leaves the tree byte-identical and the commit count unchanged.
func TestSetPriority_AC2_Refusals(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantInErr []string
	}{
		{
			name:      "unknown id",
			args:      []string{"set-priority", "G-9999", "urgent"},
			wantInErr: []string{"unknown id"},
		},
		{
			name:      "non-gap/decision target",
			args:      []string{"set-priority", "E-0001", "urgent"},
			wantInErr: []string{"does not carry a priority"},
		},
		{
			name:      "out-of-range level",
			args:      []string{"set-priority", "G-0001", "critical"},
			wantInErr: []string{"critical", "not a recognized priority level"},
		},
		{
			name:      "clear and level mutex",
			args:      []string{"set-priority", "G-0001", "urgent", "--clear"},
			wantInErr: []string{"mutually exclusive"},
		},
		{
			name:      "neither level nor clear",
			args:      []string{"set-priority", "G-0001"},
			wantInErr: []string{"pass <level>", "--clear"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := setPriorityRepo(t)
			before := revCount(t, root)
			g1Before := gapFile(t, root, "G-0001")

			args := append([]string{}, tc.args...)
			args = append(args, "--actor", "human/test", "--root", root)
			rc, _, stderr := testutil.CaptureRun(t, func() int {
				return cli.Execute(args)
			})
			if rc == cliutil.ExitOK {
				t.Errorf("rc = ExitOK, want a refusal for %s", tc.name)
			}
			for _, want := range tc.wantInErr {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr %q missing %q", stderr, want)
				}
			}
			if after := revCount(t, root); after != before {
				t.Errorf("commit count = %d, want unchanged %d", after, before)
			}
			if got := gapFile(t, root, "G-0001"); got != g1Before {
				t.Errorf("G-0001 changed on refusal")
			}
		})
	}

	t.Run("no-op already set", func(t *testing.T) {
		root := setPriorityRepo(t)
		mustRun(t, "set-priority", "G-0001", "high", "--actor", "human/test", "--root", root)
		before := revCount(t, root)

		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"set-priority", "G-0001", "high", "--actor", "human/test", "--root", root})
		})
		if rc == cliutil.ExitOK {
			t.Errorf("rc = ExitOK, want a refusal for no-op")
		}
		if !strings.Contains(stderr, "already set to") {
			t.Errorf("stderr %q should name the no-op refusal", stderr)
		}
		if after := revCount(t, root); after != before {
			t.Errorf("commit count = %d, want unchanged %d", after, before)
		}
	})

	t.Run("no-op clear already unset", func(t *testing.T) {
		root := setPriorityRepo(t)
		before := revCount(t, root)

		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"set-priority", "G-0001", "--clear", "--actor", "human/test", "--root", root})
		})
		if rc == cliutil.ExitOK {
			t.Errorf("rc = ExitOK, want a refusal for no-op clear")
		}
		if !strings.Contains(stderr, "already unset") {
			t.Errorf("stderr %q should name the no-op refusal", stderr)
		}
		if after := revCount(t, root); after != before {
			t.Errorf("commit count = %d, want unchanged %d", after, before)
		}
	})
}

// TestSetPriority_TotalReversal pins the "what verb undoes this" design
// requirement: reversal is total via the same verb. Round-trips return
// the tree byte-identical to its pre-change state.
func TestSetPriority_TotalReversal(t *testing.T) {
	t.Run("set then clear back to unset", func(t *testing.T) {
		root := setPriorityRepo(t)
		initial := gapFile(t, root, "G-0001")

		mustRun(t, "set-priority", "G-0001", "urgent", "--actor", "human/test", "--root", root)
		mustRun(t, "set-priority", "G-0001", "--clear", "--actor", "human/test", "--root", root)

		if got := gapFile(t, root, "G-0001"); got != initial {
			t.Errorf("G-0001 not restored to unset after set->clear\n got: %q\nwant: %q", got, initial)
		}
	})

	t.Run("reset forward and back via prior level", func(t *testing.T) {
		root := setPriorityRepo(t)
		mustRun(t, "set-priority", "G-0001", "high", "--actor", "human/test", "--root", root)
		initial := gapFile(t, root, "G-0001")

		mustRun(t, "set-priority", "G-0001", "low", "--actor", "human/test", "--root", root)
		mustRun(t, "set-priority", "G-0001", "high", "--actor", "human/test", "--root", root)

		if got := gapFile(t, root, "G-0001"); got != initial {
			t.Errorf("G-0001 not restored after reset round-trip\n got: %q\nwant: %q", got, initial)
		}
	})
}

// TestSetPriority_Discoverability pins AC-4: the composed
// ValidArgsFunction offers entity ids at position 0 and the fixed
// priority-level set at position 1, and the command is registered in
// the root tree.
func TestSetPriority_Discoverability(t *testing.T) {
	t.Run("entity ids at position 0", func(t *testing.T) {
		root := setPriorityRepo(t)
		t.Chdir(root)
		cmd := setpriority.NewCmd("")
		got, directive := cmd.ValidArgsFunction(cmd, nil, "")
		if directive != cobraNoFileComp {
			t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (%d)", directive, cobraNoFileComp)
		}
		want := map[string]bool{"G-0001": true, "D-0001": true, "E-0001": true}
		for id := range want {
			if !containsStr(got, id) {
				t.Errorf("position-0 completion %v missing entity id %q", got, id)
			}
		}
	})

	t.Run("priority levels at position 1", func(t *testing.T) {
		root := setPriorityRepo(t)
		t.Chdir(root)
		cmd := setpriority.NewCmd("")
		got, directive := cmd.ValidArgsFunction(cmd, []string{"G-0001"}, "")
		if directive != cobraNoFileComp {
			t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp", directive)
		}
		want := map[string]bool{"urgent": true, "high": true, "medium": true, "low": true}
		if len(got) != len(want) {
			t.Fatalf("position-1 completion = %v, want exactly the four priority levels", got)
		}
		for _, g := range got {
			if !want[g] {
				t.Errorf("unexpected completion %q at position 1", g)
			}
		}
	})

	t.Run("nothing offered at position 2", func(t *testing.T) {
		root := setPriorityRepo(t)
		t.Chdir(root)
		cmd := setpriority.NewCmd("")
		got, _ := cmd.ValidArgsFunction(cmd, []string{"G-0001", "urgent"}, "")
		if len(got) != 0 {
			t.Errorf("completion at position 2 = %v, want empty", got)
		}
	})

	t.Run("command registered in the tree", func(t *testing.T) {
		t.Parallel()
		rootCmd := cli.NewRootCmd("")
		var found bool
		for _, c := range rootCmd.Commands() {
			if c.Name() == "set-priority" {
				found = true
				break
			}
		}
		if !found {
			t.Error("set-priority not registered in the root command tree")
		}
	})
}

// TestSetPriorityNewCmd_SmokeShape pins the command shape and --help
// surface: the Use string, the --clear flag, and the Long text
// documenting <level>, --clear, and the non-gap/decision refusal.
func TestSetPriorityNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := setpriority.NewCmd("")
	if cmd.Use != "set-priority <id> <level>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "set-priority <id> <level>")
	}
	if cmd.Flags().Lookup("clear") == nil {
		t.Error("--clear flag not registered")
	}
	for _, want := range []string{"--clear", "urgent", "gap and decision"} {
		if !strings.Contains(cmd.Long, want) {
			t.Errorf("Long help missing %q:\n%s", want, cmd.Long)
		}
	}
	if cmd.ValidArgsFunction == nil {
		t.Error("ValidArgsFunction is nil; completion-drift policy requires a non-nil function")
	}
}
