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
	// errs accumulates per-ref failures from oracle construction
	// (M-0161/AC-3 / G-0203 / D-0019). Empty slice ↔ every
	// enumerated ref's first-parent index built cleanly. Non-empty
	// slice surfaces as one isolation-escape-oracle-failure
	// advisory per entry via RunProvenanceCheck.
	errs []check.OracleErr
}

// newGitBranchOracle reads the local-branch set via
// `git for-each-ref refs/heads/`, filters to main + ritual shapes,
// and indexes per-branch first-parent reachability via `git
// rev-list --first-parent <branch>` per branch.
//
// Per-ref fault tolerance (M-0161/AC-3 / D-0019):
//   - Whole-enumeration failures (corrupted packed-refs, permission
//     errors on .git/refs) abort construction and return (nil, err)
//     — there are no refs to name and the rule degrades to the
//     existing silent-skip path at provenance.go.
//   - Per-ref walk failures (`rev-list --first-parent <ref>`
//     errors on one ref while others succeed) accumulate into
//     `errs` and DO NOT abort construction. Healthy refs continue
//     to populate branchesBySHA so the [check.RunIsolationEscape]
//     rule runs against them; the failed refs surface as
//     [check.OracleErr] entries via [OracleErrors] for the
//     provenance layer to render as isolation-escape-oracle-
//     failure advisory findings.
//
// Shallow-clone fail-shut (M-0161/AC-4 / G-0204):
//   - When `git rev-parse --is-shallow-repository` reports true,
//     the oracle leaves branchesBySHA EMPTY (no half-walked index)
//     and accumulates a single typed OracleErr with Capability
//     "shallow-clone". The isolation-escape rule sees no branch
//     data and stays silent for every commit (fail-shut on
//     correctness); RunProvenanceCheck emits the new
//     isolation-escape-shallow-clone warning so the coverage gap
//     is mechanically visible to the operator.
//
// Empty-repo guard: a repo with zero local branches (e.g. a fresh
// init with no commits) returns a non-nil oracle whose
// FirstParentBranches always returns nil and OracleErrors is
// empty. This matches the rule's "unknown branch, silent"
// contract — no false positives on the startup edge.
func newGitBranchOracle(ctx context.Context, root string) (*gitBranchOracle, error) {
	// M-0161/AC-4 — shallow detection first. A shallow repo
	// short-circuits the index build because any partial index
	// would produce silent false-negatives for commits beyond
	// the shallow boundary.
	if shallow, sErr := isShallowRepository(ctx, root); sErr != nil {
		// Treat detection failure as not-shallow — the existing
		// per-ref walk continues. The detection command is a
		// trivial git plumbing call; a failure here typically
		// means a deeper repo problem the per-ref tolerance will
		// surface anyway.
		_ = sErr
	} else if shallow {
		return &gitBranchOracle{
			branchesBySHA: map[string][]string{},
			errs: []check.OracleErr{{
				Capability: "shallow-clone",
				Err:        errors.New("repository is shallow per `git rev-parse --is-shallow-repository`; unshallow with `git fetch --unshallow` (or in CI: `actions/checkout@vN` with `fetch-depth: 0`) to restore isolation-escape coverage"),
			}},
		}, nil
	}

	branches, err := listRitualBranches(ctx, root)
	if err != nil {
		return nil, err
	}
	idx := map[string][]string{}
	var perRefErrs []check.OracleErr
	for _, b := range branches {
		shas, err := firstParentSHAs(ctx, root, b)
		if err != nil {
			perRefErrs = append(perRefErrs, check.OracleErr{
				Ref:        b,
				Capability: "ref-resolution-failed",
				Err:        err,
			})
			continue
		}
		for _, sha := range shas {
			idx[sha] = append(idx[sha], b)
		}
	}
	return &gitBranchOracle{branchesBySHA: idx, errs: perRefErrs}, nil
}

// isShallowRepository runs `git rev-parse --is-shallow-repository`
// in root and reports whether the repository is shallow. Git
// emits "true"/"false" on stdout for this plumbing query.
//
// AC-4 fixture-aware: any depth-N clone (or a manually-written
// .git/shallow file) flips this to true. The detection is
// repo-level, not per-ref, so a single shallow boundary disables
// the whole rule per the fail-shut contract.
func isShallowRepository(ctx context.Context, root string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-shallow-repository")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return false, fmt.Errorf("git rev-parse --is-shallow-repository: %w\n%s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return false, fmt.Errorf("git rev-parse --is-shallow-repository: %w", err)
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

// FirstParentBranches implements [check.BranchOracle].
func (o *gitBranchOracle) FirstParentBranches(sha string) []string {
	return o.branchesBySHA[sha]
}

// OracleErrors implements [check.BranchOracle].
func (o *gitBranchOracle) OracleErrors() []check.OracleErr {
	return o.errs
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
