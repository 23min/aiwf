---
id: M-0348
title: Verify migration and establish the reduction prerequisite
status: in_progress
parent: E-0094
depends_on:
    - M-0344
    - M-0345
    - M-0346
    - M-0347
tdd: advisory
acs:
    - id: AC-1
      title: aiwf uses tracked guidance without superseded imports
      status: met
    - id: AC-2
      title: Both hosts demonstrate relevant project reads in fresh sessions
      status: open
    - id: AC-3
      title: Mixed repositories retain the correct guidance source
      status: open
    - id: AC-4
      title: Growth measurements make the reduction prerequisite reproducible
      status: open
---
## Goal

Migrate aiwf itself, verify both hosts and legacy coexistence, and leave E-0092 a measured post-delivery starting point.

## Closes

- (none)

## Context

Corpus, project installation, and selection are available. Reconcile ai-dotfiles with personal-only global instructions and repository-local engineering delivery before shared installation handover. Reuse the existing synchronization and aiwf materialization paths; do not introduce another delivery framework.

## Acceptance criteria

### AC-1 — aiwf uses tracked guidance without superseded imports

Select the packs required by its actual languages and engineering conventions, install through the new update path, preserve repository-specific rules, and remove superseded managed delivery. Verify tracked files in a clean checkout and that dotfiles synchronization does not restore legacy imports. References: `aiwf.yaml`, `.guidance/`, `AGENTS.md`, `CLAUDE.md`, and migration command records. Broad prose reduction remains E-0092's work.

### AC-2 — Both hosts demonstrate relevant project reads in fresh sessions

Observe root-started tasks touching nested files, new files, and unrelated prose, in a checkout without ai-dotfiles. Include non-coding tasks outside repositories and verify that personal-only globals require no engineering discovery or reads. Record project-override precedence, relevant reads, absence of legacy engineering reads, and limits of observable behavior. References: milestone observation record with tasks, expectations, actual observations, environment, host/model versions, and installed revision. Obtain separate approval for live service invocations.

### AC-3 — Mixed repositories retain the correct guidance source

Complete the ai-dotfiles integration and test personal-only global Claude/Codex outputs, repository-local legacy engineering delivery, and aiwf handover checks. The ai-dotfiles maintainer owns its configuration and synchronization mechanism; use its [installation documentation](https://github.com/23min/ai-dotfiles#readme) as the integration reference. Account for globally installed engineering skills as well as instruction files. Verify ai-dotfiles synchronization prepares legacy repositories on use; do not bulk-modify sibling repositories. Before any approved removal of shared global engineering delivery, verify startup setup in the environments using it, including native hook trust and enablement. Verify its diagnostics for missing helpers and failed synchronization. Test that installation and opening one repository leave unopened repositories untouched, and that sessions outside Git create no project files. Do not infer startup readiness from hook-file existence or the handover compatibility check. Exercise the released combination against old-aiwf and non-aiwf projects alongside migrated projects. Include an incompatible personal installation, failed handover, empty installed selection, disabled maintenance, and update retries; confirm legacy repository-local access or exclusive aiwf project access as applicable. Preserve personal rules; do not treat a global project/legacy router as a completed migration. References: compatibility/integration fixtures plus an observation record for actual installed setup. Do not substitute local fixture success for distribution availability.

### AC-4 — Growth measurements make the reduction prerequisite reproducible

Record before/after commits, installed guidance revision, commands, expected outputs, observed growth results, and environment. Update E-0092 to identify E-0094 and its completed migration as the prerequisite; E-0092 still owns its own frozen behavioral baseline and ceiling. References: `docs/design/growth.md`, `scripts/growth-report.py`, and E-0092. Report global/personal loading separately from upfront and task-loaded project instructions.

## Constraints

TDD is advisory for adoption and live observations; new synchronization, generation and compatibility logic uses test-first development. Do not mark observational criteria met with file-existence proxies. Keep normal approval gates for commits and publication.

## Design notes

E-0094 supplies delivery and migration. E-0092 supplies the subsequent reduction; moving guidance is not by itself proof of lower instruction load.

## Surfaces touched

ai-dotfiles personal instruction generation, repository synchronization and compatibility checks; aiwf project configuration and host files, installed guidance, growth records, and E-0092 prerequisite text.

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

- ADR-0052 — project guidance ownership and the personal-bootstrap boundary.

## Validation

### AC-1 — migration observation

Observed on 2026-09-22 in the Linux devcontainer. Migration commit:
`9f817fa5b34cdcc4e0a065f7f7e78a54a030ad83`. The diagnostic binary
`/tmp/aiwf-m0348-review --version` reported
`v0.37.1-0.20260922200201-4bd58809979e+dirty`; installed ai-dotfiles was
`73d46b2f54821a0b03187b04343347248e08540b`.

- **Installation:** `/tmp/aiwf-m0348-review update --root /tmp/m0348-ac1-1dev6k_w/repo`
  was run in a disposable local clone with the selected packs. Expected successful
  project installation and replacement of managed legacy imports; observed exit 0
  and installed corpus revision `9c9ca4b3681ca4cb8798e5af9e9124b1e761526b`.
  The reviewed migration outputs were applied to the authoring checkout. Its
  pre-existing native workflow block and ignore-file edits remained uncommitted.
  Statusline refresh was skipped because the diagnostic version could not be
  ordered against the installed release; it was not part of this migration.
- **Selection:** tracked sources include Go, Python scripts, and Playwright
  TypeScript. The JavaScript suggestion matched Playwright's `package.json`,
  without tracked JavaScript source; no additional JavaScript pack was selected.
  Project exceptions retain `CLAUDE.md` as their canonical source through
  `.guidance/project.md`.
- **Handover:** from the authoring checkout, ran
  `~/.local/bin/dotfiles-sync --root "$PWD"` and
  `~/.local/bin/dotfiles-sync --check --root "$PWD"`.
  Expected no legacy restoration; both exited 0 and reported project guidance
  ownership with legacy synchronization skipped. Python SHA-256 assertions over
  the root files, configuration, ignore file, and every `.guidance` file passed
  before/after byte equality. Root-file assertions found no legacy engineering
  imports. Original pending content matched its saved SHA-256 values.
- **Clean delivery:** `git clone --quiet --no-hardlinks --single-branch` of the
  authoring checkout into `/tmp/m0348-ac1-committed-klktnnkc/repo` completed
  successfully. Expected a clean checkout containing the committed guidance;
  `git status --porcelain` was empty. Python assertions verified each installed
  index/pack SHA-256 against `.guidance/.aiwf-owned`. Every pack also byte-matched
  `git show <installed-revision>:<pack-path>` in the corpus checkout. An export of
  the candidate Git tree, with no personal dotfiles copied, had resolving local
  guidance links, both host routes, project overrides, and no pending installation.
  `git rev-parse HEAD^{tree}` after committing matched that verified candidate tree.
- **Review and checks:** independent review approved the exact migration patch
  with no findings. `git diff --check` passed;
  `/tmp/aiwf-m0348-review check --since origin/main` reported 0 errors and the
  existing 17 warnings. No Go/build inputs changed, so the full code suite was
  not repeated for this configuration and generated-content adoption.

These observations establish delivery and handover in the migrated checkout.
They do not establish live assistant reads, migration of other checkouts, or
instruction-load reduction; those remain separate criteria.

## Deferrals

- (none)

## Reviewer notes

- (none)
