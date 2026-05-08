# AI Scaffolding Policy Candidates: Liminara Framework Mining

**Mined:** 2026-05-03 | **Source:** `/Users/peterbru/Projects/liminara/` AI-instruction layer

---

## CLAUDE.md (562 lines)

1. **commit-explicit-approval-gate** — Must never run `git commit` or `git push` without explicit human approval. "Continue", "ok", "looks good" do NOT count.
   - Bindingness: MUST | Audience: AI agent | Category: general-engineering
   - Cross-refs: `.ai/rules.md` (same rule) + `.ai/agents/builder.md`, `.ai/agents/reviewer.md`

2. **tdd-by-default** — Apply test-driven development (red → green → refactor) for logic, API, and data code.
   - Bindingness: SHOULD | Audience: both | Category: general-engineering
   - Note: "by default" suggests exceptions may apply; full enforcement in TDD-conventions file

3. **branch-coverage-hard-rule** — Every reachable conditional branch must have an explicit test before declaring done. Line-by-line audit before commit-approval prompt, not after human asks.
   - Bindingness: MUST | Audience: both | Category: general-engineering

4. **identify-agent-first** — Read the agent file at session start; adopt its role; follow its skill.
   - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

5. **branch-discipline** — Do NOT commit milestone work directly to `main`.
   - Bindingness: MUST | Audience: both | Category: general-engineering

6. **update-claude-md-current-work** — Update CLAUDE.md Current Work section after starting or wrapping a milestone.
   - Bindingness: SHOULD | Audience: both | Category: workflow-pm
   - Note: Also in `.ai/rules.md` as a governance check

7. **conventional-commits-format** — Use Conventional Commits format: `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `refactor:`.
   - Bindingness: MUST | Audience: both | Category: general-engineering

8. **artifact-layout-from-config** — Resolved artifact layout comes from framework defaults in `.ai/paths.md` and repo overrides in `.ai-repo/config/artifact-layout.json`.
   - Bindingness: MUST | Audience: AI agent | Category: meta
   - Note: Single source of truth for path resolution

9. **scratch-path-convention** — Scratch files go in `.ai-repo/scratch/<epic-id>/<milestone-id>/`, not `/tmp`. Cleaned up at wrap-milestone / wrap-epic.
   - Bindingness: MUST | Audience: both | Category: general-engineering
   - Cross-refs: `.ai/rules.md` § Scratch Files

10. **decisions-shared-log** — `work/decisions.md` is the shared decision log across all agents.
    - Bindingness: MUST | Audience: both | Category: workflow-pm

11. **agent-history-per-role** — Each agent role has its own accumulated-learnings file at `work/agent-history/<role>.md` (read-only unless current agent).
    - Bindingness: SHOULD | Audience: AI agent | Category: workflow-pm

12. **gaps-deferred-work** — Discovered gaps and deferred work go to `work/gaps.md`.
    - Bindingness: SHOULD | Audience: both | Category: workflow-pm

13. **session-start-pickup** — At session start, read `work/decisions.md`, `work/agent-history/<role>.md`, `work/gaps.md`, and CLAUDE.md Current Work section.
    - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

14. **refresh-context-on-config-change** — If rules or project config changed mid-session, user says "refresh context" to trigger full re-read.
    - Bindingness: MAY | Audience: both | Category: meta

15. **agent-routing-by-intent** — Intent determines which agent to use: builder (build/implement/fix), planner (plan/design/scope), reviewer (review/validate), deployer (release/deploy).
    - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

16. **roadmap-canonical-sequencing** — `work/roadmap.md` is the only current sequencing and build-plan source.
    - Bindingness: MUST | Audience: both | Category: project-specific

17. **artifact-layout-config-canonical** — `.ai-repo/config/artifact-layout.json` is the canonical source for roadmap, epic, milestone, and tracking paths.
    - Bindingness: MUST | Audience: AI agent | Category: project-specific

18. **sync-framework-via-script** — To change AI instruction behavior, edit `.ai-repo/` and run `./.ai/sync.sh`. Do not hand-edit generated files except CLAUDE.md Current Work section.
    - Bindingness: MUST | Audience: both | Category: meta

19. **generated-surfaces-mirror-config** — Generated assistant surfaces (under `.claude/`, `.github/`) must mirror resolved config values rather than redefine them.
    - Bindingness: MUST | Audience: AI agent | Category: meta

20. **docs-architecture-live-or-next** — `docs/architecture/` contains only live or decided-next architecture; historical material goes to `docs/history/`.
    - Bindingness: MUST | Audience: both | Category: project-specific

21. **history-is-context-not-authority** — `docs/history/` is context, not authority.
    - Bindingness: MUST | Audience: both | Category: project-specific

22. **current-behavior-wins-disputes** — If current behavior is disputed, live code, tests, and canonical persistence specs win.
    - Bindingness: MUST | Audience: both | Category: project-specific

23. **approved-next-state-wins** — If approved next-state behavior is disputed, active epic/milestone spec + decided-next architecture docs win.
    - Bindingness: MUST | Audience: both | Category: project-specific

24. **compatibility-shims-banned** — Compatibility shims are banned by default; any exception needs a named removal trigger in milestone spec + tracking doc.
    - Bindingness: MUST | Audience: both | Category: project-specific

25. **contract-matrix-is-truth** — `docs/architecture/indexes/contract-matrix.md` is the live ownership/status index for every first-class contract surface.
    - Bindingness: MUST | Audience: both | Category: project-specific

26. **contract-matrix-plan-time-declaration** — Any milestone that creates, modifies, or retires a contract surface MUST include `## Contract matrix changes` section in its spec.
    - Bindingness: MUST | Audience: both | Category: project-specific

27. **contract-matrix-wrap-time-check** — Before wrapping a milestone, reviewer verifies declared rows are present in `contract-matrix.md` with correct live-source paths. Row absence blocks wrap.
    - Bindingness: MUST | Audience: both | Category: project-specific

28. **contract-matrix-live-source-accuracy** — When a live-source file is renamed/moved/deleted/extracted, the same PR updates the row.
    - Bindingness: MUST | Audience: both | Category: project-specific

29. **doc-tree-bind-me-vs-inform-me** — `docs/` tree has two registers: bind-me (governance + schemas) rejects wrong work; inform-me (architecture + research) informs right work.
    - Bindingness: MUST | Audience: both | Category: project-specific

30. **implementation-gates-architecture-guides** — When doing work, respect implementation artifacts (schemas, governance) as hard surface; read architecture as context.
    - Bindingness: MUST | Audience: both | Category: project-specific

31. **no-docs-specs-directory** — There is deliberately no `docs/specs/` directory. Word ambiguity resolved by location.
    - Bindingness: MUST | Audience: both | Category: project-specific

32. **no-single-contracts-subtree** — Contract components live in separate directories, not under single `contracts/` subtree.
    - Bindingness: MUST | Audience: both | Category: project-specific

33. **author-sequenced-thinking-convention** — Files prefixed `NN_<descriptor>.md` are top-tier thinking docs in author sequence. Descriptor case differs by directory.
    - Bindingness: SHOULD | Audience: both | Category: project-specific

34. **supporting-material-subdirs-kebab-case** — Supporting material under docs (indexes, references, derived) lives in named subdirectories with kebab-case filenames.
    - Bindingness: MUST | Audience: both | Category: project-specific

35. **decision-records-two-surfaces** — Framework prescribes two surfaces: `work/decisions.md` (day-to-day, lightweight), `docs/decisions/NNNN-<slug>.md` (ADRs, heavy ratifying).
    - Bindingness: SHOULD | Audience: both | Category: workflow-pm

36. **adr-nygard-pattern** — ADRs follow Michael Nygard 2011 pattern (Context → Decision → Consequences + status vocabulary).
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

