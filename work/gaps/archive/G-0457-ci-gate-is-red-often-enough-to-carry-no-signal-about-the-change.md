---
id: G-0457
title: CI gate is red often enough to carry no signal about the change
status: addressed
priority: high
addressed_by_commit:
    - 24874cc0c66f640e903baf1bc0620c6169fb8e64
---
## What's missing

Nothing distinguishes "CI is red because your change broke something" from "CI is red." 32 of the 100 `go.yml` runs preceding 2026-07-30 failed, and from 2026-07-08 to 2026-07-18 it failed on essentially every run — eleven consecutive days during which 1,151 commits and eight epic wraps landed. Three causes, of different character:

- **The stress harness is the chronic cause — 18 red runs, 2026-07-11 through 2026-07-29, uninterrupted — and it sits on the every-push path twice.** `.github/workflows/go.yml`'s test step runs `go test … ./...`, which sweeps in `internal/stresstest`, whose real-binary scenarios launch concurrent `aiwf` subprocesses against disposable git repos (65–77s in CI), *and* `cmd/stresstest`, whose `TestRunRun_ScenarioAll_*` tests invoke the whole catalog — the same run `make stress` performs — for a further 42–68s. Those scenarios report runner contention as an aiwf defect rather than a timing outcome; that oracle defect is G-0468.
- **`govulncheck` blocking on stdlib CVEs was an acute burst — 21 red runs, all between 2026-07-08 and 2026-07-18, none since.** The `vuln` job installs `golang.org/x/vuln/cmd/govulncheck@latest` and runs it over `./...`. It reported `GO-2026-5856` in `crypto/tls` at the exact Go version pinned in the same workflow. No change in this repository could clear it; only a toolchain bump could, and that took ten days. The job's own comment justifies blocking on the grounds that "the dep set is small (go-cmp / goldmark / yaml.v3) so the run is ~10s" — a rationale about *dependency* CVEs that silently also governs *stdlib* CVEs, whose remediation path and latency are entirely different. The two classes were never separated. The structural exposure remains after the bump: the scanner's own version still floats, and the toolchain is pinned to an exact patch in seven places across two workflow files, so each stdlib disclosure costs seven edits. This was the third such forced bump.
- **The diff-scoped coverage gate accounts for the remaining 8 red runs**, because it can fire nowhere but CI — after the push has landed. Tracked separately as G-0469.

The stress-harness placement also contradicts a stated contract. `CLAUDE.md` describes the harness as *"dev-only tooling, never installed alongside `cmd/aiwf`, run by hand rather than scheduled or wired into `make ci`"*; the workflow disagrees, and `cmd/stresstest`'s whole-catalog test disagrees most directly of all. Whichever way that reconciles, the two surfaces should say the same thing.

## Why it matters

Validation as the chokepoint is a load-bearing commitment: `aiwf check` runs pre-commit and pre-push precisely so guarantees are mechanical rather than remembered. A gate that is red by default inverts that. During the eleven-day window every push landed against a failing signal, so nobody could have learned anything from a green run either — the gate was not weakly informative, it was uninformative in both directions.

The failure mode compounds. Once red is the baseline, the habit of reading CI decays, and the next genuinely broken build reaches trunk indistinguishable from the noise. This is the same shape as the local-gate erosion recorded in G-0179, where a CI-only linter let debt accumulate invisibly across three milestone wraps — except here it is the gate itself, not one lane of it.

It also blocks adjacent work. G-0400 wants the stress catalog widened from 10 of 38 verbs; widening a catalog that sits on the critical path of every push multiplies the flake surface rather than improving coverage.

## Resolution shape

This gap carries the placement half: restore signal by taking the timing-shaped scenarios off the every-push path, and close the `govulncheck` lane. The oracle redesign that makes those scenarios trustworthy wherever they run is G-0468, its enabling error-code fix is G-0467, and the coverage-gate tier is G-0469. Landing the placement change alone restores the signal without curing the oracles — it is a tourniquet, and G-0468 is why it is not mistaken for the cure.

1. **Take the real-binary drivers and `cmd/stresstest`'s catalog tests off the every-push path.** The seam exists in the package's own structure: each scenario splits between a `*_classify_test.go` file pinning the pure decision function against fabricated outcomes (fast, deterministic, worth running always — 16 such files) and a `*_test.go` file driving real subprocesses (slow, timing-shaped, valuable on demand). A build tag is the mechanism. `-short` is not: it is a global switch that would also disable the binary integration tests across `cmd/aiwf` and `internal/cli`, which CI wants. Gating the drivers out drops `internal/stresstest` statement coverage from 85.5% to 34.7%, which is what makes G-0468 the precondition for ever restoring them to the default lane.
2. **Pin `govulncheck`** instead of resolving `@latest` on every run, so the scanner's own version is a reviewed input rather than a moving one.
3. **Separate the stdlib-CVE lane from the dependency-CVE lane**, so a disclosure this repo cannot act on does not mask one it can.
4. **Single-source the toolchain pin**, so a stdlib bump is one edit rather than seven.

Then reconcile the `CLAUDE.md` stress-harness paragraph with whatever the workflow ends up doing.

Two decisions ride along and should be made explicitly rather than defaulted into:

- **Where the real-binary scenarios run instead** — a build-tag split inside `go.yml`, a separate `workflow_dispatch` job, or a scheduled nightly. Three different answers with three different failure modes; a nightly that nobody reads is its own version of this gap. `flake-hunt.yml` is disqualified as a destination: G-0438 records it failing on the same runner for the same reason, naming these same packages. Any destination that shares a runner with a broad test sweep reproduces the flake, because the flake tracks co-tenancy rather than machine size — run in isolation on four cores the stress packages pass five repeats out of five.
- **Whether `govulncheck` should block on stdlib findings at all.** Non-blocking has a real cost: a genuinely exploitable stdlib finding would then only warn. Blocking has the cost measured above. A third option is to stop pinning an exact patch version, so a fix arrives with the next patch release without an edit — buying remediation latency at the price of toolchain reproducibility. Weigh also that stdlib symbol results over-approximate through interface dispatch: of the four traces `GO-2026-5856` reports here, two reach TLS only through `io.Copy` and `io.WriteString` on arbitrary writers.

Sizing: `wf-patch` for the placement and workflow changes above. The oracle work is larger and is scoped in G-0468.

## Where to fix

- `.github/workflows/go.yml` — the `test` job's `go test … ./...` step, the `vuln` job's install-and-run steps, and six of the seven exact toolchain pins.
- `.github/workflows/gitleaks.yml` — the seventh exact toolchain pin.
- `internal/stresstest/*_test.go` and `cmd/stresstest/run_test.go` — the real-binary and whole-catalog drivers to gate; `*_classify_test.go` stays untouched.
- `CLAUDE.md` §"Stress-test harness" — the contract paragraph to reconcile.
- `Makefile` — `make stress` already drives the catalog by hand and is the model for the on-demand path.

Recorded as Q1 in [`docs/initiatives/quality-signal-and-cadence.md`](../../docs/initiatives/quality-signal-and-cadence.md), which carries the measured baseline this gap cites and the four related findings on gate depth, duplication absorption, commit-history readability, and backlog equilibrium.
