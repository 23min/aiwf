package check

import (
	"context"
	"os/exec"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// acks.go — M-0159/AC-3: the canonical home for the retroactive-
// acknowledgment SHA walker. Lifted from fsm_history_consistent.go
// where it originally landed for M-0136/AC-2 alongside
// illegalTransitionFindings; now exposed as an exported package
// symbol so the CLI gather layer in internal/cli/check/ can call
// it once per check invocation and pass the resulting map to every
// rule that consumes it. WalkAcknowledgedSHAs' doc comment below
// enumerates those rules for readers; the two rosters in
// internal/policies/acks_helper_lift.go enumerate them for the
// chokepoint. A new consumer is added to both — the comment is not a
// substitute for the roster that drives enforcement.
//
// The single-compute invariant and the agreement between comment and
// rosters are both policed by internal/policies/acks_helper_lift.go.

// WalkAcknowledgedSHAs walks HEAD's reachable history for commits
// carrying an `aiwf-force-for: <sha>` trailer (per M-0136) and
// returns the set of target SHAs. Consumers use it to exempt commits
// retroactively acknowledged via `aiwf acknowledge illegal`. Two
// rosters describe them, because the chokepoint checks two different
// properties of a consumer.
//
// The gather layer passes the map as a parameter to these exported
// rules, and their wiring is what classes 4a-4c check
// (ackedSHAsConsumers): FSMHistoryConsistent, RunIsolationEscape,
// RunTrailerVerbUnknown, RunIDRenameUntrailered (M-0160/AC-4),
// RunOrphanedAICommits, RunPromoteOnWrongBranch and RunProvenance
// (M-0292).
//
// Those rules, plus the leaf predicates at the end of
// FSMHistoryConsistent's forwarding chain —
// illegalTransitionFindings and forcedUntraileredFindings — make up
// the roster whose bodies must keep referencing the map, which is what
// class 4d checks (ackedSHAsBodyConsumers). So the second roster is
// the first plus those predicates. FSMHistoryConsistent is in it
// by forwarding the map to that chain rather than indexing it, the
// reference shape 4d accepts for a rule that delegates its lookup.
//
// Every consumer silences wholesale: an acknowledged SHA reports
// nothing from that rule. RunProvenance is the only one that emits
// more than one code, and it skips the commit ahead of all of them
// rather than selecting among them — see its doc.
//
// The verb consumes the walker too, to recognize
// a SHA it has already acknowledged and converge rather than append a
// duplicate record, so the verb's notion of "already acknowledged"
// is the rules' notion by construction rather than by agreement.
//
// Returns nil for non-git directories and empty histories; the
// consumers treat nil and an empty map identically (no
// exemptions). Per-SHA scoping is the closed-set guarantee — an
// acknowledgment for one SHA does NOT exempt findings against
// other commits.
//
// The walk is HEAD-reachable (not --all) because the exemption
// is DAG-scoped: a cherry-picked acknowledgment on a branch that
// doesn't include the original violation must not exempt
// findings on this branch. HEAD's reachable set is precisely the
// set of commits this branch sees, so the exemption only applies
// when the acknowledgment's history actually contains the
// offending commit.
//
// Reads via one `git log` subprocess + the gitops.ParseTrailers
// helper. Performance: O(reachable-commits) once per check
// invocation; for kernel-tree-sized repos under a second.
//
// AC-3 caller convention: the CLI gather layer at
// internal/cli/check/check.go::Run calls this exactly once and
// passes the result to every downstream rule listed above through
// a uniformly-named ackedSHAs parameter. Rule-internal recomputes
// are forbidden by PolicyAcksHelperLift (violation class 3c),
// which also pins the consumer list itself, so a rule that starts
// reading ackedSHAs without being named above fails the policy.
//
// M-0216/AC-5: derives from the shared HEAD walk (head) instead of
// spawning its own `git log HEAD` — the CLI gather layer computes
// WalkHeadCommits once and threads it in. The "Walk" name is retained
// because the acks_helper_lift policy (M-0159/AC-3) pins this exported
// symbol as the single ackedSHAs source; it now derives rather than
// walks. resolveFullSHA stays a git call (it resolves against the full
// object DB, which the in-memory HEAD set can't replicate). A nil/empty
// head yields nil — the same "no commits / no acks" signal the prior
// git-walk returned.
func WalkAcknowledgedSHAs(ctx context.Context, root string, head []HeadCommit) map[string]bool {
	if len(head) == 0 {
		return nil
	}
	acked := map[string]bool{}
	for i := range head {
		for _, tr := range head[i].Trailers {
			if tr.Key != gitops.TrailerForceFor {
				continue
			}
			sha := strings.TrimSpace(tr.Value)
			if sha == "" {
				continue
			}
			// Expand short SHAs to full SHAs so map lookups against
			// observation.Commit (always 40 hex) match. `git rev-parse
			// --verify <sha>` returns the canonical 40-char form; if
			// the lookup fails (acknowledgment targets a SHA not in
			// the local object database), the entry is dropped — the
			// predicate then falls through and fires normally, which
			// is the safe behavior.
			fullSHA := resolveFullSHA(ctx, root, sha)
			if fullSHA == "" {
				continue
			}
			acked[fullSHA] = true
		}
	}
	return acked
}

// WalkAcknowledgedSHAEntities is the per-(SHA, entity) variant of
// WalkAcknowledgedSHAs, added by G-0231 item 3 to feed
// RunUntrailedAudit's per-(commit, entity) finding shape with a
// matching per-(commit, entity) ack shape. Returns
// map[fullSHA]map[canonicalEntityID]bool.
//
// Only ack commits carrying BOTH `aiwf-force-for: <sha>` AND
// `aiwf-entity: <id>` count. SHA-only acks (the blanket shape the
// WalkAcknowledgedSHAs consumers use) do NOT suppress
// findings here — the per-(commit, entity) shape requires both
// sides. The verb's `git diff-tree` write-time check is what
// gives the (SHA, entity) pair its kernel-attested binding.
//
// Returns nil for non-git directories and empty histories; every
// consumer treats nil and an empty map identically (no
// exemptions).
//
// M-0216/AC-5: derives from the shared HEAD walk (head) — see
// WalkAcknowledgedSHAs for the retained-name / single-compute
// rationale.
func WalkAcknowledgedSHAEntities(ctx context.Context, root string, head []HeadCommit) map[string]map[string]bool {
	if len(head) == 0 {
		return nil
	}
	acked := map[string]map[string]bool{}
	for i := range head {
		var forceFor, entityID string
		for _, tr := range head[i].Trailers {
			switch tr.Key {
			case gitops.TrailerForceFor:
				forceFor = strings.TrimSpace(tr.Value)
			case gitops.TrailerEntity:
				entityID = strings.TrimSpace(tr.Value)
			}
		}
		if forceFor == "" || entityID == "" {
			// Either missing — this isn't a per-(SHA, entity) ack.
			// The bare per-SHA case is covered by
			// WalkAcknowledgedSHAs, not by this walker.
			continue
		}
		fullSHA := resolveFullSHA(ctx, root, forceFor)
		if fullSHA == "" {
			continue
		}
		// Canonicalize at ingest so a narrow-legacy trailer
		// (`aiwf-entity: G-1`) and a canonical-width finding lookup
		// (`G-0001`) match. The verb emits at canonical width
		// (entity.Canonicalize before write) but hand-rolled ack
		// commits or forward-compat shapes can write narrower
		// trailer values; reading them through Canonicalize here
		// closes the silent-miss failure mode.
		canonID := entity.Canonicalize(entityID)
		if acked[fullSHA] == nil {
			acked[fullSHA] = map[string]bool{}
		}
		acked[fullSHA][canonID] = true
	}
	return acked
}

// resolveFullSHA expands a short SHA (7-39 hex) to its full 40-char
// form via `git rev-parse --verify <sha>`. Returns the input unchanged
// when already 40 chars; returns "" when git can't resolve the SHA.
//
// Unexported because no caller outside this file needs it; the
// public surface of this file is WalkAcknowledgedSHAs.
func resolveFullSHA(ctx context.Context, root, sha string) string {
	if len(sha) == 40 {
		return sha
	}
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", sha+"^{commit}")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// findDanglingAckHint is a best-effort, local-clone-only diagnostic
// for G-0395: WalkAcknowledgedSHAs deliberately walks only HEAD's
// reachable history (see that function's own doc comment on why —
// DAG-scoping prevents a cross-branch acknowledgment from leaking
// into an unrelated branch's findings), so a history rewrite (a
// rebase, typically) that drops just the acknowledgment commit while
// leaving the originally-flagged commit reachable makes the
// illegal-transition finding reappear with no trace an acknowledgment
// ever existed.
//
// This never changes that behavior — the finding is correct to fire
// again, since the *current* history genuinely has no reachable
// acknowledgment. It only enriches the operator-facing message: git
// does not immediately destroy a commit dropped by a rebase (it
// becomes a "dangling" object, kept alive by the reflog until it
// expires or `git gc --prune` runs). Searching those dangling objects
// for a matching `aiwf-force-for: <targetSHA>` trailer recovers,
// best-effort, evidence that the finding was once acknowledged and
// names the commit so the operator can re-run `aiwf acknowledge
// illegal`.
//
// This is advisory only and proves nothing by its absence: git gc,
// a fresh clone, or CI's own checkout all make the dangling object
// unavailable, and an empty return here does not mean "never
// acknowledged" — only "no local evidence found." It intentionally
// does not persist any state of its own (CLAUDE.md's "no separate
// event log" commitment) — it reads only what git itself already
// keeps, best-effort, and forgets nothing new.
//
// Callers gate this to the already-failing path only: it runs `git
// fsck`, which walks the whole local object database and is not
// something to pay on every clean check.
func findDanglingAckHint(ctx context.Context, root, targetSHA string) string {
	cmd := exec.CommandContext(ctx, "git", "fsck", "--unreachable", "--no-reflogs")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	for _, danglingSHA := range parseDanglingCommitSHAs(string(out)) {
		if hint := danglingCommitAckHint(ctx, root, danglingSHA, targetSHA); hint != "" {
			return hint
		}
	}
	return ""
}

// parseDanglingCommitSHAs extracts the commit SHAs from `git fsck
// --unreachable`'s output ("unreachable <type> <sha>" per line),
// skipping every other object type (blob, tree, tag). Split out of
// findDanglingAckHint so the line-filtering logic is directly
// unit-testable against fabricated multi-type fsck output — a real
// git fsck run can't be coerced to emit a specific mix of dangling
// object types on demand, and downstream (`git show` on a non-commit
// object) tends to fail or return nothing usable regardless, which
// would make an integration-level test of this filter pass whether or
// not the filter itself is present.
func parseDanglingCommitSHAs(fsckOutput string) []string {
	var shas []string
	for _, line := range strings.Split(fsckOutput, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != "unreachable" || fields[1] != "commit" {
			continue
		}
		shas = append(shas, fields[2])
	}
	return shas
}

// danglingCommitAckHint reads danglingSHA's trailers and returns a
// hint string when one carries `aiwf-force-for: <targetSHA>`, or ""
// otherwise (including when danglingSHA's trailers can't be read at
// all — a defensive case, not expected for an object git fsck just
// reported).
func danglingCommitAckHint(ctx context.Context, root, danglingSHA, targetSHA string) string {
	cmd := exec.CommandContext(ctx, "git", "show", "-s", "--format=%(trailers:only=true,unfold=true)", danglingSHA)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil { //coverage:ignore defensive: git show on an object git fsck just reported as a real dangling commit in this same repo has no realistic failure mode
		return ""
	}
	for _, tr := range gitops.ParseTrailers(string(out)) {
		if tr.Key != gitops.TrailerForceFor {
			continue
		}
		if resolveFullSHA(ctx, root, strings.TrimSpace(tr.Value)) != targetSHA {
			continue
		}
		return "a commit (" + shortHash(danglingSHA) + ") carrying aiwf-force-for: " + targetSHA +
			" exists locally but is no longer reachable from HEAD — a rebase may have dropped a prior acknowledgment; " +
			"re-run `aiwf acknowledge illegal " + targetSHA + "` if so"
	}
	return ""
}
