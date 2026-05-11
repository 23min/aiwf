# Roadmap

## E-0001 — Foundations and aiwf check (done)

| Milestone | Title | Status |
|---|---|---|
| M-0001 | Session 1 deliverable: aiwf check end-to-end | done |

## E-0002 — Mutating verbs and commit trailers (done)

| Milestone | Title | Status |
|---|---|---|
| M-0002 | Session 2 deliverable: mutating verbs + structured trailers | done |

## E-0003 — Skills, history, hooks (done)

| Milestone | Title | Status |
|---|---|---|
| M-0003 | Session 3 deliverable: skills + history + hooks | done |

## E-0004 — Polish for real use (done)

| Milestone | Title | Status |
|---|---|---|
| M-0004 | Session 4 deliverable: polish + doctor + render | done |

## E-0005 — Adoption surface (done)

| Milestone | Title | Status |
|---|---|---|
| M-0005 | Session 5 deliverable: import + dry-run + skip-hook | done |

## E-0006 — Iteration I1 — Contracts (done)

| Milestone | Title | Status |
|---|---|---|
| M-0006 | I1.1 — aiwfyaml package: parse, validate, round-trip the contracts: block | done |
| M-0007 | I1.2 — narrow contract entity: drop format/artifact, status set proposed→accepted→deprecated→retired+rejected | done |
| M-0008 | I1.3 — contractverify package: verify + evolve passes; substitution runner; result reclassification | done |
| M-0009 | I1.4 — contractcheck package: structural correspondence between bindings and tree | done |
| M-0010 | I1.5 — aiwf contract bind/unbind verbs; aiwf add contract --validator/--schema/--fixtures | done |
| M-0011 | I1.6 — aiwf contract recipe verbs; embedded CUE + JSON Schema recipes; --from <path> | done |
| M-0012 | I1.7 — pre-push integration: aiwf check runs verify+evolve when bindings are present | done |
| M-0013 | I1.8 — aiwf-contract skill embedded into .claude/skills/aiwf-contract/ | done |

## E-0007 — Iteration I2 — Acceptance criteria + TDD (done)

| Milestone | Title | Status |
|---|---|---|
| M-0014 | I2.1 — milestone schema additions for ACs and TDD | done |
| M-0015 | I2.2 — composite id grammar M-NNN/AC-N | done |
| M-0016 | I2.3 — AC and TDD-phase FSMs, milestone-done precondition | done |
| M-0017 | I2.4 — --force --reason on promote and cancel | done |
| M-0018 | I2.5 — aiwf-to: trailer + history renders to/forced columns | done |
| M-0019 | I2.6 — check rules for ACs, TDD audit, milestone-done | done |
| M-0020 | I2.7a — aiwf add ac, composite-id verbs, history prefix-match | done |
| M-0021 | I2.7b — --phase flag for promote, TDD pre-cycle entry | done |
| M-0022 | I2.7c — aiwf show, per-entity aggregator | done |
| M-0023 | I2.8 — STATUS.md renders AC progress per milestone | done |

## E-0008 — Iteration I2.5 — Provenance model (done)

| Milestone | Title | Status |
|---|---|---|
| M-0024 | Step 1 — drop aiwf.yaml.actor; runtime-derive identity from git config user.email | done |
| M-0025 | Step 2 — trailer writer extensions (aiwf-principal, aiwf-on-behalf-of, aiwf-authorized-by, aiwf-scope, aiwf-scope-ends, aiwf-reason) | done |
| M-0026 | Step 3 — required-together / mutually-exclusive trailer coherence rules | done |
| M-0027 | Step 4 — scope FSM package | done |
| M-0028 | Step 5 — aiwf authorize verb (open / pause / resume) | done |
| M-0029 | Step 5b — --audit-only --reason recovery mode (G24) | done |
| M-0030 | Step 5c — Apply lock-contention diagnostic (G24) | done |
| M-0031 | Step 6 — allow-rule + scope-aware verb dispatch; prior-entity chain resolution | done |
| M-0032 | Step 7 — aiwf check provenance standing rules | done |
| M-0033 | Step 7b — pre-push trailer audit (G24 surface-the-gap half) | done |
| M-0034 | Step 8 — aiwf history rendering for provenance | done |
| M-0035 | Step 9 — aiwf show scopes block | done |
| M-0036 | Step 10 — provenance docs and embedded skills | done |
| M-0037 | Step 11 — render integration handoff to I3 | done |

