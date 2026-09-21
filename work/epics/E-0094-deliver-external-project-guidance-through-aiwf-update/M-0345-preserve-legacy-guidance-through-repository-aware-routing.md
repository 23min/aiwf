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
      tdd_phase: done
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
- ai-dotfiles maintainers refresh and commit legacy copies from the canonical engineering-guidance repository. Consumers install those committed copies; downloading the corpus is not a bootstrap dependency. Personal collaboration rules remain maintained in ai-dotfiles.
- Foreign instruction files, symlinks, and personal guidance source locations remain active until their owner reconciles them. Installation refuses before rebuilding live outputs or rewiring those locations. Generated files explicitly owned by ai-dotfiles remain managed outputs.
- Compatibility is checked against Claude/Codex delivery visible in the invoking environment, irrespective of aiwf's selected artifact hosts. No ai-dotfiles installation is required when none is present. The signal covers the documented delivery contract, not other environments, arbitrary personal imports, Copilot, or already-running sessions. The executable contract and remediation are documented in the ai-dotfiles README.

## Validation

### Compatibility bootstrap and distribution

Observed on 2026-09-20 in the Linux devcontainer as an unprivileged user, using the ai-dotfiles milestone checkout. Installation tests use isolated homes and checkout fixtures; no production home installation was performed.

- Command: `sh test/bootstrap-preservation.test.sh`. Expected foreign instructions and source locations to remain active, conflicts to refuse before rebuilding live symlink targets, repeated installation to retain settings and legacy content, copy failure to preserve Claude instructions, read-only shared configuration to remain unchanged, and profile directories to be respected. Observed every assertion passing.
- Command: `sh test/refresh-guidance.test.sh`. Expected the complete canonical set to reach legacy paths with personal content retained, stable repeat refreshes, and unavailable, missing, empty, or linked upstream input to leave previous copies and source revision intact. Observed every assertion passing against a local Git source without network dependency.
- Command: `sh test/guidance-check.test.sh`. Expected absent or unrelated personal delivery to need no ai-dotfiles installation; old global routing, unavailable personal documents, stale home/PATH launchers, unresolved instruction files, and inaccessible ancestors to refuse compatibility. Expected effective profiles and nonempty Codex override precedence to select the inspected files. Observed every assertion passing, including empty linked overrides and launcher-only installations.
- Commands: each `test/*.test.sh` suite, `sh -n` over changed shell scripts, and `git diff --check`. Expected no regressions, syntax errors, or whitespace errors. Observed every shell suite passing and both checks clean. Doctor tests also confirm profile-directory selection and its explicit manifest override.
- Command: `./refresh-guidance.sh`. Expected a fresh default-branch download from the public canonical repository, with the imported revision recorded in `guidance-source.txt`. Observed a successful refresh without changes to the existing engineering copies. Independent comparison against the recorded canonical revision confirmed the declared legacy path mapping and rubric-link adaptation.
- Vacuity: isolated-checkout mutations removed foreign-file protection, moved rebuilding ahead of preflight, wrote directly over the live Claude manifest, bypassed routing and synchronization declarations, ignored the Codex profile, published refresh files before complete validation, and omitted the source revision. Expected each broken implementation to fail its relevant suite. Observed each mutation caught; production working files were not mutated.

Independent review findings about override selection, ancestor accessibility, shared-directory writes, PATH-only synchronization, and personal-document type were reproduced and pinned in the tests above. The final full-surface review approved the implementation; its follow-up confirmed the final branch-audit test additions. Fresh-session assistant reads remain unverified.

## Deferrals

- (none)

## Reviewer notes

- (none)
