---
id: G-0637
title: Embedded ritual content ships in the binary but skips the Go workflow
status: open
---
## What's missing

The embedded ritual, skill, guidance and statusline trees are compiled into the
binary, and a change to any of them reaches consumers. No automated gate runs
the Go tests when one changes.

`internal/skills/skills.go:41` and `:50` carry `//go:embed embedded` and
`//go:embed embedded-rituals`; `guidance.go:14` and `skills.go:69` embed the
guidance fragment and the statusline. So a markdown-only commit under those
trees produces a different binary.

`.github/workflows/go.yml` — the workflow that runs build, vet, race tests,
lint and the coverage gates — filters both its `push` and `pull_request`
triggers to `**/*.go`, `go.mod`, `go.sum`, `.golangci.yml`,
`.github/workflows/go.yml` and `Makefile`. None of the embedded trees is
listed, so a commit touching only them does not start it.

Those bytes are test inputs, not just payload. Policy tests parse them —
`internal/policies/skill_fixture_helpers_test.go`,
`m0210_trailer_commit_drift.go` and `aiwfx_handoff_test.go` all read the
wrap-milestone ritual, and the wider suite reads the other embedded skills. A
prose edit can turn the policy suite red.

Nothing else covers the class. The pre-push hook chain runs `aiwf check`
(`.git/hooks/pre-push`), then `golangci-lint`, `gitleaks`, and exactly one
targeted policy test — `TestPolicy_CommentHistoryAttrition`
(`scripts/git-hooks/pre-push:89-91`). It does not run the policy suite.

Measured 2026-08-25. The last successful `go.yml` run on `main` was
`90346026a`. Two later commits changed embedded content —

```
git log --oneline 90346026a..main -- internal/skills/
  f875c1bf1 docs(rituals): put the wrap rituals' own prose in front of a reviewer (G-0635)
  9cdf7ac4f docs(rituals): prompt for what a closure just invalidated (G-0628)
```

— and `gh run list --commit <sha>` for the resulting HEAD lists only
`gitleaks`, `link-check`, `markdown-lint` and `scrub`. Both commits edited the
step-4 region of the wrap-milestone ritual that the policy tests above parse.
Both happened to be green, verified by an operator running `make ci` locally.

## Why it matters

The repo's stated model is that CI-on-push is the authoritative gate and local
`make ci` is pre-flight insurance. For this class the order is reversed: the
local run is the only thing that executes the tests, so the authoritative gate
is an operator remembering to run one.

The exposure is largest exactly where it is least visible. A ritual edit looks
like a documentation change — it touches only markdown, the doc workflows go
green, and the pull request or push reports success. Nothing in that signal
distinguishes it from a change that cannot affect the binary.

It also reaches a release tag. A tag is cut from `main`, and `main` can carry
embedded-content commits that no Go workflow has judged, so the shipped binary
can differ from any binary CI has tested.

## Resolution shape

The straightforward fix is to add the embedded trees to `go.yml`'s two path
filters, so the workflow that already knows how to judge these bytes starts
when they change. That is one edit to one file and introduces no new surface.

Its cost is the reason it is a decision rather than an edit: every ritual-prose
commit would then run the full matrix — build, vet, race, lint, coverage gates
— and ritual prose changes often. Whoever fixes this should weigh that against
the narrower alternative of a separate workflow running only
`go test ./internal/policies/` on embedded-content paths, which is cheaper per
commit but adds a second workflow to keep in step with the first.

Worth checking during the fix rather than assumed here: whether the filter
should name the embedded trees explicitly or `internal/skills/**` wholesale.
The wider pattern needs no maintenance when a new embedded tree is added; the
narrower one states its intent and does not fire on ordinary Go edits under
that package, which the `**/*.go` pattern already covers.

## Where to fix

- `.github/workflows/go.yml` — the `push` and `pull_request` path filters.
- `scripts/git-hooks/pre-push` — only if the decision is that this class should
  also be caught before the push rather than on it.
