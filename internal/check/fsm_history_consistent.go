package check

import (
	"bytes"
	"context"
	"os/exec"
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
	observations, err := walkStatusChanges(ctx, root, t)
	if err != nil {
		// AC-3/4 may route walker errors into the finding stream
		// (e.g., a "history-walk-error" subcode). For now the rule is
		// a clean no-op for trees where the per-entity git log fails
		// (rare; usually permission issues or transient cancellation).
		return nil
	}
	acksByEntity := walkAuditOnlyAcksByEntity(ctx, root)
	ackedObs := computeAckedObservations(ctx, root, observations, acksByEntity)
	var findings []Finding
	findings = append(findings, illegalTransitionFindings(observations)...)
	findings = append(findings, forcedUntraileredFindings(observations)...)
	findings = append(findings, manualEditFindings(observations, ackedObs)...)
	return findings
}

// blobReader is the rule's blob-reading dep seam introduced in
// M-0137/AC-3+5 RED phase to let tests provoke per-blob failure modes
// that real subprocesses don't reliably produce (corrupting a single
// blob is fs-dependent; cancellation kills the whole walk). Production
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
// can substitute. In M-0137/AC-3+4+5 RED phase this delegates to
// FSMHistoryConsistent and ignores the dep — the AC-4/AC-5 test
// suites assert findings the current rule does not emit, so the
// dep's behavior is observed only after GREEN swaps in the new
// batched walker.
//
// Kept unexported; tests live in package check (internal).
func fsmHistoryConsistentWithDeps(ctx context.Context, root string, t *tree.Tree, _ blobReader) []Finding {
	return FSMHistoryConsistent(ctx, root, t)
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

// walkStatusChanges enumerates DAG-aware status-change observations
// across every entity in t. Returns one observation per (entity,
// commit, parent) tuple where the entity's status at the parent
// (under the same on-disk path) differs from its value at the commit
// itself.
//
// Returns (nil, nil) when t is nil, root is empty, or root is not a
// git repo with at least one commit reachable from HEAD. Per-entity
// walker errors propagate as (nil, err) — callers can choose to
// swallow or route them as findings (FSMHistoryConsistent swallows
// for AC-1).
func walkStatusChanges(ctx context.Context, root string, t *tree.Tree) ([]statusChange, error) {
	if t == nil || root == "" {
		return nil, nil
	}
	if !hasGitCommits(ctx, root) {
		return nil, nil
	}
	var out []statusChange
	for _, e := range t.Entities {
		if e == nil || e.Path == "" {
			continue
		}
		changes, err := walkOneEntity(ctx, root, e)
		if err != nil {
			return nil, err
		}
		out = append(out, changes...)
	}
	return out, nil
}

// walkOneEntity returns DAG-aware status-change observations for a
// single entity.
//
// For each commit C in the entity's `git log --follow` history:
//
//  1. Enumerate C's parents (`git log -1 --pretty=format:%P C`).
//  2. For each parent P, read the entity file at (P, path-at-C) via
//     `git show P:<path>`. If the read fails — the file doesn't
//     exist at P under path-at-C (P pre-dates an add, or C is a
//     rename and P has the file under the OLD name), or the file
//     has no parseable frontmatter — the (P, C) pair is silently
//     skipped.
//  3. Read C's own status at path-at-C.
//  4. Emit one observation when both statuses are non-empty and
//     differ.
//
// Rename handling: `git log --follow` traverses the rename, so the
// entity's pre-rename commits appear in the touches list with their
// OLD path, and post-rename commits with their NEW path. Each commit
// is compared against its actual parent at the parent's own path
// (which matches the file's path AT that parent). The (rename
// commit, its parent) pair itself produces no observation — the
// parent has the file at the OLD name, `git show P:<NEW-name>`
// fails, the pair is skipped. Pure renames don't change status, so
// no observation is lost. The rare commit that both renames AND
// changes status is unobserved — accepted as a known non-handling.
//
// Multi-parent (merge) commits emit per-parent observations: if M
// has parents P1 and P2 with different statuses, M produces up to
// two observations. Whether merges count as "real" predicate events
// is left to the AC-2/3/4 predicates' filtering. This avoids
// baking merge semantics into the walker.
func walkOneEntity(ctx context.Context, root string, e *entity.Entity) ([]statusChange, error) {
	pairs, err := listCommitPathPairs(ctx, root, e.Path)
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, nil
	}
	var out []statusChange
	for _, p := range pairs {
		parents := commitParents(ctx, root, p.Commit)
		if len(parents) == 0 {
			// Root commit: no parent to compare against. The file may
			// have appeared at this commit (the initial import) — no
			// prior status to compute a delta against. Skip.
			continue
		}
		next := statusAtCommitPath(ctx, root, p.Commit, p.Path)
		if next == "" {
			continue
		}
		var trailers map[string]string
		isMerge := len(parents) > 1
		for _, parent := range parents {
			prior := statusAtCommitPath(ctx, root, parent, p.Path)
			if prior == "" || prior == next {
				continue
			}
			if trailers == nil {
				trailers = commitTrailers(ctx, root, p.Commit)
			}
			out = append(out, statusChange{
				EntityID:      e.ID,
				EntityKind:    e.Kind,
				Commit:        p.Commit,
				Parent:        parent,
				Path:          p.Path,
				Prior:         prior,
				Next:          next,
				Trailers:      trailers,
				IsMergeCommit: isMerge,
			})
		}
	}
	return out, nil
}

