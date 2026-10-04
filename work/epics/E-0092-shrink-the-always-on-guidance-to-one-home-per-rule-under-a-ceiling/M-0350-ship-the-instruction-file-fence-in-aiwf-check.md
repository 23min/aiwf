---
id: M-0350
title: Ship the instruction-file fence in aiwf check
status: in_progress
parent: E-0092
depends_on:
    - M-0333
tdd: required
acs:
    - id: AC-1
      title: aiwf check refuses an instruction-file commit carrying unrelated files
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf check refuses an instruction-file commit without a resolving entity
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf check refuses an instruction-file removal without a disposition block
      status: met
      tdd_phase: done
    - id: AC-4
      title: The fence is an error by default and an aiwf.yaml setting turns it off
      status: met
      tdd_phase: done
    - id: AC-5
      title: The fence's finding codes and setting are discoverable
      status: met
      tdd_phase: done
    - id: AC-6
      title: This repository runs the kernel fence in place of its internal one
      status: met
      tdd_phase: done
---
## Goal

Make every aiwf repository fence edits to its handwritten instruction files by default: `aiwf check` refuses, over unpushed commits, an instruction-file change that rides with unrelated files, names no entity, or removes text without recording where it went.

## Closes

- (none)

## Context

ADR-0053 decides the rule and its boundaries. M-0333 built the same three rules as an internal policy over this repository's commit range, and its pure core, classification and fixtures are the starting point. `aiwf check` already judges unpushed commits for `provenance-untrailered-entity-commit`, resolving the range from `--since`, else `@{u}..HEAD`, else reporting `provenance-untrailered-scope-undefined`; the fence uses the same range and the same skip.

## Acceptance criteria

### AC-1 — aiwf check refuses an instruction-file commit carrying unrelated files

A commit in the audited range that changes handwritten instruction content and also changes a file outside the allowed companions produces one finding per unrelated file, naming the commit. **Pass criterion**: fixtures cover a code file beside an edit to each of `CLAUDE.md`, `AGENTS.md`, the router and a routed document; an update carrying owned outputs, their record and `aiwf.yaml` passing; a file under `.guidance/` outside the owned record refused; a change confined to a managed block and a merge commit not judged; a routed document renamed or deleted.

### AC-2 — aiwf check refuses an instruction-file commit without a resolving entity

A commit changing handwritten instruction content with no `aiwf-entity` trailer, or one resolving to no entity, produces one finding naming the commit and the value. **Pass criterion**: fixtures for the missing and the unresolvable case each produce one finding; a live entity, an archived one, a narrow legacy id and a composite `M-NNNN/AC-N` produce none.

### AC-3 — aiwf check refuses an instruction-file removal without a disposition block

A commit that removes handwritten instruction text, a rewording included, and carries no well-formed disposition block, or a malformed one, produces one finding. **Pass criterion**: fixtures cover a removal with no block, a block whose `Disposition:` value is outside `copy of <path>`, `relocated to <path>`, `pointer to <id>`, `reworded` or `deleted`, a block written as trailers passing, and a pure addition and a re-wrap needing none.

### AC-4 — The fence is an error by default and an aiwf.yaml setting turns it off

**Pass criterion**: with no setting the fence's findings are errors, so `aiwf check` exits non-zero; with the setting off the rule reports nothing; with no audited range the existing scope-undefined advisory stands and the rule reports nothing.

### AC-5 — The fence's finding codes and setting are discoverable

**Pass criterion**: `finding-codes-are-discoverable` and `config-fields-are-discoverable` pass with the new codes and setting in the tree; the `aiwf-check` skill's finding table carries each code with its remedy. The Release note states the default and the off switch; its wording is held at review.

### AC-6 — This repository runs the kernel fence in place of its internal one

The internal fence policy and its tests are removed, and the coverage gate that ran them runs the kernel rule instead, compiled from the tree under test. **Pass criterion**: the policy suite is green without the internal fence, the coverage-gate run-pattern of the Makefile and the CI workflow names the policy that runs the kernel rule, and `aiwf check --since <epic fork point>` on the epic branch reports no fence finding — the command, expectation and output recorded in Validation.

