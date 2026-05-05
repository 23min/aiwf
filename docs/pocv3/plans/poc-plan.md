# PoC plan — sessions and iterations

This is the working document for the `poc/aiwf-v3` branch. Each session has a deliverable that runs end-to-end before moving on. Mark items as you go; commit per logical step.

The five sessions below are the original PoC build. Subsequent iterations layered on top — each with its own detailed plan in this directory — are summarized after the sessions, in the order they shipped (or, for queued work, in the order proposed). Each iteration's sub-iterations and shipped commits live in its own plan file; this document is the index.

For the design context that justifies this shape, see [`design-decisions.md`](../design/design-decisions.md). For the engineering principles, see the root [`CLAUDE.md`](../../../CLAUDE.md) and [`CLAUDE.md`](../../../CLAUDE.md).

---

## Session 1 — Foundations and `aiwf check`

**Goal:** an executable that loads the tree, validates it, reports findings. No mutating verbs yet.

- [x] Go module skeleton in place under `cmd/aiwf/` and `internal/`.
- [x] Frontmatter parser (use `gopkg.in/yaml.v3`).
- [x] Tree loader: walks `work/epics/**`, `work/gaps/**`, `work/decisions/**`, `work/contracts/**`, `docs/adr/**`. Parses every entity into a typed in-memory model.
- [x] Six kind types defined as Go structs with their hardcoded status enums.
- [x] `aiwf check` with these checks (each as a small function):
  - [x] `ids-unique` — no duplicate ids (severity: error). Detected via path prefix collision.
  - [x] `refs-resolve` — every reference field resolves to an existing entity of the kind permitted by the frontmatter schema (severity: error). Findings distinguish *unresolved* (no such id) from *wrong-kind* (id exists but is the wrong kind).
  - [x] `status-valid` — every status is in the allowed set for the kind (severity: error).
  - [x] `frontmatter-shape` — required fields present, types correct (severity: error).
  - [x] `no-cycles` — no cycle in `depends_on` (milestone DAG) or in the `supersedes`/`superseded_by` chain (ADR DAG) (severity: error).
  - [x] ~~`contract-artifact-exists` — for every contract, `artifact:` is a relative path with no `..` segments that resolves to an existing file *inside* the contract directory (severity: error).~~ **Superseded in I1:** replaced by `contract-config` (with subcodes `missing-entity`, `missing-schema`, `missing-fixtures`, `no-binding`, `path-escape`, `validator-unavailable`) that validates the binding-side correspondence between `aiwf.yaml.contracts.entries[]` and on-disk paths.
  - [x] `titles-nonempty` — title is set and non-empty (severity: warning).
  - [x] `adr-supersession-mutual` — if `A.superseded_by = B`, then `B.supersedes ⊇ {A}` (severity: warning).
  - [x] `gap-resolved-has-resolver` — addressed gap has non-empty `addressed_by` (severity: warning).
- [x] JSON output (`--format=json`) and human-readable text (default).
- [x] Exit codes: `0` clean, `1` findings, `2` usage error, `3` internal.
- [x] Synthetic-tree fixtures under `testdata/`, one per finding type.

**Deliverable:** `aiwf check` runs against a hand-crafted `work/` directory and reports findings correctly.

**Shipped (commit `162bf54`):** entity package (six kinds, status enums, id regexes, frontmatter parser), tree loader, nine validators, JSON + text renderers, exit codes, fixture-driven integration test (`testdata/clean` and `testdata/messy`).

---

## Session 2 — Mutating verbs and commit trailers

**Goal:** the verbs that produce git commits with structured trailers.

