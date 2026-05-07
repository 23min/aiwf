---
id: ADR-0004
title: Uniform archive convention for terminal-status entities
status: proposed
---
## Context

`work/gaps/` already has 66 entries — 52 in `addressed` (terminal) status, 14 `open`. The directory is unwieldy today, both at `ls`-time and on GitHub's tree view. The same shape is coming for other kinds:

- ADRs accumulate over the framework's life; older ones in `superseded` or `rejected` clutter the active set.
- Decisions (D-NNN) start small but grow as the project matures.
- The proposed `finding` entity kind (companion ADR) is highest-volume: every TDD cycle could produce 1-3 findings, terminal-resolving on triage. Without an archive convention, `work/findings/` would bloat fastest of any kind.
- Epics and milestones don't have the same volume problem today, but the principle should hold uniformly across kinds rather than carving exceptions per-kind.

The existing model: terminal-status entities stay in place in the same directory as active entities. The id is stable; status is the truth; tooling filters. That works fine when reading via `aiwf list` or `aiwf status`, but the human-facing tree view (a plain `ls`, `find`, or GitHub's directory page) doesn't reflect "what's currently active." The mental cost of *every* directory listing is "scan past 80% archived noise."

A precedent already exists. CLAUDE.md cites `docs/pocv3/archive/gaps-pre-migration.md` — the pre-migration text record was archived under an explicit `archive/` folder rather than left in place. Generalizing this precedent into a kernel convention is the natural move.

The PoC is likely to graduate. Picking the storage model that scales now — rather than retrofitting after `work/findings/` and `work/gaps/` both pass 100 entries — is the right call.

## Decision

When a verb promotes any entity to a terminal status (or `aiwf cancel` runs), the same atomic commit that flips the status **moves the file** from `work/<kind>/` to `work/<kind>/archive/`.

### Trigger

Any FSM transition into a terminal status. The FSM definitions per kind in `internal/entity/transition.go` are the source of truth for which statuses are terminal:

- epic: `done`, `cancelled`
- milestone: `done`, `cancelled`
- ADR: `accepted` (no — `accepted` is not terminal; only `superseded` and `rejected` are), `superseded`, `rejected`
- gap: `addressed`, `wontfix`
- decision: `superseded`, `rejected` (`accepted` is not terminal — decisions are accepted-and-active state)
- contract: `deprecated` (not terminal — has further transition to `retired`), `retired`, `rejected`
- finding (proposed): `resolved`, `waived`, `invalid`

The verb implementations consult `entity.IsTerminal(kind, status)` and perform the move atomically with the status flip. Both happen in the same commit per kernel principle #7.

### Storage

`work/<kind>/archive/<old-filename>`. The archive directory is **per-kind**, not global, to preserve the per-kind storage model that the rest of the framework's invariants rely on.

For ADRs (which live under `docs/adr/`), the same convention applies: `docs/adr/archive/<filename>`. The convention is "co-located archive folder, one level deep, alongside active entities of the same kind."

### Id-resolver

`aiwf` looks up entities by id across both `work/<kind>/` and `work/<kind>/archive/` (and `docs/adr/` + `docs/adr/archive/`). References stay valid — `Resolves: G-018`, `superseded_by: ADR-0001` work whether the target is active or archived.

The `tree.Tree` loader reads both directories on every load. Cost is small (the archive grows slowly relative to the active set, and both fit in memory).

### `aiwf check` shape rules

Archive subdirectories are legal locations for entities of the matching kind. New finding codes:

- `archived-entity-not-terminal` — file lives in `archive/` but frontmatter status isn't terminal. Fires after hand-edit drift (someone moved a file then changed its status, or vice versa).
- `terminal-entity-not-archived` — file lives in active dir but status is terminal. Fires for entities that pre-date this ADR landing (transitional finding) and for any verb that flips status without performing the move (regression).

Both are blocking findings under default strictness; both are mechanically resolvable by the migration verb (transitional cases) or by re-running the canonical promotion verb (regression cases).

### Display surfaces

- `aiwf list` and `aiwf status` show **active by default**. `--include-archived` (or `--archived`) includes archived entities.
- `aiwf show <id>` resolves regardless of location — id-resolver scans both. The render output indicates archived state visibly (a status field or a tag in the frontmatter render).
- `aiwf history <id>` walks across the location move trivially via the existing trailer model — the move is just another commit on the entity's path, recorded with `aiwf-verb: cancel` (or whichever terminal-promotion verb fired) and the standard `aiwf-entity:` trailer.

### Migration (one-time)

A new admin verb `aiwf archive-existing` (or a `--migrate-archive` flag on `aiwf check`) bulk-moves currently-terminal entities into their archive subdirs in a single commit per kind. Preserves git history via rename detection. Idempotent: re-runs are no-ops once the tree is consistent.

The migration is run once per consumer repo. After a clean run, the new shape rules become enforcing rather than transitional.

### Compatibility with [ADR-0001](ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md)

The proposed inbox/mint model adds an `inbox/` subdirectory under each kind for pre-mint state. The archive convention adds an `archive/` subdirectory for terminal-state. The two are orthogonal; both shape rules apply:

- `work/<kind>/<id>-<slug>.md` — active, minted entities.
- `work/<kind>/inbox/<slug>.md` — pre-mint, branch-local entities.
- `work/<kind>/archive/<id>-<slug>.md` — terminal-state, minted entities.

No path overlaps. Both verbs can coexist in the same commit if needed (rare but legal).

## Consequences

**Positive:**

- Working view of `work/<kind>/` reflects active state. After migration, `ls work/gaps/` shows ~14 entries instead of 66; `ls work/findings/` will show only open findings.
- Discovery (`find work/`, GitHub tree, file-tree IDE panes) gives humans a clear "what's currently active" without filter ceremony.
- Generalizes uniformly across all kinds. One rule, one mechanism, no per-kind exceptions.
- Compatible with ADR-0001 (proposed): inbox/mint and archive coexist without conflict.
- Existing precedent (`docs/pocv3/archive/gaps-pre-migration.md`) is already in the tree; this ADR generalizes the pattern rather than inventing something new.

**Negative:**

- Every terminal-status verb (`aiwf cancel`, `aiwf promote ... <terminal>`, `aiwf reallocate` when the renamed entity is terminal-status, etc.) gains a file-move side effect. One more thing for the verb implementations to do correctly. Tested per-verb.
- Merge edge cases: branch A archives G-018 (rename to `archive/`) while branch B edits G-018 in place. Git's rename detection usually handles this; occasionally produces a rename+modify conflict. Resolution is mechanical (take the rename, take the edit) but not zero-cost. Existing kernel pattern of "merge, run check, fix findings" handles this.
- The `aiwf archive-existing` migration verb is a one-time tool that nonetheless requires careful design (idempotency, dry-run, scoped to a single kind, preserves git rename detection). Captured as work in the implementation epic.
- Two layouts to consider in tooling: pre-archive and post-archive. CI for older branches that pre-date the change must keep working until those branches retire. Soft transitional findings cover this period.
- Cross-references in body prose that use file paths (rather than ids) become stale when the target archives. Standard kernel discipline already prefers id-form references; this ADR makes path-form references unambiguously brittle.

## Alternatives considered

- **Virtual archive (display-only filter).** No file moves; tooling filters terminal entities by default. Cheapest implementation. Doesn't solve the "what does GitHub show me" UX concern, which is the main motivation of this ADR. Rejected.
- **Manual `aiwf archive G-018` verb (decoupled from promotion).** Explicit housekeeping ritual. Accumulates debt because the discipline is manual; the existing 52 addressed gaps would still sit in `work/gaps/` until someone runs the verb. Rejected as default; the migration verb is a one-time form of this idea.
- **Global `archive/` folder rather than per-kind.** Slightly less typing in the path. Loses the per-kind storage model that the rest of the framework's invariants rely on (`tree.Tree` loader scans per-kind directories; check rules are kind-scoped). Rejected.
- **Time-based archive (`archive/2026-q2/`).** Useful for very long-lived projects with thousands of entries per kind. Premature for the PoC; revisit if archive directories themselves grow unwieldy.
- **No archive at all (status-of-truth model only).** The current state. Adequate for active filtering via tooling but loses the "what does the directory listing reflect" affordance, which becomes load-bearing as kinds scale.

## References

- CLAUDE.md "What the PoC commits to" §2 (stable ids that survive rename, cancel, collision — preserved by this ADR).
- `docs/pocv3/design/tree-discipline.md` — existing tree-shape rules; this ADR adds a sub-rule.
- `internal/entity/transition.go` — source of truth for which statuses are terminal per kind.
- Companion ADR: `finding` as a seventh entity kind (filed alongside this one).
- Precedent: `docs/pocv3/archive/gaps-pre-migration.md`.