37. **validation-pipeline-per-language** — Before any commit, the appropriate validation must pass (Elixir: mix format/credo/dialyzer/test; Python: ruff/ty/pytest; etc.).
    - Bindingness: MUST | Audience: both | Category: general-engineering

38. **one-worktree-per-epic** — One worktree per epic, not per milestone.
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

39. **epic-branch-naming** — Epic branch: `epic/<slug>`; milestone branch: `milestone/<id>` from epic branch.
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

40. **agents-never-push-without-approval** — Agents never push without human approval.
    - Bindingness: MUST | Audience: AI agent | Category: general-engineering

41. **merge-strategy-squash** — Merge strategy: squash per milestone (or per epic if milestones are small).
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

42. **q-and-a-mode-protocol** — When user says "Q&A", switch to Q&A mode: short context para → pros/cons → lean → numbered options. One question at a time.
    - Bindingness: MAY | Audience: AI agent | Category: agent-behavior

43. **never-assume-ambiguous-decisions** — Never make assumptions on ambiguous decisions. If unclear or multi-way, stop and ask.
    - Bindingness: MUST | Audience: AI agent | Category: agent-behavior

44. **tech-stack-binding** — Stack: Elixir/OTP, ETS, JSONL, ex_a2ui (Bandit + WebSock), Phoenix LiveView, Ports/containers, Python via `:port`.
    - Bindingness: MAY | Audience: both | Category: project-specific

---

## .ai/rules.md (143 lines)

45. **commits-hard-gate-summary** — NEVER run `git commit` or `git push` without explicit human approval. No agent, skill, or workflow may bypass this.
    - Bindingness: MUST | Audience: AI agent | Category: general-engineering

46. **approval-explicit-words** — What counts as approval: "commit", "go ahead and commit", "push it", "merge it", "yes, commit". NOT: "continue", "ok", "looks good", "next step", "finish up".
    - Bindingness: MUST | Audience: AI agent | Category: general-engineering

47. **commit-workflow-before-commit** — Required: (1) stage changes, (2) show what will be committed, (3) propose message, (4) STOP and wait for "commit", (5) run commit, (6) STOP and wait for push approval.
    - Bindingness: MUST | Audience: AI agent | Category: general-engineering

48. **commit-format-conventional-commits** — Use Conventional Commits v1.0.0: `feat:`, `fix:`, `chore:`, `docs:`, `test:`, `refactor:`.
    - Bindingness: MUST | Audience: both | Category: general-engineering

49. **commit-coauthor-trailer** — Add AI co-author trailer identifying which assistant wrote the commit. Mapping in `.ai-repo/config/commit.json`.
    - Bindingness: SHOULD | Audience: both | Category: general-engineering
    - Cross-refs: `.ai-repo/config/commit.json` for coAuthor mapping

50. **commit-message-imperative-mood** — Keep subject line under 72 characters, use imperative mood.
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

51. **tests-deterministic** — Tests must be deterministic — no external network calls, no time-dependent assertions.
    - Bindingness: MUST | Audience: both | Category: general-engineering

52. **tdd-write-test-first** — Write failing test → make it pass → refactor.
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

53. **branch-coverage-law** — Cover every acceptance criterion and every reachable conditional branch. Branch coverage is the rule, not a coverage percentage.
    - Bindingness: MUST | Audience: both | Category: general-engineering

54. **minimal-precise-edits** — Prefer minimal, precise edits over broad refactors.
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

55. **build-test-before-handoff** — Build and test must pass before any handoff or PR.
    - Bindingness: MUST | Audience: both | Category: general-engineering

56. **no-secrets-in-prompts** — Never paste secrets, tokens, or credentials into prompts, docs, or logs.
    - Bindingness: MUST | Audience: both | Category: general-engineering

57. **no-customer-data-in-examples** — No customer data or PII in examples — use sanitized fixtures.
    - Bindingness: MUST | Audience: both | Category: general-engineering

58. **new-dependencies-approval** — New dependencies require human approval; flag packages < 1 year old or < 100 stars.
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

59. **docs-aligned-with-contracts** — Keep docs aligned when touching contracts or schemas.
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

60. **mermaid-for-diagrams** — Use Mermaid for diagrams (not ASCII art).
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

61. **repo-language-english** — Repository language: English.
    - Bindingness: MUST | Audience: both | Category: general-engineering

62. **reference-phrasing-not-counts** — Prefer reference-phrasing over hand-written list counts when spec/roadmap bullets reference lists elsewhere.
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

63. **claude-md-current-work-narrative-only** — CLAUDE.md Current Work captures intent (active focus, why now, parked, open question) — not structural enumerations.
    - Bindingness: SHOULD | Audience: both | Category: workflow-pm

64. **current-work-max-15-lines** — Target length for Current Work: ≤ 15 lines.
    - Bindingness: SHOULD | Audience: both | Category: workflow-pm

65. **no-history-rewriting-unless-instructed** — No history-rewriting or destructive git operations unless explicitly instructed.
    - Bindingness: MUST | Audience: both | Category: general-engineering

66. **check-dirty-submodules** — Check for dirty submodule pointers before committing.
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

67. **scratch-files-in-scratch-dir** — Scratch files go in work-unit's scratch dir (resolved from `scratchPath`), not `/tmp`.
    - Bindingness: MUST | Audience: both | Category: general-engineering

68. **scratch-dir-gitignored** — Scratch dir is gitignored, inspectable, and cleaned up deterministically at wrap.
    - Bindingness: MUST | Audience: both | Category: general-engineering

69. **no-scratch-in-tracked-trees** — Do not write scratch anywhere under `work/`, `docs/`, or project source tree.
    - Bindingness: MUST | Audience: both | Category: general-engineering

70. **scratch-never-referenced** — Never reference scratch contents from committed docs, tracking files, or spec bodies.
    - Bindingness: MUST | Audience: both | Category: general-engineering

71. **gaps-append-to-gaps-file** — Discovered gaps → add to `work/gaps.md`; defer by default.
    - Bindingness: SHOULD | Audience: both | Category: workflow-pm

72. **agent-learnings-append-only** — Agent learnings → `work/agent-history/<agent>.md` (per-agent, append-only).
    - Bindingness: SHOULD | Audience: both | Category: workflow-pm

73. **history-files-archiving** — When history files exceed ~200 lines, summarize older entries and archive.
    - Bindingness: SHOULD | Audience: both | Category: workflow-pm

74. **adr-ids-monotonic** — ADR IDs are `ADR-NNNN` — monotonic sequence, zero-padded; no keyword scopes.
    - Bindingness: MUST | Audience: both | Category: general-engineering

75. **adr-keyword-slug-not-id** — Keywords belong in the filename slug, not the ID.
    - Bindingness: MUST | Audience: both | Category: general-engineering

76. **adr-template-nygard** — Scaffold new ADRs from repo's `adrTemplatePath` (default `.ai/templates/adr.md`); follows Michael Nygard 2011 pattern.
    - Bindingness: SHOULD | Audience: both | Category: general-engineering

77. **adr-status-vocabulary-closed** — ADR status vocabulary is closed: `proposed | accepted | deprecated | superseded | draft | rejected`. No other values.
    - Bindingness: MUST | Audience: both | Category: general-engineering

78. **adr-status-history-optional** — Optional Status history section for lifecycle events that don't merit supersession.
    - Bindingness: MAY | Audience: both | Category: general-engineering

79. **epic-status-vocabulary-closed** — Epic status vocabulary is closed: `proposed | planning | active | complete | absorbed | deprecated`. No other values.
    - Bindingness: MUST | Audience: both | Category: general-engineering

80. **epic-proposed-forward-plan** — `proposed` — row exists in roadmap as forward-planned placeholder; no folder, no spec yet.
    - Bindingness: MAY | Audience: both | Category: project-specific

