---
id: G-0767
title: aiwf add silently drops --relates-to, --epic or --discovered-in on a wrong kind
status: open
---
## What's missing

`aiwf add` accepts `--relates-to`, `--epic` or `--discovered-in` on a kind that
does not store it, exits 0, and drops the value. `validateAddOptsForKind`
(`internal/verb/add.go`) refuses `--depends-on` on any kind but a milestone and
`--linked-adr` on any kind but a contract, the same way it refuses `--tdd`,
`--area` and `--priority` on the wrong kind, and has no such check for these
three. `applyAddOpts`, in the same file, then writes each only for the one kind
that carries it: `--epic` as a milestone's `parent`, `--discovered-in` on a gap,
`--relates-to` on a decision. The `aiwf add` help marks them "(milestone only)",
"(gap only)" and "(decision only)". Because the value is never stored, it is
never resolved either: an id that names nothing is accepted on a wrong kind and
refused on the right one.

Measured at `3c5bd7546` (git 2.51.1, Linux) in an empty directory, as a human
actor, with an `aiwf` built from that commit on `PATH`. Output from the setup
commands is left out.

```
$ git init -q -b main r && cd r
$ git config user.name t && git config user.email t@example.invalid
$ aiwf init --no-prompt && git add -A && git commit -qm init
$ aiwf add epic --title E
$ printf "## What's missing\n\nX.\n\n## Why it matters\n\nY.\n" > ../g.md
$ printf "## Question\n\nQ.\n\n## Decision\n\nD.\n\n## Reasoning\n\nR.\n" > ../d.md
$ aiwf add gap --title G --body-file ../g.md --relates-to E-0001 --epic E-0001
aiwf add gap G-0001 "G"
$ sed -n 1,5p work/gaps/G-0001-g.md
---
id: G-0001
title: G
status: open
---
$ aiwf add decision --title D --body-file ../d.md --discovered-in E-0001
aiwf add decision D-0001 "D"
$ sed -n 1,5p work/decisions/D-0001-d.md
---
id: D-0001
title: D
status: proposed
---
$ aiwf show E-0001 --format=json | jq -c .result.referenced_by
[]
$ aiwf add gap --title G2 --body-file ../g.md --relates-to NOPE
aiwf add gap G-0002 "G2"
$ aiwf add decision --title D2 --body-file ../d.md --relates-to NOPE
work/decisions/D-0002-d2.md:1: error refs-resolve/unresolved: decision field "relates_to" references unknown id "NOPE" — hint: …

1 findings (1 errors, 0 warnings)
$ aiwf add gap --title G3 --body-file ../g.md --depends-on M-0001
aiwf add: --depends-on is only valid for kind=milestone
$ aiwf add gap --title G4 --body-file ../g.md --linked-adr ADR-0001
aiwf add: --linked-adr is only valid for kind=contract
$ aiwf check
entity-body-empty (warning) × 3 — E-0001 body section `## Goal` is empty
provenance-untrailered-scope-undefined (warning) × 1 — no upstream configured and no --since <ref>; provenance audit, body-section gate and instruction-file fence skipped

4 findings (0 errors, 4 warnings)
run `aiwf check --verbose` for each warning's location and remediation hint
```

The creates of G-0001, D-0001 and G-0002 exit 0; the decision with `NOPE` exits
1; the last two creates exit 2. `aiwf check` reports nothing about the missing
links.

## Why it matters

Whoever runs the command, a human or an assistant, is told the entity was created
and takes it to be linked; nothing records the link, so `aiwf show` on the target
lists no referrer, and nothing says the flag was ignored. The author is never
prompted toward the link the kind can carry, such as `--discovered-in` for a gap,
and a mistyped or made-up id on a wrong kind goes through unchecked.

For an `ai/` actor working under a scope, a dropped flag still counts toward the
verb's scope check, which is one way a create is admitted that `aiwf check` then
reports out of scope (G-0766).

## Related

- G-0766 — the verb and `aiwf check` judge a scoped create by different links.
