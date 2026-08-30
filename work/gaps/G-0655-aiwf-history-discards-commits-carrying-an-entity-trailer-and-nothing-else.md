---
id: G-0655
title: aiwf history discards commits carrying an entity trailer and nothing else
status: open
---
## What's missing

`aiwf history <id>` selects commits by grepping for an `aiwf-entity:` trailer
naming the id, then discards any whose `aiwf-verb:` and `aiwf-actor:` trailers
are both empty (`internal/entityview/historyevent.go`). The discard exists to
drop a false positive: `--grep` matches a wrapped prose line that begins
`aiwf-entity: <id>`, where git's trailer parser finds no genuine trailer at all.
Proxying that test through verb-or-actor also discards commits whose entity
trailer is genuine and whose only fault is carrying nothing else.

Measured 2026-08-30 over 10,709 commits: 8,344 carry an `aiwf-entity:` trailer,
and 44 carry it alone. Every one of those 44 is invisible to `aiwf history`.

The repo mandates that exact shape. `internal/policies/skill_edit_provenance_backstop.go`
fails the gate when a commit modifying a `SKILL.md` under
`internal/skills/embedded-rituals/**` carries no `aiwf-entity:` trailer, and
CLAUDE.md states plainly that `aiwf-entity` is the whole of the requirement —
no verb, because the closed set has no value meaning "I edited a shipped
surface", and no actor, because where no verb ran there is no actor to record.
So the one rule that requires a bare entity trailer produces commits the history
verb then refuses to show.

The live instance: `2c79510fa` is the commit that implemented G-0650, and it
carries `aiwf-entity: G-0650` because the backstop requires it. `aiwf history
G-0650` lists the add, two body edits and the promote — the paperwork — and not
the commit that did the work.

The same blindness is why a milestone's `## Work log` is the only index from an
acceptance criterion to the commit that implemented it: an AC implementation
commit could carry `aiwf-entity: M-NNNN/AC-N`, and composite ids are already
valid in that trailer, but history would discard it.

## Why it matters

`aiwf history` is one of the six things aiwf commits to — "no separate event
log; structured trailers make the log queryable". A commit that carries the
trailer the repo demanded and is then dropped from the query is that guarantee
failing quietly, in the direction hardest to notice: the output looks complete.

The cost compounds where the answer matters most. Asking what changed an entity
returns its status flips and body edits while omitting the code, so the record
is thorough about process and silent about work. Anyone reconstructing why a gap
was closed reads a history that never names the fix.

It also blocks work elsewhere. Retiring the milestone `## Work log` (G-0530)
depends on the kernel being able to supply the AC-to-commit link, and it cannot
while this holds.

## Where to fix

- `internal/entityview/historyevent.go` — the query already extracts eleven
  trailer keys through git's own parser; extracting `aiwf-entity` the same way
  and testing that instead of verb-or-actor distinguishes a genuine trailer from
  a prose match by the mechanism the false positive was about, rather than by a
  proxy. A surfaced commit carrying no verb needs a rendering decision.
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md`
  — the per-AC implementation commit would carry `aiwf-entity: M-NNNN/AC-N`.

Changing the filter changes existing output: the 44 commits above begin
appearing in histories that do not show them today. That is the defect being
fixed rather than a regression, but tests pinning history output will move.
