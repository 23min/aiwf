---
id: E-0049
title: 'Ritual lifecycle model: gate discipline and commit/TDD model'
status: proposed
---
# Ritual lifecycle model: gate discipline & commit/TDD model

## Goal

The ritual lifecycle's commit/TDD model is coherent and matches CLAUDE.md:
milestone implementation commits plus TDD phase evidence are honest, and the
start/wrap rituals are internally consistent. The gate model itself is delivered
by the foundation epic E-0050, which this epic builds on.

## Context

The same 2026-06-28 audit found two lifecycle-behavior problems. (1) The wrap and
release rituals drifted from CLAUDE.md's gate discipline — ungated promote, merge,
and branch-delete steps; a bundled two-push release gate — while CLAUDE.md asserts
the wraps "keep per-action gates," which is false. (2) The milestone commit model
is incoherent: the implementation code is never staged (the wrap stages only the
spec), and `tdd: required` phase promotes are bursted at wrap, collapsing the
red-before-green timeline that is the only evidence the test came first. Filed as
gaps G-0295 (gate) and G-0293 (commit/TDD), with dependent ritual fixes and a
deferred config knob.

Problem (1) — the gate fix, G-0295 — was **extracted into foundation epic E-0050**,
which this epic depends on: the generalized declared-sequence gate and the
wrap/release fixes land there first, so every milestone wrap in this epic runs
under the corrected gate. E-0049 now covers the commit/TDD model (problem 2) and
the remaining start/wrap ritual fixes.

## Scope

### In scope

- **Depends on E-0050** (gate-discipline foundation). The generalized
  declared-sequence gate and the wrap/release ritual fixes land there; this epic
  inherits the corrected gate for its own milestone wraps.
- **Model 1 commit model:** commit implementation per AC on the milestone branch;
  phase promotes fire live during the cycle, never bursted at wrap (G-0293).
- start-milestone review framing + wrap-milestone trailer-step structural test
  (G-0271, G-0219).
- start-ritual fixes: stale `branch-not-found` code reference (G-0224) and
  sovereign-acts-land-off-trunk-on-trunk-based-repos ordering (G-0116, now
  unblocked — its blocker G-0059 is addressed).
- aiwf.yaml declared-sequence-wraps opt-in knob (G-0296) — deferred; drop if the
  Tier-1 rule (delivered by E-0050) suffices.
- `wf-patch` worktree-placement default (G-0349): cut the patch branch in an
  in-repo worktree by default (mirroring `aiwfx-start-epic`), removing the hazard
  where a concurrent `aiwf` verb commits onto an in-place patch branch. The
  shipped-skill edit follows the consumer-surface id and reference rules.

### Out of scope

- The gate model itself — generalizing the declared-sequence gate and fixing the
  wrap/release drift (G-0295) — now lives in foundation epic E-0050.
- Skill-body content correctness and drift chokepoints (sibling content epic
  E-0048).
- Epic-wrap lifecycle completion — scope-end-before-done, human-only-on-done, the
  wrap-closes-named-gaps sweep (G-0111). Its own decision-first future epic; kernel
  -verb-heavy and distinct from this epic's commit-model scope.
- The remaining tier-C gaps: patch-in-kernel decision (G-0060), test-parallelism
  ship-to-consumers (G-0104), Codex materializer target (G-0178) — separate
  standing/deferred work. G-0175 (ritual trailer-key) is closed as superseded by
  G-0190's ritualVerbs allowlist.

## Constraints

- The bright line E-0050 establishes (batch local, reversible mutations; exclude
  outward/irreversible actions and timing-bearing mutations) governs this epic too:
  in particular `tdd: required` phase promotes are timing-bearing and fire live,
  never batched (G-0293).
- E-0050 lands before this epic begins, so every milestone wrap here runs under the
  corrected declared-sequence gate; the sibling content epic E-0048's edit→test
  backstop lands afterward and guards everything later.
- Sovereign acts (`--force`) remain human-only; ADR-0010's branch model holds.

## Success criteria

- [ ] A `tdd: required` milestone's implementation lands as per-AC commits, and its
      phase-ladder timestamps show `red` before `green` (live promotes, not a wrap
      burst) (G-0293).
- [ ] The start-milestone review framing and the wrap-milestone trailer step are
      fixed and structurally tested (G-0271, G-0219).
- [ ] The start-ritual stale `branch-not-found` reference and the
      sovereign-acts-on-trunk ordering are corrected (G-0224, G-0116).

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| tier-C lifecycle disposition | resolved | G-0116 folded in; G-0111 is its own future epic; G-0175 closed (superseded) |

## Milestones

<!-- execution order; runs after foundation epic E-0050 -->

1. Model 1 commit + live phase promotes (G-0293).
2. start-milestone review framing + wrap-milestone trailer test (G-0271, G-0219).
3. start-ritual fixes: branch-not-found code (G-0224) + sovereign-acts-on-trunk ordering (G-0116).
4. aiwf.yaml declared-sequence-wraps opt-in knob (G-0296) — optional/deferred.
