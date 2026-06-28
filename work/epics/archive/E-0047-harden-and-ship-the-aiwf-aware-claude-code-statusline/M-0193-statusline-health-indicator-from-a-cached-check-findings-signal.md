---
id: M-0193
title: Statusline health indicator from a cached check-findings signal
status: done
parent: E-0047
depends_on:
    - M-0192
tdd: required
acs:
    - id: AC-1
      title: check --fast runs content rules without the git-history layer
      status: met
      tdd_phase: done
    - id: AC-2
      title: statusline prefixes a health glyph from the cached --fast probe
      status: met
      tdd_phase: done
---
## Deliverable

The statusline gains a **health indicator** driven by a cached, render-safe
check verdict (G-0290), and the kernel gains the fast check surface that makes
it possible:

- **`aiwf check --fast`** — a content-only check mode. It loads the tree
  without the trunk read and runs the in-memory content rules (`check.Run`
  plus the cheap config-dependent tree rules — tree-discipline, area-unknown),
  skipping the trunk-collision / provenance / FSM-history / metrics /
  contract-validation layer that makes a full `aiwf check` seconds-to-minutes
  scale. On this repo the full check is >30s (per-entity git-history walks over
  574 entities); `--fast` is sub-second (~0.35s) — the same fast load path
  `--shape-only` already uses, but with the full in-memory rule set rather than
  tree-discipline alone.
- **Statusline health glyph** — the statusline prefixes `⚠` when
  `aiwf check --fast` reports error-severity findings, and shows nothing when
  the tree is clean or carries only warnings (so the repo's always-present
  benign warnings never pin the light on). The verdict is cached with a TTL +
  HEAD-fold exactly like the CI segment, so the hot render path only reads a
  file — it never runs a live check. The embedded copy stays byte-identical
  (existing M-0155 drift test).

## Scope: freshness from the render cadence, not pushes

G-0290's hard constraint is "never run a live full check on render." The naive
"persist the last hook check and read it" design fails in practice: pushes can
be hours apart, so a push-cadence verdict is stale for most of a session. The
fix is to tie the refresh to the **render cadence** (every prompt) the way the
CI segment already does — cache + short TTL, re-probe on miss — which is only
viable because `--fast` is render-cheap. Freshness is then bounded by the TTL
(seconds after a commit), not by the next push.

`--fast` is the **shared fast tree-health surface** the epic through-line
names: the same `check.Run` verdict `aiwf status` and `aiwf doctor` already
consume internally, now exposed render-safely for the statusline (and
scripts/CI).

## Why this milestone (per the epic)

M3 of E-0047, the keystone. Builds on the M1 harness — every statusline
assertion runs the real script against fixtures. Establishes the shared
tree-health signal that G-0289 (`aiwf doctor`) and G-0277 (`aiwf status`
divergence flag) can later surface.

### AC-1 — check --fast runs content rules without the git-history layer

A `--fast` flag on `aiwf check` loads the tree without the trunk read and runs
the in-memory content rules — `check.Run` (refs-resolve, status-valid,
ids-unique, no-cycles, body-prose-id, AC rules, …) plus the cheap
config-dependent tree rules (tree-discipline, area-unknown) — and skips the
trunk-collision / provenance / FSM-history / metrics / contract-validation
layer that makes a full check seconds-to-minutes scale.

On a tree that is shape-clean but carries a content finding (an unresolved
`addressed_by` reference), `aiwf check --fast` reports the `refs-resolve` error
and exits 1, where `aiwf check --shape-only` is blind to it and exits 0; a
clean tree exits 0, and a warnings-only tree also exits 0 (the linchpin that
keeps benign warnings from lighting the glyph).

Evidence (`internal/cli/integration/check_fast_test.go`): the
shape-only / fast / full contrast on a `refs-resolve` finding; a
provenance/git-history-layer finding the full check emits is absent under
`--fast` (the scope proof); warnings-only → exit 0; clean → exit 0; and the
`--fast --format=json` envelope. Defensive IO-error branches in `runFast` are
`//coverage:ignore`-annotated; everything else is exercised.

