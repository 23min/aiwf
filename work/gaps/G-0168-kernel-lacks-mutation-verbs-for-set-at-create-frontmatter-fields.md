---
id: G-0168
title: Kernel lacks mutation verbs for set-at-create frontmatter fields
status: open
discovered_in: M-0125
---
## What's missing

Several frontmatter fields are set at entity-creation time (via flags on
`aiwf add`) but have **no post-creation mutation verb**. To change any
of them, an operator must hand-edit the markdown file and commit
manually — bypassing the kernel's "one verb per mutation, one commit per
verb" chokepoint convention.

Per-kind enumeration of the affected fields:

| Kind | Field | Set at create via | Post-create verb |
|------|-------|-------------------|------------------|
| milestone | `tdd:` | `aiwf add milestone --tdd required\|advisory\|none` | **none** |
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
| (all) | `id` | `aiwf reallocate <id>`, `aiwf rewidth` |
| milestone | `parent` (epic) | `aiwf move <M-id> --epic <E-id>` |
| milestone | `depends_on:` | `aiwf milestone depends-on <id> --on <ids>` |
| gap | `addressed_by:` | `aiwf promote <gap> addressed --by <id>` |
| ADR | `superseded_by:` | `aiwf promote <adr> superseded --superseded-by <id>` |
| milestone | `acs[]` | `aiwf add ac`, `aiwf promote <id>/AC-N` |

## Why it matters

The kernel's design ([`docs/pocv3/design/design-decisions.md`](docs/pocv3/design/design-decisions.md) §"Every mutating
verb produces exactly one git commit"; CLAUDE.md "Kernel functionality
must be AI-discoverable") commits to:

1. Every mutating verb produces exactly one git commit with proper
   `aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:` trailers.
2. Every kernel capability is reachable through `aiwf <verb> --help`
   and auto-completion — no field mutation should require grepping
   source.
3. `aiwf history <id>` reads git log; trailers make the log queryable.

Hand-editing a frontmatter field plus a manual `git commit` violates
all three:
- The trailer is fictional (no real verb name to record).
- An AI assistant can't discover the mutation path from `--help`.
- `aiwf history` shows a verb name that doesn't resolve to any actual
  kernel verb.

The convention violation also propagates: future readers of the entity's
history can't tell whether the change was a deliberate kernel-allowed
mutation or an ad-hoc hack.

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
| milestone | `aiwf milestone tdd <M-id> --policy none\|advisory\|required` |
| gap | `aiwf gap discovered-in <G-id> --on M-NNNN` (or `--clear`) |
| decision | `aiwf decision relates-to <D-id> --on <ids>` (or `--clear`) |
| contract | `aiwf contract linked-adrs <C-id> --on ADR-NNNN[,ADR-NNNN…]` (or `--clear`) |

Each:
- Validates the new value against the field's closed set (where
  applicable — e.g., `tdd:` ∈ {none, advisory, required}).
- Validates that referenced ids resolve (where applicable —
  `discovered_in:`, `relates_to:`, `linked_adrs:`).
- Emits standard kernel trailers (`aiwf-verb: <kind>-<field>`,
  `aiwf-entity: <id>`, `aiwf-actor: <actor>`).
- Requires `--reason "..."` for trail clarity (consistent with
  `aiwf milestone depends-on --reason`).
- Optional `--force` for sovereign override (rarely needed; the
  fields are all data-shape changes, not FSM transitions).

## Workaround (current)

Until the verbs exist, the operator hand-edits the YAML and commits
manually with a descriptive but **fictional** `aiwf-verb:` trailer
naming the field (e.g., `aiwf-verb: edit-frontmatter`,
`aiwf-verb: retdd`). The commit body explains the workaround and
links to this gap.

The M-0120 commit (`0a782be7`) follows this pattern. When G-0168 is
addressed, the workaround pattern can be retroactively cleaned up if
desired — but doesn't have to be, since the verb trailer is forward-
compatible (a future `aiwf milestone tdd` verb's check could recognize
`edit-frontmatter` as a legacy synonym).

## Closing this gap

When the verbs land:
1. Each verb's `--help` documents the field, the closed set (if any),
   and the validation rules.
