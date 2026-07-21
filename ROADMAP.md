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

## E-0016 — TDD policy declaration chokepoint (closes G-0055) (cancelled)

### Goal

Make every milestone's TDD policy an explicit, recorded choice at creation time. Today, `aiwf add milestone` has no `--tdd` flag and absence of the field silently maps to `tdd: none`, so an LLM (or human) following the `aiwf-add` skill faithfully produces a code milestone with no TDD tracking. This violates the kernel's "framework correctness must not depend on LLM behavior" principle. See [G-0055](work/gaps/archive/G-0055-milestone-creation-does-not-require-a-tdd-policy-declaration.md) for the empirical evidence (E-0014's M-0049..M-0055 all created with no TDD, M-0061 reproducing the pattern this week).

End state: `aiwf add milestone --tdd required|advisory|none` is the chokepoint; `aiwf.yaml: tdd.default: required` is the project-level fallback shipped by `aiwf init`; `aiwf update` migrates existing consumer repos with loud output so the policy shift is visible exactly when it lands.

| Milestone | Title | Status |
|---|---|---|
| M-0062 | tdd flag on aiwf add milestone with project-default fallback | cancelled |
| M-0063 | aiwf.yaml tdd.default schema and aiwf init seeding | cancelled |
| M-0064 | aiwf update migration for existing aiwf.yaml with loud output | cancelled |
| M-0065 | aiwf check finding milestone-tdd-undeclared as defense-in-depth | cancelled |

## E-0017 — Entity body prose chokepoint (closes G-0058) (done)

### Goal

Make non-empty body prose a kernel-enforced property across entity kinds. The design has always specified that each entity's load-bearing body sections carry prose detail (description, examples, edge cases, references — see [`docs/pocv3/plans/acs-and-tdd-plan.md:22`](../../../docs/pocv3/plans/acs-and-tdd-plan.md), [`docs/pocv3/design/design-decisions.md:139`](../../../docs/pocv3/design/design-decisions.md)) but no chokepoint enforces it: `aiwf add` verbs scaffold bare headings, existing coherence rules only check heading↔frontmatter pairing, and the `aiwf-add` skill never prompts the operator to fill the body in. Result is repo-wide skimping — every milestone M-0049..M-0061 shipped with empty AC bodies, many entities ship with bare body sections. See [G-0058](work/gaps/archive/G-0058-ac-body-sections-ship-empty-no-chokepoint-enforces-prose-intent.md) for the AC-side evidence.

End state: `aiwf check` reports `entity-body-empty` for any entity whose load-bearing body section is empty (warning by default; error under `aiwf.yaml: tdd.strict: true`); `aiwf add ac` accepts `--body-file` per AC so the body lands in the same atomic commit (the analogous flag for other `aiwf add` verbs is captured as a follow-up gap); the `aiwf-add` skill names "fill in the body" as a required follow-up step across all kinds. Together these make the design intent mechanically enforceable rather than aspirational, for every kind that ships load-bearing body prose.

| Milestone | Title | Status |
|---|---|---|
| M-0066 | aiwf check finding entity-body-empty | done |
| M-0067 | aiwf add ac --body-file flag for in-verb body scaffolding | done |
| M-0068 | aiwf-add skill names fill-in-body as required next step | done |

## E-0018 — Operator-side dogfooding completion (closes G-0062, G-0064) (done)

### Goal

Close the operator-side gap in this repo's dogfooding of aiwf. G-0038 ("kernel repo does not dogfood aiwf") landed the planning-tree migration but explicitly closed *partial* — kernel ran `aiwf init/update/check/status/import` end-to-end, but the ritual-plugin half of operator-side dogfooding was never named as a follow-up. This session surfaced the consequence: the kernel repo's design assumes `aiwf-extensions` and `wf-rituals` are present, but neither plugin is installed for this project's scope. AI assistants invoked here cannot see ritual skills (`aiwfx-start-milestone`, `wf-patch`, etc.); the standing behavior is silent ritual absence.

This epic closes the loop with two coupled milestones: [M-0070](work/epics/archive/E-0018-operator-side-dogfooding-completion-closes-g-062-g-064/M-0070-aiwf-doctor-warning-for-missing-recommended-plugins.md) adds a kernel detection mechanism (`aiwf doctor` warns on missing recommended plugins); [M-0071](work/epics/archive/E-0018-operator-side-dogfooding-completion-closes-g-062-g-064/M-0071-install-ritual-plugins-in-kernel-repo-document-operator-setup-path.md) installs the plugins in this repo, declares them in `aiwf.yaml`, and documents the install path in CLAUDE.md. M-0070 ships first so M-0071's fix can be validated by watching the warning go silent.

End state:
- Any consumer repo can declare `doctor.recommended_plugins` in `aiwf.yaml`; missing entries surface as `aiwf doctor` warnings.
- This repo declares its own recommended plugins, has them installed, and documents the install commands so a fresh operator (human or AI) can replicate the setup without external context.
- [G-0062](work/gaps/archive/G-0062-aiwf-doctor-does-not-surface-missing-recommended-plugins-ritual-skills-aiwf-extensions-wf-rituals-can-be-silently-absent-from-a-consumer-repo-with-no-signal-to-operator-or-ai-assistant.md) and [G-0064](work/gaps/archive/G-0064-kernel-repo-dogfooding-closed-partial-g-038-without-installing-the-ritual-plugins-aiwf-extensions-wf-rituals-operator-side-surface-incomplete-here-despite-framework-design-assuming-rituals-are-present.md) close.

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

## E-0025 — Test-suite parallelism and fixture-sharing pass — closes G-0097 (done)

### Goal

Convert the Go test suite from serial-with-per-test-setup to parallel-with-shared-fixtures across the load-bearing packages, so `make test` and CI's `go test` job complete in a fraction of today's wall time. The change is mechanical (a pattern, applied per package) and the spike on `spike/test-parallel` proved the pattern works on the largest single package (`internal/verb/`: ~4× faster non-race, ~2.4× with race).

| Milestone | Title | Status |
|---|---|---|
| M-0091 | Roll out TestMain + t.Parallel across internal/* test packages | done |
| M-0092 | Roll out TestMain + t.Parallel + no-ldflags dedup to cmd/aiwf/ | done |
| M-0093 | Document test-discipline convention and lock its chokepoint | done |

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

## E-0028 — Start-epic ritual: sovereign activation with preflight + delegation (done)

### Goal

Ship the `aiwfx-start-epic` ritual plus its supporting kernel chokepoints so epic activation becomes a deliberate sovereign act with preflight checks, an explicit branch/worktree choice, and an optional principal-to-agent delegation hand-off. Closes the start-side scope of G-0063; the wrap-side concerns spawn a follow-up gap at this epic's wrap.

| Milestone | Title | Status |
|---|---|---|
| M-0094 | Add aiwf check finding epic-active-no-drafted-milestones | done |
| M-0095 | Enforce human-only actor on aiwf promote E-NN active | done |
| M-0096 | Ship aiwfx-start-epic skill with worktree and branch preflight prompts | done |
| M-0097 | Close M-0094/95/96 verification seams: M-0095 automation audit chokepoint and AC-5 drift comparator | done |

## E-0029 — Glanceable governance HTML render: layout, sidebar, chips (closes G-0114) (done)

### Goal

Make the rendered governance site usable for current-state synthesis at a glance. The layout fills the viewport with the sidebar flush-left; the sidebar surfaces gaps with the active count; per-kind index pages collapse from active/all-pair to a single file with `:target`-driven filter chips at the top; within `gaps.html` the open subset pops visually rather than sitting equally-weighted with the addressed rows.

| Milestone | Title | Status |
|---|---|---|
| M-0098 | Render-site layout overhaul: viewport-fill body, flush-left sidebar, prose cap | done |
| M-0099 | Kind-index chip filter: single emitted file per kind with :target chips | done |
| M-0100 | Sidebar adds gap entry + epic archive chip filter | done |
| M-0101 | In-page status hierarchy in gaps.html | cancelled |
| M-0107 | Repair Playwright e2e suite for current kernel state | done |

## E-0030 — Branch model chokepoint: --branch flag, sequencing, isolation-escape finding (done)

### Goal

Make [ADR-0010](docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s two-tier branch model mechanically enforceable end-to-end: AI-actor multi-commit work cannot escape a ritual branch context, the rituals create branches in the right sequence so kernel state-of-the-world stays visible from main throughout the cycle, the finding rule that catches drift is itself a cell in the layer-4 branch-choreography spec ([ADR-0011](docs/adr/ADR-0011-legal-workflow-spec-methodology.md) §"Scope"), and the human override path is gated by a typed sovereign-trailer signature.

| Milestone | Title | Status |
|---|---|---|
| M-0102 | aiwf authorize --branch flag + scope-branch trailer coupling | done |
| M-0103 | AI-side preflight: aiwf authorize refuses without ritual branch context | done |
| M-0104 | aiwfx-start-epic sequencing fix (closes G-0116) | done |
| M-0105 | aiwfx-start-milestone sequencing alignment | done |
| M-0106 | Kernel finding isolation-escape (closes G-0099) | done |
| M-0158 | Layer-4 branch-choreography spec cells + drift-policy extension | done |
| M-0159 | Real-world hardening of branch-model chokepoint | done |
| M-0160 | Operational pain — reallocate stress, trunk-collision regress, apply rollback | done |
| M-0161 | Imagination-driven hardening: shallow, force-push, rename, detached, trunk | done |
| M-0162 | Layer-4 spec-catalog refactor: bijection + Pin registry | done |

## E-0031 — Pin legal workflows, composition, and branch choreography mechanically (cancelled)

### Goal

Workflow legality — the multi-step procedures contributors walk through to ship value — moves from prose-only recipes in skill bodies to a declarative spec backed by composition integration tests and a verb-sequence fuzz harness. The chokepoint becomes "tests pass under arbitrary legal composition, including branch transitions," not "the recipe author and the recipe reader both remembered the right sequence."

| Milestone | Title | Status |
|---|---|---|
| M-0108 | Author legal-workflows.md spec enumerating every blessed workflow | cancelled |
| M-0109 | internal/workflows/ test harness with one workflow as seam test | cancelled |
| M-0110 | Per-workflow integration test coverage including G-0118 regression | cancelled |
| M-0111 | Skill-citation discipline and skill-spec drift-prevention test | cancelled |
| M-0112 | Verb-sequence fuzz harness with spec-derived seeds | cancelled |

## E-0032 — Idiomatic-Go cleanup completion and enum-adoption chokepoint (done)

### Goal

Close G-0107 by moving every top-level verb in `cmd/aiwf/` (~27 verbs across 19 single-verb files plus the 8-verb `verbs_cmd.go` cluster) into per-verb subpackages under `internal/cli/<verb>/`, shrink `cmd/aiwf/main.go` to G-0107's target ~30-line entry shape, and add the AST-based policy that prevents enum-constant adoption drift (G-0126). After this epic lands, `cmd/aiwf/` contains `main.go` only; verb code, helpers, and tests live under `internal/cli/`. The chokepoint becomes mechanical: closed-set comparison-site adoption is a CI test, not reviewer vigilance.

| Milestone | Title | Status |
|---|---|---|
| M-0113 | Consolidate trailer parser | done |
| M-0114 | Lift completion helpers to internal/cli/cliutil/completion.go | done |
| M-0115 | Move verbs_cmd.go's 8 verbs to internal/cli/<verb>/ subpackages | done |
| M-0116 | Move 16 single-command verbs to internal/cli/<verb>/ subpackages | done |
| M-0117 | Move contract, doctor, milestone (multi-subcommand) to subpackages | done |
| M-0118 | Shrink main.go to entry-only; supporting files find homes | done |
| M-0119 | Add enum_literal_adoption policy; fix surfaced literal sites | done |

## E-0033 — Pin legal kernel-verb workflows mechanically (done)

### Goal

Verify mechanically — not by prose catalog or LLM recall — that the aiwf binary only permits legal sequences of kernel-verb invocations against the planning tree, and rejects illegal ones with named exit codes or finding codes. The deliverable is a **canonical Go spec table** describing the legal/illegal frontier at the kernel-verb level, plus per-cell positive and negative tests under `internal/policies/` that exercise the binary against the table.

This epic replaces the cancelled E-0031, whose first attempt produced a prose catalog and hand-coded fuzz harness — neither of which could mechanically catch an implementation bug. The structural critique that killed E-0031: a legality spec must be a **machine-readable transition surface**, not a narrative description, and tests must drive the binary against that surface cell-by-cell.

| Milestone | Title | Status |
|---|---|---|
| M-0120 | Ratify legal-workflow spec methodology in ADR | done |
| M-0121 | Pass A audit: catalog legal-workflow rules from existing surfaces | done |
| M-0122 | Pass B first-principles: derive legal-workflow rules from entity model | done |
| M-0123 | Pass C reconcile to canonical Go spec table + drift policy | done |
| M-0124 | Positive cell coverage: legal workflows succeed with expected post-state | done |
| M-0125 | Negative cell coverage: illegal workflows rejected with named errors | done |
| M-0130 | Implement fsm-history-consistent check rule for FSM tree-invariant | done |
| M-0131 | State-aware CancelTarget for Contract: cancel deprecated targets retired | done |
| M-0136 | aiwf acknowledge-illegal: retroactive force trailer for historical violations | done |
| M-0137 | fsm-history-consistent: batched git ops + silent-swallow fix | done |

## E-0034 — Retire docs/pocv3/ and declare doc-authority hierarchy (proposed)

### Goal

Refactor `docs/` so a reader (human or LLM) can identify each file's authority tier from its path. Retire the historical `docs/pocv3/` directory by relocating its surviving content, archiving its pre-dogfooding artifacts, and declaring the resulting hierarchy in CLAUDE.md.

| Milestone | Title | Status |
|---|---|---|
| M-0126 | Triage docs/pocv3/ into per-file disposition table | draft |
| M-0127 | Relocate docs/pocv3/ contents and sweep cross-references | draft |
| M-0128 | Declare doc-authority hierarchy in CLAUDE.md | draft |
| M-0129 | Drift chokepoint: forbid docs/pocv3/ literals in Go code | draft |

## E-0035 — Devcontainer-based dev loop (done)

### Goal

Move aiwf's primary dev loop from the macOS host into a reproducible
Linux devcontainer, where macOS-specific bugs (G-0127 fork/exec
deadlock under `-race` + parallel; G-0128/G-0133 syspolicyd crashes
on unsigned Mach-O binaries) simply don't exist. The existing
host-side workarounds — `scripts/sign-and-run.sh`, in-test
`codesign` blocks, the `-parallel 8` cap — stay as graceful
fallbacks for the rare case where host execution is necessary,
but the canonical dev surface becomes the container. Success
means a fresh checkout + "Reopen in Container" gives any
contributor the same green `make ci` without remembering the
macOS DO/DON'T rules.

| Milestone | Title | Status |
|---|---|---|
| M-0132 | Land .devcontainer skeleton (features-first, Go base, project-scope plugins) | done |
| M-0133 | Multi-context kernel surfaces: portable hooks + doctor check | done |
| M-0134 | CLAUDE.md DO/DON'T refresh: container primary, macOS host fallback | done |
| M-0135 | aiwf doctor containerized-env awareness: detection + mount check | done |

## E-0036 — Reconcile impl to the legal-workflow spec, retiring deferred error codes (done)

### Goal

Make E-0033's legal-workflow spec a *fully* verified source of truth by reconciling the kernel impl to it. Concretely: retire the `deferredImplErrorCodes` IOU list so every illegal cell the spec names actually **fails-verified** through the binary, and every legality-pertinent finding code is provably referenced by a spec rule (the bidirectional-completeness guarantee). The enabling deliverable is a **typed `CodedError` pattern** that lets verb-time refusals carry a first-class, structured error code — `errors.As`-able for the JSON envelope and visible to the AC-5 spec↔impl scanner, mirroring the existing `check.Finding{Code}` shape.

| Milestone | Title | Status |
|---|---|---|
| M-0138 | Introduce typed CodedError; convert existing unstructured legality errors | done |
| M-0139 | Refuse cancel of parents with non-terminal children/ACs via coded errors | done |
| M-0140 | Classify legality finding codes; close AC-5 bidirectional arm | done |
| M-0141 | Enforce three-edge scope reachability at verb-time | done |
| M-0142 | Rename gap-resolved-has-resolver to match the gap FSM vocabulary | done |
| M-0143 | Surface Coded error codes in the JSON envelope | done |

## E-0037 — Make scope-reach an executable legality precondition in the spec (done)

### Goal

Make `scope-reach` (D-0006's three-edge scope reachability) an **executable, legality-classed predicate** in the legal-workflow spec, so the verb-time out-of-scope refusal lands inside the spec's bidirectional drift net — completing the formal-model certification M-0141 deliberately deferred.

| Milestone | Title | Status |
|---|---|---|
| M-0144 | ADR: represent a global precondition; classify out-of-scope as legality | done |
| M-0145 | Implement scope-reach in EvaluatePredicate with verb-invocation context | done |
| M-0146 | Extend cellcoverage with authorized-scope fixtures | done |
| M-0147 | Land global scope-reach rule; reclassify code; AC-5 fourth arm green | done |

## E-0038 — Agent-agnostic rituals distribution via embed-and-materialize (done)

### Goal

Make `aiwf` itself the distribution mechanism for the rituals — vendor a pinned snapshot into the aiwf repo, embed it, and materialize it on `aiwf init` / `aiwf update` — so a consumer gets the planning skills, lifecycle rituals, agents, and templates with **one command and no `/plugin` step**, and so adding a non-Claude agent target later is a new writer rather than a distribution rethink. Retire the Claude marketplace channel once the embedded path is stable. Implements ADR-0014; addresses G-0177.

| Milestone | Title | Status |
|---|---|---|
| M-0148 | Vendor-sync: pull pinned rituals snapshot into the aiwf repo + drift test | done |
| M-0149 | Embed + materialize ritual skills (aiwfx-/wf-); extend manifest + gitignore | done |
| M-0150 | Embed + materialize ritual agents (.claude/agents/) and templates | done |
| M-0151 | Agent-target seam in the materializer (Claude writer behind the seam) | done |
| M-0152 | Marketplace sunset: doctor flip, de-dupe guard, docs rewrite | done |

## E-0039 — Optional install path for the aiwf-aware statusline (closes G-0183) (done)

### Goal

Let a downstream aiwf consumer opt into the aiwf-aware Claude Code statusline
via `aiwf init/update --statusline` — portable across Linux and macOS, with
activation gated by explicit per-invocation consent — without aiwf ever quietly
editing a settings file.

| Milestone | Title | Status |
|---|---|---|
| M-0153 | Statusline script portability and robustness fixes | done |
| M-0154 | ADR: amend settings.json stance to consent-gated | done |
| M-0155 | Embed statusline and add --statusline scaffold with --scope | done |
| M-0156 | Consent-gated statusline settings wiring | done |
| M-0157 | aiwf doctor statusline block | done |

## E-0040 — Materialize per-turn aiwf guidance into consumer CLAUDE.md (closes G-0243) (done)

### Goal

Give aiwf a consent-gated write channel into the one consumer surface that is
re-injected on every turn and survives `/compact` — the consumer's `CLAUDE.md` —
so that the advisory rules aiwf cannot mechanically enforce actually bind in
consumer trees, not just in this repo.

| Milestone | Title | Status |
|---|---|---|
| M-0163 | Embed and materialize the guidance fragment | done |
| M-0164 | Wire the CLAUDE.md guidance import with consent | done |
| M-0165 | Surface unwired CLAUDE.md guidance in aiwf doctor | done |

## E-0041 — Seam conformance suites for multi-implementation kernel interfaces (cancelled)

### Goal

Make every kernel "unsigned-cheque" interface seam — a `type Foo interface { … }` that admits more than one implementation claiming interchangeability — own a single conformance matrix that proves the implementations actually agree, so silent drift between a production impl and its test double (or between two production impls) fails CI instead of shipping. This converts the kernel's per-implementation-isolated-test posture into the rubric's "one suite, parameterized over implementations" pattern (D2 §"Equivalence tests at seams"), and adds the drift policy that keeps the discipline mechanical for future seams. Closes [G-0222](work/gaps/archive/G-0222-no-shared-conformance-suites-at-unsigned-cheque-interface-seams.md).

_No milestones yet._

## E-0042 — Burn down test-quality debt across policies and the test corpus (done)

| Milestone | Title | Status |
|---|---|---|
| M-0166 | Firing fixtures for the easy-majority dark policies | done |
| M-0167 | Verified linter cleanups and fsm-invariants ledger annotation | done |
| M-0168 | Corpus-wide mutate-hunt sweep over the kernel packages | done |
| M-0169 | Directed wf-vacuity pass over the load-bearing units | done |
| M-0170 | Firing tests for linter-config rules and the dormant forbidigo fix | done |

## E-0043 — Optional area tag for grouping entities by workstream (done)

### Goal

Let a single repo hold more than one workstream — a product plus a co-developed internal tool, or a monorepo of several packages — by tagging entities with a validated, optional `area`. Roadmaps, status, and checks become scopeable per workstream, while the flat, globally-unique id space stays exactly as it is today.

| Milestone | Title | Status |
|---|---|---|
| M-0171 | Area field on root kinds and aiwf.yaml areas block with validation | done |
| M-0172 | area-unknown check finding for undeclared area values | done |
| M-0173 | aiwf add --area write path with completion and discovered-in derivation | done |
| M-0174 | --area filter on list, show, and status | done |
| M-0175 | Area grouping in status, render roadmap, and render html | done |

## E-0044 — Harden the areas feature for multi-project (1:1) monorepo use (done)

### Goal

Make `--area` filtering **trustworthy** for the multi-project monorepo — the area feature's primary intended use — by anchoring each area to the path glob of the project it represents. Once an area knows where its project lives, the kernel regains an oracle (the project's paths) it structurally lacks for a purely semantic boundary, and the checks aiwf "can't have" for a label-only tag all become buildable. The payoff: `aiwf list --area app-a` becomes a reliable "all app-a work," promoting the filter from convenience to load-bearing.

| Milestone | Title | Status |
|---|---|---|
| M-0176 | Partition totality and disjointness property test for areagroup | done |
| M-0177 | aiwf rename-area verb with atomic cross-entity rewrite | done |
| M-0178 | areas.required knob promoting untagged entities to a blocking finding | done |
| M-0179 | paths per-area config evolution with backward-compatible unmarshaler | done |
| M-0180 | Area-path dead-glob and overlap checks | done |
| M-0181 | Mistag detection via aiwf-entity trailer with acknowledge path | done |
| M-0182 | Area discoverability skill and path-hint derivation at aiwf add | done |
| M-0183 | aiwf set-area verb to tag one entity to a declared area member | done |
| M-0184 | Reserved global area value: predicate, whitelist, and verb acceptance | done |
| M-0185 | Area-path scoped-coverage check (unslotted-project detection) | done |
| M-0208 | rename-area preserves comments and sibling keys in the areas block on rename | done |

## E-0045 — Plumbing-based commit construction for aiwf verbs (done)

### Goal

Replace aiwf's fragile `git stash`-based per-verb commit isolation with a plumbing-based commit-construction primitive (temp index + `commit-tree`) that never mutates the live index or worktree — making per-verb commit atomicity robust by construction, and giving aiwf a single, reusable commit-construction substrate.

| Milestone | Title | Status |
|---|---|---|
| M-0186 | gitops commit primitive via temp-index and commit-tree | done |
| M-0187 | Opt-in gaps inbox on a never-checked-out ref | cancelled |

## E-0046 — Formalize in-repo worktrees as the default placement (done)

### Goal

Make in-repo worktrees (`.claude/worktrees/<branch>/`) aiwf's default placement for
ritual worktrees, so a Claude session inside a sandboxed devcontainer can root in its
worktree — and record the non-obvious rationale so the default is not reverted to the
sibling-worktree git convention.

| Milestone | Title | Status |
|---|---|---|
| M-0188 | Pin that the loader ignores in-repo worktrees under .claude/worktrees | done |
| M-0189 | Add worktree.dir config knob defaulting to .claude/worktrees | done |
| M-0190 | Default the start rituals to in-repo worktree placement | done |

## E-0047 — Harden and ship the aiwf-aware Claude Code statusline (done)

| Milestone | Title | Status |
|---|---|---|
| M-0191 | Behavioral test harness for the statusline + stale-CI-after-push fix | done |
| M-0192 | Statusline shows in-flight epics on every branch | done |
| M-0193 | Statusline health indicator from a cached check-findings signal | done |
| M-0194 | Ship the statusline via aiwf init/update with portability fixes | cancelled |

## E-0048 — Skill & ritual content integrity (with drift chokepoints) (done)

### Goal

Every shipped skill and ritual body is accurate, consistent, and self-contained,
and mechanical chokepoints prevent future drift. The reader of any `aiwf-*` verb
skill or `aiwfx-*` / `wf-*` ritual gets correct guidance, and a future edit that
reintroduces drift is caught at the earliest chokepoint tier its class allows —
pre-push where the check can live there, CI at the latest.

| Milestone | Title | Status |
|---|---|---|
| M-0195 | Strict skill-body id-reference discipline, check, and full sweep | done |
| M-0196 | Skill-edit structural-test backstop policy | done |
| M-0197 | Document aiwf-check finding codes + documented-superset chokepoint | done |
| M-0198 | Verb-skill factual corrections (status set, kind, paths, links) | done |
| M-0199 | wf-tdd-cycle/wf-review-code honesty and wf-doc-lint reframe | done |
| M-0200 | Skill descriptions, whiteboard, and prose polish | done |
| M-0201 | Planning-ritual body-fill via edit-body and Next-step routing | done |
| M-0202 | Fix devcontainer onboarding banner (retired plugin install) | done |
| M-0210 | Drift chokepoint for the trailered-commit block in wrap rituals | done |
| M-0211 | Migrate consumer operating guidance from CLAUDE.md to the shippable source | done |

## E-0049 — Ritual lifecycle model: gate discipline and commit/TDD model (cancelled)

### Goal

The ritual lifecycle's commit/TDD model is coherent and matches CLAUDE.md:
milestone implementation commits plus TDD phase evidence are honest, and the
start/wrap rituals are internally consistent. The gate model itself was delivered
by the foundation epic E-0050 (now done), which this epic builds on.

| Milestone | Title | Status |
|---|---|---|
| M-0203 | Generalize the declared-sequence gate; fix wrap/release drift | cancelled |
| M-0204 | Model 1: commit implementation per AC; live phase promotes | cancelled |
| M-0205 | Milestone review framing and wrap-milestone trailer test | cancelled |
| M-0206 | Start-ritual fixes: branch-not-found and sovereign-acts-on-trunk | cancelled |
| M-0207 | aiwf.yaml declared-sequence-wraps opt-in knob | cancelled |
| M-0230 | Make roadmap regeneration zero-friction in the wrap rituals | cancelled |

## E-0050 — Gate-discipline foundation: generalize the declared-sequence gate (done)

### Goal

Generalize the wf-patch declared-sequence gate into a general capability for any
sequence of local, reversible mutations — one gate that enumerates every action
verbatim, binds approval to exactly that list (subset approval allowed), and
aborts + re-gates on any deviation — and fix the wrap and release rituals that
currently violate the gate discipline CLAUDE.md *claims* they already follow.

| Milestone | Title | Status |
|---|---|---|
| M-0209 | Generalize the declared-sequence gate; fix wrap/release drift | done |

## E-0051 — Context and compute economy across the ritual lifecycle (proposed)

### Goal

The ritual lifecycle uses the right context and the right compute at each
boundary: read-heavy, judgment-light steps run on a cheap model, and the
session topology keeps build context where it accumulates while OS-enforcing
worktree isolation. The operator gets the doctrine *emitted by the rituals at the
right moment*, not buried in a doc.

_No milestones yet._

## E-0052 — Broaden the id allocator's cross-branch view to cut collisions (done)

### Goal

Reduce the id-collision *window* mechanically by widening the trunk-aware
allocator from `{working-tree ids + one trunk ref}` to `{working-tree ids + all
local refs + best-effort-fetched trunk}`, so the dominant collision classes are
caught at allocation time instead of surfacing at push via `aiwf reallocate`.
The stable-id-from-creation model is preserved entirely — no inbox, no mint, no
slug phase. `aiwf reallocate` stays the backstop for the irreducible
cross-machine concurrent race.

| Milestone | Title | Status |
|---|---|---|
| M-0212 | Union all local refs into the allocator's cross-branch id view | done |
| M-0213 | Opt-in best-effort fetch before id allocation | done |
| M-0214 | Broaden allocator and --fetch to all remote-tracking refs | done |

## E-0053 — Make aiwf check and the policies test suite fast (done)

### Goal

Cut the wall-time of `aiwf check` — the pre-push and CI chokepoint — and of
the `internal/policies` test suite, by eliminating redundant git-subprocess
fan-out and a redundant pre-push lint run, without weakening any guarantee
or moving any rule from pre-push to CI.

On the kernel's own repo (659 entities) `aiwf check` measures ~85s, almost
entirely git-subprocess overhead: a single run spawns ~895 git processes,
683 of them `git merge-base --is-ancestor` issued one-per-reflog-pair by the
orphaned-AI-commit walk. The check runs ~6 independent git-history passes
that never share a loaded history.

| Milestone | Title | Status |
|---|---|---|
| M-0215 | Profile aiwf check and the policies suite to a per-rule wall-time baseline | done |
| M-0216 | Shared per-check git-history context; collapse per-entity subprocess fan-out | done |
| M-0217 | Skip redundant pre-push golangci-lint via a last-green-lint marker | cancelled |
| M-0218 | Drive the internal/policies test suite below its ~9s floor | cancelled |
| M-0219 | Wire git commit-graph maintenance into aiwf init and update | cancelled |
| M-0220 | Re-fixture heavy real-tree check integration tests to synthetic fixtures | done |

## E-0054 — Fast read paths: single-pass render walk and read-verb grep guard (done)

### Goal

Make aiwf's read verbs — `render`, `history`, and `show` — fast in the devcontainer,
where they are subprocess-*wait* bound (the Docker/linuxkit `fork`/`exec` tax), by
cutting per-invocation git-subprocess count from O(entities × commits) to a single
shared history pass, and by removing a repo-wide authorize grep that `aiwf history`
and `aiwf show` run unconditionally.

Measured on the kernel tree (this devcontainer):

- `aiwf render --format=html` takes **~28 minutes** because it issues **~1,860+
  per-entity `git log` walks** (~3,500 subprocesses, estimated) across **two** walk
  families: per-entity history (`resolver.history` → `history.ReadHistory`, one walk
  per epic/milestone/AC/other-entity) and per-milestone provenance/scopes
  (`show.LoadEntityScopeViews`, which re-walks the milestone's history *uncached*
  and runs a full `readAllAuthorizeOpeners` grep — once **per milestone**). A
  throwaway single-pass spike rendered **byte-identical** output in **~12.8s**
  (~130×).
- `aiwf history <id>` (default text) is **~2×** slower than it needs to be: it runs
  `BuildScopeEntityMap` — a repo-wide `git log --grep 'aiwf-verb: authorize'` — on
  **every** invocation, even though the entity has no authorization and the whole
  tree holds only a handful of authorize openers (4). On a milestone with zero scopes
  the text path measured ~2.2s vs ~1.2s for `--format=json` (which skips that grep):
  ~1.0s of pure waste per call.
- `aiwf show <id>` pays the **identical** grep by a different route
  (`LoadEntityScopeViews` → `readAllAuthorizeOpeners`, run before it knows the entity
  has any scope) and measured ~3.4s. Same waste, a *second* implementation.

This epic adds a derived *read strategy*, not a second source of truth. `git log` +
trailers stays canonical (per `design-decisions.md`); the design and per-lever
worktree/merge safety analysis live in
[`docs/pocv3/design/performance.md`](../../../docs/pocv3/design/performance.md).

| Milestone | Title | Status |
|---|---|---|
| M-0221 | Single unified history walk for render | done |
| M-0222 | Path-scoped single-entity history with bloom-filter maintenance | cancelled |
| M-0223 | Guard the unconditional authorize-opener grep in the read verbs | done |

## E-0055 — Health as install status: producer health files + statusline stoplight (done)

### Goal

Give operators visibility of `aiwf` installation and configuration warnings and errors
in the Claude Code statusline: an always-visible stoplight (gray / green / yellow / red)
fed by per-producer `.claude/health.*.json` files. `aiwf` writes its own health file
from `aiwf doctor`'s warnings and errors; the statusline reads and unions the health
files and shows the maximum severity — never running a check on the render path.

| Milestone | Title | Status |
|---|---|---|
| M-0224 | aiwf health: doctor writes health.aiwf.json + statusline stoplight | done |
| M-0225 | aiwf health producer: doctor --write-health + lifecycle refresh | cancelled |
| M-0226 | Four-state statusline health stoplight from producer files | cancelled |

## E-0056 — Extend the id chokepoint across shipped surfaces; strip provenance prose (done)

### Goal

Every surface aiwf materializes into a consumer's `.claude/` — verb and ritual
skills, role-agent cards, entity templates, the always-on guidance fragment, and
the statusline — reads as imperative, consumer-scoped instruction: no
aiwf-internal entity ids, no development history, no provenance tags, no
rationale or war-stories, and no dead references to artifacts that do not ship.
The id chokepoint covers every shipped surface, so the leak class cannot recur.

| Milestone | Title | Status |
|---|---|---|
| M-0227 | Extend the id chokepoint to all shipped surfaces; clean id leaks | done |
| M-0228 | Strip shipped-prose history/rationale; broaden the authoring principle | done |
| M-0229 | Drop dead doc-links; encode reference discipline in record-decision | done |

## E-0057 — Consumer-discoverable aiwf.yaml schema via a generated example.yaml (done)

### Goal

Give every aiwf consumer a discoverable, always-fresh reference for the whole
`aiwf.yaml` schema — inside their own repo, without reading aiwf's source. A
config surface a user cannot discover is a feature that effectively does not
exist; today the entire schema is documented only in Go struct doc comments.

| Milestone | Title | Status |
|---|---|---|
| M-0231 | Struct-derived aiwf.yaml schema model and commented-YAML generator | done |
| M-0232 | Wire generator into init/update: fresh-repo scaffold and example.yaml | done |

## E-0058 — Immutable per-commit-sha cache for aiwf check's full-history revwalks (cancelled)

### Goal

Make `aiwf check`'s git-history-dependent rules cost proportional to how much
changed since the last check, not to total repository history — without
weakening the correctness guarantee those rules currently provide.

_No milestones yet._

## E-0059 — Atomic ritual materialization at worktree creation (done)

### Goal

Make a freshly-cut git worktree carry the same materialized `.claude/skills/`,
`.claude/agents/`, `.claude/templates/`, and `.claude/aiwf-guidance.md` as the main
checkout — atomically at creation time via aiwf's own tooling, with a session-level
backstop that catches any worktree created outside that path — so ritual discipline
(TDD, vacuity, rethink, gate rules) is never silently absent just because work happens
to be isolated in a worktree.

| Milestone | Title | Status |
|---|---|---|
| M-0233 | aiwf worktree add verb: atomic creation with ritual materialization | done |
| M-0234 | Rewire aiwf rituals and CLAUDE.md to use aiwf worktree add | done |
| M-0235 | Generalized hook registry: aiwf.yaml-declared, persisted consent | done |
| M-0236 | Ship the worktree-materialization-check SessionStart hook | done |

## E-0060 — Resolve cross-branch entity references at check and read time (done)

### Goal

Let a branch, worktree, or session validly reference an entity minted on a
different local branch or worktree — in `aiwf check` and in `aiwf show`/`aiwf
list` — without waiting for a merge and without copying the entity anywhere.

| Milestone | Title | Status |
|---|---|---|
| M-0259 | Add cross-branch-pending tier and collision detection to reference checks | done |
| M-0260 | Resolve and render cross-branch entity content in show and list | done |

## E-0061 — Diagnostic logging and correlation (done)

### Goal

Give aiwf a retrace-ready diagnostic surface: opt-in structured logging plus
a correlation id that ties one invocation's JSON envelope to its own log
lines, so "why did this verb do that on someone else's repo?" has a real
answer instead of whatever the operator happened to capture from stderr.

| Milestone | Title | Status |
|---|---|---|
| M-0237 | Logger core: internal/logger package and concurrent-append safety | done |
| M-0238 | Migrate bare-stderr call sites; forbidigo chokepoint | done |
| M-0239 | Correlation id wiring; ratify ADR-0017 | done |

## E-0062 — Correctness stress harness (done)

### Goal

An on-demand, real-git/real-process stress harness that exercises aiwf's
worktree, concurrency, and verb-sequencing correctness beyond what today's
example-based unit and integration tests cover, converting any violation it
finds into a reproducible gap.

| Milestone | Title | Status |
|---|---|---|
| M-0240 | Harness skeleton: driver, scenario interface, streaming report | done |
| M-0241 | Property sequences and multi-worktree contention scenarios | done |
| M-0242 | Fault injection via external observation | done |
| M-0243 | Named scenarios from G-0212 and G-0269 | done |
| M-0244 | Concurrent-writer test at scale; triage process | done |
| M-0249 | Scenario registry: wire cmd/stresstest run to the real catalog | done |
| M-0250 | Register the verb-sequence walker; extend it to move/archive/rename/retitle | done |

## E-0063 — Rewrite entity path-links on move to keep them durable (done)

### Goal

Make markdown links between entity files survive the file-moving verbs
(`archive`, `rename`, `retitle`, `reallocate`), so an author can cite an entity
with a clickable path-link and trust it stays correct — instead of watching it
rot silently the next time its target moves.

| Milestone | Title | Status |
|---|---|---|
| M-0245 | Shared link-destination rewrite primitive | done |
| M-0246 | Wire archive to rewrite link destinations on sweep | done |
| M-0247 | Wire rename and retitle to rewrite link destinations | done |
| M-0248 | Unify reallocate onto the shared rewrite primitive | done |
| M-0251 | Handle #fragment / ?query suffixes in link-destination rewrite | done |

## E-0064 — Backfill test coverage for untested CLI verb error-handling branches (done)

### Goal

Every currently-flagged untested CLI-verb error-handling branch gets either a
real regression test or a documented `//coverage:ignore`, so
`make coverage-gate` reports zero findings against these sites and the
diff-scoped coverage gate stops firing on incidental future touches to this
code.

| Milestone | Title | Status |
|---|---|---|
| M-0252 | Shared CLI-verb failure fixtures and non-CLI infra coverage backfill | done |
| M-0253 | Entity-lifecycle verb coverage backfill | done |
| M-0254 | Contract subsystem coverage backfill | done |
| M-0255 | Diagnostic and introspection verb coverage backfill | done |
| M-0256 | Bulk-input verb coverage backfill | done |

## E-0065 — Harden the stress catalog's correctness oracle (done)

### Goal

Close the blind spots identified in G-0410: broaden the stress catalog's
"`aiwf check` must stay clean" oracle beyond `verb-sequence`, and add a
concurrent-race mode that exercises promote/cancel/AC operations against
shared entity state, so a missing domain-specific verb-time guard can't ship
silently.

| Milestone | Title | Status |
|---|---|---|
| M-0257 | Broaden the check-clean oracle across ten stress scenarios | done |
| M-0258 | Race concurrent promote/cancel/AC operations against shared entity state | done |

## E-0066 — Add a priority field to gaps and decisions for a filterable backlog (done)

### Goal

Give aiwf a kernel-supported `priority` field on gaps and decisions, so the backlog can be filtered and surfaced by importance instead of read end-to-end in id order. It replaces the ad-hoc inline `Severity:` prose that nothing can query with structured state that `aiwf check`, `aiwf list`, the JSON envelope, and the HTML render all understand.

| Milestone | Title | Status |
|---|---|---|
| M-0261 | Add the priority field, its validation, and drift chokepoints | done |
| M-0262 | Add the priority write surface: set-priority verb and add --priority | done |
| M-0263 | Add the priority read surface: list/status filter, envelope, show | done |
| M-0264 | Render a priority badge in the HTML site | done |

## E-0067 — Harden the cross-branch read path: end the list/check hang, fix show --area (done)

### Goal

Filtered `aiwf list` (~10s) and `aiwf check` (~15–20s) are slow at this repository's
scale because the cross-branch scan runs collision blob-stats over every entity on
every ref, then discards nearly all of it. Make that scan lazy — collision detection
only for ids absent from the local working tree — so read verbs stay fast as the tree
and branch count grow, and fix the one cross-branch correctness bug that lives in the
same code.

| Milestone | Title | Status |
|---|---|---|
| M-0265 | Make the cross-branch collision scan lazy via a single trunk helper | done |
| M-0266 | Honor a cross-branch id's real area in show --area | done |

## E-0068 — Mechanical AC/milestone-completeness guards (done)

### Goal

Close three places where the kernel currently depends on operator vigilance instead of a mechanical chokepoint for AC/milestone completeness discipline — a milestone starting with an empty AC body, a milestone starting or finishing with zero ACs at all, and an over-strict `tdd_phase` requirement — so the AC-evidence discipline holds without relying on a human or an LLM remembering the rules.

| Milestone | Title | Status |
|---|---|---|
| M-0267 | Relax acs-shape/tdd-phase to allow absent phase until AC met | done |
| M-0268 | AC-completeness guards: zero-AC and empty-body promote refusals | done |

## E-0069 — Close the verb-layer call-graph audit findings (done)

### Goal

Close the verified findings from the verb-layer call-graph audit: fix the three
correctness bugs, collapse the hand-duplicated helpers onto the shared seams the
codebase already owns, extend `cliutil.FinishVerb`'s contract to cover its three
bypassers, and give the read-only verbs a neutral shared library — so a change to
a shared contract (commit-outcome envelope, git-plumbing helper, hook marker)
reaches every verb from one place instead of drifting per hand-rolled copy.

| Milestone | Title | Status |
|---|---|---|
| M-0269 | Fix import id allocation, show error swallowing, and scope-event sort order | done |
| M-0270 | Collapse duplicated verb-layer helpers onto their shared seams | done |
| M-0271 | Extend FinishVerb with dry-run and multi-Plan; migrate its three bypassers | done |
| M-0272 | Extract the read-side helpers into a neutral entityview package | done |
| M-0273 | Converge contract-mutating verbs on one shared diff-based validation gate | done |