- [x] `aiwf add epic --title "..."` — allocate `E-NN`, write `work/epics/E-NN-<slug>/epic.md`, commit.
- [x] `aiwf add milestone --epic E-NN --title "..."` — allocate `M-NNN`, write file under epic, commit.
- [x] `aiwf add adr --title "..."` — allocate `ADR-NNNN`, write file, commit.
- [x] `aiwf add gap --title "..." [--discovered-in M-NNN]` — allocate `G-NNN`, commit.
- [x] `aiwf add decision --title "..." [--relates-to E-NN,M-NNN]` — allocate `D-NNN`, commit.
- [x] `aiwf add contract --title "..."` — allocate `C-NNN`, create directory + `contract.md`, commit. **Note:** the original plan had `--format` and `--artifact-source` flags backed by a `contract-artifact-exists` validator that copied a schema into the contract dir. That model was replaced in I1 by *contract bindings* in `aiwf.yaml.contracts.entries[]`. The shipped `add contract` accepts `--linked-adr <ids>` and the optional atomic-bind triplet (`--validator`, `--schema`, `--fixtures`) for one-commit add+bind. See [`contracts-plan.md`](contracts-plan.md).
- [x] `aiwf promote <id> <status>` — read entity, validate transition (one Go function per kind), edit frontmatter, commit.
- [x] `aiwf cancel <id>` — promote to the kind's terminal-cancel status (`cancelled`/`wontfix`/`rejected`/`retired`).
- [x] `aiwf rename <id> <new-slug>` — `git mv` + commit. The id is preserved; title is unchanged (edit frontmatter manually if you want it tracked).
- [x] `aiwf reallocate <id|path>` — pick next free id, `git mv`, walk every entity's frontmatter and rewrite reference fields, surface body-prose references as findings, commit. Accepts a path (instead of an id) when the id is ambiguous — required after a merge collision where two files share the same id.
- [x] Every mutating verb computes the projected new tree in memory, runs `aiwf check` against the projection, and either (a) writes files and creates the single commit when clean, or (b) returns findings without touching the working tree. No rollback path: nothing is written until the projection is known good.
- [x] Every commit-producing verb writes structured trailers: `aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`. `reallocate` additionally writes `aiwf-prior-entity: <old-id>` so both the old and new id's histories are queryable.
- [x] Round-trip tests for each verb against a fresh git repo fixture.

**Deliverable:** end-to-end planning workflow works. `aiwf init && aiwf add epic && aiwf add milestone && aiwf promote ... && aiwf rename ...` produces a sensible git history.

**Shipped (commits `9230fa4`, `deaf72f`):** five mutating verbs (add for all six kinds, promote, cancel, rename, reallocate); `entity` extended with serialize/slug/templates/transitions/allocator; new `gitops` and `verb` packages; `Apply` orchestrator; PlannedFiles overlay so `contract-artifact-exists` validates the projected world; round-trip tests for every verb. Follow-up commit added edge-case coverage (reallocate by path/contract; cancel-already-terminal; same-slug rename; CLI dispatcher tests; actor regex fix) and the `projectionFindings` diff so pre-existing tree errors don't block unrelated verbs.

---

## Session 3 — Skills, history, hooks

**Goal:** the AI can use it; `git log` becomes queryable.

- [x] Skill markdown files written and embedded via `embed.FS`. Skills shipped:
  - [x] `aiwf-add` — how to create each kind with proper frontmatter.
  - [x] `aiwf-promote` — how to advance status legally per kind.
  - [x] `aiwf-rename` — how to rename without breaking references.
  - [x] `aiwf-reallocate` — how to resolve id collisions.
  - [x] `aiwf-history` — how to ask "what happened here?".
  - [x] `aiwf-check` — what `aiwf check` reports and how to fix common findings.
- [x] `aiwf init` (idempotent; safe to re-run; produces no git commit — the user commits when ready):
  - [x] writes `aiwf.yaml` (~10 lines) at the consumer repo root if missing; preserves an existing file unchanged. The `actor` field defaults to `human/<local-part-of-git-config-user.email>` (e.g., `human/peter` for `peter@example.com`); if neither `user.email` nor `user.name` is set, errors with an instruction to set git config or pass `--actor`. The actor value (whether derived or explicit) is validated against `^\S+/\S+$` before write; the same regex validates `aiwf.yaml`'s `actor:` field on every verb invocation and any `--actor` flag override.
  - [x] scaffolds `work/epics/`, `work/gaps/`, `work/decisions/`, `work/contracts/`, `docs/adr/` if missing; never modifies existing directories or their contents.
  - [x] materializes skills to `.claude/skills/aiwf-*/SKILL.md` (wipe-and-rewrite per the cache contract; non-`aiwf-*` skill directories are untouched).
  - [x] appends materialized-skill paths to `.gitignore` if not already present; does not rewrite the file.
  - [x] writes a short `CLAUDE.md` template only if the file is missing.
  - [x] installs `.git/hooks/pre-push` that runs `aiwf check`. The hook carries an `# aiwf:pre-push` marker comment. If a hook exists with the marker → overwrite (idempotent). If a hook exists without the marker → refuse with a useful error explaining how to integrate `aiwf check` into the existing hook manually, or use a hook manager (husky/lefthook) that composes hooks.
  - [x] pre-existing entity files in `work/` and `docs/adr/` are not modified or validated by `init`; they show up as findings on the next `aiwf check` and serve as the migration to-do list when adopting `aiwf` against an existing repo.
