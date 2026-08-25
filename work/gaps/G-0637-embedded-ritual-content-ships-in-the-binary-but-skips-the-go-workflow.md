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

Those bytes are test inputs, not just payload, and the coupling reaches the
prose itself. `internal/policies/cheap_fix_escape_test.go:70` locates the
wrap-milestone ritual's step-4 section by scanning `### ` headings for the
literal `wrap-side sections`; the heading it depends on is
`aiwfx-wrap-milestone/SKILL.md:75`. Rename that heading in a markdown-only
commit and the helper returns nothing, every assertion over that section
fails, and no Go workflow runs to say so.
`d0054_fixed_and_pinned_disposition_test.go` and `aiwfx_handoff_test.go` reach
the same ritual through the shared helper in `skill_fixture_helpers_test.go`.

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
`gitleaks`, `link-check`, `markdown-lint` and `scrub`. `f875c1bf1` edited the
step-4 region those tests parse; `9cdf7ac4f`'s change to the same file landed
further down, in the worktree-guard section. Both were green, but only the
first is known to have had `make ci` run against it before the push — whether
the other did is not recorded anywhere the tree can show, which is the point.

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

Its cost is smaller than it first looks. Widening the filter runs the full
matrix — build, vet, race, lint, coverage gates — on every commit touching
embedded content, and measured 2026-08-25 over the last hundred commits on
`main`, four did:

```
git log --oneline -100 --format='%h' | while read -r c; do
  git show --name-only --format='' "$c" | grep -q '^internal/skills/embedded' && echo x
done | wc -l
  4
```

At that rate the added CI is roughly one commit in twenty-five, which makes the
narrower alternative — a separate workflow running only
`go test ./internal/policies/` on embedded-content paths — a poor trade: it
saves little and leaves a second workflow to keep in step with the first.
Re-measure before deciding, since a stretch of ritual-heavy work moves the
rate.

Worth checking during the fix rather than assumed here: whether the filter
should name the embedded trees explicitly or `internal/skills/**` wholesale.
The wider pattern needs no maintenance when a new embedded tree is added; the
narrower one states its intent and does not fire on ordinary Go edits under
that package, which the `**/*.go` pattern already covers.

## Where to fix

- `.github/workflows/go.yml` — the `push` and `pull_request` path filters.
- `scripts/git-hooks/pre-push` — only if the decision is that this class should
  also be caught before the push rather than on it.
