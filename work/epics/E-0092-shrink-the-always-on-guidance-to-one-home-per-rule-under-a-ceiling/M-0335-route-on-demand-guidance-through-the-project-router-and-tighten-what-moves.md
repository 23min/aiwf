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
      status: met
    - id: AC-3
      title: Every pin on a moved passage is retired with a recorded reason
      status: met
    - id: AC-4
      title: The ceiling constant steps down to the re-homed size
      status: met
    - id: AC-5
      title: No instruction file exists below the repository root
      status: met
      tdd_phase: done
    - id: AC-6
      title: Merge and tighten dispositions are applied as the inventory records
      status: met
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

Nothing a consumer sees changes. This repository's own development guidance is re-homed: `CLAUDE.md` keeps the rules every task needs, and the rest moves into documents under `.guidance/project/` that the project router names; no embedded skill, template, fragment, verb or message changes.

## Decisions made during implementation

- This milestone runs `tdd: advisory`. AC-2 is an observation record and AC-6 a rewording held at review, so neither has a red test to precede it; AC-1 and AC-5 carry the phases recorded as their tests were written. AC-3 retires tests rather than writing one, and AC-4's constant changed after the re-homing had already shrunk the text, so no failing state existed to record for either; both carry none. Decided by the maintainer.
- A live reference to a moved section is re-pointed at the document that now holds it: Go comments and messages, normative docs and ADR pointers, the messages of CI workflows, hooks and the `Makefile`, `.golangci.yml`, `.devcontainer/project/README.md` and `CONTRIBUTING.md`. The re-pointing lands after the moves, in commits of its own, since the fence admits no file outside the guidance set beside a guidance change. A record of its date stays as written: audits, archived documents, the changelog, gap, decision and milestone bodies, an ADR's context or consequences recording what `CLAUDE.md` said when it was decided, the working catalogs `docs/design/legal-workflows-audit.md` and `docs/design/legal-workflows-first-principles.md`, `docs/design/growth.md`, and the done-list in `docs/design/design-lessons.md`. References to sections removed before this milestone are left to a gap. Decided by the maintainer.
- The ADR edits two plain commits made, `f497f3f2c` and `95c37637b`, are acknowledged with `aiwf acknowledge illegal`, one commit per ADR, rather than rebuilt through `aiwf edit-body`, which would rewrite every later commit this body cites. Later ADR edits go through `aiwf edit-body`. Decided by the maintainer.
- Claude's L2 observation made no edit, so it was run once more on the same prompt; the second run made none either, so the cell is recorded as not exercised and AC-2 is met on the cells that were. Decided by the maintainer.
- The guidance checks run when guidance changes: `go.yml`'s pull-request and push filters name `CLAUDE.md`, `AGENTS.md` and `.guidance/**`, and the cadence's skip rule names them. The rule states the files whose change requires a run rather than claiming to list everything `make ci` checks, and keeps planning-state commits skippable, as its inventory row requires. Decided by the maintainer.
- `guidance-imports-in-repository` also reports a relative import that climbs out of the repository, so no import from outside it passes. Decided by the maintainer.
- A primed clause is cut only where the agent meets its content anyway at the point of need: the scope states (`aiwf authorize --help`), the archive-backlog finding (`aiwf check` reports it), what the updater writes, and the pointer to the convergence rule (the router routes verb work). The tool-not-co-actor clause, the reversal pattern for an archived entity and the reason `aiwf worktree add` materializes artifacts stay, since an agent would make the mistake before thinking to look. Decided by the maintainer.
- `AGENTS.md`'s "Repository development rules still come from `CLAUDE.md`" stays: in its paragraph it says the rules do not come from Codex's skills, and the preamble's first paragraph names the routed documents. Its row is keep. Decided by the maintainer.
- G-0715 closes as addressed by `81d4f617e`: `CLAUDE.md` no longer counts on CI running on epic or milestone pushes. G-0728 closes as addressed by the moves that corrected both of its sentences. G-0436 stays for M-0337, which closes it with the path check that keeps it closed; both of its sites read correctly now. G-0726 stays open with its body corrected: the guidance is now in `go.yml`'s filters, the rest of the files it names are not, and nothing pins the filter. Decided by the maintainer.
- References naming sections that never moved here go to two gaps, one for code comments and an ADR, one for the contribution templates; no check of section citations is added. Decided by the maintainer.
- The guidance-reader list's pin kind goes with its last entry: D-0102 bars a pinned sentence as evidence, so no new entry can be a pin.
- `CLAUDE.md` points at a routed document by a backticked path rather than a link, so the pointer needs no ceiling read-table row; nothing checks it until M-0337's cited-path check.

