---
id: G-0297
title: 'wf-tdd-cycle/wf-review-code honesty: coverage, --force, idempotent'
status: addressed
addressed_by:
    - M-0199
---
## Problem

Honesty / correctness defects in two `wf-*` rituals. (The `wf-doc-lint`
reconciliation is owned by G-0294, which already covers the wf-doc-lint gitleaks
advice and is being expanded to also cover the four-vs-five count, the
anti-pattern scoping, and the repo-wide scope.)

- **`wf-tdd-cycle` + `wf-review-code` overstate branch coverage as mechanical.**
  Both call the branch-coverage audit a "hard rule / blocking," but the audit is
  an **agent-performed manual branch-walk**; the repo's only mechanical coverage
  gate (G-0067) is **statement**-level. An LLM can read "hard rule" as
  tool-enforced at branch granularity when nothing enforces it there.
- **`wf-tdd-cycle` steers the implementing agent toward `--force`.** Its RECORD
  step suggests `--force --reason` to record `met` ahead of `done`. But `--force`
  is sovereign / **human-only** (the kernel rejects a non-human `--force` actor),
  and recording `met` ahead of `done` under `tdd: required` bypasses the very
  audit that gives the phase ladder meaning (see G-0293).
- **`wf-tdd-cycle` misuses "idempotent."** The RED step says re-running is
  "idempotent and the FSM will refuse `red -> red`." If the FSM *refuses* it,
  re-running errors — the opposite of idempotent.

## Decision

- `wf-tdd-cycle` / `wf-review-code`: add a clause noting the branch-coverage audit
  is agent-performed; where the project's mechanical gate is statement-level, the
  manual walk is what supplies branch-level assurance.
- `wf-tdd-cycle`: reword the `--force` escape hatch as a human-only sovereign act
  (the agent surfaces the need; the human runs it); note it bypasses the TDD
  audit (cross-ref G-0293). Keep the hatch, don't remove it.
- `wf-tdd-cycle`: "idempotent" -> "redundant — the FSM refuses `red -> red`, so
  skip this step when the AC was already seeded at `red`."

## Scope

`wf-tdd-cycle`, `wf-review-code`. (wf-doc-lint -> G-0294.)
