---
id: M-0335
title: Route on-demand guidance through the project router and tighten what moves
status: in_progress
parent: E-0092
depends_on:
    - M-0349
    - M-0336
tdd: advisory
acs:
    - id: AC-1
      title: Both root entry points route to project-local guidance
      status: met
      tdd_phase: done
    - id: AC-2
      title: Both hosts read relevant guidance before root-started edits
      status: open
    - id: AC-3
      title: Every pin on a moved passage is retired with a recorded reason
      status: open
    - id: AC-4
      title: The ceiling constant steps down to the re-homed size
      status: met
    - id: AC-5
      title: No instruction file exists below the repository root
      status: met
      tdd_phase: done
    - id: AC-6
      title: Merge and tighten dispositions are applied as the inventory records
      status: open
---
## Goal

Make `.guidance/project.md` a short task router, move every on-demand rule into the documents it names, and apply the inventory's merge and tighten dispositions, so the root `CLAUDE.md` holds only primed rules.

## Closes

- (none)

## Context

M-0349's inventory gives each rule a disposition and names the on-demand documents; M-0336 has already deleted the copies. The routing block both entry points carry sends a task to `.guidance/project.md` first, then the index and the packs it needs. Today `.guidance/project.md` says to read `CLAUDE.md` in full and that it takes precedence over the packs, and the `AGENTS.md` preamble requires the same full read for Codex. After this milestone the primed development rules live only in the root instruction files and the router — `CLAUDE.md` for both hosts, the `AGENTS.md` preamble for Codex alone — Codex's required read of `CLAUDE.md` covers only those, and everything else is reached through the router.

## Acceptance criteria

### AC-1 — Both root entry points route to project-local guidance

`.guidance/project.md` names each on-demand document and the tasks that need it, and states which primed rules in `CLAUDE.md` override a pack. The `AGENTS.md` preamble requires only the primed `CLAUDE.md` content. No home-directory language import returns. **Pass criterion**: a relationship check resolves every route in `.guidance/project.md` and both entry points to an existing file and rejects a missing target. Generated blocks change only through their owning updater.

### AC-2 — Both hosts read relevant guidance before root-started edits

Record fresh sessions for both hosts, started at the repository root, for a Go change, a Python script, a TypeScript test, a new file and a prose-only task. **Pass criterion**: tool events show each host reading the relevant on-demand document or pack through the router before its first relevant edit, and the prose-only task reading no language pack. File existence alone does not establish delivery.

### AC-3 — Every pin on a moved passage is retired with a recorded reason

Every test the guidance-reader list names a pin, whose passage this milestone moves, is listed at this milestone's start and recorded here, then retired with its reason in this body's table, and its entry leaves the list. A check the moved passage still needs is a new relationship or absence check with its own entry, never a pin on the passage's new file, which D-0102 counts as guidance too. **Pass criterion**: the policy suite is green and the table accounts for every listed pin; the starting list sits in Validation. The mechanical half is the green suite; the accounting is held at review.

### AC-4 — The ceiling constant steps down to the re-homed size

Lower each host's handwritten primed ceiling to its measured size after relocation. **Pass criterion**: the relocated tree passes at the new ceilings. Record the ceiling measure's command, each host's counts before and after, and the on-demand loads.

### AC-5 — No instruction file exists below the repository root

**Pass criterion**: a policy fails when a tracked `CLAUDE.md` or `AGENTS.md` exists anywhere but the repository root, naming the path; on the tree it reports none.

### AC-6 — Merge and tighten dispositions are applied as the inventory records

Every rule M-0349 marks "merge" or "tighten" is rewritten in its one home, and every rule marked "move on demand" is in the document the inventory names. **Pass criterion**: the inventory test's reconciliation (M-0336 AC-5) reports a removal whose named destination does not exist, and the tree passes; the rewording itself is held at review, and its effect is judged by M-0338's rubric.

## Constraints

- One instruction file per host, at the root; no directory entry files.
- Rewording is allowed where the inventory records it; each removed or rewritten passage carries its disposition block in the commit body.
- Preserve source/output ownership; generated blocks change through their owner.
- Retire pins with reasons; a check a moved passage still needs is a new relationship or absence check. Do not create duplicate prose merely to satisfy an old pin.
- Record the growth delta and both hosts' loading observations.

