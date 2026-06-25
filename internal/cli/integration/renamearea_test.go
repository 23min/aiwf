package integration

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// areaRepoWithEntities builds a repo with members {platform, billing}
// and three tagged epics: E-0001 + E-0002 on platform, E-0003 on
// billing. Returns the repo root.
func areaRepoWithEntities(t *testing.T) string {
	t.Helper()
	root := setupAreaRepo(t)
	mustRun(t, "add", "epic", "--title", "Platform one", "--area", "platform", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Platform two", "--area", "platform", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Billing one", "--area", "billing", "--actor", "human/test", "--root", root)
	return root
}

func revCount(t *testing.T, root string) int {
	t.Helper()
	out, err := testutil.RunGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list: %v\n%s", err, out)
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		t.Fatalf("parse rev-count %q: %v", out, err)
	}
	return n
}

func readAiwfYAML(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "aiwf.yaml"))
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	return string(b)
}

// TestRenameArea_AC1_RewritesMemberAndEntitiesAtomically pins AC-1:
// `rename-area platform infra` rewrites the aiwf.yaml member and every
// platform-tagged entity, leaves the billing entity untouched, and
// produces exactly one new commit.
func TestRenameArea_AC1_RewritesMemberAndEntitiesAtomically(t *testing.T) {
	root := areaRepoWithEntities(t)
	before := revCount(t, root)

	mustRun(t, "rename-area", "platform", "infra", "--actor", "human/test", "--root", root)

	// aiwf.yaml: members are now infra, billing.
	yaml := readAiwfYAML(t, root)
	if !strings.Contains(yaml, "- infra") || !strings.Contains(yaml, "- billing") {
		t.Errorf("aiwf.yaml members not rewritten to {infra, billing}:\n%s", yaml)
	}
	if strings.Contains(yaml, "- platform") {
		t.Errorf("aiwf.yaml still carries the old member:\n%s", yaml)
	}

	// Each platform epic now reads area: infra; the billing epic is
	// untouched.
	for _, id := range []string{"E-0001", "E-0002"} {
		fm := frontmatterOf(readOne(t, root, filepath.Join("work", "epics", id+"-*", "epic.md")))
		if !strings.Contains(fm, "area: infra") {
			t.Errorf("%s not retagged to infra:\n%s", id, fm)
		}
	}
	billing := frontmatterOf(readOne(t, root, filepath.Join("work", "epics", "E-0003-*", "epic.md")))
	if !strings.Contains(billing, "area: billing") {
		t.Errorf("billing epic E-0003 should be untouched:\n%s", billing)
	}

	// Exactly one new commit.
	if after := revCount(t, root); after != before+1 {
		t.Errorf("commit count = %d, want %d (+1)", after, before+1)
	}
}

// TestRenameArea_AC2_TrailersAndHistory pins AC-2: the single commit
// carries aiwf-verb: rename-area, aiwf-actor: human/test, and an
// aiwf-entity: trailer per rewritten entity; `aiwf history` on a
// rewritten entity renders the rename event.
func TestRenameArea_AC2_TrailersAndHistory(t *testing.T) {
	root := areaRepoWithEntities(t)
	mustRun(t, "rename-area", "platform", "infra", "--actor", "human/test", "--root", root)

	msg, err := testutil.RunGit(root, "show", "-s", "--format=%B", "HEAD")
	if err != nil {
		t.Fatalf("git show: %v\n%s", err, msg)
	}
	for _, want := range []string{
		"aiwf-verb: rename-area",
		"aiwf-actor: human/test",
		"aiwf-entity: E-0001",
		"aiwf-entity: E-0002",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("commit message missing trailer %q:\n%s", want, msg)
		}
	}
	// The billing entity is not rewritten, so it carries no trailer.
	if strings.Contains(msg, "aiwf-entity: E-0003") {
		t.Errorf("billing entity should not get an entity trailer:\n%s", msg)
	}

	// `aiwf history` on a rewritten entity surfaces the rename event.
	_, stdout, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"history", "E-0001", "--root", root})
	})
	if !strings.Contains(stdout, "rename-area") {
		t.Errorf("aiwf history E-0001 does not render the rename event:\n%s", stdout)
	}
}

