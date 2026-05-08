---
id: G-031
title: Squash-merge from the GitHub UI defeats the trailer-survival contract
status: addressed
addressed_by_commit:
  - ad1175c
---

When a PR is squash-merged via GitHub's UI (the default merge strategy for many repos), the resulting commit on the integration branch carries a synthesized message — typically the PR title plus the body, sometimes a list of squashed commit subjects. **Trailers from individual commits are not preserved**. A feature branch with five well-formed `aiwf <verb>` commits squash-merged via the UI lands one trailer-less commit on `main`; every entity transition from those five commits is invisible to `aiwf history <id>` against the merged tree (the only commits that carried `aiwf-verb:` trailers no longer reach HEAD via first-parent).

This breaks the framework's central correctness story ("git log is the audit log") on the most common GitHub merge strategy. Surfaced during the G24 follow-up audit (after issue #5 + G30 closed): merge surfaces had not been re-walked end-to-end since I2.5, and squash-merge had no detection.

**Resolution path:**

A subcode under the existing `provenance-untrailered-entity-commit` finding flags the case explicitly so the operator gets a more specific hint:

- *Detection.* `RunUntrailedAudit` matches the commit's subject against the GitHub squash-merge regex `\s\(#\d+\)$` (PR title followed by ` (#NNN)`). When a flagged untrailered commit fits the pattern, the finding is emitted with subcode `squash-merge`. The default `(provenance-untrailered-entity-commit)` finding still applies; the subcode just specializes the hint.
- *Hint.* The hint table entry for the subcode names the merge-strategy gotcha and the recovery path: switch the repo to rebase-merge or `--no-ff` merge for branches that touch entity files, or run `aiwf <verb> <id> --audit-only --reason "..."` per entity touched.
- *Skill text.* `aiwf-check` SKILL.md gains a row for the subcode pointing at the same recovery path.
- *Pinned by* `TestRunUntrailedAudit_SquashMergeSubcode`: a fixture commit with subject `… (#42)` touching an entity file emits the finding with subcode `squash-merge`; subjects without that suffix produce the bare code.

What this fix does NOT do: recover trailers from a squash-merged commit's source SHAs (would require walking GitHub's `refs/pull/<N>/head` references — out of scope for the kernel). The detection surfaces the gap; the audit-only recovery path is the operator's lever to backfill what the squash-merge dropped.

**Limitations:**

- Detection is opportunistic: it fires only while the squash commit is in the audit's `@{u}..HEAD` (or `--since`) range. After the operator pushes/pulls, the squash commit becomes `@{u}` and is no longer scanned. The companion README entry under "Known limitations" frames squash-merge as something the operator should re-audit on the integration-branch *before* pushing further.
- The regex matches the GitHub default. Custom squash-commit-message templates that drop the `(#NNN)` suffix won't trigger the subcode (the bare warning still fires; only the hint specializes).

Severity: **High**. Real audit-trail hole on the dominant merge strategy; the framework's central promise depended on a pattern most consumers don't follow by default. Fixed by detection + hint + skill update, plus a known-limitation note in the README.

---

<a id="g32"></a>
