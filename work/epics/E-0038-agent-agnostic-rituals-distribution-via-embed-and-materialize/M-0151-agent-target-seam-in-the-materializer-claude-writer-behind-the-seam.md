---
id: M-0151
title: Agent-target seam in the materializer (Claude writer behind the seam)
status: done
parent: E-0038
depends_on:
    - M-0149
    - M-0150
tdd: required
acs:
    - id: AC-1
      title: Materializer takes a target param; Claude target preserves M2/M3 behavior
      status: met
      tdd_phase: done
    - id: AC-2
      title: Seam contract test asserts target-to-output mapping; accepts a 2nd target
      status: met
      tdd_phase: done
---
## Goal

Refactor the materializer to be parameterized by an agent target, with the Claude writer (`.claude/skills/`, `.claude/agents/`) implemented behind the seam, so additional targets (Codex `.agents/skills/`, down-converted Cursor/Copilot rules) become new writers rather than a rewrite.

## Context

M2/M3 wrote the Claude locations concretely. This milestone extracts the target seam now that there is a concrete writer to abstract over — per CLAUDE.md KISS/YAGNI, abstract on the second case, not speculatively. It unblocks the agent-agnostic future without building every target. The seam may be pulled forward into M2 if that proves cheaper at implementation time; that is a just-in-time call, not a planning-time one.

## Acceptance criteria

## Constraints

- Behavior-preserving for the Claude target — M2/M3 tests stay green with no observable change for Claude consumers.
- Do not build out non-Claude target writers here (deferred per the epic scope and the M6-deferral gap).

## Design notes

- ADR-0014 §4 (agent-target abstraction). SKILL.md is a cross-vendor open standard (agentskills.io; OpenAI Codex reads the identical frontmatter from `.agents/skills/`), so the first non-Claude target is near-verbatim.
- CLAUDE.md KISS/YAGNI — the seam is *extracted* from a concrete writer, not speculated ahead of one.

## Surfaces touched

- `internal/skills/` (materializer target parameterization), the materialize call sites in `init`/`update`.

## Out of scope

- Implementing Codex / Cursor / Copilot writers — deferred to a follow-up gap.

## Dependencies

- M2 and M3 — the concrete Claude writers this milestone abstracts over.

## References

- **ADR-0014** (§4), **E-0038**.

### AC-1 — Materializer takes a target param; Claude target preserves M2/M3 behavior

### AC-2 — Seam contract test asserts target-to-output mapping; accepts a 2nd target

## Work log

### AC-1 — target-parameterized materializer; Claude preserves behavior
Introduced `Target{Name, SkillsDir, AgentsDir, TemplatesDir}` and `ClaudeTarget` (pinning today's `.claude/{skills,agents,templates}`), and `MaterializeTo(root, target)`. `Materialize(root)` is now a thin `MaterializeTo(root, ClaudeTarget)` wrapper, so every init/update call site and all M-0149/M-0150 tests stay green with no observable change. An empty `AgentsDir` makes the agent writer a no-op (ADR-0014 §4: a host with no subagent concept). · tests: `TestMaterializeTo_ClaudeTarget`, `TestMaterialize_DefaultsToClaude`

### AC-2 — seam contract; second target accepted
Contract tests assert the target→output mapping and prove extensibility against a Codex-shaped second target (`.agents/skills/`, ADR-0014 §4) — every artifact kind routes to that target's dirs and nothing lands under `.claude/`. A no-agent target exercises the empty-`AgentsDir` skip. The second target is a test fixture; no production non-Claude writer ships (epic scope). · tests: `TestMaterializeTo_SecondTarget`, `TestMaterializeTo_NoAgentTarget`

The per-phase red→green→done→met timeline is authoritative in `aiwf history M-0151/AC-<N>`.

## Decisions made during implementation

- No new ADR/decision entity. The seam realizes ADR-0014 §4 directly; the only sub-decision (empty `AgentsDir` ⇒ agent no-op) is the ADR's own "no subagent concept" case, implemented as written.

## Validation

- **Full local CI gate set run on the epic branch** (the gap this milestone surfaced — see Reviewer notes / G-0179): `golangci-lint run ./...` **0 issues**; `go vet ./...` clean; `go build ./...` clean; `go test ./internal/skills/ ./internal/initrepo/` green; `go test -race -parallel 8` (changed pkgs) green; `go test ./internal/policies/` green; `aiwf doctor --self-check` passed (30 steps).
- **Behavior-preservation smoke:** real `aiwf init` materialized 29 skills + 4 agents + 4 templates into `.claude/`, with no `.agents/` leak — identical to the M-0150 layout.
- **Latent-lint remediation:** `golangci-lint` (not previously run on this branch) surfaced 9 CI-blocking findings introduced across M-0149/M-0150 (5 `govet shadow`, 4 `gocritic stringXbytes`); all fixed on this branch. The `skills.go` fixes are pure error-variable renames (behavior-identical); the test fixes switch `string(a)!=string(b)` to `bytes.Equal`.

## Deferrals

- Non-Claude target **writers** (Codex `.agents/skills/`, Cursor/Copilot down-converted rules) remain out of scope per the epic — the seam unblocks them as new values, but none ship here.
- **G-0179** (filed on main) — enforce the full local CI gate (`golangci-lint` et al.) mechanically at milestone/epic wrap on unpushed branches, so latent lint does not recur on the next long-lived epic branch.

## Reviewer notes

- **Wrapper, not a signature break.** `Materialize(root)` is retained as `MaterializeTo(root, ClaudeTarget)`, so no caller or existing test changed — the seam is additive. AC-1's "preserves M2/M3 behavior" is enforced by the unchanged M-0149/M-0150 suites staying green.
- **`ClaudeTarget` is a package `var`, not a `const`** (structs cannot be const). It is a pseudo-constant config value, consistent with the existing `var embedFS`/`var ritualsFS`; not mutated anywhere.
- **Second target is a test fixture.** AC-2 proves extensibility without shipping speculative writers (KISS/YAGNI) — a real Codex/Cursor writer is a future gap behind this seam.
- **Latent golangci-lint debt discovered (→ G-0179).** This milestone's self-audit ran `golangci-lint` for the first time on the epic branch and found 9 findings dating to M-0149, invisible because CI only runs on push and prior wraps validated with `go vet` (which does not enable `govet: enable-all` / gocritic). All fixed here; the systemic fix is tracked in G-0179. The fixes touch M-0149/M-0150 test files, so they appear in this milestone's diff by necessity.

