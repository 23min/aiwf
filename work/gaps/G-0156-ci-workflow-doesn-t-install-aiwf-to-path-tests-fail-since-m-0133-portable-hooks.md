---
id: G-0156
title: CI workflow doesn't install aiwf to PATH; tests fail since M-0133 portable hooks
status: open
---
## What's missing

`.github/workflows/go.yml` doesn't install `aiwf` to PATH before invoking the `test` or `selfcheck` jobs. As a result, every integration test that:

  1. Creates a temp git repo,
  2. Runs `aiwf init` (which materializes `pre-commit`/`pre-push`/`post-commit` hooks that resolve `aiwf` via `command -v aiwf` at hook fire time — per M-0133 / G-0135), then
  3. Tries to commit through an aiwf verb (`aiwf add`, `aiwf promote`, `aiwf edit-body`, etc.)

…hits the pre-commit hook, which can't find `aiwf` on PATH, exits non-zero (per the M-0133 AC-1 design: validation hooks must not silently skip), and the `git commit` fails. The verb's rollback then can't restore HEAD either, since the fresh repo has no commits yet.

Symptom in CI: 23+ consecutive `go.yml` workflow failures since May 19 (last green: commit `f4fd67ac`, immediately before the M-0133 work landed). Every push to main has been red.

Tests that fail with the same root cause:

- `TestDoctor_HonorsCoreHooksPath`, `TestRun_DoctorClean`, `TestDoctorReport_Contents`, `TestDoctorReport_HookOK`, `TestDoctorReport_PreCommitHookOK`, `TestDoctorReport_PreCommitHookGateOnly`, `TestDoctorReport_ValidatorAvailability_Warning`, `TestDoctorReport_EnvLine_InformationalOnly`, `TestDoctorReport_PreCommitHookAlien` — all assert doctor returns 0 problems; doctor correctly reports 3 (one per hook stage).
- `TestRun_DoctorSelfCheck_Passes` — invokes `aiwf doctor --self-check`; fails at step 3 (`add epic`) when the pre-commit hook can't find aiwf.
- `TestIntegrationG37_*` (12 tests), `TestProvenanceCheck_*` (4 tests), `TestScenario_*` (2 tests), `TestEditBody_StdinEndToEnd`, `TestRun_UpdateFromWorktree_WritesSharedHooks`, `TestIntegration_HonorsCoreHooksPath`, `TestDoctor_CheckLatest_ProxyDisabled` — all fail at the first commit attempt for the same reason.

The `selfcheck` CI job fails identically: `make selfcheck` runs `./bin/aiwf doctor --self-check` whose test repo's pre-commit hook tries to resolve `aiwf` via PATH and finds nothing.

## Why M-0133's validation missed this

M-0133's work log says "30/30 selfcheck steps pass" — but that was local. Every developer running `make ci` locally has `aiwf` on PATH already, because they've run `go install ./cmd/aiwf` at some point. The CI runner is a fresh ubuntu-latest VM with no prior aiwf install; the workflow only runs `actions/setup-go` and never installs aiwf to a PATH location. The portable-hooks-need-aiwf-on-PATH precondition is invisible until you run in a clean environment.

Per the kernel's "framework correctness must not depend on the LLM's behavior" rule, this is exactly the kind of silent assumption the chokepoints are meant to catch. M-0133's tests pin the new hook *shape* but don't pin the *PATH precondition* that the shape relies on.

## Fix shape

Add a `go install ./cmd/aiwf` step to the `test` and `selfcheck` jobs in `.github/workflows/go.yml`, after `actions/setup-go` and before the test/selfcheck step:

```yaml
- name: Install aiwf to PATH (required for hooks installed by aiwf init in tests)
  run: go install ./cmd/aiwf
```

`actions/setup-go` adds `$(go env GOPATH)/bin` to PATH automatically on the GitHub-hosted Linux runners, so `go install` puts `aiwf` somewhere `command -v aiwf` will find it. The `vet`, `lint`, `build`, and `vuln` jobs don't need this step — they don't run integration tests that materialize hooks.

ACs (suggested):

- AC-1: After the workflow change, the `test` job passes on a fresh CI run. Verified by pushing a no-op commit and observing the run go green.
- AC-2: After the workflow change, the `selfcheck` job passes. Verified by observing 30/30 selfcheck steps complete in CI.
- AC-3: No code changes outside `.github/workflows/go.yml` — the fix is workflow-only.

## Alternatives considered

1. **`make selfcheck: build install`** — couple the build target to a `go install`. Side effect: every local `make ci` writes to `$GOPATH/bin`, mildly annoying if the operator has intentionally pinned an older binary. Rejected: surgical workflow fix is cleaner.
2. **Per-test PATH setup** — have each affected test build the binary to a tempdir and prepend that tempdir to PATH via `t.Setenv`. Touches ~25 tests. Rejected: too invasive for a fix that's really about CI environment setup.
3. **Doctor downgrades the finding to advisory when not invoked via a hook** — would require doctor to know whether it's the hook caller. Rejected: fragile, papers over a real problem (in a real consumer setup, aiwf MUST be on PATH for hooks to work — the finding is correct, the CI fixture was wrong).

## Why this matters

The chokepoint pattern only works when chokepoints actually run. With CI red for 3 days, no commit has been gated by the full pre-push surface — only the local pre-commit hook (which catches a narrower set of findings) actually fires. Anyone could merge a change that breaks the full test matrix and would only learn about it by reading CI manually.

This is also a precedent for a kernel-side discipline: when a milestone changes hook semantics, its CI verification must include a clean-environment run, not just local `make ci`. Worth filing as a separate gap if the pattern recurs.

## History

Surfaced 2026-05-22 during the cleanup pass after the worktree-status fix triple (G-0151 / G-0153 / G-0154) and the `core.worktree` check rule (G-0155). The operator noticed the red CI status while wrapping the session; investigation confirmed the regression dates to May 19, immediately after the M-0133 merge that closed G-0135.
