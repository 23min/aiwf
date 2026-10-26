---
id: G-0750
title: aiwf list reports the checked-out branch as a cross-branch ref
status: open
---
## What's missing

`aiwf list` reports the checked-out branch as a cross-branch ref. `trunk.ScanCrossBranch` (`internal/trunk/trunk.go:488`) reads every local branch ref, the checked-out branch included, and identifies each entity by the id encoded in its path (`idsFromPaths`, line 246). `aiwf list` then adds a cross-branch row for every such id the loaded tree does not hold (`internal/cli/list/list.go:382`, with the per-id skip at lines 404-407). The loaded tree keys an entity by its frontmatter id, and it holds only what is in the working tree. So the checked-out branch shows up as a cross-branch source whenever its committed tree has an id the working tree lacks under that id.

Reproduced 2026-10-26 with aiwf built from `1e8e17e6c` (it reports `v0.42.0+dirty`), git 2.51.1, on Linux. On `main` of a repository seeded by `setup` in `docs/research/id-allocation/experiments/scripts/lib.sh` (one epic, milestone and gap), a commit adds a gap file `work/gaps/G-1234-2-second-in-request.md` whose frontmatter says `id: G-1234-2`. `aiwf check` reports two errors for this file, `frontmatter-shape` and `id-path-consistent`. Then:

```
$ aiwf list --kind gap --format=json | jq -c '.result[] | {id,path,cross_branch_ref}'
{"id":"G-0001","path":"work/gaps/G-0001-seed-gap.md","cross_branch_ref":null}
{"id":"G-1234-2","path":"work/gaps/G-1234-2-second-in-request.md","cross_branch_ref":null}
{"id":"G-1234-2","path":"work/gaps/G-1234-2-second-in-request.md","cross_branch_ref":"refs/heads/main"}
```

The same file under the name `G-1234-2-second-in-mr.md` (`c_shapes.sh`, row `mr-dash-sub`) shows the same labelling in a warning. A citation of `G-1234-2` in another gap's body, on `main`, is reported as `body-prose-id (warning) … id "G-1234" known only on refs/heads/main (not yet merged into this branch)`.

In a second case, an entity file is deleted from the working tree but not committed. `aiwf list` on `main` then lists that entity with `"cross_branch_ref":"refs/heads/main"`.

## Why it matters

`aiwf list --format=json` is the listing that agents and scripts consume. In the first case it returns two entities where there is one, and it labels the second as living on another branch. The body-prose warning says the id is "not yet merged into this branch" while that branch is the one checked out. Both describe a state that needs fixing in terms of a branch relationship that does not exist.
