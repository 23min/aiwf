---
id: E-0049
title: 'Ritual lifecycle model: gate discipline and commit/TDD model'
status: cancelled
---
# Ritual lifecycle model: gate discipline & commit/TDD model

## Goal

The ritual lifecycle's commit/TDD model is coherent and matches CLAUDE.md:
milestone implementation commits plus TDD phase evidence are honest, and the
start/wrap rituals are internally consistent. The gate model itself was delivered
by the foundation epic E-0050 (now done), which this epic builds on.

## Context

A 2026-06-28 audit found two lifecycle-behavior problems. (1) The wrap and release
rituals had drifted from CLAUDE.md's gate discipline — ungated promote, merge, and
branch-delete steps; a bundled two-push release gate — while CLAUDE.md asserted the
wraps "keep per-action gates," which was false. (2) The milestone commit model was
incoherent: the implementation code was never staged (the wrap staged only the
spec), and `tdd: required` phase promotes were bursted at wrap, collapsing the
red-before-green timeline that is the only evidence the test came first. Filed as
gaps G-0295 (gate) and G-0293 (commit/TDD), with dependent ritual fixes and a
deferred config knob.

Problem (1) — the gate fix, G-0295 — was **extracted into foundation epic E-0050**
(now done): the generalized declared-sequence gate and the wrap/release fixes
landed there, so every milestone wrap in this epic runs under the corrected gate.
E-0049 covers the commit/TDD model (problem 2) and the remaining start/wrap ritual
fixes.

**Current state (2026-07-04, settled).** Every piece of this epic's originally
planned scope is now terminal — landed, cancelled, or wontfixed — except one:

- Problem (2) itself **landed**: `aiwfx-start-milestone` commits per AC and fires
  phase promotes live (G-0293, addressed via a standalone `wf-patch`; pinned by
  `internal/policies/g0293_live_phase_promotes_test.go`). M-0204 was cancelled as
  delivered, mirroring M-0203.
- The wrap-milestone trailer step **landed**, structurally tested (G-0219,
  addressed). M-0205's other half, the start-milestone review-framing forward
  reference (G-0271), also **landed**: `aiwfx-start-milestone` step 7 was reframed
  from "self-review" to a "readiness check before handoff" that forward-references
  wrap's independent two-lens review — propagated to `aiwfx-wrap-milestone` and the
  `builder` agent card, which had repeated the same misframing. M-0205 was
  cancelled as delivered.
- The sovereign-acts-on-trunk ordering **landed** via M-0104 under E-0030 before
  this epic existed (G-0116, addressed). M-0206's other half, the stale
  `branch-not-found` code name (G-0224), also **landed** in the same patch as
  G-0271: both start rituals now name the live `rung-pair-illegal` /
  `branch-context-required` codes. M-0206 was cancelled as delivered.
  - G-0271 and G-0224 landed together as one `wf-patch` (implementation commit
    `f3c01667`, merged to mainline at `11860d38`), independently reviewed over two
    rounds (round 1 approved the core wording and flagged an incomplete
    propagation to sibling surfaces; round 2 confirmed the propagation complete).
- The deferred config knob **closed `wontfix`**: G-0296 was YAGNI — the knob
  defaults `true`, so the only config anyone would ever write is the opt-out, over
  advisory ritual prose with no kernel teeth, with no consumer demand. M-0207 was
  cancelled to match.

**What remains: roadmap-regen zero-friction (G-0350), tracked as M-0230** — the
epic's sole live milestone. E-0049 continues as a slim container for it rather
than closing outright, since the fix is topically at home under "ritual lifecycle
model" and needs a landing place.

## Scope

### In scope

- **Built on E-0050** (gate-discipline foundation, done). The generalized
  declared-sequence gate and the wrap/release ritual fixes landed there; this epic
  inherits the corrected gate for its own milestone wraps.
- **Model 1 commit model** (G-0293) — **delivered**: commit implementation per AC on
  the milestone branch; phase promotes fire live during the cycle, never bursted at
  wrap.
- start-milestone review framing (G-0271) + wrap-milestone trailer-step structural
  test (G-0219) — **both delivered**.
- start-ritual fix (G-0224) — **delivered**: the stale `branch-not-found` code name
  replaced with the live `rung-pair-illegal` in both start skills. The
  sovereign-acts-on-trunk ordering half (G-0116) was already delivered via M-0104.
- **Roadmap-regen zero-friction in the wrap rituals (G-0350) — M-0230, the epic's
  sole remaining live milestone.**

### Out of scope

- The gate model itself — generalizing the declared-sequence gate and fixing the
  wrap/release drift (G-0295) — lives in foundation epic E-0050 (done).
