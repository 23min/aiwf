# Roadmap

## E-01 — Foundations and aiwf check (done)

| Milestone | Title | Status |
|---|---|---|
| M-001 | Session 1 deliverable: aiwf check end-to-end | done |

## E-02 — Mutating verbs and commit trailers (done)

| Milestone | Title | Status |
|---|---|---|
| M-002 | Session 2 deliverable: mutating verbs + structured trailers | done |

## E-03 — Skills, history, hooks (done)

| Milestone | Title | Status |
|---|---|---|
| M-003 | Session 3 deliverable: skills + history + hooks | done |

## E-04 — Polish for real use (done)

| Milestone | Title | Status |
|---|---|---|
| M-004 | Session 4 deliverable: polish + doctor + render | done |

## E-05 — Adoption surface (done)

| Milestone | Title | Status |
|---|---|---|
| M-005 | Session 5 deliverable: import + dry-run + skip-hook | done |

## E-06 — Iteration I1 — Contracts (done)

| Milestone | Title | Status |
|---|---|---|
| M-006 | I1.1 — aiwfyaml package: parse, validate, round-trip the contracts: block | done |
| M-007 | I1.2 — narrow contract entity: drop format/artifact, status set proposed→accepted→deprecated→retired+rejected | done |
| M-008 | I1.3 — contractverify package: verify + evolve passes; substitution runner; result reclassification | done |
| M-009 | I1.4 — contractcheck package: structural correspondence between bindings and tree | done |
| M-010 | I1.5 — aiwf contract bind/unbind verbs; aiwf add contract --validator/--schema/--fixtures | done |
| M-011 | I1.6 — aiwf contract recipe verbs; embedded CUE + JSON Schema recipes; --from <path> | done |
| M-012 | I1.7 — pre-push integration: aiwf check runs verify+evolve when bindings are present | done |
| M-013 | I1.8 — aiwf-contract skill embedded into .claude/skills/aiwf-contract/ | done |

## E-07 — Iteration I2 — Acceptance criteria + TDD (done)

| Milestone | Title | Status |
|---|---|---|
| M-014 | I2.1 — milestone schema additions for ACs and TDD | done |
| M-015 | I2.2 — composite id grammar M-NNN/AC-N | done |
| M-016 | I2.3 — AC and TDD-phase FSMs, milestone-done precondition | done |
| M-017 | I2.4 — --force --reason on promote and cancel | done |
| M-018 | I2.5 — aiwf-to: trailer + history renders to/forced columns | done |
| M-019 | I2.6 — check rules for ACs, TDD audit, milestone-done | done |
| M-020 | I2.7a — aiwf add ac, composite-id verbs, history prefix-match | done |
| M-021 | I2.7b — --phase flag for promote, TDD pre-cycle entry | done |
| M-022 | I2.7c — aiwf show, per-entity aggregator | done |
| M-023 | I2.8 — STATUS.md renders AC progress per milestone | done |

## E-08 — Iteration I2.5 — Provenance model (done)

| Milestone | Title | Status |
|---|---|---|
| M-024 | Step 1 — drop aiwf.yaml.actor; runtime-derive identity from git config user.email | done |
| M-025 | Step 2 — trailer writer extensions (aiwf-principal, aiwf-on-behalf-of, aiwf-authorized-by, aiwf-scope, aiwf-scope-ends, aiwf-reason) | done |
| M-026 | Step 3 — required-together / mutually-exclusive trailer coherence rules | done |
| M-027 | Step 4 — scope FSM package | done |
| M-028 | Step 5 — aiwf authorize verb (open / pause / resume) | done |
| M-029 | Step 5b — --audit-only --reason recovery mode (G24) | done |
| M-030 | Step 5c — Apply lock-contention diagnostic (G24) | done |
| M-031 | Step 6 — allow-rule + scope-aware verb dispatch; prior-entity chain resolution | done |
| M-032 | Step 7 — aiwf check provenance standing rules | done |
| M-033 | Step 7b — pre-push trailer audit (G24 surface-the-gap half) | done |
| M-034 | Step 8 — aiwf history rendering for provenance | done |
| M-035 | Step 9 — aiwf show scopes block | done |
| M-036 | Step 10 — provenance docs and embedded skills | done |
| M-037 | Step 11 — render integration handoff to I3 | done |