## Constraints

- The rule judges handwritten content only: the root `CLAUDE.md` and `AGENTS.md` outside aiwf's managed blocks, `.guidance/project.md`, and the documents it links to. No nested instruction file is in scope (ADR-0053).
- Allowed companions are the other instruction files, the files listed in `.guidance/.aiwf-owned` and that record itself, and `aiwf.yaml`. No directory is exempt by its name alone.
- A change confined to a managed block, and a merge commit, are not judged.
- Consumers get no ceiling and no size report.
- Reuse the provenance audit's range resolution; no second definition of "unpushed".

## Design notes

- ADR-0053 — the rule, its default and its scope.
- One finding code per repair, as the provenance codes do: an unrelated file, a missing or unresolvable entity, a missing or malformed disposition block.
- The setting sits under the existing `guidance:` block in `aiwf.yaml`.
- Define the guidance set once in the kernel — the host entry points, the router, its routed documents and the managed blocks — and have this repository's internal ceiling, discoverability and reader policies read that definition rather than helpers in the internal fence this milestone removes.
- Settle before the rule blocks consumers:
  - A re-wrapped paragraph counts as a removal under the internal fence's line comparison.
  - The disposition set has no value for a rewording, so the internal fence records rewordings as `deleted`.
  - Every message line starting `Removed:` opens a block, so a stray one — prose that happens to start that way, or a block missing its `Disposition:` line — fails the commit even beside a well-formed block.
  - `aiwf init` and `aiwf update` wrapping a bare guidance import in markers moves a handwritten line into a block, which the rule reads as a removal.
  - Under ADR-0054, `aiwf init` and `aiwf update` adopt applicable guidance packs, editing `aiwf.yaml`, `.guidance/` and host instruction files for the maintainer to commit as one change. E-0095 builds that path; the rule must read such a commit as generated output, not as an unrelated change beside guidance.
  - The internal fence and `internal/initrepo` read managed blocks with different parsers: a stray `<!-- aiwf:guidance:… -->` comment makes the fence judge the whole file while `initrepo` refreshes the block in place.
- The internal fence run over this repository's history since `v0.30.0` reports several dozen historical commits that mixed an instruction-file edit with other files, measured in M-0333's Validation; sorting them informs which companion files the kernel rule allows.

## Surfaces touched

- `internal/check/` — the rule and its codes
- `internal/cli/check/` — wiring into the audited range
- `internal/config/` — the setting
- `internal/skills/embedded/aiwf-check/SKILL.md` — the finding table
- `internal/policies/guidance_fence*.go`, `Makefile`, `.github/workflows/go.yml` — the internal fence replaced by a policy running the kernel rule

## Out of scope

- A `commit-msg` refusal (ADR-0053 defers it).
- A size ceiling or report for consumers.
- This repository's internal ceiling and guidance-reader list (M-0333).

## Dependencies

- M-0333 — the internal rules and fixtures this moves into the kernel
- ADR-0053 — the decision

## Coverage notes

- (none)

## References

- ADR-0053, G-0676, E-0092

## Release note

`aiwf check` refuses, by default, a commit in the push range that changes handwritten instructions — the root `CLAUDE.md` or `AGENTS.md` outside aiwf's generated blocks, `.guidance/project.md`, or a document it links to — when the commit also carries unrelated files (`instruction-commit-unrelated-file`), names no entity in an `aiwf-entity` trailer (`instruction-commit-entity-missing`), or removes text without a disposition block (`instruction-commit-disposition-missing`). The findings are errors, so the pre-push hook blocks the push until the commit is split or amended. Set `guidance.fence_instruction_commits: false` in `aiwf.yaml` to turn the rule off. Commits already on the configured trunk are not judged when that ref is remote-tracking, as the default `refs/remotes/origin/main` is, and the rule runs only where the branch has an upstream or `--since <ref>` is passed.

