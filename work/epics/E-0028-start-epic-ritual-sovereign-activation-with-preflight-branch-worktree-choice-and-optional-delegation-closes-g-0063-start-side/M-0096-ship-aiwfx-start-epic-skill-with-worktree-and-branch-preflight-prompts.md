---
id: M-0096
title: Ship aiwfx-start-epic skill with worktree and branch preflight prompts
status: in_progress
parent: E-0028
depends_on:
    - M-0094
    - M-0095
tdd: required
acs:
    - id: AC-1
      title: Fixture exists with valid skill frontmatter and 10-step Workflow section
      status: met
      tdd_phase: done
    - id: AC-2
      title: Worktree-placement prompt is heading-scoped Q&A with three named options
      status: met
      tdd_phase: done
    - id: AC-3
      title: Sovereign-promotion step names the M-0095 rule and the override path
      status: met
      tdd_phase: done
    - id: AC-4
      title: Branch prompt is heading-scoped Q&A with G-0059 deferral note
      status: met
      tdd_phase: done
    - id: AC-5
      title: Drift-check test compares fixture to cache; skips cleanly when absent
      status: met
      tdd_phase: done
---

# M-0096 — Ship `aiwfx-start-epic` skill with worktree and branch preflight prompts

## Goal

Ship the `aiwfx-start-epic` ritual upstream in the `aiwf-extensions` rituals plugin. The skill orchestrates G-0063's preflight + sovereign promotion + optional delegation flow, with two new deliberate Q&A choices at start time: **worktree placement** and **branch shape**. Authored via the canonical fixture pattern (`internal/policies/testdata/aiwfx-start-epic/SKILL.md`) per CLAUDE.md; copied to the rituals repo at wrap; drift-checked against the local plugin cache.

## Context

M-0094 and M-0095 land the kernel chokepoints this skill relies on (`epic-active-no-drafted-milestones` finding; sovereign-act enforcement on `aiwf promote E-NN active`). With those in place, the skill's preflight has real signals to lean on instead of LLM-honor checks. The skill itself is the human-facing surface that closes G-0063's start-side concerns.

The worktree-placement prompt and the branch prompt are deliberately separate Q&A steps (Decision 4 of the planning conversation; recorded in the epic spec). The branch prompt is a placeholder pending G-0059's resolution — it asks rather than defaults, so the operator stays sovereign over the choice until a kernel-defaulted branch convention lands.

## Acceptance criteria

(ACs allocated at `aiwfx-start-milestone` time per the planner-skill convention.)

## Expected shape

- **Fixture-side authoring** — `SKILL.md` lives at `internal/policies/testdata/aiwfx-start-epic/SKILL.md` during the milestone; structural AC tests under `internal/policies/m0096_test.go` (or similar) assert content claims (presence of the worktree-placement section, the branch section, the delegation section, the sovereign-promotion step, the hand-off step). Per CLAUDE.md *Substring assertions are not structural assertions*, the assertions are heading-scoped, not flat greps.
- **Skill body** — the 10-step orchestration laid out in E-0028's scope. The worktree and branch prompts are explicit Q&A with numbered options (matching the project's existing Q&A convention).
- **Drift-check test** — asserts the fixture content matches the local marketplace cache (`~/.claude/plugins/cache/ai-workflow-rituals/.../SKILL.md`) when present; skips cleanly when absent. Matches the M-0090 precedent.
- **Wrap step** — at milestone wrap, copy the fixture content to `/Users/peterbru/Projects/ai-workflow-rituals/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md` as a separate commit there; record that commit's SHA in this milestone's *Validation* section.
- **Follow-up gap at epic wrap** — file the wrap-side concerns (scope-end-before-done + ADR + `aiwfx-wrap-epic` update + human-only enforcement on `done`) as a gap referencing E-0028.

## Dependencies

- **M-0094** — `epic-active-no-drafted-milestones` finding. The skill's drafted-milestone preflight step consumes it.
- **M-0095** — sovereign-act enforcement on `aiwf promote E-NN active`. The skill's promotion step runs against the new refusal rule; the human-actor path is the default.

## References

