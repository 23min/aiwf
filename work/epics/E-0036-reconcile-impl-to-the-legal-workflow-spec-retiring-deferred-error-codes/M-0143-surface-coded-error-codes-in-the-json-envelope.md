---
id: M-0143
title: Surface Coded error codes in the JSON envelope
status: in_progress
parent: E-0036
depends_on:
    - M-0138
tdd: required
acs:
    - id: AC-1
      title: Decision D-0013 records the envelope representation and exit-code
      status: met
      tdd_phase: done
    - id: AC-2
      title: Coded verb refusal with --format=json emits status:error + error.code, exit 1
      status: met
      tdd_phase: done
    - id: AC-3
      title: Non-coded verb error emits a well-formed envelope (message, no code)
      status: met
      tdd_phase: done
    - id: AC-4
      title: Every mutating verb accepts --format=json (uniform rollout)
      status: met
      tdd_phase: done
---
## Goal

Surface the structured code carried by a `Coded` verb error in the `aiwf --format=json` envelope, so verb-time legality refusals are machine-readable on par with `findings[].code`. This fulfils E-0036's goal clause — *"errors.As-able for the JSON envelope"* — which the foundation milestone (M-0138) deliberately left to a dedicated unit.

## Context

M-0138 introduced `entity.Coded` plus the typed errors `FSMTransitionError` and `AuthorizeKindError`, each carrying a structured code extractable via `entity.Code(err)`. But the CLI `--format=json` envelope does not yet surface that code: a verb-time refusal appears as an unstructured error. The epic's goal names the envelope as the consumer; this milestone keeps that promise. The wiring is uniform — it surfaces every `Coded` error, including M-0139's cancel codes once they land.

## Decision (recorded: D-0013)

The representation + exit-code question is settled by **D-0013** (`accepted`). The key realization: every `Coded` error originates in a *mutating* verb, none of which surface `--format=json` today (the flag lives only on read verbs) and all of which route through `cliutil.FinishVerb` printing plain text + `ExitUsage`. D-0013 chooses:

- **A2 (uniform flag)** — `--format`/`--pretty` on every mutating verb via a shared `cliutil.AddFormatFlags` registrar, threaded into the single `FinishVerb`/`DecorateAndFinish` chokepoint.
- **(a) structured error object** — an additive `error: {code, message}` slot on `render.Envelope` under `status: "error"`; `code` from `entity.Code` (`errors.As`), `message` from `err.Error()`.
- **C2 (exit unification)** — a `Coded` (legality) refusal exits `ExitFindings` (1), matching the check-time exit for the same violation class; non-`Coded` verb errors stay `ExitUsage` (2), internal failures `ExitInternal` (3).

Pre-dispatch flag-usage errors stay plain-text `ExitUsage` (out of envelope scope).

## Acceptance criteria

Each AC carries an explicit **Evidence** gate — the named test or assertion that fails if the claim breaks. "Looks right" is not evidence.

### AC-1 — Decision D-0013 records the envelope representation and exit-code

D-0013 (`accepted`) records the A2 / (a) / C2 choice with the realization that drove it. *Evidence:* a `internal/policies/` structural assertion that D-0013 resolves via the loader, is `accepted`, carries `## Context` / `## Resolution` / `## Consequences` with non-empty prose, and names the representation (`status:error` + an `error` object) and the exit-code (`ExitFindings`) inside the Resolution section (scoped, not a flat grep).

### AC-2 — Coded verb refusal with --format=json emits status:error + error.code, exit 1

Running a mutating verb that returns a `Coded` error with `--format=json` emits an envelope with `status: "error"`, `error.code` = the structured code (via `entity.Code`), `error.message` = the error text, and exits `1`. *Evidence:* a binary-level test (`internal/cli/integration/`) that runs the built `aiwf` binary on an FSM-illegal `promote` with `--format=json`, JSON-parses stdout, and asserts `status` + `error.code` by **structural field access** (not substring) + the exit code via `ExitedWithCode`.

