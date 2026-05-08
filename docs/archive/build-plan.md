# Build plan

> **Note — archived.** This document predates the research arc in [`../research/`](../research/) and is preserved as historical context. The build sequence below targets the earlier event-sourced architecture that the research walked back. See [`../research/06-poc-build-plan.md`](../research/06-poc-build-plan.md) for the current PoC plan.

**Status:** plan of record. This document describes what gets built, in what order, and the principles that guide consumer onboarding. Companion to `architecture.md`, which is the technical design this plan implements.

---

## What this is

ai-workflow is a markdown-native project-management framework for repositories where humans and AI assistants collaborate over long horizons. The architecture document covers what the framework *is*; this document covers how it *gets built*.

The build is staged. Each stage lands a coherent capability and is usable on its own. Consumers can adopt at any stage; later stages add capability without breaking earlier ones.

---

## Stages

### Stage 1 — Architecture and ground rules

**Status:** complete (this commit).

The repository opens with the foundational documents in place:

- `docs/architecture.md` — the technical design.
- `docs/build-plan.md` — this document.
- `CLAUDE.md` — engineering principles for contributors.
- `tools/CLAUDE.md` — Go-specific discipline for the engine.
- `CONTRIBUTING.md` — how engagement works during the build.
- `LICENSE`, `NOTICE` — Apache-2.0.
- `.github/` — issue templates (bug, design-question), workflow files (legacy-identifier scrub, markdown lint, link check).

No engine code yet. The architecture is the artifact that justifies the rest.

### Stage 2 — The kernel

The smallest viable engine. Lands the event-sourced kernel without any verbs that mutate structural state:

- The `aiwf` binary, single-binary subcommand-dispatched.
- The append-only event log (`.ai-repo/events.jsonl`) with the closed envelope schema, RFC 8785 canonicalization, and SHA-256 chaining.
- The pure-function projection from events to a derived read-model (`.ai-repo/graph.json`).
- `aiwf init` — bootstrap a consumer's `.ai-repo/` from an existing markdown tree (or empty).
- `aiwf verify` — replay + hash-compare; emits structured findings on drift.

Discipline: 100% line coverage on internal packages, race-detector clean, synthetic test fixtures only.

At the end of Stage 2, a consumer can `aiwf init` an empty repository and `aiwf verify` it. The kernel runs; nothing structural happens yet.

### Stage 3 — Core verbs and required modules

The base capability. After Stage 3 a consumer can manage epics and milestones end-to-end:

- Write verbs: `add`, `promote`, `pause`, `resume`, `block`, `unblock`, `cancel`, `remove`, `rename`, `hotfix`.
- Read verbs: `query`, `transitions`, `history`, `render`, `branch-name-for`, `template-for`.
- The `core` plus `epic` and `milestone` modules with their boundary contracts and templates.
- The skill-index machinery (lazy-loaded skill bodies; eager index).
- The audit pipeline skeleton with entity-integrity checks.

At the end of Stage 3, the framework is usable for a real project that doesn't need ADRs, contract bundles, or other extensions.

### Stage 4 — Optional modules

Each module ships in its own PR. Suggested order:

1. `adr` — ADR (Architecture Decision Record) tracking. Default-on.
2. `roadmap` — `ROADMAP.md` fenced-section regeneration. Default-on.
3. `narrative` — `CLAUDE.md` fenced-section maintenance. Default-on.
4. `decisions` — lightweight decisions table. Default-on.
5. `gaps` — gaps tracking. Default-on.
6. `contracts` — contract bundles + per-PR contract verification. Default-off.
7. `github-sync` — GitHub Issues mirror + drift detection. Default-off.
8. `release` — versioning + release tagging + tag-existence pre-flight. Default-off.
9. `doc-lint` — narrative drift detection (does the prose match the structural state?). Default-off.

A module's "default-on" or "default-off" status determines whether `aiwf init` enables it without explicit opt-in. The split reflects what most consumers want; both are easily overridden.

### Stage 5 — Validation

Before declaring v0.1.0, exercise the framework against a real project for a complete milestone cycle (plan → start → wrap). Note every friction point. File issues for everything that surprises.

Constraint: use only what's shipped. No "the feature isn't there yet, I'll just hand-edit" — that defeats the test. When the milestone wraps cleanly without escape hatches, Stage 5 is done.

### Stage 6 — v0.1.0

First tagged release. Marks "preview is over; this is usable but not stable." Subsequent releases follow semver: minor bumps for new modules and additive verbs; major bumps only for breaking changes to the event envelope or the boundary-contract format.

