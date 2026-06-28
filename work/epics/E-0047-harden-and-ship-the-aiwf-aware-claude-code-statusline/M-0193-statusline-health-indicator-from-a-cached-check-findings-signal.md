---
id: M-0193
title: Statusline health indicator from a cached check-findings signal
status: in_progress
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
      status: open
      tdd_phase: done
---
## Deliverable

The statusline gains a **health indicator** driven by a cached, render-safe
check verdict (G-0290), and the kernel gains the fast check surface that makes
it possible:

- **`aiwf check --fast`** — a content-only check mode. It loads the tree
  without the trunk read and runs the in-memory content rules (`check.Run`
  plus the cheap config-dependent tree rules — tree-discipline, area-unknown),
  skipping the trunk-collision / provenance / FSM-history / metrics layer that
  makes a full `aiwf check` seconds-to-minutes scale. On this repo the full
  check is >30s (per-entity git-history walks over 574 entities); `--fast` is
  sub-second — the same fast load path `--shape-only` already uses, but with
  the full in-memory rule set rather than tree-discipline alone.
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
trunk-collision / provenance / FSM-history / metrics layer that makes a full
check seconds-to-minutes scale.

On a tree that is shape-clean but carries a content finding (an unresolved
`depends_on` reference), `aiwf check --fast` reports the `refs-resolve` error
and exits 1, where `aiwf check --shape-only` is blind to it and exits 0; on a
clean tree `--fast` exits 0.

Evidence: a fixture-tree test in `internal/cli/check` asserting the
shape-only / fast / full contrast on a `refs-resolve` finding, and that a
provenance/git-history-layer finding the full check emits is absent under
`--fast` (the scope proof).

### AC-2 — statusline prefixes a health glyph from the cached --fast probe

The statusline prefixes `⚠` (red) when `aiwf check --fast` reports
error-severity findings, and shows nothing when the tree is clean or carries
only warnings — the repo's always-present benign warnings never pin the light
on. The verdict is cached with a TTL + HEAD-fold exactly like the CI segment,
so the hot render path reads a cached file and never runs a live check; the
embedded copy stays byte-identical (existing M-0155 drift test).

Evidence: M1-harness behavioral tests stub `aiwf` to a controlled JSON
envelope — error findings → `⚠` present as the leading prefix; clean →
absent; warnings-only → absent — plus a cache test asserting a second render
within the TTL does not re-invoke the probe.

