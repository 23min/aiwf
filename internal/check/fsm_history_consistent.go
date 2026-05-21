package check

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// FSMHistoryConsistent is the kernel chokepoint that makes the per-
// entity status FSM a tree-invariant rather than just a verb-
// precondition (closes G-0132 when all of M-0130 lands). The rule
// walks every entity's commit history in DAG order, observes every
// status-change commit, and — once AC-2/3/4 land the per-subcode
// predicates — emits findings per violation.
//
// M-0130 lands the rule incrementally:
//
//   - AC-1 (this file): walker scaffolding. walkStatusChanges
//     enumerates status-change observations across every entity in
//     the tree via DAG-aware per-parent comparison. FSMHistoryConsistent
//     returns no findings yet — the per-subcode predicates land in
//     AC-2/3/4. The walker's correctness (per-parent comparison,
//     rename tracking via --follow, single-commit and no-change
//     short-circuits, multi-entity independence, branched-history
//     phantom-transition avoidance) is pinned by the test suite in
//     fsm_history_consistent_test.go.
//   - AC-2: illegal-transition subcode — observation's (Prior, Next)
//     is outside entity.AllowedTransitions and Trailers lacks
//     aiwf-force.
//   - AC-3: forced-untrailered subcode — sovereign-act-shape
//     transition (per entity.IsSovereignActShape) without aiwf-force
//     trailer.
//   - AC-4: manual-edit subcode — catch-all: legal-in-FSM AND not
//     sovereign-act-shape AND no aiwf-verb trailer. Includes audit-
//     only suppression.
//   - AC-5: hint table entries + SKILL.md rows (already landed).
//   - AC-6: audit catalog update (legal-workflows-audit.md).
//
// The rule is wired in internal/cli/check/check.go's Run() alongside
// RunProvenanceCheck, NOT in this package's Run(). The per-entity
// git-walk is too expensive for the per-commit pre-commit hook's
// policy-test path; pre-push and explicit `aiwf check` invocations
// get the full audit.
//
// Walker design contract (the load-bearing correctness pin for AC-1):
//
// The walker is DAG-aware, not linearization-aware. For each commit
// C that touched an entity's file, the walker compares C's status
// against the status at each of C's actual git parents (not against
// the linearization-neighbor commit in `git log --follow` output).
// The original AC-1 design walked linearization adjacency, which
// silently produced phantom transitions across branch boundaries —
// e.g., a retitle commit on a feature branch with status=proposed
// followed in `git log` order by a promote-to-active on a parallel
// branch would emit an "active → proposed" observation that
// corresponds to no real edit. Per-parent comparison eliminates the
// phantom by structurally restricting comparisons to actual parent-
// child edges in the DAG.
func FSMHistoryConsistent(ctx context.Context, root string, t *tree.Tree) []Finding {
	if t == nil || root == "" {
		return nil
	}
	// Don't short-circuit on `hasGitCommits` here: under a cancelled
	// context, hasGitCommits' subprocess fails and returns false,
	// which would false-negative as "empty repo" and silently swallow
	// the cancellation. Let NewBlobReader's IsRepo check surface the
	// failure as a history-walk-error finding instead. Empty repos
	// flow harmlessly through batchedWalkStatusChanges (the inner
	// BulkRevwalk has its own empty-repo short-circuit).
	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil {
		// Could not open the cat-file --batch subprocess (ctx
		// cancelled, not a git repo, or fork failure). Surface as a
		// single history-walk-error rather than silently swallowing —
		// the operator sees the failure rather than a green check that
		// hides it. Pre-existing non-repo dirs (no .git) also land
		// here; the M-0130 hasGitCommits short-circuit returned (nil,
		// nil) for them, which we preserve by checking the "repo
		// doesn't exist" shape explicitly.
		if !isRepoPath(ctx, root) {
			return nil
		}
		return []Finding{{
			Code:     "fsm-history-consistent",
			Subcode:  "history-walk-error",
			Severity: SeverityError,
			Message:  "could not open git cat-file --batch subprocess: " + err.Error(),
			Field:    "status",
		}}
	}
	defer func() { _ = br.Close() }()
	return fsmHistoryConsistentWithDeps(ctx, root, t, br)
}

