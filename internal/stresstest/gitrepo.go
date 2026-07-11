package stresstest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gitInitAndConfig git-inits dir (forcing the initial branch name to
// "main" — aiwf's own default trunk-name assumption, per
// cliutil.ConfiguredTrunkBranchShortName's unconfigured default) and
// sets a deterministic commit identity. No `aiwf init` is needed —
// `aiwf add`/`promote` work against a bare git repo with no
// aiwf.yaml. Shared by every scenario in this package whose Setup
// needs a fresh disposable repo.
//
// The explicit --initial-branch matters beyond cosmetics (G-0270):
// plain `git init` defaults to "master" on some git versions/configs,
// which silently defeated promote-on-wrong-branch's trunk-tip
// resolution — any scenario promoting an epic/milestone without also
// forcing this would otherwise fire spurious findings purely because
// the fixture's default branch name doesn't match aiwf's convention,
// not because of any real branch-choreography violation.
func gitInitAndConfig(dir string) error {
	if err := runGit(dir, "init", "-q", "--initial-branch=main"); err != nil { //coverage:ignore defensive: git init on a fresh os.MkdirTemp dir has no realistic failure mode short of filesystem sabotage
		return err
	}
	return configureGitIdentity(dir)
}

// configureGitIdentity sets the deterministic commit identity every
// scenario in this package uses, without re-running `git init` — for
// a directory that already has a .git (e.g. a fresh clone), unlike
// gitInitAndConfig.
func configureGitIdentity(dir string) error {
	for _, args := range [][]string{
		{"config", "user.email", "stresstest@example.com"},
		{"config", "user.name", "stresstest"},
	} {
		if err := runGit(dir, args...); err != nil { //coverage:ignore defensive: git config in a repo this scenario itself just created or cloned has no realistic failure mode short of filesystem sabotage
			return err
		}
	}
	return nil
}

// seedActivationEpic git-inits dir and seeds one proposed epic — the
// entity a subsequent `aiwf promote <epic> active` call targets.
// Shared by HeadDriftScenario (G-0269, the prevention half of the
// "activation commit lands on the wrong branch" incident) and
// PromoteOnWrongBranchDetectionScenario (G-0270, the detection half):
// both drive the identical seeding step before diverging into their
// own Run.
func seedActivationEpic(aiwfBin, dir, title, body string) (string, error) {
	if err := gitInitAndConfig(dir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
		return "", err
	}
	addEnv, err := runAiwfJSON(aiwfBin, dir, "add", "epic", "--title", title, "--body", body)
	if err != nil { //coverage:ignore defensive: covered by the same launch-failure class other scenarios pin at runAiwfJSON's own source
		return "", fmt.Errorf("seeding the epic: %w", err)
	}
	if addEnv.Status != "ok" {
		return "", fmt.Errorf("seeding the epic: aiwf did not report ok (status=%s, error=%+v)", addEnv.Status, addEnv.Error)
	}
	return addEnv.Metadata.EntityID, nil
}

// newBareOriginWithClonesFixture creates a bare origin repo under
// dir/origin.git, seeds it with one empty commit via a throwaway
// clone, then clones it once per name in cloneNames — genuinely
// independent working copies sharing only the origin remote, not each
// other's local refs (contrast newSiblingWorktreesFixture's linked
// worktrees, which share one .git and its full ref namespace). First
// built for M-0243/AC-1's ParallelBranchReallocateScenario.
func newBareOriginWithClonesFixture(dir string, cloneNames ...string) error {
	originDir := filepath.Join(dir, "origin.git")
	if err := runGit(dir, "init", "-q", "--bare", "--initial-branch=main", originDir); err != nil { //coverage:ignore defensive: git init --bare on a fresh path under this scenario's own os.MkdirTemp dir has no realistic failure mode
		return fmt.Errorf("creating bare origin: %w", err)
	}

	seedDir := filepath.Join(dir, "seed")
	if err := runGit(dir, "clone", "-q", originDir, seedDir); err != nil { //coverage:ignore defensive: cloning a freshly-created bare repo has no realistic failure mode
		return fmt.Errorf("cloning origin to seed: %w", err)
	}
	if err := configureGitIdentity(seedDir); err != nil { //coverage:ignore defensive: configureGitIdentity's own internal branch already carries this rationale
		return err
	}
	if err := runGit(seedDir, "commit", "-q", "--allow-empty", "-m", "seed"); err != nil { //coverage:ignore defensive: an empty commit in a freshly-cloned repo has no realistic failure mode
		return err
	}
	if err := runGit(seedDir, "push", "-q", "origin", "HEAD:main"); err != nil { //coverage:ignore defensive: pushing the first commit to a freshly-created empty bare origin has no realistic failure mode
		return err
	}

	for _, name := range cloneNames {
		cloneDir := filepath.Join(dir, name)
		if err := runGit(dir, "clone", "-q", originDir, cloneDir); err != nil { //coverage:ignore defensive: cloning a repo that already has one pushed commit has no realistic failure mode
			return fmt.Errorf("cloning origin to %s: %w", name, err)
		}
		if err := configureGitIdentity(cloneDir); err != nil { //coverage:ignore defensive: see the seed clone above
			return err
		}
	}
	return nil
}

// runGit runs one git subcommand in dir, returning combined output
// wrapped into the error on failure.
func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil { //coverage:ignore defensive: exercised only through call sites whose own git operations (init/config/worktree add/merge) on a scenario-managed disposable repo have no realistic failure mode short of filesystem sabotage
		return fmt.Errorf("git %v: %w\n%s", args, err, out)
	}
	return nil
}

