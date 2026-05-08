# Skills / policy bundles the FlowTime stack would need

> Working from the corpus and the technology stack, what *skill bundles* (or
> *policy bundles*) would naturally emerge if the framework were to ship a
> "starter pack" for a project shaped like FlowTime? Each bundle ties back to
> design-space §3 categories where applicable.

The exercise is forward-looking: today FlowTime has CLAUDE.md prose + .editorconfig
+ CI + a few skills. If we were to *factor* what's there into reusable bundles, what
bundles fall out?

---

## A. Stack-shape bundles (chosen by stack detection at install time)

These would be selected by what the repo *contains* — `*.csproj` ⇒ dotnet bundle,
`Cargo.toml` ⇒ rust bundle, `package.json` + `tsconfig.json` ⇒ typescript bundle.
The dead-code-audit skill's bootstrap path already does exactly this.

### Bundle 1 — `dotnet-stack`

Maps to design-space §3 categories: *style/convention*, *verification*, *quality
threshold*.

| Policy | Default rung | Substrate |
|---|---|---|
| Naming: private fields camelCase, no underscore | 2 | `.editorconfig` (Roslyn) |
| Naming: local variables camelCase | 2 | `.editorconfig` |
| File-scoped namespaces | 1 (suggestion) | `.editorconfig` |
| Roslynator severity tuning (suppress noise; surface high-signal codes) | 2 | `.editorconfig` (with prose explaining each tuned code) |
| .NET 9 / C# 13, implicit usings, nullable enabled | 3 | `Directory.Build.props` |
| Test discovery: `WebApplicationFactory<Program>` for API tests; prefer real deps | 1 | prose + a checked example |
| Use invariant culture for parsing/formatting | 1 → 2 | grep for `CultureInfo.CurrentCulture`-misuse |
| Per-test-project hang timeout in CI | 2 | CI yaml |
| Roslynator dead-code recipe (the `dead-code-audit` recipe) | 2 (recipe-driven) | `.claude/skills/dead-code-audit/recipes/dead-code-dotnet.md` |
| Dependency: keep CLI ↔ API behavior aligned (update `.http` examples) | 1 | prose |

**Why a bundle:** every one of these ports to any other .NET project. The first
bundle's policies are *exactly* the kind of "shipped to the consumer at `aiwf init`"
content the framework's policy-as-portable subsystem should produce.

### Bundle 2 — `typescript-svelte-stack`

| Policy | Default rung | Substrate |
|---|---|---|
| pnpm workspaces convention | 2 | `pnpm-workspace.yaml` |
| Vitest covers pure logic; Playwright covers integration | 1 | prose + scaffolded test dirs |
| Svelte 5 runes (`$state`, `$derived`, `$effect`, `$props`) are compiler-recognized | 1 | prose hint for tools |
| SvelteKit route discovery via filename convention | 1 | prose hint |
| shadcn-svelte components imported via `$lib/components/ui/<name>` | 1 | prose hint |
| dead-code recipe: knip via `pnpm dlx knip --reporter json` | 2 | recipe |
| `unused exports` / `unused files` / `unused dependencies` are high-signal | 1 | recipe body |
| `unused devDependencies` review carefully (false positives common) | 1 | recipe body |

### Bundle 3 — `rust-stack`

| Policy | Default rung | Substrate |
|---|---|---|
| `cargo udeps` (nightly) for unused crate dependencies *or* `cargo machete` (stable) | 2 | recipe |
| `clippy` `dead_code` lint enabled | 2 | clippy.toml |
| `rustfmt` enforced | 2 | rustfmt.toml |

### Bundle 4 — `python-tooling-stack` (e.g., for `tools/mcp-server/`)

| Policy | Default rung | Substrate |
|---|---|---|
| `vulture` for dead-code (low-noise) *or* `pyflakes`/`deadcode` | 2 | recipe |
| Use Astral `uv` for env management | 2 | `pyproject.toml` |
| Type hints + `mypy` / `pyright` | 2 | tooling config |

### Bundle 5 — `ui-end-to-end` (cross-stack)

Maps to design-space §3: *verification*, *quality threshold*.

| Policy | Default rung | Substrate |
|---|---|---|
| Every UI-touching milestone includes Playwright tests in a real browser | 1 (hard rule) | prose |
| Specs gracefully skip if API or dev server isn't running (health probe) | 1 → 2 (shared helper) | helper code + prose |
| Cover critical paths: page load, one user interaction, reset/error path, key metric correctness | 1 (4-bullet prose) | prose |
| Playwright config: per-spec `baseURL` override allowed | 2 | playwright.config.ts |
| Vitest covers pure helpers; Playwright covers integration | 1 | prose |

