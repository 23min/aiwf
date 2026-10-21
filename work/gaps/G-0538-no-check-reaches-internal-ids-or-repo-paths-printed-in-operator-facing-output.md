---
id: G-0538
title: No check reaches internal ids or repo paths printed in operator-facing output
status: open
priority: high
---
## What's missing

Aiwf-internal ids and paths into aiwf's own source tree reach a consumer through
Go string literals that no rule scans.

**Persisted in the consumer's repo.** The four hook scripts `aiwf init` writes —
pre-push, pre-commit, commit-msg, post-commit — carried internal ids in their
shell comments, templated in `internal/initrepo/initrepo.go`. So did the
`.gitignore` header `ensureGitignore` writes, which is the worst of the set: a
hook lives in `.git/`, but `.gitignore` is a file the consumer commits, so the
id enters their history. Both are fixed, and four goldens pin the hook text
byte-for-byte, but no id rule reads a generated artifact: a new citation in a
builder lands unreported until someone reads the golden diff.

**Printed to the consumer's terminal.** `Finding.Hint` text in
`internal/check/hint.go`, rendered by `aiwf check` wherever it runs, cites
internal ids, among them `G-0155` in the `git-config-core-worktree-misset` hint,
`G-0063` in `epic-active-no-drafted-milestones`, `ADR-0004 §Reversal` in
`archived-entity-not-terminal`, `M-067` in the `entity-body-empty` hints, and
`G-0231`, `E-0030`, `D-0019` and `ADR-0010` in the provenance and branch hints.
The `entity-body-empty` hints also illustrate the `aiwf edit-body` argument
below canonical width, as `E-NN`, `M-NNN`, `G-NNN`, `D-NNN` and `C-NNN`.

Not every id shape in that file is a defect: the `body-prose-id` hints show
`M-1` and `M-007` as the malformed spellings they are about, so a rule extended
over the hints has to tell an illustration of a wrong token from a citation.

Finding messages cite ids too: `archived-entity-not-terminal` cites `ADR-0004`
(`internal/check/archive_rules.go`), `entity-id-narrow-width` cites `ADR-0008`
(`internal/check/entity_id_narrow_width.go`), and `promote-on-wrong-branch`
cites `ADR-0010` (`internal/check/promote_on_wrong_branch.go`).

Width is a separate axis from citation, and the population shows the two are
independent: some citations were unhyphenated, which is the detection floor
G-0369 owns, while others were already canonical and would pass any width rule
while being exactly as meaningless to the consumer reading them.

Measured on Linux (go 1.25) by parsing every non-test Go file under `internal/`
and `cmd/` and classifying each whitespace-bearing string literal with
skill-body-id's token rules, on the tree that carries the fix recorded below.

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

An aiwf id is meaningless in a consumer repo and rots as the entity it names
changes status or moves to archive. Where the consumer has entities of its own,
it is worse than meaningless: `ADR-0004` in a hint resolves to the consumer's
own ADR-0004, which is about something else, and `aiwf show` sends them there.

## What is already fixed, and what is not

Fixed: the four hook templates, the `.gitignore` header, the hook-collision
message in three verbs, command help (per-command Cobra help and the root help
`printHelp` renders), `aiwf doctor` output, the messages verbs return, the commit
body `aiwf archive` writes, configuration, `aiwf.yaml` and trailer errors, and the
config schema's field descriptions, which `aiwf init` writes into the consumer's
committed `aiwf.yaml` as comments. The policy `cli-text-internal-id`
(`internal/policies/cli_text_internal_ids.go`) holds every whitespace-bearing
string literal in the non-test Go files of those packages to skill-body-id's id
classification, and additionally rejects unhyphenated gap labels and paths that
exist in aiwf's source tree, other than `docs/adr/`.

Not fixed: the `Finding.Hint` literals and finding messages in `internal/check`,
and a rule over generated artifact text. `internal/check` is one more entry in
`cli-text-internal-id`'s scope list once its text is clean, less the hints that
show a malformed token as their subject.

Bare `ADR-NNNN` citations are in class. A consumer's repo allocates its own ADR
ids from the same sequence, so an aiwf ADR citation names the consumer's
unrelated ADR rather than nothing. CLAUDE.md's doc-link carve-out covers a
markdown link whose visible text stays descriptive, which a parenthetical in a
string literal is not.

## Resolution shape

**Generated artifact text** is the tractable half. The hook scripts are returned
by builder functions in `internal/initrepo`; call the builders and run the
produced bytes through the same classification `ScanPlainTextIDs` applies.
That judges the golden fixture and the live template by one standard, and
extends to any future artifact by registering its builder. The `aiwf.yaml`
comments and `aiwf.example.yaml` are generated from the schema, whose literals
the policy already judges.

**Printed output** reduces to string literals rather than one call site: flag
help reaches Cobra as `cmd.Flags().XVar(…, usage)` and command descriptions as
`cobra.Command` fields, never through the `cliutil` wrappers, and `Finding.Hint`
literals are rendered downstream by the formatter. `cli-text-internal-id`
scans literals for that reason. An id the operator passed reaches a message
through a format verb, never as part of a literal, so echoing it back needs no
exemption; a literal carrying no whitespace is data — an argument, a fixture id,
a path the code matches against — and is not judged.

Where the remaining rules live follows from whether they should be inert in a
consumer tree, as `skill-body-id` is: `internal/check` beside that rule if so,
`internal/policies` beside `cli-text-internal-id` if the property is an
aiwf-repo invariant.

Either way the enumerated definition in CLAUDE.md gains these surfaces as
members, so the prose and the chokepoints name the same set.