// isRepoPath reports whether root is a git repo via the .git
// directory or worktree-pointer file's existence. Used as the cheap
// pre-flight check that distinguishes "not a repo" (silent return)
// from "subprocess failed" (history-walk-error finding) when
// NewBlobReader errors.
//
// Doesn't use exec — pure filesystem check so a cancelled context
// doesn't false-negative on this branch the way it would on the
// gitops.IsRepo subprocess call.
func isRepoPath(_ context.Context, root string) bool {
	_, err := os.Stat(filepath.Join(root, ".git"))
	return err == nil
}

// blobReader is the rule's blob-reading dep seam introduced in
// M-0137/AC-3+5 to let tests provoke per-blob failure modes that
// real subprocesses don't reliably produce (corrupting a single blob
// is fs-dependent; cancellation kills the whole walk). Production
// satisfies this via *gitops.BlobReader's Read/Close methods.
//
// Kept unexported because the dep injection is rule-internal; no
// outside consumer needs it.
type blobReader interface {
	Read(commit, path string) ([]byte, error)
	Close() error
}

// fsmHistoryConsistentWithDeps is the testable variant of
// FSMHistoryConsistent: it accepts an explicit blobReader the test
// can substitute. The production entry-point opens a real
// *gitops.BlobReader and calls this function.
//
// Walker errors surface as `fsm-history-consistent/history-walk-
// error` findings (severity error) — partial walks still produce
// findings for the entities/commits that read successfully, while
// the failed (entity, commit) pairs each surface a walk-error
// finding. The M-0130 silent-swallow at the old FSMHistoryConsistent
// path is gone (closes the load-bearing correctness issue G-0149
// flagged).
//
// Kept unexported; tests live in package check (internal).
func fsmHistoryConsistentWithDeps(ctx context.Context, root string, t *tree.Tree, br blobReader) []Finding {
	if t == nil || root == "" {
		return nil
	}
	observations, walkErrors, fatalErr := batchedWalkStatusChanges(ctx, root, t, br)

	var findings []Finding
	if fatalErr != nil {
		// Walker-level failure (BulkRevwalk subprocess crash, ctx
		// cancelled). Emit a single rule-scoped history-walk-error;
		// per-blob walkErrors collected before the fatal are still
		// surfaced below alongside.
		findings = append(findings, Finding{
			Code:     "fsm-history-consistent",
			Subcode:  "history-walk-error",
			Severity: SeverityError,
			Message:  "walker failed: " + fatalErr.Error(),
			Field:    "status",
		})
	}
	findings = append(findings, historyWalkErrorFindings(walkErrors)...)

	acksByEntity := walkAuditOnlyAcksByEntity(ctx, root)
	ackedObs := computeAckedObservations(ctx, root, observations, acksByEntity)
	ackedSHAs := walkAcknowledgedSHAs(ctx, root)
	findings = append(findings, illegalTransitionFindings(observations, ackedSHAs)...)
	findings = append(findings, forcedUntraileredFindings(observations)...)
	findings = append(findings, manualEditFindings(observations, ackedObs)...)
	return findings
}

// statusChange records one observed status-change for an entity at
// one commit, relative to one of the commit's parents. Multi-parent
// (merge) commits may yield one observation per parent where the
// status differs at that parent's path-at-commit.
//
// Unexported because it's an intermediate value passed from the
// walker to the per-subcode predicates that land in AC-2/3/4 — not a
// public surface of the check package.
//
// Fields:
//   - EntityID, EntityKind, Path: identify the entity and where its
//     file lives at the observed commit. Path may differ across
//     observations of the same entity when `aiwf rename` has moved
//     the file (--follow tracks the history).
//   - Commit: full SHA of the status-change commit (not the parent).
//   - Parent: full SHA of the parent commit this observation is
//     relative to. For single-parent commits, this is the only
//     parent. For multi-parent (merge) commits, the same Commit may
//     appear in multiple observations — one per parent where the
//     status differed.
//   - Prior: the status field value at Parent (read at Path).
//   - Next: the status field value at Commit (read at Path).
//   - Trailers: the aiwf-* trailers parsed from Commit's message.
//     Keys are bare (no "aiwf-" prefix stripping). Used by AC-2/3/4
//     predicates to classify the change.
type statusChange struct {
	EntityID   string
	EntityKind entity.Kind
	Commit     string
	Parent     string
	Path       string
	Prior      string
	Next       string
	Trailers   map[string]string
	// IsMergeCommit is true when Commit has more than one parent —
	// a merge commit. Set uniformly by the walker; predicates apply
	// per-subcode policy per D-0010 (supersedes D-0009): all three
	// subcodes skip merge-commit observations. The walker emits
	// them so future predicates with different policies can opt in
	// without revisiting the walker.
	IsMergeCommit bool
}

