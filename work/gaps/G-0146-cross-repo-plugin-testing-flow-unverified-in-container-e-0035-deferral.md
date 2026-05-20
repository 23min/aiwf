---
id: G-0146
title: Cross-repo plugin testing flow unverified in container (E-0035 deferral)
status: open
discovered_in: M-0132
---
## What's missing

The "Cross-repo plugin testing" flow documented in `CLAUDE.md`
(authoring a `SKILL.md` as a fixture in `internal/policies/testdata/<skill-name>/`,
TDD-iterating against the fixture, copying the fixture into the
rituals plugin repo at `~/Projects/ai-workflow-rituals/` as a
separate commit there at wrap, and a drift-check test in this
repo comparing the fixture to the local marketplace cache) has
been exercised end-to-end **only on the macOS host**. The same
flow inside the E-0035 devcontainer is structurally reachable
(the sibling rituals repo can be mounted via
`.devcontainer/devcontainer.json`; the shadow-mount target for
the marketplace cache is now sanity-checked by
[M-0135](../epics/E-0035-devcontainer-based-dev-loop/M-0135-aiwf-doctor-containerized-env-awareness-detection-mount-check.md)
AC-2) but **the end-to-end flow has not been run in container**.

Three boundary crossings remain unverified:

1. **Filesystem: aiwf repo → sibling rituals repo.** From
   `/workspaces/aiwf`, can the operator `cd ../ai-workflow-rituals`
   and find the actual rituals repo? Depends on a mount entry in
   `.devcontainer/devcontainer.json` that doesn't exist today.
2. **Git identity / credentials in the rituals repo from
   container.** Does `git commit && git push` work over there
   using the container's gh credential helper (set up by M-0132's
   `.devcontainer/init.sh`)?
3. **Drift-check cache path under container resolution.** The
   check reads `~/.claude/plugins/cache/ai-workflow-rituals/.../SKILL.md`
   which resolves to `/home/vscode/.claude/plugins/cache/...` in
   container. M-0135 AC-2 partial-covers this (it sanity-checks
   the parent dir exists); the actual SKILL.md path resolution
   under the drift-check test isn't pinned for container.

## Why it matters

The next skill change shipped from the container is likely to
surface bugs in this flow (mount missing, gh credentials not
configured for rituals repo, drift-check cache path resolution
edge case). Because skills change infrequently (~quarterly), the
operator could lose hours debugging at the moment they finally
need the flow.

Filing as a gap rather than as an immediate milestone is
deliberate: the work is correct-but-rare, and the natural forcing
function is a real skill change in flight. Until that happens, the
gap holds the known-unknown so it isn't rediscovered from scratch.

## Discipline today

Author skill changes from the macOS host (where the flow is
known-working) until this gap closes, OR test the flow manually
in the container before shipping a real skill change. The drift-
check test under `internal/policies/` skips cleanly when the
marketplace cache is absent (per CLAUDE.md *"Cross-repo plugin
testing"*), so green CI without container verification doesn't
prove the flow works end-to-end in container.

## Proposed fix shape

Two candidates, in increasing cost:

1. **Mount + document only** (tight Shape A from the M-0136
   scoping conversation): add a sibling-repo mount entry to
   `.devcontainer/devcontainer.json` (pin via a small
   `internal/policies/` assertion), update
   `.devcontainer/README.md` with the cross-repo flow steps. ~30
   min. Operator-runs the flow once to verify; no smoke test.

2. **Mount + script + smoke test** (Shape B): same as Shape A
   plus a small bash script (or `make` target) that exercises
   fixture → rituals-copy → drift-check end-to-end, with a smoke
   test gated on the rituals repo being present
   (`internal/cli/integration/` skips when sibling missing). A
   few hours; surfaces drift earlier.

Shape A is the minimum-viable deferral close. Shape B is
appropriate once skill changes become more frequent or a CI
matrix milestone (separately deferred) adds the container surface
to PR validation.

## Related

- E-0035 — Devcontainer-based dev loop (the epic this concern
  originated from; scoped out as a "later milestone" in
  `epic.md`).
- M-0132 — Land .devcontainer skeleton (the milestone that
  created the container; cross-repo flow was deliberately
  out-of-scope there).
- M-0135 — `aiwf doctor` containerized-env awareness (partial
  coverage via the shadow-mount status check).
- CLAUDE.md *"Cross-repo plugin testing"* section — the
  authoritative flow description.
- [claude-code#31388](https://github.com/anthropics/claude-code/issues/31388) — once upstream lands the plugin-index fix
  that obsoletes the shadow-mount, parts of this gap simplify
  (the drift-check cache path resolution becomes less
  container-specific).