81. **epic-planning-scope-being-written** — `planning` — committed intent; folder + spec exist; scope being written. Set by `plan-epic`.
    - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

82. **epic-active-in-progress** — `active` — in progress (epic branch open, milestones landing).
    - Bindingness: SHOULD | Audience: both | Category: project-specific

83. **epic-complete-shipped-archived** — `complete` — shipped and archived to `completed/`. Set by `wrap-epic`.
    - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

84. **epic-absorbed-rolled-into-another** — `absorbed` — rolled into another epic; replacement's id goes in frontmatter `absorbed_by:`.
    - Bindingness: MAY | Audience: both | Category: project-specific

85. **epic-deprecated-obsoleted** — `deprecated` — obsoleted without replacement; keep folder for historical trail.
    - Bindingness: MAY | Audience: both | Category: project-specific

86. **epic-draft-not-status** — `draft` is NOT an epic status — reserved for ADRs. Rename legacy `status: draft` epics to `planning`.
    - Bindingness: MUST | Audience: both | Category: project-specific

87. **optional-parent-child-epic-shape** — An epic spec may declare `parent: <epic-id>` for child-of relationship.
    - Bindingness: MAY | Audience: both | Category: project-specific

88. **parent-epic-narrative-cluster** — Parent narrative captures cluster's "why this work belongs together".
    - Bindingness: SHOULD | Audience: both | Category: project-specific

89. **child-epic-own-lifecycle** — Each child epic carries own milestones and lifecycle.
    - Bindingness: SHOULD | Audience: both | Category: project-specific

90. **child-epics-normal-peers** — Children are normal peer epics: own folder, own branch, own milestones.
    - Bindingness: MUST | Audience: both | Category: project-specific

91. **parent-epic-optional-milestones** — A parent epic may have no milestones of its own; its decomposition is its children.
    - Bindingness: MAY | Audience: both | Category: project-specific

92. **subagent-cost-discipline** — When delegating to subagent, pick the cheapest model that can do the job.
    - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

93. **code-gen-planning-review-inherit-parent** — Code generation, planning, review → inherit parent (Opus). Irreducible reasoning load.
    - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

94. **research-lookup-synthesis-sonnet** — Research, lookup, synthesis, scaffold → pass `model: "sonnet"` explicitly.
    - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

95. **codebase-scans-prefer-explore** — Codebase scans / file finding → prefer `Explore` (Haiku 4.5) over `general-purpose`.
    - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

96. **explore-default-quick-thoroughness** — Default `Explore` to quick thoroughness. Escalate only when `quick` leaves real gap.
    - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

97. **conflict-resolution-precedence** — When instructions conflict: explicit user directive > project-specific docs > rules file > agent/skill defaults.
    - Bindingness: SHOULD | Audience: both | Category: meta

---

## .ai-repo/rules/contract-design.md (163 lines)

98. **contract-design-reviewer-enforcement** — Rule is enforced by the reviewer agent at PR review time on PRs that land or modify Liminara contract surfaces.
    - Bindingness: MUST | Audience: both | Category: project-specific

99. **contract-design-upstream-vs-local** — Upstream skill defines authoring; this rule enforces reviewer-side acceptance gates; overlay binds workflow to Liminara's paths.
    - Bindingness: MUST | Audience: both | Category: project-specific

100. **assertion-1-pack-adrs-cite-admin-pack-anchors** — Pack-level ADRs in E-21 must each cite a specific file + section anchor inside `admin-pack/v2/docs/architecture/`.
     - Bindingness: MUST | Audience: both | Category: project-specific

101. **assertion-1-anchor-format** — Citation format: `<file>.md §<section> — <description>`. Generic "see admin-pack" fails.
     - Bindingness: MUST | Audience: both | Category: project-specific

102. **assertion-1-e22-pending-allowance** — During E-21, cited anchor may not exist on disk. Reviewer accepts if (a) specificity named, (b) description articulates what section provides.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

103. **assertion-1-radar-only-exceptions** — ADRs deliberately Radar-only (ADR-WIRE-01, ADR-EYECUTOR-01) are exceptions marked in parent sub-epic spec.
     - Bindingness: MAY | Audience: both | Category: project-specific

104. **assertion-2-contract-matrix-rows-verified** — Every row declared in spec's *Contract matrix changes* section must land in `contract-matrix.md` with correct columns.
     - Bindingness: MUST | Audience: both | Category: project-specific

105. **assertion-2-live-source-paths-checked** — Reviewer checks that live-source path actually exists (no rotted paths).
     - Bindingness: MUST | Audience: both | Category: project-specific

106. **assertion-2-wrap-blocking** — Absent rows or rotted paths are wrap-blocking. Milestone does not wrap until matrix matches declared deltas.
     - Bindingness: MUST | Audience: both | Category: project-specific

107. **assertion-2-defensive-no-touch** — For milestones declaring "None — this milestone does not touch contract surfaces", reviewer verifies milestone indeed didn't touch any first-class surface.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

108. **assertion-3-radar-primary-admin-pack-secondary** — Pack-level contract ADRs must structure references as two tiers: primary (Radar file:line), secondary (admin-pack per Assertion 1).
     - Bindingness: MUST | Audience: both | Category: project-specific

109. **assertion-3-primary-reference-real-code** — Primary reference must be file:line into runtime/apps or committed source. Code must demonstrate contract on real work today, not test/mock.
     - Bindingness: MUST | Audience: both | Category: project-specific

110. **assertion-3-secondary-reference-required** — ADR with only Radar reference and no admin-pack secondary fails as one-pack abstraction.
     - Bindingness: MUST | Audience: both | Category: project-specific

111. **assertion-4-reference-impl-shapes** — `contract.reference_implementation` must be existing (file:line) or scheduled-to-exist (milestone ID + named file). TBD/later/demo rejected.
     - Bindingness: MUST | Audience: both | Category: project-specific

112. **assertion-4-existing-implementation** — Existing: file:line into committed source. Code must be real, running implementation today — not test, not mock, not draft.
     - Bindingness: MUST | Audience: both | Category: project-specific

113. **assertion-4-scheduled-implementation-binding** — Scheduled: milestone ID + named file (e.g. "`examples/file_watch_demo` built in M-DX-03"). Named file binding is deadline.
     - Bindingness: MUST | Audience: both | Category: project-specific

