---
id: G-0187
title: Statusline rendering has no end-to-end behavioral test
status: open
discovered_in: M-0153
---
## What's missing

`.claude/statusline.sh` is tested via three content assertions in
`internal/policies/statusline_content_test.go` (M-0153's ACs): each pins the
structural form of a fix — the `tail -r … || tac` fallback chain, the
`read -r ahead behind` parse, and the `GIT_OPTIONAL_LOCKS=0` export — by
regex over the script source. None of them runs the script.

A behavioral test would:

- Write a known-shape transcript fixture to `t.TempDir`.
- Stream a stub stdin JSON pointing at the fixture through
  `exec.Command("bash", scriptPath)`.
- Parse the rendered output (strip ANSI); assert the token segment shows the
  expected non-zero count.
- Assert similar shape invariants for the sync segment (synthetic upstream
  via a temp git repo).

## Why it matters

Mid-implementation of M-0153, the fallback chain was originally
`$(tail -r … || tac … | jq …)` — but `|` binds tighter than `||`, so a
successful `tail -r` (macOS) would have routed its raw output directly to
the command substitution, bypassing `jq` entirely and yielding the entire
reversed transcript as a "token count". The structural content assertion
**passed against that broken form** because both `tail -r` and `|| tac`
were present on the line; only the manual smoke render against a fixture
stdin caught it before the fix landed.

The shipped form uses a brace group — `{ tail -r … || tac …; } | jq …` —
which makes the precedence correct. But the test set as it stands does not
guard against a future reflow that drops the brace group. Other classes of
behavioral regression (jq filter change, ANSI sequence drift, branch-badge
miswiring) would be similarly silent under content-only assertions.

## Direction

Add `TestStatusline_Behavior_TokenSegment` (and siblings as the segments
grow) under `internal/policies/` — or a new `internal/statusline/`
subpackage if behavioral tests accumulate. Use `exec.Command("bash",
scriptPath)` with a synthesized stdin JSON pointing at a `t.TempDir`
transcript; assert against the post-ANSI-strip output.

Scope: 1–2 days; bounded by harness setup (ANSI stripping, env hygiene,
fixture git repo for the sync segment). Best landed as a follow-up after
M-0157 (the doctor block) so the same harness can also exercise the doctor
report's statusline-aware paths end-to-end.