## Design notes

- `CLAUDE.md` holds the primed development rules both hosts need, the `AGENTS.md` preamble the few that are Codex's alone, and the router its own entries. Codex reaches `CLAUDE.md` through the preamble's required read, which is why that read counts toward Codex's primed load.
- The on-demand documents and their locations are the ones M-0349 names.
- M-0334's Validation records a pilot of this routing for the Go conventions cluster: with those rules moved out of `CLAUDE.md`, Claude opened the routed documents before its first edit.
- Exercise `aiwf update` in a disposable checkout, with `HOME` pointed at a scratch directory, and verify it preserves the handwritten router and regenerates the expected routes without restoring obsolete home-directory imports.

## Surfaces touched

- `CLAUDE.md`, the `AGENTS.md` preamble, `.guidance/project.md` and the on-demand documents.
- Existing pinning tests and guidance routing checks.

## Out of scope

- Deleting copies (M-0336) and the pointer cut (M-0337).
- Rewording the shipped fragment; moves into and out of it are M-0336's.

## Dependencies

- M-0349 — the dispositions and the on-demand document set
- M-0336 — copies are gone before the remainder moves

## Coverage notes

- (none)

## References

- D-0102 — the decision AC-3 follows; the pin entries are in `guidanceReaderList`
- ADR-0054 — adopted language conventions have an external source and tracked project delivery

## Release note

## Decisions made during implementation

- This milestone runs `tdd: advisory`. AC-2 is an observation record and AC-6 a rewording held at review, so neither has a red test to precede it; the phases are recorded, as they happen, for AC-1, AC-4 and AC-5. Decided by the maintainer.
- A live reference to a moved section is re-pointed at the section's new document: normative docs, Go comments, the messages of CI workflows, hooks and the `Makefile`, and `CONTRIBUTING.md`. It lands in the commit after the move, since the fence admits no file outside the guidance set beside a guidance change. Audits, archived documents, the changelog and entity bodies are records of their date and stay as written. Decided by the maintainer.

## Validation

### AC-1 — routes, imports and the updater

- **Routes resolve.** `PolicyGuidanceCeiling` follows every link in the router and in both entry points' handwritten text and routing blocks, and reports one whose target does not exist; `TestMeasureGuidanceLoad_RouterLinksDefaultToConditional` holds that a router route to a missing file is reported. On the tree it reports none: `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/` passes.
- **No outside import.** `PolicyGuidanceImportsInRepository` reports an import under the home directory or by absolute path in either entry point, managed blocks included; on the tree it reports none.
- **The updater preserves the handwritten guidance.** Command: in a `git clone --no-hardlinks` of this branch at `81d4f617e`, with `HOME`, the XDG directories and `TMPDIR` pointed at a scratch directory, `aiwf update` built from that commit, run twice, then `git status --porcelain`. Expected: no tracked change, so the router, the routed documents and both entry points' handwritten text and managed blocks are as committed, and no import from outside the repository. Observed: exit 0 both times, `git status --porcelain` printed nothing, and `grep '@~\|@/' CLAUDE.md AGENTS.md` matched nothing. The update reported the guidance import and `AGENTS.md`'s guidance as preserved and listed no pack refresh, since `guidance.enabled` is `false`. Environment: the aiwf devcontainer, Linux, 2026-10-11.

### AC-4 — the ceilings

Command: `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/`.

| Host | Handwritten primed, at start (`d753e5447`) | After re-homing (`81d4f617e`) | Ceiling now |
|---|---|---|---|
| Claude | 7,801 | 1,280 | 1,280 |
| Codex | 7,930 | 1,420 | 1,420 |

The aiwf-generated load is unchanged at 2,460 for Claude and 2,497 for Codex. On-demand loads after re-homing, in words: `testing.md` 1,454, `verb-design.md` 1,139, `shipped-content.md` 895, `stress-harness.md` 783, `enforcement.md` 713, `environment.md` 422, `go-conventions.md` 378, `writing-docs.md` 262, `release.md` 242. Both hosts are below M-0349's targets of 1,629 and 1,758.

