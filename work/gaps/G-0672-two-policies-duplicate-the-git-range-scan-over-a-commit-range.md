---
id: G-0672
title: Two policies duplicate the git range scan over a commit range
status: open
discovered_in: M-0330
---
## What's missing

Two policies run the same git range scan and neither shares it.
`skillEditsInRange` in `internal/policies/skill_edit_provenance_backstop.go`
and `shippedDeltasInRange` in `internal/policies/changelog_completeness.go`
both build a `git log` format from the same two control-character separators,
run `git log --name-only` over `<base>..HEAD` limited to one directory, set
`cmd.Dir`, take `CombinedOutput`, and hand the result to a parser. The error
message is the same sentence in both, one word apart. The second was written
against the first: it reaches into that file for `skillEditRecSep`,
`skillEditFldSep` and `shortSkillSHA` rather than declaring its own.

Their parsers duplicate a second time. `parseSkillEditLog` and
`parseShippedDeltaLog` both split records on the separator, read a header line
into fields, take the first `aiwf-entity` trailer as the owner, and walk the
remaining lines as paths.

What differs is small and real: one asks git for the commit subject, the other
for rename detection; one excludes merges; they name different directories; and
each parser applies its own path filter.

The extraction that fits is one scanner returning, per commit in the range,
the sha, the subject, the entity trailer and the paths touched — leaving each
caller to filter those paths and build its own record.

## Why it matters

The duplication is about 15 lines of git invocation and error handling, or
about 25 if the two parsers merge as well. That is worth removing on its own
terms, and the second copy landing is the trigger this project's code-health
rule names for doing it.

It is worth recording what this duplication did **not** cause, because the
review that surfaced it argued otherwise and the claim does not survive
checking. The review held that the copy dropped the original's path filter, and
that a shared scanner would have prevented the over-broad pathspec fixed under
M-0330. Measured against both functions, the path filter is not in the scanner
at all — it is in each parser, and the two filters differ by necessity: one
keeps only files named `SKILL.md`, the other keeps every non-Go file under the
embedded trees, which hold markdown, shell, templates and agent cards. A shared
scanner would have returned a path list and left that filter exactly as much
the caller's to get wrong.

So the case for extracting is the ordinary one — two copies drift, and this
pair already differs in four ways — not that it would have prevented a specific
defect. Whoever takes this should also decide whether the parsers merge, which
is the larger half of the change and the half that needs its own review: the
older policy is landed and working, and its rename detection and
`--diff-filter=AMR` are load-bearing for a property this one does not share.
