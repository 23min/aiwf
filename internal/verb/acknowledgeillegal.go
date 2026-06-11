package verb

import (
	"context"
	"errors"
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
// M-0136/AC-4 + G-0236: sha must resolve to a commit that is either
// reachable from HEAD (the primary AC-4 case) OR present in the local
// object database as an orphan (the G-0236 reflog-fallback case).
// The fallback supports acks against `isolation-escape-orphaned-ai-
// commit` findings, whose offending SHAs are by construction
// unreachable from HEAD (they're force-pushed-away tips surfaced via
// the reflog walker at internal/check/reflog_walk.go).
//
// Typo guard preserved: a SHA that resolves to no commit fails both
// checks and is rejected. The per-SHA closed-set scoping (each ack
// covers only the named SHA, and rules only fire on SHAs they
// independently enumerated) bounds the silencing surface — accepting
// object-DB-present SHAs doesn't widen any rule's reach.
//
// Returns a Result with a Plan carrying the empty commit's trailers.
// The Apply pipeline materializes the `git commit --allow-empty` once
// the human gate clears.
func AcknowledgeIllegal(ctx context.Context, root, sha, forEntity, actor, reason string) (*Result, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("aiwf acknowledge-illegal: --reason is required (non-empty after trim)")
	}
	if !strings.HasPrefix(actor, "human/") {
		return nil, fmt.Errorf("aiwf acknowledge-illegal: --actor must be human/<name> (got %q; sovereign acts trace to a named human)", actor)
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
			return nil, fmt.Errorf("aiwf acknowledge-illegal: %w", err)
		}
	}
	if err := shaAckable(ctx, root, sha); err != nil {
		return nil, fmt.Errorf("aiwf acknowledge-illegal: %w", err)
	}
	if cleanedEntity != "" {
		if err := verifySHATouchesEntity(ctx, root, sha, cleanedEntity); err != nil {
			return nil, fmt.Errorf("aiwf acknowledge-illegal: %w", err)
		}
	}
	short := sha
	if len(short) > 8 {
		short = short[:8]
	}
	return plan(&Plan{
		Subject:    fmt.Sprintf("aiwf acknowledge-illegal %s", short),
		Body:       cleanedReason,
		Trailers:   trailers,
		AllowEmpty: true,
	}), nil
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

// shaAckable verifies the SHA is a valid acknowledge-illegal target.
// Two acceptance paths:
//
//  1. Reachable from HEAD via `git merge-base --is-ancestor <sha> HEAD`
//     (M-0136/AC-4 primary case — covers the FSM-history rules and
//     isolation-escape proper, whose offending SHAs live on trunk).
//
//  2. Present in the local object database via `git rev-parse
//     --verify <sha>^{commit}` (G-0236 fallback — covers the
//     isolation-escape-orphaned-ai-commit rule, whose offending SHAs
//     are by construction unreachable from HEAD because they're
//     force-pushed-away tips the reflog walker found).
//
// Returns nil on either path. Returns a typed error on neither —
// the SHA exists in no usable form, which catches the typo /
// copy-paste / wrong-repo failure modes the original M-0136/AC-4
// reachability check was designed to refuse.
//
// Wrapping any subprocess failures preserves the operator-facing
// signal that something IO-shaped is wrong (permissions, missing
// git binary) vs. the policy refusal.
func shaAckable(ctx context.Context, root, sha string) error {
	// Primary: HEAD-reachable. Cheapest check, covers the M-0136
	// case directly.
	reachCmd := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", sha, "HEAD")
	reachCmd.Dir = root
	if reachCmd.Run() == nil {
		return nil
	}
	// Fallback: SHA exists in object DB but isn't HEAD-reachable.
	// The G-0236 orphan case. `^{commit}` peels through tags and
	// rejects non-commit objects (trees, blobs) so an ack against
	// a blob SHA would still refuse.
	verifyCmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", sha+"^{commit}")
	verifyCmd.Dir = root
	err := verifyCmd.Run()
	if err == nil {
		return nil
	}
	// Surface unexpected subprocess failures distinctly from
	// the policy refusal so an operator can tell "git is
	// broken" from "your SHA isn't ackable."
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return fmt.Errorf("checking object DB for %q: %w", sha, err)
	}
	return fmt.Errorf("SHA %q is neither reachable from HEAD nor present in the local object database (typo? wrong repo? object pruned?)", sha)
}