## Decisions made during implementation

- A host file's managed blocks are read strictly, as `pathutil.ManagedBlockSpan` reads them: a file carrying a stray or doubled `<!-- aiwf:guidance:… -->` marker has no `aiwf:guidance` block the rule recognizes, and that block's text, the import line included, is judged handwritten; the routing block is still read as generated. `aiwf update` finds `CLAUDE.md`'s import block by its first exact START and END lines and refreshes it in such a file, where the rule reads the refresh as a handwritten change; the strict reading never lets a handwritten change through unjudged.
- When `aiwf update` wraps a bare `@.claude/aiwf-guidance.md` line of `CLAUDE.md` in markers, the rule reads that line as removed from the handwritten text, and the commit carrying the update needs an `aiwf-entity` trailer and a disposition block. Every release that writes the import, from v0.14.0 on, writes it inside markers, so a bare line is one a person typed and the wrap happens once.
- The rule skips commits already on the configured trunk ref when that ref is remote-tracking; a local branch named as the trunk can carry unpushed commits and excludes nothing. A commit on a remote-tracking trunk has been pushed: it was judged when it reached the trunk, or it predates the rule. Without the skip, a branch that merges the trunk is held to the trunk's commits since it last synced, which nobody on that branch can change. Commits a merge brings in that are not on the trunk, such as a milestone's merged into its epic, are still judged; judging only the first-parent line, as the untrailered audit does, would leave them unjudged.
- Text counts as removed when some old line is missing from the new text and the old words do not survive in their order. The line test alone reads a re-wrapped paragraph, or one re-flowed around an inserted sentence, as a removal though no word went; the word test alone passes a reordering of words, which can change what a rule says. A paragraph moved within a file and re-wrapped in the same commit still reads as a removal.
- The disposition set gains `reworded`, for a rule that stays where it was in new words. Recorded as `deleted`, a rewording cannot be told from a removed rule; M-0336 maps the inventory's `merge into` and `tighten` decisions to it.
- Every message line starting `Removed:` opens a block, so a half-written block and a sentence that happens to start that way are both malformed, beside a good block or not. Reading a `Removed:` line with no `Disposition:` after it as prose would let a forgotten disposition pass whenever another block is well formed; reading blocks only among the trailers would hold writers to git's trailer parsing.
- `guidance.fence_instruction_commits` turns the fence off, apart from `guidance.enabled`, which governs pack maintenance only; freezing the installed packs leaves the fence running.
- The coverage gate that ran the internal fence runs the kernel rule in its place, through a policy compiled from the tree under test, over the same range. The pre-push hook runs the rule through the installed `aiwf`, which until E-0092 reaches main is built from a main that lacks it, and no CI step runs `aiwf check`; without the gate, this repository's instruction-file commits would go unjudged through the rest of E-0092, and past a stale binary after it. The co-author ban backs up its hook the same way.
- A rule moving between this repository's guidance and aiwf's shipped fragment moves in two commits: one adds it to its new home, the next removes the old copy with `Disposition: copy of <new home>`. ADR-0053 keeps the fragment source out of an instruction-file commit, so the single commit M-0336 AC-6 and E-0092's constraint describe cannot pass the rule; the rule is briefly in both homes and never in neither. M-0336 AC-6 and the epic constraint change before M-0349 starts.
- When `aiwf update` hands a repository over from the legacy `ai-dotfiles` block in `CLAUDE.md`, removing that block, the rule reads the removal as handwritten, and the commit carrying the handover needs an `aiwf-entity` trailer and a disposition block, as the bare-import wrap does. The handover happens once per repository, and the block is not one aiwf writes, so it is not read as generated output.

## Validation

