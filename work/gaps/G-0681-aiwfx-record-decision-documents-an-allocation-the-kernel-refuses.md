---
id: G-0681
title: aiwfx-record-decision documents an allocation the kernel refuses
status: open
---

## What's missing

Shipped surfaces spell `aiwf add` invocations for the born-complete kinds that the
verb refuses. G-0326 made adr, gap, decision and contract refuse a create whose
load-bearing sections are empty — which is every create carrying only `--title`.
Its fix updated the verb, the check rule and the `aiwf-add` skill, and reached no
other documenting surface.

Expected: a command a shipped surface spells, runs. Measured 2026-09-14 against a
scratch consumer repo initialized by a binary built from `main` at `df0dd2c5f`:

```
$ aiwf add epic --title "A scratch epic"                         → exit 0
$ aiwf add milestone --epic E-0001 --tdd required --title "..."  → exit 0
$ aiwf add ac M-0001 --title "A scratch criterion"               → exit 0
$ aiwf add adr --title "A scratch adr"                           → exit 2
$ aiwf add decision --title "A scratch decision"                 → exit 2
$ aiwf add gap --title "A scratch gap"                           → exit 2
$ aiwf add contract --title "A scratch contract"                 → exit 2
```

The four that refuse are exactly the set `entity.IsBornComplete` returns true for
(`internal/entity/entity.go:74`). Each names its empty sections and writes nothing.

Locating the spellings:

```
$ grep -rnE 'aiwf add (adr|gap|decision|contract)[^|]*--title' --include='*.md' \
    internal/skills/ | grep -v -- '--body' | wc -l
14
```

That is the filter's yield, not the defect's extent — it requires `--title` on the
same line, so it misses `aiwfx-record-decision` at lines 94, 153 and 166 and
`aiwf-add` at 289, which spell the same refused create without it.

Of the fourteen, the two in `aiwf-add` are transcripts demonstrating the refusal
itself. The rest divide by what the spelling is doing. `aiwf-contract` at 24, 57
and 197, `aiwf-area` at 42 and 66, and `aiwfx-record-decision` at 42 and 48
instruct a reader to run it. The parentheticals in `aiwfx-whiteboard`,
`aiwfx-start-milestone` and `aiwfx-wrap-milestone` gloss a ritual invocation
instead — there the imperative is to invoke `aiwfx-record-gap`, and the spelling
describes what that ritual runs. It describes it wrongly:
`aiwfx-record-gap/SKILL.md:109` prescribes the create with `--body-file`.

## Why it matters

A reader runs what a surface tells them to run, and these surfaces materialize into
every consumer repo. The reader most exposed is an assistant with no prior about
which flags a kind requires, meeting a refusal on the first command of a documented
procedure.

The refusal names `--force --reason` alongside the correct routes, and warns in the
same sentence that `aiwf check` will flag the result at error severity and the
pre-push hook will block. `aiwfx-record-decision` undercuts that warning: its step 3
operates on the empty skeleton only `--force` produces, so a reader who has just
been told the next step fills that body reads the warning as transient. The trap is
the ritual disagreeing with the warning, not the warning being absent.

Two surfaces assert outcomes that cannot occur. `aiwf-area` at 42 pairs the refused
command with `# → derives area: app-a` in the output position of a shell
transcript. And `aiwfx-whiteboard` at 155 states the rule that every verb
invocation in a skill body must resolve to a real command available today, on the
same line as a spelling that exits 2.

## Related

- G-0326 is the change that made these kinds born-complete.
- G-0560 carries this same defect class over the `docs/` tree.
- G-0678 established the shape that runs: the create carries `--body-file`, and the
  body lands in it.
- M-0307 cancelled a sweep of a neighbouring citation defect across the same tree.
