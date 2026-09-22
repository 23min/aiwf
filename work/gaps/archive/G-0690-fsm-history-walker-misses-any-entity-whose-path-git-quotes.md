---
id: G-0690
title: FSM history walker misses any entity whose path git quotes
status: addressed
addressed_by_commit:
    - fa59ea6d4
---
## What's missing

`internal/gitops/revwalk.go` runs `git log --all --raw --no-abbrev -M` and
`parseRawPathLine` in the same file tab-splits each `--raw` line as printed. Under
git's default `core.quotePath=true`, a path holding any byte outside printable
ASCII is printed double-quoted and octal-escaped, and nothing unquotes it.
`internal/check/fsm_history_walker.go` keys its `pathToEntity` lookup on that
literal string, so the lookup misses, the touch is skipped, and the entity's status
history is never examined. The sibling walkers in
`internal/check/entity_body_section_dropped.go` and `internal/check/area_mistag.go`
run their git calls with `-c core.quotePath=false`; G-0684 records, for the
section-dropped gate, the byte class that setting does not stop git quoting — a
double quote, a backslash, a tab, a control byte. This walker sets nothing, so for
it the non-ASCII class is affected as well.

Measured in the devcontainer (Linux) with a binary built from `ab430b80a`, in a
fresh repository after `aiwf init` and one commit. Two gaps were created, the second
moved with `git mv` to a filename carrying `é`, both were promoted `open → addressed`
through `aiwf promote --by-commit`, and both were then hand-edited back to `open` in
one plain commit — the same illegal transition on two files.

```
$ git log -1 --raw --format=
:100644 100644 ec8525c 44b55f7 M	work/gaps/G-0001-plain-gap.md
:100644 100644 f38e3cd 7cee53b M	"work/gaps/G-0002-h\303\251llo.md"

$ aiwf check
work/gaps/G-0001-plain-gap.md: error fsm-history-consistent/illegal-transition: entity G-0001 status changed addressed → open in commit 05ea99ee — not a legal gap FSM transition and no aiwf-force trailer — hint: …
```

Expected: one `fsm-history-consistent/illegal-transition` finding per gap.
Observed: the ASCII-named gap only. The other is reported clean, and the run exits
1 on the sibling alone. `entity.PathKind` accepts the real filename, so the entity
loads and is audited as though its history were legal.

The path does not arise through the verbs: `aiwf add`, `aiwf rename` and
`aiwf retitle` each derive an ASCII slug (measured on the same binary — a title of
`Héllo wörld` lands as `G-0003-h-llo-w-rld.md`, and `aiwf rename G-0003 héllo` yields
`h-llo`). It takes a hand `git mv`, or a file authored outside the verbs, which the
loader accepts. `aiwf import` was not measured.

## Why it matters

`fsm-history-consistent` is what turns "every status change went through a verb or
a sovereign override" from a convention into a guarantee, and it runs in the
pre-push hook. An entity whose filename carries one non-ASCII byte is outside that
guarantee: its status can be hand-edited to any value, committed without a trailer,
and pushed, and the check reports the tree clean. Because the miss is silent, the
output cannot distinguish "this entity's history is legal" from "this entity's
history was never read".
