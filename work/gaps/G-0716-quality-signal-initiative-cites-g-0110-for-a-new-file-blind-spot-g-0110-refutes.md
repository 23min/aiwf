---
id: G-0716
title: quality-signal initiative cites G-0110 for a new-file blind spot G-0110 refutes
status: open
---
## What's missing

`docs/initiatives/quality-signal-and-cadence.md` describes G-0110 by its original
premise, which G-0110 itself now records as false. Two places state it, measured
in the main checkout at `5aae9db41`:

```
$ grep -n -E "excludes new files|new-file blind spot" docs/initiatives/quality-signal-and-cadence.md
223:  — mutation testing's `--diff <ref>` filter excludes new files entirely. The
454:   — mutation testing's new-file blind spot.
```

The first continues on line 224: "The blind spot is precisely on newly-written
code". Both sit under links to G-0110, at lines 222 and 453.

G-0110 is `addressed`, archived, and titled "mutate-diff mutates every line of each
changed package, not only changed lines". Its body states the opposite of line 223:

```
$ sed -n 45,47p work/gaps/archive/G-0110-mutate-diff-mutates-every-line-of-each-changed-package-not-only-changed-lines.md
- An untracked file is skipped either way: `git diff` does not list it. A staged or
  committed new file is mutated from the module root, so new files as such are not
  excluded.
```

The measurement behind that sentence is G-0110's own gremlins dry run, printed in
the same file at lines 24–39: run from the module root, the staged new file's
mutant reports `RUNNABLE` and the untracked one `SKIPPED`.
`docs/audits/gap-truth-audit-evidence.md` §G-0110 records an independent dry run,
on a different fixture, that agrees a new file is mutated from the module root.
Neither dry run was re-run here.

A fix lands in `docs/initiatives/quality-signal-and-cadence.md`, at the two places
the grep above prints.

## Why it matters

A reader who follows either G-0110 link for evidence of a new-file blind spot lands
on a gap saying new files are mutated: the initiative's G-0110 example rests on a
citation that contradicts it.

The initiative is a staging point for promotion into epics and gaps. Work scoped
from it inherits a blind spot the recorded measurements say does not exist, and is
aimed at the wrong cause.
