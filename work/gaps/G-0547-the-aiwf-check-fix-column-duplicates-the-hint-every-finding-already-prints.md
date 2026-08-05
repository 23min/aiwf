---
id: G-0547
title: The aiwf-check fix column duplicates the hint every finding already prints
status: open
---
## What's missing

The `aiwf-check` skill documents every finding code in tables whose columns
are `Code | Meaning | Typical fix`. That third column is a second copy of
guidance the tool already produces: `applyHints` fills `Finding.Hint` from
`hintTable` for every finding at emission time, so an operator who actually
hits a finding is told the fix twice — once by the command that just ran, and
once by a static table nobody re-derives.

Measured on the current skill: the table carries 104 rows totalling roughly
49KB, of which about 12KB — a quarter — is fix guidance. The file as a whole
is the largest shipped surface by a wide margin, over half again the size of
the next one, and it materializes into every consumer repo.

Two smaller consequences fall out of the same column and are worth resolving
together rather than separately:

- The findings tables disagree about their own shape. Three of the four are
  three-column; the warnings table is two-column, with its fix guidance
  crammed into the meaning cell behind an inline `Fix:`. Four rows had been
  written in the majority shape and sat in the minority table, so markdown
  dropped their third cell and the guidance never rendered at all.
- Nothing derives the column from the hint table, so the two can disagree
  about the same code and no check notices.

## Why it matters

The duplication is the kind a check cannot police and a reader cannot detect:
the copy and the original are far apart, only one of them runs, and the stale
one is the one a consumer reads while deciding what to do. A wrong fix in the
table is worse than no fix, because it reads as authoritative.

Size matters here for a reason particular to this file. A shipped skill is not
documentation a reader opens on demand — it is materialized into the consumer's
tree and enters an assistant's context. A quarter of the largest such surface
being a second copy of something the tool prints anyway is paid for on every
session that loads it.

## Direction

The question to answer first is what the column is for, because the answers
diverge sharply:

- **Delete it**, and let the hint the tool prints be the single source. The
  table then answers "what does this code mean and will it block me", and the
  tool answers "what do I do about it". Smallest surface; costs a reader who
  is browsing rather than reacting to a finding.
- **Derive it** — generate the column from `hintTable` at materialization time,
  so the two cannot drift. Keeps the browsing affordance, adds a generation
  step to a file that is currently hand-authored.
- **Keep it hand-written and pin it** — assert every row's fix cell against the
  code's hint. Cheapest to build, but it pins two prose texts to each other
  rather than removing the second copy.

Whichever is chosen settles the schema question too: deleting the column makes
all four tables two-column, deriving or pinning it makes them all
three-column. Splitting the warnings table's nineteen inline `Fix:` clauses
into their own column before that decision would be wasted work if the column
is going away.

## Provenance

Measured 2026-08-05 while closing G-0542, which moved rows between these
tables and prompted the question of why the file is as large as it is. The
condition predates that work; the four dropped cells were repaired there,
since their text was invisible to every consumer, but the column itself was
left alone as a separate decision.
