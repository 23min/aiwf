# CLAUDE.md — ai-workflow repo

This repo **is** the ai-workflow framework (the thing consumer repos check out as `.ai/` and bootstrap via `aiwf init`). It is not a consumer of itself — there is no `.ai/` directory here, no `.ai-repo/`, no generated `.claude/skills/` or `.github/skills/` adapters. The framework's own work tracking is `ROADMAP.md` plus GitHub Issues, not the entity model the framework produces for its consumers.

For the technical design see `docs/architecture.md`. For the build sequence see `docs/build-plan.md`. This document is the engineering discipline that governs how changes land.

---

## Engineering principles

- **KISS — keep it simple.** Prefer the boring solution. Three similar lines beats a premature abstraction. Avoid cleverness — reflection, metaprogramming, deeply nested generics, control-flow tricks — unless the simple version is demonstrably worse.
- **YAGNI — don't build for tomorrow.** No speculative interfaces, no "we might need this later" config knobs, no plugin architectures for a single implementation. Add the second case when it shows up; abstract on the third. The user can override the third-user heuristic with explicit direction (e.g. "extract `internal/common` now") — that's a deliberate choice, not a rule violation.
- **No half-finished implementations.** If a feature lands, it lands tested and documented. Stubs and TODOs in shipped code are a smell, not a milestone.
- **Future references must cite an open issue.** Skill / template / doc / changelog prose may describe future capability only when phrased as deferred work and naming the issue (e.g. `… tracked in #NN`). Never describe a future surface as if it exists today — readers can't tell whether the integration is built or planned, and prose that lies about the present rots fastest.
- **Errors are findings, not parse failures.** Tools load inconsistent state and report it; they don't refuse to start. Validation is a separate axis from loading.

For Go-specific rules (formatting, linting, testing, coverage, error handling, CLI conventions), see `tools/CLAUDE.md`.

---

## Architectural commitments

These are non-negotiable. They derive from the architecture document and apply to every change.

- **Trace-first writes.** Every structural mutation appends an event to `events.jsonl` *before* applying its effect. Never reverse the order. The event is recorded as a fact about what is being attempted; the confirmation event is what marks success. Recovery from partial failure is forward-only via the trace.
- **Immutability of done.** Terminal-state entities (`complete`, `cancelled`) never reverse. Defects in completed work spawn new entities via `aiwf hotfix`. Never add a verb that allows un-completing.
- **The engine never generates prose.** No "smart commit messages," no "AI-summarized status reports," no narrative output. The engine emits structured data; assistants and renderers shape it for humans.
- **The assistant never writes the projection or event log directly.** Always via engine verbs. Direct edits bypass validation, atomicity guarantees, and the propagation preview.
- **The assistant never invents IDs.** Engine allocates. Otherwise parallel sessions race and produce collisions.
- **The engine is invocable without an AI assistant.** Every binary takes flags, reads stable input formats, emits a JSON envelope, exits with documented codes. Humans, CI scripts, and other tools drive it directly. The AI assistant is a convenience for orchestration, not a dependency.
- **Hash-verified projections.** Use RFC 8785 (JSON Canonicalization Scheme) for canonicalization. Never roll your own. SHA-256 of the canonical form is what lets `aiwf verify` detect drift in O(1).
- **Closed-set vocabularies declared in YAML, validated in Go.** YAML for the data that varies per kind (boundary contracts). Go for the logic that interprets it. Don't grow the YAML into a Turing-complete constraint language.
- **Skills are small, composable, lazy-loaded.** Target 50-150 lines per skill. Load skill bodies on invocation, not at session start. Compound flows compose smaller skills; they are not monoliths.

---

## The principles checklist