## E-09 — Iteration I3 — Governance HTML render (done)

| Milestone | Title | Status |
|---|---|---|
| M-038 | I3 Step 1 — JSON completeness on aiwf show | done |
| M-039 | I3 Step 2 — aiwf-tests: trailer (kernel write path + opt-in warning) | done |
| M-040 | I3 Step 3 — Render package skeleton | done |
| M-041 | I3 Step 4 — aiwf render --format=html verb | done |
| M-042 | I3 Step 5 — Templates and CSS (epic + milestone + entity templates, dark mode) | done |
| M-043 | I3 Step 6 — Cross-cutting render details (Linear palette, sidebar, render report) | done |
| M-044 | I3 Step 7 — Documentation | done |

## E-10 — Upgrade flow (done)

| Milestone | Title | Status |
|---|---|---|
| M-045 | Upgrade flow ship: aiwf upgrade verb + doctor skew rows + version package | done |

## E-11 — Update broaden (done)

| Milestone | Title | Status |
|---|---|---|
| M-046 | Update broaden ship: shared installer pipeline, pre-commit STATUS.md regen, opt-out flag, doctor reporting | done |

## E-12 — Companion repo — Rituals plugin (done)

| Milestone | Title | Status |
|---|---|---|
| M-047 | Rituals plugin scaffolding: aiwfx-* and wf-* namespaces, two plugins in marketplace | done |

## E-13 — Status report (done)

| Milestone | Title | Status |
|---|---|---|
| M-048 | Status report: cross-entity summaries + dashboard + time-window views | done |

## E-14 — Cobra and completion (done)

### Goal

Migrate aiwf from stdlib `flag` to `github.com/spf13/cobra` so that every verb, subverb, flag, and closed-set value is tab-completable in bash and zsh — including dynamic enumeration of live entity ids. Establish "CLI surfaces must be auto-completion-friendly" as a load-bearing kernel principle, mechanically enforced by a drift-prevention test rather than reviewer vigilance.

| Milestone | Title | Status |
|---|---|---|
| M-049 | Bootstrap Cobra and migrate version | done |
| M-050 | Migrate read-only verbs | done |
| M-051 | Migrate mutating verbs | done |
| M-052 | Migrate setup verbs | done |
| M-053 | Completion verb and static completion | done |
| M-054 | Dynamic id completion and drift test | done |
| M-055 | Documentation pass | done |
| M-061 | Contract family migration + changelog retrofill + help-recursion test | done |
| M-069 | Retrofit TDD-shaped tests for E-14 | done |

## E-15 — Reduce planning-verb commit cardinality (done)

### Goal

Add batching capabilities to the `aiwf add` family so a planning session produces one commit per logical mutation rather than one per entity, and close the verb-route gaps that today force users to hand-edit frontmatter. Closes G-051 (commit-count explosion in planning sessions), G-052 (skill/check policy contradiction over plain-git body edits), and G-053 (no verb-flag for resolver-pointer fields on status transitions) by giving the verb routes the same expressive power that plain `git commit` currently has, while preserving the kernel's atomicity guarantee.

| Milestone | Title | Status |
|---|---|---|
| M-056 | Add --body-file to aiwf add variants | done |
| M-057 | Batched --title on aiwf add ac | done |
| M-058 | Add aiwf edit-body verb and reconcile skill | done |
| M-059 | Add resolver-pointer flags to status-transition verbs | done |
| M-060 | Bless-current-edits mode for aiwf edit-body | done |

## E-16 — TDD policy declaration chokepoint (closes G-055) (proposed)

### Goal

