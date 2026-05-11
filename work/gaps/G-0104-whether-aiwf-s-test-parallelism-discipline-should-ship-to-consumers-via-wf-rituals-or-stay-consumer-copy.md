---
id: G-0104
title: Whether aiwf's test-parallelism discipline should ship to consumers via wf-rituals or stay consumer-copy
status: open
discovered_in: E-0025
---
# Problem

E-0025 establishes a test-parallelism + fixture-sharing discipline inside aiwf — `TestMain` for env setup, `t.Parallel()` default-on, `sync.Once` for expensive shared fixtures, `-parallel 8` race cap — documented in `CLAUDE.md` and chokepoint-enforced via a `setup_test.go`-presence policy test under `internal/policies/`. The discipline solves a real problem (G-0097) and the spike numbers (~4× faster non-race, ~2.4× with race) confirm the headroom.

Downstream consumers of aiwf inherit the binary, the data model, and the `aiwf-*` skills. They **do not** inherit aiwf's `CLAUDE.md` or `internal/policies/`. A consumer that grows past trivial test-suite size will eventually hit the same wall aiwf hit — serial execution, per-test fixture rebuild, race-detector flakes — and may blame "aiwf-shaped repos" for the slowness rather than recognizing it as a generic Go-testing pattern.

# Question

Should aiwf ship this discipline to consumers, and if so, how?

Two paths:

1. **Opt-in via `wf-rituals`.** Either extend the existing `wf-tdd-cycle` skill with a "make tests fast" appendix, or add a new optional skill (`wf-test-perf` / `wf-fast-tests`). Consumers who install the rituals plugin get the playbook; consumers who don't, don't. This is the closest analog to how aiwf already ships discipline (TDD cycle, code review, doc lint, patch ritual).
2. **Pure-copy.** Consumers who want the discipline copy aiwf's `CLAUDE.md ## Test discipline` section into their own `CLAUDE.md`. No shipped artifact, no mechanical channel. This is what E-0025 assumes by default.

The author of E-0025 is also a consumer and will copy the section into their own consumer-repo `CLAUDE.md` regardless. So this question is about *whether to prescribe* for all potential future consumers, not about whether the discipline is useful.

# Tradeoffs

| Path | Pro | Con |
|---|---|---|
| Opt-in skill in `wf-rituals` | Mechanical channel; discoverable via plugin install; AI-discoverable per CLAUDE.md's "kernel functionality must be AI-discoverable" principle | Adds a maintained skill; the convention drifts independently of consumer projects' actual test setups; not all consumers want Go-shaped test discipline (or are even using Go) |
| Pure-copy | Zero ongoing maintenance; consumers BYO discipline; respects YAGNI | Convention only reaches consumers who happen to read aiwf's CLAUDE.md; no chokepoint at the consumer end; same problem may recur in N consumer repos before anyone notices |

# Not blocking

This gap is **not urgent**. E-0025 ships the aiwf-side discipline regardless of how this resolves. The question becomes interesting once a second consumer (besides the author) hits the same wall and asks for guidance — at that point the gap has signal and a decision becomes easier.

# Resolution shape

When this gap closes, the resolution is either:

- An ADR + skill addition in the rituals plugin (path 1), with a contract on what the skill covers and what stays in consumer hands.
- A "wontfix" close with rationale (path 2): "consumer-copy is the working assumption; we'll revisit if multiple consumers report friction."

The shape is decision-shaped, not implementation-shaped — `aiwf promote G-NNNN wontfix` or a small follow-up epic, not a big body of work.
