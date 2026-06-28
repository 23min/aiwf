---
id: E-0049
title: 'Ritual lifecycle model: gate discipline and commit/TDD model'
status: proposed
---
# Ritual lifecycle model: gate discipline & commit/TDD model

## Goal

The ritual lifecycle's gate model and commit/TDD model are coherent and match
CLAUDE.md: approval gates batch local, reversible mutations safely (and never
outward or timing-bearing ones), and milestone implementation commits plus TDD
phase evidence are honest.

## Context

The same 2026-06-28 audit found two lifecycle-behavior problems. (1) The wrap and
release rituals drifted from CLAUDE.md's gate discipline — ungated promote, merge,
and branch-delete steps; a bundled two-push release gate — while CLAUDE.md asserts
the wraps "keep per-action gates," which is false. (2) The milestone commit model
is incoherent: the implementation code is never staged (the wrap stages only the
spec), and `tdd: required` phase promotes are bursted at wrap, collapsing the
red-before-green timeline that is the only evidence the test came first. Filed as
gaps G-0295 (gate), G-0293 (commit/TDD), with dependent ritual fixes and a deferred
config knob.

## Scope

### In scope

- **Generalize the declared-sequence gate** to any local-mutation sequence; fix the
  wrap/release drift; write the local-vs-outward bright line into CLAUDE.md
  (G-0295). **This is the program lead — it lands first, before the sibling content
  epic, so every subsequent milestone wrap runs under batched gates.**
- **Model 1 commit model:** commit implementation per AC on the milestone branch;
  phase promotes fire live during the cycle, never bursted at wrap (G-0293).
- start-milestone review framing + wrap-milestone trailer-step structural test
  (G-0271, G-0219).
- start-ritual fixes: stale `branch-not-found` code reference (G-0224) and
  sovereign-acts-land-off-trunk-on-trunk-based-repos ordering (G-0116, now
  unblocked — its blocker G-0059 is addressed).
- aiwf.yaml declared-sequence-wraps opt-in knob (G-0296) — deferred; drop if the
  Tier-1 rule suffices.

### Out of scope

- Skill-body content correctness and drift chokepoints (sibling content epic).
- Epic-wrap lifecycle completion — scope-end-before-done, human-only-on-done, the
  wrap-closes-named-gaps sweep (G-0111). Its own decision-first future epic; kernel
  -verb-heavy and distinct from this epic's gate/commit-model scope.
- The remaining tier-C gaps: patch-in-kernel decision (G-0060), test-parallelism
  ship-to-consumers (G-0104), Codex materializer target (G-0178) — separate
  standing/deferred work. G-0175 (ritual trailer-key) is closed as superseded by
  G-0190's ritualVerbs allowlist.

## Constraints

- The declared-sequence gate's bright line: batch local, reversible mutations that
  occur at a single moment; exclude (a) outward/irreversible actions — push,
  PR-create, tag-push, remote-delete, `--force` — and (b) timing-bearing mutations
  (`tdd: required` phase promotes fire live).
- G-0295 lands first (program lead) and ships its own structural tests; the
  sibling epic's edit→test backstop lands afterward and guards everything later.
- Sovereign acts (`--force`) remain human-only; ADR-0010's branch model holds.

## Success criteria

- [ ] No ungated mutating action remains in the wrap or release rituals; the
      declared-sequence gate is documented and exercised.
- [ ] A `tdd: required` milestone's implementation lands as per-AC commits, and its
      phase-ladder timestamps show `red` before `green` (live promotes, not a wrap
      burst).
- [ ] The gate-discipline drift the audit found is gone, verifiable against
      CLAUDE.md's gate-discipline section.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| tier-C lifecycle disposition | resolved | G-0116 folded in; G-0111 is its own future epic; G-0175 closed (superseded) |

## Milestones

<!-- execution order; G-0295 runs before the sibling content epic, the rest after -->

1. **[program lead]** Generalize declared-sequence gate + fix wrap/release drift (G-0295).
   <!-- the sibling content epic runs here -->
2. Model 1 commit + live phase promotes (G-0293).
3. start-milestone review framing + wrap-milestone trailer test (G-0271, G-0219).
4. start-ritual fixes: branch-not-found code (G-0224) + sovereign-acts-on-trunk ordering (G-0116).
5. aiwf.yaml declared-sequence-wraps opt-in knob (G-0296) — optional/deferred.
