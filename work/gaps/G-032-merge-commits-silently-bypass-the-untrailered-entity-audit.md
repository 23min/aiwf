---
id: G-032
title: Merge commits silently bypass the untrailered-entity audit
status: addressed
---

`readUntrailedCommits` ran `git log --name-only`, which by default shows **no file list for merge commits** (true merge commits show diff content only with `-m` or `--cc`). A merge commit that absorbs entity-file changes from a feature branch produces an empty `Paths` slice, and `RunUntrailedAudit` skips it.

Concrete: trunk has G-001 at `open`. A feature branch makes a manual commit changing G-001 to `wontfix`. Operator merges feature → trunk via `git merge --no-ff feature`. The merge commit's message lacks aiwf trailers; its `--name-only` is empty. The audit pass on trunk says nothing — even though the audit-trail hole is real (no `aiwf-verb:` trailer ever recorded G-001's transition).

This was salvageable on the *feature* branch (the original untrailered commit was flagged before merge), but only if the operator ran `aiwf check` between the manual commit and the merge. Feature → merge → push without an interim check left the warning silent on the merge commit itself. A merge commit that itself made changes (conflict resolution touching entity files) was also silent.

**Resolution path:**

`readUntrailedCommits` now invokes `git log -m --first-parent`. Combined effects:

- *`--first-parent`*: walks first-parent ancestry of the integration branch only. Feature-branch commits are NOT shown (correctly — they're the feature branch's own warning scope). Merge commits ARE shown.
- *`-m`*: causes merge commits to show diffs against their first parent — i.e., the changes the merge introduced into the integration branch. Entity-file paths flow into the audit pass.

Together: a merge that brings in feature-branch entity-file changes surfaces those file paths. Per-(commit, entity) findings (post-G30) fire on each touched entity. Audit-only on the integration branch clears them via the same per-entity suppression path.

**Limitations:**

- Octopus merges (3+ parents) are rare and produce one record per non-first-parent diff under `-m`; the existing per-entity dedupe inside the loop handles the common case (an octopus that brings the same entity from multiple branches collapses to one finding per entity at the loop level).
- A merge commit that introduces NO new entity-file changes (the integration branch already had everything) produces an empty path list and stays silent — correct behavior.

Pinned by `TestRunUntrailedAudit_MergeCommitSurface` (in `cmd/aiwf/show_scopes_unit_test.go` next to the existing `--since` tests): a fixture with a merge commit whose second parent introduced an entity-file change is flagged.

Severity: **Medium**. Doesn't compromise correctness — `aiwf check` still fires on the original commit on the feature branch — but loses signal at the integration-branch boundary, which is exactly when the operator's last chance to repair lives.

---

<a id="g33"></a>
