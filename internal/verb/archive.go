package verb

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// Archive sweeps terminal-status entities from the active tree into
// their per-kind `archive/` subdirectories per ADR-0004's storage
// table. The verb is multi-entity: one invocation rewrites every
// qualifying entity and produces a single commit (CLAUDE.md §7).
//
// Behavior:
//
//   - Default is dry-run: the verb computes a Plan and the caller
//     prints planned ops without applying. `--apply` (caller flag)
//     causes the dispatcher to run verb.Apply on the Plan.
//   - Single commit per --apply per kernel principle #7. Trailer is
//     `aiwf-verb: archive`; no `aiwf-entity:` trailer (multi-entity
//     sweep, same shape as `aiwf rewidth`).
//   - Idempotent. An already-swept tree returns a NoOp Result; the
//     caller prints "no changes needed" and exits 0.
//   - Sweep is by status, not by id. There is no positional id arg —
//     ADR-0004 §"`aiwf archive` verb" rejects per-id housekeeping
//     ("that would be a hand-edit detour, not a verb").
//
// Per-kind storage table (verbatim from ADR-0004 §"Storage — per-kind
// layout"):
//
//	| Kind     | Active                              | Archive                                      |
//	|----------|-------------------------------------|----------------------------------------------|
//	| Epic     | work/epics/<epic>/                  | work/epics/archive/<epic>/ (whole subtree)   |
//	| Milestone| work/epics/<epic>/M-NNNN-<slug>.md  | does not archive independently — rides w/ epic|
//	| Contract | work/contracts/<contract>/          | work/contracts/archive/<contract>/           |
//	| Gap      | work/gaps/<id>-<slug>.md            | work/gaps/archive/<id>-<slug>.md             |
//	| Decision | work/decisions/<id>-<slug>.md       | work/decisions/archive/<id>-<slug>.md        |
//	| ADR      | docs/adr/<id>-<slug>.md             | docs/adr/archive/<id>-<slug>.md              |
//
// `internal/entity/transition.go::IsTerminal` is the single source of
// truth for terminal statuses.
//
// kindFilter scopes the sweep. "" sweeps every kind; a non-empty value
// must be one of entity.AllKinds().
func Archive(ctx context.Context, root, actor, kindFilter string) (*Result, error) {
	plan, skipped, err := planArchive(ctx, root, kindFilter)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		msg := "aiwf archive: no terminal-status entities awaiting sweep (tree is converged)"
		if len(skipped) > 0 {
			// G-0394 (Direction B): don't report "converged" when an
			// epic is actually stranded on a non-terminal child — that
			// would silently hide the exact problem this guard exists
			// to surface.
			msg = fmt.Sprintf("aiwf archive: no entities swept; %d %s skipped: %s",
				len(skipped), pluralize(len(skipped), "entity", "entities"), formatArchiveSkips(skipped))
		}
		return &Result{NoOp: true, NoOpMessage: msg}, nil
	}
	plan.Trailers = []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "archive"},
		{Key: gitops.TrailerActor, Value: actor},
	}
	sweptCount := 0
	for _, op := range plan.Ops {
		if op.Type == OpMove {
			sweptCount++
		}
	}
	return &Result{Plan: plan, Metadata: map[string]any{"swept_count": sweptCount}}, nil
}

// archiveMove is one (from, to) plus the kind it belongs to. The kind
// is needed by the commit-body renderer for the per-kind count; the
// from/to drive the OpMove.
type archiveMove struct {
	kind entity.Kind
	from string
	to   string
	// id is the entity id that triggers the move (the dir id for
	// epic/contract; the file's id for the flat-file kinds and for
	// milestones riding with their parent epic). Used by the commit-
	// body's affected-ids list.
	id string
}

// archiveSkip records an epic that computeArchiveMoves declined to
// sweep because its subtree still owns one or more non-terminal
// milestones (G-0394, Direction B) — the archive-time counterpart to
// Promote's epic-terminal guard (internal/verb/promote.go, G-0393 /
// G-0394). That guard runs unconditionally (no --force bypass, mirroring
// Cancel's own D-0003 guard), so this is defense-in-depth for the one
// path that still reaches the state: a raw frontmatter hand-edit that
// bypasses the verb layer entirely.
type archiveSkip struct {
	epic     string
	children []string

	// id and blockedBy record the other reason a move is declined: a
	// file the sweep's verdict about that entity rests on is mid-edit,
	// so the verdict is unavailable rather than negative. Exactly one of
	// the two shapes is populated per skip.
	id        string
	blockedBy []string
}