## E-0009 — Iteration I3 — Governance HTML render (done)

| Milestone | Title | Status |
|---|---|---|
| M-0038 | I3 Step 1 — JSON completeness on aiwf show | done |
| M-0039 | I3 Step 2 — aiwf-tests: trailer (kernel write path + opt-in warning) | done |
| M-0040 | I3 Step 3 — Render package skeleton | done |
| M-0041 | I3 Step 4 — aiwf render --format=html verb | done |
| M-0042 | I3 Step 5 — Templates and CSS (epic + milestone + entity templates, dark mode) | done |
| M-0043 | I3 Step 6 — Cross-cutting render details (Linear palette, sidebar, render report) | done |
| M-0044 | I3 Step 7 — Documentation | done |

## E-0010 — Upgrade flow (done)

| Milestone | Title | Status |
|---|---|---|
| M-0045 | Upgrade flow ship: aiwf upgrade verb + doctor skew rows + version package | done |

## E-0011 — Update broaden (done)

| Milestone | Title | Status |
|---|---|---|
| M-0046 | Update broaden ship: shared installer pipeline, pre-commit STATUS.md regen, opt-out flag, doctor reporting | done |

## E-0012 — Companion repo — Rituals plugin (done)

| Milestone | Title | Status |
|---|---|---|
| M-0047 | Rituals plugin scaffolding: aiwfx-* and wf-* namespaces, two plugins in marketplace | done |

## E-0013 — Status report (done)

| Milestone | Title | Status |
|---|---|---|
| M-0048 | Status report: cross-entity summaries + dashboard + time-window views | done |

## E-0014 — Cobra and completion (done)

### Goal

Migrate aiwf from stdlib `flag` to `github.com/spf13/cobra` so that every verb, subverb, flag, and closed-set value is tab-completable in bash and zsh — including dynamic enumeration of live entity ids. Establish "CLI surfaces must be auto-completion-friendly" as a load-bearing kernel principle, mechanically enforced by a drift-prevention test rather than reviewer vigilance.

| Milestone | Title | Status |
|---|---|---|
| M-0049 | Bootstrap Cobra and migrate version | done |
| M-0050 | Migrate read-only verbs | done |
| M-0051 | Migrate mutating verbs | done |
| M-0052 | Migrate setup verbs | done |
| M-0053 | Completion verb and static completion | done |
| M-0054 | Dynamic id completion and drift test | done |
| M-0055 | Documentation pass | done |
| M-0061 | Contract family migration + changelog retrofill + help-recursion test | done |
| M-0069 | Retrofit TDD-shaped tests for E-0014 | done |

## E-0015 — Reduce planning-verb commit cardinality (done)

### Goal

Add batching capabilities to the `aiwf add` family so a planning session produces one commit per logical mutation rather than one per entity, and close the verb-route gaps that today force users to hand-edit frontmatter. Closes G-0051 (commit-count explosion in planning sessions), G-0052 (skill/check policy contradiction over plain-git body edits), and G-0053 (no verb-flag for resolver-pointer fields on status transitions) by giving the verb routes the same expressive power that plain `git commit` currently has, while preserving the kernel's atomicity guarantee.

| Milestone | Title | Status |
|---|---|---|
| M-0056 | Add --body-file to aiwf add variants | done |
| M-0057 | Batched --title on aiwf add ac | done |
| M-0058 | Add aiwf edit-body verb and reconcile skill | done |
| M-0059 | Add resolver-pointer flags to status-transition verbs | done |
| M-0060 | Bless-current-edits mode for aiwf edit-body | done |

## E-0016 — TDD policy declaration chokepoint (closes G-0055) (proposed)

### Goal

