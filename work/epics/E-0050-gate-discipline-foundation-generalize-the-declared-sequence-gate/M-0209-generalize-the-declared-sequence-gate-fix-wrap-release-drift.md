---
id: M-0209
title: Generalize the declared-sequence gate; fix wrap/release drift
status: draft
parent: E-0050
tdd: advisory
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
