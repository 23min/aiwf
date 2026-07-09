---
id: G-0396
title: addressed_by_commit SHA is merge-fragile; derive closure from git history
status: open
discovered_in: E-0063
---
## What's missing

`addressed_by_commit` stores a raw git SHA in a gap's frontmatter, captured at
`aiwf promote <id> addressed --by-commit <sha>` time. That SHA is a denormalized
copy of a fact git already owns — "this gap's closure landed in this commit" —
and it is not stable across the operation that makes the closure real. When the
closing work rides a patch/epic branch that is reconciled (a `git merge main`
into the branch, a rebase) before the `--no-ff` merge to trunk, the commit that
actually lands on trunk carries a **different** SHA than the one recorded. The
recorded SHA is orphaned: it resolves to a real commit reachable from no branch
and not from trunk, while the fix itself shipped under a new hash.

Measured on this repo: 5 gaps (G-0020, G-0069, G-0076, G-0119, G-0226) carry
`addressed_by_commit` SHAs that are not reachable from `refs/remotes/origin/main`,
yet every one of those fixes is demonstrably on trunk under a different commit
(same commit subject, feature present in trunk source). The recorded SHA is a
pre-reconcile branch hash; the closure is real.

This is a single-source-of-truth violation (design-decisions C1): the closing
commit's identity lives in git, and frontmatter caches a fragile version of it
with no invalidation rule, so it silently rots.

## Why it matters

The immediate consequence is that any check policing "is the recorded SHA on
trunk" (the direction gap G-0357 proposed) is **unsound**: SHA-reachability
cannot distinguish a genuinely-lost fix (abandoned branch, fix never merged —
the real problem) from a benign rewritten-SHA (fix merged under a new hash — a
bookkeeping artifact). Both present identically as "recorded SHA exists but is
not on trunk," and on this repo the benign case is 100% of what such a check
finds. Building it would ship persistent warning-noise on legitimately-closed
gaps and erode trust in `aiwf check`. G-0357 should not be built as a check;
this gap records why and proposes the data-model fix instead.

The verb-time guard (G-0355) is not affected and remains the load-bearing
guarantee: it anchors on HEAD at promote time, which is correct at that moment.
The fragility is purely in the *stored* value outliving the SHA it captured.

Scope note: `addressed_by` (the entity-id resolver — "closed by milestone
`M-NNNN`") is **not** fragile, because ids survive rebase. Only the raw-SHA
resolver rots. The change is narrow to `addressed_by_commit`.

## Direction (proposed — not prescribed)

Stop storing the closing commit as a frontmatter SHA; derive it from git,
merge-stably, at read time. The closing act is already a trailered commit — the
`aiwf promote <id> addressed` commit carries `aiwf-verb: promote`,
`aiwf-entity: <id>`, and `aiwf-to: addressed`, and `aiwf history` already reads
exactly these trailers. Queried *from trunk*, it resolves to whatever version of
that commit landed on trunk, regardless of how the SHA was rewritten getting
there — merge-stable by construction. This aligns with design principle #4
(`aiwf history` reads `git log`; no separate event log) and C1 (do not cache
what git owns).

- **`aiwf show <id>` derives "closed by" from the trailered closure commit
  reachable from trunk**, rather than reading a frozen frontmatter SHA.
- **The distinct-fix-commit case** (a gap whose *fix* is a code commit separate
  from the bookkeeping promote commit) is served, if wanted, by a trailer on the
  fix commit (`aiwf-closes: <id>`), derived the same way — not by a stored SHA.
  For most gaps the promote commit *is* the closure record, so this may be
  optional.
- **Deprecate / migrate `addressed_by_commit`.** Decide whether to drop the
  field, keep it as an unverified human-note hint, or migrate existing records
  to the derived form. Existing orphaned records (the 5 above) resolve
  automatically once "closed by" is derived from trunk history.
- **G-0357 is dissolved, not fixed:** with no stored SHA, there is nothing to
  drift off trunk, so no check rule is needed. Removing the cache removes the
  problem and the check.

This warrants recording as an ADR (it changes what frontmatter is authoritative
for) plus a small loader/`show` implementation and a migration path — not a
`wf-patch`.

## Open questions for review

- Does deriving from the trailered promote commit capture **every** path a gap
  reaches `addressed` (normal promote, `--by-commit`, `--force` with no
  resolver, a raw-git-bypass closure, reopen-then-re-address)?
- Does "derive from trunk" behave correctly for the **genuine** lost-fix case
  (branch never merged) — reporting no trunk closure rather than a false one?
- Read cost: a git-log query per `aiwf show` / per closure resolution — is that
  acceptable, and does it stay off the every-push hot path (unlike G-0372)?
- Migration and backward-compat: 205 existing `addressed_by_commit` records,
  archived gaps, and any `--format json` consumer relying on the field.
- Shallow clones / detached history where trunk isn't fully present.

## Provenance

Surfaced while auditing gap G-0357's proposed check against this repo: the audit
flagged 6 off-trunk SHAs across 5 gaps, all confirmed false positives (fixes
shipped on trunk under rewritten hashes). Descends from the G-0355 verb-time
guard and supersedes G-0357's check-based direction.
