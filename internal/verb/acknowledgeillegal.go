package verb

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// AcknowledgeIllegal records a retroactive sovereign override for a
// historical commit that one of the kernel's audit rules flags. The
// acknowledgment lives as a current-day empty commit carrying:
//
//	aiwf-verb: acknowledge-illegal
//	aiwf-force-for: <sha>
//	aiwf-actor: human/<name>
//	aiwf-reason: <free-form text>
//	aiwf-entity: <id>       (only when forEntity is non-empty)
//
// The fsm-history-consistent rule (M-0136/AC-2) walks HEAD's reachable
// history for `aiwf-force-for` trailers and exempts illegal-transition
// findings whose offending commit appears as a target. Six other rules
// consume the same SHA set via the M-0159/AC-3 lift.
//
// G-0231 item 3: a SEVENTH consumer rule —
// `provenance-untrailered-entity-commit` — was added with TIGHTER
// scope. Its findings are per-(commit, entity) pairs, so it requires
// per-(SHA, entity) acks: the ack commit must carry BOTH `aiwf-force-
// for: <sha>` AND `aiwf-entity: <id>`, and the verb verifies at write
// time that <sha>'s diff actually touches <id>'s file. The
// kernel-integrity property this adds: even if the operator (human or
// LLM) writes the wrong entity id with the right SHA, the verb
// refuses before the ack lands. SHA existence was already verified
// pre-G-0231 by shaAckable; entity binding is the new check.
//
// Constraints (M-0136/AC-1, extended by G-0231 item 3):
//   - reason must be non-empty after trim (sovereign acts require a
//     written rationale).
//   - actor must be `human/...` (sovereign acts trace to a named
//     human; no LLM / bot ack).
//   - sha must match the 7-40-hex SHA pattern (the trailer's value
//     constraint, enforced via gitops.ValidateTrailer).
//   - forEntity is OPTIONAL. When empty, the ack is per-SHA blanket
//     (the legacy shape covering the first six rules). When set, the
//     verb verifies <sha>'s diff touches <id>'s file and emits
//     `aiwf-entity: <id>` in the ack commit (the per-(SHA, entity)
//     shape required by provenance-untrailered-entity-commit).
//
// M-0136/AC-4 + G-0236: sha must resolve to a real commit in the
// local object database (see shaAckable) — covering both a SHA still
// on trunk (the AC-4 case) and an orphan SHA reachable only via
// reflog (the G-0236 case: `isolation-escape-orphaned-ai-commit`
// findings' offending SHAs are by construction unreachable from HEAD,
// force-pushed-away tips surfaced via the reflog walker at
// internal/check/reflog_walk.go).
//
// Typo guard preserved: a SHA that resolves to no commit is rejected.
// The per-SHA closed-set scoping (each ack covers only the named SHA,
// and rules only fire on SHAs they independently enumerated) bounds
// the silencing surface — accepting any object-DB-present SHA doesn't
// widen any rule's reach.
//
// Returns a Result with a Plan carrying the empty commit's trailers.
// The Apply pipeline materializes the `git commit --allow-empty` once
// the human gate clears.
func AcknowledgeIllegal(ctx context.Context, root, sha, forEntity, actor, reason string) (*Result, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("aiwf acknowledge illegal: --reason is required (non-empty after trim)")
	}
	if !strings.HasPrefix(actor, "human/") {
		return nil, fmt.Errorf("aiwf acknowledge illegal: --actor must be human/<name> (got %q; sovereign acts trace to a named human)", actor)
	}
	cleanedReason := strings.TrimSpace(reason)
	cleanedEntity := strings.TrimSpace(forEntity)
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "acknowledge-illegal"},
		{Key: gitops.TrailerForceFor, Value: sha},
		{Key: gitops.TrailerActor, Value: actor},
		{Key: gitops.TrailerReason, Value: cleanedReason},
	}
	if cleanedEntity != "" {
		trailers = append(trailers, gitops.Trailer{
			Key:   gitops.TrailerEntity,
			Value: entity.Canonicalize(cleanedEntity),
		})
	}
	for _, tr := range trailers {
		if err := gitops.ValidateTrailer(tr.Key, tr.Value); err != nil {
			return nil, fmt.Errorf("aiwf acknowledge illegal: %w", err)
		}
	}
	if err := shaAckable(ctx, root, sha); err != nil {
		return nil, fmt.Errorf("aiwf acknowledge illegal: %w", err)
	}
	if cleanedEntity != "" {
		if err := verifySHATouchesEntity(ctx, root, sha, cleanedEntity); err != nil {
			return nil, fmt.Errorf("aiwf acknowledge illegal: %w", err)
		}
	}
	short := sha
	if len(short) > 8 {
		short = short[:8]
	}
	result := plan(&Plan{
		Subject:    fmt.Sprintf("aiwf acknowledge illegal %s", short),
		Body:       cleanedReason,
		Trailers:   trailers,
		AllowEmpty: true,
	})
	result.Metadata = map[string]any{"sha": sha}
	return result, nil
}

