---
id: M-0348
title: Verify migration and establish the reduction prerequisite
status: draft
parent: E-0094
depends_on:
    - M-0344
    - M-0345
    - M-0346
    - M-0347
tdd: advisory
---
## Goal

Migrate aiwf itself, verify both hosts and legacy coexistence, and leave E-0092 a measured post-delivery starting point.

## Closes

- (none)

## Context

Corpus, compatibility routing, installation, and selection are available. This is adoption and observation work, not a second implementation of delivery.

## Acceptance criteria



## Constraints

TDD is advisory for adoption and live observations; any logic defect found uses test-first fixes. Do not mark observational criteria met with file-existence proxies. Keep normal approval gates for commits and publication.

## Design notes

E-0094 supplies delivery and migration. E-0092 supplies the subsequent reduction; moving guidance is not by itself proof of lower instruction load.

## Surfaces touched

aiwf project configuration and host files, installed guidance, growth records, and E-0092 prerequisite text.

## Out of scope

E-0092's compression, universal model-compliance guarantees, or changing personal collaboration preferences.

## Dependencies

- M-0344 — Publish the external engineering guidance corpus.
- M-0345 — Preserve legacy guidance through repository-aware routing.
- M-0346 — Deliver explicitly selected project guidance through update.
- M-0347 — Suggest applicable guidance during init and update.

All preceding deliveries, with the actual compatible distributions available for the observed environment.

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
