---
id: G-0739
title: AC verbs rewrite every same-numbered AC heading in a milestone body
status: open
discovered_in: M-0361
---
## What's missing

Three verbs that write an acceptance criterion's heading rewrite every `### AC-N — …` heading in the milestone body that carries the same id, not only the criterion's own heading under `## Acceptance criteria`: `aiwf retitle M-NNNN/AC-N` and `aiwf rename M-NNNN/AC-N` rename it, and `aiwf add ac` with a body takes it over when the new criterion's id is already used by a heading elsewhere. A spec that reuses the heading shape, such as a `## Validation` subsection per criterion, is silently changed. The rewrites share one cause in `internal/verb/ac.go`: `rewriteACHeading` (reached from `retitleAC` in `retitle.go` and `renameAC`) and `upsertACSection` apply `acHeadingLinePattern` over the whole body and replace every line whose id matches.

Reproduced in scratch repositories with a binary built from a clone at `ae127e08f` (`aiwf version` prints `v0.40.1-0.20261018163152-ae127e08fa76`), on a milestone with one criterion and a body ending in

```
## Validation

### AC-1 — the record
```

committed with `aiwf edit-body`. Before `aiwf retitle M-0001/AC-1 "New title"`, `grep -n '^### AC'` printed `17:### AC-1 — Old title` and `22:### AC-1 — the record`; after it, `17:### AC-1 — New title` and `22:### AC-1 — New title`. `aiwf rename M-0001/AC-1 "Third title"` turns line 22 into `### AC-1 — Third title` the same way. With the Validation subsection headed `### AC-2 — the record` instead, `aiwf add ac M-0001 --title "Second" --body-file <file>` left `### AC-2 — Second` where the subsection was, its body text placed above the subsection's own, and no second heading under `## Acceptance criteria`. Expected in each case: the Validation heading unchanged.

## Why it matters

Changing a criterion corrupts other sections of its spec, and nothing reports it: the commit carries the verb's trailers, so it reads as a title change or a new criterion, and after each of the three `aiwf check` reports nothing about the headings. (Before the `add ac`, it reports the `### AC-2` subsection as `acs-body-coherence`, a heading with no criterion; the verb's rewrite clears that finding.) Ten milestone specs carry `### AC-N` headings under `## Validation`, M-0334, M-0338 and M-0361 among them (`for f in $(git ls-files 'work/epics/*M-*.md'); do awk '/^## /{v=($0=="## Validation")} v && /^### AC-[0-9]+/{found=1} END{exit !found}' "$f" && echo "$f"; done | wc -l` at `ae127e08f`), so retitling or renaming one of their criteria renames the subsections carrying its id, and adding a criterion whose id a subsection already uses moves that criterion's body into Validation.