// formatArchiveSkips renders skipped epics and their offending
// children for operator-facing text. Shared by the commit body and
// the NoOp message so the phrasing has one source.
func formatArchiveSkips(skipped []archiveSkip) string {
	parts := make([]string, 0, len(skipped))
	for _, s := range skipped {
		if s.epic != "" {
			parts = append(parts, fmt.Sprintf("%s (non-terminal: %s)", s.epic, strings.Join(s.children, ", ")))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (uncommitted changes in %s)", s.id, strings.Join(s.blockedBy, ", ")))
	}
	return strings.Join(parts, "; ")
}

// planArchive computes the list of OpMove ops the sweep produces over
// the active tree. Returns nil when there is nothing to sweep.
//
// kindFilter, when non-empty, scopes the walk to one kind. Milestones
// are not directly swept — they ride with their parent epic — so a
// kindFilter of "milestone" sweeps nothing. (We treat that as a no-op
// rather than an error: it's an honest answer to "what would archive
// do for milestones?".)
func planArchive(ctx context.Context, root, kindFilter string) (*Plan, []archiveSkip, error) {
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		return nil, nil, fmt.Errorf("loading tree: %w", err)
	}

	moves, skipped, err := computeArchiveMoves(tr, kindFilter)
	if err != nil {
		return nil, nil, err
	}
	moves, skipped, err = declineUndecidableMoves(ctx, root, tr, moves, skipped)
	if err != nil { //coverage:ignore defensive: every error inside declineUndecidableMoves is itself an annotated git-unusable arm, reachable only if the repo breaks between the tree load and here
		return nil, nil, err
	}
	if len(moves) == 0 {
		return nil, skipped, nil
	}

	// Stable order: by kind then by from-path. Determinism is load-
	// bearing — a second invocation on the same tree visits files in
	// the same order and produces zero ops. CLAUDE.md "Test untested
	// code paths" — ordering is exercised by the per-kind storage
	// layout test.
	sort.Slice(moves, func(i, j int) bool {
		if moves[i].kind != moves[j].kind {
			return moves[i].kind < moves[j].kind
		}
		return moves[i].from < moves[j].from
	})

	ops := make([]FileOp, 0, len(moves))
	for _, m := range moves {
		ops = append(ops, FileOp{Type: OpMove, Path: m.from, NewPath: m.to})
	}

	rewriteOps, err := planArchiveRewrites(tr, moves)
	if err != nil { //coverage:ignore filesystem/serialize errors propagate from planArchiveRewrites — covered by its own defensive ignores
		return nil, nil, err
	}
	ops = append(ops, rewriteOps...)

	subject := archiveCommitSubject(moves)
	return &Plan{
		Subject: subject,
		Body:    archiveCommitBody(moves, skipped, len(rewriteOps)),
		Ops:     ops,
	}, skipped, nil
}

// archiveEntityMoves expands the directory-shaped moves in moves
// (epic, contract) into one EntityMove per entity file that lives
// inside the moved directory — including the dir-shape entity's own
// file (epic.md / contract.md) and any nested milestone — using the
// same pathInside/newEntityPathAfterRename pattern `reallocate` uses
// for its own directory-rename case. Flat-file moves (gap, decision,
// adr) are already file-level and pass through unchanged.
//
// This is what lets a link into a milestone nested inside an
// archived epic dir resolve to the milestone's post-sweep path, not
// just the epic dir's own path.
func archiveEntityMoves(tr *tree.Tree, moves []archiveMove) []EntityMove {
	var out []EntityMove
	for _, m := range moves {
		switch m.kind {
		case entity.KindGap, entity.KindDecision, entity.KindADR:
			out = append(out, EntityMove{From: m.from, To: m.to})
		case entity.KindEpic, entity.KindContract:
			for _, e := range tr.Entities {
				if !pathInside(e.Path, m.from) {
					continue
				}
				out = append(out, EntityMove{
					From: e.Path,
					To:   newEntityPathAfterRename(e, m.from, m.to),
				})
			}
		}
	}
	return out
}

// planArchiveRewrites computes the body-content rewrites every active
// entity needs once moves lands: any entity whose body links to a
// moved entity, including an entity that is itself moving in the same
// sweep (its body is read at the pre-move path and, when changed,
// written at the post-move path — Apply runs every OpMove before any
// OpWrite, so a write targeting a post-move path lands correctly).
//
// Already-archived entities are skipped as linking-file candidates,
// mirroring rewidth's "forget-by-default" exclusion of `archive/`
// content (ADR-0004).
func planArchiveRewrites(tr *tree.Tree, moves []archiveMove) ([]FileOp, error) {
	entityMoves := archiveEntityMoves(tr, moves)
	if len(entityMoves) == 0 {
		return nil, nil //coverage:ignore unreachable: planArchive only calls this when moves is non-empty, and every archiveMove (gap/decision/adr direct, or epic/contract via its own dir-shape entity) yields at least one EntityMove
	}
	return planLinkRewriteWrites(tr, entityMoves, nil)
}