- E-0028 epic spec — full skill orchestration laid out in *Scope → In scope → Rituals plugin skill*.
- G-0063 — gap framing, preflight checks, sub-decisions.
- M-0090 — precedent for cross-repo SKILL.md fixture + drift-check + wrap-time SHA recording.
- CLAUDE.md *Cross-repo plugin testing* — convention for SKILL.md authoring location.
- CLAUDE.md *AC promotion requires mechanical evidence* — structural assertions over fixture content.

### AC-1 — Fixture exists with valid skill frontmatter and 10-step Workflow section

The skill fixture lives at `internal/policies/testdata/aiwfx-start-epic/SKILL.md` per CLAUDE.md *Cross-repo plugin testing*. Test asserts frontmatter `name: aiwfx-start-epic` plus non-empty `description:`, and the body contains a `## Workflow` section with exactly the integers 1..10 appearing as `### N.` subheadings — no gaps, no extras. Structural enumeration via regex against the Workflow-section body (not flat substring grep over the file) so a renumbering, missing step, or step floated into the wrong section fires the assertion.

### AC-2 — Worktree-placement prompt is heading-scoped Q&A with three named options

Test locates the worktree-prompt subsection by walking `## Workflow`'s `###` headings for one whose text contains "worktree" (case-insensitive) — locator is heading-content driven, not step-number driven, so a future reshuffle of step ordering does not silently break the structural drift check. Inside that subsection, three named placement options must appear by signature substring: *no worktree (work on main)* (prose, case-insensitive), `.claude/worktrees/` (path literal), and `../aiwf-` (path literal). The three together are the prompt's signature; any pair could appear in unrelated prose but all three only inside the prompt.

### AC-3 — Sovereign-promotion step names the M-0095 rule and the override path

Test locates the sovereign-promotion subsection by walking `## Workflow` for a heading whose text contains both "sovereign" and "promot" (case-insensitive). Inside that subsection, three structural claims must appear: the activation verb (`aiwf promote E-NN active`), the rule substance (`human/` — the actor requirement M-0095 enforces), and the override path (`--force --reason`). Substance, not id — the test asserts the *content* of the M-0095 rule rather than the literal "M-0095" string, so a future supersession that moves the mechanical chokepoint doesn't break the assertion.

### AC-4 — Branch prompt is heading-scoped Q&A with G-0059 deferral note

Test locates the branch-prompt subsection by walking `## Workflow` for a heading containing "branch" — with a worktree-exclusion guard, since the worktree section's heading may reference `<branch>/` paths. Inside the located subsection three structural claims hold: a stay-on-current option (prose match `stay on`, case-insensitive), a create-new option (`create`, case-insensitive), and the literal `G-0059` — the latter is the load-bearing signal that documents in-skill that the prompt is a *placeholder* pending the gap's resolution, not a settled convention.

### AC-5 — Drift-check test compares fixture to cache; skips cleanly when absent

Test in `internal/policies/aiwfx_start_epic_test.go::TestAiwfxStartEpic_AC5_DriftAgainstCache` (mirrors the M-0090 precedent). Reads `~/.claude/plugins/installed_plugins.json` to resolve the active `aiwf-extensions@ai-workflow-rituals` install path, then resolves `<installPath>/skills/aiwfx-start-epic/SKILL.md` and compares byte-for-byte against the fixture. Skips cleanly in three legitimate "absent" states: manifest missing (CI without plugin install), plugin not installed, skill not yet materialised in the active install (pre-wrap state — the rituals-repo copy lands at M-0096 wrap). Fails only on actual drift between cached bytes and fixture bytes; post-wrap the test becomes the long-term chokepoint detecting drift in either direction.

## Work log

<!-- Phase timeline lives in `aiwf history M-0096/AC-<N>`; the entries here capture
     one-line outcomes + the implementing commit's SHA (filled at wrap when the
     implementation lands as a single commit). -->

### AC-1 — Fixture exists with valid skill frontmatter and 10-step Workflow section

Fixture authored at `internal/policies/testdata/aiwfx-start-epic/SKILL.md` (~134 lines: frontmatter + Principles + Precondition + 10-step Workflow + Constraints + Anti-patterns + Next step). Test in `internal/policies/aiwfx_start_epic_test.go::TestAiwfxStartEpic_AC1_FixtureAndWorkflow` parses frontmatter via the existing `frontmatterField` helper, extracts the `## Workflow` section via `extractMarkdownSection(body, 2, "Workflow")`, and regex-enumerates `### \d+\.\s` headings — asserts the integer set is exactly {1..10}. · commit <wrap> · tests 1/1.

