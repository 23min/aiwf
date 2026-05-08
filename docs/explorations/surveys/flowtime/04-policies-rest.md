# Rest — boundary cases and items that don't bucket cleanly

> Things that are policy-shaped but don't sit cleanly in *general*, *project-
> specific*, or *workflow*. Some are meta-policies (rules about rules). Some are
> *anti-patterns* the corpus calls out explicitly. Some are policies-of-omission.
> Some are open-question shapes that are policies-in-waiting.

---

## A. Meta-policies (rules about how to write rules)

### R-1. The principles checklist (`architecture.md` §12) — every change walks 10 questions
- **Source:** Framework `CLAUDE.md` (governs framework-source changes; not a FlowTime artifact, but FlowTime's CLAUDE.md is itself a *consumer* of the same posture).
- **Rung:** 1.
- **Notes:** Pure meta-policy. "Apply this checklist to any candidate change." Aiwf-internal.

### R-2. Future references must cite an open issue (e.g., `… tracked in #NN`)
- **Source:** Framework `CLAUDE.md`.
- **Rung:** 1.
- **Notes:** Prose-discipline rule about how to talk about future capability without lying about the present. **Could move to rung 2 with an LLM-as-linter or grep on tense markers ("will," "future") near unscoped sentences.**

### R-3. "When the second case shows up; abstract on the third" (YAGNI heuristic)
- **Source:** Framework `CLAUDE.md`.
- **Rung:** 0 (judgment).
- **Notes:** Closest the framework gets to a *quantified* design rule. Provides a tiebreaker. Worth lifting as the shape of "rule with built-in revision trigger."

### R-4. Rules can be relaxed for specific cases — propose the rule change in the same PR or a prerequisite PR; don't silently violate
- **Source:** Framework `CLAUDE.md`.
- **Rung:** 1.
- **Notes:** **Meta-policy about how policies themselves are amended.** Maps directly to design-space §5 (lifecycle: contestation → supersession). Rare, important.

### R-5. Pre-PR audit: walk the diff against the rules in the governing CLAUDE.md and report rule conformance in the PR description
- **Source:** Framework `CLAUDE.md`.
- **Rung:** 1 (the discipline) + 2 (PR template prompts for it).
- **Notes:** **The "rung-1.5" pattern** — turning prose policy into a checklist that a human or LLM walks at PR time. The PR description becomes the audit log.

---

## B. Soft-signal vs blocking — a category split that earns its keep

### R-6. Soft-signal policies surface findings without blocking; blocking policies fail the build
- **Source:** Implicit across the corpus. Made explicit in `dead-code-audit/SKILL.md`.
- **Rung:** 1 (the convention).
- **Notes:** **A category of policy worth lifting into the framework's vocabulary.** The design-space doc treats severity (advisory/warning/blocking) as one axis; this is the same axis named differently. Worth aligning on one name.

### R-7. The principle that *adding* a soft-signal policy is cheap, but *promoting* it to blocking requires evidence
- **Source:** G-033 ("Treat any new analyser-warning regression on a previously-clean template as a hard regression until the canary is strict enough to catch it automatically"); D-053 (Phase 2 baseline canary first, full golden-output canon deferred).
- **Rung:** 1.
- **Notes:** **Implicit policy about policy promotion.** Worth surfacing: bind a *promotion criterion* to a soft-signal policy when it's authored.

---

## C. Policies-of-omission (deliberate non-policies)

### R-8. No multi-host adapter generation in PoC — Claude Code only
- **Source:** Framework `overview.md` ("What's deliberately not here").
- **Rung:** 0 (a deliberate non-goal).
- **Notes:** *Negative* policy. Worth marking: a non-goal is a kind of policy with a different lifecycle.

### R-9. No `task` / `story` entity
- **Source:** Framework `overview.md`.
- **Rung:** 2 (kernel-enforced; cannot create).

### R-10. No FSM-as-YAML in PoC — six kinds and statuses hardcoded in Go
- **Source:** Framework `overview.md`.
- **Rung:** 0 (a deliberate non-goal).
- **Notes:** Policy-of-omission with a future-revision trigger ("when there's a second consumer who needs to customize"). Same shape as R-3.

### R-11. No GitHub Issues / Linear / Jira / Azure DevOps sync — out of scope for the PoC
- **Source:** Framework `overview.md`.
- **Rung:** 0.
- **Notes:** Future-direction non-goal.

---

## D. Open questions / policies-in-waiting

### R-12. Snake_case telemetry-manifest vs camelCase model schema — deliberate or drift?
- **Source:** dead-code report; FlowTime P-8.
- **Rung:** 0 (open question).
- **Notes:** No mechanism to ratify. **Exemplar of the framework gap.**

### R-13. When does the class-2 capacity-aware allocator ship?
- **Source:** M-066 AC-7 (deferred-follow-up gap).
- **Rung:** 0 (deferred).
- **Notes:** Deferral is itself policy-shaped (a non-promise with a re-evaluation trigger).

### R-14. Where does the policy-version pin live for in-flight work?
- **Source:** Framework substrates exploration §7 (open questions).
- **Rung:** N/A — design question.
- **Notes:** Maps to FlowTime's `aiwf.yaml` `aiwf_version: v0.1.1` pin. The same pattern at policy granularity is missing.

### R-15. Should the framework own the CUE evaluator, or shell out to `cue vet`?
- **Source:** Framework substrates exploration §7.
- **Rung:** N/A.
- **Notes:** Process-substrate boundary question.

### R-16. How does the framework handle a policy whose runner does not exist on this machine?
- **Source:** Framework substrates exploration §7.
- **Rung:** N/A.
- **Notes:** FlowTime answers this for `dead-code-audit` via "tool failure is a finding"; the framework should have a default.

---

## E. Items that straddle buckets

### R-17. The Truth Discipline section
- **Belongs to:** general (doc hygiene), workflow (truth precedence), and meta-policy (rules about rule conflicts) all at once.
- **Notes:** Shows that any clean three-bucket split will leak. The Truth Discipline section is the most obvious leak: it's a doc-hygiene rule, a conflict-resolution rule, and a meta-policy about which policies bind, all in one prose block.

### R-18. The "deletion-stays-deleted" pattern (`work/guards/`)
- **Belongs to:** project-specific (per-milestone implementations) and workflow (the milestone-bound shape) and general (the *pattern* itself).
- **Notes:** Same shape would apply to any repo with deletion-milestones; the *instances* are project-specific.

### R-19. The `--audit-only` flag on `aiwf promote`
- **Belongs to:** workflow (the verb) and meta-policy (the *concept* of after-the-fact ratification).
- **Notes:** Worth elevating: "ratify after the fact" is a category of policy operation distinct from "ratify before the fact."

### R-20. The dead-code-audit recipes
- **Belongs to:** workflow (the audit process) and project-specific (the per-stack tools chosen) and general (the recipe shape).
- **Notes:** Same kind of recursion as Truth Discipline: a process policy that contains substrate-specific policies that get configured per-consumer.

---

## F. Anti-patterns the corpus calls out explicitly

### R-21. Auditing on the same invocation that wrote the recipe (`dead-code-audit`)
### R-22. Hand-editing generated reports (e.g., `dead-code-report.md`)
### R-23. Treating soft-signal output as a build gate (`dead-code-audit`)
### R-24. Editing entity frontmatter status by hand (skip the FSM check + commit trailer)
### R-25. Allowlisting a too-broad guard pattern (P-44 — drop the guard rather than allowlist)
### R-26. Keeping a "temporary" compatibility shim without explicit deletion criteria
### R-27. Reconstructing semantic identity in adapters/clients from `kind` / `logicalType` / file stems when compiled facts can own that truth
### R-28. Treating aspirational docs as implementation authority
### R-29. Letting one file simultaneously act as current reference and historical archive
### R-30. Restating a canonical contract in many places from memory

- **Source:** Various (`dead-code-audit/SKILL.md` "Anti-patterns"; CLAUDE.md "Truth Discipline > Guards").
- **Rung:** 1 across the board.
- **Notes:** Anti-patterns are policies-of-prohibition. Each names a *kind of mistake* with a positive corrective. **The framework should have a place for these; today they live in skill prose, easily lost.**

---

## G. Patterns worth naming

A few recurring shapes in the corpus that don't yet have a name in the design-space
doc but probably deserve one:

### Shape α — **Comment-as-policy in `.gitignore`**
- "Deprecated/removed directories - DO NOT RECREATE" (FlowTime `.gitignore`); the framework's own .gitignore has the same shape ("this repo is the framework, not a consumer of itself").
- The gitignore line is the rung-2 mechanism; the comment is the rung-1 explanation.
- Worth: the framework should recognize `.gitignore` (or an equivalent file) as a *legitimate substrate* for one class of policy.

### Shape β — **The footprint analysis as policy evidence**
- M-066 preserves the footprint analysis verbatim (engine-LOC magnitude per option, per-template diff estimates) inside the milestone spec.
- This is not the policy itself; it's the evidence that supported policy ratification.
- Worth: ratified policies should have an attached "evidence" section that survives the ratification.

### Shape γ — **The doc sweep as conflict surfacing**
- M-066 AC-3 + AC-4: walk every routing-relevant document; classify as `aligned` / `silent` / `conflicting` / `ambiguous`; revise or mark `[needs revision per ADR-NNNN]`.
- This is a *policy-application audit*: given a new policy, walk the whole repo to find places that contradict it.
- Worth: the framework should have a verb for "given this policy, find conflicts in the docs/code." Not a small ask; rung-1 LLM-driven would be a start.

### Shape δ — **Per-milestone grep guards, named after the milestone**
- `work/guards/m-E19-02-grep-guards.sh`, `m-E19-03-grep-guards.sh`, `m-E19-04-grep-guards.sh`.
- The guard is bound to a milestone (its identity is the milestone id).
- The guard's lifecycle is "live forever after that milestone closes."
- Worth: the framework should formalize "milestone produces a permanent post-close enforcement artifact."

### Shape ε — **Defense-in-depth via gitignore**
- The framework's `.gitignore` has `.ai/`, `.ai-repo/`, `work/epics/`, `.github/skills/` to prevent consumer-only paths from leaking.
- FlowTime's `.gitignore` has `.codex/`, `.github/copilot-instructions.md`, `.claude/agents/` for the same reason.
- Worth: every repo declares its own producer/consumer asymmetry. Could be a meta-policy class.

### Shape ζ — **Soft-signal contract**
- Mentioned above (R-6). Names a *category of policy* (not a single policy).
- Distinguishes "policies that surface" from "policies that block."
- Worth lifting into the framework's vocabulary alongside the design-space's bindingness axis (advisory/warning/blocking).

### Shape η — **Two-step skill: bootstrap + use**
- `dead-code-audit` separates bootstrap (write the recipes) from use (run the audit), explicitly refusing to combine them.
- Reviewable, recoverable; bad recipes don't run blind audits.
- Worth: any policy whose enforcement requires per-consumer configuration should follow the same pattern.

---

## H. Items with no clear home

### R-31. The repo's `out/`, `data/`, `runs/`, `test-results/` directories are gitignored — implies an "outputs are ephemeral" policy
- Implicit; never stated. Could be lifted.

### R-32. The fact that `apis/` is in gitignore with a "DO NOT RECREATE" comment implies the team has tried (and reverted) an `apis/` direction at some point
- Historical scar tissue. Could be lifted to a *historical-decisions log* policy.

### R-33. The presence of `templates/` (model templates) and separate `templates-draft/` and `tests/fixtures/templates/` implies a templates-have-three-lifecycle-stages model
- Authoring (`templates-draft/`) → shipped (`templates/`) → fixture (`tests/fixtures/templates/`).
- No prose names this; it's emergent from directory structure.
- Worth: rung-2 directory-shape audits could surface conventions like this.

---

## Cross-cut observation on the rest bucket

The "rest" bucket is dominated by **meta-policies** — rules about how to write,
amend, ratify, contest, and retire policies. These are the most policy-of-policies-
shaped items in the corpus, and they are also the items the framework currently has
the least scaffolding for.

**The strongest single move the framework could make from this bucket:** adopt an
explicit *policy-class* taxonomy with three classes — *engineering policies* (rules
about the code), *workflow policies* (rules about the process), and *meta-policies*
(rules about the policies themselves) — and let each class carry the right kind of
provenance and lifecycle. The third class is currently invisible inside CLAUDE.md
prose.
