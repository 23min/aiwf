---
id: E-0077
title: Collapse convergent duplication and own the exclusion inventory
status: cancelled
---
## Goal

Collapse the convergent duplication a structural sweep and the earlier
verb-layer-cleanup audit both surfaced, and put the acknowledged-duplication
inventory back under an owner — deleted where the duplication it exempts is
gone, and mechanically policed where it is not.

Addresses G-0472, G-0473, G-0533 and G-0543.

## Context

Two things are tangled here. Several families of near-identical functions each do
one job while differing only by a name or a key — the cheap kind of duplication,
where a shared unit is not merely possible but is what the code is already shaped
like. And the `dupl` exclusion list that grandfathers some of them is unowned:
the comment says the debt is tracked, and nothing tracks it.

The predicate pair in that family deserves a note, because it is easy to read as
a bug and is not one. `initrepo` tests for a legacy key with its own column-0
prefix check rather than asking `config`, and discards the `changed` return it
already gets back. But `config`'s predicate is the same column-0 prefix test —
its extra "boundary" guard is dead code under a comment that misdescribes it
(G-0477) — so no divergence is constructible today. The duplication is real; the
wrong-message consequence is a latent risk that arrives only if one side gains a
distinction the other lacks.

**The exclusion list has two halves and they need different remedies.** Eight
entries name production files; three switch the detector off across the entire
test corpus. The production half is clearable, because the duplication behind it
is finite and collapsible. The test half is not: that backlog is large and mostly
idiomatic — sibling test functions varying by one fixture value, which is what
table-driven testing is supposed to look like — so it is cleared by scoping the
detector to changed code, not by editing every file that carries one.

### Measured 2026-08-05

The inventory, and the design questions it settles:

- **Six of the eight production entries still name a live clone; two do not.**
  `internal/check/acs.go` and `internal/cli/check/check.go` exempt code that is
  no longer duplicated at the configured threshold.
- **The four clone families cover all six live entries**, with nothing orphaned:
  the hook installers and the legacy-key wrappers account for `initrepo.go`'s two
  pairs, the legacy-key strippers for `config.go`, `replaceContracts` /
  `replaceHooks` for the `aiwfyaml` pair, and the contract verb scaffolds for
  `recipes.go` / `unbind.go`. Collapsing the families empties the production list
  rather than shortening it.
- **Diff-scoping is what makes the test corpus adoptable, and it is sufficient on
  its own.** With the three test exclusions lifted, the tree carries 219 duplicate
  blocks at threshold 100, 205 of them in test files across 98 files. Scoped to
  changed code instead, that same threshold yields 4 findings over `HEAD~200..HEAD`
  — a window touching 207 Go files — and 0 over the last 60 commits. The backlog
  that motivated the exclusions never enters the comparison.
- **Raising the threshold for tests does not separate the two classes, and costs
  the signal.** At threshold 200 the tree carries 10 blocks, 8 in tests, and every
  surviving test pair is sibling functions differing by one fixture value or one
  flag — the idiomatic class the exclusion was written to protect. The one genuine
  cross-file helper clone in the diff-scoped window, a near-copy shared between
  `noop_claim_scope_test.go` and `verb_write_guard_coverage_test.go`, dies with
  them. Block length does not distinguish idiomatic structure from copy-paste on
  this corpus. The threshold stays where it is.
- **A `dupl`-only pass costs seconds**, warm or cold, so a second invocation is
  not a reason to prefer a worse placement.

## Scope

**The clone families**, each parameterized by its differing name or key:
initrepo's hook installers — `ensurePostCommitHook` carries a second `regenStatus`
axis, and `ensurePreHook` differs behaviorally, returning an error where two
siblings derive a boolean that goes false on a read fault and overwrite the hook, so
collapsing them is not purely mechanical — the legacy-key stripper wrappers, `aiwfyaml`'s `replaceContracts` / `replaceHooks` and their unflagged
`append*` mirrors, and the contract verb scaffolds — whose structure is already
pinned by M-0280's verb-scaffold test, so that test's shape has to be considered
rather than only the clone.

