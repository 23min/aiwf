# Workflow / PM-domain policies — belong in the aiwf orbit

> Rules about *how the work itself is done*: how branches are named, when commits
> happen, which agent does what, when a milestone is "done," how decisions are
> recorded, how dead-code audits are scheduled, how provenance is recorded on
> every commit. These are the items that already-or-naturally belong in the aiwf
> kernel rather than the consumer repo.

Many of these are already encoded in aiwf v3 itself; FlowTime's CLAUDE.md repeats
or specializes them. The interesting question is which of these *the framework*
should own (so they stop being repeated per consumer repo) and which the consumer
should specialize.

---

## A. The "hard rules" — the most-explicitly-binding workflow policies in the corpus

### W-1. NEVER commit or push without explicit human approval — "continue" / "ok" do not count
- **Source:** `CLAUDE.md` ("Hard Rules").
- **Rung:** 1 (prose; relies on the LLM to honor it).
- **Notes:** **The single most-stated rule in the corpus**, and currently rung 1. A pre-commit hook that requires a token from a human-readable confirmation file would move this to rung 2 — but the rule is also human-trust-shaped (the LLM should obey), and the right home is closer to model alignment than CI.

### W-2. TDD by default for logic/API/data code; red → green → refactor
- **Source:** `CLAUDE.md`.
- **Rung:** 1 (prose).

### W-3. TDD phase tracking — logic-bearing ACs in `draft` and `in_progress` milestones carry a `tdd_phase: red|green|refactor` field
- **Source:** `CLAUDE.md`.
- **Rung:** 2 (aiwf field validated; advanced via `aiwf promote M-NNN/AC-N --phase <p>`).
- **Notes:** **Rare and interesting:** turns a workflow norm (TDD) into a *trackable per-AC state*. Effectively a small FSM per AC. The framework could absorb this as a first-class kernel feature.

### W-4. Non-logic ACs (doc-only, gap-closure, full-suite gates, branch-coverage audit, process discipline) omit the phase field
- **Source:** `CLAUDE.md`.
- **Rung:** 1.
- **Notes:** Implicit AC-typing. Worth lifting: ACs have *types* (`logic`, `doc`, `gap-closure`, `audit`, `process`) and the type drives which fields apply. Currently free-form prose.

### W-5. Every red-tagged AC must reach green before the milestone wraps
- **Source:** `CLAUDE.md`.
- **Rung:** 2 (the wrap ritual checks; `aiwf check` could enforce).

### W-6. Branch coverage required before declaring done (line-by-line audit before commit-approval prompt)
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### W-7. Branch discipline — do NOT commit milestone work directly to `main`
- **Source:** `CLAUDE.md`.
- **Rung:** 1 (prose) + 2 (could be a pre-commit hook on milestone branches; not committed today).

### W-8. Conventional Commits format (`feat(...)`, `fix(...)`, `chore:`, `docs:`, `test:`, `refactor:`); no icons/emoji; subject + short bullet body capturing the milestone and key work/tests touched
- **Source:** `CLAUDE.md`.
- **Rung:** 1 (prose); could trivially be 2 with commitlint.
- **Notes:** Specifies *content* expectations beyond format ("capturing the milestone and key work/tests touched"). The content part is rung 1 forever; the format part can move to rung 2.

---

## B. Truth precedence (a meta-policy about how workflow policies relate to each other)

The Truth Discipline section in CLAUDE.md is the most policy-shaped block in the
entire corpus. It defines:

### W-9. Truth precedence (highest to lowest): code+passing tests > decisions/ADRs > epic specs > arch docs > history/exploration
- **Source:** `CLAUDE.md` ("Precedence").
- **Rung:** 1 (prose).
- **Notes:** **This is a policy about how to resolve policy conflicts.** Exactly Cedar's "policy precedence" mechanism, translated to a doc/code substrate. The framework could absorb this as a kernel-level conflict-resolution rule (with the truth classes parameterized per consumer).

