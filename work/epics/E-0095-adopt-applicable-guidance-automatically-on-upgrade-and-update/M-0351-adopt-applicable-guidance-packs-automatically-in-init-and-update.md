---
id: M-0351
title: Adopt applicable guidance packs automatically in init and update
status: in_progress
parent: E-0095
tdd: required
acs:
    - id: AC-1
      title: Unset selection adopts every applicable pack without a terminal
      status: met
      tdd_phase: done
    - id: AC-2
      title: Update adds newly applicable packs and never removes one
      status: met
      tdd_phase: done
    - id: AC-3
      title: Ignored packs are never adopted
      status: met
      tdd_phase: done
    - id: AC-4
      title: Disabled maintenance performs no detection or adoption
      status: met
      tdd_phase: done
    - id: AC-5
      title: Init and update never prompt for guidance
      status: met
      tdd_phase: done
---
## Goal

Every enabled `aiwf init` and `aiwf update` run adopts each applicable, unignored guidance pack without asking, on a terminal or not, and installs it with host routing.

## Closes

- (none)

## Context

Detection, retrieval, validation, installation, host routing and legacy handover exist in `internal/projectguidance` and `internal/initrepo`. Selection is injected as a `projectguidance.Selector`: `cliutil.GuidanceSelector` returns an interactive prompt on a terminal and a stderr suggestion report otherwise, so adoption depends on how the verb was launched. ADR-0054 replaces that with add-only automatic adoption.

## Acceptance criteria

### AC-1 — Unset selection adopts every applicable pack without a terminal

**Pass criterion**: `aiwf update` in a repository whose `aiwf.yaml` has no `guidance` block, against a local catalogue with packs matching and not matching the repository's tracked files, leaves `guidance.packs` listing exactly the matching packs, their files under `.guidance/`, and the engineering-guidance route in each selected host's instruction file, with stdin not a terminal. **Edge cases**: a pack that matches every tracked file; a repository matching no pack (no `guidance.packs` written, nothing installed); nested and gitignored files, which detection already includes and excludes. **Code references**: adoption in `internal/initrepo/project_guidance.go`; binary-level tests beside `internal/cli/update/project_guidance_test.go`.

### AC-2 — Update adds newly applicable packs and never removes one

**Pass criterion**: with `guidance.packs` already listing an installed pack, adding tracked files matching a further pack and running `aiwf update` appends that pack to `guidance.packs` and installs it, and a selected pack whose files no longer match stays selected and installed. **Edge cases**: an explicitly empty `guidance.packs: []` gains newly matching packs; order of existing entries is preserved; a second run with no new matches writes nothing; a matching pack removed from `guidance.packs` without being listed in `guidance.ignored` is adopted again on the next run. **Code references**: the adoption function `Adopt` in `internal/projectguidance/selection.go`, used by `internal/initrepo/project_guidance.go`, with a table-driven unit test and a binary-level update test.

### AC-3 — Ignored packs are never adopted

**Pass criterion**: a pack listed in `guidance.ignored` is never added to `guidance.packs` or installed, however its detection matches, and moving an adopted pack's id from `guidance.packs` to `guidance.ignored` removes its unmodified output on the next update. **Edge cases**: an id listed in both `guidance.packs` and `guidance.ignored` is refused when `aiwf.yaml` loads, so adoption never sees it; a locally edited owned file of the removed pack blocks removal and preserves the installed set. **Code references**: the adoption function's unit test; `internal/cli/update/guidance_reconsideration_test.go`.

### AC-4 — Disabled maintenance performs no detection or adoption

**Pass criterion**: with `guidance.enabled: false`, `aiwf init` and `aiwf update` perform no catalogue retrieval, detection or adoption, and leave `aiwf.yaml`, `.guidance/` and host routing byte-identical. **Edge cases**: `guidance.packs` unset and set; a matching pack present in the catalogue. **Code references**: `internal/cli/update/project_guidance_failures_test.go` (`TestRun_ProjectGuidanceDisabledPreservesInstallationWithoutRetrieval`); the unset selection and `init` are covered by the disabled cases of `internal/cli/integration/guidance_noninteractive_test.go`.

### AC-5 — Init and update never prompt for guidance

**Pass criterion**: `aiwf init` and `aiwf update` run on a real terminal adopt and install the same packs as a run without one and never read stdin for guidance, and neither verb's `--help` describes a guidance prompt. **Edge cases**: `init --no-prompt` and `update --no-prompt` on a terminal; a terminal whose stdin reaches EOF immediately. **Code references**: the pty test in `internal/cli/cliutil/guidance_terminal_linux_test.go`; the help-text drift is pinned against the verbs' registered flags rather than by phrase.

## Constraints

- `init` and `update` never commit or push.
- A guidance failure leaves installed guidance, its recorded revision and `aiwf.yaml` unchanged, and never fails unrelated update work.
- `aiwf.yaml` writes go through `aiwfyaml` and preserve unrelated fields and comments.
- Adopted pack ids are recorded in `guidance.packs` only together with a successful installation, so `aiwf.yaml` never lists a pack the working tree lacks. A run interrupted between installation and the `aiwf.yaml` write converges on the next run, which re-detects and records the same packs.

