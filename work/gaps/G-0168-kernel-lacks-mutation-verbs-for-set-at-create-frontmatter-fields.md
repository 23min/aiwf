---
id: G-0168
title: Kernel lacks mutation verbs for set-at-create frontmatter fields
status: open
priority: high
discovered_in: M-0125
---
## What's missing

These frontmatter fields are set at entity-creation time (via flags on
`aiwf add`) and have **no verb that changes one of them** afterwards:

| Kind | Field | Set at create via | Post-create verb |
|------|-------|-------------------|------------------|
| gap | `discovered_in:` | `aiwf add gap --discovered-in M-NNNN` | **none** |
| decision | `relates_to:` | `aiwf add decision --relates-to <ids>` | **none** |
| contract | `linked_adrs:` | `aiwf add contract --linked-adr ADR-NNNN` | **none** |

For comparison, fields that **do** have post-creation mutation verbs
(showing the kernel intends this to be the universal pattern):

| Kind | Field | Mutation verb |
|------|-------|---------------|
| (all) | `status` | `aiwf promote <id> <status>` / `aiwf cancel <id>` |
| AC | `tdd_phase` | `aiwf promote <id>/AC-N --phase <p>` |
| (all) | `title` (+ slug) | `aiwf retitle <id> "..."` |
| (all) | slug only | `aiwf rename <id> <new-slug>` |
| (all) | `id` | `aiwf reallocate <id>` |
| milestone | `parent` (epic) | `aiwf move <M-id> --epic <E-id>` |
| milestone | `tdd:` | `aiwf milestone tdd <M-id> --policy none\|advisory\|required` |
| milestone | `depends_on:` | `aiwf milestone depends-on <id> --on <ids>` |
| gap | `addressed_by:` | `aiwf promote <gap> addressed --by <id>` |
| ADR | `superseded_by:` | `aiwf promote <adr> superseded --superseded-by <id>` |
| milestone | `acs[]` | `aiwf add ac`, `aiwf promote <id>/AC-N` |

The one route that changes them in a trailered commit is
`aiwf import --on-collision update`, which replaces each entity a manifest
names with the manifest's frontmatter and body (`buildEntityFromEntry` and
`serializeFromEntry` in `internal/verb/import.go`). Its `--help` presents it as
bulk creation — "Bulk-create entities from a YAML/JSON manifest", with the
update example "replacing entities with explicit ids that already exist" — and
no skill names it as the way to change a field.

Reproduced 2026-10-31 with aiwf built from `c06ffdaf9` (`vcs.modified=false`),
git 2.51.1, Linux, in a fresh `aiwf init` repository with its hooks installed.
The tree holds `ADR-0001`, `ADR-0002`, `E-0001`, `E-0002`, a contract `C-0001`
linking `ADR-0001`, a gap `G-0001` discovered in `E-0001` and a decision `D-0001`
relating to `E-0001`. A manifest restates each of the contract, gap and decision whole,
frontmatter and body, with one field changed:

```
$ aiwf import manifest.json --on-collision update
aiwf import 3 entities
$ git log -1 --format=%B
aiwf import 3 entities
aiwf-verb: import
aiwf-entity: C-0001
aiwf-entity: G-0001
aiwf-entity: D-0001
aiwf-actor: human/peter
$ git diff HEAD~1 HEAD -- work/ | grep '^[-+][^-+]'
-    - ADR-0001
+    - ADR-0002
-    - E-0001
+    - E-0002
-discovered_in: E-0001
+discovered_in: E-0002
```

`aiwf check --since` the commit before the import printed `ok — no findings`.
A manifest that leaves the contract's body out writes the contract with an empty
body, and `aiwf check` reports `entity-body-section-dropped` at error for each
required section. A hand edit of `linked_adrs`, committed with
`aiwf-verb: edit-frontmatter`, is refused by the `commit-msg` hook (exit 1,
`commit-msg refuses aiwf-verb trailer value(s): ["edit-frontmatter"]`);
committed without a trailer it lands, and `aiwf check` reports
`provenance-untrailered-entity-commit` at error.

## Why it matters

An operator correcting one of these fields — a contract linking the wrong ADR, a
gap's discovery context — finds nothing in `aiwf --help`, shell completion
or a skill that says how, against the rule that kernel functionality is
AI-discoverable ([`.guidance/project/verb-design.md`](../../.guidance/project/verb-design.md);
[`docs/design/design-decisions.md`](../../docs/design/design-decisions.md)).

Each route that exists costs more than the one-field change. The import route takes a
manifest restating the whole entity, and leaving the body out empties it. The
plain edit an operator reaches for instead is refused at commit when it carries
an `aiwf-verb:` trailer naming no verb, and reported by `aiwf check` at error
when it carries none.

## Symptom that surfaced this

During the M-0125 wrap diligence audit, M-0120's
`tdd: advisory` needed changing to `tdd: none` (rationale: M-0120
ratifies the legal-workflow spec methodology in an ADR; its ACs predate
the very phase-tracking discipline they would be audited under — the
methodology M-0120 ratifies). With no `aiwf milestone tdd` verb then, the
operator (Claude Code agent) hand-edited the frontmatter and committed
with `aiwf-verb: edit-frontmatter` — a fictional verb name (no such
verb exists in source).

Commit: `0a782be7 chore(plan): set M-0120 tdd: advisory → none` on
`epic/E-0033-pin-legal-kernel-verb-workflows-mechanically`.

## Discovered in

M-0125/AC-2 wrap diligence audit. The M-0120
`tdd: advisory → none` need was the trigger; the audit of "what
frontmatter fields lack mutation verbs" surfaced the broader pattern.

## Related

- M-0136 (`aiwf acknowledge illegal` — established the pattern for
  "verb that addresses a historical-violation acknowledgment;" useful
  reference for the verb-design conventions).
- `aiwf milestone depends-on` — the closest existing precedent for
  the per-kind subverbs [D-0048](../decisions/D-0048-frontmatter-field-mutation-verbs-milestone-tdd-now-others-per-kind-deferred.md) chose for these fields.

## Downstream report (2026-06-12): FSM-coupled amend + the generic-verb fork

A downstream consumer's audit re-surfaced this gap from a wider angle.
Two refinements to record.

### The set-at-transition fields also lack an editor

The report flagged `gap.addressed_by` and `adr.superseded_by` as
"no post-creation editor" alongside `decision.relates_to`. Those two, and
the other fields an FSM transition writes, are tracked in G-0442, with the
measurements and the constraint on their editor. G-0168's scope is the
set-at-create fields: whatever verb lands for them leaves out the fields a
transition writes.

### Design fork: generic `aiwf relate` vs per-kind subverbs

The report proposed a single generic verb —
`aiwf relate <id> --field <name> --add/--set/--clear` — as an
alternative to per-kind subverbs (`aiwf decision relates-to`,
`aiwf gap discovered-in`, …). [D-0048](../decisions/D-0048-frontmatter-field-mutation-verbs-milestone-tdd-now-others-per-kind-deferred.md) settles the fork:
per-kind subverbs, not a `--field` multiplexer. G-0246 shares the
question for an ADR `relates_to` field.

Source: downstream consumer feedback, 2026-06-12.
