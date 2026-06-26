package integration

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/setarea"
)

// setAreaRepo builds a repo with declared members {platform, billing}
// and three epics: E-0001 untagged, E-0002 tagged platform, E-0003
// tagged billing. Returns the repo root.
func setAreaRepo(t *testing.T) string {
	t.Helper()
	root := setupAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Untagged one", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Platform two", "--area", "platform", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Billing three", "--area", "billing", "--actor", "human/test", "--root", root)
	return root
}

// epicFile returns the raw content of the epic file for id.
func epicFile(t *testing.T, root, id string) string {
	t.Helper()
	return readOne(t, root, filepath.Join("work", "epics", id+"-*", "epic.md"))
}

// TestSetArea_AC1_RewritesUntaggedAndRetag pins AC-1: `set-area` rewrites
// the target's area in one commit, in both the untagged->tagged and
// tagged->retagged directions, leaving every other entity byte-identical.
func TestSetArea_AC1_RewritesUntaggedAndRetag(t *testing.T) {
	t.Run("untagged to tagged", func(t *testing.T) {
		root := setAreaRepo(t)
		before := revCount(t, root)
		e2Before := epicFile(t, root, "E-0002")
		e3Before := epicFile(t, root, "E-0003")

		mustRun(t, "set-area", "E-0001", "platform", "--actor", "human/test", "--root", root)

		fm := frontmatterOf(epicFile(t, root, "E-0001"))
		if !strings.Contains(fm, "area: platform") {
			t.Errorf("E-0001 not tagged platform:\n%s", fm)
		}
		if after := revCount(t, root); after != before+1 {
			t.Errorf("commit count = %d, want %d (+1)", after, before+1)
		}
		if got := epicFile(t, root, "E-0002"); got != e2Before {
			t.Errorf("E-0002 changed; should be untouched")
		}
		if got := epicFile(t, root, "E-0003"); got != e3Before {
			t.Errorf("E-0003 changed; should be untouched")
		}
	})

	t.Run("tagged to retagged", func(t *testing.T) {
		root := setAreaRepo(t)
		before := revCount(t, root)
		e1Before := epicFile(t, root, "E-0001")
		e3Before := epicFile(t, root, "E-0003")

		mustRun(t, "set-area", "E-0002", "billing", "--actor", "human/test", "--root", root)

		fm := frontmatterOf(epicFile(t, root, "E-0002"))
		if !strings.Contains(fm, "area: billing") {
			t.Errorf("E-0002 not retagged billing:\n%s", fm)
		}
		if strings.Contains(fm, "area: platform") {
			t.Errorf("E-0002 still carries the old tag:\n%s", fm)
		}
		if after := revCount(t, root); after != before+1 {
			t.Errorf("commit count = %d, want %d (+1)", after, before+1)
		}
		if got := epicFile(t, root, "E-0001"); got != e1Before {
			t.Errorf("E-0001 changed; should be untouched")
		}
		if got := epicFile(t, root, "E-0003"); got != e3Before {
			t.Errorf("E-0003 changed; should be untouched")
		}
	})
}

// TestSetArea_AC2_TrailersAndHistory pins AC-2: the commit carries the
// set-area trailers, `aiwf history` renders the change for both a set and
// a --clear, and `aiwf check` reports no provenance-untrailered finding.
func TestSetArea_AC2_TrailersAndHistory(t *testing.T) {
	root := setAreaRepo(t)

	// A set commit.
	mustRun(t, "set-area", "E-0001", "platform", "--actor", "human/test", "--root", root)
	msg, err := testutil.RunGit(root, "show", "-s", "--format=%B", "HEAD")
	if err != nil {
		t.Fatalf("git show: %v\n%s", err, msg)
	}
	for _, want := range []string{
		"aiwf-verb: set-area",
		"aiwf-entity: E-0001",
		"aiwf-actor: human/test",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("set commit missing trailer %q:\n%s", want, msg)
		}
	}

	// A --clear commit carries the same trailer set.
	mustRun(t, "set-area", "E-0002", "--clear", "--actor", "human/test", "--root", root)
	clearMsg, err := testutil.RunGit(root, "show", "-s", "--format=%B", "HEAD")
	if err != nil {
		t.Fatalf("git show: %v\n%s", err, clearMsg)
	}
	for _, want := range []string{
		"aiwf-verb: set-area",
		"aiwf-entity: E-0002",
		"aiwf-actor: human/test",
	} {
		if !strings.Contains(clearMsg, want) {
			t.Errorf("clear commit missing trailer %q:\n%s", want, clearMsg)
		}
	}

	// aiwf history renders the set-area row for both entities. Assert
	// structurally on the JSON envelope's per-event `verb` field rather
	// than a substring of the text output — the commit subject already
	// contains "set-area", so a substring match would pass even if
	// history never parsed the aiwf-verb trailer into its own column
	// (CLAUDE.md §"Substring assertions are not structural assertions").
	for _, id := range []string{"E-0001", "E-0002"} {
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
			if ev.Verb == "set-area" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("aiwf history %s has no event with verb=set-area:\n%s", id, stdout)
		}
	}

	// aiwf check reports no provenance-untrailered-entity-commit — the
	// verb trailer suppresses the audit a hand-edit would trip.
	_, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--root", root})
	})
	if strings.Contains(stdout+stderr, "provenance-untrailered-entity-commit") {
		t.Errorf("set-area commits should not trip the untrailered-entity audit:\n%s\n%s", stdout, stderr)
	}
}

