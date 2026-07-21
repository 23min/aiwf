---
name: aiwf-add
description: Use when the user wants to add a new aiwf entity (epic, milestone, ADR, gap, decision, or contract) or an acceptance criterion under an existing milestone, or to declare a milestone's `depends_on` edges. Runs `aiwf add` (or `aiwf milestone depends-on` for post-allocation edits) so the id allocation, frontmatter, and commit happen mechanically.
---

# aiwf-add

The `aiwf add` verb creates a new entity (or acceptance criterion) and produces exactly one git commit. Skills like this one are advisory; the binary is authoritative.

## When to use

The user wants to record a new piece of planning state — a new epic, a milestone under an existing epic, an ADR, a discovered gap, a decision, a contract, or an acceptance criterion (AC) under an existing milestone.

## What to run

```bash
aiwf add <kind> --title "<title>" [kind-specific flags]
aiwf add ac <milestone-id> --title "<title>"
```

The six kinds and their required flags:

| Kind | Required flags | Notes |
|---|---|---|
| epic | `--title` | Allocates `E-NN`. |
| milestone | `--title`, `--epic <E-id>`, `--tdd <required\|advisory\|none>` | Lives under the epic's directory. Optional `--depends-on M-PPP[,M-QQQ]` declares prerequisite milestones at allocation time; each id must already exist as a milestone. |
| adr | `--title` | Allocates `ADR-NNNN` under `docs/adr/`. |
| gap | `--title` | Optional `--discovered-in <id>`. Optional `--priority <level>` (`urgent`\|`high`\|`medium`\|`low`) sets the triage priority at creation — see the `aiwf-set-priority` skill to change or clear it later. |
| decision | `--title` | Optional `--relates-to <id,id,...>`. Optional `--priority <level>` (`urgent`\|`high`\|`medium`\|`low`) — same as gap. |
| contract | `--title` | Allocates `C-NNN` and creates `work/contracts/C-NNN-<slug>/contract.md`. Optional `--linked-adr <id,id,...>` records the motivating ADRs. Pass `--validator <name> --schema <path> --fixtures <path>` together to also bind the contract in aiwf.yaml within the same commit. |
| ac | `--title`, positional milestone id | Allocates `AC-N` per-milestone (max+1 across the full `acs[]` including cancelled). Appends to the milestone's frontmatter `acs[]` and scaffolds a `### AC-N — <title>` body heading. The milestone file is rewritten in place — no separate AC file. |

## Repeated --title for batched AC creation

`aiwf add ac M-NNN` accepts repeated `--title` flags to create N acceptance criteria in one atomic commit. Each title gets a consecutive AC id (`AC-X..AC-Y`); the commit's `aiwf-entity:` trailer set carries every created composite id, so `aiwf history M-NNN/AC-X` finds the batch commit for any AC in the batch.

```bash
aiwf add ac M-001 \
  --title "first criterion" \
  --title "second criterion" \
  --title "third criterion"
```

