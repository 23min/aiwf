# Epic wrap — E-0033

**Date:** 2026-05-22
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0033-pin-legal-kernel-verb-workflows-mechanically
**Merge commit:** 8247acc8

## Milestones delivered

- M-0120 — Ratify legal-workflow spec methodology in ADR (merged 9d14a30c)
- M-0121 — Pass A audit: catalog legal-workflow rules from existing surfaces (merged 1cb2a114)
- M-0122 — Pass B first-principles: derive legal-workflow rules from entity model (merged 61fbf1d9)
- M-0123 — Pass C reconcile to canonical Go spec table + drift policy (merged 4af9f893)
- M-0124 — Positive cell coverage: legal workflows succeed with expected post-state (merged d84d0fbf)
- M-0125 — Negative cell coverage: illegal workflows rejected with named errors (merged 7783ca12)
- M-0130 — Implement fsm-history-consistent check rule for FSM tree-invariant (merged 46ff86be)
- M-0131 — State-aware CancelTarget for Contract: cancel deprecated targets retired (merged 501943c0)
- M-0136 — aiwf acknowledge-illegal: retroactive force trailer for historical violations (merged ffda659a)
- M-0137 — fsm-history-consistent: batched git ops + silent-swallow fix (merged 6d1f97a8)

## Summary

E-0033 pinned the kernel's legal verb-workflow surface mechanically. Three passes (audit catalog → first-principles → reconcile-to-canonical) produced `internal/workflows/spec/rules.go`, a closed-set spec table of every (Kind, FromState, Verb) cell with an Outcome (Legal | Illegal), ExpectedErrorCode, and RejectionLayer axis. Positive (M-0124) and negative (M-0125) drivers exercise every cell in the table end-to-end against the real binary; AC-4 meta-tests on both sides enforce that the spec and the drivers stay synchronized, with an AST-based assertion-strength tooth (M-0125/AC-4) preventing silent-skip regressions on impl-gap cells. M-0130 added the `fsm-history-consistent` check rule that walks per-entity git history in DAG order and emits findings for FSM-illegal status transitions; M-0136 introduced `aiwf acknowledge-illegal` so kernel-repo legacy squash-merge commits can be deliberately ack'd without rewriting history; M-0131 fixed Contract's `CancelTarget` mapping (`deprecated → retired`, not `accepted → retired`); M-0137 closed the batched-git-ops silent-swallow path that M-0130 surfaced. The spec table now has both forward enforcement (verb-time + check-time) and historical-acknowledgment for legacy violations.

## ADRs ratified

Every ADR listed below was authored, ratified, and is reachable from milestones in this epic.

- ADR-0011 — Legal-workflow spec methodology

## Decisions captured

Every decision listed below was captured during the epic. Each is referenced by the spec rules in `internal/workflows/spec/rules.go` and (where the impl is in flight) by a tracking gap.

- D-0002 — Contract `accepted → rejected` is legal: operational kinds admit abrupt-stop
- D-0003 — Epic `cancel` refuses with listing when any milestone is non-terminal
- D-0004 — Milestone `cancel` refuses with listing when any AC is non-terminal
- D-0005 — AC mechanical-evidence: promote-time `--evidence` flag binds claim to test symbol
- D-0006 — Scope reachability is a three-edge tree, not the full reference graph
- D-0007 — `aiwf authorize` refuses non-{epic, milestone} scope-entities

## Follow-ups carried forward

Every gap listed below is open at wrap time and tracks impl-side or kernel-improvement work that E-0033's spec surface mandates but is deliberately out of milestone scope. Two clusters:

### Deferred kernel impl (matches `deferredImplErrorCodes` in `internal/policies/m0123_ac5_drift_test.go`)

These gaps each track one entry in the M-0123/AC-5 deferred-codes allowlist. Closing them removes the corresponding entry from the allowlist, re-binds the spec cell to a real `Code: "X"` impl literal, and graduates the cell to end-to-end driver coverage.

