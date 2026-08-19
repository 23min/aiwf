---
id: D-0069
title: Reject the dispatched reading pass; keep the lab rule
status: accepted
relates_to:
    - E-0086
    - M-0310
    - M-0311
---
> **Date:** 2026-08-19 · **Decided by:** human/peter

## Question

E-0086 set out to build a milestone-preflight pass: an independent reviewer,
dispatched with a written brief, reads a milestone spec against a corpus of
commitments, entities and documents, and returns a ledger of rows. The
specification is `docs/initiatives/milestone-preflight-as-independent-review.md`;
this decision does not restate it.

One milestone of that epic reached an artifact. The epic's own planning ran the
method on itself four times and recorded the result, so whether the pass earns
its cost can be answered from measurements the work already produced rather than
from a further trial.

## Decision

The dispatched generative reading pass is rejected. E-0086 keeps one rule and
drops everything else.

**Kept — the lab rule.** A claim is settled true only where a command, its
expected result, its observed output and its environment sit together. Nothing
else settles a claim: not a reading, not a reviewer's confidence, not a citation
to a record that says so. This is a ban, so it costs once — at the moment
someone writes a claim down — and it names no procedure anyone must run.

**Dropped** — the dispatched generative reading pass; the sweep; the criterion
challenge; the ledger's disposition column as a mandated artifact; the wrap-gate
enumeration; the per-seam instrumentation; the seam instructions in
`aiwfx-wrap-milestone` and `aiwfx-wrap-epic`; the `docs/design/oracles.md` row
claiming a model-judged advisory class.

Of the dropped parts, only the reading brief reached an artifact. The rest were
planned and never built:

```
$ grep -c preflight docs/design/oracles.md \
    internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md \
    internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md
```

expected, if those seams were never written: `0` for each path. Observed: `0`
for all three.

## Reasoning

A generative review over prose has an unbounded output space. A reader asked
what a document gets wrong can always produce another row, so the finding count
measures reviewer effort rather than artifact quality, and successive rounds are
not comparable. Zero is unreachable by construction. That is fatal here
specifically, because the epic's stated purpose was to produce a number saying
whether the method works — what a pass missed and what it added — and a count
drawn from an unbounded space cannot carry that meaning.

Four measurements support this. Each names the command that produced it, what
the command would have shown had the method held, and what it actually returned.
All were run in the worktree `.claude/worktrees/initiative/preflight-amendments`,
on branch `initiative/preflight-amendments` at `087dbfc07`, with the branch
`milestone/M-0311-ship-the-reviewer-brief-as-an-invocable-skill` present locally
at `7f4d275aa`; entity queries used a binary built from that worktree by
`make diag-aiwf`, reporting `initiative/preflight-amendments@v0.32.0-787-g087dbfc07`.
Commands that read this epic's own body are pinned to `087dbfc07`, the commit
preceding this re-scope, so they keep reproducing after it.

### The pass does not converge, measured on itself

This epic's `## Spec measurement` section records four sweeps — same brief,
fresh reviewer each — against successive revisions of one epic draft. Its
closing paragraph states the outcome: later sweeps reached sources the earlier
ones did not, and *"each repair round introduced a fresh contradicted row."*
Repairing what a round found is what generates the next round's findings.

```
$ git show 087dbfc07:work/epics/E-0086-build-the-smallest-preflight-pass-that-can-be-trialled-and-can-fail/epic.md \
    | sed -n '/^## Spec measurement/,/^Every row is unsettled/p' \
    | grep '^| ' | tail -n +2 | wc -l
$ git show 087dbfc07:work/epics/E-0086-build-the-smallest-preflight-pass-that-can-be-trialled-and-can-fail/epic.md \
    | sed -n '/^## Spec measurement/,/^Every row is unsettled/p' \
    | grep '^| ' | tail -n +2 | awk -F'|' '{gsub(/ /,"",$4)} $4!=""' | wc -l
```

expected, if reading settles claims: most rows carry a value in the `measured`
column. Observed: `29` and `1`. Four sweeps returned 29 rows; exactly one was
settled by running a command. The section's own next line says the same thing in
prose — *"Every row is unsettled as to truth."*

That is the result in one ratio. Twenty-eight of twenty-nine rows are a reader's
opinion about a document, carried in a table shaped like measurement.

### The reading half mis-cleared a false claim it was pointed at

The specification records a blind trial: the sweep was run against M-0307, a
milestone cancelled at preflight, with the outcome withheld and every read
pinned to the ref preceding cancellation.

```
$ sed -n '/^### The blind trial/,/^### The premise, measured/p' \
    docs/initiatives/milestone-preflight-as-independent-review.md
```

expected, if reading is a safe verdict surface: the three defects that cancelled
M-0307 are found, or raised as needing measurement. Observed: the first row of
the trial's table reads `the premise was wrong — the path resolved for no kind,
not two | **reported sound**`. The section's own gloss: *"the reviewer did not
merely miss it but marked the claim sound."*

A reader that mis-clears is worse than no reader, because it manufactures
confidence where there was a question. The specification answers this with a
prohibition — reading may never clear a question — which is the correct fix and
is simultaneously the admission that the reading pass's output cannot serve as a
verdict.

### The same shape was measured before, on a different subject, and lost

D-0050 (`accepted`) already bans phrase-content assertions over prose. Its
Reasoning records what four review rounds over the G-0489 cohort produced.

```
$ sed -n '/^## Reasoning/,/^## Consequences/p' \
    work/decisions/D-0050-pin-ritual-prose-by-structure-not-by-phrase-content.md
