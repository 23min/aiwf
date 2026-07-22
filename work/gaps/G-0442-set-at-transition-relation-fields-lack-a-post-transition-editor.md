---
id: G-0442
title: Set-at-transition relation fields lack a post-transition editor
status: open
priority: medium
---
## What's missing

Two frontmatter fields are set only as a **side effect of an FSM
transition**, never by a standalone editor:

| Kind | Field | Set by | Amend / clear afterward? |
|------|-------|--------|--------------------------|
| gap | `addressed_by` | `aiwf promote <gap> addressed --by <id>` | **none** |
| adr | `superseded_by` | `aiwf promote <adr> superseded --superseded-by <id>` | **none** |

Each is written once, at the `open → addressed` / `accepted → superseded`
step. There is no verb to **amend, add to, or clear** either field
afterward — to credit a second resolver on an already-addressed gap, or
correct a `superseded_by` pointer, an operator must hand-edit the YAML and
commit manually.

## Why it matters

This is the same chokepoint violation G-0168 is about, on fields G-0168's
table marked "covered." They have a *set* path but no *amend* path, so the
"covered" marking is misleading. A hand-edit trips
`provenance-untrailered-entity-commit`, records a fictional `aiwf-verb:`
trailer, and leaves `aiwf history` naming a verb that resolves to nothing
— the three failures G-0168 enumerates.

## Why this is distinct from G-0168 — the FSM-coupling constraint

G-0168's four fields are **set-at-create** (no mutation path at all).
These two are **set-at-transition**: written once at an FSM step, and the
amend verb must **not** let the field be written *independently of that
transition*. Setting `superseded_by` without the `accepted → superseded`
step would re-introduce exactly the inconsistent state the FSM back-edge
prevents. So whatever verb G-0168 lands for its set-at-create relation
fields deliberately excludes these two; their editor is a separate problem
that must require the entity already be in the terminal state owning the
field — amend or clear only, never an independent set.

## Related

- G-0168 — the set-at-create half. This gap is the set-at-transition half,
  split out of G-0168's downstream-report section so each has a bounded
  scope.
- G-0246 — ADRs lack a general `relates_to` field; shares the underlying
  relation-model design question.