114. **assertion-4-verification-at-cited-milestone-wrap** — Reviewer at wrap-milestone time of *cited* milestone (not ADR's own) verifies named file/module materialized.
     - Bindingness: MUST | Audience: both | Category: project-specific

115. **assertion-4-deadline-slip-handling** — Reference-impl deadlines that slip get either re-cited (decision-log entry) or ADR is reopened.
     - Bindingness: MUST | Audience: both | Category: project-specific

116. **reviewer-not-enforce-authoring-workflow** — Reviewer does not enforce 7-step authoring workflow (draft ADR → schema → fixtures → worked example → reference implementation → verify → PR).
     - Bindingness: MUST | Audience: both | Category: project-specific

117. **reviewer-not-enforce-cue-idioms** — Reviewer does not enforce per-CUE language idioms; reads `cue vet` output, not CUE.
     - Bindingness: MUST | Audience: both | Category: project-specific

118. **reviewer-not-re-run-evolution-loop** — Schema-evolution-loop pass gated by pre-commit hook + CI; reviewer asserts discipline indirectly but doesn't re-run.
     - Bindingness: MUST | Audience: both | Category: project-specific

---

## .ai-repo/rules/liminara.md (204 lines)

119. **liminara-core-definition** — Liminara is a runtime for reproducible nondeterministic computation. Records every nondeterministic choice; enables replay, audit, intelligent caching.
     - Bindingness: MAY | Audience: both | Category: project-specific

120. **artifact-concept-immutable-content-addressed** — Artifact: immutable, content-addressed blob (SHA-256). The edges in the DAG.
     - Bindingness: MAY | Audience: both | Category: project-specific

121. **op-concept-typed-function** — Op: typed function (artifacts in → artifacts out) with a determinism class (pure, pinned_env, recordable, side_effecting).
     - Bindingness: MAY | Audience: both | Category: project-specific

122. **decision-concept-recorded-nondeterminism** — Decision: recorded nondeterministic choice (LLM response, GA selection, human approval, random seed). Enables replay.
     - Bindingness: MAY | Audience: both | Category: project-specific

123. **run-concept-event-log-and-plan** — Run: an execution = append-only event log + plan (DAG of op-nodes). Events are the source of truth.
     - Bindingness: MAY | Audience: both | Category: project-specific

124. **pack-concept-module-with-plan-fn** — Pack: a module providing op definitions and a `plan/1` function. (Reference-data callback `init/0` is approved-next.)
     - Bindingness: MAY | Audience: both | Category: project-specific

125. **governance-vs-rules-distinction** — `.ai-repo/rules/` governs how AI operates workflow (TDD, branch, commit, contract-matrix); `docs/governance/` defines how project artifacts behave.
     - Bindingness: MUST | Audience: both | Category: meta

126. **spec-word-three-senses** — "Spec" used in three senses: milestone specs (acceptance criteria under `work/epics/`), design-intent prose (live/next in `docs/architecture/`), Nygard ratification (ADRs in `docs/decisions/`).
     - Bindingness: SHOULD | Audience: both | Category: project-specific

127. **contract-word-distributed** — Contract components live in separate dirs: policy (this file), matrix (index), shim policy (governance), schemas (`docs/schemas/`), fixtures.
     - Bindingness: MUST | Audience: both | Category: project-specific

128. **plan-time-declaration-milestone-contract-changes** — Any milestone that creates/modifies/retires a contract surface MUST include `## Contract matrix changes` section with three bullets: added, updated, retired.
     - Bindingness: MUST | Audience: both | Category: project-specific

129. **plan-time-declaration-missing-blocks-approval** — Missing contract-matrix-changes section blocks spec approval.
     - Bindingness: MUST | Audience: both | Category: project-specific

130. **wrap-time-check-matrix-accuracy** — Before wrapping a milestone with declared matrix changes, reviewer verifies rows present in `contract-matrix.md` with correct live-source paths.
     - Bindingness: MUST | Audience: both | Category: project-specific

131. **wrap-time-check-row-absence-blocks** — Row absence blocks wrap.
     - Bindingness: MUST | Audience: both | Category: project-specific

132. **live-source-accuracy-same-pr** — When live-source file is renamed/moved/deleted/extracted, same PR updates row.
     - Bindingness: MUST | Audience: both | Category: project-specific

133. **live-source-drift-reviewer-miss** — Finding drift after merge is a reviewer miss and should be noted in agent history.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

134. **matrix-vs-adr-boundary** — Matrix row points at what contract is and where live source lives; ADR explains why contract has that shape. Two cross-reference but never overlap.
     - Bindingness: MUST | Audience: both | Category: project-specific

135. **roadmap-only-sequencing-source** — `work/roadmap.md` is the only current sequencing and build-plan source.
     - Bindingness: MUST | Audience: both | Category: project-specific

136. **artifact-layout-config-source-of-truth** — `.ai-repo/config/artifact-layout.json` is canonical for roadmap, epic, milestone, and tracking paths.
     - Bindingness: MUST | Audience: both | Category: project-specific

137. **generated-surfaces-mirror-not-redefine** — Generated assistant surfaces must mirror resolved config rather than redefine it.
     - Bindingness: MUST | Audience: AI agent | Category: meta

138. **framework-changes-via-sync-script** — To change AI instruction behavior, edit `.ai-repo/` and run `./.ai/sync.sh`. Do not hand-edit generated files except CLAUDE.md Current Work.
     - Bindingness: MUST | Audience: both | Category: meta

139. **decisions-two-surface-policy** — Framework prescribes two decision surfaces (`.ai/rules.md`, `.ai/paths.md`, `.ai/skills/wrap-epic.md`, `.ai/skills/workflow-audit.md`).
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

140. **work-decisions-lightweight-daily** — `work/decisions.md` — day-to-day structured entries (id, status, context, decision, consequences). Lightweight, fast to write, reviewed in-session.
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

141. **docs-decisions-heavy-ratifying** — `docs/decisions/NNNN-<slug>.md` (ADRs) — heavier ratifying records surfaced at wrap-epic. Scope: first-class boundaries, constraint changes, shim justifications, supersessions.
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

142. **when-in-doubt-ask-future-reader** — When in doubt between two decision surfaces, ask: "would a future reader regret missing the reasoning?" — if yes, write an ADR.
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

143. **validation-pipeline-languages** — Before any commit, appropriate validation must pass (Elixir/Python/JavaScript/TypeScript per language).
     - Bindingness: MUST | Audience: both | Category: general-engineering

144. **commit-conventional-commits-format** — Follow Conventional Commits v1.0.0. Include Co-Authored-By when GitHub Copilot contributed.
     - Bindingness: MUST | Audience: both | Category: general-engineering

145. **git-workflow-one-worktree-per-epic** — One worktree per epic, not per milestone.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

146. **epic-branch-naming-convention** — Epic branch: `epic/<slug>`; milestone branch: `milestone/<id>` from epic branch.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

147. **agents-never-push-without-approval-enforcement** — Agents never push without human approval.
     - Bindingness: MUST | Audience: AI agent | Category: general-engineering

148. **merge-strategy-squash-per-milestone** — Merge strategy: squash per milestone (or per epic if milestones are small).
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

149. **submodules-binding** — Three named submodules: `dag-map`, `ex_a2ui`, `proliminal.net`.
     - Bindingness: MAY | Audience: both | Category: project-specific

150. **project-structure-binding** — Named directories: `docs/`, `docs/governance/`, `docs/schemas/`, `docs/architecture/`, `docs/history/`, `docs/analysis/`, `docs/decisions/`, `runtime/`, `work/`, `work/done/`.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

151. **domain-packs-target-sequence** — Target sequence: (1) Radar, (2) VSME, (3) House Compiler, (4) DPP.
     - Bindingness: MAY | Audience: both | Category: project-specific

---

## .ai-repo/rules/tdd-conventions.md (103 lines)

152. **test-coverage-categories-always-write** — Always write: happy path, edge cases, error cases.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

153. **test-coverage-categories-when-applicable** — Write when applicable: round-trip, tamper detection, format compliance, invariants, isolation.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

154. **implementation-minimum-code** — Write the minimum code to make tests pass. No features beyond what tests require.
     - Bindingness: MUST | Audience: both | Category: general-engineering

155. **implementation-dont-modify-tests** — Do not modify test files unless they have a clear bug. If you must, explain why.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

156. **implementation-follow-code-style** — Follow the existing code style. Read neighboring files before writing new code.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

157. **implementation-prefer-simple-code** — Prefer simple, direct code over clever abstractions. Three similar lines > premature helper.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

158. **implementation-minimal-docs** — Do not add docstrings, comments, or type annotations beyond what's needed for clarity.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

159. **implementation-minimal-deps** — Keep dependencies minimal. Do not add packages without human approval.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

160. **review-format-structured** — Produce a structured review: Summary (one paragraph), Issues (severity + description), Suggestions (non-blocking), Checklist.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

161. **test-framework-conventions-binding** — Elixir: ExUnit + `tmp_dir`; Python: pytest + `tmp_path`; JavaScript: node:test or vitest.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

162. **test-names-read-as-specs** — Test names should read as specifications, not describe implementation.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

163. **never-run-full-umbrella-mix-test** — Never run full umbrella `mix test`. Scope to single app or specific file path. Umbrella has pre-existing integration-test pathology.
     - Bindingness: MUST | Audience: AI agent | Category: general-engineering

164. **never-run-tests-background** — Never use `run_in_background: true` for tests. Run in foreground with explicit timeout matching suite's expected wall time.
     - Bindingness: MUST | Audience: AI agent | Category: general-engineering

165. **beware-cross-suite-test-isolation-flakes** — Some tests pass in isolation but fail when run alongside other apps' suites. Prefer per-app suites run separately.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

166. **use-monitor-for-polling-not-sleep** — If you must poll, use `Monitor` with specific grep filter, not `sleep` / `run_in_background`.
     - Bindingness: SHOULD | Audience: AI agent | Category: general-engineering

167. **on-timeout-pull-partial-output** — Do not re-run same hanging command with longer timeout. Diagnose what's hanging and run fast subset first.
     - Bindingness: SHOULD | Audience: AI agent | Category: general-engineering

168. **subagent-heartbeat-mandatory** — Every subagent that runs TDD / tests / multi-phase implementation work must emit a heartbeat for parent monitoring.
     - Bindingness: MUST | Audience: AI agent | Category: agent-behavior

169. **subagent-parent-before-dispatch** — Parent session obligations: (1) create log dir, (2) choose log path, (3) start Monitor before spawn, (4) dispatch subagent, (5) stop Monitor on return.
     - Bindingness: MUST | Audience: AI agent | Category: agent-behavior

170. **subagent-brief-must-cite-log-path** — Brief must include exact log path subagent is expected to write to.
     - Bindingness: MUST | Audience: AI agent | Category: agent-behavior

171. **subagent-heartbeat-timestamps** — One ISO-8601-timestamped line per phase boundary: RED test written, RED confirmed failing, GREEN edit made, GREEN passing, suite run, review started, commit-approval reached.
     - Bindingness: MUST | Audience: AI agent | Category: agent-behavior

172. **subagent-final-marker-on-error** — Subagent must write final marker even on error, so agent that hits exception still produces last-line signal.
     - Bindingness: MUST | Audience: AI agent | Category: agent-behavior

173. **subagent-marker-format** — Marker format: `YYYY-MM-DDTHH:MM:SSZ <phase|note>: <short message>` (leading timestamp is grep anchor).
     - Bindingness: MUST | Audience: AI agent | Category: agent-behavior

174. **subagent-heartbeat-why-load-bearing** — Without heartbeats, human sees spinner and cancels working agent. With them, each marker becomes notification.
     - Bindingness: MUST | Audience: both | Category: agent-behavior

175. **subagent-sizing-rule** — Dispatch one subagent per bug / phase / focused change, not monolithic milestone-size agent. Smaller dispatches detect stuck one earlier.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

---

## .ai/paths.md (artifact layout reference)

176. **roadmap-default-path** — Default `ROADMAP.md`; Liminara override: `work/roadmap.md`.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

177. **epic-root-default-path** — Default `work/epics/`; Liminara: `work/epics/`.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

178. **epic-spec-default-filename** — Default `spec.md`; Liminara override: `epic.md`.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

179. **milestone-spec-path-template** — Default `work/epics/<epic>/<milestone-id>-<slug>.md`; Liminara: same.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

180. **milestone-tracking-doc-path-template** — Default `work/epics/<epic>/<milestone-id>-tracking.md`; Liminara: `work/milestones/tracking/<epic>/../<milestone-id>-tracking.md`.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

181. **completed-epic-path-template** — Default `work/epics/completed/<epic>/`; Liminara: `work/done/<epic>/`.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

182. **epic-id-pattern** — Default `E-{NN}`; Liminara: `E-{NN}[optional-letter]` (e.g., `E-14a`).
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

183. **milestone-id-pattern** — Default `m-<track>-{NN}`; Liminara: `M-<TRACK>-<NN>` (uppercase M and TRACK).
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

184. **gaps-path** — Default `work/gaps.md`; Liminara: same.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

185. **scratch-path** — Default `.ai-repo/scratch/`; Liminara: same.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

186. **scratch-audit-threshold** — Default 100 MB; Liminara: same.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

187. **decisions-path** — Default `work/decisions.md`; Liminara: same.
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

188. **agent-history-path** — Default `work/agent-history/`; Liminara: same.
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

189. **adr-path** — Default `docs/decisions/`; Liminara: same.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

190. **adr-template-path** — Default `.ai/templates/adr.md`; Liminara: same.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

191. **research-path** — Liminara override: `docs/research/`.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

192. **architecture-path** — Liminara override: `docs/architecture/`.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

193. **framework-skill-prefix** — Default `wf`; Liminara: same.
     - Bindingness: SHOULD | Audience: AI agent | Category: meta

194. **repo-skill-prefix** — Default (empty); Liminara: same.
     - Bindingness: SHOULD | Audience: AI agent | Category: meta

---

## .ai-repo/config/artifact-layout.json

195. **liminara-roadmap-path-override** — `work/roadmap.md`.
     - Bindingness: MUST | Audience: AI agent | Category: project-specific

196. **liminara-epic-spec-filename-override** — `epic.md` (not `spec.md`).
     - Bindingness: MUST | Audience: AI agent | Category: project-specific

197. **liminara-tracking-doc-path-override** — `work/milestones/tracking/<epic>/../<milestone-id>-tracking.md`.
     - Bindingness: MUST | Audience: AI agent | Category: project-specific

198. **liminara-completed-epic-path-override** — `work/done/<epic>/`.
     - Bindingness: MUST | Audience: AI agent | Category: project-specific

199. **liminara-epic-id-pattern-override** — `E-{NN}[optional-letter]`.
     - Bindingness: MUST | Audience: AI agent | Category: project-specific

200. **liminara-milestone-id-pattern-override** — `M-<TRACK>-<NN>` (uppercase).
     - Bindingness: MUST | Audience: AI agent | Category: project-specific

---

## .ai-repo/config/commit.json

201. **commit-coauthor-claude-mapping** — `Claude <noreply@anthropic.com>`.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

202. **commit-coauthor-copilot-mapping** — `GitHub Copilot <noreply@github.com>`.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

203. **commit-coauthor-codex-mapping** — `OpenAI Codex <noreply@openai.com>`.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

---

## .ai/agents/builder.md

204. **builder-responsibilities** — Implement milestone ACs; write tests first (TDD); create tracking docs; update project README and inline docs; manage branch work.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

205. **builder-skills-patch-startmilestone-tddcycle** — Use `patch` (one-off fixes), `start-milestone` (with spec), `tdd-cycle` (red-green-refactor).
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

206. **builder-inputs-needed** — Milestone spec; codebase context; previous artifacts.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

207. **builder-outputs-produce** — Application code + tests (all passing); tracking doc; updated README/docs; staged changes only (never committed without approval).
     - Bindingness: MUST | Audience: AI agent | Category: agent-behavior

208. **builder-self-review-before-handoff** — Before declaring ready, run: re-read spec, confirm every AC has test, run branch-coverage audit, mentally walk review checklist, boot app if skill exists, fix findings.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

209. **builder-branch-coverage-hard-rule-enforcement** — Every reachable conditional branch must be exercised by explicit test before milestone declared done. Audit runs before commit-approval prompt, not after human asks.
     - Bindingness: MUST | Audience: AI agent | Category: general-engineering

210. **builder-tests-deterministic** — Tests must be deterministic (no network, no clock).
     - Bindingness: MUST | Audience: both | Category: general-engineering

211. **builder-build-must-be-green** — Build must be green before declaring done.
     - Bindingness: MUST | Audience: both | Category: general-engineering

212. **builder-follow-conventions** — Follow existing code conventions. Prefer minimal changes — don't refactor unrelated code.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

213. **builder-explore-quick-by-default** — Codebase exploration: `Explore` at `quick` by default; escalate only after `quick` leaves real gap.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

214. **builder-general-purpose-for-webfetch** — When `Explore` can't help (needs WebFetch/WebSearch), use `general-purpose` with `model: "sonnet"`.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

---

## .ai/agents/reviewer.md

215. **reviewer-responsibilities** — Code review (correctness, regressions, edge cases, conventions); milestone completion validation; test coverage assessment; wrap-up documentation.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

216. **reviewer-skills-review-code-wrap-milestone-wrap-epic-audit-lint** — Use `review-code`, `wrap-milestone`, `wrap-epic`, `workflow-audit`, `doc-lint`, `doc-garden`, `quality-score`.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

217. **reviewer-inputs-needed** — Changed files (diff or staged); milestone spec; tracking doc; test results.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

218. **reviewer-outputs-produce** — Review comments; milestone summary to `work/releases/<milestone-id>-release.md`; updated tracking doc.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

219. **reviewer-handoff** — After wrap: "Milestone complete. Summary written. Ready to merge to main."
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

220. **reviewer-never-commit-push-without-approval** — NEVER run `git commit` or `git push` without explicit human approval. "Continue", "ok", "next step" do NOT count.
     - Bindingness: MUST | Audience: AI agent | Category: general-engineering

221. **reviewer-be-specific-in-feedback** — Reference files and lines.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

222. **reviewer-distinguish-blocking-vs-suggestions** — Distinguish blocking issues from suggestions.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

223. **reviewer-verify-all-acceptance-criteria** — Verify all acceptance criteria, not just "it looks good".
     - Bindingness: MUST | Audience: both | Category: general-engineering

224. **reviewer-explore-quick-by-default** — For codebase exploration during review, spawn `Explore` at `quick` thoroughness by default. Escalate only when `quick` leaves real gap.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

225. **reviewer-general-purpose-for-webfetch** — For research / lookup needing WebFetch or WebSearch, use `general-purpose` with `model: "sonnet"`.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

226. **reviewer-keep-on-parent-model** — Keep `reviewer` on parent (Opus) model when invoked as subagent — correctness judgments carry irreducible load.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

227. **reviewer-checklist** — All ACs met; tests cover ACs; tests deterministic; build passes; no unrelated changes; naming follows conventions; error handling adequate; no secrets/PII; README/docs updated if public API changed.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

---

## .ai/agents/planner.md

228. **planner-focus** — Epic/milestone planning, spec drafting, brainstorming, architecture, and research.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

229. **planner-key-skills** — `plan-epic`, `plan-milestones`, `draft-spec`, `architect`.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

230. **planner-responsibilities** — Plan features and initiatives (epics, milestones); draft specs; break down work; facilitate brainstorming/architecture; document research/decisions.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

231. **planner-workflow-new-feature** — When new feature requested: use `plan-epic` and `plan-milestones`.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

232. **planner-workflow-detailed-specs** — For detailed milestone specs, use `draft-spec`.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

233. **planner-workflow-brainstorm-architecture** — For brainstorming, architecture, or research, automatically invoke `architect` skill.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

234. **planner-file-placement-from-layout** — File placement resolves from `researchPath` (exploration) or `architecturePath` (decided design intent).
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

235. **planner-implementation-is-builder-job** — Implementation is the builder's job. Hand off once specs are approved. Don't run `start-milestone` or `tdd-cycle`.
     - Bindingness: MUST | Audience: AI agent | Category: agent-behavior

236. **planner-explore-quick-by-default** — For codebase exploration, spawn `Explore` at `quick` thoroughness by default. Escalate only when `quick` leaves real gap.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

237. **planner-general-purpose-for-research** — For research / lookup / synthesis needing WebFetch or WebSearch, use `general-purpose` with `model: "sonnet"`.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

238. **planner-keep-on-parent-model** — Keep `Plan` and `planner` on parent (Opus) model — architectural reasoning carries irreducible load.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

---

## .ai/agents/deployer.md

239. **deployer-responsibilities** — Infrastructure configuration and deployment; CI/CD setup and troubleshooting; release tagging and changelog management; health checks and rollback; container builds.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

240. **deployer-skill-release** — Use `release` skill — tag, changelog, and publish.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

241. **deployer-inputs-needed** — Merged milestone or epic (on main); infrastructure config; pipeline definitions; previous release version.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

242. **deployer-outputs-produce** — Git tags (semantic versioning); updated `CHANGELOG.md`; deployment artifacts; health check verification.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

243. **deployer-handoff** — After release: "Release v{X.Y.Z} tagged and deployed. Health checks passing."
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

244. **deployer-never-commit-push-tag-deploy-without-approval** — NEVER run `git commit`, `git push`, `git tag`, or deploy without explicit human approval.
     - Bindingness: MUST | Audience: AI agent | Category: general-engineering

245. **deployer-stage-show-stop-wait** — Stage changes, show summary, then STOP and wait.
     - Bindingness: MUST | Audience: AI agent | Category: general-engineering

246. **deployer-never-deploy-without-green-tests** — Never deploy without green tests on main.
     - Bindingness: MUST | Audience: both | Category: general-engineering

247. **deployer-semantic-versioning** — Follow semantic versioning.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

248. **deployer-document-rollback-steps** — Document rollback steps for infrastructure changes.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

249. **deployer-verify-health-checks** — Verify health checks after deployment.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

---

## .ai/skills/tdd-cycle.md

250. **tdd-when-to-use** — During milestone implementation, for each acceptance criterion or feature unit.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

251. **tdd-red-phase-failing-test** — Write test(s) that describe expected behavior; test names follow convention; use project's test framework; run → confirm FAIL for right reason.
     - Bindingness: MUST | Audience: both | Category: general-engineering

252. **tdd-green-phase-minimum-code** — Write minimum code to pass test; don't add features test doesn't require; run → confirm PASS; check no other tests broke.
     - Bindingness: MUST | Audience: both | Category: general-engineering

253. **tdd-refactor-phase-cleanup** — Remove duplication; improve naming; extract methods/classes if needed; run → confirm still GREEN.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

254. **tdd-update-tracking** — Check off acceptance criterion in tracking doc; note any decisions or deviations.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

255. **tdd-antipatterns** — Code before tests, tests that can't fail, skipping refactor, testing implementation details instead of behavior, test execution-order dependencies.
     - Bindingness: MUST-NOT | Audience: both | Category: general-engineering

256. **tdd-test-quality-checks** — Deterministic (no randomness/clock/network); independent (no shared mutable state); cover edge cases; names explain what's tested.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

257. **tdd-branch-coverage-audit-mandatory** — Every TDD cycle ends with audit confirming every reachable branch has at least one test. Hard rule.
     - Bindingness: MUST | Audience: both | Category: general-engineering

258. **tdd-branch-coverage-walk-every-branch** — Open each new/changed source file; for every if/else/switch/catch/?:/early-return, identify which test exercises each side.
     - Bindingness: MUST | Audience: both | Category: general-engineering

259. **tdd-branch-coverage-defensive-paths-count** — Defensive paths (guards, exception catches, malformed-input handlers) count as reachable — if it ships, it gets a test.
     - Bindingness: MUST | Audience: both | Category: general-engineering

260. **tdd-branch-coverage-private-helpers** — If helper is private and branch is hard to reach via public API, expose using language's friend-assembly / package-private mechanism.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

261. **tdd-branch-coverage-genuinely-unreachable** — Genuinely unreachable branches must be documented in milestone spec under "Coverage notes" with the reason.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

262. **tdd-branch-coverage-audit-before-commit-prompt** — Do not declare "every branch covered" without audit. Audit happens before commit-approval prompt, not after human asks.
     - Bindingness: MUST | Audience: both | Category: general-engineering

---

## .ai/skills/design-contract.md (sample)

263. **design-contract-when-to-use** — When authoring an ADR that specifies a contract whose validation is mechanical (CUE, JSON Schema, Protobuf, OpenAPI, Pydantic, Avro, …).
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

264. **design-contract-bundle-is-contract** — Bundle is the contract: ADR + schema + fixtures + worked example + reference implementation, reviewed together as one PR.
     - Bindingness: MUST | Audience: both | Category: project-specific

265. **design-contract-invalid-fixtures-not-optional** — Invalid fixtures are not optional. Schema's permissiveness goes untested without them.
     - Bindingness: MUST | Audience: both | Category: project-specific

266. **design-contract-reference-impl-tbd-is-wish** — TBD reference implementation is a wish, not a contract. Must be existing (file:line) or scheduled (named milestone).
     - Bindingness: MUST | Audience: both | Category: project-specific

267. **design-contract-schema-evolution-first-class** — Schema evolution is a first-class concern. Validate every committed historical fixture against HEAD schema.
     - Bindingness: MUST | Audience: both | Category: project-specific

268. **design-contract-tech-neutral-by-design** — Workflow shape and discipline live here; concrete commands/extensions/validator invocations in per-language recipes.
     - Bindingness: SHOULD | Audience: both | Category: meta

269. **design-contract-adr-path** — ADR at repo's resolved `adrPath` (default `docs/decisions/`). Use standard template `.ai/templates/adr.md`.
     - Bindingness: SHOULD | Audience: AI agent | Category: general-engineering

270. **design-contract-schema-path** — Schema location per recipe: `docs/architecture/contracts/schemas/<topic>.<ext>`. Schema is authoritative.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

271. **design-contract-fixtures-path** — Valid + invalid fixtures at `docs/architecture/contracts/fixtures/<topic>/<version>/{valid,invalid}/<name>.<ext>`.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

272. **design-contract-worked-example** — One realistic, end-to-end example with concrete domain values. Inline in ADR or separate file.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

273. **design-contract-reference-impl-existing-or-scheduled** — Cite existing (file:line) OR scheduled-to-exist (named milestone). TBD not acceptable.
     - Bindingness: MUST | Audience: both | Category: project-specific

274. **design-contract-schema-evolution-loop** — Validate every committed historical fixture against current HEAD schema. If fixture no longer validates, you've broken consumers.
     - Bindingness: MUST | Audience: both | Category: project-specific

275. **design-contract-pr-whole-bundle** — Single PR contains: ADR, schema, valid fixture(s), invalid fixture(s), worked example. Reference impl either in PR or named in ADR.
     - Bindingness: MUST | Audience: both | Category: project-specific

276. **design-contract-bundle-complete-check** — All five pieces present (or reference impl named with deadline). No TBD for any field.
     - Bindingness: MUST | Audience: both | Category: project-specific

277. **design-contract-invalid-fixture-real-failure-mode** — Invalid fixture exercises real failure mode, not just syntactic error.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

278. **design-contract-worked-example-concrete-values** — No placeholders, no lorem ipsum. Real names, real numbers, real dates.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

279. **design-contract-schema-version-required** — Even for v1.0.0, set `contract.schema_version` frontmatter field.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

280. **design-contract-repo-local-bindings** — Project-specific conventions go in `.ai-repo/skills/design-contract.md`.
     - Bindingness: SHOULD | Audience: both | Category: meta

281. **design-contract-continuous-enforcement** — `verify-contracts` skill + Go binary re-run validator + schema-evolution loop on every PR.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

282. **design-contract-wrap-time-gating** — `wrap-milestone` / `wrap-epic` re-run verification and check scheduled-to-exist reference implementations materialized.
     - Bindingness: MUST | Audience: both | Category: project-specific

283. **design-contract-audit-time-observation** — `workflow-audit` surfaces drift: stale reference-impl citations, mismatched schema versions, incomplete bundles.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

---

## .ai/skills/wrap-epic.md (sample: 150 lines of 280+)

284. **wrap-epic-principles** — Wrap is closure, not release. Branch cleanup on origin only. Archival preserves spec. Commit-approval explicit.
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

285. **wrap-epic-preserve-local-branches** — Merged milestone + epic branches deleted on origin; **local branches preserved** for graph tools.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

286. **wrap-epic-archival-preserves-spec** — Moving `work/epics/<id>/` to completed-epics keeps specs, tracking, `wrap.md` readable forever. Nothing deleted.
     - Bindingness: MUST | Audience: both | Category: project-specific

287. **wrap-epic-commit-approval-explicit** — Commit-approval is explicit per `rules.md`.
     - Bindingness: MUST | Audience: both | Category: general-engineering

288. **wrap-epic-preconditions** — (1) All milestones complete (run `workflow-audit`), (2) Epic branch up to date, (3) Working tree clean, (4) Integration target identified.
     - Bindingness: MUST | Audience: both | Category: workflow-pm

289. **wrap-epic-step-tmp-hygiene-probe** — Run `bash .ai/tools/tmp-probe.sh --days 60`. Triage listed files (keep/delete/ignore). Proceed only after all triaged.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

290. **wrap-epic-step-close-spec-scaffold-wrap** — Promote epic status (`wf-graph promote`); add `completed: YYYY-MM-DD` to epic spec; create `wrap.md` with core fields.
     - Bindingness: MUST | Audience: both | Category: workflow-pm

291. **wrap-epic-step-adr-check** — Walk epic's commits. For each candidate decision: "Would a future reader regret missing the reasoning?" If warranted, scaffold ADR.
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

292. **wrap-epic-step-contract-verify-gate** — *No-op if contracts.json absent.* Two checks: (2.25.a) bundle verification, (2.25.b) reference-impl reality check.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

293. **wrap-epic-step-contract-verify-fixtures** — Detect if epic touched contract paths. If touched, run `verify-contracts` and gate on verification findings.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

294. **wrap-epic-step-contract-verify-finding-triage** — Classify by type (fixture-rejected, fixture-accepted, evolution-regression, etc.) and offer triage: fix-now / gap / dismiss.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

295. **wrap-epic-step-reference-impl-safety-net** — For each ADR citing milestone of this epic, verify `contract.reference_implementation` resolves to real `file:line`.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

296. **wrap-epic-step-doc-lint-scoped** — Invoke `doc-lint` in scoped mode with epic's change-set. Gate on contract-drift, removed-feature-docs, uncovered-contract-surface findings.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

297. **wrap-epic-reference-phrasing-in-wrap-md** — When populating `wrap.md` sections, use reference-phrasing ("every ADR listed in *ADRs ratified*") not reproduced counts.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

298. **wrap-epic-reference-phrasing-why-load-bearing** — `workflow-audit` numeric-claim check flags drift between scalars and lists. Reference-phrasing is drift-proof.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

---

## .ai/skills/plan-milestones.md

299. **plan-milestones-when-to-use** — Epic spec exists. User says: "Break this into milestones", "Plan the work for epic X".
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

300. **plan-milestones-read-epic-spec** — Read at path resolved from artifact layout: `<epicRootPath>/<epic-slug>/<epicSpecFileName>`.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

301. **plan-milestones-independently-shippable** — Each milestone is independently shippable.
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

302. **plan-milestones-clear-criteria** — Each milestone has clear, testable acceptance criteria.
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

303. **plan-milestones-dependency-flow** — Dependencies flow forward (M1 before M2).
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

304. **plan-milestones-target-1-3-days** — Target 1-3 days of work per milestone.
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

305. **plan-milestones-sequence-by-dependency** — Order by dependency (foundational first).
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

306. **plan-milestones-group-related-work** — Group related work (don't scatter concerns).
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

307. **plan-milestones-identify-parallel** — Identify any milestones that can be parallelized.
     - Bindingness: SHOULD | Audience: both | Category: workflow-pm

308. **plan-milestones-naming-from-pattern** — Follow repo's `milestoneIdPattern` (from `.ai-repo/config/artifact-layout.json`, fall back to `.ai/paths.md`).
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

309. **plan-milestones-filename-from-template** — Spec filenames follow `milestoneSpecPathTemplate`.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

310. **plan-milestones-reference-phrasing-in-spec** — When writing success criteria/risks that reference milestone list, use reference-phrasing ("every milestone listed below") not hand-written count.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

311. **plan-milestones-register-in-graph** — For each milestone row, run `wf-graph add-milestone <milestone-id> --epic <epic-id> --title "<title>" --pretty`.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

312. **plan-milestones-graph-skip-if-absent** — Skip `wf-graph` if consumer repo hasn't bootstrapped it (`work/graph.yaml` missing). Bullet in spec body is enough.
     - Bindingness: MAY | Audience: AI agent | Category: project-specific

313. **plan-milestones-output** — Milestone plan (table in epic spec or separate doc); milestone IDs ready for spec drafting.
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

314. **plan-milestones-next-step-draft-spec** — Next: use `draft-spec` for each milestone (start with M1).
     - Bindingness: SHOULD | Audience: AI agent | Category: agent-behavior

---

## .ai-repo/skills/design-contract.md (overlay)

315. **liminara-contract-design-overlay** — Liminara-specific bindings on top of upstream `design-contract` skill. Upstream carries tech-neutral 7-step workflow; overlay adds Liminara paths + reviewer expectations.
     - Bindingness: SHOULD | Audience: both | Category: meta

316. **liminara-contract-read-order** — (1) `.ai/skills/design-contract.md` — workflow shape, (2) `.ai/docs/recipes/design-contract-cue.md` — CUE commands, (3) this overlay, (4) `.ai-repo/rules/contract-design.md` — reviewer rule.
     - Bindingness: SHOULD | Audience: both | Category: meta

317. **liminara-schemas-path-override** — Upstream defaults `docs/architecture/contracts/`. Liminara: `docs/schemas/<topic>/schema.cue`.
     - Bindingness: MUST | Audience: AI agent | Category: project-specific

318. **liminara-valid-fixtures-path-override** — `docs/schemas/<topic>/fixtures/v<N>/valid/<name>.yaml`.
     - Bindingness: MUST | Audience: AI agent | Category: project-specific

319. **liminara-invalid-fixtures-path-override** — `docs/schemas/<topic>/fixtures/v<N>/invalid/<name>.yaml`.
     - Bindingness: MUST | Audience: AI agent | Category: project-specific

320. **liminara-worked-example-path** — Inline in ADR body OR `docs/architecture/contracts-examples/<topic>-walkthrough.md` if large.
     - Bindingness: SHOULD | Audience: AI agent | Category: project-specific

321. **liminara-adr-target-directory** — `docs/decisions/`. Filename: `NNNN-<slug>.md` (no `ADR-` prefix on disk; `id: ADR-NNNN` in frontmatter).
     - Bindingness: MUST | Audience: AI agent | Category: project-specific

322. **liminara-cue-vet-validation** — Use `scripts/cue-vet path/to/file.cue` (single file) or `scripts/cue-vet` (walk whole library). valid/ must pass, invalid/ must fail.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

323. **liminara-cue-hook-install** — Pre-commit enforcement: `scripts/install-cue-hook`. Hook runs `cue vet` on staged .cue files + schema-evolution loop on fixtures.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

324. **liminara-schemas-readme** — `docs/schemas/README.md` placeholder documents layout for contributors.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

325. **liminara-contract-reviewer-discipline-1** — Anchored admin-pack citations on every pack-level ADR.
     - Bindingness: MUST | Audience: both | Category: project-specific

326. **liminara-contract-reviewer-discipline-2** — Contract-matrix rows verified at wrap.
     - Bindingness: MUST | Audience: both | Category: project-specific

327. **liminara-contract-reviewer-discipline-3** — Radar-primary / admin-pack-secondary structure.
     - Bindingness: MUST | Audience: both | Category: project-specific

328. **liminara-contract-reviewer-discipline-4** — Reference-implementation citation shapes.
     - Bindingness: MUST | Audience: both | Category: project-specific

---

## .ai-repo/skills/app-legibility.md

329. **app-legibility-quick-health-check** — Phoenix LiveView on 4005; A2UI WebSocket on 4006. `curl` tests for reachability.
     - Bindingness: MAY | Audience: both | Category: project-specific

330. **app-legibility-boot-phoenix** — Run from `runtime/`: `iex --sname liminara -S mix phx.server`. Port 4005. Ready signal in stdout.
     - Bindingness: SHOULD | Audience: both | Category: project-specific

331. **app-legibility-boot-a2ui** — Same process as Phoenix. Port 4006. Logs "starting on port 4006".
     - Bindingness: SHOULD | Audience: both | Category: project-specific

332. **app-legibility-first-time-deps** — `cd runtime && mix deps.get && mix compile`. After deps change: same.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

333. **app-legibility-compile-focused-tests** — `cd runtime && mix compile --warnings-as-errors && mix test --only <tag>` or specific test files.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

334. **app-legibility-boot-app-verify** — Boot; trigger affected surface; check outputs under `runtime/data/`.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

335. **app-legibility-full-validation-pipeline** — Elixir: `mix format && mix credo && mix dialyzer && mix test`. Python: `ruff check/format/ty/pytest`. dag-map: `npm test`.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

336. **app-legibility-shutdown-cleanly** — From IEx: `:init.stop()`. Or: `epmd -names && pkill -f 'sname liminara'`.
     - Bindingness: SHOULD | Audience: both | Category: general-engineering

337. **app-legibility-port-safety** — Never `pkill -f beam.smp` — kills all Erlang VMs. Prefer targeted shutdown.
     - Bindingness: MUST-NOT | Audience: both | Category: general-engineering

338. **app-legibility-radar-end-to-end** — `iex` session: `Liminara.Radar.Pack.discover(topic: :default)`. Watch at `http://localhost:4005/runs`.
     - Bindingness: MAY | Audience: both | Category: project-specific

---

## Summary Statistics

**Total policy candidates extracted: 338**

### By source file:
- CLAUDE.md: 44 candidates
- `.ai/rules.md`: 99 candidates
- `.ai-repo/rules/contract-design.md`: 20 candidates
- `.ai-repo/rules/liminara.md`: 54 candidates
- `.ai-repo/rules/tdd-conventions.md`: 24 candidates
- `.ai/paths.md`: 18 candidates
- `.ai-repo/config/*.json`: 9 candidates
- `.ai/agents/builder.md`: 11 candidates
- `.ai/agents/reviewer.md`: 13 candidates
- `.ai/agents/planner.md`: 11 candidates
- `.ai/agents/deployer.md`: 11 candidates
- `.ai/skills/tdd-cycle.md`: 13 candidates
- `.ai/skills/design-contract.md`: 21 candidates
- `.ai/skills/wrap-epic.md`: 15 candidates
- `.ai/skills/plan-milestones.md`: 16 candidates
- `.ai-repo/skills/design-contract.md`: 14 candidates
- `.ai-repo/skills/app-legibility.md`: 10 candidates

### By bindingness:
- MUST: 77 candidates
- SHOULD: 225 candidates
- MAY: 30 candidates
- MUST-NOT: 6 candidates

### By category:
- **general-engineering** (tests, formatting, naming, commits, validation): 95 candidates
- **project-specific** (Liminara domains, paths, structures): 108 candidates
- **workflow-pm** (planning, milestones, ADRs, roadmap, decisions): 48 candidates
- **agent-behavior** (how AI agents operate, routing, Q&A, heartbeats): 70 candidates
- **meta** (rules about rules, framework configuration): 17 candidates

### By audience:
- **Both (human + AI agent)**: 217 candidates
- **AI agent only**: 109 candidates
- **Human only**: 12 candidates