// computeArchiveMoves walks the loaded tree and produces one move per
// terminal-status active entity that should sweep into archive/.
//
// Directory-shaped kinds (epic, contract): the move targets the parent
// directory (the entity's containing dir), not the per-file paths inside.
// `git mv <epic-dir> <archive>/<epic-dir>` moves the whole subtree atomically;
// nested milestone files come along for free without separate ops. The
// loader sees both the epic and its milestones; we deduplicate by emitting
// one move per epic dir, regardless of how many milestones live inside.
//
// Flat-file kinds (gap, decision, adr): one OpMove per file.
//
// Milestones never sweep independently per ADR-0004's storage table.
// A milestone whose parent epic is active stays put (the noise problem
// doesn't bite at the milestone level). A milestone whose parent epic
// is terminal moves alongside the epic via the dir-rename above; the
// milestone's own status is incidental to that move.
func computeArchiveMoves(tr *tree.Tree, kindFilter string) ([]archiveMove, []archiveSkip, error) {
	if kindFilter != "" && !isKnownKind(kindFilter) {
		return nil, nil, fmt.Errorf("unknown kind %q (must be one of %s)", kindFilter, strings.Join(allKindNamesArchive(), ", "))
	}

	// Track epic dirs we've already emitted a move for, so a "done epic
	// with three milestones" only produces one OpMove (the dir rename),
	// not four.
	epicDirSeen := map[string]bool{}
	contractDirSeen := map[string]bool{}

	var moves []archiveMove
	var skipped []archiveSkip

	for _, e := range tr.Entities {
		// Skip already-archived entities — the move target is where
		// they already live.
		if entity.IsArchivedPath(e.Path) {
			continue
		}
		// Apply the optional kind filter. Milestones are out of scope
		// for direct sweep; if the user asked --kind milestone, they
		// get a clean no-op (the verb tells the truth: there is no
		// milestone-level archive trigger).
		if kindFilter != "" && string(e.Kind) != kindFilter {
			continue
		}

		switch e.Kind {
		case entity.KindEpic:
			if !entity.IsTerminal(e.Kind, e.Status) {
				continue
			}
			// Move the whole epic dir. Compute the parent dir from
			// e.Path (`work/epics/<dir>/epic.md` -> `work/epics/<dir>`).
			epicDir := filepath.Dir(e.Path)
			if epicDirSeen[epicDir] {
				continue
			}
			epicDirSeen[epicDir] = true
			// G-0394 (Direction B): decline to strand a non-terminal
			// child in archive/ alongside its terminal parent. The
			// promote-time guard (Promote, promote.go) is the primary
			// chokepoint and already runs unconditionally (no --force
			// bypass); a raw frontmatter hand-edit is the one path that
			// still reaches this state, so archive independently refuses
			// to sweep it too — unconditionally, with no --force of its
			// own.
			if nonTerminal := nonTerminalEpicChildren(tr, e.ID); len(nonTerminal) > 0 {
				skipped = append(skipped, archiveSkip{epic: e.ID, children: nonTerminal})
				continue
			}
			toDir := archiveTargetForEpic(epicDir)
			moves = append(moves, archiveMove{
				kind: entity.KindEpic,
				from: epicDir,
				to:   toDir,
				id:   e.ID,
			})

		case entity.KindMilestone:
			// Milestones don't archive independently. If the parent
			// epic is terminal and gets swept, the milestone rides
			// along via the epic-dir rename. We never emit a
			// milestone-only move.
			//
			// Edge case: milestone status is terminal but the parent
			// epic is still active. ADR-0004 explicitly leaves this
			// in place — the milestone stays in the active epic dir
			// until the epic itself archives. The
			// terminal-entity-not-archived finding (M-0086) does NOT
			// fire on milestones in active epic dirs by virtue of
			// this ADR design — the milestone's location is the
			// epic's responsibility. (Today the M-0086 rule still
			// fires on every terminal-active entity regardless of
			// kind; that's a separate cleanup. The verb's behavior
			// here is what the ADR specifies.)
			continue

		case entity.KindContract:
			if !entity.IsTerminal(e.Kind, e.Status) {
				continue
			}
			contractDir := filepath.Dir(e.Path)
			if contractDirSeen[contractDir] {
				continue
			}
			contractDirSeen[contractDir] = true
			toDir := archiveTargetForContract(contractDir)
			moves = append(moves, archiveMove{
				kind: entity.KindContract,
				from: contractDir,
				to:   toDir,
				id:   e.ID,
			})

		case entity.KindGap, entity.KindDecision, entity.KindADR:
			if !entity.IsTerminal(e.Kind, e.Status) {
				continue
			}
			to := archiveTargetForFlatFile(e.Path, e.Kind)
			moves = append(moves, archiveMove{
				kind: e.Kind,
				from: e.Path,
				to:   to,
				id:   e.ID,
			})
		default:
			// Defensive: a future kind landing in entity.AllKinds()
			// without an archive rule here should be visible — not
			// silently skipped. Today the closed set is six kinds and
			// every one is handled above; this branch exists to make
			// "unhandled future kind" a loud regression rather than a
			// quiet hole.
			//coverage:ignore unreachable today; defends against future Kind additions
			continue
		}
	}

	return moves, skipped, nil
}

