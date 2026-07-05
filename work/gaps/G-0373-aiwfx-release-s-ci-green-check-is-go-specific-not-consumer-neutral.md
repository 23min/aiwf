---
id: G-0373
title: aiwfx-release's CI-green check is Go-specific, not consumer-neutral
status: open
---
## Problem

`aiwfx-release`'s step-1 pre-release check ("CI is green on the last Go-affecting
commit reachable from HEAD", `SKILL.md:34-49`) hardcodes Go as the consumer's
stack: it names `go.yml` as "the primary Go workflow", greps
`'**/*.go' 'go.mod' 'go.sum'` to find "the most recent Go-affecting commit",
and cites `go test ./...` as the local check that isn't sufficient on its own
(echoed again in the Constraints section). It also cites a
`` `release(aiwf): vX.Y.Z` `` prep commit as an example of a markdown-only
commit — "aiwf" there is this project's own Conventional Commits scope,
copied verbatim from aiwf's own `CLAUDE.md`, and meaningless in a consumer
repo (inconsistent with step 4's own correctly generic `docs(changelog):
vX.Y.Z`, no scope).

## Why it matters

`aiwfx-release` (and the `deployer` agent that runs it) is a shipped,
consumer-facing ritual materialized into every project that runs `aiwf
init`/`update` — most of which are not Go projects. A consumer on Python,
TypeScript, Rust, Elixir, or C# gets a ritual whose only mechanically
followable CI-green check names the wrong workflow file and the wrong path
filter; the deployer agent either silently improvises a substitute
(undermining the skill's authority as the written procedure) or follows it
literally and finds zero "Go-affecting commits" ever, defeating the
"releases ride on green commits" guarantee the skill claims to enforce. This
is exactly the class of finding the shipped-surface neutrality principle
exists to catch — surfaced here by live-testing the release ritual on
aiwf's own repo.

## Shape (sketch)

Generalize step 1's CI-green check to reference the project's own primary CI
workflow and language/build-input files, not Go's:
- Replace "Go-affecting commit" / "primary Go workflow (e.g. `go.yml`)" with
  language-neutral phrasing (e.g. "the project's primary CI workflow"),
  letting the ritual ask or infer which workflow and which build-input globs
  matter for *this* consumer instead of hardcoding Go's.
- Replace the `'**/*.go' 'go.mod' 'go.sum'` grep and `go test ./...` mentions
  with the same project-supplied equivalents.
- Drop the `` `release(aiwf): vX.Y.Z` `` example in favor of a generic
  placeholder consistent with step 4's own `docs(changelog): vX.Y.Z` shape.
- GitHub / `gh` CLI usage elsewhere in the skill is out of scope — assuming
  GitHub as the host is an accepted project decision.
