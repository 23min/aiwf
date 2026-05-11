---
id: M-0097
title: 'Close M-0094/95/96 verification seams: M-0095 automation audit chokepoint and AC-5 drift comparator'
status: in_progress
parent: E-0028
tdd: required
acs:
    - id: AC-1
      title: CI/script audit chokepoint fires on non-forced epic-activation invocations
      status: met
      tdd_phase: done
    - id: AC-2
      title: Drift comparator helper has synthetic two-arm test (match + drift)
      status: open
      tdd_phase: done
---

# M-0097 — Close M-0094/95/96 verification seams

## Goal

Convert two verification seams left open by M-0094/M-0095/M-0096 into permanent mechanical chokepoints. The M-0095 sovereign-act rule deserves a CI test that fails if any CI workflow or script tries to run `aiwf promote E-... active` from a non-human context without `--force`. The AC-5 drift comparator deserves a synthetic two-arm test so the byte-compare logic is exercised regardless of whether the marketplace cache currently carries the rituals-repo content.

## Context

When wrapping M-0096, a confidence audit surfaced five seams across the start-epic ritual epic:

1. **M-0095's "no non-human `aiwf promote E- active` in CI/scripts" audit was claimed in the spec but not verifiably executed** — the spec's *Validation* section reads as if a grep was run; the conversation record does not show evidence of it. A durable test makes the claim load-bearing.
2. **M-0096/AC-5's drift comparator skips today.** The byte-compare arm (`string(cached) != fixture`) is reachable only in the rare drift-detected production state; it has never been exercised in any test fixture. If the comparator has a subtle bug — wrong direction, off-by-one in error message — nothing catches it.
3. Rituals-plugin manifest registration for `aiwfx-start-epic` is unverified (auto-discovery may or may not suffice).
4. `/aiwfx-start-epic` has never been executed against a real proposed epic.
5. Mutation testing has not been run against the M-0094 / M-0095 / M-0096 additions.

This milestone closes (1) and (2) with durable test chokepoints. Items (3)–(5) are operator tasks done during this milestone's self-review pass; findings either fix in-flight or open as gaps. The milestone deliberately does NOT inflate operator tasks into ACs whose only evidence is "I did it and wrote it down" — that would weaken the AC-evidence bar.

## Acceptance criteria

(ACs allocated at `aiwfx-start-milestone` time per the planner-skill convention.)

## Expected shape

- **AC-1 (M-0095 audit chokepoint)** — New test in `internal/policies/` (suggested name `aiwf_promote_epic_active_audit_test.go`) walks `.github/`, `Makefile`, and `scripts/` under repo root. Any line invoking `aiwf promote E-... active` (or equivalent) must be paired with either `--force` on the same line/heredoc, OR a documented human-actor wrapper in the surrounding context (e.g. an explicit `aiwf-actor: human/...` override, or a comment exempting the call). The test produces one finding per offending line with `path:line` and the canonical fix.
- **AC-2 (Drift comparator helper + two-arm test)** — Extract the byte-compare logic from `TestAiwfxStartEpic_AC5_DriftAgainstCache` into a typed helper (suggested signature `compareSkillToCache(fixture, cached []byte, skillPath string) error`). Returns nil on match; returns a typed error with the skill path and a fixed-up suggestion on drift. Unit-test the helper with synthetic match/drift inputs — both arms exercised regardless of cache state. The existing AC-5 test calls the helper after filesystem plumbing; the helper-level test does not depend on the cache.
- **Operator-task self-review** — Before declaring complete, run the rituals-plugin manifest verification, execute `/aiwfx-start-epic` end-to-end against a synthetic proposed epic, and run `mutate-hunt` against `internal/check/` and `internal/verb/` covering M-0094 / M-0095. Document findings under *Validation*. Real issues open as gaps (`aiwf add gap --discovered-in M-0097`); cosmetic issues fold into the in-flight milestone.

## Dependencies

- **M-0094** (done) — supplies the rule M-0097's AC-1 indirectly chokepoints around.
- **M-0095** (done) — supplies the runtime refusal AC-1's chokepoint complements.
- **M-0096** (done) — supplies the drift-check test whose comparator AC-2 hardens.

## References

- E-0028 epic spec — overall start-epic ritual scope.
- M-0094, M-0095, M-0096 — the three preceding milestones whose verification seams this milestone closes.
- G-0063 — gap framing the start-epic ritual.
- CLAUDE.md *AC promotion requires mechanical evidence* — the rule M-0097 operationalizes by converting a paper-trail audit into a durable test.
- CLAUDE.md §"Test untested code paths before declaring code paths done" — rationale for the AC-5 drift comparator's two-arm test.

### AC-1 — CI/script audit chokepoint fires on non-forced epic-activation invocations

### AC-2 — Drift comparator helper has synthetic two-arm test (match + drift)

