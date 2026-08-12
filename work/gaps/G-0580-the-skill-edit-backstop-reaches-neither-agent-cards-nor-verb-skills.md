---
id: G-0580
title: The skill-edit backstop reaches neither agent cards nor verb skills
status: open
---
## What's missing

The skill-edit structural-test backstop fails the profile-driven gate when a
`SKILL.md` under the embedded rituals changes and no test under `internal/policies/`
references its path. Its scope is exactly that: the embedded-rituals subtree, and a
filename of `SKILL.md`.

Two classes of shipped surface fall outside it. Role-agent cards, which live beside
the ritual skills but are not named `SKILL.md`. And the verb skills under
`internal/skills/embedded/`, which are `SKILL.md` files but are not under the
embedded-rituals subtree. Both materialize into a consumer repo on `aiwf update`.

## Why it matters

The backstop exists because a shipped surface edited without a referencing test is
how consumer-facing prose drifts unobserved. That reasoning applies identically to an
agent card and to a verb skill; only the path glob distinguishes them.

It is not theoretical. Both uncovered classes currently carry drifted body-scaffold
instructions, and the drift survived a prior repair pass precisely because that pass
was scoped to skills — the same scoping the backstop has.

Widening the glob is the obvious option and is not obviously right: it would impose
the mandate on more files, and a mandate costs per subject forever. Whether the
answer is a wider glob, a different backstop shape, or nothing at all is the decision
to make.
