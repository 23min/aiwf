package check

import (
	codespkg "github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/scope"
)

// CodeIsolationEscape is the typed kernel-code descriptor for the
// isolation-escape finding (M-0106, closes G-0099). The finding
// fires when an AI-actor's commit lands on a branch that doesn't
// match its active scope's recorded aiwf-branch: trailer — i.e.,
// the commit "escaped" its assigned ritual branch.
//
// The code carries [codes.ClassBranchChoreography] — the layer-4
// kernel carve-out per ADR-0011 — so the branch-policing finding
// set is enumerable independently of structural / legality codes.
//
// Severity is warning at first land (per M-0106 spec); a future
// decision (recorded as a D-NNN) may tighten to error after one
// epic of usage. This milestone does not pre-commit the tightening
// timing.
//
// The finding is the post-hoc complement of M-0103's verb-time
// preflight: the preflight refuses bad-dispatch up front; the
// finding catches drift after dispatch (subagent escaped via
// `git checkout main`, `cd ..`, `git -C <other-path>`, or a manual
// cherry-pick that violates the scope-branch coupling). Together
// the two surfaces give defense in depth against G-0099's
// "commits ended up on the wrong branch" failure mode.
var CodeIsolationEscape = codespkg.Code{ID: "isolation-escape", Class: codespkg.ClassBranchChoreography}

// BranchOracle answers per-commit branch-reachability questions the
// isolation-escape rule needs but that scope.Commit does not carry.
// Implementations are supplied by the CLI gather layer (which has
// the git context); the check rule itself stays pure.
//
// FirstParentBranches returns the set of ritual-shape branches the
// commit is reachable from along first-parent paths. The set MAY
// include "main" when a commit landed directly on the trunk. An
// empty/nil return means the commit is not on any branch the oracle
// knows about (treat as "unknown" — the rule does not fire on
// unknown-branch commits, since the kernel cannot confidently
// classify them as escaped).
type BranchOracle interface {
	FirstParentBranches(sha string) []string
}

// RunIsolationEscape applies the M-0106 branch-choreography rule
// against a commit history. The rule scans commits for those
// carrying both aiwf-actor: ai/... and aiwf-entity: <id> trailers,
// finds each candidate's active scope at the commit's time, and
// fires isolation-escape when the commit's branch does not match
// the scope's recorded aiwf-branch:.
//
// commits must be ordered oldest-first (matching the RunProvenance
// convention). oracle supplies per-commit branch info; a nil oracle
// is treated as "no branch info available" and the rule returns
// silently — a graceful degradation for environments where the
// gather layer cannot determine commit branches (e.g., a bare repo
// fragment in a test fixture without ref history).
//
// Per-commit firing: each violating commit produces its own
// finding. No aggregation, no per-entity summary — the user wants
// the cardinality so each escaped commit is individually
// addressable. AC-10.
//
// This is the Cycle 1 scaffold. The detection logic lands in
// Cycle 2; for now the function returns nil so the rule wires
// through Run() without raising findings.
func RunIsolationEscape(commits []scope.Commit, oracle BranchOracle) []Finding {
	_ = commits
	_ = oracle
	return nil
}