### AC-3 — the starting list

`guidanceReaderList` at this milestone's start, `d753e5447`, names thirteen pins. Each pins a passage this milestone moves on demand or rewrites in place, and each is retired, since D-0102 bars a sentence pinned in development guidance as evidence:

| Pin | Passage | Row | Disposition | Retired, and what still holds |
|---|---|---|---|---|
| `TestAiwfArchive_AC6_ClaudeMdNamesArchiveConvention` | the archive commitment | commit-archive | tighten | pinned the commitment's wording; `archive-sweep-pending` and the loader's archive resolution hold the convention itself |
| `TestM0154_AC2_CLAUDEMDOperatorSetupAmended` | Operator setup | operator-setup, settings-consent | move on demand | pinned the passage's wording; ADR-0015's stance stays pinned by `TestM0154_AC1_ADR0015RecordsConsentGatedStance` |
| `TestM0195_AC5_SkillBodyDisciplineInClaudeMd` | Skills policy, the shipped-surface paragraph | shipped-surfaces | move on demand | pinned the passage's wording; `skill-body-id` enforces its id-shaped subset |
| `TestM0211_AC3_AuthoringRuleNamesDividingPrinciple` | Consumer-operating guidance vs repo-development guidance | audience-split | move on demand | pinned the passage's wording; `PolicyM0211GuidanceOperatingAnchors` holds the operating anchors in the fragment |
| `TestM0234_AC4_ClaudeMdWorktreeSectionsCiteWorktreeAdd` | the worktree default and Subagent worktree isolation | worktree-default, subagent-isolation | tighten, move on demand | pinned the passages' wording; the ritual call sites stay pinned by `TestM0234_AC1_StartMilestoneCutStepInvokesWorktreeAdd` |
| `TestM0293_KernelGuidanceStatesForceIsHumanOnly` | the provenance principle and commitment; its `docs/design/design-decisions.md` half is not guidance | provenance-principle, commit-provenance | merge, tighten | its two `CLAUDE.md` rows are cut; its design-record rows stay, since `docs/design/` is not guidance |
| `TestM083_AC2_CLAUDEMdCommitment2` | the stable-ids commitment | commit-stable-ids | tighten | pinned the commitment's wording; `guidance-ids-through-links` holds the guidance to canonical placeholders |
| `TestPolicy_CLAUDEMDCLIConventionsLogging` | CLI conventions, logging | output-and-logging | move on demand | pinned the passage's wording; ADR-0017 records the decision and `internal/logger` tests the behaviour |
| `TestPolicy_ClaudeMdTestDisciplineSection` | Test discipline | test-discipline | move on demand | pinned a heading; `test-setup-presence` enforces the rule itself |
| `TestPolicy_M0128DocumentationHierarchy` | Documentation hierarchy | doc-tiers | move on demand | pinned the passage's subtree list; which tier a subtree carries is a judgment held at review |
| `TestPolicy_M0132ClaudeMdDevcontainerSection` | Devcontainer | devcontainer | move on demand | pinned the passage's wording; `devcontainer-paths-resolve` now reads the routed documents |
| `TestPolicy_M0134ClaudeMdTestRunningSections` | the two test-running subsections | tests-in-devcontainer, tests-on-macos | move on demand | pinned the passages' wording; nothing a test can re-derive is lost |
| `TestPolicy_M0228SkillsPolicyBroadenedPrinciple` | Skills policy, the shipped-surface paragraph | shipped-surfaces | move on demand | pinned the passage's wording; `skill-body-id` enforces its id-shaped subset |

Three listed readers that are not pins read moved text too: `TestSkillEditProvenance_DocumentedInClaudeMd` requires the Ritual content authoring and What's enforced and where headings in `CLAUDE.md`; `TestPolicy_NoDanglingEntityRefsInNarrativeDocs` resolves entity links in `CLAUDE.md` and `ROADMAP.md` only; `TestPolicyDevcontainerPathsResolve` does not read `.guidance/`. Both relationship checks now read the development-guidance set, and the absence check scans every guidance document.

## Deferrals

- (none)

## Reviewer notes

- (none)
