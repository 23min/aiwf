---
id: G-0564
title: Blessed workflows are enumerated nowhere; choreography is single-host
status: open
priority: medium
---
## What's missing

G-0121 named four sub-gaps. Two of them — composition tests across verb chains,
and tree-level post-conditions under arbitrary legal composition — are being
mechanized. Two remainders are not, and they differ in kind from each other as
well as from the mechanized pair.

**No declarative enumeration of blessed workflows** (G-0121 sub-gap 1). There is
no artifact a contributor, human or LLM, can read to learn which sequences of
verbs are legal, with each step's pre- and post-conditions. Each skill body
describes one workflow in prose; nothing cross-links them or asserts they
exhaust the legal surface. E-0033 built the per-verb legality vocabulary in
`internal/workflows/spec` and explicitly scoped skill-level ritual choreography
— start-epic, wrap-milestone, and the rest — out of that work as advisory only.
The artifact G-0121 proposed, naming each workflow with its entry condition, its
sequenced verb calls, and the tree-level invariants it preserves, was never
written; only intermediate audit docs exist.

**Choreography is pinned for a single host** (G-0121 sub-gap 4 remainder).
E-0030 mechanized the current single-host model: the `--branch` flag,
sequencing, the isolation-escape finding, and the `aiwf-branch` trailer.
`docs/initiatives/agent-agnostic-execution-topology.md` (2026-06-30) still names
the choreography concern as open in the broader multi-host context — where work
happens, and which checkout or branch is authoritative.

The two are filed together because both are what is left of G-0121 once
composition is mechanized, not because they share a fix. Either can be split out
when it becomes actionable on its own.

## Why it matters

The kernel's load-bearing rule is that framework correctness must not depend on
the LLM's behavior. Both remainders sit on the wrong side of it, differently.

An unenumerated workflow set is enforced only by whoever reads the skills. A
contributor cannot learn the legal boundaries without reading every skill and
inferring where they end, and skills cannot be refactored without a manual
re-walk of each one, because no artifact states what a refactor must preserve.

Single-host choreography is a narrower exposure today: aiwf targets Claude Code
only, and CLAUDE.md's deliberately-excluded list names multi-host adapter
generation among the things not built. It becomes load-bearing when that
assumption changes, which is what the initiative tracks.

## Why neither is epic-ready

Neither remainder has a settled design question behind it, which is why the
composition epic excludes both rather than carrying them.

The enumeration needs a prior answer to what makes a workflow blessed. E-0033
examined this exact surface and chose advisory; reversing that is a decision,
not a task.

The multi-host remainder is gated on an initiative that is still capture-shaped.
A sibling initiative, `docs/initiatives/agent-host-artifact-adapters.md`, is a
separate concern by its own statement — which host-facing files aiwf generates,
not where work happens — so it does not close this either.

## Related

- G-0121 — the parent. Its composition sub-gaps are mechanized elsewhere; this
  gap carries what that work deliberately leaves behind.
- D-0063 — the accepted direction for the mechanized half.
- E-0033 — built the per-verb legality vocabulary and scoped ritual choreography
  out as advisory.
- E-0030 — mechanized single-host branch choreography.
- `docs/initiatives/agent-agnostic-execution-topology.md` — where the multi-host
  concern is tracked.
