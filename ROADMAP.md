# Roadmap

> **Note:** This roadmap predates the research arc in [`docs/research/`](docs/research/) and is preserved as historical context. The stages below target the earlier event-sourced architecture that the research walked back. The active near-term work is the PoC described in [`docs/research/06-poc-build-plan.md`](docs/research/06-poc-build-plan.md), being built on the `poc/aiwf-v3` branch.

Tracks where the framework is in the build sequence described in [`docs/build-plan.md`](docs/build-plan.md). Updated as stages land.

## Status legend

- ✅ done
- 🚧 in progress
- ⏳ planned, not yet started

## Stages

### Stage 1 — Architecture and ground rules ✅

Foundational documents in place: architecture, build plan, engineering principles, contributing guide, CI scaffold (scrub, markdown lint, link check). No engine code yet.

### Stage 2 — The kernel 🚧

The smallest viable engine: events.jsonl with the closed envelope schema, RFC 8785 canonicalization and SHA-256 chaining, the pure-function projection from events to a derived read-model, `aiwf init` and `aiwf verify`. Synthetic test fixtures only.

PR breakdown:

- 🚧 PR 1 — Go infrastructure scaffold (this PR). `go.mod`, `.golangci.yml`, `Makefile`, Go CI, stub `aiwf` binary, envelope contract locked in tests.
- ⏳ PR 2 — Event envelope: JSON Schema, Go types, RFC 8785 canonicalization, SHA-256 hashing.
- ⏳ PR 3 — Append-only event log: flock-protected `O_APPEND` writes, streaming reader, idempotency-key handling.
- ⏳ PR 4 — Projection: pure function from events to graph; canonicalization of the projection itself; hash storage.
- ⏳ PR 5 — `aiwf verify` verb: replay + hash-compare; emit findings on divergence.
- ⏳ PR 6 — `aiwf init` verb: detect existing layout; bootstrap `.ai-repo/` with a `run.init` event.

### Stage 3 — Core verbs and required modules ⏳

Adds `add`, `promote`, `pause`, `resume`, `block`, `unblock`, `cancel`, `remove`, `rename`, `hotfix`. Boundary contracts for `epic` and `milestone`. Read verbs (`query`, `transitions`, `history`, `render`, `branch-name-for`, `template-for`). Skill-index machinery. Audit pipeline with entity-integrity checks.

### Stage 4 — Optional modules ⏳

Default-on modules first (`adr`, `roadmap`, `narrative`, `decisions`, `gaps`), then default-off (`contracts`, `github-sync`, `release`, `doc-lint`). Each module ships in its own PR.

### Stage 5 — Validation ⏳

A complete milestone cycle (plan → start → wrap) on a real project, using only what's shipped. No escape hatches.

### Stage 6 — v0.1.0 ⏳

First tagged release. Marks "preview is over."

---

For the architectural design see [`docs/architecture.md`](docs/architecture.md). For the build philosophy and friction-removal principles see [`docs/build-plan.md`](docs/build-plan.md).
