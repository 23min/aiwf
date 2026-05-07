---
id: G-057
title: Stray aiwf binary in repo root from local builds is not gitignored
status: open
---

## What's missing

A bare `go build ./cmd/aiwf` (or any equivalent without `-o /tmp/aiwf`) drops an `aiwf` binary in the repo root. The repo's `.gitignore` already covers `/bin/`, `/dist/`, `*.test`, and `*.out` — but not a bare `aiwf` executable at the root. Result: the binary shows up in `git status` as an untracked file (currently visible in this branch's status), inviting accidental commit.

CLAUDE.md prescribes `go build -o /tmp/aiwf ./cmd/aiwf` for local validation specifically to keep the binary out of the tree, but the prescription depends on the operator (or LLM) remembering. A typo, a copy-paste, or `go install` wired to `GOBIN=$PWD` reproduces the problem.

## Why it matters

Two failure modes, both real:

- **`git add -A` captures the binary.** A several-megabyte ELF/Mach-O blob lands in a commit, the push goes out, and now the repo's history carries a binary nobody wanted. Rewriting history to remove it is expensive and disruptive.
- **The pollution interacts with G-056.** Combined with the un-gitignored `site/` from a render run, `git status` shows ~140 untracked entries — burying any actual changes the operator wants to review or commit.

The kernel's "framework correctness must not depend on LLM behavior" principle applies here too: the canonical fix is a `.gitignore` rule, not a documented convention.

## Possible remedies

1. **Add `/aiwf` to `.gitignore`.** Anchored to root with the leading slash so the kernel's source path `cmd/aiwf/` and any future package named `aiwf` are unaffected. Smallest possible change; closes the gap completely for this repo.
2. **Document the build command path more visibly.** CLAUDE.md already says `go build -o /tmp/aiwf ./cmd/aiwf`; reinforcing it in `aiwf doctor --self-check` output (a one-line hint when a stray `./aiwf` is present) catches drift in consumer repos that copy the build incantation but not the gitignore rule.
3. **Make `aiwf init` write `/aiwf` into the consumer repo's `.gitignore`** alongside the G-056 entry for `site/`. Only matters if a consumer also builds the binary in their repo (rare), but cheap to bundle into the same marker block.

The repo-root `.gitignore` edit (1) is the load-bearing fix for the kernel repo itself. (2) and (3) are belt-and-braces for downstream consumers.

## Related

- [G-056](G-056-aiwf-render-output-site-is-not-gitignored-pollutes-consumer-working-tree.md) — same class of defect: artifact produced by the framework with no `.gitignore` coverage. The init/update marker-block design proposed for G-056 is the natural carrier for this entry too.