## Design notes

- ADR-0054 settles adoption: add-only, on every enabled run, `guidance.ignored` as the only opt-out, no guidance prompt.
- The adoption rule is one pure function over the detected matches, the current `guidance.packs` and `guidance.ignored`, returning the new selection; `init`, `update` and the report milestone consume it rather than re-deriving it.
- `--no-prompt` on `init`, `update` and `upgrade` keeps its meaning for hook consent. Their help text, the README's guidance paragraph and the unreleased CHANGELOG entry for `--no-prompt` stop describing guidance selection, and the pty test's `update-no-prompt` guidance subtest goes with the prompt.

## Surfaces touched

- `internal/initrepo/project_guidance.go`
- `internal/cli/cliutil/guidance.go`
- `internal/cli/update/update.go`, `internal/cli/initcmd/initcmd.go`, `internal/cli/upgrade/upgrade.go` (help text)
- `README.md`, `CHANGELOG.md`
- `internal/cli/cliutil/guidance_terminal_linux_test.go`

## Out of scope

- The end-of-run guidance report, which the next milestone in this epic delivers; this milestone keeps the existing step-ledger line.
- Hook-consent and statusline prompts, and the repo lock held while any prompt waits (G-0709).

## Dependencies

- ADR-0054.

## References

- ADR-0054, ADR-0052, D-0089, E-0094.

---

## Release note

`aiwf init` — including the first run in a repository — and `aiwf update` now adopt applicable engineering guidance on their own, on a terminal or not. Every pack whose detection patterns match the project's files and that is not listed in `guidance.ignored` is installed with host routing and then added to `guidance.packs`; a project that later gains matching files gains the pack on the next `aiwf update`. Recording a pack keeps the rest of `aiwf.yaml` as written where the layout allows and otherwise re-encodes it with its fields and comments. Adoption is add-only: a selected pack is never removed automatically, `packs: []` is an empty selection that adoption fills rather than a way to opt out, and `guidance.ignored` is the per-pack opt-out — moving an id there retires the pack's unmodified files. A selected pack the catalogue no longer lists is reported with its fix; `init` and `update` delete its unmodified files from `.guidance/` and install the rest of the selection, while `aiwf worktree add` leaves installed guidance unchanged. The first adoption also performs the handover from legacy ai-dotfiles guidance in `CLAUDE.md`, as an explicit selection always has. The select / not now / ignore guidance prompt is gone, and `--no-prompt` on `init`, `update` and `upgrade` now governs hook consent only. `guidance.enabled: false` still performs no download or adoption. When `aiwf.yaml` cannot record a newly applicable pack — anchors, aliases or a merge key inside `guidance` — the run leaves installed guidance, including already-selected packs, unrefreshed and reports why.

## Decisions made during implementation

- D-0103 — a selected pack the catalogue no longer lists is skipped, kept in `guidance.packs`, reported with its fix, and its unmodified files retired.
- D-0100, which governed how prompt answers were saved, is superseded now that the prompt is removed.

## Validation

- `make ci` — exit 0 over the tree committed as `d7fe331fc`: build, `go vet` (default, `stress`, `testpins` tags), `golangci-lint` (0 issues), `go test -race -parallel 8` with coverage (91.8% total), the profile-driven gate step (diff-scoped coverage audit, firing-fixture meta-gate), and `aiwf doctor --self-check` (29 steps). Linux devcontainer, 2026-09-25.
- `aiwf check` — 0 errors at `d7fe331fc`, Linux devcontainer, 2026-09-25.
- AC-3 and AC-4 were already pinned when each AC was reached. Deleting the ignored-pack filter in `projectguidance.Adopt` fails `TestAdopt_UnsetSelection`, `TestAdopt_ExistingSelectionGainsNewMatchesAndKeepsItsOwn`, `TestBinary_GuidanceNoninteractiveAdoptsApplicablePacks/{init,update}/ignored` and `TestBinary_GuidanceConfigurationRemovalAndReconsideration/ignore_then_reconsider`; deleting the `GuidanceEnabled()` check in `ensureProjectGuidance` fails the noninteractive `{init,update}/disabled` cases, `TestRun_ProjectGuidanceDisabledPreservesInstallationWithoutRetrieval` and `TestProjectGuidanceRefresh_OptOutAndDryRun/disabled`. Measured with `go test -count=1` on the mutated tree, Linux devcontainer, 2026-09-25.
- AC-5, help clause (observation): `aiwf init --help`, `aiwf update --help` and `aiwf upgrade --help` from a binary built at `069c66fc1` (the help text is unchanged at `d7fe331fc`), Linux devcontainer, 2026-09-25. Expected: no guidance prompt described. Observed: guidance is described only as adopted "without prompting"; every other mention of a prompt concerns hook consent or the statusline, and each `--no-prompt` flag reads "never prompt for hook consent; leave undecided hooks unchanged".

## Deferrals

- G-0713 — `aiwf.yaml`'s `hooks:` and `contracts:` append writers break a flow-style or indented root; found in review, present before this milestone.

## Reviewer notes

- (none)
