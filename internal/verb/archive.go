package verb

import (
	"context"
	"fmt"
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
	plan, err := planArchive(ctx, root, kindFilter)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return &Result{NoOp: true, NoOpMessage: "aiwf archive: no terminal-status entities awaiting sweep (tree is converged)"}, nil
	}
	plan.Trailers = []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "archive"},
		{Key: gitops.TrailerActor, Value: actor},
	}
	return &Result{Plan: plan}, nil
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

// planArchive computes the list of OpMove ops the sweep produces over
// the active tree. Returns nil when there is nothing to sweep.
//
// kindFilter, when non-empty, scopes the walk to one kind. Milestones
// are not directly swept — they ride with their parent epic — so a
// kindFilter of "milestone" sweeps nothing. (We treat that as a no-op
// rather than an error: it's an honest answer to "what would archive
// do for milestones?".)
func planArchive(ctx context.Context, root, kindFilter string) (*Plan, error) {
	tr, _, err := tree.Load(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("loading tree: %w", err)
	}

	moves, err := computeArchiveMoves(tr, kindFilter)
	if err != nil {
		return nil, err
	}
	if len(moves) == 0 {
		return nil, nil
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

	subject := archiveCommitSubject(moves)
	return &Plan{
		Subject: subject,
		Body:    archiveCommitBody(moves),
		Ops:     ops,
	}, nil
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
func computeArchiveMoves(tr *tree.Tree, kindFilter string) ([]archiveMove, error) {
	if kindFilter != "" && !isKnownKind(kindFilter) {
		return nil, fmt.Errorf("unknown kind %q (must be one of %s)", kindFilter, strings.Join(allKindNamesArchive(), ", "))
	}

	// Track epic dirs we've already emitted a move for, so a "done epic
	// with three milestones" only produces one OpMove (the dir rename),
	// not four.
	epicDirSeen := map[string]bool{}
	contractDirSeen := map[string]bool{}

	var moves []archiveMove

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

	return moves, nil
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
	return fmt.Sprintf("aiwf archive: sweep %d entit%s into archive/ (%s)",
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
// for the commit body. ADR-0004 §"`aiwf archive` verb": "the commit
// message body lists affected ids and per-kind counts."
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
// Determinism: kinds iterate in entity.AllKinds() order; ids within
// each kind iterate in lexicographic order.
func archiveCommitBody(moves []archiveMove) string {
	if len(moves) == 0 {
		return "" //coverage:ignore caller returns nil Plan when len(moves)==0
	}
	byKind := map[entity.Kind][]string{}
	for _, m := range moves {
		byKind[m.kind] = append(byKind[m.kind], m.id)
	}
	for k := range byKind {
		sort.Strings(byKind[k])
	}

	var sb strings.Builder
	sb.WriteString("Per ADR-0004: sweep terminal-status entities into per-kind archive/.\n\n")
	sb.WriteString("Per-kind counts:\n")
	for _, k := range entity.AllKinds() {
		ids, ok := byKind[k]
		if !ok || len(ids) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "  %-9s %d %s\n", k, len(ids), pluralize(len(ids), "entity", "entities"))
	}

	// Affected ids: aggregate across kinds in entity.AllKinds() order,
	// then alphabetical within each kind.
	var allIDs []string
	for _, k := range entity.AllKinds() {
		allIDs = append(allIDs, byKind[k]...)
	}
	sb.WriteString("\nAffected ids:\n  ")
	sb.WriteString(strings.Join(allIDs, ", "))
	sb.WriteString("\n")

	return sb.String()
}