---

## B. Ritual bundles (chosen by aiwf-mode at install time)

### Bundle 6 — `aiwf-rituals`

Already shipping as the `aiwfx-extensions` + `wf-rituals` plugins. What I'd
formalize as a *policy bundle* (not just an agent bundle):

| Policy | Default rung | Substrate |
|---|---|---|
| Branch discipline: do not commit milestone work directly to `main` | 1 → 2 (pre-commit hook) | hook + prose |
| Conventional Commits format (`feat`, `fix`, `chore`, `docs`, `test`, `refactor`); no icons; subject + short bullet body capturing milestone + key work/tests | 1 → 2 (commitlint) | commitlint config + prose |
| Never commit/push without explicit human approval | 1 (LLM honor) | prose, escalating to a hook on milestone branches |
| TDD by default (red → green → refactor) for logic/API/data code | 1 | prose |
| TDD per-AC phase tracking (`tdd_phase: red\|green\|refactor`) | 2 | aiwf field + verb |
| Every red-tagged AC must reach green before milestone wraps | 2 | aiwf check |
| Branch coverage required before declaring done | 1 (manual audit) | prose; coverlet for rung-2 elevation |
| Wrap-milestone invokes dead-code-audit (non-blocking) | 2 | wrap ritual chain |
| Dead-code-audit findings flow to `work/gaps.md` by hand if real | 1 | prose |

### Bundle 7 — `aiwf-provenance`

| Policy | Default rung | Substrate |
|---|---|---|
| Human verbs need no extra flags | 2 | aiwf kernel |
| Non-human actors must pass `--principal human/<id>` and operate inside an active scope | 2 | aiwf kernel |
| `aiwf authorize` is human-only | 2 | aiwf kernel |
| Trailers `aiwf-principal:`, `aiwf-on-behalf-of:`, `aiwf-authorized-by:` added automatically | 2 | aiwf kernel |
| `aiwf promote/cancel --audit-only --reason "..."` for after-the-fact ratification | 2 | aiwf verb |

### Bundle 8 — `aiwf-truth-discipline`

| Policy | Default rung | Substrate |
|---|---|---|
| Truth precedence: code+passing tests > decisions/ADRs > epic specs > arch docs > history | 1 | prose |
| Truth classes: `docs/` (current) | `work/epics/` (decided-next) | `docs/archive/`, `docs/releases/` (historical) | `docs/notes/` (exploration only) | 1 | prose |
| If sources disagree, surface the conflict and ask — do not auto-resolve | 1 | prose |
| Don't restate canonical contracts in many places from memory; point to the owner | 1 | prose |
| Don't let one file act as both current reference and historical archive | 1 | prose |
| Don't describe a target contract in present tense unless it is live | 1 → 2 (LLM-as-linter) | prose |
| Don't keep "temporary" compatibility shims without explicit deletion criteria | 1 | prose |
| Do not treat aspirational docs as implementation authority | 1 | prose |

---

## C. Domain bundles (chosen by repo subject matter)

These don't transfer to other repos *as-is*, but the *shape* does — the bundle
has a pattern others could specialize.

### Bundle 9 — `engine-correctness` (FlowTime-specific subject)

Pattern: pin observable engine output against approved canon; treat soft-signal
warnings as informational until promoted to blocking.

| Policy | Default rung | Substrate |
|---|---|---|
| Schema is canonical, single source of truth | 3 | schema validator |
| Forward-only migration when boundary changes; no compatibility readers | 1 | prose |
| Warnings (`val-warn`, `run-warn`) gated against per-template baselines | 2 | dictionary in test |
| Promote baseline canary → golden-output canary when evidence demands | 1 (process policy) | prose + epic E-25 |
| Pinned fixtures stored in reviewable format (JSON with stable key order) | 1 | prose |
| Tolerance values explicit and conservative; revisit per-template if needed | 1 | prose |
| Sanctioned `--regenerate` workflow for sanctioned engine changes | 2 | test mode |
| Counterexample-shaped findings (which series at which bin moved by how much) | 1 → 2 (test framework) | helpers |

The pattern transfers; the FlowTime instance is one of N possible.

### Bundle 10 — `numeric-safety` (FlowTime's NaN policy as a transferable pattern)

