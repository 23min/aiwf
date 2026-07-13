package add_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli/add"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entity"
)

// M-0253/AC-1 backfill: add.Run and its `aiwf add ac` sibling runAC
// carry the largest concentration of entity-lifecycle guards
// branch-coverage-audit flags. This file drives each flagged guard
// directly, reusing M-0252's shared fixtures where the failure mode
// matches (actor resolution) and building a verb-specific trigger
// otherwise (a nonexistent --body-file, a malformed contracts block,
// repeated --title, the `ac` subcommand's own flag-shape guards).
// The ResolveRoot and LoadTreeWithTrunk/tree.Load "fatal IO error"
// branches are `//coverage:ignore`d in add.go itself, mirroring the
// established internal/cli/archive, internal/cli/renamearea, and
// internal/cli/setarea precedent — those errors are not
// deterministically reproducible in a unit-test harness.

// runArgs bundles add.Run's many positional parameters with
// zero-value defaults so each test below only overrides what it
// needs to reach its target branch.
type runArgs struct {
	kind          entity.Kind
	title         string
	actor         string
	principal     string
	root          string
	epicID        string
	tddPolicy     string
	dependsOn     string
	discoveredIn  string
	area          string
	pathHint      string
	relatesTo     string
	linkedADRs    string
	bindValidator string
	bindSchema    string
	bindFixtures  string
	bodyFile      string
	bodyText      string
	reason        string
	fetch         bool
	force         bool
	out           cliutil.OutputFormat
}

func (a runArgs) run() int {
	return add.Run(a.kind, a.title, a.actor, a.principal, a.root,
		a.epicID, a.tddPolicy, a.dependsOn, a.discoveredIn, a.area, a.pathHint, a.relatesTo, a.linkedADRs,
		a.bindValidator, a.bindSchema, a.bindFixtures, a.bodyFile, a.bodyText, a.reason, a.fetch, a.force, a.out)
}

// execExitCode drives cmd through Cobra's real Execute path (the only
// way to reach RunE's own guard logic and the unexported runAC, which
// newACCmd wires as the `ac` subcommand's RunE) and unwraps the
// resulting *cliutil.ExitError.
func execExitCode(t *testing.T, cmd *cobra.Command, args []string) int {
	t.Helper()
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err == nil {
		return cliutil.ExitOK
	}
	var ee *cliutil.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("Execute() error = %v (%T), want *cliutil.ExitError", err, err)
	}
	return ee.Code
}