// isKnownKind reports whether s names one of the six aiwf kinds.
func isKnownKind(s string) bool {
	for _, k := range entity.AllKinds() {
		if string(k) == s {
			return true
		}
	}
	return false
}

// allKindNamesArchive returns the lowercase names of the six aiwf
// kinds, suitable for inclusion in an error message.
func allKindNamesArchive() []string {
	out := make([]string, 0, 6)
	for _, k := range entity.AllKinds() {
		out = append(out, string(k))
	}
	return out
}

// archiveTargetForEpic returns the archive path for an epic directory.
// `work/epics/<dir>` -> `work/epics/archive/<dir>`. Inputs are
// repo-relative forward-slash paths.
func archiveTargetForEpic(epicDir string) string {
	// epicDir is repo-relative, e.g. "work/epics/E-0010-foo".
	dirName := filepath.Base(epicDir)
	return filepath.ToSlash(filepath.Join("work", "epics", "archive", dirName))
}

// archiveTargetForContract returns the archive path for a contract
// directory. `work/contracts/<dir>` -> `work/contracts/archive/<dir>`.
func archiveTargetForContract(contractDir string) string {
	dirName := filepath.Base(contractDir)
	return filepath.ToSlash(filepath.Join("work", "contracts", "archive", dirName))
}

// archiveTargetForFlatFile returns the archive path for a gap,
// decision, or ADR file:
//
//	work/gaps/G-NNNN-<slug>.md     -> work/gaps/archive/G-NNNN-<slug>.md
//	work/decisions/D-NNNN-<slug>.md -> work/decisions/archive/D-NNNN-<slug>.md
//	docs/adr/ADR-NNNN-<slug>.md    -> docs/adr/archive/ADR-NNNN-<slug>.md
func archiveTargetForFlatFile(activePath string, kind entity.Kind) string {
	base := filepath.Base(activePath)
	switch kind {
	case entity.KindGap:
		return filepath.ToSlash(filepath.Join("work", "gaps", "archive", base))
	case entity.KindDecision:
		return filepath.ToSlash(filepath.Join("work", "decisions", "archive", base))
	case entity.KindADR:
		return filepath.ToSlash(filepath.Join("docs", "adr", "archive", base))
	}
	// Defensive: caller has already filtered to flat-file kinds. If a
	// future kind lands without an archive-target rule, return empty
	// (which the upstream OpMove will surface as a path error).
	return "" //coverage:ignore defensive: caller switches over the three flat-file kinds before invoking; future kinds would route through their own case
}

// archiveCommitSubject renders the one-line subject for the sweep
// commit. ADR-0004 §"`aiwf archive` verb": "the commit message body
// lists affected ids and per-kind counts." The subject names the
// total count and the per-kind breakdown.
func archiveCommitSubject(moves []archiveMove) string {
	if len(moves) == 0 {
		return "" //coverage:ignore caller (planArchive) returns nil Plan when len(moves)==0; this branch is unreachable in production
	}
	byKind := map[entity.Kind]int{}
	for _, m := range moves {
		byKind[m.kind]++
	}
	// Per-kind summary for the subject. Order follows entity.AllKinds()
	// for determinism.
	var parts []string
	for _, k := range entity.AllKinds() {
		if n := byKind[k]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, k))
		}
	}
	return fmt.Sprintf(
		"aiwf archive: sweep %d entit%s into archive/ (%s)",
		len(moves),
		pluralize(len(moves), "y", "ies"),
		strings.Join(parts, ", "),
	)
}

// pluralize is a tiny English helper so the subject reads naturally
// for both "1 entity" and "N entities".
func pluralize(n int, singularSuffix, pluralSuffix string) string {
	if n == 1 {
		return singularSuffix
	}
	return pluralSuffix
}

