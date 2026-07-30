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

One item is a bug rather than a smell, and it was found by asking why one job had
two shapes: `initrepo` re-implements `config`'s legacy-key detection predicate as a
prefix test instead of asking `config`, whose predicate is deliberately
top-level-only, and discards the `changed` return it already gets back. If the
predicates diverge, `initrepo` reports removing a field it did not remove.

Addresses G-0472 and G-0473, and clears the G-0447 remainder — G-0453, G-0454,
G-0455.

## Scope

**The detection-predicate defect first.** It is the only item that produces a wrong
message rather than a maintenance cost, it is a few lines, and it does not want to
wait behind a refactor.

**Then the clone families**, each parameterized by its differing name or key:
initrepo's hook installers (three collapse on the hook name; `ensurePostCommitHook`
carries a second `regenStatus` axis to absorb or document), the legacy-key stripper
wrappers, `aiwfyaml`'s `replaceContracts` / `replaceHooks` and their unflagged
`append*` mirrors, and the contract verb scaffolds — whose structure is already
pinned by M-0280's verb-scaffold test, so that test's shape has to be considered
rather than only the clone.

**Then the exclusion list.** Drop the two entries whose clones are gone, and own
the rest — collapsing the families above retires four of the eight. G-0473's option
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
  a small git guard rather than importing across a layer boundary. The families here
  are within-package or within-layer, which is why parameterizing is
  straightforward; any fix that reaches across a boundary distinguishes that
  decision rather than assuming it.
- **Test-file clones.** `dupl` excludes `_test.go` by design, and 210 of the 220
  clone findings are tests, fixtures or harness code. Not this epic's concern.
- **Lowering the `dupl` threshold.** Real clones sit below 100 tokens — the
  `append*` mirror pair among them — but retuning the gate is a separate decision
  from clearing what it already catches.