// TestRenameArea_AC3_RefusesAndNoPartialWrite pins AC-3: an undeclared
// <old> and an already-declared <new> both error (non-OK exit), name
// the declared set, and write nothing — no entity change, no commit.
func TestRenameArea_AC3_RefusesAndNoPartialWrite(t *testing.T) {
	t.Run("undeclared old", func(t *testing.T) {
		root := areaRepoWithEntities(t)
		before := revCount(t, root)
		yamlBefore := readAiwfYAML(t, root)

		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"rename-area", "nonsense", "infra", "--actor", "human/test", "--root", root})
		})
		if rc == cliutil.ExitOK {
			t.Errorf("rc = ExitOK, want error for undeclared <old>")
		}
		for _, want := range []string{"nonsense", "platform", "billing"} {
			if !strings.Contains(stderr, want) {
				t.Errorf("stderr %q missing %q", stderr, want)
			}
		}
		if got := readAiwfYAML(t, root); got != yamlBefore {
			t.Errorf("aiwf.yaml changed on refusal:\n%s", got)
		}
		if after := revCount(t, root); after != before {
			t.Errorf("commit count = %d, want unchanged %d", after, before)
		}
	})

	t.Run("new already declared", func(t *testing.T) {
		root := areaRepoWithEntities(t)
		before := revCount(t, root)
		yamlBefore := readAiwfYAML(t, root)

		rc, _, stderr := testutil.CaptureRun(t, func() int {
			return cli.Execute([]string{"rename-area", "platform", "billing", "--actor", "human/test", "--root", root})
		})
		if rc == cliutil.ExitOK {
			t.Errorf("rc = ExitOK, want error for already-declared <new>")
		}
		if !strings.Contains(stderr, "billing") || !strings.Contains(stderr, "already") {
			t.Errorf("stderr %q should explain billing is already declared", stderr)
		}
		if got := readAiwfYAML(t, root); got != yamlBefore {
			t.Errorf("aiwf.yaml changed on refusal:\n%s", got)
		}
		if after := revCount(t, root); after != before {
			t.Errorf("commit count = %d, want unchanged %d", after, before)
		}
	})
}

// TestRenameArea_AC4_ReverseRestoresInitialState pins AC-4: after
// `rename-area platform infra`, `rename-area infra platform` restores
// the original member name and every entity tag (round-trip).
func TestRenameArea_AC4_ReverseRestoresInitialState(t *testing.T) {
	root := areaRepoWithEntities(t)
	yamlInitial := readAiwfYAML(t, root)
	e1Initial := readOne(t, root, filepath.Join("work", "epics", "E-0001-*", "epic.md"))
	e3Initial := readOne(t, root, filepath.Join("work", "epics", "E-0003-*", "epic.md"))

	mustRun(t, "rename-area", "platform", "infra", "--actor", "human/test", "--root", root)
	mustRun(t, "rename-area", "infra", "platform", "--actor", "human/test", "--root", root)

	if got := readAiwfYAML(t, root); got != yamlInitial {
		t.Errorf("aiwf.yaml not restored after round-trip\n got: %q\nwant: %q", got, yamlInitial)
	}
	if got := readOne(t, root, filepath.Join("work", "epics", "E-0001-*", "epic.md")); got != e1Initial {
		t.Errorf("E-0001 not restored after round-trip\n got: %q\nwant: %q", got, e1Initial)
	}
	if got := readOne(t, root, filepath.Join("work", "epics", "E-0003-*", "epic.md")); got != e3Initial {
		t.Errorf("E-0003 (billing, untouched) changed across round-trip\n got: %q\nwant: %q", got, e3Initial)
	}
}

// TestRenameArea_AC5_Discoverability pins AC-5: CompleteAreaArg(0)
// offers exactly the declared members at position 0 and nothing at
// position 1, and the command is registered under `aiwf --help`.
// The skill-coverage and completion-drift policies (asserted in their
// own packages) cover the allowlist + ValidArgsFunction wiring.
func TestRenameArea_AC5_Discoverability(t *testing.T) {
	t.Run("completion offers declared members at position 0", func(t *testing.T) {
		root := setupAreaRepo(t)
		t.Chdir(root)
		got, directive := cliutil.CompleteAreaArg(0)(nil, nil, "")
		if directive != 4 /* ShellCompDirectiveNoFileComp */ {
			t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
		}
		want := map[string]bool{"platform": true, "billing": true}
		if len(got) != len(want) {
			t.Fatalf("completion = %v, want exactly platform, billing", got)
		}
		for _, g := range got {
			if !want[g] {
				t.Errorf("unexpected completion %q", g)
			}
		}
	})

	t.Run("nothing offered at position 1", func(t *testing.T) {
		root := setupAreaRepo(t)
		t.Chdir(root)
		// Wired as ValidArgsFunction=CompleteAreaArg(0): once <old> is
		// supplied, completing <new> passes args=["platform"], len 1 != 0.
		got, _ := cliutil.CompleteAreaArg(0)(nil, []string{"platform"}, "")
		if len(got) != 0 {
			t.Errorf("completion at position 1 = %v, want empty", got)
		}
	})

	t.Run("command registered in the tree", func(t *testing.T) {
		t.Parallel()
		rootCmd := cli.NewRootCmd()
		var found bool
		for _, c := range rootCmd.Commands() {
			if c.Name() == "rename-area" {
				found = true
				break
			}
		}
		if !found {
			t.Error("rename-area not registered in the root command tree")
		}
	})
}