// walkStatusChanges is a thin adapter retained for the existing
// unit tests that pre-date the M-0137 retrofit. It opens a real
// *gitops.BlobReader and delegates to batchedWalkStatusChanges,
// returning (observations, fatalErr) and dropping per-blob
// walkErrors — those are surfaced as findings only via the
// FSMHistoryConsistent / fsmHistoryConsistentWithDeps entry-points.
//
// New callers should use the entry-points directly; this helper
// stays only so M-0130's test fixtures continue to drive the same
// observation shape.
func walkStatusChanges(ctx context.Context, root string, t *tree.Tree) ([]statusChange, error) {
	if t == nil || root == "" {
		return nil, nil
	}
	if !hasGitCommits(ctx, root) {
		return nil, nil
	}
	br, err := gitops.NewBlobReader(ctx, root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = br.Close() }()
	observations, _, fatalErr := batchedWalkStatusChanges(ctx, root, t, br)
	return observations, fatalErr
}

// parseStatusFromFrontmatter extracts the status field from an
// entity file's YAML frontmatter (the block between leading `---`
// and the next `---`). Returns "" for any failure mode: missing
// frontmatter, unterminated frontmatter, YAML parse error, or
// absent status field.
//
// Accepts both `---\n` and `---\r\n` opening sequences so files
// written on Windows hosts still parse.
func parseStatusFromFrontmatter(content []byte) string {
	var rest []byte
	switch {
	case bytes.HasPrefix(content, []byte("---\n")):
		rest = content[4:]
	case bytes.HasPrefix(content, []byte("---\r\n")):
		rest = content[5:]
	default:
		return ""
	}
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return ""
	}
	var meta struct {
		Status string `yaml:"status"`
	}
	if err := yaml.Unmarshal(rest[:end], &meta); err != nil {
		return ""
	}
	return meta.Status
}

// hasGitCommits reports whether root is a git repo with at least one
// commit reachable from HEAD. Returns false for non-git directories,
// for git repos with no commits yet, and for any other condition
// that makes HEAD unresolvable.
func hasGitCommits(ctx context.Context, root string) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "--quiet", "HEAD")
	cmd.Dir = root
	return cmd.Run() == nil
}