### W-10. If code, decisions, and an architecture doc disagree, do not choose arbitrarily — report the mismatch and ask
- **Source:** `CLAUDE.md`.
- **Rung:** 1.
- **Notes:** "Surface the conflict, don't auto-resolve" — exactly the design-space §13 "auto-do something a human should decide" rule, applied to docs.

### W-11. Truth classes: `docs/` (current ground truth, code-provable) | `work/epics/` (decided-next, exploration) | `docs/archive/`, `docs/releases/` (historical) | `docs/notes/` (exploration only, never authority)
- **Source:** `CLAUDE.md`.
- **Rung:** 1 (the convention) + 2 (the gitignore hides `docs/notes/` — well, `docs/notes/` is gitignored at the framework level; FlowTime does not gitignore it, but the policy applies via the truth-class definition).

---

## C. Agent routing (Claude-code-specific workflow)

### W-12. Role agents (builder / planner / reviewer / deployer) ship via the `aiwf-extensions` plugin and drive named skills based on intent
- **Source:** `CLAUDE.md` ("Agent Routing" table).
- **Rung:** 1 (prose for the routing); 2 (the agent definitions themselves are committed).
- **Notes:** This is *aiwf-rituals* territory. Worth a dedicated skill bundle in the framework's policy library if the rituals plugin shape goes mainstream.

### W-13. After a milestone wrap, builder/reviewer should also invoke the repo-private `dead-code-audit` skill (the upstream `aiwfx-wrap-milestone` skill does not chain it)
- **Source:** `CLAUDE.md`.
- **Rung:** 1.
- **Notes:** A small but illustrative *policy gap* — the upstream skill doesn't chain a downstream one; the consumer repo has to remind the agent. The framework could allow consumer-repo skill bundles to *augment* the upstream skill chain.

### W-14. `aiwfx-wrap-epic` is shipped by the plugin but unclaimed by any agent's skill list; reviewer drives it in practice
- **Source:** `CLAUDE.md`.
- **Rung:** 0 (drift).
- **Notes:** Documents an *unclaimed* skill — a small policy hole. Worth lifting: skills should declare their primary owner.

---

## D. Provenance / audit (the principal × agent × scope model)

### W-15. Human verbs need no extra flags
- **Source:** `CLAUDE.md` ("Provenance").
- **Rung:** 2 (aiwf kernel).

### W-16. Non-human actors (`ai/...`, `bot/...`) must pass `--principal human/<id>` and operate inside an active `aiwf authorize <id> --to <agent>` scope
- **Source:** `CLAUDE.md`.
- **Rung:** 2 (aiwf kernel; trailers added automatically).

### W-17. The kernel adds `aiwf-principal:`, `aiwf-on-behalf-of:`, and `aiwf-authorized-by:` trailers automatically
- **Source:** `CLAUDE.md`.
- **Rung:** 2 (aiwf kernel).

### W-18. `aiwf authorize` is human-only
- **Source:** `CLAUDE.md`.
- **Rung:** 2 (aiwf kernel rejects agent invocation).

### W-19. Use `aiwf promote/cancel <id> --audit-only --reason "..."` to backfill an audit trail for a state already reached via a manual commit
- **Source:** `CLAUDE.md`.
- **Rung:** 2 (the `--audit-only` mode is the reified "after-the-fact ratification" verb).
- **Notes:** Rare verb worth lifting. **Most policy frameworks lack a "ratify after the fact" affordance.**

---

## E. Entity model and lifecycle

### W-20. Six entity kinds (epic, milestone, ADR, gap, decision, contract); no `task` or `story` entity
- **Source:** `CLAUDE.md`.
- **Rung:** 2 (aiwf kernel).
- **Notes:** Already a stored memory in the framework's auto-memory; consistent with the framework's deliberate non-goal.

### W-21. Don't edit entity frontmatter status by hand — use `aiwf promote` so the FSM check + commit trailer happen
- **Source:** `CLAUDE.md`.
- **Rung:** 2 (aiwf check would catch frontmatter-only state changes).

### W-22. Milestone work is tracked **inside the milestone spec itself** — single home for goal, ACs, design notes, working analysis, and work log
- **Source:** `CLAUDE.md`.
- **Rung:** 1 (the convention).
- **Notes:** Implicit policy: "no separate `*-tracking.md` files." The pre-aiwf v1 separate-tracking-doc convention is now an anti-pattern (G-035 explicitly tracks v1-residue contradictions).

