---
id: M-0243
title: Named scenarios from G-0212 and G-0269
status: done
parent: E-0062
depends_on:
    - M-0240
tdd: required
acs:
    - id: AC-1
      title: A parallel-branch reallocate race is resolved per G-0212 item 1
      status: met
      tdd_phase: done
    - id: AC-2
      title: A concurrent cross-worktree edit-body race matches G-0212 item 2
      status: met
      tdd_phase: done
    - id: AC-3
      title: Archive-during-active-scope is exercised end-to-end per G-0212 item 3
      status: met
      tdd_phase: done
    - id: AC-4
      title: Force-push and cherry-pick vs acknowledge-illegal are exercised per G-0212
      status: met
      tdd_phase: done
    - id: AC-5
      title: G-0269's HEAD-drift race is scripted, expected-red until its guard lands
      status: met
      tdd_phase: done
---

## Goal

Turn G-0212's catalogued data-loss scenarios and G-0269's HEAD-drift
incident into concrete, executable harness scenarios — the specific,
already-named risk classes this epic exists to run against, not just the
general contention/fault-injection mechanisms M-0241 and M-0242 built.

## Context

G-0212 is a reasoning-and-history-evidenced catalog (26 `reallocate` commits
in this repo's own history motivated item 1; the G-0170 incident motivated
item 2) that named risk classes without a harness to execute them. G-0269 is
a live incident report. Both have sat as prose until this milestone.

## Acceptance criteria

### AC-1 — A parallel-branch reallocate race is resolved per G-0212 item 1

Two simulated operators allocate the same id on parallel branches; the
scenario drives the merge/push sequence to the point of collision and
asserts `aiwf check` reports it and `aiwf reallocate` resolves it cleanly,
per CLAUDE.md's own "Id-collision resolution at merge time" section.

### AC-2 — A concurrent cross-worktree edit-body race matches G-0212 item 2

Two `aiwf edit-body` runs on the same entity from different worktrees,
minutes apart in wall-clock terms but concurrent in git-history terms.
Asserts git's normal last-writer-wins semantics hold and that the outcome
is at least observable (not silently different from what a single operator
would expect) — G-0212 named the *lost-edit-with-no-audit-trail* risk
specifically; this scenario checks whether that's still true or has since
improved.

### AC-3 — Archive-during-active-scope is exercised end-to-end per G-0212 item 3

An entity is archived while a child's `authorize` scope is still active.
Asserts what a subsequent `aiwf authorize --pause` (or any scope-resolution
verb) against that child actually does — surfaces the real, current
behavior as a scenario result, since G-0212 posed this as an open question
rather than a known answer.

### AC-4 — Force-push and cherry-pick vs acknowledge-illegal are exercised per G-0212

Covers G-0212 items 5 and 6: a force-push that makes an
`acknowledge illegal`-referenced SHA unreachable, and a cherry-pick of a
force-amend override commit onto a different branch. Asserts whether the
exemption is silently revoked (item 5) or silently carried over without
audit trail (item 6), surfacing the real behavior either way.

### AC-5 — G-0269's HEAD-drift race is scripted, expected-red until its guard lands

Reproduces the actual incident: a parallel session's `git checkout` lands
between a verb's preflight (which reads `HEAD`) and its commit. This
scenario is expected to fail (i.e., confirm the wrong-branch commit still
happens) until G-0269's own mechanical guard ships — a known-red case tied
to that gap, not a defect in this milestone's work.

## Constraints

- Each scenario's assertion is about *current, real* behavior — for AC-2,
  AC-3, and AC-4 in particular, don't assume the "bad" outcome G-0212 posed
  as a risk; assert what the scenario actually observes, even if it turns
  out better than G-0212 feared.
- AC-5 is allowed to fail (expected-red) without failing this milestone —
  the milestone's own acceptance is that the scenario exists and correctly
  reports red, not that the underlying race is fixed.

## Design notes

- Each of AC-1 through AC-4 either confirms a G-0212 risk is real (→ this
  milestone opens a new, precisely-scoped gap for it, since G-0212 itself
  is a catalog, not a fix) or confirms it's already handled acceptably (→
  no new gap, and G-0212's own text for that item can be marked resolved
  when G-0212 itself is eventually closed).

## Surfaces touched

- `internal/stresstest/` (new scenario files)

## Out of scope

- Fixing anything AC-1 through AC-5 find broken — that's a new gap per
  finding, triaged per this epic's manual-triage constraint, not silently
  patched inside this milestone.
- G-0211's branch-choreography surface — already covered by M-0159/E-0030,
  confirmed stale and closed while scoping this epic.

## Dependencies

- M-0240 — the harness skeleton.

## References

- G-0212 — data-loss audit for verb composition across kernel surface
- G-0269 — mutating verbs lack a HEAD-drift guard against shared-worktree session races
- CLAUDE.md §"Id-collision resolution at merge time"

---

## Work log

### AC-1 — A parallel-branch reallocate race is resolved per G-0212 item 1

Confirmed: two independent clones of one bare origin deterministically
allocate the same id (`AllocateID` is a pure max+1 function); the
merge/push sequence surfaces the collision as `ids-unique`, `aiwf
reallocate` resolves it cleanly, and the final push succeeds · commit
d56bd28f · tests 9/9

