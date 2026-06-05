---
id: G-0197
title: cellcoverage fixture stamps fictional aiwf-branch trailer value
status: open
discovered_in: M-0103
---
## What's missing

`internal/cellcoverage/AuthorizeScope` (`internal/cellcoverage/authorized_scope.go:34`) stamps a fictional ritual branch name into `verb.AuthorizeOptions.CurrentBranch` to satisfy M-0103's preflight:

```go
CurrentBranch: "epic/" + entityID + "-cellcoverage-fixture",
```

The verb's preflight then promotes this to `opts.Branch` (`internal/verb/authorize.go:286-292`), so the authorize commit carries an `aiwf-branch:` trailer with that fictional value — e.g. `aiwf-branch: epic/M-0001-cellcoverage-fixture`.

The branch by that name does not exist in the test repo. The fixture's job ends at producing an authorized scope; subsequent commits in cell-coverage tests happen on whatever branch the test repo is on (typically `master`).

## Why it matters

M-0106 (the kernel `isolation-escape` finding) walks back from each AI-actor commit to the most-recent-active scope and asserts the commit's branch is reachable from `refs/heads/<scope-aiwf-branch>`. With the fictional trailer value:

- Scope's `aiwf-branch` = `epic/M-0001-cellcoverage-fixture` (no such ref)
- AI-actor commit's branch = `master` (the fixture's test repo)
- Mismatch → `isolation-escape` fires on every cell-coverage test that lands an AI-actor commit under the fixture's scope.

The bug pattern is fully general: any future kernel rule that polices "aiwf-branch values resolve to real refs" or "commit branch matches scope branch" will surface this. Today no rule does, so cell-coverage tests pass.

## Fix shape (TBD in M-0106 design)

Three plausible directions:

1. **Stub the branch into existence** — `git branch <fictional-name>` inside the fixture's test repo so `refs/heads/<aiwf-branch>` resolves. Cheap; preserves the trailer-emission contract M-0103 mandates.
2. **Skip the trailer in fixture mode** — have the fixture use `Force=true` + a fixture-reason so the preflight is bypassed and no `aiwf-branch:` trailer lands. Cleaner but adds `aiwf-force:` to fixture commits, which other tests asserting "no force trailers" might break.
3. **Make the fixture cut the actual branch and check it out** — the fixture becomes self-consistent: the branch name in the trailer matches the test repo's HEAD.

M-0106's work is the natural place to choose; the trade-off depends on how strict the kernel finding is.

## Scope target

E-0030 / M-0106 — kernel finding `isolation-escape` ships here. Surfacing this in M-0103 (the preflight that introduces the trailer-emission semantic) so M-0106 picks it up at design time.