### AC-3 — Non-coded verb error emits a well-formed envelope (message, no code)

A non-`Coded` verb error (e.g. an unknown entity id) with `--format=json` still emits a well-formed envelope: `status: "error"`, `error.message` set, `error.code` empty/omitted, exit `2` — proving the change is additive and the code field is optional. *Evidence:* a binary-level test parsing the envelope on a non-coded error path, asserting the absent code and the `2` exit.

### AC-4 — Every mutating verb accepts --format=json (uniform rollout)

Every mutating verb accepts `--format=json` — the A2 uniform-rollout guarantee. *Evidence:* a `cmd/aiwf` test that walks the assembled root command tree and asserts every leaf command either registers a `--format` flag or is named in an explicit read-only/exempt allowlist (with rationale); a new mutating verb shipped without the flag fails CI unless consciously exempted.

## Constraints

- Additive to the envelope schema — don't break existing `--format=json` consumers.
- Extract the code via `entity.Code` (`errors.As`), never by parsing the message text.
- `tdd: required`.

## Out of scope

The `Coded` pattern and the typed errors themselves (M-0138); the cancel codes (M-0139) — though this milestone surfaces them once they exist.

## Dependencies

M-0138 (the `Coded` pattern + the first codes). Closes the envelope clause of E-0036's goal.

## Work log

The investigation reframed the milestone: every `Coded` error originates in a mutating verb, none of which surfaced `--format=json` (read-verb-only) and all of which routed through `cliutil.FinishVerb` printing plain text + `ExitUsage`. D-0013 recorded the A2 / (a) / C2 choice. The core (envelope `error` slot, `cliutil.OutputFormat` + `AddFormatFlags` + the `FinishVerb`/`DecorateAndFinish` rework) and the 4 AC tests landed in one feature commit `9cdbbd8c`; the 14-verb flag rollout was authored by the `aiwf-extensions:builder` subagent on this worktree (no isolation, no commits) and parent-verified (build/vet/lint + full suite). Per-AC RED was demonstrated (AC-4 fired listing the 6 genuinely-missing commands; an AC-2 throwaway mutation forcing the code empty drove RED, reverted).

### AC-1 — Decision D-0013 records the envelope representation and exit-code

D-0013 (`accepted`) records A2 (uniform flag via the shared chokepoint), (a) (additive `error:{code,message}` object under `status:"error"`), and C2 (Coded refusal → `ExitFindings`). commit `9cdbbd8c` · test: `TestM0143_AC1_Decision` (`internal/policies/`) — loader-resolved, status + named sections + scoped Resolution assertions (`ExitFindings`/`error`/`status`).

### AC-2 — Coded verb refusal with --format=json emits status:error + error.code, exit 1

`FinishVerb` extracts the code via `entity.Code` and emits a `status:"error"` envelope with `error.code`/`error.message`; a Coded error exits `1` (C2). commit `9cdbbd8c` · test: `TestCodedEnvelope_VerbRefusal_AC2` — two verbs / two codes (`fsm-transition-illegal` via promote, `authorize-kind-not-allowed` via authorize), structural field access on the parsed envelope + exit-code + empty-stderr assertions. RED proven (code-empty mutation).

### AC-3 — Non-coded verb error emits a well-formed envelope (message, no code)

A non-`Coded` error (unknown entity) emits `status:"error"` with `error.message` set and `error.code` omitted, exit `2` — additive, code optional. commit `9cdbbd8c` · test: `TestCodedEnvelope_NonCodedError_AC3`.

### AC-4 — Every mutating verb accepts --format=json (uniform rollout)

Every Runnable leaf command either registers `--format` or is named in an explicit `formatExempt` allowlist with rationale. commit `9cdbbd8c` · test: `TestFormatFlagUniformRollout_AC4` (`internal/cli/integration/`, walks `cli.NewRootCmd()`). RED proven (initial run listed 6 missing commands). `import`/`rewidth` + read-display commands are exempted (bespoke / non-`FinishVerb` paths) and tracked in G-0169.

