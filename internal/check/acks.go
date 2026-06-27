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
// it once per check invocation and pass the resulting map to all
// four rules that consume it (fsm-history-consistent,
// isolation-escape, trailer-verb-unknown, id-rename-untrailered;
// the fourth added at M-0160/AC-4).
//
// The single-compute invariant is policed by
// internal/policies/acks_helper_lift.go.

// WalkAcknowledgedSHAs walks HEAD's reachable history for commits
// carrying an `aiwf-force-for: <sha>` trailer (per M-0136) and
// returns the set of target SHAs. The set is consumed by
// illegalTransitionFindings, RunIsolationEscape,
// RunTrailerVerbUnknown, and RunIDRenameUntrailered (M-0160/AC-4)
// to exempt commits that have been retroactively acknowledged via
// `aiwf acknowledge illegal`.
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
// passes the result to all four downstream rules through a
// uniformly-named ackedSHAs parameter (id-rename-untrailered
// added at M-0160/AC-4 as the fourth consumer). Rule-internal
// recomputes are forbidden by PolicyAcksHelperLift (violation
// class 3c).
func WalkAcknowledgedSHAs(ctx context.Context, root string) map[string]bool {
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
// `aiwf-entity: <id>` count. SHA-only acks (the legacy seven
// rules' blanket shape via WalkAcknowledgedSHAs) do NOT suppress
// findings here — the per-(commit, entity) shape requires both
// sides. The verb's `git diff-tree` write-time check is what
// gives the (SHA, entity) pair its kernel-attested binding.
//
// Returns nil for non-git directories and empty histories; the
// consumer treats nil and an empty map identically (no
// exemptions).
func WalkAcknowledgedSHAEntities(ctx context.Context, root string) map[string]map[string]bool {
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
	acked := map[string]map[string]bool{}
	parts := strings.Split(string(out), "\x00")
	for i := 0; i+1 < len(parts); i += 2 {
		trailerBlock := parts[i+1]
		if trailerBlock == "" {
			continue
		}
		parsed := gitops.ParseTrailers(trailerBlock)
		var forceFor, entityID string
		for _, tr := range parsed {
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
