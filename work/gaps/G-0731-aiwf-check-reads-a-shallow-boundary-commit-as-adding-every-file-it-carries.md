---
id: G-0731
title: aiwf check reads a shallow boundary commit as adding every file it carries
status: open
discovered_in: M-0350
---
## What's missing

In a shallow clone, `aiwf check` reads a commit whose parent lies beyond the clone's boundary as adding every file it carries, because the commit is diffed against the empty tree. Two rules that judge the audited range read it that way: the instruction-file fence, through the `git log --root` walk in `check.WalkInstructionCommits` (`internal/check/instruction_fence.go`), and the untrailered-entity audit, through `ReadUntrailedCommits` (`internal/cli/check/provenance.go`).

Measured 2026-10-04 in the aiwf devcontainer, git 2.51.1, with `aiwf` built from `5cf5fc43a`. The upstream's `main` holds `CLAUDE.md`, `app.go` and an epic created with its verb trailers; its `feature` branch adds one commit that changes only `app.go`. The clone is `git clone --depth 1 --branch feature --no-single-branch`, then `git fetch --depth 1 origin main`, so both tips are listed in `.git/shallow`:

```
$ aiwf check --since origin/main
CLAUDE.md: error instruction-commit-entity-missing: commit d7dfcf30 changes handwritten instructions (CLAUDE.md) but carries no aiwf-entity trailer, …
app.go: error instruction-commit-unrelated-file: commit d7dfcf30 changes handwritten instructions (CLAUDE.md) and also app.go, …
work/epics/E-0001-probe/epic.md: error instruction-commit-unrelated-file: commit d7dfcf30 changes handwritten instructions (CLAUDE.md) and also work/epics/E-0001-probe/epic.md, …
error provenance-untrailered-entity-commit: commit d7dfcf3 touched E-0001 with no aiwf-verb: trailer — …
isolation-escape-shallow-clone (warning) × 1 — isolation-escape coverage is incomplete: this repository is a shallow clone …
5 findings (4 errors, 1 warnings)
```

The same branch in a full clone reports none of the four errors.

## Why it matters

A CI job that checks out a branch shallowly, so that the branch tip is a shallow boundary, and fetches its base to name it in `aiwf check --since <base>` fails on errors the commit did not cause, however deep the base is fetched. The fence reports every file as riding beside an instruction edit and asks for a trailer and a split; the untrailered audit reports every entity file as edited without a verb. `isolation-escape-shallow-clone` says that coverage is incomplete, not that these errors are false, so the operator is sent to repair commits that are sound.
