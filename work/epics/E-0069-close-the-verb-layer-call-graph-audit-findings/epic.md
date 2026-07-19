---
id: E-0069
title: Close the verb-layer call-graph audit findings
status: proposed
---

# E-0069 — Close the verb-layer call-graph audit findings

## Goal

Close the verified findings from the verb-layer call-graph audit: fix the three
correctness bugs, collapse the hand-duplicated helpers onto the shared seams the
codebase already owns, extend `cliutil.FinishVerb`'s contract to cover its three
bypassers, and give the read-only verbs a neutral shared library — so a change to
a shared contract (commit-outcome envelope, git-plumbing helper, hook marker)
reaches every verb from one place instead of drifting per hand-rolled copy.

## Context

A full call-graph trace across every verb package
([`docs/initiatives/verb-layer-cleanup.md`](../../../docs/initiatives/verb-layer-cleanup.md),
each finding adversarially verified by an independent skeptic pass) found the
enforced mutating-verb spine intact but surfaced compounding local defects:
one verb hand-rolls id allocation and reopens a cross-branch collision exposure
(G-0426), one silently swallows git-read errors its siblings treat as fail-loud
(G-0427), a timestamp sort breaks across timezones (G-0428), the
fail/envelope/`withCommitSHA` triad is reimplemented in three CLI packages,
several verb pairs duplicate structurally identical logic, `doctor` hardcodes
marker strings `initrepo` exports for it, and the read-only verbs depend on each
other's CLI packages instead of a neutral library. The prevention mechanisms
shipped first (G-0422: the documented, enforced `projectionFindings` scope;
G-0423: the `dupl` clone-detection tripwire); this epic lands the fixes the
audit mapped. Builds on E-0052 (the allocator's cross-branch view) and the
M-0116 per-verb-package migration.

## Scope

### In scope

- The three bug fixes: route `import`'s auto-id path through
  `entity.AllocateID` (G-0426); make `show`'s history/scope-read failures fail
  loud matching `render`/`history` precedent (G-0427); normalize scope-event
  timestamps before sorting (G-0428).
- Mechanical housekeeping behind existing seams: one parameterized path-rewrite
  helper for `rename`/`reallocate`; `acknowledgeillegal` onto
  `gitops.IsAncestor`/`CommitExists`; one shared cascade guard for
  `Cancel`/`Promote` (and `Cancel` into its own file); `reflog_walk` onto
  `gitops.LocalBranchRefs`; porcelain-only annotations on the four unreferenced
  `gitops` functions; `doctor` onto `initrepo`'s exported marker helpers;
  `completeHookNames` deduplicated; the release-doc note that `aiwf upgrade`
  provides no automated rollback.
- Extending `cliutil.FinishVerb` with dry-run and multi-`Plan` support;
  migrating `archive`, `rewidth`, and `import` onto it; deleting all three
  duplicated envelope triads.
- Extracting the verified Cobra-free read-side helpers out of
  `show`/`history` into a neutral package consumed by `render`, `check`, and
  `status`.
- Recording the two judgment calls as decision entities during planning:
  contract-subsystem validation-gate convergence, and `rewidth`'s
  archive-sweep scope.

### Out of scope

- The verb-*surface* extension family — G-0168 (set-at-create field mutation
  verbs), G-0073 (cross-kind `depends_on`), G-0282 (inverse-coverage policy) —
  different concern, sequenced after.
- G-0276 (retiring git-stash verb isolation for index-only scoping) — same
  layer, its own risk profile.
- Build work for the contract-gate convergence and rewidth-sweep questions:
  this epic records the decisions; any resulting build is a follow-up.
- The audit's "future option" multi-agent sweep of the sink packages
  (`entity`, `tree`, `gitops`, `check`) — separate initiative, separate
  trigger.

## Constraints

- **The write-isolation DAG is untouchable.** Every change preserves the
  one-writer property enforced by `internal/policies/verbs_validate_then_write.go`.
- **Envelope behavior preservation.** The `FinishVerb` extension must not change
  any existing verb's success/error envelope bytes; the three migrated verbs'
  envelopes are pinned by tests before their triads are deleted.
- **The extraction is mechanical only.** The read-side library move is
  import-path changes on the verified Cobra-free surface — no algorithm
  changes, no API redesign — and is sequenced last, after the smaller fixes
  are green.
- **Bug fixes land test-first** (`tdd: required` on that milestone); each
  housekeeping item lands with a referencing test or rides an existing one.
- **The `dupl` tripwire stays green without new baseline entries** — deleting
  the triads must not be replaced by fresh clones.

## Success criteria

- [ ] `aiwf import` with `id: auto` allocates through `entity.AllocateID` and
  sees sibling local/remote-branch ids; G-0426 closed.
- [ ] `aiwf show` fails loud on history/scope read errors, matching its
  siblings; G-0427 closed.
- [ ] Scope events render in true chronological order across timezones in both
  `show` and `render`; G-0428 closed.
- [ ] The fail/envelope/`withCommitSHA` triad exists exactly once, in
  `cliutil`; `archive`, `rewidth`, and `import` route through `FinishVerb`.
- [ ] `rename`/`reallocate` share one path-rewrite helper; `Cancel`/`Promote`
  share one cascade guard; no mutating verb shells out to git directly.
- [ ] `doctor` detects markers via `initrepo`'s exported helpers; no hardcoded
  marker literals remain in `doctor`.
- [ ] `render`, `check`, and `status` consume read-side helpers from a neutral
  package, not from sibling `internal/cli` verb packages.
- [ ] Both judgment calls named in *In scope* are recorded as decision
  entities.
- [ ] Every gap listed in *References* as addressed is closed.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Contract-subsystem gates: converge on one scoped-projection-check concept, or accept the three styles as justified by blast radius? | no | Decision entity during milestone planning |
| `rewidth` archive-sweep scope: match `reallocate`'s archive-inclusive sweep, or keep active-tree-only? | no | Decision entity during milestone planning; a widening would be its own follow-up, not this epic |
| Name of the neutral read-side package | no | Decided at the extraction milestone; lean `internal/entityview` |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The extraction touches import graphs of `render`, `check`, and `status` at once | med | Sequenced last; mechanical-only constraint; the audit already verified the ~638-line Cobra-free surface |
| `FinishVerb` extension changes an existing verb's envelope | med | Envelope-pinning tests land before any dispatcher migrates |
| Housekeeping diff is wide (many files) | low | One acceptance criterion per finding; independently revertable commits |

## Milestones

- `M-0269` — bug fixes (`tdd: required`): import id allocation, show
  fail-loud, timezone sort, plus the id-allocation presence policy — closes
  the three gaps above · depends on: —
- `M-0270` — mechanical housekeeping: the shared-seam collapses listed in
  *In scope* · depends on: —
- `M-0271` — `FinishVerb` contract (`tdd: required`): dry-run +
  multi-`Plan`; migrate the three bypassers; delete the triads · depends
  on: —
- `M-0272` — read-side extraction into the neutral package · depends on:
  `M-0269`, `M-0270`, `M-0271`

## ADRs produced

- Candidates, decided at wrap per the ADR harvest: the `FinishVerb` outcome
  contract as the single envelope seam; the neutral read-side library as the
  read-verb dependency rule.

## References

- Source map: [`docs/initiatives/verb-layer-cleanup.md`](../../../docs/initiatives/verb-layer-cleanup.md)
  (findings F2–F14, verification pass, scoped cleanup targets).
- Gaps addressed: G-0426, G-0427, G-0428.
- Related, deferred: G-0168, G-0073, G-0282, G-0276.
- Prior prevention work: G-0422 (projectionFindings scope documented/enforced),
  G-0423 (dupl tripwire).
- Key source: `internal/verb/`, `internal/cli/cliutil/apply.go`,
  `internal/cli/show/`, `internal/cli/history/`, `internal/initrepo/`,
  `internal/policies/verbs_validate_then_write.go`.