- Skill-body content correctness and drift chokepoints (sibling content epic E-0048,
  done).
- Epic-wrap lifecycle completion — scope-end-before-done, human-only-on-done, the
  wrap-closes-named-gaps sweep (G-0111). Its own decision-first future epic;
  kernel-verb-heavy and distinct from this epic's commit-model scope.
- The remaining tier-C gaps: patch-in-kernel decision (G-0060), test-parallelism
  ship-to-consumers (G-0104), Codex materializer target (G-0178) — separate
  standing/deferred work. G-0175 (ritual trailer-key) is closed `wontfix`, superseded
  by G-0190's ritualVerbs allowlist.
- `wf-patch` worktree-placement default (G-0349) — closed `wontfix`: a worktree in the
  lowest-ceremony ritual taxes the common solo/sequential path for a hazard that only
  bites concurrent multi-session use. Revisit with a targeted guard (warn when an
  `aiwf` mutation lands on a `patch/` branch) if it recurs.
- aiwf.yaml declared-sequence-wraps knob (G-0296) — closed `wontfix`: the knob
  defaults `true`, so the only config anyone would ever write is the opt-out, over
  advisory ritual prose with no kernel teeth, with no consumer demand today. M-0207
  cancelled to match. Reopen if real demand appears.

## Constraints

- The bright line E-0050 established (batch local, reversible mutations; exclude
  outward/irreversible actions and timing-bearing mutations) governs this epic too:
  in particular `tdd: required` phase promotes are timing-bearing and fire live,
  never batched (G-0293).
- E-0050 landed before this epic's remaining work, so every milestone wrap here runs
  under the corrected declared-sequence gate; the sibling content epic E-0048's
  edit→test backstop also landed and guards later skill edits.
- Sovereign acts (`--force`) remain human-only; ADR-0010's branch model holds.

## Success criteria

- [x] A `tdd: required` milestone's implementation lands as per-AC commits, and its
      phase-ladder timestamps show `red` before `green` (live promotes, not a wrap
      burst) — delivered via G-0293, pinned by
      `internal/policies/g0293_live_phase_promotes_test.go`.
- [x] The wrap-milestone trailer step is fixed and structurally tested (G-0219).
- [x] The start-milestone review framing forward-references the wrap's independent
      two-lens review, structurally tested (G-0271) — commit `f3c01667`.
- [x] The start-ritual stale `branch-not-found` code name is corrected to
      `rung-pair-illegal`, structurally tested (G-0224) — commit `f3c01667`.
- [x] The sovereign-acts-on-trunk ordering is corrected (G-0116, done via M-0104).
- [ ] Roadmap-regen zero-friction lands in the wrap rituals (G-0350, M-0230) — the
      epic's sole remaining open criterion.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| tier-C lifecycle disposition | resolved | G-0116 folded in (addressed); G-0111 is its own future epic; G-0175 closed `wontfix` (superseded by G-0190); G-0349 closed `wontfix` |
| M-0204 disposition | resolved | cancelled 2026-07-04 as delivered via the G-0293 patch, mirroring M-0203 |
| M-0207 / G-0296 disposition | resolved | G-0296 closed `wontfix` (YAGNI — no consumer demand for the knob); M-0207 cancelled to match |
| G-0271 / G-0224 delivery vehicle | resolved | both addressed via one `wf-patch` (commit `f3c01667`, merged `11860d38`), independently reviewed over two rounds; M-0205 and M-0206 cancelled as delivered |
| epic disposition | resolved | kept as a slim container for M-0230 (the sole remaining live milestone) rather than closed outright — G-0350 is topically at home here |

## Milestones

<!-- status of each milestone — settled 2026-07-04; only M-0230 remains live -->

- **M-0203** — generalize the declared-sequence gate — **cancelled** (extracted to
  E-0050).
- **M-0204** — Model 1 commit + live phase promotes (G-0293) — **cancelled**,
  delivered via the G-0293 `wf-patch`, mirroring M-0203.
- **M-0205** — start-milestone review framing + wrap-milestone trailer test (G-0271,
  G-0219) — **cancelled**, both delivered (G-0219 earlier; G-0271 via commit
  `f3c01667`).
- **M-0206** — start-ritual fixes (G-0224, G-0116) — **cancelled**, both delivered
  (G-0116 via M-0104; G-0224 via commit `f3c01667`).
- **M-0207** — aiwf.yaml declared-sequence-wraps knob (G-0296) — **cancelled**;
  G-0296 closed `wontfix` as YAGNI.
- **M-0230** — roadmap-regen zero-friction in the wrap rituals (G-0350) — **draft,
  the epic's sole remaining live milestone.**