When evaluating any proposed change, walk this checklist (it is also the framework's §12 in `architecture.md`, reproduced here as the rubric for PRs):

1. Does it move structural truth toward the engine, or away from it? Toward is good.
2. Does it move content toward the assistant, or away from it? Toward is good.
3. Does it require assistant judgment to validate structure? If yes, the schema is missing a declaration; fix the schema, don't add cleverness.
4. Does it require engine support for content? If yes, the engine is sliding toward generating prose. Stop.
5. Does it preserve trace-first ordering? Every mutation must record an event before applying its effect.
6. Does it preserve the projection's hash-verifiability? If a change makes the projection non-canonical or non-replayable, it's wrong.
7. Does it preserve atomicity? A change that can leave structure in a half-mutated state needs a recovery story.
8. Does it close a verb-set gap or open a new one? Symmetric verb sets are easier to reason about than asymmetric ones.
9. Does it auto-do something a human should decide? If yes, replace the auto-do with a propagation preview.
10. Could a non-AI consumer (a CI script, a different agent, a human at the CLI) drive this with the same outcome? If no, the engine has grown too dependent on a specific assistant.

---

## Pre-PR audit

Before opening a PR that touches a directory governed by a `CLAUDE.md` file (root or scoped), walk the diff against the rules in that file and report rule conformance in the PR description. Treat this as part of the work, not a follow-up — the rules exist to prevent the class of failure the PR description should already have eliminated.

For Go work under `tools/`, the conformance items are listed in the `tools/CLAUDE.md` "Pre-PR checklist." For framework-source changes (skills, templates, contracts, install script), the rules in this file plus directory-scoped files apply.

**Every PR with a user-visible change must add an entry to `CHANGELOG.md` under `[Unreleased]`** — bug fix (`### Fixed`), new capability (`### Added`), behavior change (`### Changed`), or removal (`### Removed`). Cite the issue (`(#NN)`) and lead with the user-observable effect, not the diff. Internal-only refactors with no observable effect can skip; when in doubt, add the entry. Verify in the pre-PR audit that the entry is present.

If a rule needs to be relaxed for a specific case, propose the rule change in the same PR or a separate prerequisite PR — don't silently violate.

---

## What lives where

- `framework/modules/` — the modules. Each module is self-contained: skills, contracts, schemas, templates, adapter rules.
- `framework/schemas/` — JSON Schemas for the action envelopes the engine validates.
- `framework/contracts/` — base boundary contracts for the always-present kinds (epic, milestone).
- `tools/cmd/aiwf/` — the single binary's entry point.
- `tools/internal/` — engine internals: eventlog, projection, verify, validate, mutate.
- `install.sh` — the consumer-facing installer. Idempotent. Detects existing layout; doesn't clobber consumer files.
- `tests/` — test scripts and golden fixtures. Synthetic content only.
- `docs/` — framework documentation. `architecture.md` (technical design), `build-plan.md` (build sequence), this file, plus per-topic deep-dives as they're added.
- `ROADMAP.md`, `CHANGELOG.md` — what's planned / what shipped.
- `README.md` — quick-start for new consumers.

---

## How to validate changes

```bash
go test -race ./tools/...                 # unit tests
golangci-lint run                         # linters
bash tests/test-install.sh                # sandbox test of the installer
```

All three should pass before opening a PR. CI runs all of them on every PR.

There is no devcontainer. Work directly against macOS / Linux; the installer must stay portable across both.

---

## Changing framework sources

- **Skill** (`framework/modules/<name>/skills/*.md`): edit; the test suite validates that adapters generate cleanly for both Claude (`.claude/skills/wf-<name>/SKILL.md`) and Copilot (`.github/skills/<name>/SKILL.md`) hosts.
- **Boundary contract** (`framework/modules/<name>/contracts/*.yaml`): changes the engine's rules for that kind. The validator must be extended in the same PR. Changes are breaking for consumers that relied on prior behavior — bump the framework changelog and document the migration.
- **Action schema** (`framework/schemas/actions/*.json`): the LLM/engine boundary contract. Schema changes are versioned; the engine accepts events at any prior schema version it has migration code for. Adding a new action requires its schema, the engine handler, and the contract entries that authorize it per kind.
- **Template** (`framework/modules/<name>/templates/*.md`): pure scaffolding. Renders should be deterministic given inputs; tests assert this with golden files.
- **Installer** (`install.sh`): every change is a potential break for existing consumers. Run `tests/test-install.sh` against fresh and pre-installed fixtures.

---

## Work tracking

This repo does **not** dogfood the framework yet — the engine isn't stable enough to track its own construction without a bootstrap problem. Until it is, work tracking lives in three text artifacts plus GitHub, wired together so CI can enforce the link between them.

### The three layers

1. **Plan** — `ROADMAP.md` (long view) and `docs/build-plan.md` (ordered build sequence). Each row in `build-plan.md` maps to one tracking Issue. New work that is not on the build plan needs justification in the issue's "Why now" field; if the deviation reflects a real change in direction, update `build-plan.md` in the same PR.
2. **Spec** — the Issue body. Every non-trivial Issue uses the `task` template, which requires explicit **Acceptance criteria** (a bullet checklist where each bullet is independently verifiable — by a command, a file, or a test) and a **Principles-checklist risks** section. The PR description re-asserts the acceptance bullets and shows how each is met. There is no separate spec doc per task; the Issue *is* the spec.
3. **Verify** — CI (`.github/workflows/pr-conventions.yml`) mechanically enforces: PR title is a Conventional Commit, PR body cites an Issue or Discussion, `CHANGELOG.md` was modified under `[Unreleased]` (skip with the `internal-only` label). Substantive review — "is this actually the right thing?" — stays human, against the principles checklist.

### Issue types

- **`task`** — the workhorse. A unit of planned work; pins to a build-plan row when one exists. Use this for anything that isn't a bug or a design question.
- **`bug`** — shipped behavior diverges from documented behavior.
- **`design-question`** — `docs/architecture.md` is unclear, inconsistent, or missing rationale.
- Feature ideas and "should we?" questions are **not** Issues until they have converged. They start as GitHub Discussions; once a direction is agreed, the Discussion graduates to a `task` Issue that cites it.

### PR conventions

- Every PR description cites the Issue or Discussion that established the work is wanted. Drive-by PRs without prior conversation will be asked to open one.
- The PR template (`.github/pull_request_template.md`) prompts for: the closing `Closes #NN` line, re-asserted acceptance criteria, principles-checklist conformance notes, and CHANGELOG confirmation.
- Commit-message style: Conventional Commits (`feat(...)`, `fix(...)`, `docs(...)`, `chore(...)`, `refactor(...)`, `test(...)`, `build(...)`, `ci(...)`, `perf(...)`, `revert(...)`). One commit per logical change; small commits beat large ones for review.
- The `internal-only` label exempts a PR from the CHANGELOG-touch check. Use it sparingly: refactors with no observable effect, CI-only tweaks, comment fixes. When in doubt, add a CHANGELOG entry.

### When to revisit this

Once the engine ships milestone tracking and `aiwf verify` is stable for at least one release, revisit whether to migrate this repo's work tracking onto its own framework. At that point dogfooding strengthens the product instead of risking it.

---

## When assisting in this repo

The work here is *meta*: changes to skills and contracts change how AI behaves in *other* repos that consume this framework. Think about downstream impact, not just local correctness. The architecture document's principles checklist is the rubric — apply it.

Don't scaffold `.ai-repo/` or entity directories in this repo. Those belong in consumer repos. This repo's work tracking is `ROADMAP.md` plus GitHub Issues plus Discussions.

If a change to the installer or any module surface might break existing consumers, note the migration path in the PR description and in `CHANGELOG.md`.
