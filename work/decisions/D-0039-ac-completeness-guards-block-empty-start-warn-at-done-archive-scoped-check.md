---
id: D-0039
title: 'AC-completeness guards: block empty start, warn at done, archive-scoped check'
status: accepted
priority: high
relates_to:
    - G-0216
    - G-0286
    - G-0334
---
# D-0039 — AC-completeness guards: block empty start, warn at done, archive-scoped check

> **Date:** 2026-07-19 · **Decided by:** human/peter

## Question

G-0216, G-0286, and G-0334 all touch `internal/check/acs.go` and share one theme — does the kernel mechanically enforce the AC/milestone completeness discipline, or does it rely on operator vigilance. G-0334 (a milestone can start and finish with zero ACs) and G-0216 (a milestone can start with AC bodies that are empty prose) each left open how strict the new guard should be, and whether a check-time finding introduced now would immediately redden the tree with pre-existing history. Three forks needed settling before the cluster is spec-ready:

1. Should a zero-AC milestone be refused at `draft → in_progress`, or only warned about?
2. Should a zero-AC milestone also be gated at `done`, given it may already have been gated at start?
3. Should G-0216's new check-time finding (empty AC body on an `in_progress`/`done` milestone) need a new grandfathering mechanism to stay forward-only, or can it reuse machinery the file already has?

(G-0286 — relaxing `acs-shape/tdd-phase` so an absent phase is legal until an AC reaches `met` — was reviewed alongside this cluster because it shares the same file and theme, but it is a scope-relaxation bug fix with no open fork; it needed no decision and is not covered by this entry.)

## Decision

1. **`draft → in_progress` on a zero-AC milestone is refused** by the promote verb, with the standard `--force --reason "..."` override.
2. **`done` on a zero-AC milestone is not refused** — instead, a new warning-severity check-time finding surfaces it (extending the existing `milestoneDoneIncompleteACs` pattern), so the state is visible without a second hard stop.
3. **G-0216's check-time finding for empty AC bodies reuses the existing `entity.IsArchivedPath(e.Path)` archive-scoping guard** already applied by every sibling rule in `internal/check/acs.go` (`acsShape`, `acsTDDAudit`, `acsBodyCoherence`, `milestoneDoneIncompleteACs`, `milestoneCancelledIncompleteACs`). No new timestamp- or marker-based grandfather mechanism is introduced.

## Reasoning

**Point 1 — block, don't warn.** G-0334's own "why it matters" section already diagnoses the current state as vacuous, vigilance-dependent discipline: a zero-AC milestone can traverse its whole lifecycle without ever substantiating anything. A warning-only finding (the option G-0334 originally leaned toward, mirroring the advisory `epic-active-no-drafted-milestones`) doesn't change that outcome — it only makes the vacuity visible, which the tree already tolerates for dozens of other warnings. A hard block does change the outcome, and it closes a real gaming path: G-0216 already proposes blocking "AC exists with an empty body," so leaving "no AC at all" as only a warning would let the exact same discipline be dodged by omitting the AC entirely. `--force --reason` keeps genuinely AC-less (coordination-only, exploratory) milestones unblocked, at the cost of one flag and one recorded sentence — friction that lands only on the case where friction is the point, not on the common path of a milestone that was always going to carry ACs.

Considered and rejected: warning-only (Option A above) — rejected because it doesn't address the vacuity G-0334 names, and creates an asymmetry with G-0216's own proposed hard block for the sibling problem.

**Point 2 — warn at done, don't block again.** By the time a milestone reaches `done`, the meaningful decision was already made at the start-time gate (point 1): either the milestone was force-started with a recorded reason, or it was already `in_progress` before this rule shipped (grandfathered). Requiring `--force --reason` a second time at `done` asks the same question twice for the force-started case, and is punitive for the grandfathered case — the work already happened; refusing to close it over a paperwork gap that predates the rule doesn't correct anything. A warning-severity check-time finding still gives the grandfathered case visibility for triage, without a second friction point that adds no new decision.

Considered and rejected: a second hard block at `done` (symmetric with point 1) — rejected as redundant friction with no new information gained; no additional check at `done` at all — rejected because it leaves the grandfathered case with zero visibility, undermining the audit trail the rest of the cluster is building.

**Point 3 — reuse the file's own archive-scoping convention.** Every existing rule in `internal/check/acs.go` — `acsShape`, `acsTDDAudit`, `acsBodyCoherence`, `milestoneDoneIncompleteACs`, `milestoneCancelledIncompleteACs` — already skips archived milestones via the same one-line `entity.IsArchivedPath(e.Path)` guard (per ADR-0004's "shape-and-health rules are active-tree only" convention). Checked empirically this session (`aiwf list --kind milestone --status in_progress` / `--status done` against the live tree): there are currently zero non-archived milestones in either status, so reusing this guard produces zero blast radius today, and going forward it only exposes actively-being-worked milestones to the new rule — a completed milestone permanently leaves scope the moment it's archived, exactly like every other AC-shape rule already behaves.

Considered and rejected: a timestamp/marker-based grandfather keyed to when the rule landed — rejected as speculative complexity (new machinery to trace rule-landing-time against an entity's transition history) for a problem that, measured against the actual tree, doesn't currently exist; tree-wide with no archive scoping at all — rejected because it breaks the file's own established precedent and would resurface old completed work as permanent errors, contradicting G-0216's own stated intent of no retroactive refusal.

## Consequences

- `internal/verb/promote.go` (or the milestone-specific promote handler) gains a zero-AC refusal at `draft → in_progress`, mirroring the shape of G-0216's own proposed empty-body refusal.
- `internal/check/acs.go` gains two new findings: a warning at `done` with zero ACs (extends `milestoneDoneIncompleteACs`'s pattern), and G-0216's empty-AC-body finding, both scoped with the existing archive guard.
- G-0216 and G-0334's bodies should be updated to reflect these settled points before either is scoped into milestone ACs.
- G-0286 remains an independent, undecided-nothing bug fix and can be implemented on its own schedule relative to the other two.