### AC-2 — A concurrent cross-worktree edit-body race matches G-0212 item 2

Confirmed, empirically: merging two sibling worktrees' independent
`edit-body` edits to the same entity always produces a real git
conflict — never a silent last-writer-wins overwrite. Better than
G-0212 feared: maximally observable, not silent · commit 50aac81c ·
tests 11/11

### AC-3 — Archive-during-active-scope is exercised end-to-end per G-0212 item 3

Confirmed, empirically: G-0212's literal fear does not hold — `aiwf
show` and `aiwf authorize --pause` both correctly resolve a scope that
survives an archive sweep. A different, real gap surfaced instead:
`aiwf archive` can sweep a non-terminal milestone alongside its
terminal parent (promote-to-done carries no non-terminal-children
guard, unlike `aiwf cancel`), producing a tree `aiwf check` only flags
after the fact. Filed as G-0393 · commit 7e56e8cf · tests 12/12

### AC-4 — Force-push and cherry-pick vs acknowledge-illegal are exercised per G-0212

Item 5 confirmed real: a rebase dropping just the `acknowledge illegal`
commit (keeping the originally-flagged commit reachable) silently
revives the suppressed finding — the same reachability effect a
force-push produces. Filed as G-0395. Item 6 confirmed: a
force-override's trailers survive cherry-pick verbatim, so the new
branch's commit is trusted exactly as the original was — the current,
by-design trust model, not a narrower bug a check could catch without
breaking legitimate cherry-picks · commit e478902d, 3774e597 · tests
17/17

### AC-5 — G-0269's HEAD-drift race is scripted, expected-red until its guard lands

Reproduces the actual incident deterministically — no real concurrency
needed, since the defect is a plain time-of-check to time-of-use gap
between two sequential steps, not a timing race to win. The real run
confirms the incident still reproduces (1 violation), as expected per
this AC's own Constraints; a mutation probe caught and fixed a real
vacuity gap (a count-only assertion let a message-swapping mutant
survive) · commit ee034a7d · tests 21/21

## Decisions made during implementation

- (none)

## Validation

- `go build ./...` — clean.
- `go test -race -parallel 8 -count=1 ./...` (full repo) — green.
- `make lint` (full `golangci-lint` set) — 0 issues.
- `aiwf check` — clean (only the pre-existing, unrelated `provenance-untrailered-scope-undefined` warning about no upstream configured in this worktree).
- `make coverage-gate` (diff-scoped branch-coverage audit) — green after each AC's commit and again after the wrap-review fix commit.
- Independent two-lens review (fresh-context, dispatched before any AC's evidence was trusted at face value):
  - **Code-quality**: approve, zero blocking defects. Every claim verified by measurement — git plumbing hand-traced for 3 of 5 scenarios, the `classifyHeadDrift` mutation-probe claim independently re-confirmed, envelope field shapes checked against the real `check`/`show`/`gitops` source, all gates re-run and green. One non-blocking, cosmetic observation (count-only assertions on 4 of 5 classify tests) — applied anyway for consistency (see below).
  - **Design-quality**: one genuine, actionable finding — `currentBranch` and `headSHA`, each introduced in one AC's own scenario file, were both also called from AC-5's `head_drift.go`. This package's own precedent (`readGapFile`'s comment) sets the relocation bar at 2 consumer files, not 3; applied.
- Both review findings landed as one corrective commit (`refactor(stresstest): apply M-0243 wrap-review findings`), re-verified green (build, full stresstest suite, lint, coverage-gate) before wrap.

## Deferrals

- G-0393 — aiwf archive can sweep a non-terminal milestone alongside its terminal parent (discovered in AC-3)
- G-0395 — acknowledge illegal is revoked when the ack commit becomes unreachable (discovered in AC-4)

## Reviewer notes

- Every AC's finding is grounded in a real, hands-on empirical experiment against the actual compiled binary before any Go code was written — several initial hypotheses (e.g., AC-3's first read of "archive breaks scope resolution," later found to be a branch-reachability artifact of the experiment's own setup, not a real defect) were revised after a control experiment contradicted them. The milestone's own findings reflect the corrected understanding, not the first guess.
- AC-4 deliberately treats its two sub-findings asymmetrically: item 5 (ack revocation via rebase) is reported as a violation, since it's a genuine audit-trail regression with no alternative correct behavior; item 6 (cherry-picked force-override carryover) reports only premise breaks, never the carryover itself, since the observed behavior is the current, accepted, by-design trust model (aiwf-force + human actor is trusted wherever it appears) — a narrower mechanical check couldn't catch the carryover specifically without also breaking this repo's own legitimate milestone-to-epic merge cherry-picks.
- AC-5 is the one scenario in this milestone whose real run correctly ends "red" (1 violation) rather than "green" — per its own Constraints, that is the AC's designed acceptance criterion, not a defect in this milestone's work. It stays red until G-0269's own guard ships.
- Two new gaps were filed as a direct result of this milestone's scenarios (G-0393, G-0395); fixing either is explicitly out of scope here, per this epic's manual-triage constraint.
