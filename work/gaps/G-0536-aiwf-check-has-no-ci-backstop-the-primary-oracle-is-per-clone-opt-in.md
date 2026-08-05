---
id: G-0536
title: aiwf check has no CI backstop; the primary oracle is per-clone opt-in
status: open
priority: high
---
## What's missing

`aiwf check` runs from two git hooks — `--shape-only` at pre-commit, the full
rule set at pre-push — and from nowhere else. No workflow under
`.github/workflows/` invokes it: `make ci` is `vet lint test-cov
coverage-gate-only selfcheck`, and `selfcheck` drives `aiwf doctor --self-check`
against a temporary repo rather than against this one's planning tree.

Git hooks are not committable, so both hook positions are opt-in per working
copy. `aiwf init` / `aiwf update` writes the chain-aware hooks at
`.git/hooks/<name>`; `make install-hooks` symlinks the kernel-specific `.local`
hooks into that chain. A clone where neither has run carries no check at all,
and `--no-verify` removes it on a clone where both have.

Every other locally-firing gate has a second position: the linters, the race
detector, the policy suite, and the gitleaks scan all run again on push. This
one does not.

## Why it matters

`aiwf check` is the mechanism behind a stated commitment —
[`design-decisions.md`](../../docs/design/design-decisions.md) names it as what
makes the guarantees real — and the commitment is unconditional while the
mechanism is opt-in.

The rules it carries alone are the ones with no other detector: id collisions
against trunk, FSM history consistency, provenance trailers on entity commits,
`body-prose-id`, and `skill-body-id`. A tree that lands any of those on trunk
stays wrong until some later pre-push fires on unrelated work and reports a
finding the person running it did not create.

The failure is silent in both directions. An uninitialized clone reports nothing
missing, and CI reports green.

## Resolution shape

A workflow step running `aiwf check` on push and pull request. Three wrinkles
keep it from being a one-liner. The history-walking rules need full history, so
the checkout needs `fetch-depth: 0` rather than the default shallow one; and the
step needs the binary on PATH, which the `selfcheck` job already arranges for
the same reason, so the pattern to copy is in the same file.

The third is not a configuration detail and blocks the step from landing green:
reference resolution consults the refs the asking machine holds, so an id minted
on a local branch that was never pushed is unreachable from any CI checkout and
resolves as an error there while the author's pre-push hook reports a warning.
`fetch-depth` does not reach it — the ref is absent from the remote, not merely
unfetched. On the tree as it stands the step reports errors on day one. Which
view is authoritative is a question this gap inherits rather than answers;
G-0556 holds it.

The pre-push position stays where it is. This adds a backstop rather than
relocating the chokepoint: an oracle that only speaks in CI arrives after the
context that would have made the fix cheap is gone, which is why the early rung
is the one that shapes the work and the late rung is the one that catches what
bypassed it.

Recorded in [`oracles.md`](../../docs/design/oracles.md) §"Position is per clone
for the local rungs".