### W-23. Older `*-log.md` / `*-tracking.md` files in `work/archived-epics/` are pre-aiwf v1 residue
- **Source:** `CLAUDE.md`.
- **Rung:** 1.
- **Notes:** Explicit deprecation of an old convention. Could move to rung 2 with a check that flags new files matching the old patterns.

### W-24. Epic dirs stay in place under `work/epics/E-NN-<slug>/` regardless of status — aiwf v3's truth surface is the frontmatter, not the path
- **Source:** `M-066` ("epic completion" AC).
- **Rung:** 2 (aiwf kernel).

---

## F. Dead-code audit (a workflow skill that *is* a small policy framework)

The `dead-code-audit` skill is, on close reading, a **policy framework in
miniature** — recipe-driven, polyglot, multi-rung, with a soft-signal contract.
Worth listing what it exemplifies as a policy-shaped workflow:

### W-25. Wrap-milestone invokes `dead-code-audit` as a non-blocking step
- **Source:** `dead-code-audit/SKILL.md` ("Integration with `wrap-milestone`").
- **Rung:** 2 (the skill is invoked from the wrap ritual).

### W-26. Bootstrap path (no recipes found) is conversation-shaped: detect stacks, propose tools, write recipes, exit; do not audit on the same invocation
- **Source:** `dead-code-audit/SKILL.md`.
- **Rung:** 1 (the discipline) + 2 (the skill enforces "exit before audit").
- **Notes:** **Excellent design pattern: a two-step skill where step 1 produces the configuration and step 2 uses it.** Reviewable, recoverable, no surprise audits with bad recipes.

### W-27. Soft-signal contract: never mutates code, never fails the build, always exits 0 from the audit path
- **Source:** `dead-code-audit/SKILL.md`.
- **Rung:** 1 + 2 (the exit code is the gate; the prose is the rationale).
- **Notes:** Worth lifting as a *policy class*: "soft-signal" policies surface findings without blocking. The framework's principle taxonomy could include this as one of the kinds.

### W-28. Recipe-driven, per-stack: `.claude/skills/dead-code-audit/recipes/dead-code-<stack>.md`; one recipe per stack; polyglot repos run sequentially
- **Source:** `dead-code-audit/SKILL.md`.
- **Rung:** 2.
- **Notes:** Recipes have required frontmatter (name, fileExts, tool, toolCmd) + optional (excludePaths) + free-text body for blind-spot hints. **Same shape as a policy-with-substrate-pointer.**

### W-29. Tool failure is a finding, not a wrap blocker — emit a per-stack section noting the failure with stderr captured
- **Source:** `dead-code-audit/SKILL.md`.
- **Rung:** 1 + 2 (the skill behavior).
- **Notes:** Same shape as design-space §5 "Don't pretend CUE is JSON Schema": fail loud, don't hide.

### W-30. Findings classified into four buckets: confirmed-dead-suspects | tool-flagged-but-live | intentional-public-surface | needs-judgement
- **Source:** `dead-code-audit/SKILL.md`.
- **Rung:** 1 (the classification convention).
- **Notes:** Each bucket has different downstream implications. Worth lifting: findings have a *type*, not just a severity.

### W-31. Blind-spot sweep: orphan fixtures, stale ADRs, helpers retained "for stability," deprecated aliases, schema fields with no consumers
- **Source:** `dead-code-audit/SKILL.md`.
- **Rung:** 1 (LLM judgment) + 2 (where structural).
- **Notes:** Five named *categories of structural finding the tool cannot see*. The recipe body extends this list per stack. **Designed-for-LLM-augmentation.**

### W-32. Hand-editing `work/dead-code-report.md` is an anti-pattern — it's overwritten on every audit run; findings worth keeping go to `work/gaps.md`
- **Source:** `dead-code-audit/SKILL.md` ("Anti-patterns").
- **Rung:** 1.
- **Notes:** Discipline rule about a generated artifact. Same shape as STATUS.md (regenerated by aiwf).

