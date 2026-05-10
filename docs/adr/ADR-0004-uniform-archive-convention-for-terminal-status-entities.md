---
id: ADR-0004
title: Uniform archive convention for terminal-status entities
status: accepted
---
## Context

`work/gaps/` already has 80+ entries — most in terminal status. The directory is unwieldy today, both at `ls`-time and on GitHub's tree view. The same shape is coming for other kinds:

- ADRs accumulate over the framework's life; older ones in `superseded` or `rejected` clutter the active set.
- Decisions (D-NNN) start small but grow as the project matures.
- The proposed `finding` entity kind (companion ADR-0003) is highest-volume: every TDD cycle could produce 1-3 findings, terminal-resolving on triage. Without an archive convention, `work/findings/` would bloat fastest of any kind.
- Epics and milestones don't have the same volume problem today, but the principle should hold uniformly across kinds rather than carving exceptions per-kind.

The existing model: terminal-status entities stay in place in the same directory as active entities. The id is stable; status is the truth; tooling filters. That works fine when reading via `aiwf list` or `aiwf status`, but the human-facing tree view (a plain `ls`, `find`, or GitHub's directory page) doesn't reflect "what's currently active." The mental cost of *every* directory listing is "scan past 80% archived noise."

A precedent already exists. CLAUDE.md cites `docs/pocv3/archive/gaps-pre-migration.md` — the pre-migration text record was archived under an explicit `archive/` folder rather than left in place. Generalizing this precedent into a kernel convention is the natural move.

## Decision

Terminal-status entities live under per-parent `archive/` subdirectories. Movement is **decoupled from FSM promotion** — `aiwf promote` and `aiwf cancel` flip frontmatter status only; a separate verb `aiwf archive` sweeps qualifying entities into their archive subdirs. Drift is bounded by an advisory check finding plus an optional configurable threshold.

Decoupling promotion from relocation buys two things the atomic-move model didn't: (1) a grace period after close — just-finished epics and milestones stay in their normal tree position for inspection, post-review rituals, and link-still-resolves-the-obvious-way behavior; (2) promotion verbs stay one-purpose, with no file-move side effect to test per-verb.

### Storage — per-kind layout

The convention is "one level deep `archive/` subdirectory, alongside active entities of the same parent." Concretely:

| Kind | Active location | Archive location | Trigger |
|---|---|---|---|
| Epic | `work/epics/<epic>/` (directory) | `work/epics/archive/<epic>/` (whole subtree moves) | terminal status (`done`, `cancelled`) |
| Milestone | `work/epics/<epic>/M-NNN-<slug>.md` (file inside epic dir) | **does not archive independently** — rides with the parent epic when the epic archives | n/a |
| Contract | `work/contracts/<contract>/` (directory) | `work/contracts/archive/<contract>/` (whole subtree moves) | terminal status (`retired`, `rejected`) |
| Gap | `work/gaps/<id>-<slug>.md` | `work/gaps/archive/<id>-<slug>.md` | terminal status (`addressed`, `wontfix`) |
| Decision | `work/decisions/<id>-<slug>.md` | `work/decisions/archive/<id>-<slug>.md` | terminal status (`superseded`, `rejected`) |
| ADR | `docs/adr/<id>-<slug>.md` | `docs/adr/archive/<id>-<slug>.md` | terminal status (`superseded`, `rejected`) |

`internal/entity/transition.go::IsTerminal` is the source of truth for which statuses are terminal per kind.

**Why milestones don't archive independently.** Milestones live as flat files inside their parent epic's directory, not as a top-level kind. A `done` milestone under an `active` epic stays in the epic dir until the epic itself archives. The noise problem doesn't bite at the milestone level the way it does for gaps or findings — an active epic typically carries a handful of milestones, not 80+. When the parent epic archives, its whole subtree (including any nested milestone files) moves under `work/epics/archive/<epic>/` in a single rename.

**Doc-archive scope.** Only `docs/adr/` participates in this archive convention as a `docs/` tree. `docs/pocv3/archive/` is a different sense — historical text-record archive, not entity archive — and is out of scope for the rules below. The broader question of which `docs/` trees are normative vs. exploratory vs. archival deserves its own treatment in a separate gap.

### `aiwf archive` verb

A new top-level verb sweeps active-located terminal-status entities into their archive subdirs.

```
aiwf archive [--apply] [--kind <kind>] [--root <path>]
```

- **Default is dry-run.** Without `--apply`, the verb prints the planned moves and exits without touching the tree. Re-run with `--apply` to commit.
- **Single commit per `--apply` invocation.** Per kernel principle #7, one verb invocation produces one commit. The commit message body lists affected ids and per-kind counts; the trailer is `aiwf-verb: archive` (no `aiwf-entity:` trailer — multi-entity sweeps are a special case in the trailer-keys policy).
- **Optional `--kind` scopes the sweep.** Useful when one kind is the volume offender. Without the flag, all kinds are swept.
- **No id positional.** The verb sweeps by status, not by id. There is no "archive this specific entity" mode — that would be a hand-edit detour, not a verb.

Concretely, `aiwf archive --apply` walks each kind, finds entities whose status is terminal and whose location is the active dir (or — for epics/contracts — whose parent dir is in the active position), `git mv`s them into the archive subdir, and produces one commit. Idempotent: re-runs on a clean tree are a no-op.

### Reversal — what verb undoes archive?

**You don't, deliberately.** The FSM is one-directional ("there is no 'demote'. Edit frontmatter directly if you need to back out a transition; markdown is the source of truth" — `internal/entity/transition.go`); archive is the structural projection of FSM-terminality. The kernel does not provide an `aiwf reactivate`, `aiwf un-archive`, or any reverse sweep.

The canonical pattern when a closed entity needs revisiting is to **file a new entity that references the archived one**. `Resolves: G-018` from a new G-NNN remains valid because the loader resolves ids across both active and archive directories — references stay live indefinitely.

If a contributor hand-edits frontmatter to take a status off terminal (legal at the markdown layer; status is the source of truth), the next `aiwf check` surfaces an `archived-entity-not-terminal` finding. The remediation is to revert the hand-edit, not to relocate the file. There is no auto-reconciliation in the active→terminal direction triggered by hand-edits in the reverse direction.

### Drift control

Decoupling means the tree is never strictly in convergence. Three layers bound the drift:

1. **Advisory check finding `archive-sweep-pending`.** `aiwf check` reports the count of terminal-status entities currently in active dirs. The message is specific and actionable: *"47 terminal entities awaiting `aiwf archive --apply`. Set `archive.sweep_threshold` in aiwf.yaml to escalate to blocking past N."* Hidden when zero.
2. **Configurable hard threshold.** `aiwf.yaml`'s `archive.sweep_threshold` (default unset) flips the advisory finding to blocking past the named count. Teams choose their own discipline; the default is permissive (no threshold).
3. **Skill-level nudges** in wrap rituals. `aiwfx-wrap-epic`, `aiwfx-wrap-milestone`, and similar end-of-cycle skills suggest running `aiwf archive --dry-run` at the natural moment when sweep is on the operator's mind. Advisory; doesn't satisfy CLAUDE.md §5 alone, but pairs with (1) and (2) for defense in depth.

Layer (1) is always on. Layer (2) is consumer-controlled. Layer (3) is rituals-plugin-controlled.

### `aiwf check` shape rules

Archive directories are legal locations for terminal-status entities of the matching kind. Check rules split by purpose:

- **Tree-integrity rules traverse archive in full:** `ids-unique` (id collision matters across active+archive), parse-level errors (a malformed frontmatter is still a problem in archive), and the new convergence findings introduced below.
- **Shape and health rules skip archive entirely:** `acs-shape`, `entity-body-empty-ac`, `acs-tdd-audit`, `acs-body-coherence`, `milestone-done-incomplete-acs`, `unexpected-tree-file`, etc. Archived entities are out of scope for active linting — per the forget-by-default principle, their per-rule cleanliness is not the kernel's concern.
- **Reference-validity** (`refs-resolve` in `internal/check/check.go`): id-form references in frontmatter resolve across both active and archive directories. References from active → archived ids are legal and unflagged. References from archive → active ids are not linted (the active side is fine; the archive side is out of scope for health rules).

**New finding codes:**

- `archived-entity-not-terminal` — file lives in `archive/` but frontmatter status isn't terminal. Fires after hand-edit drift. Blocking under default strictness; remediation is to revert the hand-edit (not to relocate the file — see Reversal above).
- `terminal-entity-not-archived` — file lives in active dir but status is terminal. Fires for entities awaiting sweep — the normal transient state under the decoupled model. **Advisory by default; not blocking.** Counted by `archive-sweep-pending`.
- `archive-sweep-pending` — aggregate finding reporting the count of `terminal-entity-not-archived` instances. Advisory; configurable to blocking past `archive.sweep_threshold`.

### Id resolver

`aiwf` looks up entities by id across both `work/<kind>/` and `work/<kind>/archive/` (and `docs/adr/` + `docs/adr/archive/`). References stay valid — `Resolves: G-018`, `superseded_by: ADR-0001` work whether the target is active or archived.

The `tree.Tree` loader reads both directories on every load. Cost is small (the archive grows slowly relative to the active set, and both fit in memory).

### Display surfaces

- **`aiwf list`** shows active by default. `--archived` includes archived entities (existing flag; semantics already match this convention).
- **`aiwf status`** is strictly active-only — no `--archived` flag. The narrative view is forward-looking; archive inspection lives in `aiwf list --archived`. The tree-health section gains a one-liner when sweep is pending: *"Sweep pending: N terminal entities not yet archived (run `aiwf archive --dry-run` to preview)."* Hidden when 0. Recent-activity already surfaces sweep commits naturally via `git log`.
- **`aiwf show <id>`** resolves regardless of location. The render output indicates archived state visibly (a status field or a tag in the frontmatter render).
- **`aiwf history <id>`** walks across the location move trivially via the existing trailer model — the sweep commit's `aiwf-verb: archive` trailer is recognized by history; the path-rename is what `git log -- <path>` follows for per-id timelines.
- **`aiwf render --format=html`** segregates archived entities at the index level. Per-kind index pages render active-only by default (the page reachable from the home nav); a separate full-set index page (`<kind>/all.html`) renders the whole set; per-entity HTML pages render regardless of status (so deep links from external sources don't 404). Static `<a>` nav between views — no JS layer. Concrete UI affordances beyond this segregation rule (filter chips, JS-driven view switching, etc.) are render-implementation decisions captured under a downstream render milestone, not pinned by this ADR.

The active-by-default pattern across these surfaces is the discoverability inversion of CLAUDE.md's "Kernel functionality must be AI-discoverable": archived entities are deliberately *less* discoverable in the default scan. AI assistants and humans alike scan active first; archive is opt-in via explicit flag, explicit id reference, or navigation to the full-set surface.

### Migration

There is no separate one-time migration verb. The first run of `aiwf archive --apply` in an existing repo *is* the migration: it sweeps all currently-terminal entities into their archive subdirs in one commit. Subsequent runs sweep only what's accumulated since.

Operators ratifying this ADR run `aiwf archive --dry-run` first to preview the move, then `aiwf archive --apply` to commit. The same verb covers the bulk historical migration and the recurring small sweeps that follow.

### Compatibility with [ADR-0001](ADR-0001-mint-entity-ids-at-trunk-integration-via-per-kind-inbox-state.md)

The proposed inbox/mint model adds an `inbox/` subdirectory under each kind for pre-mint state. The archive convention adds an `archive/` subdirectory for terminal-state. The two are orthogonal; both shape rules apply:

- `work/<kind>/<id>-<slug>.md` — active, minted entities.
- `work/<kind>/inbox/<slug>.md` — pre-mint, branch-local entities.
- `work/<kind>/archive/<id>-<slug>.md` — terminal-state, minted entities.

No path overlaps. Both verbs can coexist in the same commit if needed (rare but legal).

## Consequences

**Positive:**

- Working view of `work/<kind>/` reflects active state (modulo unswept terminals; bounded by drift control). After the first `aiwf archive --apply`, `ls work/gaps/` shrinks from 80+ to ~28 entries; `ls work/findings/` will show only open findings.
- Discovery (`find work/`, GitHub tree, file-tree IDE panes) gives humans a clear "what's currently active" without filter ceremony.
- Generalizes uniformly across all kinds. The per-kind storage table reflects the actual kind-shape (file vs. directory; nested vs. top-level), not exceptions to the rule.
- Compatible with ADR-0001 (proposed): inbox/mint and archive coexist without conflict.
- **Promotion verbs stay one-purpose.** `aiwf promote` and `aiwf cancel` don't gain a file-move side effect; their tests don't grow archive-aware branches.
- **Grace period for inspection.** Just-closed epics and milestones stay in their normal tree position until the next sweep — operators can run wrap rituals, skim the dust just settled, follow `aiwf show` paths without "where did it go?" friction.
- **Single verb covers migration and recurrence.** No transitional `aiwf archive-existing` to deprecate; first run *is* the migration.
- **Stable deep links from the rendered site.** Per-entity HTML pages render regardless of status; external links survive archive moves.

**Negative:**

- **Tree is never strictly in convergence.** Between promotion and the next sweep, location and status diverge. Drift is bounded by the advisory finding + optional threshold; consumers who want strict convergence configure the threshold low.
- **One more verb to learn** (`aiwf archive`). Marginal cognitive cost; the verb's semantics are obvious from the name and the dry-run-default discipline gives it natural safety.
- **Merge edge cases.** Branch A archives G-0018 (rename to `archive/`) while branch B edits G-0018 in place. Git's rename detection usually handles this; occasionally produces a rename+modify conflict. Resolution is mechanical (take the rename, take the edit) but not zero-cost. Standard kernel pattern of "merge, run check, fix findings" handles this.
- **Cross-references in body prose that use file paths (rather than ids) become stale when the target archives.** Standard kernel discipline already prefers id-form references; this ADR makes path-form references unambiguously brittle. **G-0091** captures the work item for a preventive check rule; the existing post-hoc lychee CI workflow is the safety net in the meantime.
- **Hand-edit drift in the wrong direction is a finding the operator must clean up by reverting status, not by re-locating.** A small UX friction the first time someone tries to "un-archive" an entity by editing frontmatter; documented in the Reversal section so the recovery is unambiguous.

## Alternatives considered

- **Atomic move on terminal-promotion** (the original draft of this ADR). Status flip and file move happen in the same commit; always-converged tree; no drift. Loses the grace period for post-close inspection; couples FSM mutation with housekeeping in every terminal-promotion verb; forces every verb (`promote`, `cancel`, `reallocate` when target is terminal-status, etc.) to grow a file-move side effect that must be tested per-verb. The convergence benefit is primarily cosmetic — the kernel reads location and status independently, and `aiwf list` filters by status either way. Rejected in favor of the decoupled model after weighing the grace-period UX gain against the bounded drift cost.
- **Virtual archive (display-only filter).** No file moves; tooling filters terminal entities by default. Cheapest implementation. Doesn't solve the "what does GitHub show me" UX concern, which is the main motivation of this ADR. Rejected.
- **Manual `aiwf archive G-018` verb (per-id housekeeping).** Explicit per-entity archive verb decoupled from status. Opens a third state ("status active, but I want this archived anyway") that has no semantic basis given the kernel principle that location is a redundant projection of status. Rejected — the sweep verb's "archive what's terminal" rule preserves the projection.
- **Global `archive/` folder rather than per-kind.** Slightly less typing in the path. Loses the per-kind storage model that the rest of the framework's invariants rely on (`tree.Tree` loader scans per-kind directories; check rules are kind-scoped). Rejected.
- **Time-based archive (`archive/2026-q2/`).** Useful for very long-lived projects with thousands of entries per kind. Premature for the framework's current scale; revisit if archive directories themselves grow unwieldy.
- **No archive at all (status-of-truth model only).** The current state. Adequate for active filtering via tooling but loses the "what does the directory listing reflect" affordance, which becomes load-bearing as kinds scale.
- **Filter chips with JS in the rendered site.** Considered for `aiwf render`'s archive surface — multi-select chips, JSON sidecar, URL hash for deep-linkable filter state. Real ergonomic gain for governance browsing. Rejected for now in favor of the simpler static segregation (separate index pages, plain `<a>` nav) — chip UI is a render-impl decision that belongs in a downstream render milestone, not the archive ADR.

## References

- CLAUDE.md "What aiwf commits to" §2 (stable ids that survive rename, cancel, collision — preserved by this ADR).
- CLAUDE.md "What aiwf commits to" §3 (pre-push hook is the chokepoint).
- CLAUDE.md "What aiwf commits to" §5 (framework correctness must not depend on LLM behavior — drift control is a finding, not a skill prompt alone).
- CLAUDE.md "What aiwf commits to" §7 (every mutating verb produces exactly one git commit — the sweep is one commit).
- CLAUDE.md "Designing a new verb" — verb-design rule answered by the Reversal section.
- `internal/entity/transition.go::IsTerminal` — source of truth for terminal statuses per kind.
- `internal/check/check.go::refsResolve`, `internal/entity/refs.go::ForwardRefs` — id-form ref resolution (archive-safe by id-resolution scope spanning active+archive).
- Companion **ADR-0003** — `finding` (F-NNN) as a seventh entity kind; co-evolved alongside this ADR because findings are the highest-volume archive consumer.
- Related **G-0071** — `entity-body-empty` lifecycle-blindness; **closed by M-0075** via status-gating (rule skips terminal-status entities and draft-milestone ACs). This ADR's location-gating (shape rules skip `archive/`) is an orthogonal defense-in-depth layer; once M-0075 ships, terminal entities are skipped on status grounds before they're ever swept into archive.
- Related **G-0091** — body-prose path-form refs to entity files have no preventive check (filed as the natural follow-up to this ADR).
- Related **G-0092** — no documented hierarchy of doc authority across `docs/` (filed as the natural follow-up to this ADR's doc-archive-scope clarification).
- Precedent: `docs/pocv3/archive/gaps-pre-migration.md` — existing pre-migration text record archived under an explicit `archive/` folder.
