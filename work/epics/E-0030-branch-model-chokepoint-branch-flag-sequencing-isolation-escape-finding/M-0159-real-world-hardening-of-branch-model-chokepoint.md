---
id: M-0159
title: Real-world hardening of branch-model chokepoint
status: in_progress
parent: E-0030
depends_on:
    - M-0102
    - M-0103
    - M-0104
    - M-0105
    - M-0106
    - M-0158
tdd: required
acs:
    - id: AC-1
      title: Combinatorial real-git E2E test framework under internal/cli/integration
      status: met
      tdd_phase: done
    - id: AC-2
      title: M-0106 isolation-escape paths covered by real-git E2E integration tests
      status: met
      tdd_phase: done
    - id: AC-3
      title: walkAcknowledgedSHAs lifted to shared helper, three rules consume it
      status: met
      tdd_phase: done
    - id: AC-4
      title: acknowledge-illegal silences isolation-escape and forced-untrailered
      status: met
      tdd_phase: done
    - id: AC-5
      title: trailer-verb-unknown rule consumes ackedSHAs via shared helper
      status: met
      tdd_phase: done
    - id: AC-6
      title: Cherry-pick gather-side detects cherry picked from commit markers
      status: met
      tdd_phase: done
    - id: AC-7
      title: Cellcoverage fixture aiwf-branch values resolve or are exempted
      status: open
      tdd_phase: red
    - id: AC-8
      title: branch-cell-override-f-nnnn-waiver Kind corrected to finding
      status: open
      tdd_phase: red
    - id: AC-9
      title: internal/check/hint.go names canonical override for isolation-escape
      status: open
      tdd_phase: red
---
## Goal

Land the **combinatorial real-git E2E test framework** for the branch-choreography surface, then use it to ship the override-convergence work (G-0208 + G-0214 / G-0196 + G-0202 + G-0213 sequencing constraint). After this milestone the kernel's branch-policing surface is system-pinned, not just unit-pinned; the override surface is symmetric across the three concrete consumers (`fsm-history-consistent`, `isolation-escape`, `trailer-verb-unknown`); and any future operator hitting one of these scenarios discovers a working override path through `aiwf --help` + tab-completion + skill.

This is **Tier 1 evidence-backed work** per the history-mining audit (June 2026):

- M-0106 itself shipped with the kernel finding effectively disabled for 4 cycles (F-1: CLI passed `nil` for the oracle). The seam-vs-layer gap is documented historical evidence.
- `aiwf-force` trailer has been used 12+ times in production (real overrides, not test fixtures). Operators hand-crafting trailers is documented friction; G-0208's UX gap is grounded.
- G-0214 / G-0196: one real consumer already caught by the `acknowledge-illegal` / `forced-untrailered` asymmetry.
- G-0213: the cellcoverage fictional-branch landmine is a sequencing constraint — must be addressed in the same commit set as any branch-resolution rule that M-0159 introduces.

## Context

The M-0158 honest-scope audit surfaced 11 real-world failure modes catalogued as G-0200 through G-0210. The user's directive during M-0159 planning ("third iteration"):

> *This is so critical to get correct ... I want all thinkable scenarios and realistic combinations to be tested, ie combinatorial. What if. It can happen. Verbs can be composed in any way for any reason ... in the worst case, data loss.*

A confidence-audit workflow (June 2026) then surfaced six test-integrity issues across M-0102..M-0106 + M-0158 (one tautological sabotage, one name/assertion contradiction, one SHA-distinctness fake, one acknowledged-tautological, one self-contradictory docstring, one cross-cell match-bleed). Those landed pre-M-0159 in commit `d43c1f27`.

A history-mining subagent investigation then reframed M-0159 priority from "imagination-driven completeness" to "evidence-driven sequencing":

- **Real evidence in history** (squash-merge override `f4ea7329`/`fdc539b8`, 12+ production `aiwf-force` uses, 26 reallocate commits, G-0167/G-0170 incidents) drives this milestone (M-0159) and M-0160.
- **No in-repo evidence** for G-0200/G-0201/G-0203/G-0204/G-0205/G-0206/G-0207/G-0209 — but per the user's "if we can imagine it, it will happen" principle, these are not dropped; they sequence into M-0161 (Tier 3 imagination-driven hardening) so other operators with different workflows still get coverage.

## Scope split (E-0030 hardening epic — three milestones)

This milestone is **M-0159 (Tier 1)**:

- The combinatorial real-git E2E test framework (closes G-0211).
- Override convergence: extend `acknowledge-illegal` to silence isolation-escape via shared helper-lift, with the same lift covering `forced-untrailered` asymmetry (closes G-0208 + G-0214 + G-0196).
- Cherry-pick gather-side CLI implementation (closes G-0202).
- `trailer-verb-unknown` wires to the shared ack-walk helper (third concrete consumer per the audit).
- Cellcoverage fixture branch-resolution fix (closes G-0213 — sequencing constraint).
- M-0158 spec-table Kind="finding" correctness fix (the rules.go uncommitted patch from M-0158).

**M-0160 (Tier 2 evidence-backed operational pain)** covers:

- Reallocate-stress combinatorial test (26 historical incidents).
- G-0167-class trunk-collision regression test (rename detection).
- G-0170-class apply-rollback data-preservation test.

**M-0161 (Tier 3 imagination-driven hardening)** covers:

- G-0200 (trunk config), G-0201 (cross-rung carve-out), G-0203 (BranchOracle typed errors), G-0204 (shallow clones), G-0205 (force-push), G-0206 (branch rename), G-0207 (detached HEAD), G-0209 (ritual step ordering), G-0210 (M-0158 cell catalog full refactor).
- All require combinatorial E2E coverage via the M-0159 framework.
- "If we can imagine it, it will happen" — different operators have different risk tolerances; coverage is mandatory even without in-repo evidence.

