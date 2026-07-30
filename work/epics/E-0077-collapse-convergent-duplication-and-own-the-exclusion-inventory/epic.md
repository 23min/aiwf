---
id: E-0077
title: Collapse convergent duplication and own the exclusion inventory
status: proposed
---
## Goal

Collapse the convergent duplication a structural sweep and the earlier
verb-layer-cleanup audit both surfaced, put the acknowledged-duplication
inventory back under an owner, and repair the instrument that measures both.

Addresses G-0472, G-0473 and G-0462, and clears the G-0447 remainder — G-0453,
G-0454, G-0455.

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

**Why the flaky-gate gap is in this epic.** Every measurement this work depends
on — which clone families `dupl` still flags, which exclusion entries are live,
whether a collapse removed a finding — is taken by running golangci-lint. G-0462's
second mechanism makes that instrument lie in a specific and relevant way: a
concurrent golangci-lint anywhere on the machine causes the guarded-rule harness
to report the rule as dormant, which is the exact defect the harness exists to
detect. This repo routinely carries several worktrees at once, so the condition is
ordinary rather than exotic. G-0462 is fixed before the inventory is measured, and
its first mechanism (`ETXTBSY` on exec of a just-written file) rides along because
the gap deliberately tracks the two together: what makes them expensive is the
shared symptom — a red gate that goes green on re-run — not the separate causes.

## Scope

**The instrument, first.** G-0462's two mechanisms: a bounded tolerance for
`ETXTBSY` where shared test helpers write and then exec a file, and a
golangci-lint child that runs against a cache it owns rather than the
machine-global default. In both cases the diagnostic has to stop misreporting the
cause — the contention message currently accuses the lint configuration of being
dormant.

**The clone families**, each parameterized by its differing name or key:
initrepo's hook installers — `ensurePostCommitHook` carries a second `regenStatus`
axis, and `ensurePreHook` differs behaviorally, returning an error where two
siblings derive a boolean that goes false on a read fault and overwrite the hook, so
collapsing them is not purely mechanical — the legacy-key stripper wrappers, `aiwfyaml`'s `replaceContracts` / `replaceHooks` and their unflagged
`append*` mirrors, and the contract verb scaffolds — whose structure is already
pinned by M-0280's verb-scaffold test, so that test's shape has to be considered
rather than only the clone.

**Then the exclusion list.** Measured, the four families above cover every one of
the six live entries; with G-0473's two stale entries dropped, the production
exclusion list can be deleted outright rather than re-owned. G-0473's option
3 is the durable half: a test asserting every `dupl` path exclusion still
corresponds to a live clone, so an exemption that outlives its duplication fails
rather than lingering. That is the shape G-0264 used for dormant `forbidigo`
config, generalized from "the rule is dormant" to "the exemption is dormant".

**Then the G-0447 remainder** — G-0453 (SHA-abbreviation helpers, needs a width
decision), G-0454 (three id-shape parsers), G-0455 (heading-walk state machines,
flagged evaluate-first). Each carries an explicit decide-before-extracting flag, so
none is purely mechanical.

## Out of scope

- **Cross-package sharing as a default.** D-0045 (accepted) deliberately duplicated
  a small git guard rather than importing across a layer boundary, on
  dependency-coupling grounds. Three of the four families are within one package, so
  parameterizing is straightforward. The legacy-key family is not: `initrepo` and
  `config` sit in different layering tiers, and its own gap is titled for spanning
  two layers. `initrepo` already imports `config` legally, so no new edge is
  required — but D-0045's reasoning is about coupling rather than direction, so the
  question is live for that family and has to be answered rather than assumed.
- **Test-file clones.** Most clone findings are in tests, fixtures or harness code.
  They are excluded by this repo's own `.golangci.yml` rule with its stated
  rationale, not by anything inherent to `dupl` — golangci-lint lints test files by
  default. Not this epic's concern either way.
- **Lowering the `dupl` threshold.** Retuning the gate is a separate decision from
  clearing what it already catches. Note that the `append*` mirror pair is unflagged
  because both its files are on the exclusion list, not because it falls below the
  threshold.
- **The third flaky-gate mechanism.** Timing-dependent stress-scenario oracles are
  the same class of symptom, tracked as G-0468 with an independent remedy. Not
  pulled in here.
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
- Each G-0447 remainder gap carries an explicit decide-before-extracting flag.
  None is a mechanical extraction, and none is scheduled as one.