## Validation

### Gates

At `0805fae26`, in the aiwf devcontainer, Linux, `go1.25.12 linux/amd64`, 2026-10-11:

- `AIWF_COVERAGE_BASE=eba3965ea make ci` exits 0: vet, lint, the race-enabled suite with coverage, total `92.0%` of statements, the profile-driven gates, the diff-scoped coverage gate among them, and `self-check passed (29 steps)`.
- `aiwf check` reports 0 errors. Its warnings on this milestone are `acs-tdd-audit` on AC-2, AC-3, AC-4 and AC-6, which this milestone's `tdd: advisory` decision accepts.

### AC-1 — routes, imports and the updater

- **Routes resolve.** `PolicyGuidanceCeiling` follows every link in the router and in both entry points' handwritten text and routing blocks, and reports one whose target does not exist; `TestMeasureGuidanceLoad_RouterLinksDefaultToConditional` holds that a router route to a missing file is reported. On the tree it reports none: `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/` passes.
- **No outside import.** `PolicyGuidanceImportsInRepository` reports an import under the home directory, by absolute path, or by a relative path leaving the repository, in either entry point, managed blocks included; `TestOutsideImports` and `TestPolicyGuidanceImportsInRepository_ReadsManagedBlocks` pin it, and on the tree it reports none.
- **The updater preserves the handwritten guidance.** Command: in a `git clone --no-hardlinks` of this branch at `81d4f617e`, with `HOME`, the XDG directories and `TMPDIR` pointed at a scratch directory, `aiwf update` built from that commit, run twice, then `git status --porcelain`. Expected: no tracked change, so the router, the routed documents and both entry points' handwritten text and managed blocks are as committed, and no import from outside the repository. Observed: exit 0 both times, `git status --porcelain` printed nothing, and `grep '@~\|@/' CLAUDE.md AGENTS.md` matched nothing. The update reported the guidance import and `AGENTS.md`'s guidance as preserved and listed no pack refresh, since `guidance.enabled` is `false`. Route regeneration was not exercised for the same reason: the run fetched nothing and rewrote no route. Environment: the aiwf devcontainer, Linux, 2026-10-11.

### AC-4 — the ceilings

Command: `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/`.

| Host | Handwritten primed, at start (`d753e5447`) | After re-homing (`81d4f617e`) | At the wrap, and the ceiling |
|---|---|---|---|
| Claude | 7,801 | 1,280 | 1,275 |
| Codex | 7,930 | 1,420 | 1,415 |

The aiwf-generated load is unchanged at 2,460 for Claude and 2,497 for Codex. On-demand loads after re-homing, in words: `testing.md` 1,454, `verb-design.md` 1,139, `shipped-content.md` 895, `stress-harness.md` 783, `enforcement.md` 713, `environment.md` 422, `go-conventions.md` 378, `writing-docs.md` 262, `release.md` 242. Both hosts are below M-0349's targets of 1,629 and 1,758. The last column follows the trims and wording fixes the wrap review decided; each ceiling equals its host's measured load.

### AC-2 — reads before root-started edits

- **Command:** `scripts/e0092-observe/run.sh 95c37637b0d7b2f4b9617d5b22eef74851b912bb <host> <task> <n>` for each host and the M-0334 rubric tasks T1 (a Go change), L1 (a Python script in a new file), L2 (a TypeScript test) and T6 (prose only), two runs at a time, then `scripts/e0092-observe/summarize.py` over the runs; no judge, since this criterion is about reads. The harness is the one M-0334 records, as frozen at `5092de5fd`, and each prompt is the rubric's.
- **Expected:** before its first edit, each host reads the router and the routed document or pack its task needs; the prose-only task reads no language pack.
- **Observed,** each document as the runs were shown it before / after their first edit:

| task | host | runs | reads | task-loaded words |
|---|---|---|---|---|
| L1 | claude | 1 | router 1/0, index 1/0, python 1/0 | 168 |
| L1 | codex | 1 | CLAUDE.md 1/0, router 1/0, index 1/0, code-health 1/0, python 1/0, environment.md 1/0 | 1101 |
| L2 | claude | 2 | none | 0 |
| L2 | codex | 1 | CLAUDE.md 1/0, router 1/0, index 1/0, code-health 1/0, typescript 1/0, environment.md 1/0 | 1087 |
| T1 | claude | 1 | router 1/0, go-conventions.md 1/0, testing.md 1/0, verb-design.md 1/0 | 2971 |
| T1 | codex | 1 | CLAUDE.md 1/0, router 1/0, index 1/0, code-health 1/0, go 1/0, environment.md 1/0, go-conventions.md 1/0, shipped-content.md 1/0, testing.md 1/0, verb-design.md 1/0 | 5084 |
| T6 | claude | 1 | router 1/0, stress-harness.md 1/0, writing-docs.md 1/0 | 1045 |
| T6 | codex | 1 | CLAUDE.md 1/0, router 1/0, index 1/0, testing.md 1/0, writing-docs.md 1/0 | 1716 |

  - Every run that edited read the router and the routed document or pack for its task before its first edit; neither prose-only run read a language pack.
  - Claude's L2 is not exercised. Both runs found that `e2e/playwright/tests/render.spec.ts` already checks the fixture's criterion titles, made no edit, and asked which of several options to take; the second run was made to get an edit and did not. With no edit, there is no read before one to observe. Neither run opened the router.
  - `codex-read-claude`, judged mechanically by the harness, complied in all four Codex runs: each saw every line of `CLAUDE.md` before its first edit.
- **Environment:** the aiwf devcontainer, Linux, runs ending 2026-10-11T14:20Z to 15:23Z.
  - checkout `95c37637b`, installed guidance `9c9ca4b3681ca4cb8798e5af9e9124b1e761526b`; `aiwf` built at the checkout, `v0.0.0-20261011140621-95c37637b0d7`;
  - Claude Code 2.1.291 at `claude-opus-5-5`, effort `high`; Codex CLI 0.162.1 at `gpt-6-astra`, effort `medium`;
  - the personal overlay: `~/.guidance` at `d7180c35589fa8bd0d1d69df227d8b358862a82e` with no dirty files. It differs from the baseline's `334e60cd8`, and Codex CLI from its 0.160.0, so these runs are not a comparison with M-0334's; M-0338 makes that comparison.

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

Three listed readers that were not pins read moved text too: the skill-edit absence test, then `TestSkillEditProvenance_DocumentedInClaudeMd` and now `TestSkillEditProvenance_RetiredMandateAbsentFromGuidance`, required two `CLAUDE.md` headings; `TestPolicy_NoDanglingEntityRefsInNarrativeDocs` resolved entity links in `CLAUDE.md` and `ROADMAP.md` only; `TestPolicyDevcontainerPathsResolve` did not read `.guidance/`. Both relationship checks now read the development-guidance set, and the absence check scans every guidance document.

### AC-5 — no nested instruction file

`PolicyInstructionFilesAtRoot` lists tracked files with `git ls-files` and reports a `CLAUDE.md` or `AGENTS.md` anywhere but the root, by path. `TestInstructionFilesBelowRoot` pins the rule, `TestPolicyInstructionFilesAtRoot_ReadsTrackedFiles` pins that it judges tracked files and that a directory git cannot list is an error, and on the tree `TestPolicy_InstructionFilesAtRoot` reports none.

### AC-6 — dispositions applied

`TestE0092Inventory` passes: every removal block since M-0336's start names its row and carries the row's decision, and each destination it names exists; `TestInventoryReconcileFindings` pins that a removal naming a destination that does not exist is reported. The rewording was read at the wrap review against each tighten row's list of what it must carry.

## Deferrals

- G-0735 — comments, a test message and an ADR cite `CLAUDE.md` rules under names or numbers it does not carry; none names a section this milestone moved.
- G-0736 — the contribution templates ask for a checklist, an audit and commands the repository does not hold where they point.

## Reviewer notes

- (none)