```

expected, if a review loop over prose converges: successive rounds return fewer
blocking findings. Observed: *"successive rounds found 3, 8, 9, then 10 blocking
findings, with no downward trend"*, and of roughly thirty findings across those
rounds, *"more than half were defects in the assertions rather than in the prose
they guarded."*

That is the same failure mode in different dress: a mechanical-looking surface
over prose which *"looks objective while behaving like a judgment finding — the
class D5 itself says does not converge under looping."* The preflight pass is
the generative version of what D-0050 already rejected in its assertion-shaped
version, and nothing in this epic argued why the generative form would converge
where the mechanical form did not.

### What the epic produced, priced

```
$ git diff --numstat main...milestone/M-0311-ship-the-reviewer-brief-as-an-invocable-skill -- '*.go'
$ git diff --numstat main...milestone/M-0311-ship-the-reviewer-brief-as-an-invocable-skill -- work/ docs/ \
    | awk '{a+=$1;d+=$2} END {print a, d}'
$ git diff --stat --diff-filter=A main...milestone/M-0311-ship-the-reviewer-brief-as-an-invocable-skill \
    -- work/ docs/adr/ | tail -1
$ git show milestone/M-0311-ship-the-reviewer-brief-as-an-invocable-skill:internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-preflight-brief/SKILL.md \
    | wc -l
```

expected: a shipped artifact roughly proportionate to the planning behind it.
Observed, across the whole range from the merge base with `main`: `112` lines of
Go in one test file; `1226` added and `84` deleted lines under `work/` and
`docs/`; `8` new entity files, three of them gaps; and a shipped skill of `141`
lines.

The range spans the epic's planning and its one milestone together — restricted
to the milestone's own commits it is `+5 -3` on one file and no new entities, so
the prose is the epic's, not that milestone's. Either way the ratio is the
finding: eight new entities and twelve hundred lines of prose produced one
141-line skill whose third section instructs its reader not to measure.

```
$ git show milestone/M-0311-ship-the-reviewer-brief-as-an-invocable-skill:internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-preflight-brief/SKILL.md \
    | grep -n '^## '
```

expected, if the shipped brief permits measurement: no prohibition among its
headings. Observed: eight headings, the third being `## You may not measure`.

### What survives, and why it is the part with evidence

The specification names the lab as its best-evidenced thread.

```
$ sed -n '/^### T5 — the lab/,/^### T6/p' \
    docs/initiatives/milestone-preflight-as-independent-review.md
$ git show 087dbfc07:work/epics/E-0086-build-the-smallest-preflight-pass-that-can-be-trialled-and-can-fail/epic.md \
    | sed -n '/^## Milestones/,/^## References/p'
```

expected, if the epic sequenced by evidence: the lab is built first. Observed:
the specification calls it *"the best-evidenced part of the specification: in
the motivating episode, measuring the spec's claims caught every defect on its
own"*, and the epic's Milestones section lists it fourth of seven. It was never
reached.

The epic therefore sequenced the unevidenced reading half ahead of the evidenced
measuring half, spent its budget there, and left the one part with a recorded
success behind it unbuilt. Keeping the lab rule and dropping the rest inverts
that ordering at no build cost, because the rule is a ban rather than a
procedure: it constrains what may be written down, and needs no dispatch, no
ledger, no seam and no instrumentation to take effect.

### Alternatives rejected

- **Run the trial as planned and let the number decide.** This was the epic's
  own design, and the four sweeps above are that trial's first data. It returned
  29 rows and settled 1. Continuing spends six more milestones producing a count
  whose scale is set by how hard the reviewer looked.
- **Keep the pass and bound it — cap the rows, or narrow the corpus.** A cap
  makes consecutive runs comparable by fiat rather than by convergence, and the
  measurement showed later sweeps reaching sources earlier ones did not, so a
  narrower corpus changes which rows appear rather than how many. Neither
  touches the unbounded output space; both hide it.
- **Keep the ledger without the pass.** The ledger's value is its `measured`
  column, which the lab rule already governs. The other four columns are the
  reading pass's output format, and a mandated artifact for output nobody
  generates is a per-subject cost with no subject.

## Consequences

- **E-0086 closes re-scoped**, with Scope, Success criteria, Milestones and its
  Threads table reduced to the lab rule.
- **M-0310 is cancelled.** Its subject is what a pass run records about
  itself — instrumentation for a pass that no longer runs.
- **D-0067 and D-0068 lose their subject and stay `accepted`.** D-0067 places
  disposition at a wrap gate rather than in a check rule; D-0068 defines what
  counts as a defect the pass missed. Each remains correct about what it
  decided, and neither now has a pass to govern. Whether they are superseded is
  a separate call, deliberately not taken here.
- **The initiative is not rewritten.** It stays a Forward-looking document; its
  Threads table records which threads were promoted, which were dropped, and
  why, as a dated inventory.
- **`wf-preflight-brief` is the reading half**, shipped by M-0311 on an unmerged
  branch. Its disposition is an open question at the time of writing and is not
  settled here.

## Provenance

Decided while re-scoping E-0086, from measurements the epic's own planning and
the specification's blind trial had already recorded. The specification is
unchanged by this decision except for its Threads inventory.