**Then the production exclusion list**, which the collapse empties. With the two
stale entries dropped alongside, it is deleted outright rather than re-owned.
G-0473's option 3 is the durable half: a test asserting every remaining `dupl`
path exclusion still corresponds to a live clone, so an exemption that outlives
its duplication fails rather than lingering. That is the shape G-0264 used for
dormant `forbidigo` config, generalized from "the rule is dormant" to "the
exemption is dormant".

**Then the test corpus** (G-0533), where the same detector is switched off across
the larger and faster-growing half of the codebase. The three test exclusions are
replaced by a diff-scoped pass, so only duplication in code a change is already
touching is judged and the existing backlog stays invisible.

**Where each half fires.** The production half needs no new wiring: once the
exclusions are gone it is the ordinary `golangci-lint run` that `make lint` — and
so `make check-fast` — already performs, which is the earliest mechanical tier
this repo has for Go source. The diff-scoped test pass needs a base ref, so it is
a target of its own composed into `check-fast`, backstopped at pre-push and in
CI on the same base expression the coverage gate and the comment-attrition scan
already share. `make lint` stays whole-tree and branch-independent, so "is this
tree clean" remains answerable separately from "is my branch clean". Composing
the new target into `check-fast` puts it at the AC boundary mechanically — that
is what runs before each commit — rather than depending on a ritual being
invoked.

**Both halves block from the start**, rather than reporting first. The finding is
deterministic — two blocks either are near-copies or they are not — so the
failure mode a soak period guards against does not arise here: the worst case is
a correct finding someone disagrees with, not a gate that goes red for reasons
unrelated to the change under test. That case already has an escape hatch in the
exclusion list, which this epic makes self-policing, so an exemption added under
protest dies when the duplication it names does. A reporting phase would instead
produce output with no reader, and leave the gate's real adoption to a second
decision that has to be taken by someone who has stopped thinking about it.

**And the harness that measures all of it** (G-0543). This epic ships a
dormant-exemption test built on the same guarded-rule apparatus whose own claims
are not pinned by its own tests. Repairing that first is the ordering rule the
epic already applies elsewhere: the instrument is trusted only after it is
repaired.

## Out of scope

- **Cross-package sharing as a default.** D-0045 (accepted) deliberately duplicated
  a small git guard rather than importing across a layer boundary, on
  dependency-coupling grounds. Three of the four families are within one package, so
  parameterizing is straightforward. The legacy-key family is not: `initrepo` and
  `config` sit in different layering tiers, and its own gap is titled for spanning
  two layers. `initrepo` already imports `config` legally, so no new edge is
  required — but D-0045's reasoning is about coupling rather than direction, so the
  question is live for that family and has to be answered rather than assumed.
- **Clearing the existing test-corpus backlog.** Diff-scoping makes it irrelevant
  rather than resolving it; those blocks stay until a change touches them, the
  same bargain the coverage audit and the comment-attrition scan already strike.
- **Retuning the `dupl` threshold.** Measured above and settled: scoping, not
  length, is what the test corpus needed. Note that the `append*` mirror pair is
  unflagged because both its files are on the exclusion list, not because it falls
  below the threshold.
- **The G-0447 remainder.** G-0453, G-0454 and G-0455 are each flagged
  decide-before-extracting and none shares a subject with the exclusion inventory.
  They are cleanup of their own and do not belong to this epic's arc.
- **The remaining flaky-gate mechanism.** Timing-dependent stress-scenario oracles
  are the same class of symptom, tracked as G-0468 with an independent remedy.
- **The dead boundary guard itself.** `config`'s extra guard is dead code under a
  comment that misdescribes it (G-0477). Collapsing the legacy-key family may
  close it incidentally; whether it does is decided when that family is collapsed,
  and G-0477 stands on its own until then.

## Constraints

- D-0045 (accepted) is the standing reasoning against cross-package sharing by
  default. The legacy-key family spans two layering tiers, so its collapse
  answers D-0045's coupling argument rather than assuming direction settles it.
- `ensurePreHook` returns an error where two siblings derive a boolean that goes
  false on a read fault and overwrite the hook. A mechanical collapse of the hook
  installers would convert a read fault into a silent overwrite; the behavioral
  difference is the design input, not an implementation detail.
- M-0280's verb-scaffold test already pins the contract verb scaffolds' structure.
  Parameterizing them changes what that test is asserting, so the test's shape is
  part of the change.