// readGapFile reads the one gap entity file id names under root's
// work/gaps/ directory, tolerating any slug. First built for
// M-0242/AC-2's MidWriteKillScenario, then reused by AC-4's
// DiskFaultScenario — living in this shared fixture-helper file
// rather than staying stranded in the single-AC file that first
// needed it, now that more than one scenario depends on it.
func readGapFile(root, id string) ([]byte, error) {
	matches, err := filepath.Glob(filepath.Join(root, "work", "gaps", id+"-*.md"))
	if err != nil { //coverage:ignore defensive: the only error filepath.Glob returns is ErrBadPattern, and this package's own literal pattern is well-formed by construction
		return nil, fmt.Errorf("globbing for gap %s under %s: %w", id, root, err)
	}
	if len(matches) != 1 {
		return nil, fmt.Errorf("expected exactly one gap file for %s under %s, found %d: %v", id, root, len(matches), matches)
	}
	return os.ReadFile(matches[0])
}

// newSiblingWorktreesFixture creates a main repo with a seed commit
// under dir/main, then adds two sibling worktrees (actor-a, actor-b)
// off it — dir/wt-a, dir/wt-b. Shared by every scenario whose Setup
// needs two independent working copies of one repo (M-0241/AC-3,
// AC-5).
func newSiblingWorktreesFixture(dir string) error {
	mainDir := filepath.Join(dir, "main")
	if err := os.MkdirAll(mainDir, 0o755); err != nil { //coverage:ignore defensive: mainDir is a fresh subdirectory of RunScenario's own os.MkdirTemp result, no realistic failure mode short of filesystem sabotage
		return fmt.Errorf("creating main repo dir: %w", err)
	}
	if err := gitInitAndConfig(mainDir); err != nil { //coverage:ignore defensive: gitInitAndConfig's own internal branch already carries this rationale
		return err
	}
	if err := runGit(mainDir, "commit", "-q", "--allow-empty", "-m", "seed"); err != nil { //coverage:ignore defensive: an empty commit in a freshly-initialized repo has no realistic failure mode
		return err
	}
	if err := runGit(mainDir, "worktree", "add", "-q", "-b", "actor-a", filepath.Join(dir, "wt-a")); err != nil { //coverage:ignore defensive: adding a worktree at a fresh, never-before-used path has no realistic failure mode
		return err
	}
	if err := runGit(mainDir, "worktree", "add", "-q", "-b", "actor-b", filepath.Join(dir, "wt-b")); err != nil { //coverage:ignore defensive: see the actor-a worktree add above
		return err
	}
	return nil
}

// currentBranch returns dir's currently checked-out branch name.
// First built for M-0243/AC-3's ArchiveDuringActiveScopeScenario, then
// reused by AC-5's HeadDriftScenario — living in this shared
// git-primitive file rather than staying stranded in the single-AC
// file that first needed it, now that more than one scenario depends
// on it.
func currentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil { //coverage:ignore defensive: reading HEAD's branch name right after this scenario's own git init has no realistic failure mode
		return "", fmt.Errorf("reading current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// headSHA returns dir's current HEAD commit SHA. First built for
// M-0243/AC-4's ForceOverrideDurabilityScenario, then reused by AC-5's
// HeadDriftScenario — living in this shared git-primitive file rather
// than staying stranded in the single-AC file that first needed it,
// now that more than one scenario depends on it.
func headSHA(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil { //coverage:ignore defensive: reading HEAD in a repo this scenario itself just committed to has no realistic failure mode
		return "", fmt.Errorf("reading HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