- G-0139 — Implement cancel refusal on non-terminal children/ACs per D-0003 and D-0004 (titles `epic-cancel-non-terminal-children`, `milestone-cancel-non-terminal-acs`)
- G-0140 — Implement `--evidence` flag on `aiwf promote AC met` per D-0005 (title `ac-evidence-missing`)
- G-0141 — `authorize-kind-not-allowed` per D-0007 — Phase 1 (verb-time refusal) landed in M-0125/AC-2 (`internal/verb/authorize.go`); Phase 2 (structured-code emission so AC-5 drift policy resolves the code) remains
- G-0142 — Structured `fsm-transition-illegal` error from `entity.ValidateTransition`
- G-0143 — Implement scope-tree three-edge reachability per D-0006

### M-0125 + wrap-session discoveries

- G-0144 — Rename `gap-resolved-has-resolver` to match Q8 `addressed_by` semantics
- G-0145 — Classifier for legality-pertinent finding codes (AC-5 impl-spec arm)
- G-0160 — Per-edge FSM coverage drift unpoliced (spec table vs `entity.transitions`)
- G-0161 — AntiRules negative coverage: assert kernel does not enforce each anti-rule
- G-0166 — `RejectionLayerCheckTime` cells rejected at verb-time by kernel (spec/impl axis mismatch on `gap-resolved-has-resolver` and `acs-tdd-audit` cells)
- G-0167 — `ids-unique/trunk-collision` false positive on retitle + body enrichment (fix in flight on branch `fix/trunk-collision-rename-threshold` — trailer-driven rename detection)
- G-0168 — Kernel lacks mutation verbs for set-at-create frontmatter fields (`tdd:`, `discovered_in:`, `relates_to:`, `linked_adrs:`)

## Handoff

**Ready for next epic:**

- The spec table in `internal/workflows/spec/rules.go` is the canonical "what the kernel commits to" surface. Future verb work consults it; future cells extend it.
- Positive + negative drivers, plus the AC-4 meta-tests, are the chokepoint. Adding an Illegal cell without a negative subtest fails CI; adding a Legal cell without a positive subtest fails CI.
- `fsm-history-consistent` check rule (M-0130) and `aiwf acknowledge-illegal` (M-0136) are the FSM-history side: legacy violations get explicit ack commits; new violations fire pre-push.
- ADR-0011 records the methodology; future spec-revision work should reference it.

**Deliberately left open:**

- The 5 M-0123-era deferred-impl gaps (G-0139, G-0140, G-0141 Phase 2, G-0142, G-0143). A focused follow-up epic that lands the structured-code emission pattern across all five would close G-0141's Phase 2 and the four sibling gaps in one sweep.
- G-0166's reclassification or kernel-softening of the two `RejectionLayerCheckTime` cells.
- G-0168's frontmatter-mutation verbs. Touches the `aiwf milestone tdd`, `aiwf gap discovered-in`, `aiwf decision relates-to`, `aiwf contract linked-adrs` verb shapes.
- G-0167's fix is committed on `fix/trunk-collision-rename-threshold` but not yet merged to main; the wrap push uses `--no-verify` as the documented workaround until that PR lands.

## Doc findings

**Scope:** scoped — docs touched by the epic since merge-base with `origin/main`. Subject files: `docs/pocv3/design/legal-workflows-audit.md`, `docs/adr/ADR-0011-legal-workflow-spec-methodology.md`, `internal/skills/embedded/aiwf-acknowledge-illegal/SKILL.md`, `internal/skills/embedded/aiwf-check/SKILL.md`.

**Findings:** none actionable.

- Broken code references: 0.
- Removed-feature docs: 0.
- Orphan files: 0.
- Documentation TODOs: 2 false-positive matches (both are prose mentions of "TODO" in the context of documenting kernel principles — `docs/pocv3/design/legal-workflows-audit.md:397` quotes CLAUDE.md's "no TODOs in shipped code" principle; `internal/skills/embedded/aiwf-check/SKILL.md:98` discusses how the kernel treats `<!-- TODO -->` HTML comments in entity bodies). Neither is an unaddressed TODO marker — both are content describing TODO discipline.

No findings — 4 docs checked.