- Gates on the tree committed as `71d6c8363`, in the aiwf devcontainer, `go1.25.12 linux/amd64`. `make check-fast` (go vet, golangci-lint, the full test suite): expected exit 0; observed exit 0, golangci-lint `0 issues.` `go test -exec= -covermode=atomic -coverprofile=coverage.out -coverpkg=./internal/... -parallel 8 ./...`: expected exit 0; observed exit 0. `AIWF_COVERAGE_BASE=1b83e7830 make coverage-gate-only` over that profile, the milestone's base: expected exit 0; observed exit 0.
- M-0350 AC-5 — `go test -count=1 -run 'TestPolicy_FindingCodesAreDiscoverable|TestPolicy_ConfigFieldsAreDiscoverable|TestPolicy_FindingCodesDocumentedInSkill|TestPolicy_FindingCodesHaveHints' ./internal/policies/`, expected to pass; observed passing at `0da2d5e52` in the aiwf devcontainer, `go1.25.12 linux/amd64`. These policies predate the milestone. They failed during M-0350 AC-1, when `instruction-commit-unrelated-file` first existed without its hint and skill row, and each code's hint and row then landed in the commit that added the code — `32a352145`, `cfea1bf9c`, `34bfecba1` — and the setting's documentation in `fe64dcf5d`, because the pre-commit hook runs the policy suite. M-0350 AC-5's phases were recorded after those events.
- M-0350 AC-6 — at `a95ada7d1` in the aiwf devcontainer, `go1.25.12 linux/amd64`, with `aiwf` built from that commit, on 2026-10-03: `aiwf check --since a61f3d8de`, the epic's fork point, expected to report no `instruction-commit-*` finding; observed exit 0 with `3 findings (0 errors, 3 warnings)`, none of them the fence's. The same rule run as the coverage gate, `AIWF_COVERAGE_BASE=origin/main go test -count=1 -run 'TestPolicy_InstructionFence$' ./internal/policies/`, over the 187 commits on the branch and not on main: expected to pass; observed passing.

## Deferrals

- G-0731 — a commit that is a shallow clone's boundary is read as adding every file it carries, so the fence and the untrailered audit report errors it did not cause.

## Reviewer notes

- Obligations this change adds, counted from `git diff 1b83e7830..HEAD` over the finding-code, config-field, policy-test and run-pattern registries:
  - three finding codes, each with a hint entry and an `aiwf-check` skill row; owner ADR-0053, retired only by superseding it;
  - `guidance.fence_instruction_commits`, whose schema entry, generated example and Claude baseline inventories move together; owner ADR-0053;
  - `PolicyInstructionFence` in both coverage-gate run-patterns, pinned by `TestInstructionFence_WiredIntoCoverageGate`, with serial skip-list entries for its environment test and for two verb-level tests in `internal/cli/check`; owner E-0092, kept while the pre-push hook can run an `aiwf` older than the tree, as the co-author ban's backstop is;
  - the reader rule's arm for `projectguidance` constants, under which a test naming guidance that way is listed in `guidanceReaderList`; owner D-0102, retiring with it;
  - a narrow-id allowlist entry for `internal/check/instruction_fence_test.go`, retiring with that file.
- Accepted or declined:
  - the walk reads each revision's router and owned record twice, about 1–2 ms per commit;
  - the owned record is parsed by the installer and by the fence separately, while E-0095 leaves its format unchanged;
  - an insertion that inverts a rule's meaning is not a removal, by the removal test's rule;
  - the subtraction lens's rewrite of the walker's error handling is declined;
  - the `fs.ErrNotExist` guard in `guidanceSetConstants` stays, since without it the unparseable-file case of `TestGuidanceReaderTests_Errors` passes on the missing-package error instead;
  - `CLAUDE.md`'s phrase "the guidance fence" stays;
  - the `instruction-commit-entity-missing` remedy carries no test: D-0070 rules out a phrase pin over the shipped skill, and the remedy was checked by running it.
- M-0336 AC-6 and E-0092's constraint still describe one-commit moves; the two-commit move of the Decision above replaces them in the cut pass before M-0349.
- The deciding full review, over `1b83e7830..af4436aa6`, returned request-changes; its findings are fixed and pinned, a scoped review of the fixes confirmed them, and its four remaining items are fixed and confirmed by test and mutation.
