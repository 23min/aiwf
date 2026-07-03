---
id: G-0314
title: 'Configurable TDD promote-gate cadence: streamlined default, strict opt-in'
status: addressed
addressed_by_commit:
    - 4e1c54e3
---
## Problem

`tdd: required` milestones record each AC's phase progression
(red -> green -> done -> met) as `aiwf promote M-NNNN/AC-N --phase ...` /
`... met` commits. Streamlined cadence — each phase promote fires live as its
own mechanically-evidenced commit, without a per-promote human approval ask —
is now the shipped default: the embedded guidance fragment
(`internal/skills/embedded-guidance/aiwf-guidance.md`) states it directly, so
every consumer gets the relief this gap originally argued for, not just this
repo. That closes the friction this gap documented: up to four HITL
round-trips per AC before any implementation code even lands.

What remains open is the opt-in direction: a team that wants tighter,
per-phase human control over `tdd: required` promotes has no supported way to
ask for it today short of hand-editing their own CLAUDE.md, which forks from
the shipped guidance.

## Why streamlined is the right default

- **Local + reversible** — frontmatter transitions on the AC's own lifecycle,
  undoable by another promote; nothing leaves the machine.
- **Mechanically grounded** — `green` / `met` are *defined by* a passing test.
  The test is the evidence (per "framework correctness must not depend on LLM
  behavior"); a human approving promote-to-green adds no correctness the test
  did not already provide.
- **The real control points are elsewhere** — the wrap review (re-verifies
  every AC + its test) and the push (outward, always gated).
- **Approval and timing are independent axes.** The "never batch timing-bearing
  mutations" rule protects *timing fidelity*; per-phase HITL *approval* was a
  separate concern this gap conflated. Streamlined cadence actually *improves*
  timing fidelity: promoting `green` the instant the test passes fires the
  commit at the real moment, whereas HITL-gating delays it and creates the very
  batching pressure the timing-bearing rule warns against.

## Remaining scope: a `strict` opt-in

An `aiwf.yaml` knob — `tdd.promote_gate: streamlined | strict` (name TBD):

- **streamlined (default, shipped)** — intra-cycle AC-state promotes
  (red/green/done/met) flow live without a per-promote gate; the wrap review
  and the push are the human control points.
- **strict (opt-in, not yet built)** — every phase/met promote is an
  individual HITL gate, for projects that want tight control.

Advisory assistant-behavior config only (not a kernel mechanism): read by the
`wf-tdd-cycle` ritual and the guidance fragment. The kernel just runs the verb.
Guardrails carried over unchanged: outward/irreversible actions never
streamline regardless of the knob; the wrap review and push gate remain; the
assistant still reports met transitions even when un-gated, so the human sees
them at the wrap.

**Deferred** until a concrete team asks for `strict`: building the config
surface (schema field, config-loader wiring, ritual conditional) for a value
nobody has requested yet is speculative ahead of real demand. Build it when
that demand shows up, not before.

## Sequencing

If/when built, still belongs in E-0049, next to M-0204 (commit implementation
per AC; live phase promotes) — not the now-closed E-0048. No longer blocking
or urgent: the default-cadence fix that motivated this gap has already
shipped independently of the opt-in knob.