### AC-2 — Worktree-placement prompt is heading-scoped Q&A with three named options

New helper `findWorktreePromptSection` (heading-content driven, not step-number driven) locates step 5's subsection by walking `## Workflow`'s `###` headings for one mentioning "worktree". Test asserts three placement markers (`no worktree` prose, `.claude/worktrees/` path, `../aiwf-` path) — prose markers case-insensitive, path markers case-sensitive. Branch-coverage extra exercises the helper's two defensive return arms (missing `## Workflow`, no matching heading). · commit <wrap> · tests 2/2 (happy + 2-subcase branch-coverage).

### AC-3 — Sovereign-promotion step names the M-0095 rule and the override path

New helper `findSovereignPromotionSection` locates step 8's subsection by walking for a heading containing both "sovereign" and "promot" (case-insensitive). Test asserts three structural claims inside the section: `aiwf promote E-NN active` (the verb), `human/` (the rule substance), `--force --reason` (the override path). Test asserts the rule's *substance*, not the literal "M-0095" string — a future ADR or milestone-id reallocation that moves the chokepoint does not break the assertion. Branch-coverage extra covers the helper's two defensive arms. · commit <wrap> · tests 2/2.

### AC-4 — Branch prompt is heading-scoped Q&A with G-0059 deferral note

New helper `findBranchPromptSection` walks `## Workflow` for a heading containing "branch", with a worktree-exclusion guard so the worktree section (whose path literal `<branch>/` mentions "branch") does not match. Test asserts three claims inside the located section: a stay-on-current option (`stay on`, case-insensitive), a create-new option (`create`, case-insensitive), and the literal `G-0059` — the latter is the load-bearing signal that documents the prompt as a *placeholder* pending the branch-model gap's resolution. Branch-coverage extra covers three arms: missing workflow, no branch heading, only a worktree heading mentioning `<branch>/`. · commit <wrap> · tests 2/2.

### AC-5 — Drift-check test compares fixture to cache; skips cleanly when absent

`TestAiwfxStartEpic_AC5_DriftAgainstCache` reads `~/.claude/plugins/installed_plugins.json`, resolves the active `aiwf-extensions@ai-workflow-rituals` install path, then resolves `<installPath>/skills/aiwfx-start-epic/SKILL.md` and compares byte-for-byte against the fixture. Skip arms exercised in current state (manifest present, plugin installed, skill not yet materialised → pre-wrap skip path). Drift arm exercised post-wrap once the rituals-repo carries the fixture. · commit <wrap> · tests 1/1 (skips cleanly pre-wrap by design).

## Decisions made during implementation

- **Heading-content locator over step-number locator.** Each of the three `find<X>PromptSection` / `find<X>PromotionSection` helpers locates its target by heading-content match (case-insensitive substring of a domain word like "worktree", "sovereign+promot", "branch") rather than by step number (`### 5.`, `### 6.`, `### 8.`). This makes the structural drift checks tolerant of step-renumbering without weakening the assertion — the heading words ARE the structural claim. Precedent: `findMergeStepSection` in `aiwfx_wrap_epic_test.go` (M-0090).
- **Drift-check skips on "skill not yet materialised" rather than failing.** Diverges slightly from M-0090's pattern (which fails on "not materialised"). The rationale: the AC-5 test's job is to detect *drift between two existing copies*, not to police whether deployment has happened. A pre-wrap milestone legitimately has the fixture in this repo and nothing in the rituals repo; treating that as failure would force red-on-pre-wrap which is noise. Post-wrap the test becomes the long-term drift chokepoint exactly as M-0090's does.
- **AC test order in the file follows the ritual rhythm, not numeric order.** The file's top-down order is: AC-1 (fixture frame), AC-5 (drift check, similarly file-level), then the three structural-content tests AC-2/AC-3/AC-4 each next to their respective `find<X>` helper. The numerically-out-of-order grouping co-locates each helper with its test, which is the right read-flow for the file.
- **Substance-over-id for the AC-3 assertion.** AC-3's test asserts the *content* of the M-0095 rule (the `human/` requirement and the `--force --reason` override) rather than the literal "M-0095" string. A future ADR ratification or milestone-id reallocation that moves the mechanical chokepoint should not break the assertion; what readers land on the section to learn is the rule's substance, not its bookkeeping id.
- **Drift-check pattern aligns with M-0090, with one variant.** Same overall shape as `TestAiwfxWrapEpic_AC3_CacheComparison` — `installed_plugins.json` parsing, install-path resolution, byte-compare against fixture. The variant (skip on not-materialised instead of fail) is documented above and in the test's docstring.

