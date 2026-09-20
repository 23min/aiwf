---
id: M-0345
title: Preserve legacy guidance through repository-aware routing
status: in_progress
parent: E-0094
depends_on:
    - M-0344
tdd: required
acs:
    - id: AC-1
      title: Global entry points choose one engineering guidance source
      status: met
      tdd_phase: done
    - id: AC-2
      title: Synchronization preserves the installed project owner
      status: met
      tdd_phase: done
    - id: AC-3
      title: Compatibility distribution retains legacy content and personal settings
      status: open
      tdd_phase: green
    - id: AC-4
      title: Fresh sessions demonstrate legacy fallback and exclusive project reads
      status: open
---
## Goal

Make ai-dotfiles safe for a machine containing both migrated and unmigrated repositories before aiwf starts handing over guidance ownership.

## Closes

- (none)

## Context

The corpus and installed-index ownership shape are defined. ai-dotfiles currently concatenates all modules for Codex and writes per-project Claude language imports. Compatibility must be delivered through its existing bootstrap, without requiring downstream users to upgrade aiwf.

## Acceptance criteria

### AC-1 — Global entry points choose one engineering guidance source

Generated Claude and Codex instructions keep personal rules unconditional and route engineering reads to installed project guidance or legacy files. Include code-health in this boundary. Empty project selections and disabled maintenance retain project ownership; a missing project index retains legacy access. References: ai-dotfiles `build.sh`, source guidance modules, and generated instruction fixtures. Copilot behavior must not change incidentally when its current shared build input changes.

### AC-2 — Synchronization preserves the installed project owner

`dotfiles-sync` does not recreate legacy imports in an aiwf-owned repository and retains existing behavior elsewhere. Exercise managed index recognition, absent or unrelated indexes, nested working directories, foreign instructions, repeated runs, and shared-machine repositories with different owners. References: `bin/dotfiles-sync` and `test/dotfiles-sync.test.sh`. No automatic migration merely from a directory named `.guidance`.

### AC-3 — Compatibility distribution retains legacy content and personal settings

The bootstrap keeps legacy guidance readable at its existing locations while consuming the canonical corpus without a second independently maintained source. Existing personal preferences survive installation. Test the bootstrap in isolated homes and preserve other host outputs; document the compatible installation signal that aiwf's preflight can inspect. References: ai-dotfiles build/install paths, resolved at implementation start.

### AC-4 — Fresh sessions demonstrate legacy fallback and exclusive project reads

Record Claude and Codex reads in unmigrated, migrated, and explicitly empty-selection repositories on the same machine. Confirm personal instructions remain and the migrated case does not load the legacy engineering bundle. Test fixtures model the agreed installed-index shape. References: compatibility observation record in the milestone. File presence or assistant assertions alone do not establish the read path; report the actual observable evidence and limits.

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

- D-0097 — Unreadable project guidance requires operator direction before guidance-dependent work.

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
