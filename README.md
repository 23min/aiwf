# ai-workflow

> Disciplined AI-assisted software engineering, with the human always in the loop. AI writes; the framework validates; humans decide.

ai-workflow is a framework for software repositories where humans and AI assistants build software together over long horizons. It is human-in-the-loop by design: AI does the creative work — writing code, drafting decisions, authoring documentation — humans make the calls that matter, and a deterministic engine in between records every structural change as an event, validates it against typed contracts, and projects it into a hash-verified read-model. The result is a project history you can audit, replay, and trust, even when most of the work was AI-generated.

---

## Why this exists

AI assistants generate code, decisions, and documentation faster than projects accumulate the trail of who decided what, when, and why. We are entering what might be called the *Prove-It era* of AI-assisted software development: the moment when "the AI wrote it" stops being a sufficient answer to "how did this get here?" Compliance, engineering hygiene, team trust, and the basic comprehensibility of long-running projects all push the same way — every structural change to a project should be answerable.

ai-workflow is provenance-first by construction. Every structural change is recorded as a typed event before its effect is applied. Every event names the actor (human, AI assistant, CI), the correlating intent, and a content hash. The project's read-model is reproducibly derived from that log; if it ever diverges, the framework reports it. The result is an auditable, replayable project history that survives long-running, multi-agent, AI-assisted work without losing its receipts.

---

## Determinism, by construction

The framework's engine is deterministic. Given the same event log, it produces the same projection — bit-for-bit, every time, on every machine. This is what makes audit and replay cheap: any divergence between the projection on disk and the projection re-derived from the events is detectable in O(1) by hash comparison.

AI assistants are stochastic; project state cannot afford to be. The framework draws that line and holds it: AI proposes through typed actions, the engine validates and persists deterministically, humans review and decide. The boundary is sharp on purpose.

---

## How it fits together

```
                                 ┌────────────────────────────┐
   ┌───────────────────┐         │         The engine         │
   │  AI assistant     │ propose │                            │
   │  (Claude, Copilot,├────────►│  validate → trace-first    │
   │   GPT, others)    │         │  append event → apply      │
   └─────────▲─────────┘         │  effect → confirm event    │
             │                   │                            │
             │ findings          │  +  hash-verified           │
             │                   │     projection (graph.json) │
             │                   │  +  audit pipeline          │
             │                   └────────────┬───────────────┘
             │                                │
             │             ┌──────────────────┘
             │             │ surfaces (structured findings,
             │             │ propagation previews, queries)
             │             ▼
             │   ┌────────────────────┐
             └───┤  Human in the loop │
                 │  (review, decide,  │
                 │   author intent)   │
                 └────────────────────┘
```

Three actors. AI assistants compose typed action envelopes; the engine validates against per-kind boundary contracts and applies effects through trace-first writes; humans review the propagation preview and make the calls that matter. Markdown specs in the repo are the shared source of truth; everything else (the event log, the projection, the adapter surfaces) is derived or generated.

For the full design see [`docs/architecture.md`](docs/architecture.md).

---

## Status

**Preview / under construction.** This is a clean rewrite of an internal framework, being developed in the open. The architecture document is settled; the implementation lands in stages (see [`docs/build-plan.md`](docs/build-plan.md)). Not production-ready.

**Engagement:**

- [Issues](../../issues) track bugs and confirmed work items. Two templates: `bug` (against shipped behavior) and `design-question` (clarifies the architecture doc).
- [Discussions](../../discussions) host design RFCs, feature ideas, and "should we?" questions. Feature ideas start here; once a discussion converges, it graduates to an issue.
- Pull requests reference an issue or discussion in their description. New here? Read `docs/architecture.md` first; most "should we add X?" questions have an answer there.

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the full engagement model.

---

## Get started

> Stages 2-3 add the engine and the install script. Until then, this section sketches the intended flow; commands work from Stage 3 onward.

**Recommended (manual, auditable):**

```bash
git submodule add https://github.com/23min/ai-workflow-v2.git .ai
cd .ai && git checkout v0.1.0   # pin to a release
cd .. && bash .ai/install.sh
```

The installer detects existing repository structure (looks for `docs/decisions/`, `ROADMAP.md`, `CLAUDE.md`, etc.), suggests modules to enable, shows a preview of what it will create or modify, and confirms before writing anything. It never overwrites consumer-owned files — fenced sections are appended with delimiters the engine can manage on subsequent runs.

**Convenience (one-liner; review before running):**

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/23min/ai-workflow-v2/main/install.sh)
```

Both forms are idempotent. Re-running reconciles state; it does not redo.

A `go install` path for the binary lands in Stage 3 once `aiwf` is shippable.

---

## Documents

- **[Architecture](docs/architecture.md)** — data model, event-sourced kernel, LLM/engine boundary, verbs, contracts, modules, principles checklist.
- **[Build plan](docs/build-plan.md)** — what gets built, in what order, and the friction-removal principles guiding consumer onboarding.
- **[Contributing](CONTRIBUTING.md)** — engagement model, issue/PR conventions, pre-PR audit.
- **[Engineering principles](CLAUDE.md)** — the discipline contributors are expected to apply.

---

## License

Apache-2.0. See [`LICENSE`](LICENSE) and [`NOTICE`](NOTICE).
