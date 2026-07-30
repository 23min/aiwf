---
id: E-0077
title: Collapse convergent duplication and own the exclusion inventory
status: proposed
---
## Goal

Collapse the convergent duplication a structural sweep and the earlier
verb-layer-cleanup audit both surfaced, and put the acknowledged-duplication
inventory back under an owner.

Two things are tangled here. Several families of near-identical functions each do
one job while differing only by a name or a key — the cheap kind of duplication,
where a shared unit is not merely possible but is what the code is already shaped
like. And the `dupl` exclusion list that grandfathers some of them is unowned: the
comment says the debt is tracked, and nothing tracks it.

The predicate pair in that family deserves a note, because it is easy to read as a
bug and is not one. `initrepo` tests for a legacy key with its own column-0 prefix
check rather than asking `config`, and discards the `changed` return it already gets
back. But `config`'s predicate is the same column-0 prefix test — its extra
"boundary" guard is dead code under a comment that misdescribes it (G-0477) — so no
divergence is constructible today. The duplication is real; the wrong-message
consequence is a latent risk that arrives only if one side gains a distinction the
other lacks.

Addresses G-0472 and G-0473, and clears the G-0447 remainder — G-0453, G-0454,
G-0455.

## Scope

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
