---
id: G-0473
title: The dupl exclusion catalogue is unowned and two entries are stale
status: open
---
## What's missing

`.golangci.yml` grandfathers eight production files from `dupl`, under a comment
calling each one real duplication and saying it is "tracked for follow-up triage
rather than fixed as part of this patch." Nothing tracks them.

The same comment says they were "Newly surfaced by enabling dupl (not part of the
original verb-layer-cleanup audit)" — so they fall outside the scope of G-0423,
the gap that turned the linter on. G-0423 pre-authorized exclusions as a class
("expect and document legitimate exceptions up front") and delegated its *own*
four instances to the verb-layer-cleanup initiative doc's scoped cleanup list.
That doc is now archived with `status: realized`. The eight were never in either
place: not in the audit, not in the initiative list, and not in any gap.

Two of the eight entries are also stale. A whole-tree run at threshold 100 with
the exclusions lifted reports no clone in `internal/check/acs.go` or
`internal/cli/check/check.go`. Both files still exist; the duplication that
earned them a place is gone.

The remaining six are live: `internal/aiwfyaml/aiwfyaml.go`,
`internal/aiwfyaml/hooks.go`, `internal/cli/contract/recipes.go`,
`internal/cli/contract/unbind.go`, `internal/config/config.go` and
`internal/initrepo/initrepo.go`.

## Why it matters

A stale entry is not tidy-up. The exclusions are path-scoped, and a clone whose
*both* ends sit inside an excluded file is silenced entirely — verified by
injecting a matching pair into `acs.go` and an identical pair into non-excluded
`acks.go`: under the repo's own config the `acs.go` findings vanish while the
`acks.go` ones fire. So each stale entry is an open blind spot, sized to a whole
file, protecting nothing. (A cross-file pair with one end outside still fires from
that end, so the blindness is total only for same-file clones.)

The untracked half matters differently. The comment tells a reader the debt is
tracked, and a reader who checks finds nothing tracking it. That is worse than an
exclusion honestly labelled as accepted, because it spends the reader's trust: the
next person who wonders whether these clones are known will conclude they are, and
stop looking.

The exclusion list is also the only inventory of acknowledged duplication in the
repo. Read as a catalogue rather than as build configuration, it answers "what did
we already decide to defer" — which it stops doing once its entries no longer
correspond to clones and no entity stands behind them.

## Options

1. **Drop the two stale entries and own the six that remain** — either under this
   gap or by pointing the config comment at the gap that collapses them. Smallest
   change; restores both the tripwire's coverage and the comment's honesty.
2. **Clear the list by collapsing the clones**, so the exclusions can be deleted
   rather than owned. The right end state, tracked as G-0472, but a larger body of
   work, and the exclusions stay wrong until it lands.
3. **Make staleness self-detecting** — a test asserting every `dupl` path
   exclusion still corresponds to a live clone, so an entry that outlives its
   duplication fails rather than lingering. This is the shape G-0264 used for
   dormant `forbidigo` config: a linter rule that detected nothing was a vacuous
   chokepoint, and the fix was a test that fails when the rule stops firing. It
   generalizes from "the rule is dormant" to "the exemption is dormant."

Option 1 should land regardless — it is small, and both other options leave the
comment's false claim standing unless it is fixed alongside. Option 3 is the more
durable companion and is what makes this class not recur; it is also the answer to
the question option 1 only settles once. Option 2 is the destination and should
not block either.

## Related

- G-0472 — the clone families these exclusions cover; collapsing them retires four
  of the eight entries
- G-0470 — the live sibling concern about `.golangci.yml` surface hygiene
- G-0423 — enabled `dupl` and pre-authorized the exclusion class
- G-0264 — the dormant-chokepoint pattern option 3 borrows

## Scope

Surfaced by a `wf-structural-sweep` pass after E-0073 wrapped, reading the
exclusion list as an inventory rather than as pass/fail.

Measurement note: golangci-lint truncates at `max-issues-per-linter` (default 50)
and `max-same-issues` (default 3). A run without both zeroed reports a fraction of
the clones and makes more entries look stale than are.