// TestSetArea_AC3_Refusals pins AC-3: every refusal path leaves the tree
// byte-identical and the commit count unchanged.
func TestSetArea_AC3_Refusals(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		wantInErr   []string
		noAreasRepo bool
	}{
		{
			name:      "unknown id",
			args:      []string{"set-area", "E-9999", "platform"},
			wantInErr: []string{"unknown id"},
		},
		{
			name:      "undeclared member",
			args:      []string{"set-area", "E-0001", "nonsense"},
			wantInErr: []string{"nonsense", "platform", "billing"},
		},
		{
			name:      "milestone target",
			args:      []string{"set-area", "M-0001", "platform"},
			wantInErr: []string{"E-0001", "parent epic"},
		},
		{
			name:      "composite AC target",
			args:      []string{"set-area", "M-0001/AC-1", "platform"},
			wantInErr: []string{"E-0001", "parent epic"},
		},
		{
			name:      "clear and member mutex",
			args:      []string{"set-area", "E-0002", "platform", "--clear"},
			wantInErr: []string{"mutually exclusive"},
		},
		{
			name:      "no-op already tagged",
			args:      []string{"set-area", "E-0002", "platform"},
			wantInErr: []string{"already tagged"},
		},
		{
			name:      "no-op clear already untagged",
			args:      []string{"set-area", "E-0001", "--clear"},
			wantInErr: []string{"already untagged"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := setAreaRepo(t)
			// E-0001 needs a milestone for the milestone/composite cases.
			mustRun(t, "add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Child", "--actor", "human/test", "--root", root)

			before := revCount(t, root)
			e1Before := epicFile(t, root, "E-0001")
			e2Before := epicFile(t, root, "E-0002")

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
			if got := epicFile(t, root, "E-0001"); got != e1Before {
				t.Errorf("E-0001 changed on refusal")
			}
			if got := epicFile(t, root, "E-0002"); got != e2Before {
				t.Errorf("E-0002 changed on refusal")
			}
		})
	}

	t.Run("no areas block declared", func(t *testing.T) {
		root := setupCLITestRepo(t)
		mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
		mustRun(t, "add", "epic", "--title", "Lonely", "--actor", "human/test", "--root", root)
		before := revCount(t, root)

		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"set-area", "E-0001", "platform", "--actor", "human/test", "--root", root})
		})
		if rc == cliutil.ExitOK {
			t.Errorf("rc = ExitOK, want a refusal with no areas block")
		}
		if !strings.Contains(stderr, "not a declared member") {
			t.Errorf("stderr %q should name the (empty) declared set", stderr)
		}
		if after := revCount(t, root); after != before {
			t.Errorf("commit count = %d, want unchanged %d", after, before)
		}
	})
}

// TestSetArea_AC4_TotalReversal pins AC-4: reversal is total via the same
// verb. Three round-trips each return the tree byte-identical to its
// pre-change state.
func TestSetArea_AC4_TotalReversal(t *testing.T) {
	t.Run("untag tag then clear back to untagged", func(t *testing.T) {
		root := setAreaRepo(t)
		initial := epicFile(t, root, "E-0001")

		mustRun(t, "set-area", "E-0001", "platform", "--actor", "human/test", "--root", root)
		mustRun(t, "set-area", "E-0001", "--clear", "--actor", "human/test", "--root", root)

		if got := epicFile(t, root, "E-0001"); got != initial {
			t.Errorf("E-0001 not restored to untagged after tag->clear\n got: %q\nwant: %q", got, initial)
		}
	})

	t.Run("retag forward and back via prior member", func(t *testing.T) {
		root := setAreaRepo(t)
		initial := epicFile(t, root, "E-0002") // platform

		mustRun(t, "set-area", "E-0002", "billing", "--actor", "human/test", "--root", root)
		mustRun(t, "set-area", "E-0002", "platform", "--actor", "human/test", "--root", root)

		if got := epicFile(t, root, "E-0002"); got != initial {
			t.Errorf("E-0002 not restored after retag round-trip\n got: %q\nwant: %q", got, initial)
		}
	})

	t.Run("clear then re-tag via prior member", func(t *testing.T) {
		root := setAreaRepo(t)
		initial := epicFile(t, root, "E-0002") // platform

		mustRun(t, "set-area", "E-0002", "--clear", "--actor", "human/test", "--root", root)
		mustRun(t, "set-area", "E-0002", "platform", "--actor", "human/test", "--root", root)

		if got := epicFile(t, root, "E-0002"); got != initial {
			t.Errorf("E-0002 not restored after clear->re-tag\n got: %q\nwant: %q", got, initial)
		}
	})
}

