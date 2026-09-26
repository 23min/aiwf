---
id: G-0715
title: Full CI skips epic and milestone pushes that CLAUDE.md counts on
status: open
discovered_in: M-0333
---
## What's missing

A push to an epic or a milestone branch runs none of the full CI workflows, while `CLAUDE.md` §"How to validate changes" says CI runs the full gate on every push and names CI on the epic-branch push, beside the pre-push hook, as a milestone wrap's safety net.

Where the defect shows: `CLAUDE.md` lines 84, 92 and 94.

```
$ grep -n -o "CI runs the full gate on every push\|CI-on-push when the epic branch is pushed\|The authoritative gate is CI-on-push" CLAUDE.md
84:CI runs the full gate on every push
92:CI-on-push when the epic branch is pushed
94:The authoritative gate is CI-on-push
```

Where a fix would land: the `push` triggers in `.github/workflows/`. Each is scoped to `main`; `go.yml` also names `poc/**`, the prefix of a PoC branch that merged into `main` on 2026-05-08 (`e0a7fe559`) and no longer exists, and filters its push trigger by path, so even a `main` push that changes no listed path runs no Go workflow. Gitleaks runs on every push and is not affected.

```
$ grep -n -A1 "^  push:" .github/workflows/{go,link-check,markdown-lint,scrub}.yml
go.yml:27:  push:
go.yml:28-    branches: [main, "poc/**"]
link-check.yml:12:  push:
link-check.yml:13-    branches: [main]
markdown-lint.yml:9:  push:
markdown-lint.yml:10-    branches: [main]
scrub.yml:12:  push:
scrub.yml:13-    branches: [main]

$ git ls-remote --heads origin 'poc/*'
(no output)
```

Observed 2026-09-26 in the devcontainer, after pushing an epic branch and a milestone branch whose ranges change Go files:

```
$ gh run list --branch epic/E-0092-shrink-the-always-on-guidance-to-one-home-per-rule-under-a-ceiling --json name,conclusion,headSha --jq '.[] | "\(.name) \(.conclusion) \(.headSha[0:9])"'
gitleaks success 5a8a63c55
$ gh run list --branch milestone/M-0333-fence-project-guidance-while-preserving-managed-updates --json name,conclusion,headSha --jq '.[] | "\(.name) \(.conclusion) \(.headSha[0:9])"'
gitleaks success 9ac951c97
```

## Why it matters

A milestone wrap runs `make check-fast`, not the full gate, with CI on the epic-branch push as half of its stated safety net. That half never runs, so the race detector and the rest of the full gate first run on an epic's work at the epic-to-`main` merge, after every milestone in it has wrapped, and a reader of `CLAUDE.md` trusts a check that does not happen.

The documentation checks are delayed the same way: a broken link, a markdown-lint failure or a forbidden identifier introduced on an epic branch is first reported on `main`.
