---
id: G-0504
title: doctor byte-checks only verb skills; ritual and guidance drift read as healthy
status: open
priority: high
---
## What's missing

`aiwf doctor` byte-compares only the verb skills. Three distinct artifact
families materialize into a consumer's `.claude/`, and drift in two of them is
invisible:

- **Verb skills — byte-checked.** `skills.List()` (`internal/skills/skills.go`)
  reads the `embedded/` tree and filters on the `aiwf-` name prefix. Those
  entries are the entire corpus `skillDrift` compares in
  `internal/cli/doctor/doctor.go`, and they are what the `skills: ok (N skills,
  byte-equal to embed)` line asserts. The claim is true, and it is scoped to that
  family alone.
- **Rituals — presence only.** `checkRitualsResult`
  (`internal/cli/doctor/check_rituals.go`) asks `skills.MaterializedRituals` for
  present-versus-missing and reports ok when nothing is missing. Content is never
  compared. `ListRituals`, `ListRitualAgents` and `ListRitualTemplates` all exist
  and return exactly the bytes a comparison would need; the doctor package calls
  none of them.
- **The guidance fragment — wiring only.** `internal/cli/doctor/guidance.go`
  holds a single reporter, which verifies that the consumer's `CLAUDE.md` carries
  the import. The fragment's own bytes are never examined.

So `skills: ok` alongside `rituals: ok` is fully compatible with every ritual
skill, every role-agent card, every entity template, and the always-on guidance
fragment being arbitrarily stale.

## Why it matters

The always-on guidance fragment is `@`-imported into the consumer's `CLAUDE.md`,
so a drifted copy silently governs an assistant's behavior for the whole session.
The rituals are the procedures an assistant follows to plan, implement and wrap
work. Both are the surfaces where staleness is least visible and most
consequential, and both are the ones `doctor` declines to inspect — while
reporting health in a way a reader reasonably takes to cover them.

Observed rather than hypothesized: a session found four ritual skills and the
guidance fragment all differing from their embedded sources while `aiwf doctor`
reported `skills: ok (19 skills, byte-equal to embed)` and `rituals: ok (27
artifacts materialized)`.

The same session showed the drift runs in both directions. The materialized
guidance fragment carried a *newer* version stamp than the installed binary — the
artifacts had been written by one binary and the verbs were being run by another.
That is the ordinary consequence of this repo's own worktree-binary discipline,
where a diagnostic build coexists with the installed one, so the condition is
routine rather than exotic.

Distinct from G-0471, which concerns a binary older than the worktree's source.
This is the artifact-versus-embed axis: the binary can be perfectly current while
the materialized artifacts are stale, and vice versa. Neither detects the other.

## Resolution shape

Extend the drift comparison to the two unchecked families, reusing the list
functions that already exist rather than adding machinery — the missing piece is
the comparison and its report line, not the data source.

Two things to settle while doing it. Whether ritual drift is an error like verb-
skill drift or a warning, given rituals are advisory-only elsewhere in the report
and a missing ritual is deliberately non-fatal today. And whether the report
keeps one `rituals:` line covering both presence and content, or splits them, so
an operator can tell "never materialized" from "materialized and since drifted" —
the two have different causes and the same remedy.

Detection is half of it. The two planning rituals fill an entity's body from
`.claude/templates/<kind>-spec.md` and name `aiwf update` as the remedy only
when that file is absent — the one state a reader notices unaided. Against a
present-but-stale template the instruction gives no signal at all: measured
2026-08-13, this repo's own materialized `epic-spec.md` carries the pre-M-0306
nesting of `### Out of scope` under `## Scope`, so an epic drafted by following
it carries no `out_of_scope` key, with `aiwf check` green. That file is
gitignored, so neither review nor history sees it either. The instruction and
the report belong in one change — the refresh needs no detection to be correct,
and a report that names drift is worth more beside a ritual that refreshes
before it reads.

Worth pinning the property itself, not only the fix: a test that materializes an
artifact, mutates a byte, and asserts the report names it. Without that, the next
family added to the embed rejoins the unchecked set silently, which is how the
two current families got here.

## Where to fix

- `internal/cli/doctor/doctor.go` — the drift comparison and its report line.
- `internal/cli/doctor/check_rituals.go` — presence-only ritual result.
- `internal/cli/doctor/guidance.go` — import-wiring-only guidance report.
- `internal/skills/skills.go` — the ritual and agent/template list functions the
  comparison can reuse as-is.
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md`
  and its `aiwfx-plan-milestones` sibling — the step-5 instruction whose
  `aiwf update` remedy reaches only an absent template.

## Related

- G-0471 — the binary-versus-source staleness axis, addressed by E-0076. This gap
  is the artifact-versus-embed axis; neither detects the other.
