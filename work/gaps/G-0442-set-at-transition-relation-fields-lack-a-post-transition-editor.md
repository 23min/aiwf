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
| gap | `addressed_by` | `aiwf promote <gap> addressed --by <id>` | re-point or add only with `--force --reason`; **no clear** |
| adr | `superseded_by` | `aiwf promote <adr> superseded --superseded-by <id>` | re-point only with `--force --reason`; **no clear** |

Each is written at the `open → addressed` / `accepted → superseded` step.
After it, `aiwf promote` writes the field again without `--force` only to
fill it while it is empty. To re-point it, or to credit a second resolver on
an already-addressed gap, the verb refuses (exit 1) unless given
`--force --reason`, and no verb clears either field.

Reproduced 2026-10-31 with aiwf built from `c06ffdaf9` (`vcs.modified=false`),
git 2.51.1, Linux, in a fresh `aiwf init` repository holding `ADR-0001`,
`ADR-0002`, `ADR-0003`, `E-0001`, `E-0002` and a gap `G-0001`:

```
$ aiwf promote G-0001 addressed --by E-0001
aiwf promote G-0001 open -> addressed
$ aiwf promote G-0001 addressed --by E-0002
aiwf promote: G-0001 is already addressed and already carries a resolver; this verb backfills an empty resolver, it does not re-point one — pass --force --reason to override
$ aiwf promote G-0001 addressed --by E-0002 --force --reason 're-point the resolver'
aiwf promote G-0001 addressed -> addressed
$ aiwf promote G-0001 addressed --by E-0001,E-0002 --force --reason 'credit a second resolver'
aiwf promote G-0001 addressed -> addressed
$ aiwf promote G-0001 addressed --by ''
G-0001 is already addressed; nothing to change
$ aiwf promote G-0001 addressed --clear
aiwf: unknown flag: --clear
$ aiwf promote ADR-0001 accepted
aiwf promote ADR-0001 proposed -> accepted
$ aiwf promote ADR-0001 superseded --superseded-by ADR-0002
aiwf promote ADR-0001 accepted -> superseded
$ aiwf promote ADR-0001 superseded --superseded-by ADR-0003
aiwf promote: ADR-0001 is already superseded and already carries a resolver; this verb backfills an empty resolver, it does not re-point one — pass --force --reason to override
$ aiwf promote ADR-0001 superseded --superseded-by ADR-0003 --force --reason 'correct the pointer'
aiwf promote ADR-0001 superseded -> superseded
$ aiwf promote ADR-0001 superseded --superseded-by ''
ADR-0001 is already superseded; nothing to change
```

The run also printed the field after several steps, cut above: after the forced
re-point `addressed_by` held `E-0002`, after the forced add `E-0001` and
`E-0002`, and the forced re-point left `superseded_by: ADR-0003`. A second gap,
promoted to `addressed` with `--force` and no resolver, took one from a later
`aiwf promote … addressed --by E-0001` without `--force`.

`aiwf import --on-collision update`, which replaces an entity whole (G-0168),
does without `--force` what the verb refuses. In the same repository, an update
restating `ADR-0001` with `superseded_by: ADR-0002` re-pointed it, and one
restating `ADR-0001` and `G-0001` without either field dropped both, the status
unchanged each time. After each, `aiwf check --since` the commit before it
reported no error; after the second, a `gap-addressed-has-resolver` warning.

## Why it matters

Correcting a resolver or supersession pointer that names the wrong entity is an
ordinary fix, but `aiwf promote` takes it only as an override, with
`--force --reason`, and removing a pointer that should not be there has no verb.
The one route that needs no `--force`, an import update, re-points or drops
either field with no error from `aiwf check`, so the override the verb asks for
is skipped by the route an operator is left with.

## Why this is distinct from G-0168 — the FSM-coupling constraint

G-0168's fields are **set-at-create**.
These two are **set-at-transition**: written at an FSM step, and the
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
