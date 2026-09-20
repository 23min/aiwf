---
id: M-0344
title: Publish the external engineering guidance corpus
status: draft
parent: E-0094
tdd: advisory
acs:
    - id: AC-1
      title: The catalogue describes packs without aiwf language code
      status: open
    - id: AC-2
      title: The initial corpus preserves existing engineering guidance
      status: open
    - id: AC-3
      title: The source can be consumed without aiwf tooling
      status: open
---
## Goal

Establish the plain Markdown corpus and minimal catalogue that both legacy distribution and aiwf delivery can consume, without changing downstream loading behavior.

## Closes

- (none)

## Context

E-0094 selects `23min/engineering-guidance` as the canonical source. The maintained ai-dotfiles checkout contains the source material and detector. Inventory those sources before packaging; the selected repository's remote availability remains unverified.

## Acceptance criteria

### AC-1 — The catalogue describes packs without aiwf language code

Catalogue entries identify Markdown documents, applicability descriptions, and simple marker/path/extension patterns. Validate unique ids, resolvable documents, safe paths, and names that disclose opinionated tooling; no executable hooks or dependency parsing. References: external catalogue and its validation fixtures; `/workspaces/ai-dotfiles/bin/dotfiles-sync` supplies existing detection rules to inventory.

### AC-2 — The initial corpus preserves existing engineering guidance

Map all existing engineering and language guidance to the external corpus, including code-health and the material used by aiwf. Compare against a recorded source revision, allowing only documented packaging, naming, and reference changes; personal collaboration, approval, machine, and session rules remain outside the corpus. References: ai-dotfiles `guidance/`, engineering-related skill sources, and the external corpus inventory. The comparison establishes migration fidelity, not permanent prose pins.

### AC-3 — The source can be consumed without aiwf tooling

The default branch supplies the documented catalogue and Markdown, with no aiwf runtime, package installer, registry, or planning tree needed to read them. Verify the local repository first and record remote retrieval after publication approval, including command, expectation, observation, and source revision. References: the external repository's README and catalogue. Do not claim the remote exists from a local fixture.

## Constraints

Keep the catalogue declarative and small. Preserve wording. Do not introduce new languages to meet an arbitrary coverage target. The installed index ownership marker and minimum routing semantics must be specified for the next delivery, including empty selections; avoid a general protocol framework.

## Design notes

E-0094 owns the agreed delivery behavior; D-0089 records the proposed ownership boundary. Choose the exact schema and pack-to-path mapping during this milestone, before implementing consumers.

## Surfaces touched

External repository Markdown, catalogue, README, and validation fixtures; ai-dotfiles source inventory.

## Out of scope

Changing global instructions, migrating any consumer, or implementing the aiwf updater.

## Dependencies

None. Publication requires separate approval.

## References

- E-0094 — delivery scope and compatibility requirements.

---

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
