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
corpus has no duplication detector. With the exclusions lifted the tree carries
over two hundred duplicate blocks at or above the threshold, nearly all of them
in test files; the counts are in the dated measurement below. The test corpus is
also the larger half of the codebase by line count, and the half growing faster.

G-0473 covers a different catalogue — the eight production-file path exclusions,
two of which no longer correspond to any clone. Its option 4 and the threshold
resolution below are the same edit, reached from either side.

## Why it matters

This is the one duplication detector the repo already owns, and the surface it
cannot see is the one where duplication is least likely to be noticed by other
means: a near-copy of a test helper reads as ordinary boilerplate, and no
reviewer scanning a diff sees the other 90 files it resembles.

It is also the cheapest instrument on the list, in the sense that no new
apparatus has to be built. What blocks adoption is not the detector but the
backlog: that many findings arriving at once is unadoptable as a gate, which is
why the exclusions exist and why they have outlived their reasoning.

## Measured 2026-08-05

`dupl` emits both ends of every clone, so its finding count is twice its pair
count — worth holding onto, because dropping it doubles every number below.

- **Lifting the three exclusions at the configured threshold** puts 209 blocks on
  the tree, 205 of them in test files across 90 test files. (Lifting all eleven
  exclusions, production entries included, gives 219 across 98 files.) That is
  the backlog the exclusions hide, and it is not clearable by hand.
- **Diff-scoped at the same threshold**, over a 200-commit window touching 207 Go
  files, the detector fires four blocks — two pairs. One is a same-file pair of
  table-driven subtest blocks, precisely the idiomatic class the exclusion comment
  defends. The other is a genuine cross-file near-copy of a helper shared between
  two policy tests. The honest yield is one true finding per two hundred commits,
  at half precision. Over the most recent sixty commits: nothing.
- **A higher threshold for tests does not separate the two classes.** At twice the
  configured threshold the tree holds ten blocks, eight in tests, and every
  surviving test pair is sibling functions differing by one fixture value or one
  flag. The single genuine cross-file finding dies at that threshold with them.
  Block length does not distinguish idiomatic structure from copy-paste on this
  corpus. A test-specific threshold is therefore not a usable dial, and a
  diff-scoped build has only one: the base ref.
- **Raising the global threshold clears the tree instead.** At two and a half
  times the configured threshold, with every path exclusion deleted and the
  hook-installer clone exempted at its two declarations, the whole tree reports
  zero. Annotating only one end of that clone leaves the other firing.

## Resolution shape

Two shapes remain, trading coverage against apparatus.

**Diff-scoped at the configured threshold** catches that one true finding per two
hundred commits. It costs a second config file, a Makefile target, hook and CI
wiring, and a test pinning the wiring. CI has no host for it today: the lint job
checks out at default depth and so has no merge-base to diff against, while the
job that does carry full history has no `golangci-lint` on PATH.

**Raising the global threshold and emptying the path-exclusion catalogue** costs
a threshold edit, the deletion of all eleven exclusion rules and the comment
blocks justifying them, and a `//nolint:dupl` rationale on the two hook-installer
declarations that stay a clone — plus, per G-0473, a recorded decision for that
rationale to cite, so the exemption rests on something settled rather than on an
open gap's lean. No second invocation, no CI change, and no new gate. G-0473's
option 4 reaches the same end state from the catalogue's side: the one surviving
exemption moves to its source sites instead of remaining a path rule that blinds
the detector across the whole file. It gives the test corpus a detector it has
never had, at a sensitivity that would have caught nothing in the measured
window.

The second is the smaller change and the one to take first. What it gives up
should be stated rather than softened: detection in the band between the two
thresholds goes away, and that band is where the single genuine finding in the
measured window lived. The structural-sweep ritual does not backfill it — that
ritual runs the clone detector the project already gates on, so its sensitivity
moves with this threshold rather than compensating for it.

Dormancy afterwards is therefore weak evidence, and worth naming as such: it
would establish only that no clone above the raised threshold has arrived, and
nothing about the band below it, which is the band a diff-scoped build exists to
watch. What the raise really buys is a tripwire where there is none today, and a
decision deferred rather than foreclosed.

## Sequencing

Nothing blocks. G-0462, which had to be resolved before any golangci-lint
measurement here could be trusted, is addressed. Both shapes above are
independent of the clone collapses in G-0472, and the threshold raise is the
same edit G-0473's catalogue cleanup wants.
