---
id: M-0159
title: Real-world hardening of branch-model chokepoint
status: draft
parent: E-0030
depends_on:
    - M-0102
    - M-0103
    - M-0104
    - M-0105
    - M-0106
    - M-0158
tdd: required
---
## Goal

Address the real-world failure modes the M-0158 honest-scope audit
surfaced. The E-0030 epic's existing milestones (M-0102 through
M-0106 and M-0158) build the branch-model chokepoint against the
synthetic-fixture test set; this milestone brings the chokepoint
from "watertight against fixtures" to "robust against real-world
multi-branch workflows."

## Context

The M-0158 wrap discussion (the user's third-pass review of the
spec-table milestone) surfaced that significant real-world failure
modes are not covered by the existing milestones. The user's
explicit guidance: *"This is a super important epic and we cannot
[afford] to have corner cases. ... Creating and editing entities
when we have multiple branches needs to be rigorously correct
because friction is super costly."*

The gaps catalog this milestone consumes:

- [G-0200](../../gaps/G-0200-preflight-main-only-carve-out-generalize-to-trunk-name-from-aiwf-yaml.md) — hardcoded `"main"` in the carve-out
- [G-0201](../../gaps/G-0201-authorize-preflight-carve-out-accepts-cross-rung-ritual-mismatches.md) — cross-rung carve-out mismatches
- [G-0202](../../gaps/G-0202-isolation-escape-cherry-pick-gather-side-implement-cli-detection.md) — cherry-pick gather-side not implemented
- [G-0203](../../gaps/G-0203-branchoracle-firstparentbranches-conflates-lookup-failed-with-no-branches.md) — oracle typed-error distinction + fail-shut sub-concern
- [G-0204](../../gaps/G-0204-branchoracle-silent-on-shallow-clones-ci-fetch-depth-1.md) — shallow-clone silent escape
- [G-0205](../../gaps/G-0205-branchoracle-silent-on-force-pushed-away-violating-commits.md) — force-push silent escape
- [G-0206](../../gaps/G-0206-branchoracle-false-positive-on-branch-renames-after-authorize.md) — branch-rename false positive
- [G-0207](../../gaps/G-0207-detached-head-handling-untested-in-preflight-and-oracle.md) — detached-HEAD untested
- [G-0208](../../gaps/G-0208-aiwf-force-amend-override-has-no-operator-ux-path.md) — `aiwf-force` amend has no UX
- [G-0209](../../gaps/G-0209-ritual-step-ordering-is-advisory-only-no-kernel-enforcement.md) — SKILL.md ritual ordering is advisory only
- [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md) — M-0158 over-specification refactor

These 11 gaps together are the work that, when complete, lets
E-0030 ship with a chokepoint that holds against real-world
git workflows.

## Pre-decided design

The milestone is gap-consuming: each AC is shaped around closing
one or more of G-0200 through G-0210 with mechanical evidence.
The exact ACs are drafted as part of the `aiwfx-start-milestone`
ritual when this milestone is started; the AC seed set is below.

## Out of scope

- New behavioral surface: M-0102 through M-0106 already deliver
  the verb-time and check-time chokepoints. This milestone
  hardens them; it does not add a new chokepoint.
- M-0158's spec-table refactor is in scope (G-0210); registering
  new cells is not (no new cells are needed once the
  documentation-only ones are removed).
- Master/dev/develop trunk-name configuration is in scope (G-0200);
  generalizing beyond named trunks (e.g., arbitrary "current ref
  is parent") is not.

## Dependencies

- **M-0102 through M-0106 and M-0158** — this milestone hardens
  the chokepoints those milestones deliver. All five are `done`
  (M-0158 wraps before this one starts per the sequential plan).

## Acceptance criteria

<!--
AC seed set (to be drafted with `aiwf add ac` at start-milestone time):

1. BranchOracle handles shallow clones without silent escape (closes G-0204)
2. BranchOracle handles force-pushed history without silent escape (closes G-0205)
3. BranchOracle handles branch renames (closes G-0206)
4. Detached-HEAD handling explicitly tested in preflight + oracle (closes G-0207)
5. Cherry-pick gather-side implemented end-to-end (closes G-0202)
6. aiwf-force amend has operator-discoverable UX path — verb or skill (closes G-0208)
7. M-0158 cell catalog refactored to mechanical-weight-only set (closes G-0210)
8. Hardcoded "main" generalized to aiwf.yaml.allocate.trunk short name (closes G-0200)
9. Cross-rung carve-out mismatches addressed via hierarchical predicate or documented exception (closes G-0201)
10. BranchOracle typed-error distinction (closes G-0203)
11. SKILL.md ritual ordering: either kernel enforcement OR documented limitation (closes G-0209)

These are the seed set; the start-milestone ritual will refine and
allocate them.
-->