- [x] `aiwf update` — remove every `.claude/skills/aiwf-*/` directory and re-materialize from the binary's embedded skills (no commit; updates gitignored files). Directories not matching `aiwf-*` are untouched (user-authored skills are namespace-isolated).
- [x] `aiwf history <id>` — read `git log` filtered for `aiwf-entity: <id>` *or* `aiwf-prior-entity: <id>` trailers (so reallocate events are visible from both the old and new id). Default output is one line per event: `DATE  ACTOR  VERB  DETAIL  COMMIT`, where `DETAIL` is the commit subject line shaped by the verb at commit time (`"title"` for add, `old → new` for promote, `slug → <new>` for rename, `→ cancelled` for cancel, `<old-id> → <new-id>` for reallocate). `--format=json` mirrors `aiwf check`'s machine-readable contract. Trailer-matched events only — `aiwf history` does not show side-effect file edits (use `git log -- <path>` for byte-level history).
- [x] `aiwf doctor` — check binary version vs. `aiwf.yaml`'s `aiwf_version`, byte-compare each materialized skill against its embedded version and report drift, check id-collision health.
- [x] Tests: `aiwf init` in a fresh git repo produces the expected layout; `aiwf history` returns the expected events for a multi-step fixture.

**Deliverable:** in a fresh consumer repo, `aiwf init` sets things up; the AI host (Claude Code) sees the skills; the pre-push hook catches errors before push.

**Shipped:** new `skills` package with embedded `aiwf-*/SKILL.md` files for the six verbs; new `config` package owning `aiwf.yaml` parse/validate/write (and `--actor` resolution now consults it); new `initrepo` package with idempotent setup (config, scaffolding, skill materialization, `.gitignore` append, `CLAUDE.md` template, marker-aware `pre-push` hook); four new CLI subcommands (`init`, `update`, `history`, `doctor`); `gitops.GitDir` helper for worktree-aware hook install; `aiwf history` consumes structured trailers via `git log --grep` with `\x1f`/`\x1e` field separators and queries both `aiwf-entity:` and `aiwf-prior-entity:` so reallocate events surface from either id; `aiwf doctor` byte-compares embedded vs. on-disk skills and runs `ids-unique` from `check`. Coverage: `initrepo`, `skills`, `config` unit-tested; CLI dispatcher tests cover init/update/history/doctor through the top-level `run`.

---

## Session 4 — Polish for real use

**Goal:** ready for use on a real project.

- [x] `aiwf render roadmap` — print a markdown table of epics + milestones; with `--write` updates `ROADMAP.md` and commits.
- [x] `aiwf doctor --self-check` — runs all the verbs against a temp directory.
- [x] Error-message polish — every finding is one line, names file:line, suggests a fix.
- [x] README polish — clear install instructions, quick-start that works.
- [x] A short usage walk-through in `docs/` showing a typical first session.

**Deliverable:** the framework is good enough to start using on a real project.

---

## Session 5 — Adoption surface

**Goal:** the framework can be adopted in repos that already have planning data, without aiwf needing to know what produced that data.

The shape of this session is set by the design constraint that aiwf must be a clean public surface: any knowledge of a specific prior planning system stays out of the aiwf source tree, fixtures, and docs. The public surface is generic; producer-side conversion happens entirely in private tooling.

- [x] `aiwf init --dry-run` — print the actions `init` would take without writing anything. Same exit codes as `init`.
- [x] `aiwf init --skip-hook` — perform `init` without installing the pre-push hook. For repos that want the framework but aren't ready to gate pushes on `aiwf check`.
- [x] `aiwf import <manifest.yaml>` — generic batch entity creator. Reads a declarative manifest (see [`import-format.md`](../migration/import-format.md)), validates the projected tree, and writes one atomic commit (default) or one commit per entity (`commit.mode: per-entity`).
  - [x] YAML and JSON manifest parsers (same schema, two lexers).
  - [x] Two-pass id resolution: explicit ids reserved first, `auto` ids allocated next.
  - [x] Reference resolution against the union of existing-tree ids and manifest-declared ids.
  - [x] `--dry-run`, `--on-collision={fail,skip,update}` flags.
  - [x] Single-mode commits use `aiwf-verb: import`; per-entity-mode commits match the per-entity `add` trailers.
  - [x] Synthetic-tree fixtures inline in tests covering: clean import, id collision (all three modes), ref-resolution across manifest entries, mixed explicit + `auto`, dry-run.
