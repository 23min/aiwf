package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// setupAreaRepo initializes a repo with a declared areas block
// (members: platform, billing). Returns the repo root.
func setupAreaRepo(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "areas:\n  members:\n    - platform\n    - billing\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	return root
}

// readOne globs for exactly one file under root and returns its content.
func readOne(t *testing.T, root, glob string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, glob))
	if err != nil || len(matches) != 1 {
		t.Fatalf("glob %s: matches=%v err=%v", glob, matches, err)
	}
	b, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read %s: %v", matches[0], err)
	}
	return string(b)
}

// frontmatterOf returns the YAML frontmatter block of an entity file's
// raw content (the text between the leading `---` and the closing `---`).
func frontmatterOf(s string) string {
	if !strings.HasPrefix(s, "---\n") {
		return ""
	}
	rest := s[len("---\n"):]
	if i := strings.Index(rest, "\n---"); i >= 0 {
		return rest[:i]
	}
	return rest
}

// TestRunAdd_AreaSetViaDispatcher pins M-0173/AC-1 + AC-6 (set path):
// `aiwf add epic --area <declared>` writes the area into the created
// entity's frontmatter through the real dispatcher.
func TestRunAdd_AreaSetViaDispatcher(t *testing.T) {
	root := setupAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Platform work", "--area", "platform", "--actor", "human/test", "--root", root)
	fm := frontmatterOf(readOne(t, root, "work/epics/E-*/epic.md"))
	if !strings.Contains(fm, "area: platform") {
		t.Errorf("epic frontmatter missing `area: platform`:\n%s", fm)
	}
}

// TestRunAdd_AreaRejected pins M-0173/AC-2: an undeclared --area, and an
// --area with no areas block, are both usage errors (exit 2) that create
// no entity and name the offending value (and the declared set).
func TestRunAdd_AreaRejected(t *testing.T) {
	t.Run("undeclared value", func(t *testing.T) {
		root := setupAreaRepo(t)
		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"add", "epic", "--title", "X", "--area", "nonsense", "--actor", "human/test", "--root", root})
		})
		if rc != cliutil.ExitUsage {
			t.Errorf("rc = %d, want ExitUsage (%d)", rc, cliutil.ExitUsage)
		}
		for _, want := range []string{"nonsense", "platform", "billing"} {
			if !strings.Contains(stderr, want) {
				t.Errorf("stderr %q missing %q", stderr, want)
			}
		}
		if matches, _ := filepath.Glob(filepath.Join(root, "work", "epics", "E-*", "epic.md")); len(matches) != 0 {
			t.Errorf("no entity should be created on rejection; found %v", matches)
		}
	})

	t.Run("no areas block", func(t *testing.T) {
		root := setupCLITestRepo(t)
		mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"add", "epic", "--title", "X", "--area", "platform", "--actor", "human/test", "--root", root})
		})
		if rc != cliutil.ExitUsage {
			t.Errorf("rc = %d, want ExitUsage (%d)", rc, cliutil.ExitUsage)
		}
		if !strings.Contains(stderr, "areas") {
			t.Errorf("stderr %q should mention the missing areas block", stderr)
		}
		if matches, _ := filepath.Glob(filepath.Join(root, "work", "epics", "E-*", "epic.md")); len(matches) != 0 {
			t.Errorf("no entity should be created on rejection; found %v", matches)
		}
	})
}

// TestRunAdd_AreaRejectedForMilestone pins M-0173/AC-3 through the
// dispatcher: --area on a milestone errors and creates nothing.
func TestRunAdd_AreaRejectedForMilestone(t *testing.T) {
	root := setupAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Parent", "--actor", "human/test", "--root", root)
	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Child", "--area", "platform", "--actor", "human/test", "--root", root})
	})
	if rc == cliutil.ExitOK {
		t.Errorf("rc = ExitOK, want error for --area on a milestone")
	}
	if !strings.Contains(stderr, "area") || !strings.Contains(stderr, "root") {
		t.Errorf("stderr %q should explain --area is for root kinds only", stderr)
	}
	if matches, _ := filepath.Glob(filepath.Join(root, "work", "epics", "E-*", "M-*.md")); len(matches) != 0 {
		t.Errorf("no milestone should be created on rejection; found %v", matches)
	}
}

