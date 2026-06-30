---
id: M-0217
title: Skip redundant pre-push golangci-lint via a last-green-lint marker
status: cancelled
parent: E-0053
tdd: required
---
## Goal

Stop the pre-push `golangci-lint` from re-running when the exact tree state
was already linted green moments earlier (gap `G-0318`).

Deliverable: a last-green-lint marker — record the linted HEAD SHA after a
successful lint, and have the `pre-push.local` gate skip the re-lint when the
recorded SHA equals current HEAD and the working tree is clean. Any commit or
working-tree change invalidates the marker, so an unverified state still pays
full lint. The guarantee (`G-0179`: long-lived branches accumulate lint debt
invisibly) is preserved; only the provably-redundant re-run is skipped.

## Notes

Open question to settle during the work: whether golangci-lint's own warm
cache already makes the re-run cheap enough that the marker is not worth the
moving part. Measure first. Acceptance criteria authored when the milestone
starts.
