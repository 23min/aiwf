---
# Field vocabulary — the allowed statuses, which fields are optional, and
# what each reference accepts — is printed by `aiwf schema milestone`;
# the value set behind a flag is in `aiwf add milestone --help`.
id: M-NNNN
title: <imperative title>
parent: E-NNNN
status: draft
depends_on: []
tdd: none
acs: []                  # filled by `aiwf add ac`, never by hand
---

<!-- A spec states what will be built and what is excluded. The reasoning behind a
     choice — a rejected alternative, why an approach fails, a trade-off argued out —
     belongs in an ADR or a decision record; reference that record by id from
     `## Design notes` rather than reproducing the argument. Name the record and
     what it settles for this work; do not restate what it argues. Delete this
     comment after copying. -->

## Goal

<1–2 sentences: what this milestone achieves.>

## Closes

<!-- Gaps this milestone sets out to close, one id per line, recorded now rather
     than reconstructed at wrap. List a gap only when this work is expected to
     resolve it: a gap the work merely touches, or punts, belongs under
     `## Deferrals`. `aiwfx-wrap-milestone` reads this section and closes each id
     listed; one the work advanced without finishing gets its claim corrected
     there instead. Delete the section when the milestone closes nothing.
     One line per gap: the id, then what this milestone resolves in it. -->

- (none)

## Context

<!-- 2–3 sentences: what exists before this milestone, what must be in place, what
     changed to make it possible now. Prior milestones, blocking dependencies
     resolved, decisions landed. Not a re-telling of the epic, and not an argument
     for the work. -->

<What exists before this milestone? What prior milestones does it build on? What
changed to make it possible now?>

## Acceptance criteria

<!-- ACs are first-class kernel state under aiwf I2. Add each via:
       aiwf add ac <milestone-id> --title "<observable behavior>"
     The verb appends the AC to frontmatter `acs:` (seeded at the pre-cycle
     empty phase regardless of tdd policy — the live red promote records the
     failing test later) and scaffolds a `### AC-<N> — <title>`
     heading below this section. Don't hand-edit `acs:` — the position-stable
     allocator and the body-coherence check both depend on the verb path.

     Each AC must be observable behavior, not an implementation detail.
       Good:  "When X occurs, the system emits Y with property Z."
       Bad:   "X is tested." / "Refactor complete." / "Feature implemented." -->

### AC-1 — <observable behavior>

<Prose: examples, edge cases, references to `ADR-NNNN` / `D-NNNN` / surfaces touched.>

### AC-2 — <observable behavior>

<Prose…>

## Constraints

- <Non-negotiable invariants, banned shortcuts, shim-policy exceptions with a named removal trigger>

## Design notes

- <Locked decisions approved before implementation. Reference ADRs by id (`ADR-NNNN`) or aiwf decisions (`D-NNNN`)>

## Surfaces touched

<!-- Optional — 1–5 items, not an exhaustive file dump. A pointer so an implementer knows
     where to start reading. Omit for small or obvious milestones. -->

- <path or module>

## Out of scope

- <What this milestone explicitly does NOT do>

## Dependencies

- <Prior milestone, external dep, decision record — what must exist before starting>

## Coverage notes

<!-- Optional — reachable branches the implementation deliberately leaves untested, with the reason.
     The wf-tdd-cycle branch-coverage hard rule expects every reachable branch to have a
     test. Genuinely unreachable branches (defensive null checks the type system already
     guarantees, etc.) are documented here. -->

- <branch> — <why it can't be reached>

## References

- <ADRs (`ADR-NNNN`), aiwf decisions (`D-NNNN`), related specs, external docs>

---

<!-- The sections below are populated continuously through implementation and
     finalized at `aiwfx-wrap-milestone`. aiwf does not validate their contents
     (prose is human-owned), but `aiwfx-start-milestone` / `aiwfx-wrap-milestone`
     rely on the structure. -->

## Release note

<!-- The user-visible delta of this milestone, written for someone reading
     release notes who will never see this spec: what a consumer can now do, or
     what changed under them.

     The epic wrap composes the epic's changelog entry from these notes, and
     that entry is copied verbatim into the changelog — so this is the last
     point at which the change is described by someone who did the work.
     Reconstructing it later from milestone titles is what leaves a shipped
     change undescribed.

     Not the account of how it was built, which `## Work log` holds, and not
     the epic's summary, which the epic wrap writes. One short paragraph, or one line per
     user-facing change. "No user-visible change" is a valid note; empty is
     not. -->

## Work log

<!-- The narrative an AC's history cannot hold: what a detour cost, why an
     approach was abandoned, what a commit does not say about itself. The AC's
     own history answers which commit implemented it, once that commit carries
     the AC in its entity trailer, so the SHA below is a locator beside the
     account rather than the record of the link.
     One entry per AC (preferred) or per meaningful unit of work. Append-only;
     never rewrite earlier entries.
       Header:     "AC-<N> — <short title>" or "<short title>" if not AC-scoped.
       First line: <one-line outcome> · commit <SHA> · tests <N/M>
     That line is the entry. The phase timeline lives in
     `aiwf history <milestone-id>/AC-<N>`. Design reasoning belongs in the code it
     explains; anything else has its own section. -->

### AC-1 — <short title>

<one-line outcome> · commit <SHA> · tests <N/M>

## Decisions made during implementation

<!-- Decisions that came up mid-work that were NOT pre-locked above in `## Design
     notes`. For each: what was decided, why, and a link to the ADR or decision id
     that captures the durable reasoning (use `aiwfx-record-decision`).
     If no new decisions arose, say "None — all decisions are pre-locked above." -->

- (none)

## Validation

<!-- Pasted at wrap. Test-suite results, build output, any project-specific lint
     or type-check. -->

## Deferrals

<!-- Work this milestone deliberately punted. Apply the cheap-fix test first: a
     change that is small, lands in a file this milestone already touches, and is
     covered by a test you are already writing gets made now rather than filed.
     Each deferral that survives the test must be opened as a gap entity
     (`aiwf add gap --title "..." --discovered-in <milestone-id>`) and the
     resulting gap id mirrored here, so the deferral survives. -->

- (none)

## Reviewer notes

<!-- Trade-offs, deliberate omissions, places where the obvious approach was
     rejected, and the deciding review's own outcome. Filled at wrap, after
     that review — so the review whose outcome it records cannot have read it.
     A later reviewer reads it first. -->

- (none)
