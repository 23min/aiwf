---
name: aiwf-check
description: Use when the user wants to validate the planning tree or asks why `aiwf check` reported a finding. Explains each finding code and the typical fix.
---

# aiwf-check

The `aiwf check` verb is a pure function from the working tree to a list of findings. It runs as a `pre-push` git hook; that hook is the chokepoint that turns the framework's guarantees into mechanical enforcement.

## When to use

- The user wants to know "is the tree clean?".
- A push was blocked by the pre-push hook and the user is asking what the finding means.
- A verb refused to write because the projection introduced a finding.

## What to run

```bash
aiwf check                  # human-readable text
aiwf check --format=json    # JSON envelope for tooling
aiwf check --format=json --pretty
aiwf check --since <ref>    # explicit base for the provenance untrailered-entity audit
aiwf check --shape-only     # tree-discipline rule only; used by the pre-commit hook
```

### Two chokepoints, by hook

| Hook | What it runs | What it catches |
|---|---|---|
| `pre-commit` | `aiwf check --shape-only` | Stray files under `work/` (`unexpected-tree-file`). Fast LLM-loop signal — the bad commit never lands. Agent-agnostic (any client running `git commit` triggers it). Blocks only when `aiwf.yaml: tree.strict: true`; otherwise warns and proceeds. |
| `pre-push` | Full `aiwf check` | Everything else: frontmatter shape, refs resolve, FSM, provenance, contract config. Audit chokepoint where push-blocking is appropriate; tolerant of WIP between commits. |

`--shape-only` skips the trunk read, provenance walk, and contract validation, so the pre-commit hook stays fast and never blocks on transient WIP findings. Use it directly only when you want the fast subset; for normal validation, plain `aiwf check` is the right invocation.

### `--since <ref>` — provenance audit scope

The untrailered-entity audit (`provenance-untrailered-entity-commit`) walks a single revision range. The default is `@{u}..HEAD`, so commits already pushed to the upstream are someone else's responsibility to repair.

When the branch has **no upstream** (a fresh feature branch, or a branch whose remote was deleted), the default range is undefined and the audit is **skipped** with one `provenance-untrailered-scope-undefined` warning, rather than scanning all of `HEAD` and flooding the operator with commits already merged in from trunk. To opt back in, either configure an upstream (`git push -u origin <branch>`) or pass `--since <ref>`:

```bash
aiwf check --since main      # walk main..HEAD on the local branch
aiwf check --since HEAD~50   # walk the last 50 commits
```

## Findings (errors)