## Validation

- `go test -race -count=1 ./...` — 25 packages, 0 FAIL lines, exit 0.
- `golangci-lint run ./internal/policies/` — 0 issues.
- `go test -coverprofile=… ./internal/policies/` — 82.3% of statements; the new helpers (`findWorktreePromptSection`, `findSovereignPromotionSection`, `findBranchPromptSection`) each at 100% branch coverage via the dedicated `_BranchCoverage` table tests.
- `aiwf check` (kernel planning tree from the worktree) — 0 errors; 4 advisory warnings (terminal-entity-not-archived × 2 for M-0094/M-0095 awaiting sweep; archive-sweep-pending × 1; provenance-untrailered-scope-undefined × 1 for no-upstream worktree branch). None block wrap.
- Doc-lint sweep against M-0096's change-set — clean. Every cross-reference (verbs, skills, finding codes, entity ids) resolves; no orphans, no TODOs.
- `wf-doc-lint` (scoped) — no findings.
- **Rituals-repo SHA recorded at wrap:** `87fc790` (`87fc79088d98fa0acf87bab2eb9b3c3641190bf0`) on `ai-workflow-rituals` `main` — subject `feat(aiwfx-start-epic): introduce start-epic ritual (aiwf E-0028 / M-0096)`, +133/-0 in `plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md`. Local-only; the rituals-repo push is a separate gate per CLAUDE.md *Executing actions with care*.

## Deferrals

- (none) — every AC met in this milestone.

## Reviewer notes

- **Cross-repo wrap step.** Per CLAUDE.md *Cross-repo plugin testing*, the fixture content at `internal/policies/testdata/aiwfx-start-epic/SKILL.md` is the canonical authoring location during the milestone. At wrap, the content is copied to the rituals repo at `/Users/peterbru/Projects/ai-workflow-rituals/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md`, committed there as a separate commit, and that commit's SHA is recorded above in *Validation*. The drift-check test (`TestAiwfxStartEpic_AC5_DriftAgainstCache`) flips from skip-clean to active drift-check once the rituals-repo copy lands and `/reload-plugins` runs locally.
- **`aiwfx-start-epic` is not yet registered.** The rituals plugin's `plugin.json` may need a registration entry for the new skill — this is handled in the rituals repo, not here. The wrap-step copy includes adding the skill to whatever the rituals-repo plugin manifest needs (or confirming SKILL.md drop alone suffices, since the existing skills in the same directory are auto-discovered by Claude Code).
- **Skill is not invocable in this session.** Even after the rituals-repo copy lands, the skill won't be invocable in *this* Claude session until `/reload-plugins` runs. That's expected; the AC tests assert the fixture's structural shape, not its invocability — invocability is the operator's verification path after wrap.
- **Why the AC-5 drift-check pre-wrap skip is a design choice, not a bug.** M-0090's equivalent test FAILS when the skill is not materialised. M-0096's SKIPS. The rationale is in *Decisions made during implementation* above: the test's job is to detect drift between two existing copies, not to police the wrap step's completion. A future cleanup that aligned the two tests' shapes is fine but not required — both shapes serve the long-term drift-detection role correctly.
- **Trailer convention for the wrap commit** — same as M-0094 / M-0095 precedent: `aiwf-verb: implement`, `aiwf-entity: M-0096`, `aiwf-actor: human/peter`. The `implement` verb is a synthetic trailer-only marker (no CLI surface).
- **Branch-coverage hard rule** — every reachable branch in each new helper has a positive test (happy-path AC tests) plus an explicit `_BranchCoverage` table test for the defensive arms. The drift-check test (`AC-5`) has reachable branches whose coverage depends on the runtime environment — the skip arms are exercised in current state; the drift-detection arm is exercised post-wrap and in production. This mirrors M-0090's accepted shape.