---

## Modules and pluggability

The framework ships as a small **core kernel** plus **opt-in modules**. New consumers don't carry capability they don't use; existing consumers can adopt new modules incrementally.

### Module structure

Every module ships in `framework/modules/<name>/` with this shape:

```text
framework/modules/<name>/
├── MODULE.yaml          # metadata: name, description, version, requires, provides, default
├── README.md            # human-readable doc
├── skills/              # markdown skills the module exports (lazy-loaded)
├── contracts/           # boundary contracts for any new entity kinds
├── schemas/             # JSON schemas for any new actions
├── templates/           # templates for scaffolding
└── adapters/            # rules-injection text per host (claude, copilot)
```

A module declares what entity kinds it provides, what actions it adds to the engine's vocabulary, what skills it exports, and what conditions in a consumer repo should suggest enabling it.

### Onboarding flow

`aiwf init` is the single entry point. It:

1. Detects existing structure in the consumer repo (looks for `docs/decisions/`, `ROADMAP.md`, `CLAUDE.md`, etc.).
2. Suggests modules based on detection plus the default-on/default-off split.
3. Shows a preview of what will be created or modified.
4. Confirms with the user.
5. Writes `.ai-repo/config/modules.yaml` and `.ai-repo/events.jsonl` with a `run.init` event.
6. Generates the consumer's adapter surfaces (`.claude/skills/`, `.github/skills/`, framework section in `CLAUDE.md`).
7. Runs `aiwf verify` to confirm consistency.

The installer never overwrites consumer-owned files — it appends fenced sections to `CLAUDE.md` and similar files with clear delimiters that subsequent `aiwf` runs manage automatically.

### Friction-removal principles

These guide every choice in the installer and module system:

1. **Detection over prescription.** If a directory exists that maps to a module, suggest enabling the module. Don't force a layout.
2. **Smart defaults.** The default-on/default-off split reflects what most consumers want. Override is opt-in, not opt-out.
3. **No hidden state.** Every config decision lands in a file the consumer can see and edit (`.ai-repo/config/modules.yaml`). The installer never persists a setting silently.
4. **Reversible at every step.** Disabling a module removes its skills from the adapter surface but does not delete entity data. Re-enabling restores it.
5. **Respect existing content.** Never overwrite consumer-owned files. Append fenced sections that the engine can manage.
6. **One command bootstrap.** `git submodule add <url> .ai && bash .ai/install.sh` works for everyone. A `bash <(curl ...)` form is offered as a convenience for those who want it.
7. **Idempotent.** Running `aiwf init` again on an installed repo reconciles, doesn't redo. Re-running after enabling a new module just adds the new surface area.
8. **Errors as findings.** Even at install time. If the installer can't satisfy a request, it reports a structured finding and continues with everything else.

### What modules deliberately don't do

Per the framework's KISS / YAGNI commitments:

- No plugin marketplace.
- No remote module loading.
- No third-party module API — modules ship in this repo only, added via PR.
- No dynamic dependency resolution at runtime.
- No per-module versioning — modules version with the framework.

If any of these become genuinely needed, they get filed as design Discussions. Not before.

---

## Engagement model during the build

This repository is in preview. The architecture is settled; the implementation is being built in stages. Engagement is welcome but expected to be informed.

**Issues** track bugs and confirmed work items. Two templates ship:

- **Bug** — against shipped behavior. Requires version, reproduction, expected vs actual.
- **Design question** — clarifies the architecture document. References the section in question.

There's no "feature request" template by design. Feature ideas route to **Discussions**, where they can be argued out before they become Issues.

**Pull requests** are gated by prior conversation. A PR description references the Issue or Discussion that established the work is wanted. Drive-by PRs without prior conversation will be asked to start one.

This split keeps signal high during the build phase. It also means new contributors should read the architecture document before opening anything — most "should we add X?" questions are already answered in `architecture.md`.

---

## Success criteria

The build is successful when:

1. A new consumer can run `aiwf init` and have a working setup within 10 minutes.
2. A complete milestone cycle (plan → start → wrap) runs end-to-end without manual edits to structural files.
3. The architecture document and engineering principles have been stable for at least 30 days post-v0.1.0.
4. CI runs all checks on every commit and has been green since the founding commit.
5. At least one external user (not the framework's author) has used it on a real project and reported back.

When all five hold, the framework is no longer "in preview." It's a usable tool with a stable enough surface to recommend to others.
