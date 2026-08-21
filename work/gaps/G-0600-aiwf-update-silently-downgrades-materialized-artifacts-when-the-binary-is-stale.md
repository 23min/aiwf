---
id: G-0600
title: aiwf update silently downgrades materialized artifacts when the binary is stale
status: open
priority: high
---
## What's missing

`aiwf update` materializes each artifact from the running binary's embedded
snapshot without consulting what is already on disk. When the binary predates the
artifacts, the write moves their content backwards, and the report calls it
`updated`.

Measured on the date of filing, in this repo. The always-on guidance fragment had
just been extended and the clause was present in `.claude/aiwf-guidance.md`. The
`aiwf` on PATH was `v0.32.0`, built before that commit. Running `aiwf update` with
it printed:

    updated    .claude/aiwf-guidance.md  (materialized from embedded guidance)

after which the clause was gone. Re-running `update` from a binary built at current
source restored it, byte-identical to the copy taken beforehand. Nothing in the
report distinguished the two runs.

The data a guard would need already exists for one family. The fragment's header
comment stamps the version that wrote it, in the form
`aiwf-version: <version>`. Nothing reads that stamp back. The verb skills carry no
stamp at all — their frontmatter is name and description only.

## Why it matters

The guidance fragment is `@`-imported into the consumer's `CLAUDE.md`, so a reverted
copy governs an assistant's behaviour for the whole session. It is gitignored, so
neither review nor git history shows the revert. The operator's only signal is a
line reading `updated`.

The tool routes the operator into it. Run against the same tree, `aiwf doctor`
reported five verb skills drifted with `aiwf update` named as the remedy, while
reporting the guidance fragment healthy. Following that remedy with the binary
that printed it is what reverts the fragment. The instruction is correct about the
skills and destructive for the guidance, and the report gives the operator nothing
to tell those apart.

The condition is routine rather than exotic. This repo's worktree-binary discipline
puts a diagnostic build alongside the installed one deliberately, so two binaries of
different ages are reachable by design, and the documented discipline covers
`check` and `doctor` — where a stale binary yields a wrong answer the operator may
question — but not `update`, where it yields a success message and older bytes.

A reading on which this is working as designed is available: `update`'s contract is
to regenerate from this binary, and regenerating is what it did. That reading is
what makes the failure quiet. A verb whose contract permits it to destroy newer work
while reporting success should say so at the moment it does, whatever its contract.

## Resolution shape

Have `update` compare before it writes, and refuse or report a downgrade rather than
narrating it as `updated`. Refusal wants an override for the case where the operator
genuinely intends to roll artifacts back.

Two things to settle while doing it.

Comparing versions is not a plain ordering. The stamp observed carried a release tag
in one binary and a pseudo-version with a `+dirty` suffix in the other, and the dirty
dev build was the newer of the two. A naive string or semver comparison gets the
aiwf-repo case backwards, which is the case where the trap fires most.

Stamping is uneven. Only the guidance fragment carries a version today, so a
comparison covers one family and the rest stay silent. Extending the stamp to the
other materialized families is the same change or an explicit non-goal, not an
oversight to discover later.

Worth pinning the property rather than only the fix: materialize an artifact from
one version, run `update` from an older one, and assert the run does not silently
replace it.

The doctor remedy line belongs in the same pass. A report that names drifted skills
and prints `aiwf update` as the fix should not be the thing that reverts a fragment
it reported healthy.

## Related

- G-0504 — `doctor` byte-checks only the verb skills, so ritual and guidance drift
  read as healthy. That gap is about detection and records the newer-artifact-than-
  binary direction as an observation; this one is about what `update` does when it
  writes. Detecting the drift would not have prevented the revert, and the doctor
  line quoted above is the surface where the two meet.
- G-0471 — the binary-versus-source axis: no chokepoint detects a verb run by a
  binary older than the worktree's source. That is the condition; this is one
  consequence of it that carries data loss.
