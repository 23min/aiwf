# Epic wrap — E-0035

**Date:** 2026-05-20
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0035-devcontainer-dev-loop
**Merge commit:** _epic branch fast-forwarded to main after each
milestone's `--no-ff` merge per the hybrid worktree model; no
distinct epic-merge commit at wrap. The wrap-artefact commit
(this commit) and the `aiwf promote E-0035 done` commit carry
the `aiwf-verb: wrap-epic` / `aiwf-verb: promote` trailers that
anchor the boundary in `aiwf history E-0035`._

## Milestones delivered

- M-0132 — Land .devcontainer skeleton (features-first, Go base,
  project-scope plugins) (merged `f4fd67ac`).
- M-0133 — Multi-context kernel surfaces: portable hooks + doctor
  check (merged `be80b5a3`).
- M-0134 — CLAUDE.md DO/DON'T refresh: container primary, macOS
  host fallback (merged `638e4c24`).
- M-0135 — `aiwf doctor` containerized-env awareness: detection
  + mount check (merged `a3628323`).

## Summary

E-0035 made the devcontainer the dominant dev surface for this
repo. M-0132 landed the skeleton (Go-base devcontainer +
project-scope plugin install + the shadow-mount workaround for
[claude-code#31388](https://github.com/anthropics/claude-code/issues/31388)).
M-0133 fixed the three kernel surfaces that broke under
multi-context use — portable PATH-relative hook lookup
(G-0135), worktree-aware `aiwf update` writing to shared hooks
dir (G-0136), and `aiwf doctor` reading `enabledPlugins` from
`.claude/settings.json` instead of the path-strict
`installed_plugins.json` (G-0138). M-0134 re-positioned
`CLAUDE.md`'s test-running guidance so the container-primary
path leads and the macOS-host wrapper is clearly the fallback,
backed by a mechanical structural assertion. M-0135 added two
informational lines to `aiwf doctor` — `env:` (container vs
host) and `plugin-index-mount:` (shadow-mount health) — so
operators land on a quick "where am I + is the workaround
healthy" signal without grepping for it.

The full goal as originally framed in the epic spec was
"devcontainer-based dev loop available, dogfooded on this repo,
and cross-repo (Liminara, FlowTime) ready." The dogfooding bar
was met; the cross-repo bar was deliberately deferred to G-0146
once the dogfooding loop proved itself enough to warrant pinning
the cross-repo flow rather than re-deriving it per consumer.

## ADRs ratified

- none — the load-bearing decisions (PATH-relative hook lookup,
  enabledPlugins as source of truth, container-primary doctrine
  in CLAUDE.md) live in the respective milestone spec bodies plus
  CLAUDE.md itself. If a future ADR ratifies any of these
  retroactively, it links back to the M-0133 / M-0134 specs as
  the original written record.

## Decisions captured

- none — every mid-flight decision (red→green bundled commits
  for policy-shaped ACs; hybrid worktree model with epic branch
  as persistent home and per-milestone branches off main;
  removed-test-rather-than-worked-around for the actor-resolution
  edge case in M-0135) was captured in the respective milestone
  spec's `## Decisions made during implementation` section. No
  cross-cutting decisions surfaced that warranted a standalone
  `aiwf add decision` record.

## Follow-ups carried forward

- **G-0146** — Cross-repo plugin testing flow unverified in
  container (E-0035 deferral). The devcontainer dogfooding loop
  is proven for this repo; cross-repo (Liminara, FlowTime)
  verification was deliberately deferred to a follow-up gap so
  the cross-repo flow can be pinned once, with the dogfooding
  baseline as reference.

## Doc findings

`wf-doc-lint` was not run as a separate sweep at epic wrap.
Per-milestone wrap passes (M-0132, M-0133, M-0134, M-0135) each
inspected their own diff and reported clean. The epic's
change-set is fully covered by the union of those four sweeps;
no cross-milestone doc concerns surfaced during integration.

## Handoff

Ready for the next epic:

- The devcontainer is the default dev surface; new contributors
  open the repo in VS Code → "Reopen in Container" and `make
  ci` runs green without macOS-specific setup. `CLAUDE.md` leads
  with the container-primary path; the macOS host fallback is
  documented for the rare case.
- The kernel surfaces that broke under multi-context use
  (hooks, doctor's plugin source-of-truth) now hold across
  worktrees, devcontainers, and re-clones. The host shadow-mount
  workaround for upstream issue #31388 has a doctor-level health
  check.
- The hybrid worktree model (epic branch as persistent home,
  per-milestone branches off main, `--no-ff` merges back to
  main) held cleanly across all 4 milestones. Same shape is
  available for the next epic; no further plumbing required.

Deliberately left open:

- **Cross-repo dogfooding** (G-0146) — the proven loop on this
  repo is the reference; cross-repo pinning waits for a forcing
  function (a consumer reporting friction, a planned multi-repo
  feature) rather than speculative work.
- **Container-aware advice strings on existing doctor messages**
  (e.g. "rebuild the container" vs "run `aiwf init`" in select
  contexts) — deferred from M-0135 as preachy without observed
  friction; gap-candidate, not filed.
- **Removing the shadow-mount workaround entirely** — gated on
  [claude-code#31388](https://github.com/anthropics/claude-code/issues/31388)
  shipping a fix. When it lands, the `~/.claude-linux/plugins`
  symlink + the in-container bind-mount + the
  `plugin-index-mount:` doctor line can all retire in one PR
  here plus Liminara plus FlowTime.