- [x] ~~`wf-track` skill — describes the convention of maintaining a tracking document alongside an in-progress milestone.~~ **Removed during the prefix rename (poc/aiwf-rename-skills) — the tracking-doc convention moves to `aiwfx-track` in the companion rituals plugin (see [`rituals-plugin-plan.md`](rituals-plugin-plan.md)).** aiwf core stays narrow: tracking docs are not entities, not validated, and not aiwf's concern.
- [x] Roadmap `## Candidates` rendering — `aiwf render roadmap` includes the verbatim contents of any `## Candidates` (or `## Backlog`) section it finds in `ROADMAP.md`. The section is human-curated, free-form, and not parsed as entities. Promoting a candidate is an explicit `aiwf add epic` step.
- [x] `docs/pocv3/migration/from-prior-systems.md` — a generic migration guide. Frames migration as a two-stage producer-side job (tidy source data; project to manifest), then `aiwf import`. References no specific prior system.

**Deliverable:** a consumer repo with existing planning data can be adopted by writing a private producer that emits an import manifest, iterating against `aiwf import --dry-run`, and committing the result. aiwf has no awareness of how the manifest was produced.

**Shipped (commits `edcdf3d`, `841effc`, `ea5381a`, `e69f4ea`, this commit):** import manifest format spec; `aiwf init --dry-run` and `--skip-hook` flags with refactored ensure* steps; `aiwf import` verb in `internal/manifest` (parser + structural validator) and `internal/verb/import.go` (two-pass id resolution, forward refs across manifest, all three collision modes, single + per-entity commit modes); CLI integration with `--dry-run`/`--on-collision`/`--actor` flags; `aiwf render roadmap` preserves a hand-curated `## Candidates`/`## Backlog` block round-trip; generic migration guide framing the public/private boundary.

---

## Iteration I1 — Contracts

**Goal:** mechanical contract verification (schema + fixtures) as a first-class part of the pre-push chokepoint, without aiwf shipping any validator binary or branching on language. Full design in [`contracts-plan.md`](contracts-plan.md).

The eight sub-iterations:

- [x] **I1.1** — `aiwfyaml` package: parse, structurally validate, and round-trip-write the `contracts:` block.
- [x] **I1.2** — narrow the contract entity (drop `format`/`artifact`); status set `proposed → accepted → deprecated → retired`, plus `rejected`.
- [x] **I1.3** — `contractverify` package: verify and evolve passes; substitution runner; result reclassification ("all valid rejected" → `validator-error`).
- [x] **I1.4** — `contractcheck` package: structural correspondence between bindings and tree (missing-entity, missing-schema, missing-fixtures, no-binding); composes with the rest of `aiwf check`.
- [x] **I1.5** — `aiwf contract bind/unbind` verbs; `aiwf add contract --validator/--schema/--fixtures` for atomic add+bind in one commit.
- [x] **I1.6** — `aiwf contract recipe` verbs (list/show/install/remove); embedded markdown recipes for CUE and JSON Schema; custom validator install via `--from <path>`.
- [x] **I1.7** — pre-push integration: `aiwf check` runs verify+evolve when bindings are present; terminal-state contracts skipped.
- [x] **I1.8** — `aiwf-contract` skill: embedded SKILL.md materialized into `.claude/skills/aiwf-contract/`.

**I1 hardening (commit `06b33bc`):** edge-case coverage across the contract surface — anchors/aliases rejection, validator-name reference checks, recipe round-trip, atomic add+bind rollback, terminal-state suppression in verify, multi-version evolve.

**Post-I1 gap fixes** (see [`gaps.md`](../gaps.md)) further hardened the contract surface: G1 added path-escape detection in `contractcheck`/`contractverify`; G3 demoted `validator-unavailable` to a warning by default with opt-in `strict_validators` and added a doctor section listing each validator's availability.

**Deliverable:** a consumer repo can declare a CUE or JSON Schema contract via `aiwf contract recipe install <name>` + `aiwf add contract --validator … --schema … --fixtures …` (one commit), populate `<fixtures>/v1/{valid,invalid}/`, and have `aiwf check` (and the pre-push hook) verify the bundle on every push.

---

## Iteration I2 — Acceptance criteria + TDD

**Goal:** first-class acceptance criteria as namespaced sub-elements of milestones, and opt-in TDD enforcement per milestone. Full design and step-by-step build sequence in [`acs-and-tdd-plan.md`](acs-and-tdd-plan.md).