Pattern: name the IEEE 754 hazards your domain meets; assign each site to a tier;
maintain the per-site enforcement table; require the policy doc be updated when
adding a new site.

The shape would transfer to any system doing extensive floating-point arithmetic
(simulators, ML frameworks, financial calc, scientific computing).

### Bundle 11 — `service-port-topology` (devcontainer-shaped)

Pattern: name every port the project uses; explain *why* each must not be killed
naively (port-forwarder lives there); ship safe-kill helpers; document recovery
procedures.

| Policy | Default rung | Substrate |
|---|---|---|
| Document default ports with rationale | 1 | prose |
| Ship safe-kill helpers (`kill-port-NNNN`) per service | 2 | task |
| Filter by process name when killing on a port | 1 → 2 (task wraps it) | task |
| SIGTERM first, then SIGKILL only if still alive; never `kill -9` first | 1 | prose |
| Verify processes before killing | 1 | prose |

---

## D. Meta-bundles

### Bundle 12 — `meta-policies`

Rules about how policies themselves are written, amended, ratified, retired. The
"rest" bucket is mostly populated by these.

| Policy | Default rung | Substrate |
|---|---|---|
| Soft-signal vs blocking is a category, not a severity slider | 1 | prose |
| Policy promotion (soft-signal → blocking) requires evidence (incident, metric, or canary) | 1 | prose |
| Anti-patterns are policies-of-prohibition; each names a kind of mistake with a positive corrective | 1 | a structured anti-patterns list |
| Hand-editing generated artifacts is forbidden | 1 (per-artifact prose) | repeated |
| Future references must cite an open issue | 1 | prose |
| Rule relaxations land in the same PR as the rule change (no silent violations) | 1 | prose |
| Pre-PR audit walks the diff against the governing CLAUDE.md and reports conformance in the PR body | 1 → 2 (PR template) | template |
| YAGNI heuristic: build for the second case; abstract on the third | 0 | prose only |
| When a tool fails, the tool failure is a finding (not silent pass) | 1 | prose |

These are the most under-supported in current tooling. The framework absorbing
even half of them would be a real consolidation.

---

## E. Bundle composition — how a real repo would assemble its policy set

For FlowTime, the assembled set would be approximately:

| Layer | Bundles included | Source |
|---|---|---|
| Kernel | `aiwf-provenance`, `aiwf-truth-discipline`, `meta-policies` | framework |
| Workflow | `aiwf-rituals` | framework (rituals plugin) |
| Stack | `dotnet-stack`, `typescript-svelte-stack`, `rust-stack`, `ui-end-to-end` | framework (stack-detected) |
| Domain | `engine-correctness`, `numeric-safety`, `service-port-topology` | repo-private (or shared FlowTime org) |

The kernel + workflow + stack layers are the framework's contribution; the domain
layer is the consumer's contribution.

This decomposition matches the design-space §12 "registry + per-target enforcement"
shape: kernel + workflow + stack come from a central registry; domain stays local.

---

## F. What the corpus reveals about the framework's job

Walking the corpus this way clarifies what the framework's actual job is:

1. **Provide a vocabulary** so consumers stop reinventing "soft-signal vs blocking,"
   "rule with rationale," "supersession with provenance," etc.
2. **Provide a substrate-pointer model** so consumers can wire `.editorconfig`,
   `commitlint`, `clippy.toml`, `Roslynator`, `knip`, `vulture`, `Playwright config`,
   `EARS prose`, etc. as enforcement surfaces under one policy entity.
3. **Provide standard bundles** (the §A-§D list) so a `aiwf init` for a polyglot
   .NET+Svelte+Rust repo lands ~80% of the policies with one command.
4. **Provide ratification + supersession + waiver verbs** so the lifecycle is owned
   by the kernel, not improvised per repo.
5. **Provide auditing verbs** (the §G "doc sweep" shape: walk the repo against the
   policy set, classify, surface conflicts) so the policy-application work scales.

Conspicuously absent from this list: a CUE evaluator, a Rego runtime, a TLA+
checker. None of them are *required* by the FlowTime corpus. The corpus's
enforcement is dominated by lints, schemas, integration tests, CI jobs, shell
scripts, and prose. The substrate work (CUE / Rego / TLA+) earns its keep when a
*specific* policy class needs it; it is not a precondition for the framework to
ship value.

The strongest "first version" inference from this exercise: **ship the kernel +
workflow + stack bundles in §A-§B, with prose-form policies pointing at
substrate-appropriate runners, and let the domain bundles (§C) emerge from the
first three or four real consumers.**
