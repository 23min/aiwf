---
id: E-0049
title: 'Ritual lifecycle model: gate discipline and commit/TDD model'
status: proposed
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

**Current state (2026-07-04).** Much of this epic's scope has already landed, some
out-of-band — the sections below are annotated accordingly:

- Problem (2) itself is **resolved**: `aiwfx-start-milestone` now commits per AC and
  fires phase promotes live (G-0293, addressed via a standalone `wf-patch`; pinned
  by `internal/policies/g0293_live_phase_promotes_test.go`). M-0204 records this; its
  milestone disposition (cancel-as-delivered vs. keep-as-record) is an open
  follow-up.
- The wrap-milestone trailer step is done and structurally tested (G-0219,
  addressed).
- The sovereign-acts-on-trunk ordering was fixed by M-0104 under E-0030 before this
  epic existed (G-0116, addressed).

What genuinely remains: the start-milestone review-framing forward reference
(G-0271 defect #1), the stale `branch-not-found` code name in two skills (G-0224),
the deferred config knob (G-0296), roadmap-regen zero-friction (G-0350), and the
`wf-patch` worktree-placement default (G-0349, no milestone yet).

## Scope

### In scope

- **Built on E-0050** (gate-discipline foundation, done). The generalized
  declared-sequence gate and the wrap/release ritual fixes landed there; this epic
  inherits the corrected gate for its own milestone wraps.
- **Model 1 commit model** (G-0293) — **delivered**: commit implementation per AC on
  the milestone branch; phase promotes fire live during the cycle, never bursted at
  wrap.
- start-milestone review framing (G-0271 defect #1, open) + wrap-milestone
  trailer-step structural test (G-0219, done).
- start-ritual fix (G-0224, open): replace the stale `branch-not-found` code name —
  dead at its emission site, `rung-pair-illegal` is what the verb emits today — in
  the two start skills. The sovereign-acts-on-trunk ordering half (G-0116) is already
  done via M-0104.
- aiwf.yaml declared-sequence-wraps knob (G-0296) — deferred; drop if the Tier-1 rule
  (delivered by E-0050) suffices. The default is on, so the knob is an opt-*out*, not
  opt-in.
- `wf-patch` worktree-placement default (G-0349): cut the patch branch in an in-repo
  worktree by default (mirroring `aiwfx-start-epic`), removing the hazard where a
  concurrent `aiwf` verb commits onto an in-place patch branch. The shipped-skill
  edit follows the consumer-surface id and reference rules. **No milestone yet** —
  needs one, or land it as a standalone `wf-patch` and drop from scope.
- Roadmap-regen zero-friction in the wrap rituals (G-0350) — M-0230.

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
- [ ] The start-milestone review framing forward-references the wrap's independent
      two-lens review, structurally tested (G-0271 defect #1).
- [ ] The start-ritual stale `branch-not-found` code name is corrected to
      `rung-pair-illegal`, structurally tested (G-0224).
- [x] The sovereign-acts-on-trunk ordering is corrected (G-0116, done via M-0104).

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| tier-C lifecycle disposition | resolved | G-0116 folded in (now addressed); G-0111 is its own future epic; G-0175 closed `wontfix` (superseded by G-0190) |
| G-0349 milestone home | open | give it a milestone under this epic, or land it as a standalone `wf-patch` and drop from scope |
| M-0204 disposition | open | cancel-as-delivered (mirroring M-0203) or keep as a draft record — its scope already landed via the G-0293 patch |

## Milestones

<!-- status of each milestone; E-0050 (foundation) has landed -->

- **M-0203** — generalize the declared-sequence gate — **cancelled** (extracted to
  E-0050).
- **M-0204** — Model 1 commit + live phase promotes (G-0293) — **delivered** via
  patch; milestone disposition open.
- **M-0205** — start-milestone review framing + wrap-milestone trailer test (G-0271,
  G-0219) — G-0219 done; **G-0271 defect #1 remains**.
- **M-0206** — start-ritual fixes — the G-0116 sovereign-acts-on-trunk half is
  already done via M-0104, leaving **only G-0224** (`branch-not-found` code name).
- **M-0207** — aiwf.yaml declared-sequence-wraps knob (G-0296) — optional/deferred;
  neutral framing (opt-out, not opt-in). Title still reads "opt-in" and needs an
  `aiwf retitle`.
- **M-0230** — roadmap-regen zero-friction in the wrap rituals (G-0350).
- **(unassigned)** — `wf-patch` worktree-placement default (G-0349): needs a
  milestone or a standalone `wf-patch`.
