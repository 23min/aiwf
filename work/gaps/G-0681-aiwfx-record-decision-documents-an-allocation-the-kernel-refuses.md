---
id: G-0681
title: aiwfx-record-decision documents an allocation the kernel refuses
status: open
---
## What's missing

`internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-record-decision/SKILL.md`
documents an allocation the kernel refuses. Step 2 allocates the id with
`aiwf add adr` / `aiwf add decision` carrying only `--title`. G-0326 made adr and
decision born-complete, so `aiwf add` refuses a body whose load-bearing sections
are empty — which is every body that command would write.

Expected: the command step 2 names allocates the id. Measured 2026-09-14 against
a scratch consumer repo initialized by a binary built from `main` at `df0dd2c5f`
(`D-0001` below is that scratch repo's first id, not this tree's):

```
$ aiwf add decision --title "A decision recorded per the ritual"; echo "exit=$?"
aiwf add: D-0001: empty load-bearing body section(s) `## Question`, `## Decision`,
`## Reasoning`; a decision is referenceable the instant this commit lands, so its
body must carry meaning at creation — pass --body "..." or --body-file <path> with
real prose, or --force --reason "..." to create anyway (aiwf check will still flag
it at error severity and the pre-push hook will still block until it's filled in)
exit=2

$ ls work/decisions/ | wc -l
0
```

`aiwf add adr --title` refuses the same way, naming `## Context`, `## Decision`
and `## Consequences`. Nothing is written and no commit lands, so the ritual stops
at its first mechanical step.

The refusal names `--body-file`, but the skill binds that flag to `aiwf edit-body`
at every mention — first at line 96 inside step 5, and again through step 7. No
`aiwf add` invocation anywhere in the file carries a body flag, and `--body` does
not appear at all.

Four steps rest on the allocation succeeding and are now false with it: step 2
("aiwf creates the file with the minimal body skeleton"), step 3 ("the minimal
skeleton `aiwf add` just wrote"), step 5 (`--relates-to` at allocation "keeps step
7 a body-only bless"), and step 7 ("The `aiwf add` already produced one commit").
So do the `description:` frontmatter, which states the same three-beat sequence,
and the Constraints bullet at the end.

## Why it matters

A reader following the ritual hits a refusal at its first command. The recovery
the refusal offers first — `--body-file` — is the right one, and the skill gives
no instruction for using it at that point.

The wrong recovery is the one the ritual itself argues for. `--force --reason`
creates the entity carrying exactly the empty-sectioned skeleton that step 2
promises and step 3 then operates on, so it is the only route to the state the
next step assumes. The refusal warns that `aiwf check` will flag it and the
pre-push hook will block, but a reader who has just been told the next step fills
that body reads the warning as transient. The trap is the ritual disagreeing with
the warning, not the warning being absent.

The record is misdescribed too. The skill says the `aiwf add` scaffold and a later
`aiwf edit-body` body fill are two commits that `aiwf history` shows. For an adr
or a decision the body-fill commit never occurs — the body lands in the create
commit — so a reader looking for the second commit the skill names will not find
one.

## Related

- G-0326 is the change that made these kinds born-complete. Its fix updated the
  `aiwf-add` verb skill and did not reach this ritual.
- G-0678 established the corrected shape for gaps, whose ritual allocates with
  `--body-file` in one commit.