// TestSetArea_AC5_Discoverability pins AC-5: the composed
// ValidArgsFunction offers entity ids at position 0 and declared members
// at position 1, and the command is registered in the root tree.
func TestSetArea_AC5_Discoverability(t *testing.T) {
	t.Run("entity ids at position 0", func(t *testing.T) {
		root := setAreaRepo(t)
		t.Chdir(root)
		cmd := setarea.NewCmd()
		got, directive := cmd.ValidArgsFunction(cmd, nil, "")
		if directive != cobraNoFileComp {
			t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (%d)", directive, cobraNoFileComp)
		}
		want := map[string]bool{"E-0001": true, "E-0002": true, "E-0003": true}
		for _, g := range got {
			if !want[g] {
				t.Errorf("unexpected completion %q at position 0 (want entity ids)", g)
			}
		}
		for id := range want {
			if !containsStr(got, id) {
				t.Errorf("position-0 completion %v missing entity id %q", got, id)
			}
		}
	})

	t.Run("declared members at position 1", func(t *testing.T) {
		root := setAreaRepo(t)
		t.Chdir(root)
		cmd := setarea.NewCmd()
		got, directive := cmd.ValidArgsFunction(cmd, []string{"E-0001"}, "")
		if directive != cobraNoFileComp {
			t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp", directive)
		}
		want := map[string]bool{"platform": true, "billing": true}
		if len(got) != len(want) {
			t.Fatalf("position-1 completion = %v, want exactly platform, billing", got)
		}
		for _, g := range got {
			if !want[g] {
				t.Errorf("unexpected completion %q at position 1 (want declared members)", g)
			}
		}
	})

	t.Run("nothing offered at position 2", func(t *testing.T) {
		root := setAreaRepo(t)
		t.Chdir(root)
		cmd := setarea.NewCmd()
		got, _ := cmd.ValidArgsFunction(cmd, []string{"E-0001", "platform"}, "")
		if len(got) != 0 {
			t.Errorf("completion at position 2 = %v, want empty", got)
		}
	})

	t.Run("command registered in the tree", func(t *testing.T) {
		t.Parallel()
		rootCmd := cli.NewRootCmd()
		var found bool
		for _, c := range rootCmd.Commands() {
			if c.Name() == "set-area" {
				found = true
				break
			}
		}
		if !found {
			t.Error("set-area not registered in the root command tree")
		}
	})
}

// TestNewCmd_SmokeShape pins the command shape and --help surface: the
// Use string, the --clear flag, and the Long text documenting <member>,
// --clear, and the milestone refusal.
func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := setarea.NewCmd()
	if cmd.Use != "set-area <id> <member>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "set-area <id> <member>")
	}
	if cmd.Flags().Lookup("clear") == nil {
		t.Error("--clear flag not registered")
	}
	for _, want := range []string{"--clear", "areas.members", "parent epic"} {
		if !strings.Contains(cmd.Long, want) {
			t.Errorf("Long help missing %q:\n%s", want, cmd.Long)
		}
	}
	if cmd.ValidArgsFunction == nil {
		t.Error("ValidArgsFunction is nil; completion-drift policy requires a non-nil function")
	}
}

// TestSetArea_AuthorizedAIWithinScope is the POSITIVE inverse of
// TestRenameArea_AuthorizedAIRefused: a scoped ai/claude agent whose
// ACTIVE scope reaches the target entity is ALLOWED to run `set-area`
// because the verb carries a non-empty TargetID (VerbAct reachability is
// satisfiable). The retag lands, exits 0, and the entity's area changes.
//
// REGRESSION GUARD: if a future change clears the ProvenanceContext
// TargetID in setarea.go's Run (mirroring rename-area's human-only
// posture), the scoped AI would no longer reach the scope and would be
// refused — this test would then go red.
func TestSetArea_AuthorizedAIWithinScope(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	root, bin := setupAreaScopeRepo(t)

	stdout, stderr, code := runSplit(t, root, bin,
		"set-area", "E-0001", "billing",
		"--actor", "ai/claude", "--principal", "human/peter", "--format=json")
	if code != 0 {
		t.Fatalf("authorized-AI set-area exit = %d, want 0 (allowed)\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("JSON mode must write nothing to stderr; got:\n%s", stderr)
	}
	var env codedEnvelope
	if jerr := json.Unmarshal([]byte(stdout), &env); jerr != nil {
		t.Fatalf("stdout is not a single JSON envelope: %v\nstdout:\n%s", jerr, stdout)
	}
	if env.Status != "ok" {
		t.Errorf("status = %q, want \"ok\" (the scoped AI is allowed)", env.Status)
	}
	if env.Error != nil {
		t.Errorf("allowed verb must carry no error object; got %+v", env.Error)
	}

	// The retag actually landed: E-0001 now reads area: billing.
	fm := frontmatterOf(epicFile(t, root, "E-0001"))
	if !strings.Contains(fm, "area: billing") {
		t.Errorf("E-0001 not retagged to billing by the authorized AI:\n%s", fm)
	}
}

// cobraNoFileComp is cobra.ShellCompDirectiveNoFileComp (4), inlined to
// avoid importing cobra just for the constant in assertions.
const cobraNoFileComp = 4

// containsStr reports whether s is in xs.
func containsStr(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