### W-33. Treating the report as a build gate is an anti-pattern — it's a soft signal, the whole point
- **Source:** `dead-code-audit/SKILL.md`.
- **Rung:** 1.

### W-34. Auditing on the same invocation that wrote the recipe is an anti-pattern — bootstrap exits before auditing on purpose; re-run
- **Source:** `dead-code-audit/SKILL.md`.
- **Rung:** 2 (the skill enforces).

---

## G. Wrap, ratification, and supersession

### W-35. On epic completion, frontmatter is promoted to `status: done` via `aiwf promote E-25 done`; `ROADMAP.md` is regenerated via `aiwf render roadmap --write`; a wrap artefact at `work/epics/<epic>/wrap.md` captures what shipped
- **Source:** `M-066` (E-25 epic).
- **Rung:** 2 (aiwf verb-gated).

### W-36. ADR ratification gates downstream milestones — implementation milestone cannot start before the ADR is `accepted`
- **Source:** `M-066` (the entire structure of E-25).
- **Rung:** 1 (the prose) + 2 (depends-on chain in milestone metadata).
- **Notes:** **This is exactly the supersession + lifecycle story the design-space doc anticipates.** ADR FSM (`proposed → accepted → superseded`) gates milestone FSM transitions.

### W-37. Documentation lifecycle: a sibling gap (G-035) tracks "pre-aiwf v1 framework docs survived migration and contradict the v3 model"
- **Source:** `M-066` references G-035.
- **Rung:** 1 (gap entity) + 2 (during M-066 doc sweep, conflicting docs are revised or marked).
- **Notes:** Cross-version doc cleanup is itself policy-shaped. Worth: a verb for "version-aware doc audit."

---

## H. Cross-cutting workflow rules

### W-38. No `task` or `story` entity — issue trackers do that better; framework's smallest unit is the milestone
- **Source:** aiwf overview (and CLAUDE.md by reference).
- **Rung:** 2 (aiwf kernel doesn't allow it).

### W-39. Engine API requires `WebApplicationFactory<Program>` for tests; prefer real dependencies over mocks
- **Source:** `CLAUDE.md`.
- **Rung:** 1.

### W-40. CI runs all 8 test projects in sequence with named jobs (Expressions, Core, Engine, Synthetic Adapters, Sim, UI, API, Integration, CLI)
- **Source:** `.github/workflows/build.yml`.
- **Rung:** 2.
- **Notes:** Implicit policy: "every project gets its own CI job" — unstated but adhered to. Could be its own meta-rule.

### W-41. `aiwf.yaml` pins consumer to a specific aiwf version (`aiwf_version: v0.1.1`)
- **Source:** `aiwf.yaml`.
- **Rung:** 2 (aiwf kernel reads at init/upgrade).
- **Notes:** Version-pinning policy at the consumer level. The framework's upgrade story owns this.

---

## Cross-cut observations on the workflow bucket

1. **The framework already enforces the strongest workflow policies** (entity FSMs, provenance trailers, ratification gates). The consumer repo (FlowTime) repeats them in CLAUDE.md as *reminders to the AI*, not as load-bearing enforcement.
2. **The consumer-specific workflow extensions are small and few**: TDD phase tracking (W-3), custom branch naming (P-39), dead-code-audit invocation (W-13). The rest is either generic (CLAUDE.md prose) or kernel-owned.
3. **The dead-code-audit skill is a working blueprint for "policy with substrate pointer."** It has a generic shell + per-substrate recipes + structured findings + soft-signal contract. The framework's policy primitive could explicitly model this shape.
4. **The biggest workflow-policy gap is `--audit-only` for non-aiwf domains.** The pattern of "ratify a state after the fact, with reason + commit trailer" exists for FSM transitions but not for, e.g., post-hoc decision-recording about a code area.
5. **Truth Precedence (W-9 / W-11) is the closest thing to a *governance* policy in the corpus** — it tells you how to resolve conflicts between policies that disagree. The design-space doc names this as the territory governance covers; FlowTime has it, in prose, as one paragraph in CLAUDE.md.