### AC-2 — statusline prefixes a health glyph from the cached --fast probe

The statusline prefixes `⚠` (red) when `aiwf check --fast` reports
error-severity findings, and shows nothing when the tree is clean or carries
only warnings — the repo's always-present benign warnings never pin the light
on. The verdict is cached with a TTL + HEAD-fold exactly like the CI segment,
so the hot render path reads a cached file and never runs a live check; the
embedded copy stays byte-identical (existing M-0155 drift test).

Evidence (`internal/policies/statusline_behavioral_test.go`, M1 harness): an
`aiwf` stub controls the probe's exit code — error findings (exit 1) → `⚠` as
the leading prefix; clean / warnings-only (exit 0) → absent; a probe error
(exit >1, e.g. an old binary lacking `--fast`) → degrades to no glyph; and a
cache test where a second render within the TTL keeps showing the cached
verdict even though the stub flipped to clean (proving the probe was not
re-run).

## Work log

- **AC-1 / AC-2 — met.** The `--fast` mode + statusline glyph + behavioral
  coverage landed in `5fd867d8`
  (`feat(check): add --fast content-only check mode + statusline health glyph`).
  Investigation surfaced the real shape of the problem: a full `aiwf check` is
  >30s on this 574-entity tree (the per-entity git-history rules), so a live
  render-time check is impossible; but `check.Run` (the in-memory rules,
  already shared by `status`/`doctor`/`show`/`render`/`rewidth`) is sub-second.
  `--fast` exposes that as a first-class render-safe surface; the statusline
  drives the glyph off its exit code, cached like the CI segment.
- The glyph fires on **errors only** (exit 1), by design: `aiwf check` exits 0
  for a warnings-only tree, so the benign warnings (`archive-sweep-pending`,
  `terminal-entity-not-archived`, …) that are always present in this repo never
  pin the indicator on.
- `--fast` deliberately omits **contract validation** for v1 (the verify half
  shells external validators and is not render-safe; the cheap in-memory config
  half is left out too). A contract-config error is a rarer class the full
  pre-push check still catches; folding the in-memory half in is tracked as a
  follow-up gap (discovered-in M-0193).

## Validation

- `go test ./internal/cli/check/ ./internal/cli/integration/ -run
  'CheckFast|FlagShape|BadFormat|ShapeOnly|FlagsHaveCompletion'` and
  `go test ./internal/policies/ -run 'TestStatusline_M019[123]|TestM0155'` —
  green: AC-1 (5 tests), AC-2 (4 tests), the M-0191/M-0192 regression (the
  health segment degrades on the non-aiwf fixtures), and the embed drift test.
- `make check-fast` green (vet + lint + full test suite); `golangci-lint`
  clean on the changed packages; the `--fast` completion-drift policy passes.
  Full `make ci` runs at the wrap-merge into the epic branch.
- Human-verified renders: clean live worktree → no glyph
  (`● Opus … · ▸ → E-0047/→ M-0193 · …`); a throwaway repo with a broken ref →
  `⚠ ● Opus …` (red ⚠ leads the line). `--fast` timed at ~0.35s vs ~30–69s for
  the full check on the real tree.

## Reviewer notes

- An independent fresh-context reviewer approved with no blocking findings. It
  confirmed the load-bearing claims by measurement (`--fast` 0.35s vs full 69s;
  warnings-only → exit 0 → no glyph; the cache test is non-vacuous; the embedded
  mirror is byte-identical; coverage maps exactly to the three
  `//coverage:ignore` defensive branches).
- Advisories addressed inline: the contract-validation exclusion is now named in
  the `--fast` flag help and the `runFast` doc comment; the statusline header's
  "sub-100ms" note is corrected to acknowledge the cache-miss probe cost.
- Tracked as a follow-up gap: whether the cheap in-memory contract-config check
  should be folded into `--fast` so the glyph also catches contract-config
  errors (currently a false-clean-only gap, zero effect in this contracts-free
  repo). Deferred (minor): a deterministic "aiwf absent from PATH" degrade test.