Atomic-on-failure: if any title is empty, prosey, or otherwise rejected, the entire batch aborts before disk work — no partial-batch commit. `--tests` is rejected when N>1 (a single test-metrics value can't apply unambiguously to multiple ACs); seed test metrics one AC at a time when needed.

Single `--title` still works exactly as before — same subject shape, same single `aiwf-entity:` trailer.

## --body / --body-file for ride-along body content

By default, `aiwf add` lands a per-kind body template (e.g., `## Goal`, `## Scope` headings on an epic). To replace that with real body prose in the same atomic create commit — and avoid a follow-up untrailered hand-edit that triggers `provenance-untrailered-entity-commit` — pass either `--body "<text>"` (inline, for short prose captured mid-flow) or `--body-file <path>` (or `--body-file -` to read from stdin):

```bash
aiwf add gap --title "Validators leak temp files" --body-file gap-body.md
echo "## Goal\n\nFleshed out goal." | aiwf add epic --title "Caching" --body-file -
aiwf add gap --title "Retry loop spins forever" --body "## What's missing

Retrying a failed fetch has no backoff cap; a persistent network
partition spins the retry loop indefinitely.

## Why it matters

The process never exits and pins its CPU core until someone notices
and kills it by hand."
```

`--body` and `--body-file` are mutually exclusive — passing both is a usage error (exit 2). Valid for all six kinds (epic, milestone, adr, gap, decision, contract). For acceptance criteria the flag exists too but with positional-pairing semantics — see *"--body-file for AC body scaffolding"* below (`--body` has no AC equivalent there; use `--body-file`). Body content from either flag must not begin with a leading `---` (YAML frontmatter delimiter) — refused with a clear error rather than silently stripped, so the create commit can't accidentally produce a double-frontmatter file.

## --body-file for AC body scaffolding (positional pairing)

`aiwf add ac M-NNN` accepts `--body-file <path>` (repeatable) so each AC's body content lands in the same atomic create commit as its frontmatter and `### AC-N — <title>` heading. Pairing is positional: the Nth `--body-file` populates the body of the Nth `--title`.

```bash
# Single AC with body content from a file
aiwf add ac M-001 --title "Rejects malformed YAML" --body-file ac1-body.md

# Multi-AC, positional pairing — one --body-file per --title, equal counts required
aiwf add ac M-001 \
  --title "Rejects malformed YAML"   --body-file ac1-body.md \
  --title "Reports the offending line" --body-file ac2-body.md

# Stdin shorthand — only valid with exactly one --title
echo "Concrete pass criteria..." | aiwf add ac M-001 --title "Matches semver" --body-file -
```

Same leading-`---` rejection as the whole-entity flag. AC-specific rules:

- **Equal counts required.** When `--body-file` is provided at all, the count of `--body-file` flags must equal the count of `--title` flags. Mismatched counts refuse pre-allocation with exit 2 — the verb does not partially populate.
- **Stdin only with single --title.** `--body-file -` is valid only when exactly one `--title` is given; stdin is one stream and cannot be split positionally. Multi-title invocations using `-` refuse before any read so the operator's piped input isn't consumed on a doomed call.
- **Omitting --body-file is valid** (any count of `--title`). The bare `### AC-N — <title>` heading is still scaffolded with no body content under it; the `entity-body-empty` finding from `aiwf check` is the chokepoint that surfaces the empty-body case at validation time. The body is not optional in the long run, but the friction-reducing flag is opt-in at create time so multi-AC quick-scaffold flows still work.

## What aiwf does

1. Allocates the next free id by scanning the working tree and the configured trunk ref (default `refs/remotes/origin/main`; override via `aiwf.yaml: allocate.trunk`). For ACs the scan is the milestone's `acs[]`. The trunk read is silently skipped when the repo has no remotes configured; an explicitly-configured trunk ref that doesn't resolve is a hard error so the operator notices.
2. Writes the new entity file with proper frontmatter (`id`, `title`, `status` set to the kind's initial status). For ACs, appends to the parent milestone's `acs[]` and scaffolds the body heading.
3. When the parent milestone is `tdd: required`, an AC is seeded with `tdd_phase: red` — the only legal entry phase under the FSM. Otherwise `tdd_phase` is left absent.
4. Validates the projected tree before touching disk; if a finding would be introduced, aborts with no changes.
5. Creates one commit carrying `aiwf-verb: add`, `aiwf-entity: <id>` (composite `M-NNN/AC-N` for ACs), `aiwf-actor: <actor>` trailers. When the operator is non-human (`ai/<id>`, `bot/<id>`), the kernel additionally requires a `--principal human/<id>` flag and stamps `aiwf-principal:` on the commit. If an active authorization scope (see `aiwf-authorize`) covers the new entity's parent / references, `aiwf-on-behalf-of:` and `aiwf-authorized-by:` are added too.
6. **Refuses an empty body for born-complete kinds; scaffolds a placeholder for the rest.** `gap`, `decision`, `adr`, and `contract` have no draft phase — they're live and referenceable the instant this commit lands — so `aiwf add` refuses to create one whose load-bearing `## <Section>` content is empty, unless `--force --reason "<text>"` overrides it (see *"Empty-body gate for born-complete kinds"* below). Epic, milestone, and AC bodies keep the placeholder-then-fill workflow: `aiwf check` reports `entity-body-empty` for any of those sections that ships empty (warning by default; error under `aiwf.yaml: tdd.strict: true`). Fill the body before declaring the entity complete — see *"After `aiwf add <kind>`: fill in the body"* below.

## Empty-body gate for born-complete kinds (`gap`, `decision`, `adr`, `contract`)

`gap`, `decision`, `adr`, and `contract` have no draft phase in their FSM — the entity is live and referenceable the moment the create commit lands, so an empty body is never "by design" the way a freshly-scaffolded milestone's is. `aiwf add` refuses to create one of these four kinds when a required top-level body section (`## <Section>` per the table in *"After `aiwf add <kind>`: fill in the body"* below) is empty: no content, all-whitespace, headings-only, or only an HTML comment.

```
$ aiwf add gap --title "Retry loop spins forever"
aiwf add: G-NNNN: empty load-bearing body section(s) `## What's missing`, `## Why it matters`; a gap is referenceable the instant this commit lands, so its body must carry meaning at creation — pass --body "..." or --body-file <path> with real prose, or --force --reason "..." to create anyway (aiwf check will still flag it at error severity and the pre-push hook will still block until it's filled in)
```

Fix it by passing real prose via `--body "<text>"` or `--body-file <path>`, or create it anyway with `--force --reason "<one-sentence justification>"` — the same sovereign-override shape `aiwf promote --force --reason` uses; `--reason` must be non-empty (after trim) whenever `--force` is set, and the override lands an `aiwf-force: <reason>` trailer on the create commit (only when the gate was actually bypassed — a no-op `--force` on epic/milestone, or on a born-complete kind whose body was already non-empty, stamps no trailer). `--force` is not a full override: the entity still lands with an empty body, which the check-time backstop below still catches. Epic, milestone, and `aiwf add ac` are unaffected — they keep the placeholder-then-fill workflow (see *"After `aiwf add <kind>`: fill in the body"* below).

`aiwf check`'s `entity-body-empty` rule is the backstop for hand-authored files or `--force`-created entities that bypass the verb gate: for these four kinds it always fires at **error** severity, independent of `aiwf.yaml: tdd.strict`, so a hollow gap/decision/adr/contract that slips past the verb still blocks the pre-push hook.

## Allocating ids across branches and clones

Allocation scans the working tree, every local branch, every remote-tracking
ref, and the configured trunk ref, so *where* you allocate rarely matters — but
a few operating rules keep ids collision-free across a team:

- **One machine, multiple worktrees:** nothing to do — sibling worktrees share
  an object store, so the allocator already sees their ids.
- **Separate clones:** run `aiwf add --fetch` (a best-effort `git fetch --all`)
  so a teammate's id on any *pushed* branch is seen and skipped, and
  **push promptly after `aiwf add`** so your new id reaches others' next fetch.
- **Allocate on whatever branch you're working on** — you don't need to create
  entities on trunk to avoid collisions.

What to expect:

- A peer who allocated but has **not pushed** is invisible; if two of you take
  the same id before either pushes, you collide — resolve with `aiwf reallocate`
  (see the `aiwf-reallocate` skill). Prompt pushing shrinks that window.
- A `--fetch` failure (offline, unreachable remote) never blocks the add: it
  warns and allocates against the local view.
- An entity on an unmerged branch can't be referenced **by id in another
  branch's prose** until it reaches trunk (the `body-prose-id` check resolves
  only against the working tree and trunk). Backtick the id until then, or
  allocate on the trunk branch if a parallel branch must reference it now.

## Milestone `depends_on`: declare DAG edges via verb

Milestone-to-milestone dependencies live in the `depends_on:` frontmatter array. Two writer surfaces, both producing one atomic commit with `aiwf-verb` trailers:

```bash
# At allocation time: pass --depends-on
aiwf add milestone --epic E-01 --tdd required \
  --title "Bootstrap" --depends-on M-001,M-002

# Post-allocation: dedicated verb
aiwf milestone depends-on M-NNN --on M-PPP[,M-QQQ]

# Empty the list
aiwf milestone depends-on M-NNN --clear
```

Replace-not-append semantics: a second `--on` invocation replaces the list, it does not extend. To add a single dependency to an existing list, the operator passes the full updated list. `--on` and `--clear` are mutually exclusive.

Each id passed to `--depends-on` or `--on` must resolve to an existing milestone before the verb commits — typos and pre-allocation references are refused with an error naming the unresolvable id. Cycle detection happens at the next `aiwf check` (and pre-push hook); the writers don't pre-check global DAG validity. Cross-kind dependencies (e.g. milestone depends on ADR) are out of scope today; a gap captures the generalization if the friction earns it.

Don't hand-edit `depends_on:` directly — `aiwf edit-body` refuses frontmatter changes, and a plain `git commit` against the milestone file triggers `provenance-untrailered-entity-commit`. Both writer verbs above leave a trailered commit `aiwf history M-NNN` can render.

## After `aiwf add <kind>`: fill in the body

`aiwf add` is step 1 of 2. The verb writes correct frontmatter and an atomic create commit; the body prose under each `## <Section>` heading is **required, not optional**, across all six top-level kinds and ACs. The kernel doesn't fail closed on missing prose at create time so the verb stays cheap, but `aiwf check` surfaces empty bodies as `entity-body-empty` findings, and any milestone or epic or AC with a hollow body is half-shipped.

The load-bearing body sections per kind:

| Kind | Required body sections |
|---|---|
| epic | `## Goal`, `## Scope`, `## Out of scope` |
| milestone | `## Goal`, `## Approach`, `## Acceptance criteria` |
| ac | The `### AC-N — <title>` body (one paragraph covering pass criteria, edge cases, and code references) |
| gap | `## What's missing`, `## Why it matters` |
| adr | `## Context`, `## Decision`, `## Consequences` |
| decision | `## Question`, `## Decision`, `## Reasoning` |
| contract | `## Purpose`, `## Stability` |

Two ways to land the body content:

- **Two-step (default)**: `aiwf add <kind> --title "..."` creates the entity with empty body sections; then edit the file and run `aiwf edit-body <id>` to commit the prose with proper trailers. Works for every kind today. Right when the body shape isn't fully clear yet — let the file scaffold first, then iterate the prose.
- **One-step (in-verb)**: pass `--body-file <path>` (or `-` for stdin) on `aiwf add` so the body lands in the same atomic create commit as the frontmatter. Available for all six top-level kinds and for ACs (see the body-file sections above). Right when the body content is **already drafted** — mining from a design doc, a prior conversation, a code comment, or a CLI tool's stderr that named the defect. Landing it in the create commit avoids the follow-up untrailered hand-edit (and the `provenance-untrailered-entity-commit` warning that would otherwise fire on the next `aiwf check`).

### Locating the rich body template

`aiwf add` writes a *minimal valid* skeleton — enough headings to clear `aiwf check`, not the house-style shape. The richer per-kind template is materialized at `.claude/templates/<kind>.md` by `aiwf update`: `adr.md` and `decision.md` (the Nygard-structured ADR / decision bodies, carrying the `# <id> — <title>` H1 and date / decided-by header the skeleton omits) and `epic-spec.md` / `milestone-spec.md` (the planning-spec scaffolds). For ADR and decision the `aiwfx-record-decision` ritual reads the template and fills it in; for epic and milestone the planning rituals do. **`gap` and `contract` have no rich template — the skeleton is their whole shape.** If `.claude/templates/` is absent, the templates aren't materialized: run `aiwf update` — don't reconstruct a body by copying an existing entity, which drifts from the canonical template and drops the H1 / header.

### What to write per kind

The per-kind table above lists *which* sections must be non-empty; this subsection covers *what* to write in each. The recommendations are advisory — `aiwf check` asserts presence, not structure — but they shape the project's default; an LLM (or human) skimming this skill produces better entities by following them than by inventing a shape.

**Acceptance criteria.** One paragraph (not an essay, not a one-liner) covering three things: (a) the **pass criterion** — the assertable claim, "under inputs X the system produces Y"; (b) the **edge cases** the test must cover — boundary values, malformed inputs, error paths, concurrency; (c) the **code references** — the file or function the AC will land against, or the test file that pins it. The forward references trade a little churn (paths can move) for a lot of context (a future reader doesn't have to grep for the call site).

```markdown
### AC-3 — Validates frontmatter shape on add

The verb refuses to write when frontmatter would be malformed.
**Pass criterion**: `aiwf add gap --title ""` exits 2 with the
literal `title is empty` in stderr; no file is written; no commit.
**Edge cases**: leading/trailing whitespace on `--title` (treated as
empty), non-UTF-8 bytes (refused with `invalid encoding`), multi-line
title (refused, single line). **Code references**: validation in
`cmd/aiwf/add_cmd.go` (the `validateTitle` helper); regression tests
in `cmd/aiwf/add_cmd_test.go`.
```

**Epics.** `## Goal` describes the problem the epic solves and what success looks like — one paragraph, no longer than four sentences. `## Scope` enumerates what's in (one bullet per major piece of work, often a milestone). `## Out of scope` enumerates what's deliberately not — usually the most-tempting adjacent work, with a one-line "why not yet."

**Milestones.** `## Goal` describes the chunk of value this milestone ships. `## Approach` is the implementation sketch — which packages get touched, which existing patterns get extended, what the verb / rule / file shape will be. `## Acceptance criteria` is the heading container; the actual ACs land as `### AC-N — <title>` sub-elements with their own bodies.

**Gaps.** `## What's missing` is the **concrete defect** — what specifically doesn't exist or doesn't work; one paragraph naming the symptom and the affected surface. `## Why it matters` is the consequence — what fails, who notices, what bug class this enables; one paragraph naming the operational impact.

**ADRs / decisions.** `## Context` (or `## Question`) frames the choice the team faces. `## Decision` records the choice in one or two sentences. `## Consequences` (or `## Reasoning`) names the trade-offs accepted — what becomes easy, what becomes harder, what we'd revisit if a constraint changed.

**Contracts.** `## Purpose` names what the schema captures and who consumes it. `## Stability` names the contract's evolution posture (frozen, additive-only, breaking-allowed-with-migration), with a sentence on what triggers a version bump.

```markdown
## What's missing

`aiwf add gap` accepts `--discovered-in <id>` but does not validate
that the referenced entity exists. A typo (`M-008` for `M-007`) lands
silently; only `aiwf check` catches it later, and only as a
`refs-resolve/unresolved` warning rather than at the point of intent.

## Why it matters

Operators who file gaps mid-flow rely on the kernel to catch finger-
errors at the verb boundary. A silently-accepted typo means the gap
points at no entity at all, the audit trail loses a meaningful link,
and the operator has to repair the gap separately later — exactly
the failure class the verb-time projection check exists to prevent.
```

Skip the prose and `aiwf check` reports the omission. Don't ship a half-written entity hoping the body "follows later" — the design's "prose is not parsed" principle (see `docs/archive/pocv3/acs-and-tdd-plan.md` and `docs/design/design-decisions.md`) treats body content as the spec; the title is a label, not a substitute.

## Provenance flags

| Flag | When |
|---|---|
| `--actor <role>/<id>` | Override the runtime-derived identity (default: `human/<localpart-of-git-config-user.email>`). Rarely needed by hand. |
| `--principal human/<id>` | **Required** when `--actor` is non-human (`ai/...`, `bot/...`); **forbidden** when `--actor` is `human/...`. The principal is who is accountable. |

If the LLM is invoked turn-by-turn by a human (HITL / tool mode), pass `--actor ai/<id> --principal human/<id>`. For autonomous work, the human first runs `aiwf authorize <scope-entity> --to ai/<id>`; then agent verbs work the same way and the kernel matches the scope automatically.

## Don't

- Don't hand-edit frontmatter to "skip allocation" — the id allocator + commit trailer chain is what makes history queryable.
- Don't pre-create the directory; `aiwf add` does that.
- Don't pass `--actor` unless the user asked for a specific actor; the default (derived from git config user.email) is correct.
- Don't omit `--principal` when invoking as a non-human actor — the verb refuses with a `provenance-trailer-incoherent` finding.
- Don't manually edit the milestone's `acs[]` to "fix" a gap from a cancelled AC — AC ids are position-stable. After cancelling AC-2, the next `aiwf add ac` allocates AC-3, not a recycled AC-2.
- Don't leave load-bearing body sections empty for any entity kind — the title is a label, not a spec. For `gap`/`decision`/`adr`/`contract` this is refused at creation (see *"Empty-body gate for born-complete kinds"* above); for epic/milestone/AC, `aiwf check` surfaces the omission as `entity-body-empty` (warning by default; error under `aiwf.yaml: tdd.strict: true`). The body is the spec — write the prose detail (description, examples, edge cases, references) before declaring the entity complete. See *"After `aiwf add <kind>`: fill in the body"* above for the per-kind shapes.

## Tree discipline — `work/` is aiwf's domain

`work/` is the entity tree, not a scratch space. Tree-shape changes — creating an entity, renaming, status transitions, adding ACs — go through verbs (`aiwf add`, `aiwf rename`, `aiwf promote`, `aiwf reallocate`). **Do not write a new file under `work/` directly.** The id allocator, FSM, atomic-commit, repo lock, and trailer pipeline are all bypassed by hand-writes, and the resulting state is silently inconsistent.

Every entity-file mutation goes through a verb route:

- **Creating an entity with body content already drafted**: use `aiwf add --body-file <path>` so the body lands in the same atomic create commit.
- **Editing the body of an existing entity**: use `aiwf edit-body <id> --body-file <path>` (see the `aiwf-edit-body` skill). Frontmatter stays the domain of structured-state verbs (promote / rename / cancel); body-prose edits go through `aiwf edit-body` so the commit carries proper trailers.

Plain `git commit` against an entity file triggers `provenance-untrailered-entity-commit` on the next `aiwf check` — that warning is real, not a false positive. Backfill with `--audit-only` if you genuinely had to bypass the verb route, but the typical answer is "use `aiwf edit-body` instead."

What `aiwf check` reports as `unexpected-tree-file`:

- Any file under `work/*` whose path is not one of the six recognized shapes (epic, milestone, gap, decision, contract, ADR — see `docs/design/tree-discipline.md`). Severity: warning by default; **error** when `aiwf.yaml: tree.strict: true`.
- Files inside a contract's directory (`work/contracts/C-NNN-*/`) are auto-exempt — schemas and fixtures live there legitimately.
- Globs in `aiwf.yaml: tree.allow_paths` are exempt for project-specific carve-outs.

If the user asks to "add a note about X" or similar prose work, edit the relevant entity's body — don't create a stray file. If the prose doesn't fit any existing entity, the right answer is usually a new entity (`aiwf add gap "..."` for a defect, `aiwf add decision "..."` for a directional choice) — not a free-floating file under `work/`.
