---
id: D-0012
title: Rename gap-resolved-has-resolver to gap-addressed-has-resolver
status: accepted
relates_to:
    - M-0142
    - G-0144
    - E-0036
---
## Context

The check finding code `gap-resolved-has-resolver` was named when the gap FSM used `resolved` as its addressed terminal. The current gap FSM (`entity.transitions[KindGap]`) has terminals `addressed` and `wontfix` — there is no `resolved` state. The rule fires when a gap reaches `addressed` with neither `addressed_by` (entity refs) nor `addressed_by_commit` (commit SHAs) set. A reader of the code, or of `aiwf check` output, has to mentally translate `resolved` to `addressed`. G-0144 (discovered in M-0123) records the drift.

## Resolution

Rename the finding code `gap-resolved-has-resolver` → `gap-addressed-has-resolver`, matching the FSM's `addressed` terminal. The rename is atomic (one commit) across the emission (`internal/check/check.go`), the hint table (`internal/check/hint.go`), the spec rule's `ExpectedErrorCode` (`internal/workflows/spec/rules.go`), the embedded skill docs, and every string-matching test, fixture, and golden under `internal/`. The internal rule identifier (`gapResolvedHasResolver`) and its unit test are renamed in the same pass for full vocabulary coherence, though they carry no dashes and are not operator-visible.

**Downstream-consumer caveat.** The finding code is the stable key in the `aiwf check --format=json` `findings[].code` surface. Renaming it is a breaking change for any downstream tool that pins the literal `gap-resolved-has-resolver` — such a consumer must refresh to `gap-addressed-has-resolver`. The verified surface is small: no `aiwf.yaml` config knob names a finding code (severity escalation takes a bool/int, never a code string), and no committed rendered artifact (`STATUS.md`, `ROADMAP.md`) embeds the literal — so a rename cannot silently break a consumer's config or checked-in views. The only persistent reference that breaks is a hand-written script that parses `aiwf check --format=json` and matches the literal. aiwf is pre-1.0 and no known external consumer pins this code; the break is recorded here and in CHANGELOG. We accept the break rather than carry a dual-emit alias (see Alternatives).

## Alternatives considered

1. **Keep the old name (rejected).** Zero churn, but perpetuates the reader-translation tax the rule's readability rests on, and leaves `aiwf check` output naming a non-existent FSM state.
2. **Dual-emit / alias both codes (rejected).** Emit both `gap-resolved-has-resolver` and `gap-addressed-has-resolver` for a deprecation window. Two codes for one condition is worse drift than the rename: it doubles the finding surface, complicates the hint table and the AC-5 spec↔impl scanner, and defers the break rather than taking it cleanly. The rename is already upgrade-gated per consumer (a consumer only sees it when it upgrades the `aiwf` binary), which is the deprecation window — the alias adds permanent kernel cost to re-buy a window the release cadence already provides. YAGNI.
3. **Rename to a different vocabulary (rejected).** e.g. `gap-addressed-no-resolver`. The existing `has-resolver` framing names the *required* postcondition, matching the sibling `adr-supersession-mutual` naming style (name the required invariant, not its negation); keep it.

## Consequences

- One commit renames the code across all non-archive `internal/` surfaces; the M-0142 absence chokepoint (`internal/policies/`) fails CI if the old literal `gap-resolved-has-resolver` reappears anywhere under `internal/`.
- Downstream `--format=json` consumers pinning the old code break and must refresh; recorded in CHANGELOG under the next release. For the dogfooding repos, a per-repo confirmation is one command — `grep -rn "gap-resolved-has-resolver" . --exclude-dir=.git` — run before upgrading that repo's `aiwf`; an empty result in tracked scripts/CI means the upgrade is transparent (gitignored skills regenerate via `aiwf update`).
- Historical records keep the old name: archived gaps/epics, the legal-workflows audit docs (point-in-time analysis, which also reference the long-gone `resolved-by:` field), and G-0166's quoted analysis are not rewritten (forget-by-default / decision-is-decision). Their old-name references remain accurate as of their authoring.
- The spec table's gap-FSM illegal cell continues to reference the code by its new `ExpectedErrorCode`, so the AC-5 spec→impl resolution and M-0140's legality-classification arm keep resolving it.
