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

// CodeIsolationEscapeOracleFailure is the advisory finding code
// surfacing oracle partial-coverage states (M-0161/AC-3 / G-0203 /
// D-0019). One finding fires per per-ref failure accumulated
// during [BranchOracle] construction, naming the ref and the
// underlying failure mode (ref-resolution-failed today;
// shallow-clone and reflog-disabled added by AC-4 and AC-5).
//
// Severity is warning at first land per the M-0125 ratchet
// pattern: surface advisory at first, tighten to error after one
// epic of usage if the false-positive rate stays low. The
// contract is fail-shut on rule correctness — the
// [isolation-escape] rule does NOT fire on commits whose branch
// resolution was lost to a failed ref, so the advisory exists
// for operator visibility, not as a blocker. See D-0019 for the
// fail-shut-on-correctness / fail-open-on-coverage contract.
var CodeIsolationEscapeOracleFailure = codespkg.Code{ID: "isolation-escape-oracle-failure", Class: codespkg.ClassBranchChoreography}

// CodeIsolationEscapeShallowClone is the warning finding code
// surfacing shallow-clone-induced total-coverage failure
// (M-0161/AC-4 / G-0204). One finding fires when the repository
// is shallow per `git rev-parse --is-shallow-repository`, naming
// the remediation directly: unshallow with `git fetch
// --unshallow` (or in CI, `actions/checkout@vN` with
// `fetch-depth: 0`).
//
// Severity is warning, NOT advisory — a shallow clone is a
// total-coverage failure (not a per-ref partial failure as
// AC-3's isolation-escape-oracle-failure tracks), so the
// operator-visibility weight is higher. The deliberate exception
// to D-0019 Alternative D's "ride the typed slice" rule per the
// AC-4 body line 292.
//
// Per the AC-4 fail-shut-on-correctness contract: on shallow,
// the per-SHA branch map is left EMPTY (no false positives from
// half-walked first-parent indexes), so the isolation-escape
// rule stays silent regardless of what the shallow window would
// otherwise expose. The new code's job is to make that coverage
// gap mechanically visible.
var CodeIsolationEscapeShallowClone = codespkg.Code{ID: "isolation-escape-shallow-clone", Class: codespkg.ClassBranchChoreography}

// CodeIsolationEscapeOrphanedAICommit is the warning finding
// code surfacing AI-actor commits orphaned by non-fast-forward
// updates ("force-push") on ritual branches (M-0161/AC-5 /
// G-0205). One finding fires per orphaned AI commit detected
// via reflog walk; the kernel cannot determine from the orphan
// alone whether it was on the correct branch at force-time
// (the rewrite removed the audit trail), so the rule surfaces
// the orphan for operator review with hint text pointing at
// `aiwf acknowledge illegal` for deliberate sovereign cleanup.
//
// Severity is warning per the M-0125 ratchet pattern; future
// D-NNN may tighten to error after one full epic of usage.
//
// Composes with AC-3: when core.logAllRefUpdates=false the
// reflog is absent and detection cannot run — that case rides
// AC-3's isolation-escape-oracle-failure advisory with
// Capability "reflog-disabled" rather than introducing a
// separate finding code, per the D-0019 fail-shut-on-correctness
// /fail-open-on-coverage contract.
var CodeIsolationEscapeOrphanedAICommit = codespkg.Code{ID: "isolation-escape-orphaned-ai-commit", Class: codespkg.ClassBranchChoreography}

