---
id: G-0314
title: 'Configurable TDD promote-gate cadence: streamlined default, strict opt-in'
status: open
---
## Problem

`tdd: required` milestones record each AC's phase progression
(red -> green -> done -> met) as `aiwf promote M-NNNN/AC-N --phase ...` /
`... met` commits. Under the current gate discipline (CLAUDE.md §"Gate
discipline survives compaction"), every mutating commit is a human gate and
phase promotes are flagged *timing-bearing* (can't be batched). The literal
reading is one HITL approval per phase per AC — up to four round-trips per AC,
before any implementation code is even committed. For a 5-AC milestone that is
~20 stop-and-ask gates for local, reversible, mechanically-grounded state
transitions.

## Why the per-phase gate is low-value

A gate lets the human redirect before something costly or irreversible. The
phase/met promotes are none of those:

- **Local + reversible** — frontmatter transitions on the AC's own lifecycle,
  undoable by another promote; nothing leaves the machine.
- **Mechanically grounded** — `green` / `met` are *defined by* a passing test.
  The test is the evidence (per "framework correctness must not depend on LLM
  behavior"); a human approving promote-to-green adds no correctness the test
  did not already provide.
- **The real control points are elsewhere** — the wrap review (re-verifies every
  AC + its test) and the push (outward, always gated).

## The orthogonality insight

Approval and timing are independent axes. The "never burst phase promotes" rule
protects *timing fidelity*; per-phase HITL *approval* is a separate concern. And
auto-flow actually *improves* timing fidelity: promoting `green` the instant the
test passes fires the commit at the real moment, whereas HITL-gating delays it
and creates the very batching pressure the rule warns against. The per-phase gate
is the worst of both — friction now, burst pressure later.

## Proposed fix

An `aiwf.yaml` knob — `tdd.promote_gate: streamlined | strict` (name TBD):

- **streamlined (default)** — the assistant flows the intra-cycle AC-state
  promotes (red / green / done / met) live, without a per-promote HITL gate; the
  wrap review and the push are the human control points.
- **strict (opt-in)** — every phase / met promote is an individual HITL gate, for
  projects that want tight control.

It is **advisory assistant-behavior config**, not a kernel mechanism: read by the
`wf-tdd-cycle` ritual and the guidance fragment (the same layer the gate
discipline itself lives in). The kernel just runs the verb.

### Guardrails (must survive streamlining)

- **Outward / irreversible actions never streamline** — push, merge-to-mainline,
  tag, `--force` stay individually gated regardless of the knob. The knob is
  narrowly about intra-cycle AC-state promotes.
- **The implementation-commit checkpoint stays** — wherever code lands, the wrap
  review plus push gate remain.
- **Visibility** — even un-gated, the assistant reports the met transitions; the
  human sees them at the wrap.

Pairs naturally with the foreseen per-AC test-existence check (CLAUDE.md §"AC
promotion requires mechanical evidence": "Discipline is the chokepoint until a
kernel finding-rule lands that polices test-existence per AC"). Once `met` is
mechanically guarded by "has a test," streamlined cadence is even safer.

## Sequencing

Belongs in E-0049 (ritual lifecycle: gate discipline and commit/TDD model), next
to M-0204 (commit implementation per AC; live phase promotes) — not E-0048.
Surfaced 2026-06-29 during M-0195's first TDD cycle, when the per-phase gate
count made the friction concrete. Interim: M-0195 and the rest of E-0048 adopt
the streamlined cadence by operator direction pending this knob.
