---
id: G-0328
title: Golden-fixture byte-identity comparator for aiwf check
status: open
discovered_in: M-0216
---
## Context

E-0053 / M-0216 refactored `aiwf check`'s history-walking rules to derive from
shared in-memory structures (the commit DAG, one HEAD walk, and `--raw` blob
ids) instead of per-rule git fan-out. The load-bearing correctness constraint
was that `aiwf check --format=json` findings stay **byte-identical** before and
after the refactor.

That property was verified two ways: (1) the standing per-rule fixture suites
(`TestBulkRevwalk_*`, `TestFSMHistoryConsistent_*`, the oracle / acks / cherry /
provenance suites), which fail on any behaviour change; and (2) a one-time
manual cross-binary diff on the live tree (old pre-AC-2 binary vs new — 31 = 31,
sorted-identical), repeated at each increment.

## The gap

There is no **standing** test that reproduces the byte-identity claim
mechanically. A true old-binary-vs-new-binary test is not feasible (you cannot
build pre-refactor code inside a unit test), and the milestone's validation
prose only records that the measurement happened — it is documentation, not a
test. Raised by the M-0216 third-pass review (Finding 3).

## Proposed resolution

Add a golden-fixture regression guard: a frozen synthetic fixture repo
(committed under `testdata/`) plus a golden snapshot of its
`aiwf check --format=json` output; the test asserts the current binary
reproduces the snapshot byte-for-byte. This does not retroactively prove the
refactor's fidelity (the golden is captured post-refactor) but it catches
**future** drift in the shared-context rules against a stable, reviewable
baseline — the standing guard the one-time manual diff cannot be.

The fixture must be synthetic and obviously fictional (per the repo's
golden-file convention), and the snapshot regenerated through a named `make`
target so an intended change is a reviewable golden update rather than a silent
edit.

## Acceptance sketch

- A `testdata/` fixture repo plus a golden `check --format=json` snapshot.
- A test that fails if the current binary's output diverges from the golden.
- A documented regeneration path for intended changes.