2. The verbs participate in shell auto-completion (per CLAUDE.md
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
- G-0141 (Phase 2 — structured-code emission for verb errors; tangentially
  related, since `aiwf milestone tdd` would emit structured errors for
  invalid policy values).

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

There is no way to **amend, add to, or clear** either field after the
transition without hand-editing frontmatter — the same chokepoint
violation this gap is about, on fields the table marked "covered." So
the real split is set-at-create (this gap's four fields, no path at
all) vs set-at-transition (`addressed_by`, `superseded_by` — written
once at the FSM step, no amend afterward). Both want an editor; the
FSM-coupled pair needs one that does **not** let the relation field be
written independently of its transition.

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

## Re-discovery (2026-06-26): the `tdd:` field, from the upgrade direction

A fresh session hit this gap again — the opposite direction to the M-0120 case
above: *strengthening* a milestone from `tdd: advisory` to `tdd: required` after
establishing its ACs were genuinely testable. Same missing verb, but the upgrade
direction surfaced two things the original downgrade case did not.

### The two directions are not symmetric — `tdd:` has directional integrity semantics

- **Upgrade** (`advisory → required`, `none → advisory`) *strengthens* the
  "met requires `tdd_phase: done`" gate. A normal, desirable mid-flight decision;
  it should be frictionless.
- **Downgrade** (`required → advisory|none`) *weakens* that gate — the class of
  act aiwf elsewhere routes through a sovereign `--reason` (and human actor) path
  (`acknowledge-illegal`, `--force`).

This sharpens the verb shape proposed above: the `--reason` should not be uniform
"trail clarity" (as the table suggests for all four fields) but **directional** —
required on the weakening direction, optional on the strengthening one. It also
sharpens the inverse-coverage-policy gap filed on the `epic/E-0044` worktree (its
id is unsettled cross-branch until that epic merges): that registry classifies an
inverse by *existence* (A/B/C/D), but `tdd:` shows a verb can need a fifth
property — *the inverse exists but is governance-weighted*. The undo of
"set tdd required" is "set tdd advisory", and that undo is the gated one.

### The friction has a deeper root than the missing verb — an over-strict check rule

Even with a perfect verb, the upgrade breaks the tree today. ACs added under an
`advisory` milestone carry no `tdd_phase`; the moment the milestone becomes
`required`, `internal/check/acs.go` (~line 153, `ac.TDDPhase == "" && tddRequired`)
errors `acs-shape/tdd-phase` on **every** phaseless AC, regardless of status. The
phase enum is `red|green|refactor|done`, with no "not started" member — so the
only way to clear the tree is to seed untouched ACs to `red`, which *lies*
(`red` means "failing test written").

But the design commits (CLAUDE.md #8) only to **"AC `met` requires
`tdd_phase: done`"** — not to "every AC has a phase." So `acs.go:153` is
**stricter than the commitment**; it over-demands. The cleaner fix is therefore
two-part, and the check half may matter more than the verb:

- **Relax `acs-shape/tdd-phase`** so an absent phase is legal until an AC is `met`
  (absent = "not started"). Then `advisory → required` is non-disruptive, the
  verb's "auto-seed to `red` vs refuse" question *evaporates*, and the real gate
  (met → done) is untouched — bringing the check back in line with #8.
- **Add the verb** for the trailered policy flip.

If the strict reading is instead kept deliberately (`required` means "every AC is
phase-tracked from creation"), the verb must **refuse with an actionable hint**
naming the ACs to seed — never auto-seed to `red`, which manufactures false state.

### Prerequisites for closing (the rest of today's verb-surface cluster)

When the `tdd:` verb lands as a `milestone` subverb, two sibling gaps gate its
discoverability and coverage:

- **G-0285** (root `--help` banner drift) — the banner omits the entire
  `milestone` namespace today, so the new subverb would be invisible there until
  G-0285's chokepoint lands. This same omission is *why* the verb felt missing:
  `aiwf milestone depends-on` already ships but the banner hides it.
- **G-0284** (skill-coverage subverb blind spot) — namespace subverbs escape the
  skill-coverage policy today, so the new verb could ship uncovered; G-0284 names
  exactly this class.

Verb-name convergence: the re-discovery proposed `aiwf milestone set-tdd`; the
table above proposes `aiwf milestone tdd --policy <…>`. Pick one shape when
implementing.

Source: design session, 2026-06-26.
