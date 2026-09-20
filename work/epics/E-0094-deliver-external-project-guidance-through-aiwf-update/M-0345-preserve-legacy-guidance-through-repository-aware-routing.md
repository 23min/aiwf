---
id: M-0345
title: Preserve legacy guidance through repository-aware routing
status: draft
parent: E-0094
depends_on:
    - M-0344
tdd: required
---
## Goal

Make ai-dotfiles safe for a machine containing both migrated and unmigrated repositories before aiwf starts handing over guidance ownership.

## Closes

- (none)

## Context

The corpus and installed-index ownership shape are defined. ai-dotfiles currently concatenates all modules for Codex and writes per-project Claude language imports. Compatibility must be delivered through its existing bootstrap, without requiring downstream users to upgrade aiwf.

## Acceptance criteria



## Constraints

Use a short routing instruction and an ownership check, not launcher wrappers, plugins, or a repository registry. This is assistant-directed loading, not a native conditional import. Old sessions require restarting. Legacy distribution remains until an explicit future compatibility decision authorizes removal.

## Design notes

E-0094 defines per-repository ownership and compatibility. The bootstrap must update its routing and required legacy content coherently; a failed bootstrap must retain a usable legacy setup.

## Surfaces touched

ai-dotfiles `build.sh`, `bin/dotfiles-sync`, its tests, guidance source modules, and bootstrap.

## Out of scope

aiwf project installation, removing legacy guidance support, and changing Copilot policy.

## Dependencies

- M-0344 — Publish the external engineering guidance corpus.

The external corpus and installed-index shape from the preceding delivery.

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