## Pre-decided design

**Test discipline (load-bearing).** Every M-0159 AC requires at least one real-git integration test under `internal/cli/integration/`. The test builds aiwf via `buildAiwfBinary`, sets up a real git repo via `tempRepo`, runs verbs as subprocess invocations, and asserts stdout/stderr/exit-code/trailers/envelope output. Rule-level unit tests stay as cheap regression catches but are NEVER substitutes. **No stubs anywhere.** A test body that doesn't exercise its named claim either gets reframed to match what it actually pins, or rewritten to pin the claim — never deleted, never left as a placeholder.

**G-0208 architecture (Path B with modifications, per confidence-audit workflow).** Lift `walkAcknowledgedSHAs` from `fsm_history_consistent.go` into a shared helper at `internal/check/acks.go` (or equivalent). Three concrete consumers: `fsm-history-consistent` (existing), `isolation-escape` (new), `trailer-verb-unknown` (the third user, currently named-but-not-wired in `trailer_verb_unknown.go:25-29`). The CLI gather layer computes the acked-SHA set once and passes it to all three rules. No `--code` flag, no new `aiwf-force-for-code` trailer, no Cobra rename — the rule does the per-rule SHA matching.

**G-0213 cellcoverage fix (sequencing-load-bearing).** Before landing any rule that reads `aiwf-branch:` against a "must resolve" check, the cellcoverage fixture's fictional branch value must be addressed. Per G-0213, three options: create the branch in the fixture setup, sentinel-trailer the fixture for rule exemption, or have the rule fail-open on empty BranchOracle. Decision lands in M-0159 itself (within the rule-adoption AC).

## Out of scope

- M-0160 and M-0161 work (their gaps remain in their respective milestones).
- New verb addition for G-0208 — Path B keeps the surface to one verb (`acknowledge-illegal`); no new verb shipped this milestone.
- Branch-resolution rule (e.g., "aiwf-branch must point to a real ref") — that's M-0161 work, deliberately gated behind the cellcoverage landmine fix landing first.
- Generalized retroactive override for arbitrary kernel codes — only the three concrete consumers above. The architectural primitive (shared walk + per-rule recognition) supports future expansion but no speculative scaffolding lands now.

## Dependencies

- **M-0102 through M-0106 + M-0158** — all `done`. M-0159 hardens what they delivered.
- **Commit `d43c1f27`** — pre-M-0159 patch round (6 test-integrity fixes) landed before this milestone starts.
- **G-0211, G-0213, G-0214 + existing G-0196, G-0202, G-0208** — gaps consumed by this milestone.

## Acceptance criteria

<!--
AC seed set (to be allocated via `aiwf add ac` at start-milestone time, after the AC framing is confirmed with the user):

1. Combinatorial real-git E2E test framework under internal/cli/integration/branch_scenarios_test.go: scenario-table driver, tempRepo helpers (shallow, rename, force-push, detached-HEAD, cherry-pick, amend, merge setups), envelope assertions. (G-0211)

2. M-0106 paths covered by real-git E2E: every existing M-0106 unit-tested scenario gets a parallel integration test that builds the binary, drives subprocess verbs, asserts envelope output. Closes the "shipped disabled" class.

3. walkAcknowledgedSHAs lifted to internal/check/acks.go; consumed by fsm-history-consistent, isolation-escape, and trailer-verb-unknown rules through a single ackedSHAs map[string]bool parameter populated by the CLI gather layer.

4. acknowledge-illegal extended to cover isolation-escape AND forced-untrailered subcodes via the shared helper. Real-git E2E: AI escape → aiwf acknowledge-illegal <sha> --reason → aiwf check silent; AI authorship preserved on original commit. (G-0208 + G-0214 + G-0196)

5. trailer-verb-unknown wired to consume ackedSHAs through the lifted helper. Real-git E2E: historical stray commit acked → check silent. Converts the docstring promise at trailer_verb_unknown.go:25-29 into mechanical truth.

6. Cherry-pick gather-side implemented in the CLI: real (cherry picked from commit <sha>) markers in commit bodies populate the cherryPicked map. Real-git E2E: git cherry-pick -x of an isolation-escape commit → check silent. (G-0202)

7. Cellcoverage fixture branch-resolution decision landed in the same commit set as any new branch-reading rule. (G-0213)

8. M-0158 spec-table Kind="finding" correctness fix (the uncommitted rules.go change addressing the M-0158 wrap miss).

9. internal/check/hint.go updated to name aiwf acknowledge-illegal as the canonical override invocation for isolation-escape findings; substring-tested at integration level.

These 9 are the seed set; aiwfx-start-milestone refines and allocates them.
-->

### AC-1 — Combinatorial real-git E2E test framework under internal/cli/integration

### AC-2 — M-0106 isolation-escape paths covered by real-git E2E integration tests

### AC-3 — walkAcknowledgedSHAs lifted to shared helper, three rules consume it

### AC-4 — acknowledge-illegal silences isolation-escape and forced-untrailered

### AC-5 — trailer-verb-unknown rule consumes ackedSHAs via shared helper

### AC-6 — Cherry-pick gather-side detects cherry picked from commit markers

### AC-7 — Cellcoverage fixture aiwf-branch values resolve or are exempted

### AC-8 — branch-cell-override-f-nnnn-waiver Kind corrected to finding

### AC-9 — internal/check/hint.go names canonical override for isolation-escape