- The inventory is measured only after G-0462's second mechanism is fixed. A
  measurement taken while another golangci-lint runs reports a rule as dormant
  when it is working fine.

## Success criteria

- [ ] The guarded-rule harness runs golangci-lint against a cache it owns, so a
      concurrent lint elsewhere on the machine cannot make it report a live rule
      as dormant; if contention still occurs, the message names contention rather
      than a dormant rule.
- [ ] The shared test helpers that write and then exec a file no longer fail the
      suite on `ETXTBSY`, and the tolerance is bounded so a genuine permissions
      failure still fails.
- [ ] Every clone family listed in *In scope* is either collapsed to one
      parameterized unit or carries a recorded reason for staying separate.
- [ ] The production `dupl` path-exclusion list is empty, or every remaining
      entry is asserted by a test to still correspond to a live clone.
- [ ] A `dupl` path exclusion that outlives the duplication it exempts fails a
      gate rather than lingering.
- [ ] Each gap in the G-0447 remainder is resolved by its own recorded decision —
      extracted, or explicitly left alone — not left as an open flag.
- [ ] G-0472, G-0473, G-0462, G-0453, G-0454 and G-0455 are promoted to
      `addressed`.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Whether the legacy-key family parameterizes across the `initrepo`/`config` layer boundary, given D-0045's coupling reasoning. `initrepo` already imports `config` legally, so no new edge is required, but coupling is the argument, not direction | yes | milestone-planning, before that family is touched |
| The width decision for the SHA-abbreviation helpers | yes, for G-0453 | flagged explicitly by G-0453; decided before extraction |
| Whether the heading-walk state machines should be unified at all — flagged evaluate-first, so "leave them" is a valid answer | yes, for G-0455 | evaluated before extraction, and the answer recorded either way |
| Whether collapsing the legacy-key family closes G-0477's dead guard or leaves it as separate work | no | decided when that family is collapsed |
| Whether the `ETXTBSY` fix is a bounded retry in the shared helpers or a fixture materialized once before parallel work starts | no | G-0462 leans retry for the general case and a shared fixture where one is natural; either satisfies the criterion |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The hook installers are collapsed mechanically, losing `ensurePreHook`'s error return and converting a read fault into a silent hook overwrite | high | the behavioral difference is named as a constraint; that family is not treated as a mechanical extraction |
| The exclusion inventory is measured while another golangci-lint runs, producing a wrong list of live clones and a wrong conclusion about what can be deleted | high | G-0462's second mechanism is fixed first; the measurement is not taken before it |
| Parameterizing the contract verb scaffolds invalidates M-0280's verb-scaffold test, and the test is adjusted to fit the new shape rather than re-derived | medium | the test's shape is part of the change, stated as a constraint |
| A decide-before-extracting gap is extracted anyway because the extraction is easy and the decision is not | medium | each remainder gap's success criterion is a recorded decision, and "explicitly left alone" satisfies it |
| The exclusion list is emptied and later regrows without an owner, since deleting it also deletes the place an owner would be named | medium | the dormant-exemption test is the durable half, and it is a success criterion in its own right rather than a side effect of the deletion |

## Milestones

Not yet allocated; ids are assigned when `aiwfx-plan-milestones` runs. Candidate
deliverables, in execution order:

- Repair the instrument — G-0462's two mechanisms, and the diagnostic that
  misreports contention as a dormant rule. First, because every measurement after
  it depends on it.
- Collapse the clone families, each answering its own named obstacle: the hook
  installers' behavioral difference, the legacy-key family's layer question, the
  contract scaffolds' pinned test.
- Delete or own the exclusion list, and land the test that fails when an
  exemption outlives its duplication.
- The G-0447 remainder, each gap decided before it is extracted.

## References

- G-0472 — convergent duplication surfaced by the structural sweep
- G-0473 — the `dupl` exclusion list is unowned; stale entries and the durable fix
- G-0462 — intermittent test failures: `ETXTBSY` on exec, golangci-lint cache contention (`high`)
- G-0453 / G-0454 / G-0455 — the G-0447 remainder, each decide-before-extracting
- G-0477 — the dead boundary guard under a comment that misdescribes it
- G-0468 — timing-dependent stress oracles; same symptom class, independent remedy
- G-0179 — per-worktree golangci-lint cache scoping, the fix `make lint` already applies
- G-0264 — the dormant-config test shape this epic generalizes to dormant exemptions
- D-0045 — deliberate duplication over a cross-layer import, on coupling grounds
- M-0280 — the verb-scaffold test pinning the contract verb scaffolds' structure
