---
id: G-0712
title: Verbs re-emit entity ids at the width the operator typed
status: open
---
## What's missing

ADR-0008 §Parser tolerance states that "the kernel never re-emits narrow-width ids —
it only accepts them on input". The verbs do re-emit them: an id handed to a verb is
resolved against the tree — a narrow spelling finds its entity through read
tolerance — and then written at the width the operator typed.

- **Frontmatter references.** `applyAddOpts` (`internal/verb/add.go`) stores
  `--epic`, `--depends-on`, `--discovered-in` and `--relates-to` as typed; `aiwf move`
  stores `parent` as typed (`internal/verb/move.go`), as does `aiwf milestone
  depends-on` for `depends_on` (`internal/verb/milestone_depends_on.go`) and
  `applyResolverFlags` (`internal/verb/promote.go`) for `promote --by`'s
  `addressed_by`. `aiwf import` copies `depends_on`, `discovered_in` and `relates_to`
  from the manifest as written; it resolves only `id` and `parent`
  (`buildEntityFromEntry`, `internal/verb/import.go`).
- **The contract binding in `aiwf.yaml`.** `ContractBind`
  (`internal/verb/contractbind.go`) writes the binding's `id` as typed, and the
  `contract-config` finding then carries that spelling as its `entity_id` in
  `aiwf check --format=json`, a surface ADR-0008 §Renderer canonicalization requires
  to be canonical.
- **Commit subjects.** `aiwf promote`, `aiwf cancel` and `aiwf retitle`, at entity and
  acceptance-criterion granularity, and `aiwf contract bind` and `unbind`, put the
  typed id in the subject; their `aiwf-entity:` trailer is canonical. The
  `--audit-only` subjects in `internal/verb/auditonly.go` format the id the same way
  (read, not run). `aiwf add`, `add ac`, `move`, `rename`, `reallocate`, `set-area`,
  `set-priority`, `authorize`, `edit-body` and `milestone depends-on` write canonical
  subjects.

Three code comments present the verbatim write as the convention and assign width
normalization to `aiwf rewidth`, a verb ADR-0039 retired: the `DependsOn`
assignment in `internal/verb/milestone_depends_on.go`, `promoteWouldWrite`'s doc
comment in `internal/verb/promote.go`, and the depends-on option comment in
`internal/verb/add.go`.

Out of this defect: `prior_ids` records the id an entity carried before, so its width
is history rather than a reference; and ADR ids cannot be narrow, since their grammar
starts at four digits, so `--linked-adr` and `--superseded-by` refuse a narrow ADR id.

No check reports a narrow reference: `entity-id-narrow-width` judges only an entity's
own id, and `body-prose-id` only body prose.

Measured on Linux in bash, with an `aiwf` binary built from `5795b9530`, in fresh
scratch repositories on branch `main` after `aiwf init --skip-hook` with `hosts: []`:

    $ aiwf add epic --title One; aiwf add epic --title Two          -> E-0001, E-0002
    $ aiwf add milestone --epic E-01 --tdd none --title First       -> M-0001
    $ aiwf add milestone --epic E-0001 --tdd none --depends-on M-001 --title Second
    $ aiwf add gap --title "Probe gap" --discovered-in M-001 --body …
    $ aiwf add decision --title "Probe decision" --relates-to E-01 --body …
    $ aiwf move M-0002 --epic E-02
    $ aiwf promote G-0001 addressed --by M-001
    $ aiwf contract bind C-001 --validator cue --schema schemas/r.cue --fixtures fixtures/r
    stored:  M-0001  parent: E-01
             M-0002  parent: E-02, depends_on: [M-001]
             G-0001  discovered_in: M-001, addressed_by: [M-001]
             D-0001  relates_to: [E-01]
             aiwf.yaml contracts.entries:  - id: C-001
    subjects [aiwf-entity trailer]:
             aiwf contract bind C-001                        [C-0001]
             aiwf promote E-01 proposed -> active            [E-0001]
             aiwf retitle M-001 -> "First renamed"           [M-0001]
             aiwf retitle M-001/AC-1 -> "…"                  [M-0001/AC-1]
             aiwf promote M-001/AC-1 open -> met             [M-0001/AC-1]
             aiwf cancel M-001/AC-2 -> cancelled             [M-0001/AC-2]
             aiwf cancel D-001 -> rejected                   [D-0001]
             aiwf contract unbind C-001                      [C-0001]
    $ aiwf check --format=json
             no finding about a narrow id; contract-config finding: entity_id "C-001"
    $ grep -rl E-0002 work
             work/epics/E-0002-two/epic.md
    $ aiwf list --kind milestone --parent E-0002
             M-0002  draft  Second  E-0002

    $ aiwf import m.yaml    (milestone M-0002 declares depends_on: [M-001])
    stored:  M-0002  depends_on: [M-001]

Expected, for every measurement above — each names a referent whose own id is
canonical — the canonical id. A reference to an archived entity that still carries
a legacy-width id is a separate case the tree already argues both ways:
`buildEntityFromEntry` resolves `parent` to the referent's stored spelling so the
pointer stays in step with it, and ADR-0039 keeps legacy-width archived entities
permanently readable.

## Why it matters

A plain-text search for an entity's id misses what references it. In the
measurement, `aiwf list` puts M-0002 under E-0002, but `grep -rl E-0002 work` finds
only the epic's own file, because M-0002 records its parent as `E-02`. A person or
an assistant searching the tree with grep or an editor gets an incomplete answer,
and nothing tells them so.

Narrow spellings accumulate unreported. Every reader inside aiwf resolves through a
canonicalizing lookup, so no verb fails and no check fires; each narrow reference
stays in frontmatter, in the `aiwf.yaml` the consumer commits, and in commit
subjects that no later edit rewrites. The JSON output of `aiwf check` hands one
straight to any tool that consumes it.

The records that would stop a reader acting on this are wrong. The comments in
`add.go`, `promote.go` and `milestone_depends_on.go` make the behaviour read as
deliberate by pointing at a verb that no longer exists. G-0505 declined the
`aiwf import` case with the reason "nothing fails — every consumer resolves the four
reference fields through a canonicalizing lookup, and import is the only route that
reaches them": `add`, `move`, `milestone depends-on` and `promote --by` reach the
same fields, and a plain-text search is a consumer that does not canonicalize.

## Related

- G-0505 — the `aiwf import` case, closed wontfix and archived.
- G-0707 — narrow ids in the text aiwf prints.
- ADR-0039 — retired `aiwf rewidth`.