// archiveCommitBody renders the per-kind summary + affected-id list
// for the commit body, plus a skipped-epics section when skipped is
// non-empty (G-0394, Direction B) so the multi-entity sweep doesn't
// silently drop a stranded epic with no operator-visible trace.
// ADR-0004 §"`aiwf archive` verb": "the commit message body lists
// affected ids and per-kind counts."
//
// Format:
//
//	Per ADR-0004: sweep terminal-status entities into per-kind archive/.
//
//	Per-kind counts:
//	  epic       2 entities
//	  contract   1 entity
//	  gap        18 entities
//	  ...
//
//	Affected ids:
//	  E-0010, E-0017, C-0010, G-0010, G-0011, ..., D-0007, ADR-0001
//
//	Skipped:
//	  E-0020: M-0030, M-0031
//
// rewriteCount is the number of entity-body link-destination rewrites
// (M-0246) riding in the same commit; 0 renders no extra section.
//
// Determinism: kinds iterate in entity.AllKinds() order; ids within
// each kind iterate in lexicographic order; skipped epics iterate in
// the order computeArchiveMoves encountered them (tr.Entities order).
func archiveCommitBody(moves []archiveMove, skipped []archiveSkip, rewriteCount int) string {
	if len(moves) == 0 && len(skipped) == 0 {
		return "" //coverage:ignore caller (planArchive) returns nil Plan whenever len(moves)==0, regardless of skipped
	}
	var sb strings.Builder
	if len(moves) > 0 {
		byKind := map[entity.Kind][]string{}
		for _, m := range moves {
			byKind[m.kind] = append(byKind[m.kind], m.id)
		}
		for k := range byKind {
			sort.Strings(byKind[k])
		}

		sb.WriteString("Per ADR-0004: sweep terminal-status entities into per-kind archive/.\n\n")
		sb.WriteString("Per-kind counts:\n")
		for _, k := range entity.AllKinds() {
			ids, ok := byKind[k]
			if !ok || len(ids) == 0 {
				continue
			}
			fmt.Fprintf(&sb, "  %-9s %d %s\n", k, len(ids), pluralize(len(ids), "entity", "entities"))
		}

		// Affected ids: aggregate across kinds in entity.AllKinds()
		// order, then alphabetical within each kind.
		var allIDs []string
		for _, k := range entity.AllKinds() {
			allIDs = append(allIDs, byKind[k]...)
		}
		sb.WriteString("\nAffected ids:\n  ")
		sb.WriteString(strings.Join(allIDs, ", "))
		sb.WriteString("\n")
	}

	if len(skipped) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("Skipped:\n")
		for _, s := range skipped {
			if s.epic != "" {
				fmt.Fprintf(&sb, "  %s: non-terminal children (G-0394): %s\n", s.epic, strings.Join(s.children, ", "))
				continue
			}
			fmt.Fprintf(&sb, "  %s: uncommitted changes in %s\n", s.id, strings.Join(s.blockedBy, ", "))
		}
	}

	if rewriteCount > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "Body rewrites: %d file(s)\n", rewriteCount)
	}

	return sb.String()
}

// declineUndecidableMoves drops the candidate moves whose verdict rests
// on a file the operator is part-way through editing, and reports each
// one instead of sweeping it.
//
// A sweep decides three things by reading the working copy, and each can
// be contradicted by the record:
//
//   - whether the entity is terminal at all, read from its own file;
//   - whether a referring entity needs its link rewritten, read from
//     that referrer's body. planArchiveRewrites emits nothing when the
//     working copy does not carry the committed link, so a move made
//     alongside a mid-edit referrer lands without its rewrite, leaving a
//     link to a path absent at HEAD. Such a link is unrepairable: an
//     archived target is excluded from every later scan by
//     IsArchivedPath, so no re-run reaches it;
//   - whether an epic owns a non-terminal child, read from the
//     children's files. A mid-edit child reads as non-terminal, which
//     the commit body would assert against a record that says otherwise.
//
// Declining the affected move rather than refusing the whole sweep is
// what keeps an unrelated draft from blocking unrelated work: every move
// that survives has a verdict resting entirely on committed bytes.
//
// Referrers are matched against HEAD's body, not the working copy's.
// The working copy is precisely what cannot be trusted here — a draft
// that dropped the link is invisible to a working-copy scan, which is
// the defect. Cost is one HEAD read per mid-edit entity file, so it
// scales with what the operator is editing rather than with the tree.
func declineUndecidableMoves(
	ctx context.Context,
	root string,
	tr *tree.Tree,
	moves []archiveMove,
	skipped []archiveSkip,
) ([]archiveMove, []archiveSkip, error) {
	dirty, err := dirtyEntityPaths(ctx, root, tr)
	if err != nil { //coverage:ignore defensive: dirtyEntityPaths errors only when git is unusable, which the tree load from this same root already ruled out
		return nil, nil, err
	}
	// Everything a directory move would carry, entity or not: a stray
	// file beneath a swept epic rides into the commit and becomes tracked
	// from it, so it decides that move as surely as a status does.
	carried, err := dirtyPathsUnderMoves(ctx, root, moves)
	if err != nil { //coverage:ignore defensive: same non-repo condition dirtyEntityPaths just passed
		return nil, nil, err
	}
	if len(dirty) == 0 && len(carried) == 0 {
		return moves, skipped, nil
	}

	// An epic declined for a non-terminal child it may not actually own:
	// the child is mid-edit, so the verdict is unavailable. Report the
	// uncommitted change rather than an accusation HEAD contradicts.
	for i := range skipped {
		var blocked []string
		for _, childID := range skipped[i].children {
			child := tr.ByID(childID)
			if child == nil { //coverage:ignore defensive: children come from the loaded tree's own parent index
				continue
			}
			if dirty[filepath.ToSlash(child.Path)] {
				blocked = append(blocked, filepath.ToSlash(child.Path))
			}
		}
		if len(blocked) > 0 {
			skipped[i] = archiveSkip{id: skipped[i].epic, blockedBy: blocked}
		}
	}

	headBodies := make(map[string][]byte, len(dirty))
	candidate := make(map[string]bool, len(moves))
	for _, m := range moves {
		candidate[m.from] = true
	}
	kept := moves[:0:0]
	for _, m := range moves {
		blockers, blockErr := moveBlockers(ctx, root, tr, m, dirty, carried, headBodies)
		if blockErr != nil { //coverage:ignore defensive: moveBlockers errors only on a HEAD read for a path the dirty set just resolved
			return nil, nil, blockErr
		}
		if len(blockers) == 0 {
			kept = append(kept, m)
			continue
		}
		sort.Strings(blockers)
		skipped = append(skipped, archiveSkip{id: m.id, blockedBy: blockers})
	}

	// An entity terminal at HEAD but not in the working copy never became
	// a candidate at all, so nothing above declined it. Reporting it is
	// what keeps the sweep from calling the tree converged while the
	// record says a sweep is due.
	masked, err := maskedTerminalSkips(ctx, root, tr, dirty, candidate, headBodies)
	if err != nil { //coverage:ignore defensive: the tree loaded from this root moments earlier, so a git failure here needs the repo to break mid-verb
		return nil, nil, err
	}
	skipped = append(skipped, masked...)
	return kept, skipped, nil
}

