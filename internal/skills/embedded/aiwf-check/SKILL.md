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

## Findings (warnings)

| Code | Meaning |
|---|---|
| `titles-nonempty` | Title is missing or whitespace-only. |
| `adr-supersession-mutual` | ADR A says it's superseded by B, but B does not list A in its `supersedes`. |
| `gap-resolved-has-resolver` | Gap is `addressed` but `addressed_by` is empty. |
| `unexpected-tree-file` | A file under `work/` is not a recognized entity file — tree-shape changes go through `aiwf <verb>`, not direct writes. Promoted to **error** when `aiwf.yaml: tree.strict: true`. Configure exemptions via `aiwf.yaml: tree.allow_paths` (list of `filepath.Match` globs). Files inside a contract's directory (`work/contracts/C-NNN-*/`) are auto-exempt. See `docs/pocv3/design/tree-discipline.md`. |
| `provenance-untrailered-entity-commit` | A commit in the audit range (`@{u}..HEAD` by default; see `--since`) touched an entity file with no `aiwf-verb:` trailer (manual `git commit`). One finding **per (commit, entity)** — a commit touching three entities emits three findings, each tagged with its entity id. Repair with `aiwf <verb> <id> --audit-only --reason "..."` per entity; the matching finding clears on the next push. Audit-only on `M-NNN/AC-N` rolls up to `M-NNN` for matching. |
| `provenance-untrailered-entity-commit/squash-merge` | Same finding, specialized when the offending commit's subject ends with ` (#NNN)` — i.e., GitHub's default squash-merge pattern. Squash-merging through the GitHub UI silently drops the squashed commits' aiwf-verb trailers, even when the source commits were well-formed. Either change the repo's merge strategy to rebase-merge or `--no-ff` merge for branches that touch entity files, or run `aiwf <verb> <id> --audit-only --reason "..."` per entity touched to backfill the audit trail. |
| `provenance-untrailered-scope-undefined` | The audit range is undefined: the branch has no upstream and `--since <ref>` was not passed. The audit is **skipped**. Configure an upstream (`git push -u origin <branch>`) or pass `--since <ref>` to opt back in. |
| `acs-tdd-tests-missing` | An AC at `tdd_phase: done` under a `tdd: required` milestone has no `aiwf-tests:` trailer on any commit in its history. Gated by `aiwf.yaml.tdd.require_test_metrics: true`; default off. Fix: re-run the cycle through `aiwf promote --phase ... --tests "pass=N fail=N skip=N"`, or set the YAML field to `false` to silence. |
| `entity-body-empty` | An entity's load-bearing body section is empty — no non-whitespace content (other than headings) between the section heading and the next heading or EOF. HTML comments do not satisfy the rule. Per-kind subcodes name which kind fired: `entity-body-empty/epic`, `entity-body-empty/milestone`, `entity-body-empty/ac`, `entity-body-empty/gap`, `entity-body-empty/adr`, `entity-body-empty/decision`, `entity-body-empty/contract`. Fix: write prose for the named section via `aiwf edit-body <id>`; for ACs, `aiwf add ac --body-file` (M-067) scaffolds the body during create. Severity escalates to **error** under `aiwf.yaml: tdd.strict: true`. |

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
