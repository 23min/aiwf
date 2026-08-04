---
id: G-0538
title: No check reaches internal ids printed in operator-facing output
status: open
priority: medium
---
## What's missing

Three verbs print an aiwf-internal gap id to a consumer's terminal:

- `internal/cli/initcmd/initcmd.go:136`
- `internal/cli/update/update.go:155`
- `internal/cli/worktree/worktree.go:168`

Each emits `hook chain collision (G45).` on the path where a `.local` hook
already exists. That token names G-0045 in this repo and nothing at all in a
consumer's, and the three verbs carrying it are among the ones a consumer runs
most.

No rule reaches them. `skill-body-id` fires over `*.md` under
`internal/skills/embedded{,-rituals,-guidance}/**` plus the `#` comments of
`embedded-statusline/*.sh`; `body-prose-id` scans entity bodies; `doc-id-width`
scans the configured documentation corpus. Go string literals belong to none of
those surfaces, so text that materializes in a consumer's terminal is held to a
weaker standard than text that materializes in their `.claude/` directory.

The unhyphenated spelling is separately invisible, which G-0369 owns. The two
absences are independent: written canonically, an internal id in a
consumer-facing print would still pass, because no rule reads Go strings.

## Why it matters

E-0078 purged narrow-id debris from shipped surfaces and closed. These three
lines are that debris and survived it, because "shipped surface" is defined by
enumeration — SKILL.md bodies and their `description:` frontmatter, entity
templates, role-agent cards, the always-on guidance fragment, the statusline's
comments — and every member of that list is a markdown or shell artifact.
Operator-facing output is shipped by function and absent from the definition.

The reasoning behind the rule applies here unchanged: an aiwf id is meaningless
in a consumer repo and rots as the entity it names changes status or moves to
archive. A consumer reading `hook chain collision (G45)` cannot resolve it and
has no reason to want to.

## Resolution shape

Two halves, and only the second carries a decision.

The citations are a text fix. The message reads better without the id at all —
the lines that follow it already state the condition and the remedy.

The chokepoint is the open question, in two parts. What surface should a rule
scan, and how does it separate operator-facing text from diagnostic text? The
`cliutil` wrappers are the seam that already exists: `forbidigo`, backed by the
`logging-chokepoint` AST policy, establishes that operator text routes through
`cliutil` and diagnostics through `log/slog`. A rule scanning string literals
passed to the `cliutil` text wrappers therefore has a defined surface rather
than all of Go source.

Where it lives follows from whether it should be inert in a consumer tree, as
`skill-body-id` is: `internal/check` beside that rule if so, `internal/policies`
beside the other AST scans if the property is an aiwf-repo invariant.

Either way the enumerated definition in CLAUDE.md gains operator-facing output
as a member, so the prose and the chokepoint name the same set.
