package check

import (
	"fmt"
	"slices"
	"strings"

	codespkg "github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/gitops"
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
// addressable (AC-10).
//
// Algorithm (per commit, in chronological order):
//
//  1. Skip if the commit is not an AI commit on an entity (must
//     carry both aiwf-actor: ai/... and aiwf-entity: <id>).
//  2. Find the most recent opened-scope commit on the same entity
//     in the preceding commits. If none, skip (AC-9 — no scope,
//     no policing). Cycle 3 will further filter on the scope's
//     current state (paused → silent per AC-5).
//  3. Read the scope's aiwf-branch: trailer. If absent — legacy
//     pre-M-0102 scope — skip (non-retroactive per epic
//     §"Out of scope").
//  4. Ask the oracle for the commit's branch set. If empty —
//     "unknown branch" — skip (do not fire on commits the kernel
//     cannot confidently classify).
//  5. If the bound branch is in the commit's branch set, silent
//     (AC-4 — commit rides bound branch).
//  6. Otherwise fire isolation-escape with the commit's SHA, the
//     entity id, the bound branch, and the actual branch list as
//     evidence.
func RunIsolationEscape(commits []scope.Commit, oracle BranchOracle) []Finding {
	if oracle == nil {
		return nil
	}

	// Per-entity index of opened-scope commits, oldest-first. Built
	// in one pass so per-commit lookup is O(scopes-on-entity), not
	// O(commits). The Cycle 3 extension will track scope-end events
	// (paused/ended) on the same shape.
	type openerRecord struct {
		chronoIdx int
		branch    string
	}
	openersByEntity := map[string][]openerRecord{}

	for i := range commits {
		c := &commits[i]
		idx := indexCommitTrailersForProvenance(c.Trailers)
		if idx[gitops.TrailerVerb] != "authorize" || idx[gitops.TrailerScope] != "opened" {
			continue
		}
		entity := idx[gitops.TrailerEntity]
		if entity == "" {
			continue
		}
		branch := idx[gitops.TrailerBranch]
		openersByEntity[entity] = append(openersByEntity[entity], openerRecord{
			chronoIdx: i,
			branch:    branch,
		})
	}

	var findings []Finding
	for i := range commits {
		c := &commits[i]
		idx := indexCommitTrailersForProvenance(c.Trailers)

		actor := idx[gitops.TrailerActor]
		entity := idx[gitops.TrailerEntity]
		if !strings.HasPrefix(actor, "ai/") || entity == "" {
			continue
		}
		// Don't police the scope-opening / pausing / resuming commits
		// themselves — those land on the parent ritual branch by ritual
		// design and the rule's algorithm would mis-classify them. Only
		// post-scope-open work commits are in scope.
		if idx[gitops.TrailerVerb] == "authorize" {
			continue
		}

		openers := openersByEntity[entity]
		if len(openers) == 0 {
			continue // AC-9 — no scope opened on this entity, no policing.
		}

		// Find the most recent opener that precedes (or equals) this
		// commit chronologically. Walk back from the newest.
		var bound string
		var found bool
		for j := len(openers) - 1; j >= 0; j-- {
			if openers[j].chronoIdx <= i {
				bound = openers[j].branch
				found = true
				break
			}
		}
		if !found {
			continue // commit predates every opener — no scope to escape from.
		}
		if bound == "" {
			continue // pre-M-0102 scope without aiwf-branch: trailer; non-retroactive.
		}

		actualBranches := oracle.FirstParentBranches(c.SHA)
		if len(actualBranches) == 0 {
			continue // unknown branch — do not fire on a commit the oracle can't classify.
		}
		if slices.Contains(actualBranches, bound) {
			continue // AC-4 — commit rides the bound branch.
		}

		// AC-1 / AC-2 / AC-3 — commit landed on a branch other than
		// the bound one. Fire one finding per violating commit (AC-10).
		findings = append(findings, Finding{
			Code:     CodeIsolationEscape.ID,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"commit %s: aiwf-actor: %q on %q escapes the active scope's bound branch %q",
				short(c.SHA), actor, strings.Join(actualBranches, ","), bound,
			),
			EntityID: entity,
		})
	}
	return findings
}
