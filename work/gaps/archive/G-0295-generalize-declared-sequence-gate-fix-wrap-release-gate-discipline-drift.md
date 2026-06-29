---
id: G-0295
title: Generalize declared-sequence gate; fix wrap/release gate-discipline drift
status: addressed
addressed_by_commit:
    - 7e6be1f1
---
## Problem

CLAUDE.md §"Gate discipline survives compaction" asserts that the single
declared-sequence gate exception is *"wf-patch only; milestone and epic wraps
**keep per-action gates** until the shape has proven itself there too."* The wrap
and release rituals do not actually have those per-action gates, so CLAUDE.md
describes a state of the world that is false:

- `aiwfx-wrap-milestone` runs `aiwf promote M-NNNN done` (step 5), the milestone
  -> epic merge commit, and the origin branch-delete (step 11) with **no gate**;
  the Constraints enumerate only the commit (step 8) and push (step 10) gates.
- `aiwfx-wrap-epic` runs `aiwf promote E-NN done` (step 8) ungated, and step 10
  allows *"batch approval for the full list"* of origin branch deletions.
- `aiwfx-release` step 6 bundles two origin pushes (`git push origin main` and
  `git push origin <tag>`) under a single approval.

A surfaced from an audit of the embedded skills against CLAUDE.md.

## Decision (Tier 1)

Generalize the wf-patch declared-sequence gate into a **general capability** for
any sequence of *local, reversible* mutations: one gate that **enumerates every
action verbatim**, binds approval to exactly that list (subset approval allowed),
and **aborts + re-gates on any deviation**. Add it as a standing rule in
CLAUDE.md's gate-discipline section and in `.claude/aiwf-guidance.md`; point the
wrap rituals at it; rewrite CLAUDE.md's "wf-patch only" scope sentence.

### Bright line (the load-bearing safety claim)

Batch local, reversible mutations that occur at a single moment anyway (the
terminal wrap sequence). **Exclude two classes:**

- **(a) outward / irreversible actions** — push to origin, PR-create, tag-push,
  remote-branch delete, `--force` — always stand alone, never batched. (`--force`
  is additionally sovereign / human-only.)
- **(b) mutations whose signal IS their timing** — `tdd: required` phase promotes
  fire live, never batched; collapsing them to one timestamp fabricates the
  appearance of TDD (see G-0293).

The rationale: the mechanical guarantees (pre-commit/pre-push hooks, `aiwf check`,
CI) already catch bad *end-states* regardless of prompt count; gates uniquely
protect the outward/irreversible actions the hooks cannot reverse. So batching
local mutations costs nothing mechanical, while outward actions keep standing
gates.

### Specific fixes

- `aiwfx-release` step 6: split the two origin pushes into two separate push
  gates (both outward).
- `aiwfx-wrap-milestone` / `aiwfx-wrap-epic`: replace the ungated promote / merge
  / branch-delete steps with a single declared-sequence gate over the terminal
  local sequence (promote-done + local merge + cleanup), push excluded.

## Scope

CLAUDE.md, `.claude/aiwf-guidance.md`, `aiwfx-wrap-milestone`, `aiwfx-wrap-epic`,
`aiwfx-release`. Tier 2 (an `aiwf.yaml` opt-in knob) is tracked as a separate
follow-up gap. Cross-references G-0293 (live phase promotes).