Make every milestone's TDD policy an explicit, recorded choice at creation time. Today, `aiwf add milestone` has no `--tdd` flag and absence of the field silently maps to `tdd: none`, so an LLM (or human) following the `aiwf-add` skill faithfully produces a code milestone with no TDD tracking. This violates the kernel's "framework correctness must not depend on LLM behavior" principle. See [G-0055](../../gaps/G-055-milestone-creation-does-not-require-a-tdd-policy-declaration.md) for the empirical evidence (E-0014's M-0049..M-0055 all created with no TDD, M-0061 reproducing the pattern this week).

End state: `aiwf add milestone --tdd required|advisory|none` is the chokepoint; `aiwf.yaml: tdd.default: required` is the project-level fallback shipped by `aiwf init`; `aiwf update` migrates existing consumer repos with loud output so the policy shift is visible exactly when it lands.

| Milestone | Title | Status |
|---|---|---|
| M-0062 | tdd flag on aiwf add milestone with project-default fallback | draft |
| M-0063 | aiwf.yaml tdd.default schema and aiwf init seeding | draft |
| M-0064 | aiwf update migration for existing aiwf.yaml with loud output | draft |
| M-0065 | aiwf check finding milestone-tdd-undeclared as defense-in-depth | draft |

## E-0017 — Entity body prose chokepoint (closes G-0058) (done)

### Goal

Make non-empty body prose a kernel-enforced property across entity kinds. The design has always specified that each entity's load-bearing body sections carry prose detail (description, examples, edge cases, references — see [`docs/pocv3/plans/acs-and-tdd-plan.md:22`](../../../docs/pocv3/plans/acs-and-tdd-plan.md), [`docs/pocv3/design/design-decisions.md:139`](../../../docs/pocv3/design/design-decisions.md)) but no chokepoint enforces it: `aiwf add` verbs scaffold bare headings, existing coherence rules only check heading↔frontmatter pairing, and the `aiwf-add` skill never prompts the operator to fill the body in. Result is repo-wide skimping — every milestone M-0049..M-0061 shipped with empty AC bodies, many entities ship with bare body sections. See [G-0058](../../gaps/G-058-ac-body-sections-ship-empty-no-chokepoint-enforces-prose-intent.md) for the AC-side evidence.

End state: `aiwf check` reports `entity-body-empty` for any entity whose load-bearing body section is empty (warning by default; error under `aiwf.yaml: tdd.strict: true`); `aiwf add ac` accepts `--body-file` per AC so the body lands in the same atomic commit (the analogous flag for other `aiwf add` verbs is captured as a follow-up gap); the `aiwf-add` skill names "fill in the body" as a required follow-up step across all kinds. Together these make the design intent mechanically enforceable rather than aspirational, for every kind that ships load-bearing body prose.

| Milestone | Title | Status |
|---|---|---|
| M-0066 | aiwf check finding entity-body-empty | done |
| M-0067 | aiwf add ac --body-file flag for in-verb body scaffolding | done |
| M-0068 | aiwf-add skill names fill-in-body as required next step | done |

## E-0018 — Operator-side dogfooding completion (closes G-0062, G-0064) (done)

### Goal

Close the operator-side gap in this repo's dogfooding of aiwf. G-0038 ("kernel repo does not dogfood aiwf") landed the planning-tree migration but explicitly closed *partial* — kernel ran `aiwf init/update/check/status/import` end-to-end, but the ritual-plugin half of operator-side dogfooding was never named as a follow-up. This session surfaced the consequence: the kernel repo's design assumes `aiwf-extensions` and `wf-rituals` are present, but neither plugin is installed for this project's scope. AI assistants invoked here cannot see ritual skills (`aiwfx-start-milestone`, `wf-patch`, etc.); the standing behavior is silent ritual absence.

This epic closes the loop with two coupled milestones: [M-0070](M-070-aiwf-doctor-warning-for-missing-recommended-plugins.md) adds a kernel detection mechanism (`aiwf doctor` warns on missing recommended plugins); [M-0071](M-071-install-ritual-plugins-in-kernel-repo-document-operator-setup-path.md) installs the plugins in this repo, declares them in `aiwf.yaml`, and documents the install path in CLAUDE.md. M-0070 ships first so M-0071's fix can be validated by watching the warning go silent.

End state:
- Any consumer repo can declare `doctor.recommended_plugins` in `aiwf.yaml`; missing entries surface as `aiwf doctor` warnings.
- This repo declares its own recommended plugins, has them installed, and documents the install commands so a fresh operator (human or AI) can replicate the setup without external context.
- [G-0062](../../gaps/G-062-aiwf-doctor-does-not-surface-missing-recommended-plugins-ritual-skills-aiwf-extensions-wf-rituals-can-be-silently-absent-from-a-consumer-repo-with-no-signal-to-operator-or-ai-assistant.md) and [G-0064](../../gaps/G-064-kernel-repo-dogfooding-closed-partial-g-038-without-installing-the-ritual-plugins-aiwf-extensions-wf-rituals-operator-side-surface-incomplete-here-despite-framework-design-assuming-rituals-are-present.md) close.

| Milestone | Title | Status |
|---|---|---|
| M-0070 | aiwf doctor warning for missing recommended plugins | done |
| M-0071 | Install ritual plugins in kernel repo + document operator setup path | done |

## E-0019 — Parallel TDD subagents with finding-gated AC closure (proposed)

### Goal

Land **parallel TDD subagent execution with finding-gated AC closure**, so multi-AC milestones can run their cycles concurrently with mechanical guarantees against the M-0066/AC-1 branch-coverage drift class of bugs. The end state: a milestone with N independent ACs spawns N TDD-cycle subagents in worktree isolation; each runs its own red→green→refactor+audit; concerns surface as `finding` (F-NNN) entities; the human triages findings before AC closure; subagents structurally cannot waive their own findings.

_No milestones yet._

## E-0020 — Add list verb (closes G-0061) (done)

### Goal

Ship `aiwf list` as the AI's hot-path read primitive over the planning tree, route AI discovery to it via a split-skill design that demotes `aiwf status` to its real role (human-curated narrative), and lock the discoverability surface against drift via a kernel policy. Closes G-0061, whose central observation — *"AI assistants are instructed to invoke a non-existent verb"* — remains true today on every materialized consumer repo.

| Milestone | Title | Status |
|---|---|---|
| M-0072 | aiwf list verb, status filter-helper refactor, contract-skill drift fix | done |
| M-0073 | aiwf-list skill, aiwf-status skill tightening | done |
| M-0074 | skill-coverage policy, judgment ADR, CLAUDE.md skills section, G-0061 closure | done |

## E-0021 — Open-work synthesis: aiwfx-whiteboard skill replaces critical-path.md (done)

### Goal

Graduate the open-work synthesis pattern — the tiered landscape, recommended sequence, and pending-decisions Q&A flow that produced [`work/epics/critical-path.md`](../critical-path.md) — into a reproducible kernel feature. Ship a synthesis skill that any AI assistant routing through it can produce a fresh, current critical-path-style narrative on demand, with a Q&A gate for the operator to walk through pending decisions one at a time.

| Milestone | Title | Status |
|---|---|---|
| M-0078 | Planning-conversation skills design ADR (placement, tiering, name rationale) | done |
| M-0079 | aiwfx-whiteboard skill: classification rubric, output template, Q&A gate | done |
| M-0080 | Whiteboard skill fixture validation; retire critical-path.md; close E-0021 | done |

## E-0022 — Planning toolchain fixes (closes G-0071, G-0072, G-0065) (done)

### Goal

Ship three Tier 1 kernel-discipline fixes together before E-0020 implementation begins. Each removes a recurring source of noise or workaround in the planning workflow: lifecycle-gate the `entity-body-empty` rule (G-0071), add a writer surface for milestone `depends_on` (G-0072), and add a `retitle` verb for entities and ACs (G-0065). After this epic, planning a multi-milestone epic produces a clean tree, milestones declare their DAG via verb, and titles can be corrected when scope shifts.

| Milestone | Title | Status |
|---|---|---|
| M-0075 | Lifecycle-gate entity-body-empty rule (closes G-0071) | done |
| M-0076 | Writer surface for milestone depends_on (closes G-0072) | done |
| M-0077 | aiwf retitle verb for entities and ACs (closes G-0065) | done |

## E-0023 — Uniform 4-digit kernel ID width (closes G-0093) (done)

### Goal

Land ADR-0008's policy in code and on disk. The kernel canonicalizes every id kind to 4 digits, parsers tolerate narrower legacy widths on input, the new `aiwf rewidth` verb migrates a consumer's active tree on demand, and this repo runs the verb as one of N consumers. After this epic, §07's Slice 2 ships F at canonical F-NNNN with no separate decision; downstream consumers run `aiwf rewidth` when they're ready; new consumers post-graduation are born canonical.

| Milestone | Title | Status |
|---|---|---|
| M-0081 | Canonical 4-digit IDs in parser, renderer, and allocator | done |
| M-0082 | Implement aiwf rewidth verb and apply to this repo's tree | done |
| M-0083 | Drift check, normative-doc amendments, and skill content refresh | done |

## E-0024 — Implement uniform archive convention (ADR-0004) (done)

### Goal

Land the `aiwf archive` verb and the convergence machinery so terminal-status entities live under per-parent `archive/` subdirectories, decoupled from FSM promotion, with drift bounded by an advisory check finding plus an optional configurable threshold.

| Milestone | Title | Status |
|---|---|---|
| M-0084 | Loader and id resolver span active and archive directories | done |
| M-0085 | aiwf archive verb (dry-run default, --apply, --kind) | done |
| M-0086 | Three new archive check-rule findings and existing-rule scoping | done |
| M-0087 | Display surfaces for archived entities (status, show, render) | done |
| M-0088 | Configuration knob, embedded skill, and CLAUDE.md amendment | done |

## E-0025 — Test-suite parallelism and fixture-sharing pass — closes G-0097 (proposed)

### Goal

Convert the Go test suite from serial-with-per-test-setup to parallel-with-shared-fixtures across the load-bearing packages, so `make test` and CI's `go test` job complete in a fraction of today's wall time. The change is mechanical (a pattern, applied per package) and the spike on `spike/test-parallel` proved the pattern works on the largest single package (`internal/verb/`: ~4× faster non-race, ~2.4× with race).

| Milestone | Title | Status |
|---|---|---|
| M-0091 | Roll out TestMain + t.Parallel across internal/* test packages | draft |
| M-0092 | Roll out TestMain + t.Parallel + no-ldflags dedup to cmd/aiwf/ | draft |
| M-0093 | Document test-discipline convention and lock its chokepoint | draft |

## E-0026 — aiwf check per-code summary by default (closes G-0098) (done)

### Goal

Change the default text output of `aiwf check` from one line per finding to one line per finding-code with a count and a sample message. Errors continue to print per-instance (each is actionable); warnings collapse to per-code summaries; `--verbose` prints the full unaggregated detail. JSON envelope output is unchanged — machines still get every finding.

| Milestone | Title | Status |
|---|---|---|
| M-0089 | Per-code text-render summary with --verbose fallback | done |

## E-0027 — Trailered merge commits from aiwfx-wrap-epic (closes G-0100) (done)

### Goal

Change `aiwfx-wrap-epic`'s merge step so the merge commit it produces carries `aiwf-verb: wrap-epic`, `aiwf-entity: E-NNNN`, and `aiwf-actor: human/<id>` trailers. The merge commit becomes self-describing — `aiwf history E-NNNN` surfaces the merge event — and the kernel's `provenance-untrailered-entity-commit` rule stays strict, with the ritual now aligned to pass it cleanly.

| Milestone | Title | Status |
|---|---|---|
| M-0090 | aiwfx-wrap-epic emits trailered merge commits; fixture + drift-check tests | done |

## E-0028 — Start-epic ritual: sovereign activation with preflight, branch/worktree choice, and optional delegation (closes G-0063 start-side) (active)

### Goal

Ship the `aiwfx-start-epic` ritual plus its supporting kernel chokepoints so epic activation becomes a deliberate sovereign act with preflight checks, an explicit branch/worktree choice, and an optional principal-to-agent delegation hand-off. Closes the start-side scope of G-0063; the wrap-side concerns spawn a follow-up gap at this epic's wrap.

| Milestone | Title | Status |
|---|---|---|
| M-0094 | Add aiwf check finding epic-active-no-drafted-milestones | done |
| M-0095 | Enforce human-only actor on aiwf promote E-NN active | done |
| M-0096 | Ship aiwfx-start-epic skill with worktree and branch preflight prompts | draft |

