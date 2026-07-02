---
id: G-0326
title: aiwf add permits empty load-bearing bodies on born-complete kinds
status: open
---
## What's missing

`aiwf add` lets a **born-complete** entity — gap, decision, ADR, contract — be
created with an empty load-bearing body. The kernel's `entity-body-empty` rule
(M-066, `internal/check/entity_body.go`) fires only at **warning** severity, so
the creating commit is green and nothing stops a body-less entity from landing.

"Born-complete" means the kind has **no draft phase** in which an empty body is
by design. The gap FSM is `open → addressed | wontfix` (`internal/entity/
entity.go:515`) — a gap is born `open` and live. Contrast a milestone, which is
born `draft` and whose ACs legitimately ship shape-first and gain prose as TDD
proceeds (the explicit `StatusDraft` suppression at `entity_body.go:175`). The
warning's leniency is principled for draft-bearing kinds; it is mis-applied to
born-complete kinds, whose body *is their reason for existing* — an unarticulated
gap is a title pretending to be a gap.

The body must be present at the **creation commit**, not "filled in later,"
because on a trunk-based repo there is no "later" that is safe.

## Why it matters

aiwf is trunk-based: maintainers `aiwf add gap` **directly on `main`**. A
trunk-created entity is **referenceable the instant the `add` commit lands** —
the id is live and the next `aiwf add` can set `discovered_in:` / `addressed_by:`
/ `parent:` or cite it in prose. There is no quarantine window between creation
and referenceability, so any "catch it at a downstream boundary" design (a
branch→trunk merge gate) structurally cannot fire for the normal path — there is
no merge.

aiwf validates that a reference *exists*, not that it is *substantive*. An
empty-bodied entity passes "exists" while failing "means anything," leaving live
pointers resolving to hollow shells. This is not hypothetical: the perf-backlog
gaps `G-0322`–`G-0325` were filed body-less, and **`M-0219`'s Goal already
references `G-0322`** — a milestone pointing at an empty gap. Their rationale
lived only in M-0216's §Validation, which archives when the epic closes, so they
were on track
to **outlive their own justification**. The empty bodies were caught by a human
`wf-rethink` reviewer at wrap, not by the kernel — converting a durability
guarantee into operator vigilance, exactly the dependency CLAUDE.md's "framework
correctness must not depend on the LLM's behavior" forbids.

## Proposed fix shape

- **Verb-time gate (primary).** `aiwf add {gap,decision,adr,contract}` refuses to
  create the entity when a required body section is empty (all-whitespace /
  headings-only). It must live in the verb, not only the check, because (a)
  creation is the sole safe point on trunk, and (b) it must not depend on whether
  the verb's programmatic commit triggers the pre-commit hook. Same
  `--force --reason "..."` sovereign override the FSM gates already use.
- **Ergonomics (so capture stays fast).** Today `aiwf add` accepts body input
  only via `--body-file` (and stdin `-`), which is heavy mid-flow. Add an inline
  `--body "..."` flag (and/or an `$EDITOR` fallback that opens the kind template)
  so in-flow capture is one line: `aiwf add gap --title "X" --body "Y misses Z;
  matters because W"`. The title already carries the *what*; the gate just demands
  the two sentences of *why* you would write anyway.
- **Check-time backstop.** Escalate `entity-body-empty` to **error** for
  born-complete kinds (independent of the existing `tdd.strict` escalation), so a
  hand-authored file that bypasses the verb still surfaces the finding on every
  `aiwf check`. Forward-only; pre-rule body-less entities (e.g. `G-0322`–`G-0325`)
  are grandfathered until hand-filled.
- **Scope line.** Draft-bearing kinds keep their current behavior: milestones /
  epics and milestone ACs stay on the warning + draft-status suppression they
  already have. The new gate applies only to kinds with no draft phase.

## Related

- **G-0216** establishes the precedent pattern — verb-time refusal + check-time
  finding for empty AC bodies — but gates the milestone `draft → in_progress`
  *transition* for *AC sub-elements*. This gap generalizes that pattern to the
  *creation* of *born-complete root kinds*, where the chokepoint is the `add`
  verb rather than a later promote.
- Surfaced while recovering the empty bodies of `G-0322`–`G-0325` (E-0053 perf
  backlog); see M-0216 §Validation/§Deferrals for that rationale.
