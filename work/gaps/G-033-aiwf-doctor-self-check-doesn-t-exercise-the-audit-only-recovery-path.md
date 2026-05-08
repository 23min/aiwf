---
id: G-033
title: '`aiwf doctor --self-check` doesn''t exercise the audit-only recovery path'
status: addressed
addressed_by_commit:
  - ad1175c
---

The G24 recovery story has three load-bearing pieces (manual commit detection, `--audit-only` empty-diff repair, lock-contention diagnostics in `Apply`). Self-check covered init / add / promote / cancel / render / etc., but did not drive the recovery loop end-to-end. A regression in the suppression rule (issue #5's all-or-nothing was such a regression) wouldn't be caught by CI's self-check stage; it'd ship until a user noticed.

**Resolution path:**

New self-check step (after the existing `cancel` step) that:

1. Synthesizes a manual untrailered commit that touches an entity file.
2. Runs `aiwf check`; asserts a `provenance-untrailered-entity-commit` finding with the expected entity-id is present.
3. Runs `aiwf cancel <id> --audit-only --reason "self-check"`.
4. Runs `aiwf check`; asserts the previously-emitted finding for that entity is gone.

The step also exercises the per-entity suppression that issue #5 fixed — a regression in that path would fail the assertion at step 4.

Severity: **Medium**. CI safety-net pattern, same shape as G9's "self-check covers every verb" rule. Small fix; pinned the recovery path that until now was only covered by unit tests in `internal/check/provenance_test.go`.

---

<a id="g34"></a>