// verifySHATouchesEntity runs `git diff-tree --no-commit-id
// --name-only -r --root <sha>` and walks the diff's path list.
// Returns nil when one of the paths resolves to an entity id (via
// PathKind + IDFromPath) whose canonical form matches the canonical
// form of forEntity. Returns a typed error when no path resolves to
// that entity (i.e., the SHA didn't touch the claimed entity — the
// LLM-invented-binding case G-0231 item 3 is built to refuse).
//
// `--root` is load-bearing: without it, root commits (no parent)
// produce an empty diff and the verification would refuse acks
// against the very first commit (which often introduces entity
// files in a fresh repo). Merge commits' first-parent diff is the
// same view RunUntrailedAudit audits, so the verification is
// consistent with what the rule sees.
func verifySHATouchesEntity(ctx context.Context, root, sha, forEntity string) error {
	cmd := exec.CommandContext(ctx, "git", "diff-tree", "--no-commit-id", "--name-only", "-r", "--root", sha)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("diff-tree for %s: %w", sha, err)
	}
	// Roll a composite forEntity (M-NNN/AC-N) up to its parent before
	// comparing: the diff-walking side below resolves a touched
	// milestone path to its bare parent id (M-NNN via IDFromPath), so
	// the comparison must be parent-against-parent or every per-AC ack
	// misses (G-0237). The emitted aiwf-entity trailer keeps the full
	// composite — only this touches-the-file check rolls up.
	want := entity.Canonicalize(entity.CompositeRoot(forEntity))
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		kind, ok := entity.PathKind(path)
		if !ok {
			continue
		}
		id, ok := entity.IDFromPath(path, kind)
		if !ok {
			continue
		}
		if entity.Canonicalize(id) == want {
			return nil
		}
	}
	return fmt.Errorf("SHA %s does not touch entity %s (its diff names no file resolving to %s; refusing operator-attested binding without mechanical evidence — G-0231 item 3)",
		sha, want, want)
}

// shaAckable verifies sha resolves to a real commit, via the shared
// gitops.CommitExists primitive (F3) instead of a hand-rolled
// exec.Command("git", "rev-parse", "--verify", ...) call.
// gitops.CommitExists's "^{commit}" peels through tags and rejects
// non-commit objects (trees, blobs), so an ack against a blob SHA
// still refuses.
//
// Existence — not HEAD-reachability — is the actual acceptance
// criterion: the isolation-escape-orphaned-ai-commit rule's offending
// SHAs are by construction unreachable from HEAD (G-0236 — they're
// force-pushed-away tips the reflog walker found), so a
// reachable-from-HEAD check would wrongly refuse exactly the SHAs
// that rule needs acked. Reachability implies existence for every SHA
// git can compute ancestry against, so gating on reachability instead
// of (or in addition to) existence can only ever narrow acceptance in
// the wrong direction — never add a legitimate discrimination
// existence alone misses.
//
// Returns a typed error when sha resolves to no commit, catching the
// typo / copy-paste / wrong-repo failure modes the original M-0136/AC-4
// check was designed to refuse.
func shaAckable(ctx context.Context, root, sha string) error {
	exists, err := gitops.CommitExists(ctx, root, sha)
	if err != nil {
		//coverage:ignore defensive: CommitExists maps an unresolvable sha to (false,nil); a non-nil err needs git absent or a broken workdir, not reachable deterministically in-process (mirrors promote.go's validateAddressedByCommit)
		return fmt.Errorf("checking object DB for %q: %w", sha, err)
	}
	if !exists {
		return fmt.Errorf("SHA %q does not resolve to a commit in the local object database (typo? wrong repo? object pruned?)", sha)
	}
	return nil
}
