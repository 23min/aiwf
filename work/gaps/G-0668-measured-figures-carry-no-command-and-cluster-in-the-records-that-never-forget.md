---
id: G-0668
title: Measured figures carry no command, and cluster in the records that never forget
status: open
---
## What's missing

A measured figure written into planning prose keeps its answer and discards its
question. *"10,801 commits, of which 54 carry the trailer alone."* *"G-0657
records 50 commits."* Each came from a command someone ran; the prose records the
number and not the command.

A later reader cannot re-run it, because there is nothing to re-run. They
re-derive it instead, choosing their own predicate, and arrive at a different
number — which reads as a correction rather than as an answer to a different
question. On M-0327 one count ran 50 → 51 → 50 → *"50 under the predicate this
milestone ships, and 53 under the looser one the gap states"* across three review
rounds, ending by stating both. The behaviour under review was sound throughout;
the rounds were spent on the figure.

The failure takes two forms with one cause.

- **Wrong when written** — a count computed under a predicate other than the one
  the sentence names. M-0327's is this form.
- **Right when written, then stale** — a coverage percentage or an inventory
  count, correct on the day and rotting as the code moves: *"207 existing
  violations to grandfather"*, *"the 203 sites"*, *"87% statement"*. The
  tree-wide sample is mostly this form.

Both close the same way: record the command and the snapshot beside the figure,
so the next reader re-runs rather than re-derives. The kernel already requires
that shape where an acceptance criterion claims an observation — the command, the
result expected, the output observed, the environment. Nothing asks it of a
figure in a `## Context` paragraph or a gap body.

**Where the unsourced figures sit.** Measured at `056e03c49` over all 1,218
entity bodies under `work/`: strip frontmatter and fenced code, split on blank
lines, mask entity ids, ISO dates, semver, shas and `file:line` spans, then count
a paragraph as asserting a figure when it carries a thousands-separated number, a
percentage, an `N of M`, or a two-or-more-digit number bound to a plural noun —
and as carrying its source when the same paragraph holds a backticked `git` /
`aiwf` / `make` / `grep` / `go test` invocation or a commit sha.

- Live gaps, decisions and ADRs — records that do **not** forget — 79 of 119 such
  paragraphs carry no command.
- Milestone specs and wrap artefacts, which archive and forget — 203 of 515.

The comparison is the finding, not the levels. Both sides are measured by the
same biased predicate, so its error largely cancels between them; it would take a
bias differing by entity kind to overturn the ordering.

**Two measurements not worth repeating.** The absolute count is unreliable: on a
fourteen-paragraph sample the figure predicate runs near 55% precision, and the
source predicate misses a method stated in prose rather than as a command, so it
errs in both directions at once. And there is no trend — split at the timeline
midpoint, z = 0.85, on a date attribution that resolves 119 of the 880 paragraphs.

## Why it matters

D-0084 homes a record by lifetime: reasoning goes where it cannot go stale, and
an implementation argument goes into the milestone spec precisely because that
file is already on the forgetting pipeline. For figures the same logic runs
backwards. The records that never forget — a live gap, an accepted decision — are
where unsourced figures are densest, and they are the ones a later reader trusts
because nothing about them announces an expiry.

It also costs review at its most expensive point. Across M-0327's eight rounds
the figure corrections were themselves corrected, because each round re-derived
rather than re-ran, and a round that cannot reproduce the previous round's number
cannot tell a mistake from a different question.

Writing rules do not reach this. The shipped rule against copying a fact another
record holds does not apply, and correctly so: these numbers are genuine one-off
measurements that no other record carries, and writing them down is right. What
is absent is the half that makes one checkable.

Related but distinct: G-0659 records the two classes of overturned claim and
routes the drift class; this is the figure class specifically, which routing does
not reach, since a wrong or stale number is equally wrong in every destination.
G-0665 asks which surface owns an acceptance criterion's body; a figure rule
binds wherever a figure may be written, which is wider than that.
