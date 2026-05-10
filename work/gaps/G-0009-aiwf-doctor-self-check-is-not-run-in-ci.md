---
id: G-0009
title: '`aiwf doctor --self-check` is not run in CI'
status: addressed
addressed_by_commit:
  - 07f8a84
---

Resolved in commit `07f8a84` (ci(aiwf): G9 — run aiwf doctor --self-check in CI). New `selfcheck` job in `.github/workflows/go.yml` builds the binary and runs `aiwf doctor --self-check` end-to-end. New `make selfcheck` target for local parity, folded into `make ci`. The push trigger paths gain `Makefile` so a Makefile-only change still runs CI. End-to-end regressions (broken trailers, hook installer drift, missing skills, init-against-fresh-repo failures) are now caught at the CI layer rather than waiting for a user to discover them on upgrade.

---
