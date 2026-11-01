---
id: G-0759
title: Verb-time body-prose-id findings give a body line against the file's path
status: open
---
## What's missing

For one unresolved id in an entity body, `aiwf check` reports its line in the
entity's file, while the verbs that scan a body before writing it report its line
in the body, after the file's path.
`ScanBodyProseID` (`internal/check/body_prose_id.go`) returns a line "within
body", and its comment says "callers that want file-relative Line (the
bodyProseID tree-walk rule) add the body's start-of-file line offset
themselves". The tree-walk rule does. The verb-time callers do not: they report
the finding against the entity's file path with the body-relative line —
`internal/verb/editbody.go` (both modes), `internal/verb/add.go`,
`internal/verb/import.go` and `internal/verb/reallocate.go`.

Reproduced 2026-11-01 with aiwf built from `main` at `c06ffdaf9`
(`vcs.modified=false`), git 2.51.1, Linux, with the hooks `aiwf init` installs, in a fresh repository holding a gap `G-0001` created by `aiwf add gap --title
"A gap"` from the body `## What's missing\n\nM.\n\n## Why it matters\n\nW.\n`, with
a line naming an unallocated id appended to it in the working copy; `../gap2.md`
holds `## What's missing\n\nSee G-0999.\n\n## Why it matters\n\nW.\n`:

```
$ printf '\nSee G-0999 for more.\n' >> work/gaps/G-0001-*.md
exit 0
$ grep -n 'G-0999' work/gaps/G-0001-*.md
14:See G-0999 for more.
$ grep -c '' work/gaps/G-0001-*.md
14
$ aiwf check
work/gaps/G-0001-a-gap.md:14: error body-prose-id/unresolved: G-0001 body prose contains unknown id "G-0999" (no entity allocated at this id) — hint: the body prose references a well-formed id that resolves to no entity; fix it with `aiwf edit-body <id>` — check the spelling, or wrap in backticks if the prose is discussing a hypothetical id shape rather than a real reference
provenance-untrailered-scope-undefined (warning) × 1 — no upstream configured and no --since <ref>; provenance audit, body-section gate and instruction-file fence skipped

2 findings (1 errors, 1 warnings)
run `aiwf check --verbose` for each warning's location and remediation hint
exit 1
$ aiwf edit-body G-0001
work/gaps/G-0001-a-gap.md:9: error body-prose-id/unresolved: G-0001 body prose contains unknown id "G-0999" (no entity allocated at this id)

1 findings (1 errors, 0 warnings)
exit 1
$ sed -n '9p;14p' work/gaps/G-0001-*.md

See G-0999 for more.
$ grep -n 'G-0999' ../gap2.md
3:See G-0999.
$ aiwf add gap --title 'Another gap' --body-file ../gap2.md
work/gaps/G-0002-another-gap.md:3: error body-prose-id/unresolved: G-0002 body prose contains unknown id "G-0999" (no entity allocated at this id)

1 findings (1 errors, 0 warnings)
exit 1
```

`aiwf check` points at line 14, where the token is; `aiwf edit-body` points at
line 9 of the same file, which is blank. `aiwf add` reports line 3, the token's
line in the body file, against the path of the entity file it would have
written, and did not write. `import` and `reallocate` do the same, in the same
environment, `G-0001` and `G-0002` in the second run created by `aiwf add gap`
from the first body above:

```
$ cat ../m.yaml
version: 1
entities:
  - kind: gap
    id: G-0001
    frontmatter:
      title: Imported gap
      status: open
    body: |
      ## What's missing

      Something is wrong.

      ## Why it matters

      See G-0999 for more.
$ aiwf import ../m.yaml
work/gaps/G-0001-imported-gap.md:7: error body-prose-id/unresolved: G-0001 body prose contains unknown id "G-0999" (no entity allocated at this id)

1 findings (1 errors, 0 warnings)
exit 1
$ ls work/gaps
$ printf '\nSee G-0001 and G-0999.\n' >> work/gaps/G-0002-second-gap.md && git commit -q --no-verify -am 'plant a token'
exit 0
$ grep -n 'G-0999' work/gaps/G-0002-second-gap.md
14:See G-0001 and G-0999.
$ aiwf reallocate G-0001
work/gaps/G-0002-second-gap.md:9: error body-prose-id/unresolved: G-0002 body prose contains unknown id "G-0999" (no entity allocated at this id)

1 findings (1 errors, 0 warnings)
exit 1
```

`aiwf import` reports line 7, the token's line in the manifest's body, against a
file it did not write; `aiwf reallocate` reports line 9 for the token on line 14.

## Why it matters

An operator told `work/gaps/G-0001-a-gap.md:9` opens that line and finds
nothing there; the line to fix is further down by the lines before the body,
five here, and `aiwf check` names a different line for the same finding. An
editor or terminal that jumps to `path:line` lands on the wrong line. From
`aiwf add` and `aiwf import`, the path names a file the refusal left unwritten.