// illegalTransitionFindings emits one fsm-history-consistent finding
// per observation whose (Prior, Next) is not an edge in the kind's
// FSM AND whose commit lacks an aiwf-force trailer AND whose commit
// is not a merge.
//
// M-0130/AC-2 predicate per the spec body §3:
//
//	"Subcode illegal-transition — change is not in the FSM and no force trailer"
//
// Per D-0010 (supersedes D-0009), the predicate SKIPS merge-commit
// observations. The rationale: merge commits emit per-parent
// observations that produce routine noise on every feature-branch
// integration (main's pre-merge view of a milestone at `draft` vs
// the merge result at `done` looks like an illegal `draft → done`
// even though the actual progression on the feature branch was
// `draft → in_progress → done`, all legal). Non-merge commits still
// audit normally — a direct hand-edit, a buggy verb, or an attempted
// skip-ahead promote on any branch is caught by per-parent comparison
// at the original commit.
//
// The aiwf-verb trailer's presence is not part of the predicate —
// illegal is illegal regardless of who tried to make the change. A
// verb-routed illegal transition without force is the "verb's FSM
// check drifted from entity.AllowedTransitions" case, which this
// rule deliberately catches as the tree-level chokepoint.
//
// Force-trailer presence (key-present; value irrelevant) exempts
// the transition: it's the kernel's sovereign override and the
// trailer records the human's accountability per the provenance
// model.
//
// M-0136/AC-2 extension: ackedSHAs carries the set of commit SHAs
// that have been retroactively acknowledged via `aiwf
// acknowledge-illegal` — current-day commits with an
// `aiwf-force-for: <historical-sha>` trailer in HEAD's reachable
// history. Observations whose Commit appears in ackedSHAs are
// exempted (same shape as the inline aiwf-force exemption, but
// recorded out-of-band so the historical commit doesn't need to be
// rewritten).
//
// The exemption is per-SHA (M-0136/AC-2 scoped): an acknowledgment
// for one SHA does NOT exempt findings against other illegal
// commits. Per-SHA scoping is the closed-set guarantee — there is
// no "exempt everything" knob.
func illegalTransitionFindings(observations []statusChange, ackedSHAs map[string]bool) []Finding {
	var out []Finding
	for i := range observations {
		o := &observations[i]
		if o.IsMergeCommit {
			continue
		}
		if isLegalTransition(o.EntityKind, o.Prior, o.Next) {
			continue
		}
		if _, hasForce := o.Trailers[gitops.TrailerForce]; hasForce {
			continue
		}
		if ackedSHAs[o.Commit] {
			continue
		}
		out = append(out, Finding{
			Code:     "fsm-history-consistent",
			Subcode:  "illegal-transition",
			Severity: SeverityError,
			Message: "entity " + o.EntityID + " status changed " + o.Prior + " → " + o.Next +
				" in commit " + shortHash(o.Commit) +
				" — not a legal " + string(o.EntityKind) +
				" FSM transition and no aiwf-force trailer",
			Path:     o.Path,
			EntityID: o.EntityID,
			Field:    "status",
		})
	}
	return out
}

