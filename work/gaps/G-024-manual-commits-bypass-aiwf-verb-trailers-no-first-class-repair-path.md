---
id: G-024
title: Manual commits bypass `aiwf-verb:` trailers; no first-class repair path
status: addressed
---

Resolved across I2.5 steps 5b, 5c, and 7b: `aiwf cancel <id> --audit-only --reason "..."` and `aiwf promote <id> <status> --audit-only --reason "..."` (commit `bc4183e`) record properly-trailered empty-diff commits on entities already at the named state; `Apply` classifies `index.lock` failures and surfaces the holder PID via `lsof` with no silent retries (commit `6cc0648`); a `provenance-untrailered-entity-commit` warning fires on every push for commits ahead of `@{u}` that touch entity files without `aiwf-verb:` (commit `0e44ad6`); the warning clears once the audit-only commit lands (commit `be2ea27`). Cross-cutting integration test in `9c1b010`. The "git log is the audit log" promise now has both a surface-the-gap signal and a first-class recovery verb.

When a mutating verb (`aiwf cancel`, `aiwf promote`, …) fails partway through and the operator finishes the work with a plain `git commit`, the resulting commit lands without the structured trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`). The entity reaches its correct state — `aiwf check` is clean — but `aiwf history <id>` and `aiwf status` (both filter `git log --grep "^aiwf-verb: "`) report no event for the change. The audit trail goes silent for events that did happen.

Observed concretely: in a working session, three gap closures (G-021, G-030, G-031 in a separate consumer tree) were committed manually after `aiwf cancel` failed on `.git/index.lock` contention each time. The frontmatter reflects `wontfix`, but `aiwf history` returns "no history" for all three.

There is no clean recovery verb. `aiwf cancel <id> --force --reason "..."` looks like the natural backfill, but `Cancel` at `internal/verb/promote.go:107-109` still errors `"already at target"` even under `--force` (the function-doc comment at lines 91-92 makes this explicit: the guard is intentional because there is no diff to write). The only currently-available repair is an empty hand-crafted commit with the right trailers — i.e., the same kind of manual commit that produced the problem.

**Probable cause of the lock contention.** `aiwf cancel` takes its own lock at `.git/aiwf.lock` (separate from git's `.git/index.lock`), so the two don't collide directly. Inside the verb, `verb.Apply` runs `git mv` → `git add` → `git commit` as subprocesses; the pre-commit hook then runs `aiwf status --format=md` (read-only `git log`) plus `git add STATUS.md`. None of that should contend with itself. The likely culprit is an external process — VS Code's git extension, a file-watcher, or a stale `.git/index.lock` from a prior crash — holding `index.lock` just long enough for the in-flight `git commit` to fail. Capturing the actual `index.lock` error (stderr from a failed `aiwf cancel`) and `lsof .git/index.lock` is the diagnostic next step; the lock-contention root-cause is its own thread, not in scope here.

**Failure modes and consequences.**

1. *Audit-trail gap.* `aiwf history` / `aiwf status` cannot see the change. Downstream readers conclude "no recent activity" when there was; decisions made on those outputs are reading from incomplete data.
2. *Provenance gap.* "Who, when, why" is recoverable only by re-reading the manual commit's prose, which doesn't follow the trailer schema and isn't queryable.
3. *No first-class repair.* The framework provides no verb to backfill an audit-only event. The recovery path that exists is to make the same kind of manual commit that created the problem.
4. *Silent invariant violation.* `aiwf check` passes because frontmatter is consistent. The framework's core promise — "git log is the audit log" (kernel decisions §3 / §4) — is broken without raising any alarm.
5. *Recurrence risk.* If the contention is environmental (concurrent IDE, watcher), it will recur; the framework treats every commit failure as fatal and does not retry, log, or surface the offending process.

**Resolution path.** Folded into I2.5 (`provenance-model-plan.md` steps 5b, 5c, 7b). Three-part fix:

1. *Audit-only recovery mode* — `aiwf cancel <id> --audit-only --reason "..."` and `aiwf promote <id> <status> --audit-only --reason "..."`. Records a properly-trailered, empty-diff commit on an entity already at its target state. Plan step 5b.
2. *Diagnostic instrumentation in `Apply`* — classify lock-contention failures, surface the holder PID via `lsof`, point the operator at the audit-only recovery path. No silent retries. Plan step 5c.
3. *Pre-push trailer audit* — new `provenance-untrailered-entity-commit` warning in `aiwf check` for commits ahead of `@{u}` that touch entity files without `aiwf-verb:`. Plan step 7b.

Severity: **High**. The framework's central correctness story (git log is the audit log) had an unsignalled hole; the I2.5 fix surfaces the gap (warning) and provides the recovery verb (`--audit-only`).

---

<a id="g27"></a>