- The diff-scoped pass uses the same base expression as `make coverage-gate` and
  the pre-push comment-attrition scan, so local and CI agree on what is in scope.
  A third base-ref convention would be a third place for that fact to drift.
- The detector is a ban, not a mandate: it costs once and obligates no artifact
  per new subject. A remedy that grew per-file or per-package annotations would
  fail the addition test this repo holds and is not the shape to reach for.

## Success criteria

- [ ] The guarded-rule harness's claims are pinned by its own tests, so a row
      that cannot fail is caught by the suite rather than by reading it.
- [ ] Every clone family listed in *Scope* is either collapsed to one
      parameterized unit or carries a recorded reason for staying separate.
- [ ] The production `dupl` path-exclusion list is empty.
- [ ] A `dupl` path exclusion that outlives the duplication it exempts fails a
      gate rather than lingering.
- [ ] `dupl` judges the test corpus, scoped to changed code, and the gate is
      reachable from the inner-loop target that runs before each commit as well
      as from pre-push and CI.
- [ ] A newly introduced clone — production or test — fails a gate before the
      change that carries it leaves the machine, demonstrated by a fixture rather
      than asserted.
- [ ] G-0472, G-0473, G-0533 and G-0543 are promoted to `addressed`.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Whether the legacy-key family parameterizes across the `initrepo`/`config` layer boundary, given D-0045's coupling reasoning. `initrepo` already imports `config` legally, so no new edge is required, but coupling is the argument, not direction | yes | milestone-planning, before that family is touched |
| Whether collapsing the legacy-key family closes G-0477's dead guard or leaves it as separate work | no | decided when that family is collapsed |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The hook installers are collapsed mechanically, losing `ensurePreHook`'s error return and converting a read fault into a silent hook overwrite | high | the behavioral difference is named as a constraint; that family is not treated as a mechanical extraction |
| The dormant-exemption test is built on the guarded-rule harness before that harness's own claims are pinned, so the new test inherits the defect it exists to prevent | high | G-0543 is resolved first; the epic's ordering rule is that the instrument is repaired before it is trusted |
| Parameterizing the contract verb scaffolds invalidates M-0280's verb-scaffold test, and the test is adjusted to fit the new shape rather than re-derived | medium | the test's shape is part of the change, stated as a constraint |
| The production exclusion list is emptied and later regrows without an owner, since deleting it also deletes the place an owner would be named | medium | the dormant-exemption test is the durable half, and it is a success criterion in its own right rather than a side effect of the deletion |
| The diff-scoped test pass fires on idiomatic sibling subtests often enough to be read as noise, and is muted wholesale rather than tuned | medium | the measured firing rate makes a wholesale mute a large response to a small signal; any exclusion it added is itself subject to the dormant-exemption test, and retreating to report-only is one flag rather than a redesign |

## Milestones

Not yet allocated; ids are assigned when `aiwfx-plan-milestones` runs. Candidate
deliverables, in execution order:

- Pin the guarded-rule harness's own claims. First, because the dormant-exemption
  test this epic ships is built on that apparatus.
- Collapse the clone families, each answering its own named obstacle: the hook
  installers' behavioral difference, the legacy-key family's layer question, the
  contract scaffolds' pinned test.
- Delete the production exclusion list, and land the test that fails when an
  exemption outlives its duplication.
- Point the detector at the test corpus, diff-scoped, wired at the inner-loop
  boundary and backstopped at pre-push and CI.

## References

- G-0472 — convergent duplication surfaced by the structural sweep
- G-0473 — the `dupl` exclusion list is unowned; stale entries and the durable fix
- G-0533 — the detector is switched off across the test corpus
- G-0543 — the guarded-rule harness makes claims its own tests do not pin
- G-0477 — the dead boundary guard under a comment that misdescribes it
- G-0468 — timing-dependent stress oracles; same symptom class, independent remedy
- G-0462 — the intermittent-gate mechanisms this work's measurements depended on, since resolved
- G-0264 — the dormant-config test shape this epic generalizes to dormant exemptions
- G-0453 / G-0454 / G-0455 — the G-0447 remainder, each decide-before-extracting, deliberately not in this epic
- D-0045 — deliberate duplication over a cross-layer import, on coupling grounds
- M-0280 — the verb-scaffold test pinning the contract verb scaffolds' structure
