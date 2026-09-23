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

### AC-2 — host observations

Observed on 2026-09-22 and 2026-09-23 in the Linux devcontainer. Every run
started at the root of a clean clone of the migrated checkout at commit
`9f817fa5b34cdcc4e0a065f7f7e78a54a030ad83`, carrying no repository-local
ai-dotfiles, with installed guidance revision
`9c9ca4b3681ca4cb8798e5af9e9124b1e761526b`. The installed personal setup was
personal-only throughout: the global Claude instruction file carries the
personal-only marker and imports one collaboration fragment, and the legacy
language packs remain on disk beneath the personal agents directory, unimported.
Host versions: Codex CLI 0.156.0 with `gpt-6-astra`; Claude Code 2.1.278 with
`claude-sonnet-5`. Raw event streams and answers were captured to session-local
temporary directories, which do not survive the machine; what a later reader
needs to re-run is recorded here.

The nested-Go task, issued verbatim to both hosts: *plan a small change to
`internal/version/version.go` adding diagnostic logging around its remote
version lookup; do not implement, modify files, access the network or run
tests; inspect the repository as needed and explain where the diagnostic logs
go by default, how logging is enabled, and which validation commands apply;
list the instruction files actually consulted.* Expected in each case: the
session discovers the project override, the index and the packs relevant to the
task, applies the project logging rule over the general Go pack's stderr
convention, reads no legacy engineering guidance, and leaves the checkout
unchanged. Tool events, not the assistant's claimed read list, were the evidence.

- **Codex, nested Go.** `codex exec --cd <clone> --sandbox read-only
  --ephemeral --json --model gpt-6-astra -c model_reasoning_effort="medium"`.
  Exit 0 in 50.4 seconds. Completed-command events carry actual reads of
  `CLAUDE.md`, `.guidance/project.md`, `.guidance/index.md`, the code-health
  guide and the Go guide, followed by logger source and `go.mod`. The answer
  applies the project rule over the pack: logging off by default, enabled
  through the environment knobs or the YAML block, default XDG-state-home file
  destination, stderr only when explicitly configured. No observed command
  reads legacy engineering files. `git status --porcelain` stayed empty.
- **Claude, nested Go, effort `xhigh`.** `claude --print --output-format
  stream-json --verbose --permission-mode dontAsk --tools Read,Glob,Grep
  --allowedTools Read,Glob,Grep --strict-mcp-config --mcp-config
  '{"mcpServers":{}}'`, prompt on stdin, effort resolved from installed
  settings. The run did not complete: a 180-second outer timeout fired and the
  stream's terminating event is `error_during_execution` after 28 turns, so it
  reports nothing about the answer. The host injected three instruction files —
  the personal-only global, its collaboration fragment, and the project
  `CLAUDE.md` at 66,036 bytes carrying the managed engineering-guidance block.
  No legacy engineering pack was injected. No tool input named `.guidance`, the
  pending-install marker, or the legacy trees.
- **Claude, nested Go, effort `medium`.** Same argv with `--effort medium`,
  matching the Codex arm's reasoning effort; the prompt was byte-identical to
  the run above. Exit 0 in 85.4 seconds, 14 turns. Again no `.guidance` read.
  Project-override precedence nonetheless reached the answer correctly: opt-in,
  off by default, the environment knobs and YAML block as the enablers, the
  XDG-state-home daily file as the default destination, stderr as an explicit
  choice only, and the repository's own validation cadence. Asked which
  instruction files it consulted, it named the project `CLAUDE.md` and
  ADR-0017 and nothing else — the claim matches the tool events, so there is
  no overclaim to discount.
- **Claude, Python task, effort `medium`.** The same harness against a task
  whose answer the project instruction file does not contain: plan a `--json`
  output mode for `scripts/growth-report.py` and explain which formatting,
  linting, type-checking and test conventions apply. The checkout carries no
  Python tooling configuration, no Makefile target and no CI step for it, so the
  only source in the tree is the Python pack. Exit 0 in 29.4 seconds. Tool calls
  two through six of thirteen are the block's own sequence, executed before the
  answer formed: the pending-marker check, `.guidance/project.md`,
  `.guidance/index.md`, then the Python pack. The answer names that pack's
  toolchain, and then weighs it against the repository — observing that no
  configuration or target wires it here and that the script is in fact exercised
  by a Go-side policy test — rather than importing tooling the repository does
  not use.

Both hosts therefore demonstrate relevant project reads. On Claude the reads
are need-driven rather than unconditional: skipped where the project
instruction file already answers, which is what the managed block itself
prescribes when it gives handwritten overrides precedence over pack guidance,
and performed in the prescribed order where only a pack answers. Position in
the host file does not drive this. A fixture experiment — a throwaway clone
with the block moved from the end of the project instruction file to just below
its opening paragraph, every other input byte-identical to the second Claude
run — bought a pending-marker check but no directed reads, while the Python
task drew the full sequence from the block's committed position at the end of
the file. That fixture carries an edit no committed tree has and is recorded
here only for what it rules out.

Limits. Installed personal globals remained present: these are clones without
repository-local ai-dotfiles, not a machine lacking ai-dotfiles. Tool events
establish explicit reads, not instructions the host injects automatically.
Claude tool access was restricted to reading, globbing and grepping, so a read
outside the working directory would have been refused — but none was attempted,
so refusal masks nothing. The clones were never updated after cloning, so the
gitignored materialized host artifact the project instruction file imports was
absent and its import line stood unresolved; these sessions ran without aiwf's
own workflow guidance. A task straddling the two shapes, where the project
instruction file partly answers and a pack would complete it, is untested and is
the case most likely to behave differently. New files, unrelated prose, and
non-coding tasks outside repositories remain open observations under this
criterion.

## Deferrals

- (none)

## Reviewer notes

- (none)
