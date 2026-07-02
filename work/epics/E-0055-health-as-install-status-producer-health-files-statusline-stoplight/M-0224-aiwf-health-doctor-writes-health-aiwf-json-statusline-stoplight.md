---
id: M-0224
title: 'aiwf health: doctor writes health.aiwf.json + statusline stoplight'
status: in_progress
parent: E-0055
tdd: required
acs:
    - id: AC-1
      title: doctor exposes its warnings and errors with severity and message
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf writes .claude/health.aiwf.json from doctor's warnings and errors
      status: met
      tdd_phase: done
    - id: AC-3
      title: statusline reads the health files and renders the four-state stoplight
      status: met
      tdd_phase: done
---
## Deliverable

Give operators visibility of `aiwf`'s installation and configuration warnings and errors
in the statusline. Two halves, landed as one milestone:

- **Producer** — surface `aiwf doctor`'s warnings and errors as structured problems and
  write them to `.claude/health.aiwf.json` (the fixed ai-dotfiles schema).
- **Consumer** — the statusline reads `.claude/health.*.json`, unions across producers,
  and renders a four-state stoplight at the maximum severity, never running a check on the
  render path.

## Acceptance criteria

### AC-1 — doctor exposes its warnings and errors with severity and message

`aiwf doctor` collects its problem states — the blocking checks (today's error count) and
the advisory ones — as `{severity, message}`, `severity` one of `warn` / `error`. The
existing human report is unchanged. Evidence: a repo with a known problem (e.g. a missing
`aiwf.yaml`) yields an error-severity problem whose message names it; a clean section
yields none.

### AC-2 — aiwf writes .claude/health.aiwf.json from doctor's warnings and errors

`aiwf doctor --write-health` maps those problems onto the fixed ai-dotfiles schema
(`{generated_at, findings:[{source:"aiwf", severity, message}]}`; empty `findings` when
healthy) and atomic-writes the file to the main checkout's `.claude/`, resolved even from a
linked worktree; `aiwf update` writes it too. Evidence: seam tests driving `doctor
--write-health` and `aiwf update`; a linked-worktree resolution test; the healthy →
empty-findings mapping.

### AC-3 — statusline reads the health files and renders the four-state stoplight

The statusline globs `.claude/health.*.json`, unions the findings, and prefixes the line
with a four-state stoplight at the maximum severity: green `●` healthy, yellow `▲` warn,
red `▲` error, gray `●` unknown (no readable file). It runs no check on the render path.
Evidence: behavioral tests per state, the cross-producer union (max severity wins), and the
all-corrupt → gray degrade.

## Work log

- **AC-1 / AC-2 / AC-3 — met.** `aiwf doctor` surfaces warnings/errors as `[]Problem`
  (byte-identical human report); `WriteHealth` + `aiwf doctor --write-health` writes
  `.claude/health.aiwf.json` (main-checkout-resolved, atomic), refreshed by `aiwf update`;
  the statusline renders the four-state stoplight from the health files. Delivered as one
  change; per-AC phase timeline in `aiwf history M-0224/AC-<N>`.

## Validation

- `make check-fast` green (full `golangci-lint` + `go vet` + full `go test` suite).
- `go build ./...` clean; `bash -n internal/skills/embedded-statusline/statusline.sh` clean.
- Diff-scoped coverage: the two CLI seams (`runWriteHealth`, `aiwf update`'s refresh call)
  are exercised by `internal/cli/integration/health_producer_test.go`; the two unreachable
  filesystem-fault error branches carry `//coverage:ignore`.

## Reviewer notes

- Two rounds of independent fresh-context review (`reviewer` agent) gated closure. Round 1
  → request-changes: B1 (untested CLI seams), B2 (no worktree-resolution evidence), B3
  (all-corrupt rendered green vs ADR-0026's "none parse → gray"), C1 (consumer implemented
  off its own milestone). Round 2 verified five of six resolved and caught R1 (an
  `update.go` error branch missing the `//coverage:ignore`); R1 fixed → approve.
- Scope was deliberately minimized after an initial over-built single-source-of-truth
  refactor of the whole doctor report was reverted. The shipped change is an additive
  `problems int → []Problem` thread; doctor's human output is unchanged.
- M-0226 (originally the statusline-consumer split) was folded in as AC-3 and cancelled;
  M-0225 (originally the producer split) was folded into AC-2 and cancelled — the whole
  feature lands as this one milestone.