| Code | Meaning | Typical fix |
|---|---|---|
| `ids-unique` | Two entities share an id. Almost always from a parallel-branch merge. | `aiwf reallocate <path>` on the loser. |
| `ids-unique/trunk-collision` | An id allocated on this branch is also allocated on the configured trunk ref (default `refs/remotes/origin/main`) at a different path — i.e. two different entities now share it across branches. The cross-tree variant of `ids-unique`. | `aiwf reallocate <path>` on whichever side hasn't reached trunk yet. The pre-push hook surfaces this before the colliding push lands. |
| `frontmatter-shape` | Required field missing or malformed. | Add the field; check the kind's id format. |
| `status-valid` | Status is not in the kind's allowed set. | Pick a status from the kind's set (see `aiwf-promote`). |
| `refs-resolve/unresolved` | A reference points at an id that does not exist. | Either the target was never created, or the id is mistyped. |
| `refs-resolve/unresolved-milestone` | The composite-id reference's milestone half (`M-NNN/AC-N`) names a milestone that does not exist. | Fix the milestone id or create the milestone. |
| `refs-resolve/unresolved-ac` | The composite-id reference's AC half (`M-NNN/AC-N`) names an AC that does not exist on the milestone. | Fix the AC number or add the missing AC. |
| `refs-resolve/wrong-kind` | A reference points at an entity of the wrong kind. | A milestone's `parent` must be an epic; an ADR's `supersedes` must be ADRs; etc. |
| `body-prose-id/malformed-shape` | An entity's body prose contains an id-shaped token whose suffix isn't a valid id (letter suffix `M-a`, uppercase placeholder `M-NNNN`, or narrow-numeric `M-1`). | Replace with the canonical allocated id (`M-0001`), or wrap in backticks if the prose is discussing id syntax. Conversational sequential labels like `M-1`/`M-2` belong in chat, not committed prose. |
| `body-prose-id/unresolved` | An entity's body prose references a well-formed id (`M-9999`) that resolves to no entity. | Fix the spelling, or wrap in backticks if the prose is discussing a hypothetical id rather than a real reference. |
| `body-prose-id/unresolved-milestone` | A composite id in body prose (`M-NNNN/AC-N`) names a milestone that does not exist. | Fix the milestone id or remove the reference. |
| `body-prose-id/unresolved-ac` | A composite id in body prose names an AC that does not exist on the parent milestone. | Fix the AC number or add the AC. |
| `no-cycles` | A cycle in the milestone `depends_on` DAG or the ADR `supersedes` chain. | Remove a back-edge. |
| `no-cycles/depends_on` | The cycle is in milestone `depends_on` edges. | Break a back-edge in the milestone DAG. |
| `no-cycles/supersedes` | The cycle is in ADR `supersedes` edges. | Break the chain — an ADR cannot transitively supersede itself. |
| `case-paths` | Two entity paths differ only in case. Linux commits both; macOS / Windows case-insensitive filesystems collapse them to one entity. | `git mv` one of the directories so the names differ in more than case. |
| `load-error` | A file under `work/` failed to parse — malformed YAML frontmatter, unreadable file, or a structural issue the loader couldn't recover from. | Open the named file and fix the parse issue; subsequent checks run once load succeeds. |
| `contract-config` | A contract binding in `aiwf.yaml` references an id with no entity, a missing schema/fixtures path, or a contract entity has no binding. | Run `aiwf contract bind` / `aiwf add contract`, fix the path, or `aiwf contract unbind`. |
| `contract-config/missing-entity` | The binding's `id:` points at a contract entity that doesn't exist. | Either create the contract entity or remove the stale binding. |
| `contract-config/missing-schema` | The binding's `schema:` path doesn't exist on disk. | Fix the path or create the schema file. |
| `contract-config/missing-fixtures` | The binding's `fixtures:` directory doesn't exist on disk. | Fix the path or create the fixtures tree. |
| `contract-config/no-binding` | A contract entity exists but no binding in aiwf.yaml references its id. | `aiwf contract bind <id> --validator <name> --schema <path> --fixtures <path>`. |
| `fixture-rejected` | A `valid/` fixture failed the schema. | Make the schema accept it, or move it to `invalid/`. |
| `fixture-accepted` | An `invalid/` fixture passed the schema. | Tighten the schema, or move to `valid/`. |
| `evolution-regression` | A historical `valid/` fixture fails the HEAD schema. | Revert the schema change, migrate the fixture, or rebind. |
| `validator-error` | Every valid fixture for a contract was rejected — the schema or validator invocation is likely broken. | Inspect the captured stderr and fix the schema or validator command. |
| `environment` | Validator binary not on PATH. | Install it (see the recipe's install instructions) or fix `command:` in `aiwf.yaml`. |
| `acs-shape/id` | An AC's id doesn't match `AC-N` or doesn't follow the per-milestone `1..max` ordering. | Fix the id in the milestone's `acs[]` list. |
| `acs-shape/title` | An AC's title is missing or whitespace-only. | Fill in the title. |
| `acs-shape/status` | An AC's status is not in `{open, met, cancelled}`. | Use one of the three statuses. |
| `acs-shape/tdd-phase` | An AC's `tdd_phase` is set on a milestone that is not `tdd: required`, OR it's set to a value not in `{red, green, refactor, done}`. | Either set the milestone to `tdd: required`, remove the field from the AC, or fix the phase value. |
| `acs-shape/tdd-policy` | An AC at `tdd_phase: done` is in a milestone that is not `tdd: required`. | Either flip the milestone to `tdd: required` or remove the `tdd_phase` field. |
| `acs-body-coherence/missing-heading` | The frontmatter `acs[]` lists an AC, but the body has no `### AC-N — <title>` heading for it. | Run `aiwf add ac` (which scaffolds the heading), or hand-edit the body to add it. |
| `acs-body-coherence/orphan-heading` | The body has an `### AC-N — ...` heading but the frontmatter `acs[]` list does not include AC-N. | Either remove the heading or add the missing AC to `acs[]`. |
| `acs-body-coherence/duplicate-heading` | The `## Acceptance criteria` section repeats a `### AC-N` heading for the same id. A duplicate of an id that is also in frontmatter is neither missing nor orphan, so it would otherwise pass clean. Scoped to the AC section, so the `## Work log` convention (which repeats `### AC-N — <outcome>` headings) is not flagged. | Delete the extra heading; keep exactly one `### AC-N` per AC in the section. `aiwf add ac` now rewrites a placeholder heading in place rather than appending a second one. |
| `archived-entity-not-terminal` | A file lives under a per-kind `archive/` subdirectory but its frontmatter status is not terminal — i.e., a contributor hand-edited the status off-terminal after the entity was swept (per ADR-0004 §"Reversal"). The remediation is to **revert the hand-edit, not to relocate the file** — the kernel does not provide a reverse-archive verb; the canonical pattern when a closed entity needs revisiting is to file a new entity that references the archived one. | Restore the status to a terminal value, or file a new entity that resolves/supersedes the archived one. |

## Findings (warnings)

| Code | Meaning |
|---|---|
| `titles-nonempty` | Title is missing or whitespace-only. |
| `roadmap-case-collision` | More than one case-variant of the generated roadmap artifact exists at the repo root (e.g. both `ROADMAP.md` and `roadmap.md`). Only physically possible on a case-sensitive filesystem; `aiwf render roadmap --write` reconciles to a single existing variant but cannot pick between two, so it leaves this advisory for you to resolve. Fix: `git rm` one variant so a single canonical `ROADMAP.md` (or the lowercase convention the repo already uses) remains. |
| `adr-supersession-mutual` | ADR A says it's superseded by B, but B does not list A in its `supersedes`. |
| `gap-addressed-has-resolver` | Gap is `addressed` but `addressed_by` is empty. |
| `epic-active-no-drafted-milestones` | An epic at status `active` has zero milestones at status `draft`. The kernel-side preflight signal for `aiwfx-start-epic` (per G-0063): an active epic without queued draft work is a forward-motion gap. **Strict-literal reading** — the rule asks "what's queued next?", not "is anything in flight?", so it stays firing through the epic's lifecycle until either a new milestone is drafted or the epic is wrapped. Fix: `aiwf add milestone --epic E-NN --tdd <policy> --title "..."` to queue the next milestone, or `aiwf promote E-NN done` if all planned work is in flight or done. |
| `unexpected-tree-file` | A file under `work/` is not a recognized entity file — tree-shape changes go through `aiwf <verb>`, not direct writes. Promoted to **error** when `aiwf.yaml: tree.strict: true`. Configure exemptions via `aiwf.yaml: tree.allow_paths` (list of `filepath.Match` globs). Files inside a contract's directory (`work/contracts/C-NNN-*/`) are auto-exempt. See `docs/pocv3/design/tree-discipline.md`. |
| `provenance-untrailered-entity-commit` | A commit in the audit range (`@{u}..HEAD` by default; see `--since`) touched an entity file with no `aiwf-verb:` trailer (manual `git commit`). One finding **per (commit, entity)** — a commit touching three entities emits three findings, each tagged with its entity id. Repair with `aiwf <verb> <id> --audit-only --reason "..."` per entity; the matching finding clears on the next push. Audit-only on `M-NNN/AC-N` rolls up to `M-NNN` for matching. |
| `provenance-untrailered-entity-commit/squash-merge` | Same finding, specialized when the offending commit's subject ends with ` (#NNN)` — i.e., GitHub's default squash-merge pattern. Squash-merging through the GitHub UI silently drops the squashed commits' aiwf-verb trailers, even when the source commits were well-formed. Either change the repo's merge strategy to rebase-merge or `--no-ff` merge for branches that touch entity files, or run `aiwf <verb> <id> --audit-only --reason "..."` per entity touched to backfill the audit trail. |
| `provenance-untrailered-scope-undefined` | The audit range is undefined: the branch has no upstream and `--since <ref>` was not passed. The audit is **skipped**. Configure an upstream (`git push -u origin <branch>`) or pass `--since <ref>` to opt back in. |
| `trailer-verb-unknown` | A commit's `aiwf-verb:` trailer carries a value that is not in the closed set of registered verbs and subverbs — every command path from `aiwf`'s Cobra tree, joined by hyphens (e.g. `add`, `add-ac`, `milestone-depends-on`, `render-roadmap`). Typical sources: an LLM-fabricated value on a hand-rolled Conventional-Commits commit (e.g. `aiwf-verb: implement` on a `feat(...)` commit — the worked example from G-0150), or a plugin-side ritual verb that lives outside aiwf's CLI. **Split severity (G-0218 Patch 2):** commits whose ancestry includes the commit-msg-hook-install SHA emit at **error** with a remediation hint — the hook would have refused them at composition time, so landing them required `--no-verify` or git plumbing. Pre-hook history and any clone where the hook-install SHA is unreachable (shallow clone, fork divergence) stay at **warning** so addressed_by_commit refs and historical fabrications aren't retroactively broken. Fix: if the trailer was fabricated, amend the commit and drop the line — plain `feat(...)` / `fix(...)` code commits don't need an `aiwf-verb:` trailer. If a plugin emits the value, change the plugin to use a verb name aiwf registers (or to omit the trailer). Sovereign-human override: `aiwf acknowledge-illegal <sha> --reason "..."` silences the specific commit's finding without rewriting history. Closes G-0150; G-0218 Patch 2 tightens severity for post-hook. |
| `id-rename-untrailered` | A commit between `merge-base(HEAD, trunk)` and HEAD renames an id-bearing entity file (`work/<kind>/<id>-<slug>.md` or the equivalent per `entity.PathKind`) AND lacks an `aiwf-verb:` trailer in the rename-class closed set (`retitle` / `rename` / `reallocate` / `archive` / `move`). The chokepoint catches the CLAUDE.md §"Id-collision resolution at merge time" operator-discipline failure mode: resolving a trunk-collision via inline `git mv` instead of `aiwf reallocate <new-id-or-path>`. The immediate trunk-collision finding clears (gitops' rename detection paired the move via G-0167's trailer-driven path or G-0109's cumulative-similarity fallback), but the kernel trailer history misses the renumber event — `aiwf history G-old` doesn't bridge to the new id, cross-references in body prose aren't rewritten, and any future check rule keyed on `aiwf-verb: reallocate` doesn't see the rename. **Warning severity at first land** (M-0160/AC-4 design); future tightening to error is deferred to a D-NNN once one epic of usage demonstrates the discipline. Canonical resolution: `aiwf reallocate <new-id-or-path>` records the renumber with the proper trailer set and rewrites cross-references. Sovereign-human override: `aiwf acknowledge-illegal <sha> --reason "..."` silences the specific commit's finding without rewriting history (for renames that were deliberate). |
| `acs-tdd-tests-missing` | An AC at `tdd_phase: done` under a `tdd: required` milestone has no `aiwf-tests:` trailer on any commit in its history. Gated by `aiwf.yaml.tdd.require_test_metrics: true`; default off. Fix: re-run the cycle through `aiwf promote --phase ... --tests "pass=N fail=N skip=N"`, or set the YAML field to `false` to silence. |
| `terminal-entity-not-archived` | An entity has a terminal status (e.g. `done`, `addressed`, `wontfix`) but its file is still in an active dir — the normal transient state under ADR-0004's decoupled model. One warning per pending-sweep entity. **Advisory by default**; the `archive.sweep_threshold` knob (M-0088) flips this to blocking past N. The aggregate `archive-sweep-pending` finding summarizes the count. Fix: run `aiwf archive --dry-run` to preview the sweep, then `aiwf archive --apply` to move terminals into their per-kind `archive/` subdir in one commit. |
| `archive-sweep-pending` | Aggregate finding reporting the **count** of terminal-entity-not-archived instances. Per-tree (no path/entity id). **Hidden when zero.** Advisory by default; the `archive.sweep_threshold` knob (M-0088) flips this to blocking past N. Fix: same as `terminal-entity-not-archived` — run `aiwf archive --dry-run` to preview the sweep, then `aiwf archive --apply` to clear the backlog. |
| `entity-id-narrow-width` | The active tree mixes narrow and canonical (4-digit) entity-id widths — i.e., the tree is mid-migration to the ADR-0008 canonical-width policy. Per ADR-0008 §"Drift control" the rule is **silent on uniform trees** (either all-narrow or all-canonical) and fires only when both widths coexist outside `<kind>/archive/`. One warning per narrow active entity. Archive entries never participate in the active-tree state assessment. Fix: run `aiwf rewidth --apply` to canonicalize the active tree in a single commit, or hand-correct the rogue narrow file if it landed by allocator regression / hand-edit. Pre-migration consumers (uniform-narrow trees) stay silent indefinitely — the kernel does not nag. |
| `entity-body-empty` | An entity's load-bearing body section is empty — no non-heading non-whitespace content between the section heading and the next heading or EOF. HTML comments do **not** satisfy the rule (`<!-- TODO -->` is operator intent to defer, not the prose the design specifies). Per-kind subcodes name which kind fired: `entity-body-empty/epic`, `entity-body-empty/milestone`, `entity-body-empty/ac`, `entity-body-empty/gap`, `entity-body-empty/adr`, `entity-body-empty/decision`, `entity-body-empty/contract`. **Asymmetric semantics**: top-level `## Section` bodies treat sub-headings as content (a milestone's `## Acceptance criteria` is non-empty when it contains `### AC-N` headings, even with no parent-level prose); `### AC-N` bodies require true non-heading prose, since they are the leaf-prose container. **Grandfather rule**: this finding is independent of `acs-tdd-audit` — empty-body warnings on historical `met` + `tdd_phase: done` ACs do **not** retroactively re-engage the TDD audit. Fix: write prose for the named section. For ACs created post-M-067, `aiwf add ac --body-file <path>` scaffolds the body at create time; for existing ACs and all other kinds today, edit the file and run `aiwf edit-body <id>` (G-066 tracks bringing `--body-file` to the other six creation verbs). The check is permissive about *what* the prose is — paragraphs, bullet lists, code blocks, single sentences all clear the rule (kernel principle: prose is not parsed). Severity escalates to **error** under `aiwf.yaml: tdd.strict: true`. |
| `fsm-history-consistent/illegal-transition` | A status-change commit moves an entity's `status:` between two values that are **not** connected by a legal edge in the kind's FSM (e.g., epic `proposed → done` skipping `active`), AND the commit has no `aiwf-force:` trailer to record a sovereign override. The kernel chokepoint that makes the per-entity status FSM a **tree-invariant** rather than just a verb-precondition — closes G-0132. **Error severity** (blocks pre-push). The walk is per-entity via `git log --follow`, so renames preserve history; the prior status is read from `git show <parent-sha>:<path>`. Fix: re-route the change through `aiwf promote <id> <to>` (which only accepts FSM-legal moves) or `aiwf cancel <id>`; when the exceptional flip is genuinely warranted, re-run the verb with `--force --reason "..."` so the override rides in the trailers. |
| `fsm-history-consistent/forced-untrailered` | A status-change commit matches a **sovereign-act shape** — a transition the kernel deliberately requires explicit override for, such as epic `proposed → active` (ADR-0007's ratification semantics) — but the commit carries no `aiwf-force:` trailer. The transition itself is recognized by the FSM; the missing trailer is what fires the finding. **Error severity**. Fix: re-run the verb with `--force --reason "..."` so the sovereign nature of the act is recorded, or undo the change via the corresponding inverse verb. The aiwf-actor on a force-trailered commit must be `human/...` (provenance rule), which is what makes the sovereign trail auditable. |
| `fsm-history-consistent/manual-edit` | A status-change commit is a legal FSM step (the transition exists in the kind's FSM) but the commit message has **no `aiwf-verb:` trailer** at all — a hand-edit + `git commit` that bypassed the kernel verb path. Overlaps with `provenance-untrailered-entity-commit` (which fires per-entity on any untrailered touch) but with FSM-specific framing and **error severity** (vs the provenance code's warning). Audit-only suppression: a later commit carrying `aiwf-audit-only:` with an `aiwf-entity:` matching this entity clears the finding for the (commit, entity) pair, mirroring how provenance-untrailered-entity-commit works. Fix: re-route through `aiwf promote` / `aiwf cancel`, or — when the change is already merged and rewriting is wrong — run `aiwf <verb> <id> --audit-only --reason "..."` per affected entity so the audit-only commit records the trail. |
| `fsm-history-consistent/history-walk-error` | The walker hit a real failure reading the named entity's commit history during the batched walk — a subprocess crash, a blob-read protocol error, or a context cancelled mid-walk. M-0137 introduces this subcode and the partial-preservation contract it pins: under M-0130 the rule silently swallowed walker errors and returned zero findings, so one transient subprocess crash under load wiped every finding from the rule invisibly. The retrofit (M-0137/AC-3+4+5) routes through `gitops.BulkRevwalk` (one `git log` subprocess for the whole repo) and `gitops.BlobReader` (one long-lived `git cat-file --batch` for status reads); per-blob read failures emit one `history-walk-error` per affected (entity, commit) pair while other entities' findings still surface alongside (CLAUDE.md §Engineering principles — "Errors are findings, not parse failures"). **Error severity**. Fix: re-run `aiwf check` to confirm whether the failure is transient (concurrent test load on macOS, kernel resource pressure); if it repeats, inspect `git fsck` and `.git/objects/` permissions in the consumer repo. |

## Provenance findings (errors)

These fire on commit history, not tree state. Each names the offending commit's short SHA in its message.

| Code | Meaning | Typical fix |
|---|---|---|
| `provenance-trailer-incoherent` | A required-together pair is partial, or a mutually-exclusive pair are both present (e.g., `aiwf-on-behalf-of:` without `aiwf-authorized-by:`, `aiwf-actor: ai/...` without `aiwf-principal:`, `aiwf-actor: human/...` *with* `aiwf-principal:`). | Re-create the commit using the correct verb invocation; `--principal human/<id>` is required when the actor is non-human. |
| `provenance-force-non-human` | `aiwf-force:` present on a commit whose `aiwf-actor:` is not `human/...`. | `--force` is sovereign — only humans wield it. Have a human invoke the verb directly. |
| `provenance-actor-malformed` | `aiwf-actor:` does not match `<role>/<id>`. | `git config user.email` is malformed; fix it (see `aiwf doctor`). |
| `provenance-principal-non-human` | `aiwf-principal:` role is not `human/`. | Principal must be human/<id>; agents and bots cannot be principals. |
| `provenance-on-behalf-of-non-human` | `aiwf-on-behalf-of:` role is not `human/`. | Same as principal — rebuild from the originating authorize commit. |
| `provenance-authorized-by-malformed` | `aiwf-authorized-by:` is not 7–40 hex. | Copy the correct SHA from `aiwf history <scope-entity>`. |
| `provenance-authorization-missing` | The authorize SHA does not name an `aiwf-verb: authorize / aiwf-scope: opened` commit. | Typo or stale SHA after force-push; use the full SHA. |
| `provenance-authorization-out-of-scope` | The verb's target entity has no reference path to the scope-entity. | Either authorize the right entity or work on something the existing scope already reaches. |
| `provenance-authorization-ended` | The scope was already ended (terminal-promote / revoke). | Open a fresh scope with `aiwf authorize <id> --to <agent>`. |
| `provenance-no-active-scope` | An `ai/...` actor produced a commit with no `aiwf-on-behalf-of:`. | Open an authorization scope, or run the verb as the human directly. |
| `provenance-audit-only-non-human` | `aiwf-audit-only:` present on a non-human actor's commit. | Only humans may backfill audit trails. |

## Don't

- Don't bypass the pre-push hook with `--no-verify` to "fix it later" — broken state on `main` is the thing this hook exists to prevent.
- Don't try to make findings disappear by deleting files; `aiwf cancel <id>` is the right way to retire an entity.
- Don't try to "amend away" a `provenance-untrailered-entity-commit` warning — `aiwf <verb> <id> --audit-only --reason "..."` is the first-class per-entity repair path and keeps history append-only. One audit-only commit clears one entity's finding; commits that touched multiple entities need one audit-only per entity (or a single audit-only on a parent that all touched ids reach via composite-id rollup).
