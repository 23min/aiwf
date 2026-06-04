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
	// distanceFromTip maps (branch, SHA) → position in branch's
	// first-parent chain (0 = tip, growing toward root). Used by
	// BranchOfSHA (M-0161/AC-6) to disambiguate the "SHA appears
	// on multiple branches" case: the branch where the SHA is
	// CLOSEST to the tip is the canonical owner. This is the
	// rename-invariant: a renamed branch's tip is the same as
	// (or a descendant of) the pre-rename tip; sibling branches
	// cut from the same trunk ancestor have the SHA deeper in
	// their first-parent chain.
	distanceFromTip map[string]map[string]int
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

	// M-0161/AC-5 — reflog-availability detection. When
	// core.logAllRefUpdates is false, ref updates are not
	// recorded and WalkOrphanedAICommits cannot find anything.
	// Surface this via AC-3's typed-error contract with
	// Capability "reflog-disabled" so RunProvenanceCheck emits
	// the isolation-escape-oracle-failure advisory naming the
	// remediation.
	var reflogDisabledErr []check.OracleErr
	if disabled, rErr := isReflogDisabled(ctx, root); rErr == nil && disabled {
		reflogDisabledErr = append(reflogDisabledErr, check.OracleErr{
			Capability: "reflog-disabled",
			Err:        errors.New("core.logAllRefUpdates=false; the reflog is not recorded so force-push orphan detection cannot run; set core.logAllRefUpdates=true (or remove the setting) to restore isolation-escape-orphaned-ai-commit coverage"),
		})
	}

	branches, err := listRitualBranches(ctx, root)
	if err != nil {
		return nil, err
	}
	idx := map[string][]string{}
	dist := map[string]map[string]int{}
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
		// Build per-branch distance-from-tip index. shas[0] is the
		// tip (distance 0); shas[1] is the parent (distance 1);
		// etc. The distance lets BranchOfSHA prefer the branch
		// where the recorded SHA is closest to the tip (M-0161/
		// AC-6 rename-invariant).
		dist[b] = map[string]int{}
		for i, sha := range shas {
			idx[sha] = append(idx[sha], b)
			if _, already := dist[b][sha]; !already {
				dist[b][sha] = i
			}
		}
	}
	// Concatenate reflog-disabled (whole-repo) and per-ref
	// errors so the consumer sees both classes through the
	// single OracleErrors() slice (D-0019 fail-shut /
	// fail-open contract).
	reflogDisabledErr = append(reflogDisabledErr, perRefErrs...)
	return &gitBranchOracle{branchesBySHA: idx, distanceFromTip: dist, errs: reflogDisabledErr}, nil
}

// isReflogDisabled returns whether `core.logAllRefUpdates` is
// configured to false. The git default is true for non-bare
// repos. When the config is absent the value reads as empty;
// we treat anything other than "false" as enabled.
func isReflogDisabled(ctx context.Context, root string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "config", "--get", "core.logAllRefUpdates")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		// Exit 1 means the key is not set; default (true) applies.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	val := strings.TrimSpace(strings.ToLower(string(out)))
	return val == "false", nil
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

// BranchOfSHA implements [check.BranchOracle] (M-0161/AC-6) —
// returns the ritual branch where sha is CLOSEST TO THE TIP in
// the branch's first-parent chain. Empty return ↔ SHA is
// unknown to the oracle (orphaned, on a non-ritual branch, or
// repo-shallow-truncated past the SHA).
//
// The rename-invariant: a `git branch -m foo bar` rename
// preserves the branch's tip SHA — so the recorded SHA is at
// position 0 (tip) on the renamed-to branch. Sibling branches
// cut from a shared ancestor (or main itself) have the SHA at
// some deeper position. Preferring the smallest distance gives
// us the genuine "this is the same branch, just renamed"
// identification.
//
// Ties (two branches with the same minimum distance) are
// resolved by preferring the first ritual-shape entry over
// trunk, then by alphabetical order — same deterministic
// behavior the oracle's first-parent index already exhibits.
func (o *gitBranchOracle) BranchOfSHA(sha string) string {
	branches := o.branchesBySHA[sha]
	if len(branches) == 0 {
		return ""
	}
	// First pass: filter to ritual-shape candidates. The bound
	// branch is by definition a ritual branch (per ADR-0010 +
	// the M-0102 authorize verb's branch-shape requirement);
	// trunk (main) is excluded so a SHA shared between trunk
	// and a newly-cut ritual sibling resolves to the ritual
	// owner. If NO ritual candidate exists (legacy / off-shape
	// fixture), fall back to the full candidate set.
	var candidates []string
	for _, b := range branches {
		if b != "main" {
			candidates = append(candidates, b)
		}
	}
	if len(candidates) == 0 {
		candidates = branches
	}
	// Second pass: pick the candidate where the recorded SHA
	// is closest to the tip (smallest distance). The rename
	// invariant: a renamed branch's tip preserves the SHA
	// (distance 0) or advances FROM it (distance grows with
	// new commits). Sibling branches cut from a shared
	// ancestor have the SHA at the shared-ancestor distance
	// — typically deeper than the renamed branch's distance
	// to a recently-recorded SHA.
	bestBranch := ""
	bestDist := -1
	for _, b := range candidates {
		d, ok := o.distanceFromTip[b][sha]
		if !ok {
			continue
		}
		if bestBranch == "" || d < bestDist || (d == bestDist && b < bestBranch) {
			bestBranch = b
			bestDist = d
		}
	}
	return bestBranch
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
