---
id: G-0533
title: dupl is switched off across the test corpus, the larger and faster-growing half
status: open
priority: medium
---
## What's missing

`dupl` runs at threshold 100 over production code and is switched off across the
test corpus by three exclusions in `.golangci.yml` — `_test.go`, `testutil/`,
and `cellcoverage/`. The exclusion comment gives a real reason: table-driven
subtests are idiomatic here and produce near-identical bodies by design, which
is expected structure rather than the duplicated-logic class the linter was
turned on to find.

The reason is sound and the consequence is still that the larger part of the
corpus has no duplication detector. Measured with the exclusions lifted, the
tree carries 211 duplicate blocks at or above the threshold, 207 of them in test
files across 91 files. The test corpus is also the larger half of the codebase
by line count, and the half growing faster.

G-0473 covers a different catalogue — the eight production-file path exclusions,
two of which no longer correspond to any clone. It does not reach these three,
and [`../../docs/design/growth.md`](../../docs/design/growth.md) currently
implies that it does.

## Why it matters

This is the one duplication detector the repo already owns, and the surface it
cannot see is the one where duplication is least likely to be noticed by other
means: a near-copy of a test helper reads as ordinary boilerplate, and no
reviewer scanning a diff sees the other 90 files it resembles.

It is also the cheapest instrument on the list, in the sense that no new
apparatus has to be built. What blocks adoption is not the detector but the
backlog: 207 findings arriving at once is unadoptable as a gate, which is why
the exclusions exist and why they have outlived their reasoning.

## Resolution shape

Diff-scoping is the pattern that makes an existing backlog irrelevant, and this
repo already applies it three times — the comment-attrition scan, the coverage
audit, and the skill-edit backstop all compare against a base ref so untouched
code never blocks a change. `golangci-lint` supports the same shape natively via
`--new-from-rev`, so the work is configuration rather than a new engine.

Two knobs decide the signal-to-noise: the base ref, and a threshold for tests
set higher than production's. Idiomatic subtest bodies are short; genuine
copy-paste is long, and a separate threshold is what distinguishes them without
relitigating the exclusion's original argument.

Report before gating. The first run's output is the measurement that says
whether the higher threshold actually separates the two classes, and that
measurement does not exist yet.

## Sequencing

After G-0462. The epic's ordering rule is that the instrument is repaired before
it is trusted, and a detector newly pointed at 91 files is exactly the case
where a gate red for unrelated reasons would be read as noise from this change.
