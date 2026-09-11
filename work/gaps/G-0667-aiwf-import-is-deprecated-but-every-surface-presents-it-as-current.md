---
id: G-0667
title: aiwf import is deprecated but every surface presents it as current
status: open
discovered_in: M-0329
---
## What's missing

`aiwf import` is deprecated, and no artefact says so. Every surface a reader or
an AI can reach presents it as a current verb:

- It is wired into the root command (`internal/cli/root.go`) and builds, runs,
  and commits like any other verb.
- `aiwf import --help` describes the manifest and its flags with no deprecation
  notice.
- `internal/policies/skill_coverage.go` allowlists it as an "ops verb;
  bulk-create from manifest, --help suffices" — so the policy that would
  otherwise demand a skill points at the one surface carrying no notice.
- Two shipped skills instruct a consumer to use it.
  `internal/skills/embedded/aiwf-contract/SKILL.md` says to land contract
  entities via `aiwf import` as the migration path;
  `internal/skills/embedded/aiwf-check/SKILL.md` names it as a way a milestone
  reaches an already-terminal epic.
- `CLAUDE.md` asserts it still ships, as the reason the archived migration doc
  stays reachable.
- `docs/archive/migration/from-prior-systems.md` specifies its manifest
  contract.

The deprecation lives in the maintainer's intent and nowhere a check, a reader,
or an assistant can find it.

## Why it matters

A consumer following the shipped contract skill reaches for a verb that is not
going to be maintained, and nothing warns them. That is the failure the
AI-discoverability principle exists to prevent, inverted: the surfaces are
consistent and consistently wrong.

It also makes a correct scoping decision unreadable. `aiwf import` is the one
body-producing seam M-0329 left without a membership gate, and deprecation is
the reason. Written down nowhere, that reads as an oversight rather than a
choice, and the next reviewer re-raises it — as one already did.

Deciding what "deprecated" means here is part of the work: a `--help` notice and
a runtime warning, removal with a stated trigger, or the verb kept as-is with the
status recorded only in the design docs. The manifest contract it implements is
specified in an archived migration doc, so removal is not free.
