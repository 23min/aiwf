---
id: ADR-0054
title: Adopt applicable guidance packs automatically on update
status: proposed
---
> **Date:** 2026-09-24 · **Decided by:** Peter Bruinsma

## Context

aiwf delivers engineering guidance packs from an external catalogue as tracked
`.guidance/` files, routed into each selected host's instruction file. Under
D-0089 and the E-0094 delivery, a pack became project policy only through
explicit selection: `aiwf update` detected applicable packs and prompted on an
interactive terminal, and a noninteractive run printed suggestions and adopted
nothing.

In practice that makes adoption depend on how update was launched. `aiwf
upgrade` re-executes `update` in place, so an upgrade run from a script, a pipe
or an agent session has no terminal to prompt on. It prints a suggestion to
stderr and leaves the repository without `.guidance/`. A consumer upgrading to
the release that shipped delivery found no `.guidance/` afterwards and had to
work backwards from an `aiwf doctor` line to learn that the feature was sitting
unadopted. Guidance that has to be discovered and opted into by hand reaches
few repositories, and the ones it misses are the ones whose maintainers never
ran update at a terminal.

Host selection already works the other way: with `hosts` unset, aiwf detects
hosts on every run and acts on the result. Guidance cannot copy that model
exactly. Hosts only decide which untracked artifacts are written, while
guidance writes tracked files. Re-detecting the whole set on every run would
make the installed guidance change with the repository's contents from run to
run, with no single change a maintainer reviews as "we adopted this".

Alternatives considered:

- **Keep explicit selection, make it louder.** A prominent summary and an
  adoption flag. Rejected: it improves the signal but leaves every upgrade
  ending without guidance until someone acts, which is the friction this
  decision removes.
- **Detect on every run without recording.** Mirror `hosts` exactly. Rejected
  for the tracked-output drift described above.
- **Adopt once on the first run, then only suggest.** Rejected: a language
  added later would sit as a suggestion, recreating the original problem for
  every repository that grows.

D-0089's concern stands: a pack can carry an opinionated toolchain, and
detecting a language is not the maintainer choosing that toolchain. The answer
here is that adoption is a working-tree change aiwf never commits, pack names
disclose their tooling, and declining a pack is one recorded, durable choice.

## Decision

`aiwf init`, `aiwf update` and, through it, `aiwf upgrade` adopt applicable
guidance packs automatically.

- **Add-only adoption on every run.** Each run with guidance maintenance
  enabled adds every catalogue pack whose detection matches the repository and
  that is not listed in `guidance.ignored` to `guidance.packs`, installs it,
  and writes host routing. A pack is never removed automatically; a selected
  pack with no current matches is retained.
- **No guidance prompt.** Adoption behaves the same on a terminal, in a
  script, in CI and under `aiwf upgrade`. Declining a pack means listing its
  id in `guidance.ignored`; removing an adopted pack means moving its id there.
- **Explicit report on every run.** Update ends with a guidance section
  stating what it adopted and installed and at which source commit, what was
  already installed, what is ignored, and anything that blocked installation
  together with the action that clears it.
- **Explicit configuration still wins.** `guidance.enabled: false` stops
  detection, adoption and refresh. Existing entries in `guidance.packs` and
  `guidance.ignored` are honoured as written.
- **Unchanged from D-0089 and ADR-0052.** Language and engineering content is
  authored outside aiwf and delivered as tracked project files readable without
  aiwf. The ownership, compatibility check and legacy-handover rules of
  ADR-0052 apply to automatically adopted packs exactly as to selected ones.

## Consequences

- An upgrade leaves a repository with `.guidance/`, its index and host routing
  whenever the catalogue has an applicable pack and handover is compatible, and
  a later update picks up packs for languages the repository gains.
- Adoption edits `aiwf.yaml`, `.guidance/` and host instruction files in the
  working tree without asking. Nothing is committed, so the maintainer reviews
  the diff before it becomes policy; this is the same consent model ADR-0018
  applies to the host guidance import.
- A stray file can pull in a pack the maintainer does not want. The cost is one
  `guidance.ignored` entry, recorded once.
- Repositories still on ai-dotfiles legacy delivery reach handover on their
  next update. Recognized generated legacy imports are replaced; handwritten
  legacy references and an unavailable compatibility checker block handover,
  and the report names the fix.
- Adoption depends on the catalogue's detection patterns, including any pack
  that matches every repository. What gets adopted can change when the
  catalogue changes, and each such change arrives as a reviewable diff.
- The select / not now / ignore prompt from E-0094 is removed. The
  noninteractive suggestion report is folded into the guidance report.
- D-0089 is superseded by this record. Its content-ownership rule is carried
  forward above.

## Validation

Re-examine this decision if maintainers routinely add entries to
`guidance.ignored` right after an upgrade, which would mean detection is
adopting packs projects do not want, or if the catalogue gains overlapping
packs for one language whose conventions conflict when both are adopted.

## References

- D-0089 — language guidance is externally owned and delivered as project
  policy; superseded by this record.
- ADR-0052 — project guidance ownership and legacy handover, retained.
- ADR-0018 — consent for user-owned file edits.
- E-0094 — external project guidance delivery; M-0347 — the suggestion
  prompt this record removes.