// walkAcknowledgedSHAs walks HEAD's reachable history for commits
// carrying an `aiwf-force-for: <sha>` trailer (per M-0136) and
// returns the set of target SHAs. The set is consumed by
// illegalTransitionFindings to exempt acknowledged illegal-transition
// observations.
//
// Returns nil for non-git directories and empty histories; the
// consumer treats nil and an empty map identically (no exemptions).
//
// The walk is HEAD-reachable (not --all) because the exemption is
// DAG-scoped: a cherry-picked acknowledgment on a branch that
// doesn't include the original violation must not exempt findings
// on this branch. HEAD's reachable set is precisely the set of
// commits this branch sees, so the exemption only applies when the
// acknowledgment's history actually contains the offending commit.
//
// Reads via one `git log` subprocess + the gitops.ParseTrailers
// helper. Performance: O(reachable-commits) once per check
// invocation; for kernel-tree-sized repos under a second.
func walkAcknowledgedSHAs(ctx context.Context, root string) map[string]bool {
	if root == "" || !hasGitCommits(ctx, root) {
		return nil
	}
	cmd := exec.CommandContext(ctx, "git", "log",
		"--pretty=format:%H%x00%(trailers:unfold=true)%x00",
		"HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	acked := map[string]bool{}
	parts := strings.Split(string(out), "\x00")
	for i := 0; i+1 < len(parts); i += 2 {
		// parts[i] is the commit SHA (one acknowledged each); parts[i+1]
		// is its trailer block.
		trailerBlock := parts[i+1]
		if trailerBlock == "" {
			continue
		}
		parsed := gitops.ParseTrailers(trailerBlock)
		for _, tr := range parsed {
			if tr.Key != gitops.TrailerForceFor {
				continue
			}
			sha := strings.TrimSpace(tr.Value)
			if sha == "" {
				continue
			}
			acked[sha] = true
		}
	}
	return acked
}

// isLegalTransition reports whether (prior → next) is an edge in
// the kind's FSM. Returns false when the kind is unrecognized (no
// FSM to validate against), when prior is not a recognized state
// for the kind, or when next is not in prior's outgoing edge set.
//
// Sub-FSM kinds (KindAC, KindTDDPhase declared in the
// workflows/spec package) are not reachable here today: the walker
// enumerates only entity-level file paths, and AC / TDD-phase
// state lives in milestone frontmatter, not in its own files. If
// a future kind extension adds per-AC files (or similar), this
// helper widens accordingly.
func isLegalTransition(k entity.Kind, prior, next string) bool {
	for _, allowed := range entity.AllowedTransitions(k, prior) {
		if allowed == next {
			return true
		}
	}
	return false
}

// shortHash returns the 8-character abbreviated form of a commit
// SHA. Falls back to the original string when shorter than 8 chars.
func shortHash(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// manualEditFindings emits one fsm-history-consistent finding per
// observation whose (Prior, Next) is FSM-legal, NOT a sovereign-act-
// shape, whose commit lacks an aiwf-verb trailer, is not a merge,
// and whose commit is not ack-covered (per ackedObs: an audit-only
// commit exists in HEAD's reachable history that is a descendant of
// the offending commit AND carries aiwf-entity matching this entity).
//
// M-0130/AC-4 predicate per the spec body §3:
//
//	"Subcode manual-edit — change has no aiwf-verb: trailer at all
//	 (overlaps with provenance-untrailered-entity-commit but with FSM-
//	 specific framing)"
//
// The predicate is the catch-all of D-0008's disjoint partition:
//
//   - AC-2 owns FSM-illegal transitions.
//   - AC-3 owns FSM-legal sovereign-act-shape transitions.
//   - AC-4 owns FSM-legal non-sovereign-act-shape transitions where
//     the kernel was bypassed.
//
// The three subcodes partition the legal-status-change observation
// space; each observation triggers at most one subcode by construction.
//
// Per D-0010 (supersedes D-0009), the predicate SKIPS merge-commit
// observations. Rationale carries over from AC-2/AC-3: merges produce
// per-parent integration noise that doesn't represent a real edit; a
// non-aiwf-verb manual edit will be caught at the original commit by
// the non-merge per-parent edge.
//
// Severity is WARNING, aligned with the parallel
// provenance-untrailered-entity-commit rule that surfaces the same
// shape from the provenance side. The intended user response is the
// audit-only backfill (`aiwf <verb> --audit-only --reason "..."`),
// which records a separate commit acknowledging the manual flip
// without rewriting history. ERROR severity would block pushes for
// state that is already correct on disk; warning gives the operator
// space to backfill the trail.
//
// Audit-only suppression per D-0008 + the parallel rule's cooperation
// pattern: the audit-only commit lives on a SEPARATE, later commit (an
// empty commit carrying aiwf-audit-only + aiwf-entity), not on the
// offending status-change commit itself. walkAuditOnlyAckedEntities
// pre-collects the per-entity ack set; the predicate consults it via
// the acked map. The suppression is scoped strictly to manual-edit;
// illegal-transition and forced-untrailered are unaffected (per D-0008,
// audit-only doesn't claim FSM-legality or sovereign-discipline).
//
// Ack semantics are DAG-aware: an ack covers an observation only when
// the ack commit is a descendant of the observation commit (the ack
// genuinely came AFTER the flip in topology). A cherry-picked ack
// landing on a branch that doesn't include the offence does NOT
// suppress the finding. computeAckedObservations does the per-pair
// ancestor check via cached `git rev-list <ack>` ancestor sets;
// callers receive the ackedObs map and need only check membership.
func manualEditFindings(observations []statusChange, ackedObs map[string]bool) []Finding {
	var out []Finding
	for i := range observations {
		o := &observations[i]
		if o.IsMergeCommit {
			continue
		}
		if !isLegalTransition(o.EntityKind, o.Prior, o.Next) {
			continue
		}
		if entity.IsSovereignActShape(o.EntityKind, o.Prior, o.Next) {
			continue
		}
		if _, hasVerb := o.Trailers[gitops.TrailerVerb]; hasVerb {
			continue
		}
		if ackedObs[o.Commit] {
			continue
		}
		out = append(out, Finding{
			Code:     "fsm-history-consistent",
			Subcode:  "manual-edit",
			Severity: SeverityWarning,
			Message: "entity " + o.EntityID + " status changed " + o.Prior + " → " + o.Next +
				" in commit " + shortHash(o.Commit) +
				" — legal " + string(o.EntityKind) +
				" FSM transition but commit has no aiwf-verb trailer (kernel bypassed)",
			Path:     o.Path,
			EntityID: o.EntityID,
			Field:    "status",
		})
	}
	return out
}

// walkAuditOnlyAcksByEntity walks every commit reachable from HEAD
// and returns entity ID → list of ack commit SHAs (deduplicated). The
// entity ID keys are canonicalized with composite roots rolled up
// (`M-001/AC-1` rolls up to `M-0001`), mirroring the existing
// provenance audit's compositeRoot+Canonicalize discipline.
//
// computeAckedObservations consumes the per-entity ack lists and
// performs the per-(obs, ack) ancestor check to produce the
// observation-level ack set. Returning lists (not a flat set) lets the
// consumer's ancestor cache amortize: each ack commit's ancestor set
// is computed once and reused across every observation that might be
// covered by it.
//
// Returns nil for non-git directories and empty histories; the
// consumer treats nil and empty as equivalent (no acks).
func walkAuditOnlyAcksByEntity(ctx context.Context, root string) map[string][]string {
	if root == "" || !hasGitCommits(ctx, root) {
		return nil
	}
	// %x00 between fields keeps trailer blocks (which contain newlines)
	// distinguishable from the SHA boundary. The trailing %x00 closes
	// the last commit's trailer block so the parser doesn't drift into
	// the next SHA.
	cmd := exec.CommandContext(ctx, "git", "log",
		"--pretty=format:%H%x00%(trailers:unfold=true)%x00",
		"HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	parts := strings.Split(string(out), "\x00")
	// parts layout: [SHA, trailers, SHA, trailers, …, possibly trailing
	// chunk]. Even indices are SHAs, odd are trailer blocks. The first
	// SHA may have leading whitespace from git's inter-record newline.
	acks := make(map[string][]string)
	for i := 0; i+1 < len(parts); i += 2 {
		sha := strings.TrimSpace(parts[i])
		if sha == "" {
			continue
		}
		parsed := gitops.ParseTrailers(parts[i+1])
		var hasAuditOnly bool
		var entID string
		for _, t := range parsed {
			switch t.Key {
			case gitops.TrailerAuditOnly:
				hasAuditOnly = true
			case gitops.TrailerEntity:
				entID = strings.TrimSpace(t.Value)
			}
		}
		if !hasAuditOnly || entID == "" {
			continue
		}
		canonID := entity.Canonicalize(compositeRoot(entID))
		acks[canonID] = append(acks[canonID], sha)
	}
	return acks
}

// computeAckedObservations returns the set of observation commit SHAs
// that are properly ack-covered: for the observation's entity, at
// least one ack commit exists whose ancestor set includes the
// observation's commit. "Ancestor of the ack" means the offending
// commit is reachable from the ack — i.e., the ack came AFTER the
// flip in DAG topology, the natural retrospective-acknowledgment
// direction.
//
// Why the ancestor check matters: cherry-picking an ack onto a branch
// that doesn't include the flip would, under naive set-membership
// semantics, falsely suppress the finding. The ack is FOR a flip that
// the ack's branch never observed; the ack's content is reused but
// the topology says it doesn't cover anything on this branch. The
// ancestor check pins the suppression to "the operator saw the flip
// and acknowledged it" — i.e., the ack commit's history actually
// contains the flip.
//
// Performance: each ack's ancestor set is computed once via
// `git rev-list <ack-sha>`, then reused across every observation in
// the same entity. For a tree with M acks and N observations, that's
// M `git rev-list` calls + N×M map lookups. The rev-list per ack is
// O(reachable-commits); for the kernel tree (~thousand commits, a
// handful of acks) the overhead is well under a second.
func computeAckedObservations(ctx context.Context, root string, observations []statusChange, acksByEntity map[string][]string) map[string]bool {
	if len(acksByEntity) == 0 || len(observations) == 0 {
		return nil
	}
	ancestorCache := make(map[string]map[string]bool)
	ackedObs := make(map[string]bool)
	for i := range observations {
		o := &observations[i]
		canonID := entity.Canonicalize(compositeRoot(o.EntityID))
		acks := acksByEntity[canonID]
		for _, ackSHA := range acks {
			ancestors, cached := ancestorCache[ackSHA]
			if !cached {
				ancestors = revListAncestors(ctx, root, ackSHA)
				ancestorCache[ackSHA] = ancestors
			}
			if ancestors[o.Commit] {
				ackedObs[o.Commit] = true
				break
			}
		}
	}
	return ackedObs
}

// revListAncestors returns the set of commit SHAs reachable from sha
// (sha itself plus all of its ancestors), via `git rev-list <sha>`.
// Returns nil on any read failure; callers treat nil as the empty set
// (no ancestors → no suppression).
func revListAncestors(ctx context.Context, root, sha string) map[string]bool {
	cmd := exec.CommandContext(ctx, "git", "rev-list", sha)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	ancestors := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			ancestors[line] = true
		}
	}
	return ancestors
}

// forcedUntraileredFindings emits one fsm-history-consistent finding
// per observation whose (Prior, Next) is a sovereign-act-shape (per
// entity.IsSovereignActShape), whose commit's aiwf-actor trailer is
// NOT human/-prefixed, whose commit lacks an aiwf-force trailer, AND
// whose commit is not a merge.
//
// M-0130/AC-3 predicate per the spec body §3:
//
//	"Subcode forced-untrailered — change matches a sovereign-act shape
//	 (e.g., epic proposed → active) but lacks the force trailer"
//
// The spec body's framing is shorthand; the predicate mirrors M-0095's
// verb gate (requireHumanActorForSovereignAct), which the kernel's
// provenance doctrine ratifies via entity/sovereign.go's defining
// comment: a sovereign-act-shape transition requires "a `human/` actor
// by default, or `--force --reason \"...\"` from a non-human actor."
// Either gesture satisfies the discipline; both gates exempt
// accordingly. AC-3 is the tree-level audit chokepoint behind the
// verb gate, so the predicates must agree.
//
// Per D-0010 (supersedes D-0009), the predicate SKIPS merge-commit
// observations. The rationale carries over from AC-2: merges produce
// per-parent integration noise that doesn't represent a real edit;
// sovereign-act-shape edits routed across a feature branch will be
// caught at the original commit by the non-merge per-parent edge.
//
// Disjoint with AC-2's illegal-transition by construction (D-0008's
// closed-set invariant): every entry in entity.SovereignActShapes is
// FSM-legal — sovereign-act-shape is a property over legal transitions,
// never below them. So a single observation can satisfy at most one of
// the two predicates' core gates.
//
// The aiwf-verb trailer's presence is NOT part of the predicate. A
// verb-mediated activation by a non-human actor without `--force`
// still fires — that is precisely the case the rule was authored to
// catch (older binary, sloppy bot, etc.).
//
// Trust-model note: both human-actor and --force are honor-system
// trailers — the operator (LLM or human) writes them based on
// runtime-derived identity (`git config user.email`). The kernel's
// provenance model accepts adversarial subversion as out of scope and
// relies on the transparent git-log audit trail as backstop. The
// predicate's job is surfacing visible discipline gaps, not blocking
// adversarial behavior.
func forcedUntraileredFindings(observations []statusChange) []Finding {
	var out []Finding
	for i := range observations {
		o := &observations[i]
		if o.IsMergeCommit {
			continue
		}
		if !entity.IsSovereignActShape(o.EntityKind, o.Prior, o.Next) {
			continue
		}
		if _, hasForce := o.Trailers[gitops.TrailerForce]; hasForce {
			continue
		}
		if strings.HasPrefix(o.Trailers[gitops.TrailerActor], "human/") {
			continue
		}
		out = append(out, Finding{
			Code:     "fsm-history-consistent",
			Subcode:  "forced-untrailered",
			Severity: SeverityError,
			Message: "entity " + o.EntityID + " status changed " + o.Prior + " → " + o.Next +
				" in commit " + shortHash(o.Commit) +
				" — sovereign-act-shape " + string(o.EntityKind) +
				" transition by non-human actor without aiwf-force trailer",
			Path:     o.Path,
			EntityID: o.EntityID,
			Field:    "status",
		})
	}
	return out
}
