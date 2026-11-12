---
id: G-0764
title: flake-hunt on a non-main branch fails a policy test that needs origin/main
status: open
priority: medium
---
## What's missing

`TestPolicy_ThisRepoTreeHasNoDanglingReference`
(`internal/policies/tree_self_contained_test.go:35-37`) loads the tree with
`cliutil.LoadTreeWithTrunk`. That load errors when the repository has
remote-tracking refs but not the trunk ref `refs/remotes/origin/main`
(`internal/trunk/trunk.go:96-108`); a repository with no remote-tracking refs
skips the trunk read instead.

The `flake-hunt` job in `.github/workflows/flake-hunt.yml` checks out with
`actions/checkout`'s default depth-1 fetch (line 80), which fetches only the
dispatched commit, into `refs/remotes/origin/<branch>`. On `main` that is the
trunk ref: the scrub run on `main` at `8ef91d79c` (43296563739), whose checkout
takes the same default, fetched into `refs/remotes/origin/main`. On any other
branch, `origin/main` is absent and the test fails on every repeat.

Run 43292158635 was dispatched on `patch/actions-node24` at `778c2c0ab`. Its
checkout step ran

```
/usr/bin/git -c protocol.version=2 fetch --no-tags --prune --no-recurse-submodules --depth=1 origin +778c2c0ab6e3cf7d3e1144997565b3cf5b978d00:refs/remotes/origin/patch/actions-node24
```

and the test failed all 10 repeats with
`trunk ref "refs/remotes/origin/main" does not resolve in this repo`.

Reproduced at `8ef91d79c` (go1.25.12, git 2.51.1, Linux) with the same ref
layout, starting from the repository root:

```
$ R=$PWD; SHA=$(git rev-parse main); D=$(mktemp -d)
$ git init -q $D/c && cd $D/c && git remote add origin file://$R
$ git fetch -q --depth 1 origin +$SHA:refs/remotes/origin/other
$ git checkout -q -b other refs/remotes/origin/other
$ go test -count=1 -run '^TestPolicy_ThisRepoTreeHasNoDanglingReference$' ./internal/policies/
--- FAIL: TestPolicy_ThisRepoTreeHasNoDanglingReference (0.08s)
    tree_self_contained_test.go:37: loading tree with the cross-branch view: trunk ref "refs/remotes/origin/main" does not resolve in this repo; fetch it or set allocate.trunk in aiwf.yaml
FAIL	github.com/23min/aiwf/internal/policies	0.093s
```

After `git fetch -q --depth 1 origin +refs/heads/main:refs/remotes/origin/main`,
the same test prints `ok`. The first fetch is by commit id on purpose: fetching
`main` by name also writes `origin/main` through the remote's default refspec, and
a plain clone writes it too, so neither shows the failure.

## Why it matters

The workflow's header (lines 7–10) names a run before tagging a release as its
intended use, which runs on `main` and is unaffected. A run started on any other
branch, which `workflow_dispatch` allows and which is how a change to the
workflows or tests is checked before it merges, reports `internal/policies`
failed for a test that is not flaky, so that package cannot get a clean result
from a branch run.