// TestNewCmd_RunE_TooManyArgs covers add.go's RunE closure guard: more
// than one positional arg after the kind is a usage error. cobra's
// Args: cobra.MinimumNArgs(1) only enforces a floor, so this needs a
// real Execute() to reach the RunE-local check.
func TestNewCmd_RunE_TooManyArgs(t *testing.T) {
	t.Parallel()
	if rc := execExitCode(t, add.NewCmd(""), []string{"epic", "extra"}); rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestNewCmd_RunE_RepeatedTitleRejected covers the RunE closure's
// second guard: --title may not repeat for a non-ac kind (repetition
// is reserved for `aiwf add ac`'s batch-creation shape).
func TestNewCmd_RunE_RepeatedTitleRejected(t *testing.T) {
	t.Parallel()
	if rc := execExitCode(t, add.NewCmd(""), []string{"epic", "--title", "T1", "--title", "T2"}); rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_BodyFileAndBodyTextMutuallyExclusive covers the G-0326
// ride-along body source guard: --body and --body-file are mutually
// exclusive, checked before any root/tree work.
func TestRun_BodyFileAndBodyTextMutuallyExclusive(t *testing.T) {
	t.Parallel()
	rc := runArgs{kind: entity.KindEpic, bodyFile: "x.md", bodyText: "hello"}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_ForceWithoutReason covers the G-0326 --force gate: --force
// requires a non-empty --reason, checked before any root/tree work.
func TestRun_ForceWithoutReason(t *testing.T) {
	t.Parallel()
	rc := runArgs{kind: entity.KindEpic, force: true}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_ResolveActorFailure covers Run's cliutil.ResolveActor guard
// using M-0252's BrokenGitIdentity fixture. Serial: BrokenGitIdentity
// uses t.Setenv, which panics under t.Parallel.
func TestRun_ResolveActorFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)
	root := t.TempDir()
	rc := runArgs{kind: entity.KindEpic, root: root}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_BodyFileReadFailure covers the --body-file read guard past
// a successful root/actor/lock/tree-load sequence: a nonexistent
// --body-file path makes cliutil.ReadBodyFile fail.
func TestRun_BodyFileReadFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	missing := filepath.Join(root, "does-not-exist.md")
	rc := runArgs{kind: entity.KindEpic, actor: "human/test", root: root, bodyFile: missing}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_ContractBindValidatorLoadContractsDocFailure covers the
// contract-kind --validator path's cliutil.LoadContractsDoc guard,
// reusing the malformed-contracts-block trigger already proven at
// internal/cli/renamearea/renamearea_test.go and
// internal/cli/setarea/setarea_test.go.
func TestRun_ContractBindValidatorLoadContractsDocFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("contracts:\n  bindings:\n    - not a valid binding\n"), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	rc := runArgs{kind: entity.KindContract, actor: "human/test", root: root, bindValidator: "validator-name"}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_LoadTreeWithTrunkConfigParseFailure covers Run's
// cliutil.LoadTreeWithTrunk guard: a syntactically malformed
// aiwf.yaml makes config.Load return a parse error that
// LoadTreeWithTrunk propagates as-is (config.go's Load only swallows
// config.ErrNotFound — a missing file — not a parse failure). This is
// distinct from tree.Load's per-file LoadError case (a malformed
// *entity* file, which WriteMalformedEntity covers): a malformed
// aiwf.yaml is a fatal load error, not a findings-shaped one.
func TestRun_LoadTreeWithTrunkConfigParseFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"),
		[]byte("areas:\n  members: [unclosed\n"), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}
	rc := runArgs{kind: entity.KindEpic, actor: "human/test", root: root}.run()
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}

// TestNewCmd_AC_NoTitle covers runAC's --title-required guard: no
// --title at all is a usage error, checked before any root/tree work.
func TestNewCmd_AC_NoTitle(t *testing.T) {
	t.Parallel()
	if rc := execExitCode(t, add.NewCmd(""), []string{"ac", "M-0001"}); rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestNewCmd_AC_BodyFileCountMismatch covers runAC's M-067/AC-3 guard:
// when any --body-file is given, its count must match --title's count
// (positional pairing).
func TestNewCmd_AC_BodyFileCountMismatch(t *testing.T) {
	t.Parallel()
	rc := execExitCode(t, add.NewCmd(""), []string{
		"ac", "M-0001",
		"--title", "T1",
		"--body-file", "f1.md", "--body-file", "f2.md",
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestNewCmd_AC_StdinBodyFileWithMultipleTitles covers runAC's
// M-067/AC-5 guard: `--body-file -` (stdin) is only valid with a
// single --title, since stdin is one stream that can't be split
// positionally across a multi-AC batch.
func TestNewCmd_AC_StdinBodyFileWithMultipleTitles(t *testing.T) {
	t.Parallel()
	rc := execExitCode(t, add.NewCmd(""), []string{
		"ac", "M-0001",
		"--title", "T1", "--title", "T2",
		"--body-file", "real1.md", "--body-file", "-",
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestNewCmd_AC_BodyFileReadFailure covers runAC's per-file
// cliutil.ReadBodyFile guard: a nonexistent --body-file path.
func TestNewCmd_AC_BodyFileReadFailure(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "does-not-exist.md")
	rc := execExitCode(t, add.NewCmd(""), []string{
		"ac", "M-0001", "--title", "T1", "--body-file", missing,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestNewCmd_AC_BodyFileFrontmatterRejected covers runAC's M-067/AC-4
// guard: a --body-file whose content begins with a `---` frontmatter
// delimiter is refused (the AC body is appended after a heading the
// verb owns; an embedded frontmatter block would break the document).
func TestNewCmd_AC_BodyFileFrontmatterRejected(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "body-with-frontmatter.md")
	if err := os.WriteFile(path, []byte("---\nleading frontmatter\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	rc := execExitCode(t, add.NewCmd(""), []string{
		"ac", "M-0001", "--title", "T1", "--body-file", path,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestNewCmd_AC_ResolveActorFailure covers runAC's own
// cliutil.ResolveActor guard using M-0252's BrokenGitIdentity fixture.
// Serial: BrokenGitIdentity uses t.Setenv, which panics under
// t.Parallel.
func TestNewCmd_AC_ResolveActorFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)
	root := t.TempDir()
	rc := execExitCode(t, add.NewCmd(""), []string{
		"ac", "M-0001", "--title", "T1", "--root", root,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}
