---
id: E-NNNN
title: <imperative title>
status: proposed         # aiwf epic statuses: proposed | active | done | cancelled
depends_on: []           # optional: prior epic ids; e.g. [E-0002]
completed:               # optional: YYYY-MM-DD, filled at wrap
---

# E-NNNN — <Epic Title>

## Goal

<1–2 sentences: what problem does this solve? What value does it deliver?>

## Context

<What exists today? Why is this work needed now? What prior epics does it build on?>

## Scope

### In scope

- <Feature or capability>

### Out of scope

- <Explicitly excluded item>

## Constraints

- <Technical invariant, non-negotiable rule, shim-policy exception with a named removal trigger>

## Success criteria

<!-- Observable outcomes at epic close — not tests. "Users can do X" or "system exhibits Y"
     rather than "feature X is implemented." Milestone ACs are testable; epic success is
     visible.

     Avoid hand-written list counts. When a criterion references a list defined elsewhere
     in this spec, phrase it as a reference, not a reproduced count.
       Bad:  "All 16 ADRs are merged."
       Good: "Every ADR listed in the *ADRs produced* table below is merged." -->

- [ ] <Measurable outcome — observable pass/fail at epic close>

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| <question> | <yes/no> | <where/when it gets resolved> |

## Risks (optional)

| Risk | Impact | Mitigation |
|---|---|---|
| <risk> | <high/med/low> | <plan or deferral> |

## Milestones

<!-- Bulleted list, ordered by execution sequence. Status is NOT carried here — it lives
     in each milestone's frontmatter. Update this list when milestones are added, renamed,
     or re-sequenced. Milestone ids are global (M-NNNN), not epic-scoped. -->

- [M-NNNN](work/epics/E-NNNN-<slug>/M-NNNN-<slug>.md) — <one-line description> · depends on: —
- [M-NNNN](work/epics/E-NNNN-<slug>/M-NNNN-<slug>.md) — <one-line description> · depends on: M-NNNN

## ADRs produced (optional)

<!-- ADRs ratified or written during this epic. Reference by id. -->

- ADR-NNNN — <title>

## References

- <Decision records, source paths, related epics, external docs>