// OracleErr is a per-ref or repo-wide oracle-construction failure
// surfaced by [BranchOracle.OracleErrors]. The Capability tag
// names what coverage was lost; the underlying Err is preserved
// for diagnostic surface in the
// isolation-escape-oracle-failure advisory's hint text.
//
// Ref is the ritual ref the failure pertains to (e.g.
// "epic/E-0001-engine"). Ref MAY be empty for repo-wide failures
// (corrupted packed-refs at enumeration time); per D-0019 point
// 4, that case is handled at the construction-error path rather
// than via OracleErrors, but the field shape leaves room for a
// future repo-wide kind.
//
// Capability classes (M-0161/AC-3..AC-5):
//
//   - "ref-resolution-failed" (AC-3) — `git rev-list
//     --first-parent <ref>` failed on a single ritual ref while
//     `for-each-ref` enumerated cleanly. The ref's first-parent
//     index could not be built.
//   - "shallow-clone" (AC-4) — the ref's first-parent walk hit
//     the shallow-depth horizon. Reserved.
//   - "reflog-disabled" (AC-5) — `core.logAllRefUpdates=false`
//     detected at gather time; AC-5's reflog-walk extension
//     cannot fire on the affected ref. Reserved.
type OracleErr struct {
	Ref        string
	Capability string
	Err        error
}

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
//
// OracleErrors returns per-ref construction failures accumulated
// at gather time (M-0161/AC-3 / G-0203). Empty slice ↔ every
// enumerated ref's first-parent index built cleanly; non-empty
// slice ↔ at least one ref failed and the rule's coverage is
// incomplete for those refs. Consumers (RunProvenanceCheck)
// surface one [CodeIsolationEscapeOracleFailure] finding per
// entry, naming the ref and the underlying error in the hint —
// see D-0019 for the fail-shut-on-correctness /
// fail-open-on-coverage contract that orders OracleErrors with
// FirstParentBranches.
type BranchOracle interface {
	FirstParentBranches(sha string) []string
	OracleErrors() []OracleErr
	// BranchOfSHA returns the current ritual-shape branch (or
	// "main") whose first-parent index contains sha. An empty
	// return means the SHA is not on any branch the oracle
	// knows about — either the branch was deleted, the SHA is
	// orphaned, or the SHA was never on a ritual ref.
	//
	// Used by M-0161/AC-6's rename-survival path:
	// `aiwf-branch-sha:` trailer values from authorize commits
	// resolve to whatever branch currently reaches that SHA,
	// regardless of how the branch has been renamed. A returned
	// empty string composes with AC-3's fail-shut-on-correctness
	// contract — the rule does not fire (no false positive when
	// binding is lost); the partial-coverage state surfaces via
	// the existing isolation-escape-oracle-failure advisory
	// when configured.
	//
	// Resolution is deterministic but order-dependent on the
	// underlying branchesBySHA index: when the SHA appears on
	// multiple branches, the first-listed entry wins. The
	// production gitBranchOracle iterates ritual refs in
	// for-each-ref order; tests using fakeOracle should not
	// rely on a particular branch being first when the SHA
	// has multiple owners (use single-owner fixtures or
	// extend the fake).
	BranchOfSHA(sha string) string
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
// cherryPicked is the set of commit SHAs the gather layer identified
// as `git cherry-pick -x` re-authors of upstream commits: both
// (a) the commit body carries the `(cherry picked from commit <sha>)`
// marker line that `git cherry-pick -x` writes by default AND
// (b) the commit's git committer email differs from its git author
// email (the identity gap that `git cherry-pick` produces by default
// when a different identity re-authors the original — committer
// becomes the current user, author preserved from source). When a
// commit's SHA is in this set the rule treats it as a sovereign
// human re-author (corner case 8 / AC-6) and suppresses any
// isolation-escape finding against it; the audit trail lives in
// the committer-vs-author identity gap and the marker itself.
// A nil/empty map means "no cherry-pick info available"; the rule
// then polices as usual (no false negatives — only known-cherry-picks
// are suppressed).
//
// The both-signals contract is the gather-side's per-commit
// derivation, implemented by check.WalkCherryPicks in
// internal/check/cherry_picks.go (M-0159/AC-6 / G-0202). Either
// signal alone is insufficient: a fabricated marker (no real
// cherry-pick) lacks the gap; an amended commit (gap without -x)
// lacks the marker. Real-git E2E coverage lives in
// internal/cli/integration/branch_scenarios_ac6_test.go.
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
//
// M-0159/AC-3: ackedSHAs carries the set of commit SHAs that have
// been retroactively acknowledged via `aiwf acknowledge illegal`.
// The CLI gather layer computes the map once per check invocation
// (via WalkAcknowledgedSHAs in acks.go) and passes it to all
// three rules that consume it; rule-internal recompute is
// forbidden. A commit whose SHA appears in ackedSHAs is exempted
// from this rule's finding even when it would otherwise fire
// (same per-SHA closed-set scoping the M-0136/AC-2
// illegal-transition exemption uses). Nil or empty map means "no
// acknowledgments" and the rule polices as usual.
func RunIsolationEscape(commits []scope.Commit, oracle BranchOracle, cherryPicked, ackedSHAs map[string]bool) []Finding {
	if oracle == nil {
		return nil
	}

	// Per-entity index of opened-scope commits, oldest-first. Built
	// in one pass so per-commit lookup is O(scopes-on-entity), not
	// O(commits). The endedAt slot tracks the chrono position of
	// the first aiwf-scope-ends: <opener-sha> trailer for the
	// opener; -1 sentinel means "still open through the inspected
	// window." Per the spec line 86: a scope is "active at C's
	// time" only if its opener precedes C AND its end (if any)
	// follows C. Without this tracking the rule false-positives on
	// AI commits made after the scope-entity reached terminal
	// status (F-3 from the M-0106 retrospective review).
	type openerRecord struct {
		chronoIdx int
		endedAt   int // chrono position of aiwf-scope-ends; -1 = never ended
		branch    string
		branchSHA string // M-0161/AC-6: aiwf-branch-sha trailer value (40-char hex); empty for legacy authorize commits without the trailer
	}

	// First sub-pass: index `aiwf-scope-ends: <opener-sha>` trailers
	// keyed on opener SHA. The "first" termination wins — a sequence
	// of scope-ends on the same opener is unusual but the kernel
	// treats the earliest as the binding-loss event. Same pattern as
	// provenance.go's buildEndedAtIndex.
	endsByOpenerSHA := map[string]int{}
	for i := range commits {
		for _, tr := range commits[i].Trailers {
			if tr.Key != gitops.TrailerScopeEnds {
				continue
			}
			if _, already := endsByOpenerSHA[tr.Value]; already {
				continue
			}
			endsByOpenerSHA[tr.Value] = i
		}
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
		branchSHA := idx[gitops.TrailerBranchSHA]
		endedAt := -1
		if pos, ok := endsByOpenerSHA[c.SHA]; ok {
			endedAt = pos
		}
		openersByEntity[entity] = append(openersByEntity[entity], openerRecord{
			chronoIdx: i,
			endedAt:   endedAt,
			branch:    branch,
			branchSHA: branchSHA,
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

		// Find the most recent opener that precedes this commit AND
		// whose scope is still active at this commit's time. Per
		// spec line 86 "active at C's time" = opened before C AND
		// (never ended OR ended after C). When the most-recent-
		// preceding opener has already ended, the binding is gone
		// → silent (F-3 — no false positives on commits after
		// scope-entity reaches terminal status).
		var bound string
		var boundSHA string
		var found bool
		for j := len(openers) - 1; j >= 0; j-- {
			rec := openers[j]
			if rec.chronoIdx > i {
				continue // opener follows this commit; not yet in scope.
			}
			if rec.endedAt >= 0 && rec.endedAt <= i {
				break
			}
			bound = rec.branch
			boundSHA = rec.branchSHA
			found = true
			break
		}
		if !found {
			continue // commit predates every opener, or the most-recent-preceding scope ended before C.
		}
		if bound == "" {
			continue // pre-M-0102 scope without aiwf-branch: trailer; non-retroactive.
		}

		// M-0161/AC-6: SHA-first resolution. When the opener
		// recorded aiwf-branch-sha, resolve to whatever branch
		// currently reaches that SHA — surviving renames
		// transparently. Empty resolution + no SHA-trailer
		// available falls through to the legacy name-only path
		// (backwards-compatible for pre-AC-6 authorize commits;
		// G-0225 documents the legacy carve-out).
		resolved := bound
		if boundSHA != "" {
			if shaResolved := oracle.BranchOfSHA(boundSHA); shaResolved != "" {
				resolved = shaResolved
			}
			// If the SHA doesn't resolve (branch deleted, SHA
			// orphaned by reflog GC, repo shallow), keep `resolved`
			// at the name-based bound. The actualBranches check
			// below stays fail-shut: if the AI commit's branch
			// list doesn't include `resolved`, we'd fire — but the
			// caller (RunProvenanceCheck) is also surfacing an
			// AC-3 oracle-failure advisory naming the unreachable
			// binding, so the operator has both signals. The
			// fail-shut-on-correctness for this case is the
			// existing "actualBranches empty → silent" check
			// below; deletion of the SHA-bound branch typically
			// also drops the AI commits from any first-parent
			// index, leaving actualBranches empty.
		}

		actualBranches := oracle.FirstParentBranches(c.SHA)
		if len(actualBranches) == 0 {
			continue // unknown branch — do not fire on a commit the oracle can't classify.
		}
		if slices.Contains(actualBranches, resolved) {
			continue // AC-4 — commit rides the bound branch (post-SHA-resolution).
		}

		// M-0159/AC-3 — retroactive acknowledgment. When an operator
		// runs `aiwf acknowledge illegal <escape-sha> --reason "..."`
		// after the escape has landed, the new commit carries an
		// aiwf-force-for: <escape-sha> trailer. The CLI gather layer
		// pre-computed the set of acknowledged SHAs (see
		// WalkAcknowledgedSHAs in acks.go) and passes the map here;
		// commits in the set are exempted with the same closed-set
		// per-SHA scoping the M-0136/AC-2 illegal-transition
		// exemption uses. A nil/empty map means "no acknowledgments"
		// and the rule polices as usual.
		if ackedSHAs[c.SHA] {
			continue
		}

		// AC-6 — sovereign cherry-pick re-author. When a human runs
		// `git cherry-pick -x <ai-sha>` to land the AI's commit on a
		// different branch, the resulting commit carries the original
		// AI's trailers (so it looks like an escape) but the committer
		// has flipped to the human and the body carries the
		// `(cherry picked from commit <sha>)` marker. The gather layer
		// records both signals; the rule suppresses the finding so
		// the cherry-pick path is not penalized.
		if cherryPicked[c.SHA] {
			continue
		}

		// AC-1 / AC-2 / AC-3 — commit landed on a branch other than
		// the bound one. Fire one finding per violating commit (AC-10).
		// M-0161/AC-6: the message names the post-rename `resolved`
		// branch (what the binding currently resolves to via SHA);
		// when the scope was bound by name only, `resolved == bound`.
		findings = append(findings, Finding{
			Code:     CodeIsolationEscape.ID,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"commit %s: aiwf-actor: %q on %q escapes the active scope's bound branch %q",
				short(c.SHA), actor, strings.Join(actualBranches, ","), resolved,
			),
			EntityID: entity,
		})
	}
	return findings
}
