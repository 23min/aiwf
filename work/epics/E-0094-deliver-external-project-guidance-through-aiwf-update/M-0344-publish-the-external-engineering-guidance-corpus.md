---
id: M-0344
title: Publish the external engineering guidance corpus
status: draft
parent: E-0094
tdd: advisory
---
## Goal

Establish the plain Markdown corpus and minimal catalogue that both legacy distribution and aiwf delivery can consume, without changing downstream loading behavior.

## Closes

- (none)

## Context

E-0094 selects `23min/engineering-guidance` as the canonical source. The maintained ai-dotfiles checkout contains the source material and detector. Inventory those sources before packaging; the selected repository's remote availability remains unverified.

## Acceptance criteria



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
