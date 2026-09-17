---
id: G-0691
title: check --shape-only and --fast pass a malformed aiwf.yaml the full check refuses
status: open
---
## What's missing

`internal/cli/check/check.go` reaches `aiwf.yaml` differently on its three paths.
The full path loads the tree through `cliutil.LoadTreeWithTrunk`, which refuses
with exit 3 when the file does not parse. `runShapeOnly` and `runFast` load the
tree through `tree.Load` and then guard their own `config.Load` with
`cfgErr == nil && cfg != nil`, continuing with the defaults otherwise: non-strict
tree loading, no `allow_paths`, an empty severity policy. `config.Load`
distinguishes a missing file (`ErrNotFound`) from a parse failure; the guard does
not consult that distinction.

Measured in the devcontainer (Linux) with a binary built from `ab430b80a`, in a
fresh repository after `aiwf init` and one commit, then
`echo "tdd: [unterminated" >> aiwf.yaml`:

```
$ aiwf check; echo exit=$?
aiwf check: loading tree: loading aiwf.yaml: parsing aiwf.yaml: yaml: line 46: did not find expected ',' or ']'
exit=3

$ aiwf check --shape-only; echo exit=$?
ok — no findings
exit=0

$ aiwf check --fast; echo exit=$?
ok — no findings
exit=0
```

Expected: the three paths agree that the configuration cannot be read. Observed: the
full path refuses; the shape-only and fast paths report a clean tree.

The defaults are also weaker than what the broken file asked for. Measured in the
same environment on a binary built from `7f2551bec`: with `tree:` / `strict: true`
in a valid `aiwf.yaml` and a stray `work/gaps/stray.txt`, both cheaper paths report
`error unexpected-tree-file` and exit 1; with the same file corrupted, both report
`warning unexpected-tree-file` and exit 0.

## Why it matters

The pre-commit hook runs `--shape-only`, so it is the signal an operator sees on
every commit. On a malformed configuration it reports a clean tree, and the first
refusal arrives at the pre-push hook, after the commits that trusted the green
signal have been made. The cheaper path is also the more permissive one: a
strict-mode setting or a severity override in the broken file is not applied, so a
finding the operator configured to block passes as a warning — the inverse of what
a cheaper approximation of the gate should do.