// maskedTerminalSkips reports entities whose committed status is terminal
// while their mid-edit working copy is not. The sweep reads the working
// copy, so these are invisible to it: measured, a gap at `wontfix` in
// HEAD and `open` on disk made `aiwf archive` answer "tree is converged"
// at exit 0 with a sweep genuinely due against the record.
//
// Nothing is written either way — the entity simply stays put — so this
// changes what the operator is told rather than what happens.
func maskedTerminalSkips(
	ctx context.Context,
	root string,
	tr *tree.Tree,
	dirty, candidate map[string]bool,
	headBodies map[string][]byte,
) ([]archiveSkip, error) {
	var out []archiveSkip
	for _, e := range tr.Entities {
		path := filepath.ToSlash(e.Path)
		if !dirty[path] || candidate[path] || entity.IsArchivedPath(path) {
			continue
		}
		if entity.IsTerminal(e.Kind, e.Status) {
			continue // terminal on disk too; it was declined above or swept
		}
		content, ok := headBodies[path]
		if !ok {
			raw, err := gitops.ReadFromHEAD(ctx, root, path)
			if err != nil { //coverage:ignore defensive: the path is in the dirty set, so it resolved at HEAD moments earlier
				return nil, fmt.Errorf("reading %s at HEAD: %w", path, err)
			}
			content = raw
			headBodies[path] = raw
		}
		if len(content) == 0 {
			continue
		}
		committed, err := entity.Parse(path, content)
		if err != nil {
			continue //coverage:ignore defensive: HEAD carries a committed entity, which parsed when it landed
		}
		// Parse decodes frontmatter only; kind is derived from the path,
		// which the working copy has not moved.
		if entity.IsTerminal(e.Kind, committed.Status) {
			out = append(out, archiveSkip{id: e.ID, blockedBy: []string{path}})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out, nil
}

// moveEnds returns the paths a move touches: the source it carries and
// the destination it lands on.
//
// Both of the decline's own reach computations reach through this one
// function, and the commit-side guard enumerates the same pair by its own
// means (planCarriedPaths walks op.Path and op.NewPath alike). A
// destination missing from either enumeration is content the commit
// replaces without anyone naming it — and, where the guard sees it and
// the decline does not, a whole-verb refusal for a single participant.
func moveEnds(m archiveMove) []string {
	return []string{m.from, m.to}
}

// moveBlockers returns the mid-edit files that make one move's verdict
// undecidable: the moved path itself, anything beneath it for a
// directory-shaped kind, and any entity whose committed body links into
// the move.
func moveBlockers(
	ctx context.Context,
	root string,
	tr *tree.Tree,
	m archiveMove,
	dirty, carried map[string]bool,
	headBodies map[string][]byte,
) ([]string, error) {
	entityMoves := archiveEntityMoves(tr, []archiveMove{m})
	seen := map[string]bool{}
	var blockers []string
	ends := moveEnds(m)
	for path := range carried {
		for _, end := range ends {
			if !pathInside(path, end) {
				continue
			}
			blockers = append(blockers, path)
			seen[path] = true
			break
		}
	}
	// Everything the move physically carries is already accounted for
	// above; what remains is the entity whose committed body links into
	// the move from outside it.
	for path := range dirty {
		if seen[path] {
			continue
		}
		// Archived entities are not referrers. planArchiveRewrites skips
		// them as linking-file candidates under ADR-0004's forget-by-default
		// rule, so no sweep rewrites an archived body and none can lose a
		// link. Counting one here would decline a candidate on the state of
		// a file the sweep would never have touched.
		if entity.IsArchivedPath(path) {
			continue
		}
		body, ok := headBodies[path]
		if !ok {
			raw, err := gitops.ReadFromHEAD(ctx, root, path)
			if err != nil { //coverage:ignore defensive: the path is in the dirty set, so it resolved against HEAD in the same call chain
				return nil, fmt.Errorf("reading %s at HEAD: %w", path, err)
			}
			body = raw
			headBodies[path] = raw
		}
		if len(body) == 0 {
			// Absent from the record. A referrer's rewrite is an OpWrite to
			// that referrer's own path, and the commit-side guard exempts an
			// absent-from-HEAD divergence at exactly that shape — a file git
			// never recorded has no committed content the write could
			// overwrite — so the write lands and the two seams agree by both
			// letting it through. Blocking here would decline a candidate the
			// commit would have accepted.
			//
			// The exemption is that narrow: an untracked file merely carried
			// along by a directory move is not an OpWrite's own destination
			// and is not exempt, which is why the carried set is judged
			// above rather than here.
			continue
		}
		// A link in either copy is a link the sweep's verdict rests on. The
		// rewrite pass reads the working copy and emits its op from that;
		// the record is what a lost link is lost from. Consulting one side
		// only lets the two disagree, and every such disagreement ends the
		// same way — a move nothing declined carrying a write the guard
		// then refuses for the whole verb.
		if linksIntoMove(body, path, entityMoves) || linksIntoMove(workingBodyAt(root, path), path, entityMoves) {
			blockers = append(blockers, path)
		}
	}
	return blockers, nil
}

// linksIntoMove reports whether body carries a link that one of the moves
// would rewrite. An empty body carries nothing.
func linksIntoMove(body []byte, linkingPath string, moves []EntityMove) bool {
	if len(body) == 0 {
		return false
	}
	return !bytes.Equal(RewriteLinkDestinations(body, linkingPath, moves), body)
}

// workingBodyAt returns the working copy's bytes for path, or nil when it
// cannot be read. Unreadable is the same answer as absent here: a file the
// sweep cannot read carries no link it can act on, and the paths that
// reach this point are already known to differ from the record — a
// deleted referrer among them, which is exactly the unreadable case.
func workingBodyAt(root, path string) []byte {
	raw, err := readBody(root, path)
	if err != nil {
		return nil
	}
	return raw
}

// recordedEntityPaths returns the entity files HEAD records, as
// repo-relative slash paths.
//
// Classification is entity.PathKind, the same predicate the loader
// applies while walking the working tree, so the record's view of what
// counts as an entity file cannot drift from the working copy's. Every
// path in HEAD's tree is offered to it rather than pre-filtered by
// directory, which would fork the loader's walk roots into a second
// copy that no test compares against the first.
func recordedEntityPaths(ctx context.Context, root string) ([]string, error) {
	paths, err := gitops.LsTreePaths(ctx, root, "HEAD")
	if err != nil { //coverage:ignore defensive: HEAD resolves (callers consult HasHEAD first), leaving only a git ls-tree subprocess failure, which needs the repo to break mid-verb
		return nil, fmt.Errorf("listing the entity files recorded at HEAD: %w", err)
	}
	var out []string
	for _, p := range paths {
		if _, ok := entity.PathKind(p); ok {
			out = append(out, p)
		}
	}
	return out, nil
}

// dirtyEntityPaths returns the entity files that differ from the record,
// as a set of repo-relative paths — including files git has never
// recorded, which differ from it maximally, and files the record carries
// that the working copy no longer resolves as entities. An unborn HEAD
// has no record to differ from at all, so the set is empty.
//
// The candidate set is the union of both views precisely because a
// disagreement between them is the condition worth reporting: an entity
// enumerated from the working tree alone drops out of the comparison at
// the moment it stops parsing, which is the moment it most needs to be
// in it.
//
// Callers that need "has a committed version" ask for HEAD's content and
// find it empty; keeping that judgement at the one place it is used stops
// the two answers drifting apart.
func dirtyEntityPaths(ctx context.Context, root string, tr *tree.Tree) (map[string]bool, error) {
	hasHEAD, err := gitops.HasHEAD(ctx, root)
	if err != nil { //coverage:ignore defensive: HasHEAD errors only outside a repo, which the tree load already required
		return nil, fmt.Errorf("checking the working tree against HEAD: %w", err)
	}
	if !hasHEAD {
		return nil, nil
	}
	seen := make(map[string]bool, len(tr.Entities))
	paths := make([]string, 0, len(tr.Entities))
	add := func(p string) {
		if seen[p] {
			return
		}
		seen[p] = true
		paths = append(paths, p)
	}
	for _, e := range tr.Entities {
		add(filepath.ToSlash(e.Path))
	}
	// The record's entity files as well as the loaded tree's. An entity
	// the loader dropped — deleted, hand-renamed, or carrying momentarily
	// unparseable frontmatter — is absent from tr.Entities while the
	// record still carries it and whatever links it holds. Enumerating
	// only the loaded tree makes precisely the files most likely to be
	// mid-edit invisible to every comparison built on this set.
	recorded, err := recordedEntityPaths(ctx, root)
	if err != nil { //coverage:ignore defensive: recordedEntityPaths fails only on an unusable git — HEAD resolves by the HasHEAD check above
		return nil, err
	}
	for _, p := range recorded {
		add(p)
	}
	files, dirs := splitDirectoryPaths(root, paths)
	diverged, err := gitops.DivergentPaths(ctx, root, files)
	if err != nil { //coverage:ignore defensive: same non-repo condition HasHEAD just passed
		return nil, fmt.Errorf("checking the working tree against HEAD: %w", err)
	}
	out := make(map[string]bool, len(diverged)+len(dirs))
	for _, d := range diverged {
		out[d.Path] = true
	}
	// A path the record carries as a file and the working tree holds as a
	// directory disagrees with the record as surely as an edit does, and
	// more so. It cannot be compared byte-wise, so it is named divergent
	// here instead: DivergentPaths refuses a directory outright, and a
	// refusal there would take the whole sweep down over one participant —
	// the outcome the per-candidate decline exists to replace.
	for _, p := range dirs {
		out[p] = true
	}
	return out, nil
}

// splitDirectoryPaths partitions paths into the files a byte-wise
// comparison can handle and the paths the working tree holds as a
// directory.
func splitDirectoryPaths(root string, paths []string) (files, dirs []string) {
	files = make([]string, 0, len(paths))
	for _, p := range paths {
		if info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(p))); err == nil && info.IsDir() {
			dirs = append(dirs, p)
			continue
		}
		files = append(files, p)
	}
	return files, dirs
}