Make every milestone's TDD policy an explicit, recorded choice at creation time. Today, `aiwf add milestone` has no `--tdd` flag and absence of the field silently maps to `tdd: none`, so an LLM (or human) following the `aiwf-add` skill faithfully produces a code milestone with no TDD tracking. This violates the kernel's "framework correctness must not depend on LLM behavior" principle. See [G-055](work/gaps/G-055-milestone-creation-does-not-require-a-tdd-policy-declaration.md) for the empirical evidence (E-14's M-049..M-055 all created with no TDD, M-061 reproducing the pattern this week).

End state: `aiwf add milestone --tdd required|advisory|none` is the chokepoint; `aiwf.yaml: tdd.default: required` is the project-level fallback shipped by `aiwf init`; `aiwf update` migrates existing consumer repos with loud output so the policy shift is visible exactly when it lands.

| Milestone | Title | Status |
|---|---|---|
| M-062 | tdd flag on aiwf add milestone with project-default fallback | draft |
| M-063 | aiwf.yaml tdd.default schema and aiwf init seeding | draft |
| M-064 | aiwf update migration for existing aiwf.yaml with loud output | draft |
| M-065 | aiwf check finding milestone-tdd-undeclared as defense-in-depth | draft |

## E-17 — Entity body prose chokepoint (closes G-058) (done)

### Goal

Make non-empty body prose a kernel-enforced property across entity kinds. The design has always specified that each entity's load-bearing body sections carry prose detail (description, examples, edge cases, references — see [`docs/pocv3/plans/acs-and-tdd-plan.md:22`](docs/pocv3/plans/acs-and-tdd-plan.md), [`docs/pocv3/design/design-decisions.md:139`](docs/pocv3/design/design-decisions.md)) but no chokepoint enforces it: `aiwf add` verbs scaffold bare headings, existing coherence rules only check heading↔frontmatter pairing, and the `aiwf-add` skill never prompts the operator to fill the body in. Result is repo-wide skimping — every milestone M-049..M-061 shipped with empty AC bodies, many entities ship with bare body sections. See [G-058](work/gaps/G-058-ac-body-sections-ship-empty-no-chokepoint-enforces-prose-intent.md) for the AC-side evidence.

End state: `aiwf check` reports `entity-body-empty` for any entity whose load-bearing body section is empty (warning by default; error under `aiwf.yaml: tdd.strict: true`); `aiwf add ac` accepts `--body-file` per AC so the body lands in the same atomic commit (the analogous flag for other `aiwf add` verbs is captured as a follow-up gap); the `aiwf-add` skill names "fill in the body" as a required follow-up step across all kinds. Together these make the design intent mechanically enforceable rather than aspirational, for every kind that ships load-bearing body prose.

| Milestone | Title | Status |
|---|---|---|
| M-066 | aiwf check finding entity-body-empty | done |
| M-067 | aiwf add ac --body-file flag for in-verb body scaffolding | done |
| M-068 | aiwf-add skill names fill-in-body as required next step | done |

## E-18 — Operator-side dogfooding completion (closes G-062, G-064) (done)

### Goal

Close the operator-side gap in this repo's dogfooding of aiwf. G-038 ("kernel repo does not dogfood aiwf") landed the planning-tree migration but explicitly closed *partial* — kernel ran `aiwf init/update/check/status/import` end-to-end, but the ritual-plugin half of operator-side dogfooding was never named as a follow-up. This session surfaced the consequence: the kernel repo's design assumes `aiwf-extensions` and `wf-rituals` are present, but neither plugin is installed for this project's scope. AI assistants invoked here cannot see ritual skills (`aiwfx-start-milestone`, `wf-patch`, etc.); the standing behavior is silent ritual absence.

This epic closes the loop with two coupled milestones: [M-070](work/epics/E-18-operator-side-dogfooding-completion-closes-g-062-g-064/M-070-aiwf-doctor-warning-for-missing-recommended-plugins.md) adds a kernel detection mechanism (`aiwf doctor` warns on missing recommended plugins); [M-071](work/epics/E-18-operator-side-dogfooding-completion-closes-g-062-g-064/M-071-install-ritual-plugins-in-kernel-repo-document-operator-setup-path.md) installs the plugins in this repo, declares them in `aiwf.yaml`, and documents the install path in CLAUDE.md. M-070 ships first so M-071's fix can be validated by watching the warning go silent.

End state:
- Any consumer repo can declare `doctor.recommended_plugins` in `aiwf.yaml`; missing entries surface as `aiwf doctor` warnings.
- This repo declares its own recommended plugins, has them installed, and documents the install commands so a fresh operator (human or AI) can replicate the setup without external context.
- [G-062](work/gaps/G-062-aiwf-doctor-does-not-surface-missing-recommended-plugins-ritual-skills-aiwf-extensions-wf-rituals-can-be-silently-absent-from-a-consumer-repo-with-no-signal-to-operator-or-ai-assistant.md) and [G-064](work/gaps/G-064-kernel-repo-dogfooding-closed-partial-g-038-without-installing-the-ritual-plugins-aiwf-extensions-wf-rituals-operator-side-surface-incomplete-here-despite-framework-design-assuming-rituals-are-present.md) close.

| Milestone | Title | Status |
|---|---|---|
| M-070 | aiwf doctor warning for missing recommended plugins | done |
| M-071 | Install ritual plugins in kernel repo + document operator setup path | done |

## E-19 — Parallel TDD subagents with finding-gated AC closure (proposed)

### Goal

Land **parallel TDD subagent execution with finding-gated AC closure**, so multi-AC milestones can run their cycles concurrently with mechanical guarantees against the M-066/AC-1 branch-coverage drift class of bugs. The end state: a milestone with N independent ACs spawns N TDD-cycle subagents in worktree isolation; each runs its own red→green→refactor+audit; concerns surface as `finding` (F-NNN) entities; the human triages findings before AC closure; subagents structurally cannot waive their own findings.

_No milestones yet._

## E-20 — Add list verb (closes G-061) (proposed)

### Goal

Ship `aiwf list` as the AI's hot-path read primitive over the planning tree, route AI discovery to it via a split-skill design that demotes `aiwf status` to its real role (human-curated narrative), and lock the discoverability surface against drift via a kernel policy. Closes G-061, whose central observation — *"AI assistants are instructed to invoke a non-existent verb"* — remains true today on every materialized consumer repo.

| Milestone | Title | Status |
|---|---|---|
| M-072 | aiwf list verb, status filter-helper refactor, contract-skill drift fix | draft |
| M-073 | aiwf-list skill, aiwf-status skill tightening | draft |
| M-074 | skill-coverage policy, judgment ADR, CLAUDE.md skills section, G-061 closure | draft |

## E-21 — Open-work synthesis: recommended-sequence skill (replaces critical-path.md) (proposed)

### Goal

Graduate the open-work synthesis pattern — the tiered landscape, recommended sequence, and pending-decisions Q&A flow that produced [`work/epics/critical-path.md`](work/epics/critical-path.md) — into a reproducible kernel feature. Ship a synthesis skill that any AI assistant routing through it can produce a fresh, current critical-path-style narrative on demand, with a Q&A gate for the operator to walk through pending decisions one at a time.

| Milestone | Title | Status |
|---|---|---|
| M-078 | Planning-conversation skills design ADR (placement, tiering, name rationale) | draft |
| M-079 | aiwfx-whiteboard skill: classification rubric, output template, Q&A gate | draft |
| M-080 | Whiteboard skill fixture validation; retire critical-path.md; close E-21 | draft |

## E-22 — Planning toolchain fixes (closes G-071, G-072, G-065) (done)

### Goal

Ship three Tier 1 kernel-discipline fixes together before E-20 implementation begins. Each removes a recurring source of noise or workaround in the planning workflow: lifecycle-gate the `entity-body-empty` rule (G-071), add a writer surface for milestone `depends_on` (G-072), and add a `retitle` verb for entities and ACs (G-065). After this epic, planning a multi-milestone epic produces a clean tree, milestones declare their DAG via verb, and titles can be corrected when scope shifts.

| Milestone | Title | Status |
|---|---|---|
| M-075 | Lifecycle-gate entity-body-empty rule (closes G-071) | done |
| M-076 | Writer surface for milestone depends_on (closes G-072) | done |
| M-077 | aiwf retitle verb for entities and ACs (closes G-065) | done |

