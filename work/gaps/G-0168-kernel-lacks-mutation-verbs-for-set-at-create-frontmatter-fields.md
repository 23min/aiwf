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

During the M-0125 wrap diligence audit (this session), M-0120's
`tdd: advisory` needed changing to `tdd: none` (rationale: M-0120
ratifies the legal-workflow spec methodology in an ADR; its ACs predate
the very phase-tracking discipline they would be audited under — the
methodology M-0120 ratifies). With no `aiwf milestone tdd` verb, the
operator (Claude Code agent) hand-edited the frontmatter and committed
with `aiwf-verb: edit-frontmatter` — a fictional verb name (no such
verb exists in source).

Commit: `0a782be7 chore(plan): set M-0120 tdd: advisory → none` on
`epic/E-0033-pin-legal-kernel-verb-workflows-mechanically`.

## Proposed verb shapes

The kernel already establishes the **subcommand-per-kind pattern** via
`aiwf milestone depends-on`. Following that idiom:

| Kind | Verb shape |
|------|------------|
| gap | `aiwf gap discovered-in <G-id> --on M-NNNN` (or `--clear`) |
| decision | `aiwf decision relates-to <D-id> --on <ids>` (or `--clear`) |
| contract | `aiwf contract linked-adrs <C-id> --on ADR-NNNN[,ADR-NNNN…]` (or `--clear`) |

Each:
- Validates that referenced ids resolve (where applicable —
  `discovered_in:`, `relates_to:`, `linked_adrs:`).
- Emits standard kernel trailers (`aiwf-verb: <kind>-<field>`,
  `aiwf-entity: <id>`, `aiwf-actor: <actor>`).
- Requires `--reason "..."` for trail clarity (consistent with
  `aiwf milestone depends-on --reason`).
- Optional `--force` for sovereign override (rarely needed; the
  fields are all data-shape changes, not FSM transitions).

## Closing this gap

When the verbs land:
1. Each verb's `--help` documents the field, the closed set (if any),
   and the validation rules.
2. The verbs participate in shell auto-completion (per
   [`.guidance/project/verb-design.md`](../../.guidance/project/verb-design.md),
   "CLI surfaces must be auto-completion-friendly").
3. Skills cover each verb per ADR-0006 (per-verb skill default, OR
   topical skill if multiple verbs cluster — `aiwf-milestone-*` and
   `aiwf-contract-*` are natural topical bundles).
4. The drift policy (`internal/policies/skill_coverage.go`) catches
   any verb landing without skill coverage.
5. Promote G-0168 to `addressed` with `--by M-NNNN` (whichever
   milestone(s) land the verbs).

## Discovered in

M-0125/AC-2 wrap diligence audit (this session). The M-0120
`tdd: advisory → none` need was the trigger; the audit of "what
frontmatter fields lack mutation verbs" surfaced the broader pattern.

## Related

- M-0136 (`aiwf acknowledge-illegal` — established the pattern for
  "verb that addresses a historical-violation acknowledgment;" useful
  reference for the verb-design conventions).
- `aiwf milestone depends-on` — the closest existing precedent for
  the subcommand pattern this gap proposes for other fields.

## Downstream report (2026-06-12): FSM-coupled amend + the generic-verb fork

A downstream consumer's audit re-surfaced this gap from a wider angle.
Two refinements to record.

### The set-at-transition fields also lack an editor

The report flagged `gap.addressed_by` and `adr.superseded_by` as
"no post-creation editor" alongside `decision.relates_to`. The first
two are not in the "missing" table above because they *do* have a set
path — but only as a **side effect of the FSM transition** that writes
them:

- `aiwf promote <gap> addressed --by <id>` sets `addressed_by` at the
  `open → addressed` step.
- `aiwf promote <adr> superseded --superseded-by <id>` sets
  `superseded_by` at the `accepted → superseded` step.

No verb **amends, adds to, or clears** either field after the
transition — the same missing verb this gap is about, on fields the
table marked "covered." `aiwf import --on-collision update` rewrites them
as it rewrites the fields above: on 2026-10-31 it changed an `addressed`
gap's `addressed_by` with the status unchanged, and `aiwf check` reported
no error. So the real split is set-at-create (this gap's fields) vs
set-at-transition (`addressed_by`, `superseded_by` — written
once at the FSM step, no amend afterward). Both want an editor; the
FSM-coupled pair needs one that does **not** let the relation field be
written independently of its transition.

The set-at-transition pair is now tracked as its own gap, G-0442 —
G-0168's scope is bounded to the set-at-create fields. Whatever verb
lands for the set-at-create relation fields excludes `addressed_by` /
`superseded_by`; their editor is a distinct problem, constrained to
amend/clear on an already-transitioned entity (never an independent set).

### Design fork: generic `aiwf relate` vs per-kind subverbs

The report proposed a single generic verb —
`aiwf relate <id> --field <name> --add/--set/--clear` — as an
alternative to the per-kind subverbs this gap proposes
(`aiwf decision relates-to`, `aiwf gap discovered-in`, …). The fork is
genuine and should be settled before any verb lands:

- **Per-kind subverbs** (this gap's shape) mirror the established
  `aiwf milestone depends-on` idiom and can stay FSM-aware — the
  gap/adr verbs can refuse to set the coupled field out of band and
  only `--add`/`--clear` an already-transitioned entity.
- **Generic `aiwf relate`** is more uniform but risks becoming a
  blessed "edit any relation field" escape hatch. Critically, it would
  let an operator set `superseded_by` *without* the
  `accepted → superseded` transition — re-introducing exactly the
  inconsistent state the FSM back-edge was designed to prevent.

Lean: per-kind subverbs (or a field-aware generic verb that hard-refuses
FSM-coupled fields out of band). The decision is worth an
`aiwfx-record-decision` before the implementing milestone, and it pairs
with the ADR `relates_to` schema question filed as its sibling gap.

Source: downstream consumer feedback, 2026-06-12.