ACs are not a seventh kind — they're structured sub-elements addressed by composite id `M-NNN/AC-N`, validated by `aiwf check`, with the audit rule "AC `met` requires `tdd_phase: done`" when the milestone is `tdd: required`. The legacy v1 tracking-doc convention dies; AC list moves into the milestone doc itself (frontmatter + matching body sections).

The eleven sub-iterations:

- [x] **I2.1** — milestone schema additions for ACs and TDD (commit `35b766e`).
- [x] **I2.2** — composite id grammar `M-NNN/AC-N` (commit `07a0a43`).
- [x] **I2.3** — AC and TDD-phase FSMs, milestone-done precondition (commit `d759f17`).
- [x] **I2.4** — `--force --reason` on promote and cancel (commit `414df2f`).
- [x] **I2.5** — `aiwf-to:` trailer + history renders to/forced columns (commit `6cc3f8b`).
- [x] **I2.6** — check rules for ACs, TDD audit, milestone-done (commit `c857588`).
- [x] **I2.7a** — `aiwf add ac`, composite-id verbs, history prefix-match (commit `7fe22c3`).
- [x] **I2.7b** — `--phase` flag for promote, TDD pre-cycle entry (commit `253d7f4`).
- [x] **I2.7c** — `aiwf show`, per-entity aggregator (commit `3f743a8`).
- [x] **I2.8** — STATUS.md renders AC progress per milestone (commit `db69611`).
- [x] **I2.10** — embedded skills + CLAUDE.md cover ACs and TDD (commit `1f2cd91`).
- [x] **I2.11** — reverse-ref index, `aiwf show referenced_by` (commit `0352f0d`). Load-bearing prerequisite for I2.5 step 6 and I3.

**Deliverable:** milestones declare their ACs in frontmatter + matching body sections; `aiwf check` enforces "AC met requires TDD phase done" when `tdd: required`; `aiwf show M-NNN` aggregates the milestone's AC + phase view; STATUS.md renders progress per milestone.

---

## Iteration I2.5 — Provenance model

**Goal:** separate *who is accountable* (principal, always human) from *who ran the verb* (operator/actor, may be LLM or bot); gate authorized agent work via a typed scope FSM (`active | paused | ended`) opened with `aiwf authorize`; keep `--force` as a sovereign human-only override. Full design in [`../design/provenance-model.md`](../design/provenance-model.md); build sequence in [`provenance-model-plan.md`](provenance-model-plan.md).

The eleven build steps (steps 1–10 shipped; step 11 is an I3 handoff placeholder):

- [x] **Step 1** — drop `aiwf.yaml.actor`, runtime-derive identity from `git config user.email` (commit `3932819`).
- [x] **Step 2** — trailer writer extensions for provenance (`aiwf-principal`, `aiwf-on-behalf-of`, `aiwf-authorized-by`, `aiwf-scope`, `aiwf-scope-ends`, `aiwf-reason`) (commit `f56d707`).
- [x] **Step 3** — required-together / mutually-exclusive trailer coherence rules (commit `0f5dcae`).
- [x] **Step 4** — scope FSM package (commit `f8315c0`).
- [x] **Step 5** — `aiwf authorize` verb (open / pause / resume) (commit `3b573c7`).
- [x] **Step 5b** — `--audit-only --reason` recovery mode for G24 (commit `bc4183e`).
- [x] **Step 5c** — `Apply` lock-contention diagnostic for G24 (commit `6cc0648`).
- [x] **Step 6** — allow-rule + scope-aware verb dispatch; prior-entity chain resolution (commits `2e09f4d`, `22d7c63`).
- [x] **Step 7** — `aiwf check` provenance standing rules (commit `9e7d2fc`).
- [x] **Step 7b** — pre-push trailer audit (G24 surface-the-gap half) (commit `0e44ad6`); audit-only clears the warning (commit `be2ea27`).
- [x] **Step 8** — `aiwf history` rendering for provenance (commit `428014e`).
- [x] **Step 9** — `aiwf show` scopes block (commit `126ac42`).
- [x] **Step 10** — provenance docs and embedded skills (commit `a4cd468`).
- [ ] **Step 11** — render integration handoff to I3 (placeholder; actual work in `governance-html-plan.md`).

**Follow-up coverage** (commits `cd1e165`, `044f5e6`, `9c1b010`): coverage-gap tests, composite/self/multi-auth/empty-range/ancestor/skill-drift fixtures, and the cross-cutting integration scenarios from §4 of the plan.

