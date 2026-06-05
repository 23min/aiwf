---
id: D-0023
title: M-0162/AC-3 cell-expansion deferred for reallocate_scenarios_test.go
status: proposed
---
## Question

The M-0162/AC-3 body's enumerated file-list (spec line 232) names
`branch_scenarios_*.go`, `isolation_escape_*.go`,
`detached_head_*.go`, `promote_wrong_branch_*.go`,
`authorize_scenarios_test.go`. The list does NOT include
`reallocate_scenarios_test.go` (M-0160/AC-1's 7-scenario surface
covering renumber + cross-branch collision + epic-atomic rename +
audit-trail invariants).

Should AC-3 cell expansion cover the 7 reallocate scenarios for
bijection completeness, or is the body's file-list scope binding?

## Decision

**Defer reallocate_scenarios_test.go from AC-3's cell-expansion
surface.** The body's enumerated file list is binding for AC-3
closure. The 7 reallocate scenarios remain unpinned at AC-3 wrap
and are tracked for a follow-up addition if AC-4's bijection
review surfaces the omission as material.

## Reasoning

- **Body scope is explicit.** Spec line 232 enumerates files
  individually; reallocate is not listed. The M-0161/AC-9 body's
  forecast counts (paraphrased "76 total") summed only the listed
  surfaces. Including reallocate would expand AC-3's scope beyond
  the body's text.

- **AC-4's bijection catches the gap if it matters.** When AC-4
  meta-tests land, invariant #2 ("every Pin references an
  existing cell") and #1 ("every cell has at least one Pin") run
  on whatever the test surface actually exercises. If a future
  contributor adds Pin calls to reallocate scenarios without
  adding cells, the bijection meta-test fires loudly. The deferral
  doesn't create silent debt — AC-4 is the structural backstop.

- **reallocate is branch-choreography-adjacent, not central.** The
  reallocate verb stamps `aiwf-prior-entity` trailers and rewrites
  cross-references atomically; the rule it pins is uniqueness +
  audit-trail-bridging, not branch-rung legality. The AC-3
  cell-expansion is about branch-choreography matrix density —
  trunk-name / rung-pair / oracle-state / shallow-clone /
  force-push / rename / detached-HEAD / promote-on-wrong-branch.
  Reallocate's matrix is orthogonal.

- **Bounded follow-up cost.** If reallocate inclusion becomes
  desirable (e.g., AC-4 reviewer flags the asymmetry), the work
  is mechanical: 7 stamps via scripts/m0162-stamp-cellid.sh + 7
  cell additions via scripts/m0162-build-ac3-cells.py. No new
  framework changes required.

## Status

Recorded at M-0162/AC-3 reviewer wrap (response to reviewer S6
finding: "out-of-scope decision without a tracked deferral").
Promotion: stays `proposed` until either (a) AC-4 reviewer
explicitly closes the question by accepting reallocate's omission,
or (b) a follow-up milestone wires the 7 cells, at which point
this decision is `superseded` by the resolving milestone's
implementing AC.

## References

- M-0162 body §"### AC-3 — M-0161 cell expansion: organic count
  via bijection invariants" (spec line 232 — file enumeration).
- `internal/cli/integration/reallocate_scenarios_test.go` — the
  7-scenario surface not covered.
- M-0162/AC-3 reviewer audit, S6 finding: "reallocate_scenarios
  out-of-scope without tracked deferral".
