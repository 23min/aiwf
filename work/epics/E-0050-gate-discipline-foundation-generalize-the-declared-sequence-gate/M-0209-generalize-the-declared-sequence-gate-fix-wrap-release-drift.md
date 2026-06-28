---
id: M-0209
title: Generalize the declared-sequence gate; fix wrap/release drift
status: draft
parent: E-0050
tdd: advisory
acs:
    - id: AC-1
      title: Generalized declared-sequence gate documented in CLAUDE.md and guidance
      status: open
    - id: AC-2
      title: aiwfx-release splits the two origin pushes into separate push gates
      status: open
    - id: AC-3
      title: aiwfx-wrap-milestone batches its terminal local steps in one gate
      status: open
    - id: AC-4
      title: aiwfx-wrap-epic batches promote-done and cleanup in one gate
      status: open
---

## Goal

Generalize the wf-patch declared-sequence gate (CLAUDE.md §"Gate discipline
survives compaction") into a general capability for any sequence of local,
reversible mutations — one gate that enumerates every action verbatim, binds
approval to exactly that list (subset approval allowed), and aborts + re-gates on
any deviation. Document the standing rule in CLAUDE.md's gate-discipline section
and `.claude/aiwf-guidance.md`, rewriting the false "wf-patch only; milestone and
epic wraps keep per-action gates" scope sentence.

Then fix the three rituals that violate it today: `aiwfx-release` (split the
bundled two-push gate into two separate push gates), `aiwfx-wrap-milestone` and
`aiwfx-wrap-epic` (replace the ungated promote / merge / cleanup steps with the
declared-sequence gate, push excluded). The bright line — batch local, reversible
mutations; exclude outward / irreversible actions and timing-bearing mutations
(`tdd: required` phase promotes fire live) — is the load-bearing safety claim,
pinned by structural tests under `internal/policies/`.

Source: G-0295. Extracted from E-0049 into foundation epic E-0050 so both E-0048
and E-0049 milestone wraps inherit the corrected gate.

## Acceptance criteria

### AC-1 — Generalized declared-sequence gate documented in CLAUDE.md and guidance

### AC-2 — aiwfx-release splits the two origin pushes into separate push gates

### AC-3 — aiwfx-wrap-milestone batches its terminal local steps in one gate

### AC-4 — aiwfx-wrap-epic batches promote-done and cleanup in one gate