// TestRunAdd_GapDerivesArea pins M-0173/AC-5: a gap with --discovered-in
// and no explicit --area derives its area from the discovered-in entity's
// effective area (epic direct, milestone two-hop); an untagged source
// leaves the gap untagged; an explicit --area always overrides.
func TestRunAdd_GapDerivesArea(t *testing.T) {
	cases := []struct {
		name         string
		discoveredIn string
		explicitArea string
		wantArea     string // "" means untagged
	}{
		{"derive from tagged epic", "E-0001", "", "platform"},
		{"derive from milestone two-hop", "M-0001", "", "platform"},
		{"untagged source leaves gap untagged", "E-0002", "", ""},
		{"explicit area overrides derivation", "E-0001", "billing", "billing"},
		{"explicit area with untagged source", "E-0002", "billing", "billing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := setupAreaRepo(t)
			// E-0001 tagged platform, with a milestone M-0001 under it.
			mustRun(t, "add", "epic", "--title", "Tagged", "--area", "platform", "--actor", "human/test", "--root", root)
			mustRun(t, "add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Child", "--actor", "human/test", "--root", root)
			// E-0002 untagged.
			mustRun(t, "add", "epic", "--title", "Untagged", "--actor", "human/test", "--root", root)

			args := []string{"add", "gap", "--title", "Leak", "--discovered-in", tc.discoveredIn, "--actor", "human/test", "--root", root}
			if tc.explicitArea != "" {
				args = append(args, "--area", tc.explicitArea)
			}
			mustRun(t, args...)

			fm := frontmatterOf(readOne(t, root, "work/gaps/G-*.md"))
			if tc.wantArea == "" {
				if strings.Contains(fm, "area:") {
					t.Errorf("expected untagged gap, got area in frontmatter:\n%s", fm)
				}
				return
			}
			if !strings.Contains(fm, "area: "+tc.wantArea) {
				t.Errorf("gap area = (frontmatter below), want `area: %s`:\n%s", tc.wantArea, fm)
			}
		})
	}
}

// TestRunAdd_GapDerivesUndeclaredAreaAsIs pins the deliberate design
// split the reviewer flagged: derivation copies the discovered-in
// entity's EFFECTIVE area verbatim, without re-validating it against the
// declared set. An undeclared source area (only reachable via a hand-edit
// or import, since the write path validates --area) is carried through —
// the M-0172 area-unknown check is the backstop, not the write path. Pin
// it so a future "helpfully re-validate the derived value" change is a
// red test, not a silent behavior change.
func TestRunAdd_GapDerivesUndeclaredAreaAsIs(t *testing.T) {
	root := setupAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Legacy", "--actor", "human/test", "--root", root)
	// Hand-edit the epic to carry an UNDECLARED area (the --area write path
	// would reject this value; a hand-edit / import is the only way in).
	matches, err := filepath.Glob(filepath.Join(root, "work", "epics", "E-*", "epic.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("locate epic: matches=%v err=%v", matches, err)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read epic: %v", err)
	}
	patched := strings.Replace(string(raw), "status: proposed\n", "status: proposed\narea: legacy-undeclared\n", 1)
	if patched == string(raw) {
		t.Fatalf("failed to inject area into epic frontmatter:\n%s", raw)
	}
	if err := os.WriteFile(matches[0], []byte(patched), 0o644); err != nil {
		t.Fatalf("write epic: %v", err)
	}

	mustRun(t, "add", "gap", "--title", "Leak", "--discovered-in", "E-0001", "--actor", "human/test", "--root", root)
	fm := frontmatterOf(readOne(t, root, "work/gaps/G-*.md"))
	if !strings.Contains(fm, "area: legacy-undeclared") {
		t.Errorf("derivation should copy the effective area verbatim (no re-validation); gap frontmatter:\n%s", fm)
	}
}

// TestRunAdd_AreaCompletion pins M-0173/AC-4: the --area completion
// function offers exactly the declared areas.members, and gracefully
// returns nothing when no config is discoverable. Serial (t.Chdir
// mutates process-wide cwd, which ResolveRoot("") reads).
func TestRunAdd_AreaCompletion(t *testing.T) {
	t.Run("offers declared members", func(t *testing.T) {
		root := setupAreaRepo(t)
		t.Chdir(root)
		got, directive := cliutil.CompleteAreaFlag()(nil, nil, "")
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp", directive)
		}
		want := map[string]bool{"platform": true, "billing": true}
		if len(got) != len(want) {
			t.Fatalf("completion = %v, want exactly platform, billing", got)
		}
		for _, g := range got {
			if !want[g] {
				t.Errorf("unexpected completion %q (want only platform, billing)", g)
			}
		}
	})

	t.Run("graceful no-op without config", func(t *testing.T) {
		// A bare tempdir with no aiwf.yaml up the tree: ResolveRoot("")
		// fails and the completion collapses to an empty list rather than
		// erroring in the shell.
		t.Chdir(t.TempDir())
		got, _ := cliutil.CompleteAreaFlag()(nil, nil, "")
		if len(got) != 0 {
			t.Errorf("completion with no config = %v, want empty", got)
		}
	})
}

// TestConfiguredAreaMembers pins the single-source accessor M-0173 reads:
// the declared members for a repo with an areas block, and nil when no
// aiwf.yaml is present (the graceful-tolerant path).
func TestConfiguredAreaMembers(t *testing.T) {
	t.Parallel()
	t.Run("returns declared members", func(t *testing.T) {
		t.Parallel()
		root := setupAreaRepo(t)
		got := cliutil.ConfiguredAreaMembers(root)
		if len(got) != 2 || got[0] != "platform" || got[1] != "billing" {
			t.Errorf("members = %v, want [platform billing]", got)
		}
	})
	t.Run("nil when no aiwf.yaml", func(t *testing.T) {
		t.Parallel()
		if got := cliutil.ConfiguredAreaMembers(t.TempDir()); got != nil {
			t.Errorf("members = %v, want nil (no aiwf.yaml)", got)
		}
	})
}
