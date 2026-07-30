---
id: G-0457
title: CI gate is red often enough to carry no signal about the change
status: open
priority: high
---
## What's missing

Nothing distinguishes "CI is red because your change broke something" from "CI is red." The `go.yml` workflow failed 31 of its last 100 runs, and from 2026-07-08 to 2026-07-18 it failed on essentially every run — eleven consecutive days during which 1,151 commits and eight epic wraps landed. Two independent causes, both still live, and neither reports on the change under test:

- **`govulncheck` blocks on stdlib CVEs against a pinned toolchain.** The `vuln` job (`.github/workflows/go.yml:150-159`) installs `golang.org/x/vuln/cmd/govulncheck@latest` and runs it over `./...`. It began reporting `GO-2026-5856` in `crypto/tls` at the exact Go version pinned in the same workflow. No change in this repository could clear it; only a toolchain bump could, and that took ten days. The job's own comment justifies blocking on the grounds that "the dep set is small (go-cmp / goldmark / yaml.v3) so the run is ~10s" — a rationale about *dependency* CVEs that silently also governs *stdlib* CVEs, whose remediation path and latency are entirely different. The two classes were never separated.
- **The stress harness runs inside the default test step.** `.github/workflows/go.yml:94` runs `go test … ./...`, which sweeps in `internal/stresstest`. That package's real-binary scenarios launch concurrent `aiwf` subprocesses against disposable git repos: ~34s locally, ~77s in CI, and timing-shaped by construction. `TestConcurrentIDAllocationScenario_RealBinary_NConcurrentActorsAllGetDistinctIDs` failed the three most recent red runs with *"only 7/8 concurrent actors succeeded — expected all to serialize successfully within repolock's timeout"* — runner contention, not a defect the scenario was written to catch.

The second cause also contradicts a stated contract. `CLAUDE.md` describes the harness as *"dev-only tooling, never installed alongside `cmd/aiwf`, run by hand rather than scheduled or wired into `make ci`"*; the workflow disagrees. Whichever way that reconciles, the two surfaces should say the same thing.

## Why it matters

Validation as the chokepoint is a load-bearing commitment: `aiwf check` runs pre-commit and pre-push precisely so guarantees are mechanical rather than remembered. A gate that is red by default inverts that. During the eleven-day window every push landed against a failing signal, so nobody could have learned anything from a green run either — the gate was not weakly informative, it was uninformative in both directions.

The failure mode compounds. Once red is the baseline, the habit of reading CI decays, and the next genuinely broken build reaches trunk indistinguishable from the noise. This is the same shape as the local-gate erosion recorded in G-0179, where a CI-only linter let debt accumulate invisibly across three milestone wraps — except here it is the gate itself, not one lane of it.

It also blocks adjacent work. G-0400 wants the stress catalog widened from 10 of 38 verbs; widening a catalog that sits on the critical path of every push multiplies the flake surface rather than improving coverage.

## Resolution shape

Three separable changes, no design work beyond the two decisions named below:

1. **Take the real-binary stress scenarios off the every-push path.** The seam already exists in the package's own structure: each scenario is split between a `*_classify_test.go` file pinning the pure decision function against fabricated outcomes (fast, deterministic, worth running always — 16 such files) and a `*_test.go` file driving real subprocesses (slow, timing-shaped, valuable on demand). Gate the latter behind a build tag or `testing.Short()`; keep the former in the default run.
2. **Pin `govulncheck`** instead of resolving `@latest` on every run, so the scanner's own version is a reviewed input rather than a moving one.
3. **Separate the stdlib-CVE lane from the dependency-CVE lane**, so a disclosure this repo cannot act on does not mask one it can.

Then reconcile the `CLAUDE.md` stress-harness paragraph with whatever the workflow ends up doing.

Two decisions ride along and should be made explicitly rather than defaulted into:

- **Where the real-binary scenarios run instead** — a build-tag split inside `go.yml`, a separate `workflow_dispatch` job, or a scheduled nightly. Three different answers with three different failure modes; a nightly that nobody reads is its own version of this gap.
- **Whether `govulncheck` should block on stdlib findings at all.** Non-blocking has a real cost: a genuinely exploitable stdlib finding would then only warn. Blocking has the cost measured above. The current setting is not the result of weighing those.

Sizing: `wf-patch`. The scope is workflow configuration, a test-gating tag, and one documentation paragraph — no kernel change and no new policy.

## Where to fix

- `.github/workflows/go.yml` — the `test` job's `go test … ./...` step and the `vuln` job's install-and-run steps.
- `internal/stresstest/*_test.go` — the real-binary drivers to gate; `*_classify_test.go` stays untouched.
- `CLAUDE.md` §"Stress-test harness" — the contract paragraph to reconcile.
- `Makefile` — `make stress` already drives the catalog by hand and is the model for the on-demand path.

Recorded as Q1 in [`docs/initiatives/quality-signal-and-cadence.md`](../../docs/initiatives/quality-signal-and-cadence.md), which carries the measured baseline this gap cites and the four related findings on gate depth, duplication absorption, commit-history readability, and backlog equilibrium.