## Decisions made during implementation

- **D-0013 — Surface Coded verb refusals as a `status:error` envelope object, exit 1** (`accepted`). A2 (uniform `--format` via the shared `FinishVerb`/`DecorateAndFinish` chokepoint) + representation (a) (additive `error:{code,message}`) + C2 (legality refusal exits 1, unifying with check-time). Rejected: findings-reuse and bare top-level code (representation), A1 legality-only (CLI inconsistency), keep-exit-2 (semantically "usage" is wrong for a well-formed-but-illegal action).

## Validation

```
CGO_ENABLED=0 go build ./...            # exit 0
go test ./... -count=1 -parallel 8      # 56 packages ok · 0 failures (3 clean runs; see Reviewer notes on the verb flake)
golangci-lint run ./internal/cli/... ./internal/render/... ./internal/policies/...  # 0 issues
aiwf check                              # 0 errors · 8 warnings (pre-existing: M-0102 ×5, G-0061 ×3)
```

Per-AC mechanical evidence (all green): `TestM0143_AC1_Decision` (AC-1); `TestCodedEnvelope_VerbRefusal_AC2` (AC-2); `TestCodedEnvelope_NonCodedError_AC3` (AC-3); `TestFormatFlagUniformRollout_AC4` (AC-4). Plus `TestSuccessEnvelope_FormatJSON` (success path) and `TestOutputFormat_EmitHelpers` (`internal/cli/cliutil/`, all six emit text/JSON branches — the branch-coverage chokepoint).

## Deferrals

- **G-0169** — `import`, `rewidth`, and the read-display commands (`contract recipes`, `contract recipe show`, `render roadmap`) route around the shared `FinishVerb` chokepoint and still lack `--format=json`. D-0013's A2 scope is the chokepoint; these are recorded as `formatExempt` entries (with the gap id) in `TestFormatFlagUniformRollout_AC4` and tracked for follow-up wiring. No deferred or cancelled ACs (all four `met`).

## Reviewer notes

- **The reframe is the substance.** The milestone title reads "add a field"; the code said "give mutating verbs a JSON error path." The decision and the 14-verb rollout follow from that.
- **C2 is a behavior change (2→1) in both formats**, not just JSON. It unifies a legality refusal's exit code with the check-time exit for the same violation. The M-0125 negative driver requires only non-zero, so it stays green; no test pinned exit 2 for these refusals (verified before committing).
- **JSON mode writes a single clean envelope to stdout, nothing to stderr** — the binary tests assert empty stderr so a CI consumer can parse stdout directly.
- **Branch coverage via a direct cliutil unit test.** Rather than reachability-archaeology for `emitFindings` (the `HasErrors(result.Findings)` arm), `TestOutputFormat_EmitHelpers` calls all three emit helpers directly in both modes — deterministic coverage of all six branches, independent of which verb reaches each path at the binary seam. It is serial (swaps `os.Stdout`/`os.Stderr`) and noted in `setup_test.go`'s serial list.
- **Subagent rollout, parent-verified.** The 14-verb flag threading was the `aiwf-extensions:builder` subagent's; the parent independently re-ran build/vet/lint and the full suite, and reviewed the diff, before committing.
- **Latent M-0142 drift fixed in passing.** `check_summary_binary_test.go` carried a gofmt map-alignment drift from M-0142's rename (missed because that milestone only lint-checked `./internal/check/`). gofmt-aligned here; a tree-wide `gofmt -l` confirmed it was the only escape.
- **The verb-package flake.** One of four full-suite runs failed 4 `internal/verb` tests (promote/move); they pass in isolation and the suite is green on re-run. The diff touches no `internal/verb` file — it is the documented git-contention flake (G-0097/G-0127 class), environmental, not a regression.

