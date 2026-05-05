---
id: G-030
title: '`git log --grep` false-positives leak prose-mention commits into Recent activity / `aiwf history`'
status: addressed
addressed_by_commit:
  - 7141f2a
---

`aiwf status` (Recent activity table) and `aiwf history <id>` both pre-filter `git log` with `--grep "^aiwf-verb: "` (or the anchored entity variant). The grep matches any line in the commit message that starts with the literal string — including **wrapped prose paragraphs** in hand-authored commit bodies that quote trailer keys. Real example from this repo: commit `18a00e6` ("docs(aiwf): I2.5 + I3 planning sweep") has the wrapped prose

```
…fold the audit-trail manual-commit gap (no
aiwf-verb: trailers) into I2.5 as steps 5b…
```

The second line happens to start with `aiwf-verb:` because of the line-wrap. The grep matches; the record lands in the candidate set; the parsed-trailer columns (`%(trailers:key=aiwf-verb,…)`) correctly find no structured trailer (Git's trailer parser has stricter rules than the naïve grep). Result: a row in the output table with the expected date and subject but **empty Actor and Verb columns** — visually noise, semantically wrong (the framework's "trailered commit" set was contaminated with prose mentions).

Caught while auditing this repo's `STATUS.md` after v0.2.1 shipped: the kernel repo's "Recent activity" had 5 false-positive rows, every one a docs commit whose body referenced trailer keys in prose.

**Resolution path:**

1. *Post-filter on parsed trailers, not just grep.* Both `readRecentActivity` (`status_cmd.go`) and `readHistory` (`admin_cmd.go`) already extract trailer columns via `%(trailers:key=…,valueonly=true,…)`; the fix is to discard records where the trailer column is empty (Git's trailer parser found no actual trailer for the key the caller cares about). The grep stays as an I/O-narrowing pre-filter; correctness is gated on the parsed columns. Two-line change per caller.
2. *Pin the regression.* Add a fixture commit in tests whose body wraps a sentence such that a line starts with `aiwf-verb:`, assert it does **not** appear in `aiwf status` recent activity or in `aiwf history`.
3. *Audit other `--grep` callers* (`provenance.go`, `provenance_check.go`, `scopes.go`, `show_scopes.go`, `admin_cmd.go` `loadAuthorizedScopes`). Each of those parses the trailer columns inside its loop and acts only on parsed-trailer presence, so a prose-line false-positive produces an empty trailer set that no rule branches on — structurally safe. Confirmed by inspection; documented in the fix commit so future readers don't re-litigate.

Severity: **Medium**. Doesn't affect correctness in any of the standing rules (provenance findings, scope FSM) — those iterate parsed trailers and ignore empty records — but corrupts the user-facing read views (`aiwf status` Recent activity, `aiwf history`) that are the daily surface.

---

<a id="g31"></a>
