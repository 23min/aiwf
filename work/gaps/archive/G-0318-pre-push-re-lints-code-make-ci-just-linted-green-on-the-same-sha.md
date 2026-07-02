---
id: G-0318
title: Pre-push re-lints code make ci just linted green on the same SHA
status: wontfix
discovered_in: M-0196
---
## What's missing

The pre-push lint gate (`pre-push.local`, G-0179) runs the full `golangci-lint` on every Go-touching push. When the operator has *just* run `make ci` (or `make check-fast`) green on the **same** `HEAD` with a clean tree, that lint is redundant — the identical linter, identical SHA, ran moments earlier. During a milestone wrap the lint therefore runs at least twice (once in the local full gate, once in the push hook) for zero new information.

## Why it matters

It is one of the recurring-redundancy taxes that make the wrap+push sequence slow (surfaced during the M-0196 wrap, alongside a sibling gap for the policy-suite runtime). The lint is a real boundary guarantee (G-0179: long-lived branches accumulate lint debt invisibly), so it must not be *removed* — only skipped when provably already-satisfied.

## Proposed fix shape

A "last-green-lint marker": after a successful `golangci-lint` run (in `make check-fast` / `make ci` / the hook itself), record the linted `HEAD` SHA (e.g. an untracked `.git/aiwf-lint-green` file). The `pre-push.local` lint gate reads it and **skips** the re-lint when **(a)** the recorded SHA equals the current `HEAD` **and (b)** the working tree is clean (no uncommitted changes — the hook lints the working tree, per its own scope-approximation note). Any commit or working-tree change invalidates the marker, so the lint re-runs whenever the verified state is stale.

- Keeps the guarantee: an unverified or changed tree still pays the full lint.
- Removes the double-run at the push boundary right after a green local gate.
- KISS: a single SHA file + two conditions; no caching of lint *results*, just "was THIS exact state already linted green."

Open question: whether to key on `golangci-lint`'s own cache instead (it already caches per-file analysis), measuring whether a warm-cache re-run is already cheap enough that the marker isn't worth the moving part. Measure before building.

## Discovered in

M-0196 — the epic-branch push re-ran the full `golangci-lint` immediately after a green `make ci` on the same SHA, part of the wrap+push slowness the operator flagged.