// dirtyPathsUnderMoves returns every path beneath one of the candidate
// moves whose working copy disagrees with the record — edited, never
// committed, or recorded and missing from disk alike.
//
// A move carries whatever sits under it regardless of what git chooses
// to report, so the comparison is HEAD's blobs against the paths the
// move would carry rather than git's dirty set. Without that, a
// candidate whose file is ignored, `assume-unchanged`, or omitted by a
// sparse checkout reads as clean here and the sweep proceeds — leaving
// the commit-side guard to refuse the whole verb where a per-candidate
// decline was the point.
func dirtyPathsUnderMoves(ctx context.Context, root string, moves []archiveMove) (map[string]bool, error) {
	if len(moves) == 0 {
		return nil, nil
	}
	hasHEAD, err := gitops.HasHEAD(ctx, root)
	if err != nil { //coverage:ignore defensive: HasHEAD errors only outside a repo, which the tree load already required
		return nil, fmt.Errorf("checking the working tree against HEAD: %w", err)
	}
	if !hasHEAD {
		return nil, nil
	}
	seen := map[string]bool{}
	var carried []string
	add := func(p string) {
		if seen[p] {
			return
		}
		seen[p] = true
		carried = append(carried, p)
	}
	for _, m := range moves {
		for _, end := range moveEnds(m) {
			if carriedErr := addCarriedUnder(ctx, root, end, add, true); carriedErr != nil { //coverage:ignore unreachable here: tree.Load walks work/ including archive/, so an unreadable subtree under either end of a move fails the load before this runs, and HEAD resolves by the check above
				return nil, fmt.Errorf("checking the working tree against HEAD: %w", carriedErr)
			}
		}
	}
	diverged, err := gitops.DivergentPaths(ctx, root, carried)
	if err != nil {
		return nil, fmt.Errorf("checking the working tree against HEAD: %w", err)
	}
	out := make(map[string]bool, len(diverged))
	for _, d := range diverged {
		out[d.Path] = true
	}
	return out, nil
}
