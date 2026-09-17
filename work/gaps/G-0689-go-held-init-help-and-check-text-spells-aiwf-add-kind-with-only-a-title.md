---
id: G-0689
title: Go-held init, help and check text spells aiwf add <kind> with only a title
status: open
---
## What's missing

Three surfaces that ship from Go source, rather than from embedded markdown, tell
a reader to create an entity with `aiwf add <kind> --title "..."` alone. `aiwf add`
refuses that create for every kind but an epic: a milestone also needs `--epic`
and `--tdd`, and an adr, gap, decision or contract is created with its body.

- `internal/initrepo/initrepo.go` `CLAUDETemplate` — the `CLAUDE.md` that
  `aiwf init` writes into a consumer repo that has none pairs the spelling with
  "epic, milestone, adr, gap, decision, contract".
- `internal/cli/root.go` `printHelp` — the `aiwf --help` verb list shows
  `add <kind> --title "..."`, and its "Flags for 'add'" block lists `--body-file`
  without saying which kinds need a body.
- `internal/check/hint.go` — the `archived-entity-not-terminal` hint says to open
  a new entity "via `aiwf add <kind> --title "..."`".

The ban that holds shipped skills and `--help` Examples to this,
`internal/policies/born_complete_add_spelling_test.go`, reads neither Go string
constants nor the top-level help, and a `<kind>` placeholder names no kind it
could judge.

Measured 2026-09-17 in a scratch repo, with a binary built from `main` at
`4a360e1f3` (stamped `+dirty` for an uncommitted `TODO.md` edit and an untracked
initiative doc, neither of which the binary compiles in or embeds):

```
$ aiwf init
$ grep -n 'aiwf add' CLAUDE.md
8:- `aiwf add <kind> --title "..."` — create an entity (epic, milestone, adr, gap, decision, contract).
$ aiwf add epic --title "Kind epic probe"; echo "exit $?"
aiwf add epic E-0001 "Kind epic probe"
exit 0
$ aiwf add milestone --title "Kind milestone probe"; echo "exit $?"
aiwf add: --tdd <required|advisory|none> is required for kind=milestone (G-055: every milestone must declare its TDD policy explicitly)
exit 2
$ aiwf add adr --title "No body adr"; echo "exit $?"
aiwf add: ADR-0001: empty load-bearing body section(s) `## Context`, `## Decision`, `## Consequences`; an adr is referenceable the instant this commit lands, so its body must carry meaning at creation — pass --body "..." or --body-file <path> with real prose, or --force --reason "..." to create anyway (aiwf check will still flag it at error severity and the pre-push hook will still block until it's filled in)
exit 2
$ aiwf --help | grep 'add <kind>'
  add <kind> --title "..."       create a new entity of the given kind
$ aiwf add gap --title "Probe gap" --body-file g.md
$ aiwf promote G-0001 wontfix --reason probe
$ aiwf archive --apply
$ sed -i 's/^status: wontfix/status: open/' work/gaps/archive/G-0001-probe-gap.md
$ aiwf check --verbose | grep archived-entity-not-terminal
work/gaps/archive/G-0001-probe-gap.md:4: error archived-entity-not-terminal: [...] if the entity genuinely needs revisiting, open a new one that references it via `aiwf add <kind> --title "..."` (ADR-0004 §Reversal)
```

## Why it matters

A consumer's first `CLAUDE.md` and the top-level help are among the first things
an assistant reads in a new repo, and the check hint is what it follows when an
archived entity needs revisiting and a replacement has to be opened. Each hands it
a create that exits 2 for five of the six kinds. The refusal names the fix, so the
cost is a failed command rather than bad state.

## Related

- G-0681 removed the same bodyless create from the shipped skills and rituals,
  and from `aiwf add --help`.