**Policies sweep** (commits `0b5354e`, `bd9fce7`, `79ccc42`): new `policies` package encodes 23 repo-wide audit-trail invariants on top of the I2.5 trailer surface.

**Deliverable:** every commit carries unambiguous principal × agent × scope provenance; `aiwf authorize` opens scoped agent work that auto-ends on terminal scope-entity status; `--force` and `--audit-only` are sovereign human-only acts; `aiwf check` catches incoherent or out-of-scope commits at push time; `aiwf history` and `aiwf show` render the full picture.

---

## Companion repo — Rituals plugin

**Goal:** opinionated engineering rituals (TDD cycles, code review, doc lint, patch workflows) that layer on top of `aiwf`, distributed as a separate Claude Code plugin marketplace. Full architecture in [`rituals-plugin-plan.md`](rituals-plugin-plan.md). Lives in the companion repo `../ai-workflow-rituals` — *not* in the aiwf kernel tree.

**Status:** shipped. The marketplace + two plugins (`wf-rituals`, `aiwf-extensions`) are pushed and `/plugin marketplace add` validated. The aiwf kernel surfaces the plugin as the recommended next step (commit `92326aa`); install / verify via `aiwf rituals` (`cmd/aiwf/rituals.go`).

**Coupling boundary:** the rituals plugin is `aiwf`-aware (skills name aiwf verbs); the aiwf kernel is *not* rituals-aware beyond the surfacing step. Tracking-doc conventions, TDD cycle micro-rituals, and review patterns live in the rituals repo and stay out of the kernel's contract.

---

## Queued / not started

Plans that exist as proposals but have no implementation commits yet. Sub-iterations and design rationale live in each plan's own document; this section is the index.

| Iteration | Plan | Status | One-liner |
|---|---|---|---|
| **I3** | [`governance-html-plan.md`](governance-html-plan.md) | shipped (steps 1–7 + v0.2.0 polish: palette, sidebar, status page, brand mark, cache-busting; see plan status table §11) | Static-site HTML render of canonical planning state (per-repo governance page). |
| (untiered) | [`status-report-plan.md`](status-report-plan.md) | shipped (`renderStatusMarkdown` + `PlannedEpics` + mermaid flowcharts in `cmd/aiwf/status_cmd.go`; auto-regenerated `STATUS.md` via the pre-commit hook) | Markdown status renderer with embedded mermaid diagrams; extends `aiwf status` with a third format. Renderer change, not new state. |
| (untiered) | [`upgrade-flow-plan.md`](upgrade-flow-plan.md) | shipped (all 9 steps; `internal/version`, `aiwf upgrade` verb, `aiwf doctor` `binary:` / `pin:` / `latest:` rows, `--check-latest`, `--self-check` coverage; tags `v0.1.0` → `v0.2.1` live on the proxy) | `aiwf upgrade` verb + git-tag releases + skew detection in `aiwf doctor`. |

Pick-up order is not committed in advance; real-use friction surfaces the next priority.

---

## Cross-cutting kernel work

Kernel-mechanics changes that don't fit a feature-iteration shape but landed as their own coherent passes.

### Broaden `aiwf update`

**Goal:** make `aiwf update` the upgrade verb that refreshes every artifact the consumer is opted into (skills, hooks, the new pre-commit STATUS.md regenerator). Full plan in [`update-broaden-plan.md`](update-broaden-plan.md).

**Status:** implemented across commits `88727c6` (kernel-shift docs) → `855996a` (self-check covers the round-trip). The pre-commit hook for STATUS.md regeneration is default-on with `status_md.auto_update: false` as the clean opt-out. Touched `internal/initrepo/`, `internal/config/`, `cmd/aiwf/admin_cmd.go`, `cmd/aiwf/selfcheck.go`, plus the design and README docs.

---

## Total

The framework is small, self-contained, self-validating, adoptable against existing planning data, contract-aware, AC + TDD-aware, and provenance-aware — with a companion rituals plugin layered on top. Real use surfaces the next priority; nothing else is committed to in advance.

---

## Notes for the working sessions

- The PoC branch is not planned to merge back to `main`. Commit directly on the branch; no PR ceremony.
- Conventional Commits subject lines (`feat(aiwf): ...`, `chore(aiwf): ...`, `docs(poc): ...`) keep the log readable.
- If session 3's deliverable is not reached within a reasonable timebox, abandon and patch the existing framework instead. The PoC's value is bounded; do not over-invest.
- When in doubt, the smaller change is the right change. KISS and YAGNI from the root [`CLAUDE.md`](../../../CLAUDE.md) are load-bearing here.
