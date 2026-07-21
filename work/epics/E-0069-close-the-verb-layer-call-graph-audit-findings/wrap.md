# Epic wrap — E-0069

**Date:** 2026-07-21
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0069-close-the-verb-layer-call-graph-audit-findings
**Merge commit:** 7c1d4a44

## Milestones delivered

- M-0269 — Fix import id allocation, show error swallowing, and scope-event sort order (merged 6d809980)
- M-0270 — Collapse duplicated verb-layer helpers onto their shared seams (merged c674ef46)
- M-0271 — Extend FinishVerb with dry-run and multi-Plan; migrate its three bypassers (merged 4384176f)
- M-0272 — Extract the read-side helpers into a neutral entityview package (merged 7d5d3fdc)
- M-0273 — Converge contract-mutating verbs on one shared diff-based validation gate (merged f09d6fb0)

## Summary

Closes the verified findings from the verb-layer call-graph audit
(`docs/initiatives/verb-layer-cleanup.md`): three correctness bugs
fixed (cross-branch id-allocation exposure in `import`, a
fail-loud/fail-silent git-read inconsistency in `show`, a
timezone-fragile scope-event sort), the hand-duplicated verb-layer
helpers collapsed onto their shared seams, `cliutil.FinishVerb`'s
contract extended to cover its three bypassers (`archive`, `rewidth`,
`import`), the read-only verbs given a neutral `internal/entityview`
library free of CLI-package dependencies, and the contract-mutating
verbs (`bind`, `unbind`, `recipe install`, `recipe remove`) converged
onto one shared diff-based validation gate in place of three
divergent per-verb gate styles. Scope held to the audit's own
findings — every milestone in *Milestones delivered* traces to a
named finding or bug id from the audit doc.

## ADRs ratified

- none

## Decisions captured

- D-0041 — contract verbs converge on one shared diff-based validation gate
- D-0043 — track `aiwf upgrade`'s missing rollback as a gap, not doc prose
- D-0044 — add an `ErrInternal` marker to `FinishVerbOutcome`'s err contract
- D-0045 — `entityview` duplicates the empty-repo git guard rather than importing `cliutil`
- D-0046 — diff the shared contract gate by finding identity, not the full struct

## Follow-ups carried forward

- G-0421 — cross-branch `milestone show --area` should honor the parent epic's area
- G-0429 — collapse the duplicated history/scope read tail in `show.go`'s view builders
- G-0430 — `aiwf upgrade` has no automated rollback for a bad binary swap

## Doc findings

Scoped doc-lint sweep across the epic's full change-set (102 files, `main..epic` diff): no changes touched `docs/` or `CLAUDE.md`. Every backticked `aiwf <verb>` invocation across the touched `work/` entity files resolves against the current CLI surface, except `aiwf upgrade --rollback` — correctly hedged as a non-existent, open-question flag in both G-0430 and D-0043, not drift. No TODO/FIXME markers introduced. No findings.

## Handoff

All five milestones the audit's findings mapped to this epic are
done; `docs/initiatives/verb-layer-cleanup.md`'s finding list is
closed for the scope this epic took on. Three gaps carry forward
(above) as legitimate follow-ups the audit surfaced but this epic
deliberately didn't take on — none block closing E-0069. Nothing left
open in the epic's own scope.
