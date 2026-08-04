---
id: G-0538
title: No check reaches internal ids printed in operator-facing output
status: open
priority: medium
---
## What's missing

Aiwf-internal ids reach a consumer through Go string literals, on two surfaces
that no rule scans.

**Persisted in the consumer's repo.** The four hook scripts `aiwf init` writes —
pre-push, pre-commit, commit-msg, post-commit — carried internal ids in their
shell comments, templated in `internal/initrepo/initrepo.go`. So did the
`.gitignore` header `ensureGitignore` writes, which is the worst of the set: a
hook lives in `.git/`, but `.gitignore` is a file the consumer commits, so the
id enters their history. Both are fixed; the surface is not.

**Printed to the consumer's terminal.** Three shapes, of which only the first is
fixed: the hook-chain-collision message in `aiwf init` / `update` /
`worktree add`; Cobra flag help, where `--help` on several verbs still names an
aiwf gap id in the usage string; and `Finding.Hint` text in `internal/check`,
rendered by `aiwf check` wherever it runs.

No rule reaches either. `skill-body-id` fires over `*.md` under
`internal/skills/embedded{,-rituals,-guidance}/**` plus the `#` comments of
`embedded-statusline/*.sh`; `body-prose-id` scans entity bodies; `doc-id-width`
scans the configured documentation corpus. Go string literals belong to none of
those surfaces, so text that materializes in a consumer's terminal — or in their
`.git/hooks/` — is held to a weaker standard than text that materializes in
their `.claude/` directory.

Width is a separate axis, and the population proves the two are independent.
Some of these citations were unhyphenated, which is the detection floor G-0369
owns; others were already canonical, and would have passed any width rule while
being exactly as meaningless to the consumer reading them.

## Why it matters

E-0078 purged narrow-id debris from shipped surfaces and closed. This population
is that debris and survived it, because "shipped surface" is defined by
enumeration — SKILL.md bodies and their `description:` frontmatter, entity
templates, role-agent cards, the always-on guidance fragment, the statusline's
comments — and every member of that list is a file that exists as a file. Text
assembled in Go and written out at runtime is shipped by function and absent
from the definition.

The statusline entry is the proof the reasoning already extends this far:
somebody decided shell comments in a materialized artifact count. A generated
hook script is that same artifact class, and differs only in being built from a
string literal rather than embedded as one.

The reasoning behind the rule applies unchanged: an aiwf id is meaningless in a
consumer repo and rots as the entity it names changes status or moves to
archive. A consumer reading a gap id in the hook aiwf installed in their repo
cannot resolve it and has no reason to want to.

## What is already fixed, and what is not

Fixed: the four hook templates, the `.gitignore` header, the hook-collision
message in three verbs, and the one `Finding.Hint` that named the commit-msg
hook's own gap. Four goldens now pin the hook text byte-for-byte, so that half
cannot regress silently.

Not fixed: Cobra flag help across several verbs, `aiwf doctor` output, and the
remaining `Finding.Hint` literals. These were left deliberately — they are a
systematic population rather than a handful, and sweeping them by hand is what
this gap exists to stop repeating. E-0078 swept the enumerated shipped surfaces
by hand and every surface named here survived it, because a hand sweep cleans
the instances someone thought to look at. The rule is the fix; the sweep is what
runs once afterwards so the rule starts green.

Bare `ADR-NNNN` citations in flag help are the same shape and were left with
them. CLAUDE.md's carve-out covers a markdown link whose visible text stays
descriptive, which a parenthetical in a usage string is not — but ADRs are
stable where gaps rot, so whether they are in-class is a judgment to settle when
the rule is written rather than assumed here.

## Resolution shape

The surfaces want different mechanisms, and the obvious single seam does not
cover them.

**Generated artifact text** is the tractable half. The hook scripts are returned
by builder functions in `internal/initrepo`; call the builders and run the
produced bytes through the same rule `skill-body-id` already applies to the
embedded surfaces. That judges the golden fixture and the live template by one
standard, and extends to any future artifact by registering its builder.

**Printed output** does not reduce to one call site. `forbidigo` and the
`logging-chokepoint` policy establish that operator text routes through the
`cliutil` wrappers, which suggests scanning literals passed to those wrappers —
but that misses most of the population. Flag help is handed to Cobra via
`cmd.Flags().XVar(…, usage)`, never to `cliutil`; `Finding.Hint` literals live
in `internal/check` and are rendered downstream by the formatter. A rule scoped
to `cliutil` call sites would pass a tree that still leaks on every `--help`.

The shape that covers it is a scan over the *string literals* of the
consumer-reachable packages — `internal/cli/...`, `internal/check`'s hint and
message tables — rather than over one call site. That is a wider net and needs
an exemption story for the cases where an id legitimately appears (an entity id
the operator themselves passed, echoed back in an error).

Where the rule lives follows from whether it should be inert in a consumer tree,
as `skill-body-id` is: `internal/check` beside that rule if so,
`internal/policies` beside the other AST scans if the property is an aiwf-repo
invariant. The generated-artifact half and the string-literal half may well land
in different homes for that reason.

Either way the enumerated definition in CLAUDE.md gains both surfaces as
members, so the prose and the chokepoint name the same set.
