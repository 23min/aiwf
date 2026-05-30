---
id: G-0186
title: promote --by-commit records unresolvable commit SHAs with no validation
status: addressed
addressed_by_commit:
    - c9689ae1
---
## Problem

`aiwf promote --by-commit <sha>` records the value into a gap's `addressed_by_commit` frontmatter **verbatim, with no check that the SHA resolves to a real commit**. The sibling flag `--by` (which writes `addressed_by`, carrying entity ids) *does* validate existence and rejects unknown ids; `--by-commit` does not. `aiwf check` does not close the gap either: `gap-addressed-has-resolver` only asserts the field is non-empty, never that the SHA is resolvable. A well-formed-but-fake 7-hex-character string passes both gates silently.

Code references:

- `internal/verb/promote.go` (~369-376): the `--by-commit` branch — comment states the SHAs are "recorded verbatim without entity-existence checks"; value is only `splitCSV`'d and trimmed.
- `internal/verb/promote.go` (~352-367): contrast — the `--by` branch validates each id via `tree.ByID` and errors on unknown.
- `internal/verb/promote.go` (~386-388): direct assignment `target.AddressedByCommit = p.addressedByCommit` (replace, not append).
- `internal/check/check.go` (~1136): `gap-addressed-has-resolver` checks only `len(AddressedBy)==0 && len(AddressedByCommit)==0`.

## How it surfaced

During the wrap of G-0185, an orchestration placeholder SHA `8f3c2a1` (chosen before the real fix commit existed, then executed by mistake) was recorded as G-0185's `addressed_by_commit`. `8f3c2a1` resolves to no object in the repo (`git cat-file -e 8f3c2a1` → "Not a valid object name"). `aiwf check` stayed green throughout — the field was non-empty, so the resolver rule was satisfied. The corruption was caught only by a manual `git cat-file -e` and corrected via a sovereign `--force` re-stamp (commit `6c1c63ec`, resolver now `f7fd1f99`).

## Impact

`addressed_by_commit` exists to answer "which commit fixed this gap" — for `aiwf history`, audit, and future traceability. A value that resolves to nothing is worse than an empty field: it reads as authoritative while pointing at nothing, and nothing mechanical catches it. This is a correctness hole in a provenance surface, exactly the class "framework correctness must not depend on the operator getting it right." Severity is low (cosmetic/traceability; it does not block validation), but the fix is cheap and the chokepoint is missing.

## Resolution (sketch)

1. **Write-time validation in the promote verb (primary chokepoint).** For each `--by-commit` SHA, verify it resolves against the repo — e.g. `git rev-parse --verify <sha>^{commit}` or `git cat-file -e <sha>^{commit}` — and reject unknown SHAs with a clear error, mirroring how `--by` validates entity ids. At promote time the operator is in the repo where the fix commit lives, so resolvability is the right precondition. Must accept abbreviated SHAs (the legitimate value `f7fd1f99` is a short SHA; `git rev-parse --verify` handles abbreviations).

2. **Optional check-time rule (secondary, with care).** A `check` rule that walks `addressed_by_commit` and flags unresolvable entries would catch drift, but carries a real tension: a commit may legitimately be absent in a shallow clone, before the fixing branch merges, or in a cross-repo reference. So a check-time rule should be advisory (warning) and/or guarded against false positives in those cases — not a hard error. Write-time validation is the cleaner primary fix; decide on the check rule separately.

## Done when

- `aiwf promote G-0185 addressed --by-commit deadbeef` (a non-resolving but well-formed SHA) is rejected with an error, by a test under `internal/verb/` or `internal/cli/` that drives the verb and asserts the rejection.
- A valid short SHA (e.g. an actual `HEAD` abbreviation in the test repo) is accepted.
- If a check rule is added: a fixture gap with an unresolvable `addressed_by_commit` yields the new finding; a resolvable one yields none; the shallow-clone/absent-commit tension is documented in the rule's comment.

## Refs

- Discovered while wrapping **G-0185** (whose resolver was the corrupted field). Corrected by sovereign re-stamp `6c1c63ec`.
- Related: the `--by` entity-id validation precedent in the same verb; the `gap-addressed-has-resolver` finding (`check.go`).
