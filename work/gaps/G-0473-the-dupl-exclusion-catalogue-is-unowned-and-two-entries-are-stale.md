---
id: G-0473
title: The dupl exclusion catalogue is unowned and two entries are stale
status: open
priority: medium
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
correspond to clones and no entity stands behind them. Two entries answer the
question wrongly and the rest answer it correctly, so emptying the list is
legitimate only if what the correct entries carried survives elsewhere. It does:
every live entry names a family G-0472 tracks, and the one clone that outlives a
threshold raise carries its own acknowledgement as a per-site `//nolint`
rationale, which has the advantage of sitting next to the code it excuses rather
than in a file nobody reads for this purpose.

## Options

1. **Drop the two stale entries and own the six that remain** — either under this
   gap or by pointing the config comment at the gap that collapses them. Smallest
   change; restores both the tripwire's coverage and the comment's honesty.
2. **Collapse the clones the collapse improves**, so those exclusions can be
   deleted rather than owned. G-0472 judges one of the four families worth
   collapsing, so this clears part of the list and names the rest as kept rather
   than deferred. A larger body of work, and the exclusions stay wrong until it
   lands.
3. **Make staleness self-detecting** — a test asserting every `dupl` path
   exclusion still corresponds to a live clone, so an entry that outlives its
   duplication fails rather than lingering. This is the shape G-0264 used for
   dormant `forbidigo` config: a linter rule that detected nothing was a vacuous
   chokepoint, and the fix was a test that fails when the rule stops firing. It
   generalizes from "the rule is dormant" to "the exemption is dormant."
4. **Raise the global `dupl` threshold until the tree is clean, and empty the
   path-exclusion catalogue.** Measured 2026-08-05: at two and a half times the
   configured threshold, with every path exclusion deleted and the surviving
   hook-installer clone exempted at its two declarations, the whole tree reports
   zero. All eight entries here go, both stale ones among them, along with the
   three test-corpus exclusions G-0533 covers.

   One clone survives the raise — the hook-installer pair — and it needs an
   exemption of some form. Take it as `//nolint:dupl` on the two function
   declarations rather than as a path rule: measured, the directive on both ends
   reports zero while annotating only one leaves the other firing, and it exempts
   the clone instead of blinding the detector across a 1700-line file, which is
   the file-scope cost described above. The repo already requires a one-line
   rationale on every `//nolint`, so the honest label this gap is about and the
   linter's own obligation are the same obligation, discharged at the site rather
   than in a catalogue.

   What that rationale must not say is "deferred, tracked by a gap". G-0472 leans
   toward leaving this family duplicated, which makes the exemption durable
   rather than pending — but G-0472 is `open`, and resting a permanent label on
   an open gap's lean asserts a settledness nothing holds, which is the shape of
   the defect this gap exists to describe. The exemption should cite a recorded
   decision that the four hook installers stay duplicated because the shared unit
   is worse than the duplication. Writing that decision is part of this option,
   not a follow-on.

   The raise's cost is real and prospective — clone detection between the old
   threshold and the new one is given up. Today that forgives nothing, since
   every production clone below the raise already sits in an excluded file, but
   it will miss clones in that band later. Two pairs it gives up are not covered
   by G-0472's families: `internal/cellcoverage/fixture.go` and
   `internal/cli/cliutil/testutil/capture.go`, both test-support packages in all
   but filename, and neither tracked anywhere today.

Option 1 should land regardless — it is small, and options 2 and 3 leave the
comment's false claim standing unless it is fixed alongside. Option 4 subsumes
option 1 for a comparably small edit, at the prospective cost named above, and
is the smallest change that also gives the test corpus a detector it has never
had. That detector arrives at a sensitivity which would have caught nothing in
G-0533's measured window, so the gain is a tripwire for the future rather than a
backlog it surfaces today.

Option 3's verdict depends on which end state lands, and it is not the same
verdict either way. It detects staleness, which is real, but not the regrowth it
is easy to mistake it for: an exclusion added tomorrow for a live clone passes by
construction, and fails only once someone later collapses that clone and leaves
the entry behind.

If the catalogue is kept — one path entry for the hook installers — option 3 is
cheap and does real work. It passes today, since the clone is live, and it fires
the day someone collapses that family after G-0557 and forgets the entry, which
is the likeliest future site of precisely this defect. Under option 4's
`//nolint` narrowing it turns vacuous instead: the exemption leaves the surface
option 3 scans, and a test over path exclusions has nothing to read when there
are none. That is a different verdict from "unnecessary", reached for a different
reason, and if the guard is still wanted it argues for scanning the `//nolint`
rationales rather than the config.

The G-0264 analogy carries less than it appears to under either end state,
because the two subject sets move in opposite directions: the guarded-rule set is
permanent and grows, while this catalogue is debt being driven toward zero.

Option 2 is worth doing for the one family G-0472 judges collapsible, and should
not block any of the above — but it is no longer this gap's destination. Under
option 4 the only acknowledged clone left is one the recorded decision keeps.

## Related

- G-0472 — the clone families these exclusions cover; collapsing all four would
  retire six of the eight entries, though it recommends collapsing only one
- G-0470 — the live sibling concern about `.golangci.yml` surface hygiene
- G-0423 — enabled `dupl` and pre-authorized the exclusion class
- G-0264 — the dormant-chokepoint pattern option 3 borrows

## Scope

Surfaced by a `wf-structural-sweep` pass after E-0073 wrapped, reading the
exclusion list as an inventory rather than as pass/fail.

Measurement note: golangci-lint truncates at `max-issues-per-linter` (default 50)
and `max-same-issues` (default 3). A run without both zeroed reports a fraction of
the clones and makes more entries look stale than are.