// commitPathPair couples a commit SHA with the file path at that
// commit. The path may differ across pairs when --follow has
// traversed a rename; reading the file's content at a given commit
// requires both values (`git show <sha>:<path>`).
type commitPathPair struct {
	Commit string
	Path   string
}

// listCommitPathPairs returns (commit SHA, path-at-commit) pairs for
// every commit that touched currentPath, INCLUDING merge commits.
// Uses --follow so renames are tracked across the path's history;
// --name-only gives the path-at-each-commit, parsed alongside the
// COMMIT-prefixed SHA lines.
//
// The `-m` flag is load-bearing for D-0009's merge-policy contract:
// without it, `git log --follow` silently excludes merge commits
// (`--follow` defaults to first-parent semantics; merges are
// invisible). `-m` treats each merge as a patch against each parent,
// which makes the merge commit appear in `--name-only` whenever the
// file differs from at least one parent. The walker's per-parent
// comparison (via commitParents) then emits observations relative
// to each parent, and AC-2 fires on illegal-transition merges per
// D-0009. Confirmed empirically: without -m, a merge that integrates
// an illegal feature-branch state into trunk produces zero merge-
// commit observations and AC-2 cannot catch the integration.
//
// The custom "COMMIT %H" prefix distinguishes the SHA lines from the
// path lines without relying on whitespace heuristics (git's default
// --pretty output mixes blank lines and metadata in ways that break
// naive parsing).
//
// Order doesn't matter to the DAG-aware walker — each pair is
// compared against its commit's actual git parents independently —
// so we deliberately do not reverse the result. (The original AC-1
// design relied on adjacency-in-list semantics and needed reverse-
// ordering; the bug that triggered M-0130's AC-1 redo originated in
// that very adjacency assumption.)
func listCommitPathPairs(ctx context.Context, root, currentPath string) ([]commitPathPair, error) {
	cmd := exec.CommandContext(ctx, "git", "log",
		"--follow", "-m", "--name-only",
		"--pretty=format:COMMIT %H",
		"--", currentPath)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// Dedupe by (commit, path): with -m, a merge commit whose content
	// differs from BOTH parents appears twice in the output (once
	// per parent diff). The walker's per-parent comparison runs
	// inside walkOneEntity, so the duplicate listing would emit
	// duplicate observations and inflate AC-2's finding count.
	// Deduping at the pair level collapses to one entry per
	// (commit, path) — the per-parent fan-out then happens once,
	// downstream.
	seen := make(map[string]struct{})
	var pairs []commitPathPair
	var pendingCommit string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "COMMIT "); ok {
			pendingCommit = rest
			continue
		}
		if pendingCommit == "" {
			continue
		}
		key := pendingCommit + "\x00" + line
		if _, dup := seen[key]; dup {
			pendingCommit = ""
			continue
		}
		seen[key] = struct{}{}
		pairs = append(pairs, commitPathPair{Commit: pendingCommit, Path: line})
		pendingCommit = ""
	}
	return pairs, nil
}

// commitParents returns the parent SHAs of the named commit. Returns
// nil for the root commit (no parents) and for any read failure.
// Multi-parent (merge) commits return all parents in git's declared
// order — first-parent is conventionally the mainline-being-merged-
// into; the walker treats all parents uniformly.
func commitParents(ctx context.Context, root, commit string) []string {
	cmd := exec.CommandContext(ctx, "git", "log", "-1",
		"--pretty=format:%P", commit)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil
	}
	return strings.Fields(trimmed)
}

// statusAtCommitPath reads the entity file at the named commit +
// path via `git show <commit>:<path>` and parses the status field
// from its YAML frontmatter. Returns "" when the file doesn't exist
// at that commit, has no frontmatter delimiter, has no status field,
// or fails YAML parsing — all four "I can't determine the status
// here" cases collapse to the same skip-this-pair signal.
func statusAtCommitPath(ctx context.Context, root, commit, path string) string {
	cmd := exec.CommandContext(ctx, "git", "show", commit+":"+path)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return parseStatusFromFrontmatter(out)
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

// commitTrailers reads the commit's aiwf-* trailers (and any other
// trailer-shaped lines) and returns them as a key → value map.
// Returns nil when git emits no trailers or the read fails.
//
// Multiple trailers with the same key collapse to the last value,
// which is sufficient for the AC-2/3/4 predicates' boolean-ish use
// ("is aiwf-verb present?", "is aiwf-force present?"). If a future
// subcode needs multi-value-per-key semantics, switch to the slice
// form.
func commitTrailers(ctx context.Context, root, commit string) map[string]string {
	cmd := exec.CommandContext(ctx, "git", "log", "-1",
		"--pretty=%(trailers:only=true,unfold=true)", commit)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	parsed := gitops.ParseTrailers(string(out))
	if len(parsed) == 0 {
		return nil
	}
	m := make(map[string]string, len(parsed))
	for _, tr := range parsed {
		m[tr.Key] = tr.Value
	}
	return m
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
func illegalTransitionFindings(observations []statusChange) []Finding {
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
