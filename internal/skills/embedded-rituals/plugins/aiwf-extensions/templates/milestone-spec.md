---
id: M-NNN
title: <imperative title>
parent: E-NN              # required: the parent epic id
status: draft             # aiwf milestone statuses: draft | in_progress | done | cancelled
depends_on: []            # optional: prior milestone ids the DAG depends on
tdd: none                 # optional: required | advisory | none (default none)
acs: []                   # optional: filled by `aiwf add ac <M-id> --title "..."`
---

# M-NNN — <Milestone Title>

## Goal

<1–2 sentences: what this milestone achieves.>

## Context

<!-- 2–3 sentences: what exists before this milestone, what must be in place, why now.
     Prior milestones, blocking dependencies resolved, decisions landed.
     Not a re-telling of the epic. -->

<What exists before this milestone? What prior milestones does it build on? Why now?>

## Acceptance criteria

<!-- ACs are first-class kernel state under aiwf I2. Add each via:
       aiwf add ac M-NNN --title "<observable behavior>"
     The verb appends the AC to frontmatter `acs:` (with `tdd_phase: red` seeded
     when the milestone is `tdd: required`) and scaffolds a `### AC-<N> — <title>`
     heading below this section. Don't hand-edit `acs:` — the position-stable
     allocator and the body-coherence check both depend on the verb path.

     Each AC must be observable behavior, not an implementation detail.
       Good:  "When X occurs, the system emits Y with property Z."
       Bad:   "X is tested." / "Refactor complete." / "Feature implemented." -->

### AC-1 — <observable behavior>

<Prose: examples, edge cases, references to ADR-NNNN / D-NNN / surfaces touched.>

### AC-2 — <observable behavior>

<Prose…>

## Constraints

- <Non-negotiable invariants, banned shortcuts, shim-policy exceptions with a named removal trigger>

## Design notes

- <Locked decisions approved before implementation. Reference ADRs by id (ADR-NNNN) or aiwf decisions (D-NNN)>

## Surfaces touched (optional)

<!-- 1–5 items, not an exhaustive file dump. A pointer so an implementer knows where to
     start reading. Omit for small or obvious milestones. -->

- <path or module>

## Out of scope

- <What this milestone explicitly does NOT do>

## Dependencies

- <Prior milestone, external dep, decision record — what must exist before starting>

## Coverage notes (optional)

<!-- Reachable branches the implementation deliberately leaves untested, with the reason.
     The wf-tdd-cycle branch-coverage hard rule expects every reachable branch to have a
     test. Genuinely unreachable branches (defensive null checks the type system already
     guarantees, etc.) are documented here. -->

- <branch> — <why it can't be reached>

## References

- <ADRs (ADR-NNNN), aiwf decisions (D-NNN), related specs, external docs>

---

<!-- The sections below are populated continuously through implementation and
     finalized at `aiwfx-wrap-milestone`. They replace the v1 `tracking-doc.md`
     convention; aiwf does not validate their contents (prose is human-owned),
     but `aiwfx-start-milestone` / `aiwfx-wrap-milestone` rely on the structure. -->

## Work log

<!-- One entry per AC (preferred) or per meaningful unit of work. Append-only;
     never rewrite earlier entries.
       Header:    "AC-<N> — <short title>" or "<short title>" if not AC-scoped.
       First line: <one-line outcome> · commit <SHA> · tests <N/M>
     Optional prose paragraph for non-obvious context: what changed, file:line
     references, why a detour was needed. Phase transitions for `tdd: required`
     milestones should be visible here too (red/green/refactor/done) — but the
     authoritative record is `aiwf history M-NNN/AC-<N>` via the kernel's
     trailers, so don't duplicate the timeline here. -->

### AC-1 — <short title>

<one-line outcome> · commit <SHA> · tests <N/M>

## Decisions made during implementation

<!-- Decisions that came up mid-work that were NOT pre-locked above in `## Design
     notes`. For each: what was decided, why, and a link to the ADR or D-NNN id
     that captures the durable reasoning (use `aiwfx-record-decision`).
     If no new decisions arose, say "None — all decisions are pre-locked above." -->

- (none)

## Validation

<!-- Pasted at wrap. Test-suite results, build output, any project-specific lint
     or type-check. Replaces the v1 tracking-doc's Validation block. -->

## Deferrals

<!-- Work this milestone deliberately punted. Each must be opened as a gap entity
     (`aiwf add gap --title "..." --discovered-in M-NNN`) and the resulting
     G-NNN id mirrored here, so the deferral survives. -->

- (none)

## Reviewer notes

<!-- Trade-offs, deliberate omissions, places where the obvious approach was
     rejected. Filled at wrap; the reviewer agent reads this first. -->

- (none)
