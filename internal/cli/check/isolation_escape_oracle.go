package check

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/23min/aiwf/internal/branchparse"
	"github.com/23min/aiwf/internal/check"
)

// gitBranchOracle is the production [check.BranchOracle]
// implementation used by RunProvenanceCheck. It computes per-commit
// first-parent reachability across the set of ritual-shape branches
// (plus `main`) by running `git rev-list --first-parent <branch>`
// once per branch and indexing the result.
//
// The set is built eagerly at construction (one fork per branch),
// then served from memory by FirstParentBranches. For a repo with
// N ritual branches and M commits behind them, that's N+1 invocations
// of git and ~O(M) memory — proportional to the same data
// RunProvenanceCheck already pulls down for `git log`. The
// alternative — per-call `git merge-base --is-ancestor` — would be
// O(violations × N) forks at check time, slower for the common
// case where the check is silent.
//
// "Branches the oracle knows about" is intentionally narrow: only
// `main` plus refs whose name parses as a ritual shape per
// [branchparse.ParseEntityFromBranch]. A commit reachable from
// `feature/foo` only would return an empty slice from
// FirstParentBranches — which the rule treats as "unknown branch,
// silent." That's the documented degradation per the
// [check.BranchOracle] doc.
//
// Closes M-0106/F-1 (oracle CLI gather-side implementation).
type gitBranchOracle struct {
	// branchesBySHA maps a commit SHA to the list of ritual-shape
	// (or main) local branches that reach it via first-parent. nil
	// for SHAs not seen during construction.
	branchesBySHA map[string][]string
}

// newGitBranchOracle reads the local-branch set via
// `git for-each-ref refs/heads/`, filters to main + ritual shapes,
// and indexes per-branch first-parent reachability via `git
// rev-list --first-parent <branch>` per branch. Returns nil and the
// error if any git invocation fails; the caller (RunProvenanceCheck)
// can decide whether to skip the rule or surface the error.
//
// Empty-repo guard: a repo with zero local branches (e.g. a fresh
// init with no commits) returns a non-nil oracle whose
// FirstParentBranches always returns nil. This matches the rule's
// "unknown branch, silent" contract — no false positives on the
// startup edge.
func newGitBranchOracle(ctx context.Context, root string) (*gitBranchOracle, error) {
	branches, err := listRitualBranches(ctx, root)
	if err != nil {
		return nil, err
	}
	idx := map[string][]string{}
	for _, b := range branches {
		shas, err := firstParentSHAs(ctx, root, b)
		if err != nil {
			return nil, fmt.Errorf("indexing first-parent of %q: %w", b, err)
		}
		for _, sha := range shas {
			idx[sha] = append(idx[sha], b)
		}
	}
	return &gitBranchOracle{branchesBySHA: idx}, nil
}

// FirstParentBranches implements [check.BranchOracle].
func (o *gitBranchOracle) FirstParentBranches(sha string) []string {
	return o.branchesBySHA[sha]
}

// Compile-time check: gitBranchOracle satisfies check.BranchOracle.
var _ check.BranchOracle = (*gitBranchOracle)(nil)

// listRitualBranches returns the local-branch set filtered to main
// + ritual shapes (epic/E-NNNN-..., milestone/M-NNNN-..., patch/...).
// Other-shape branches (feature/foo, chore/bar, the integration
// branches the kernel doesn't recognize) are excluded — commits on
// them are intentionally "unknown" to the rule.
func listRitualBranches(ctx context.Context, root string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "for-each-ref", "refs/heads/", "--format=%(refname:short)")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git for-each-ref: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git for-each-ref: %w", err)
	}
	var ritual []string
	for _, line := range strings.Split(string(out), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if name == "main" || branchparse.ParseEntityFromBranch(name) != "" {
			ritual = append(ritual, name)
		}
	}
	return ritual, nil
}

// firstParentSHAs returns the SHAs along the first-parent path from
// the tip of branch back to the root. A non-existent branch (e.g.
// stale ref) returns a git exit error; the caller decides whether
// to surface or skip.
func firstParentSHAs(ctx context.Context, root, branch string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--first-parent", branch)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git rev-list --first-parent %s: %w\n%s", branch, err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git rev-list --first-parent %s: %w", branch, err)
	}
	var shas []string
	for _, line := range strings.Split(string(out), "\n") {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		shas = append(shas, s)
	}
	return shas, nil
}
