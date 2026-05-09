# AIWF workflow diagnostic — current state analysis

**Purpose.** This document is a fact-grounded analysis of the current aiwf workflow as expressed in its skills and templates. It identifies what's working structurally, where the architecture's stated intent diverges from the behavior it actually produces, and how those divergences interact with known LLM-coding-agent failure modes.

**What this is not.** It is not a proposal. Recommendations, examples of alternative shapes, FSM redesigns, and migration paths are out of scope and live in the companion *AIWF architecture proposal* document.

**Sources of fact.** This analysis is grounded in the actual content of seven artefacts: `templates/milestone-spec.md`, `aiwfx-plan-epic`, `aiwfx-plan-milestones`, `aiwfx-start-milestone`, `wf-tdd-cycle`, `aiwfx-wrap-milestone`, and `aiwfx-wrap-epic`. Statements about the `aiwf` kernel itself reflect what those skills explicitly invoke (verbs, FSM behavior, finding codes); kernel internals not exposed through the skills are out of scope.

---

## 1. The current architecture in brief

The system is built on a deliberate two-layer separation:

- **Kernel (`aiwf`)** — strict invariant enforcement. The kernel owns id allocation, position-stable AC slots, status FSMs, an `aiwf-verb` audit trail in commit trailers, a closed-tree shape under `work/`, and a fixed set of audit codes (`acs-shape`, `acs-tdd-audit`, `acs-body-coherence`, `milestone-done-incomplete-acs`, `unexpected-tree-file`). Phase names and status values are enumerables; the kernel's FSM refuses transitions outside the legal set.

- **Skills** — the LLM-facing layer that guides agents to produce work satisfying the kernel's invariants. Skills carry the prose, heuristics, and sequencing the kernel intentionally doesn't.

The entity set:

| Entity | ID | Parent | Purpose |
|---|---|---|---|
| Epic | `E-NN` | none | Coordination unit |
| Milestone | `M-NNN` | epic | Independently shippable work unit |
| Acceptance Criterion | `M-NNN/AC-N` | milestone | Observable behavior to verify (compound id) |
| ADR | `ADR-NNNN` | none | Durable architectural decision |
| Decision | `D-NNN` | none | Project-scoped decision (more local than ADR) |
| Gap | `G-NNN` | discovered-in milestone | Deferred work that survives milestone close |

The skills:

| Skill | Phase | Role |
|---|---|---|
| `aiwfx-plan-epic` | Plan | Scopes a new epic, allocates `E-NN`, fills the epic spec |
| `aiwfx-plan-milestones` | Plan | Decomposes an epic into milestones, just-in-time |
| `aiwfx-start-milestone` | Execute | Promotes to in_progress, runs preflight, dispatches per-AC cycles |
| `wf-tdd-cycle` | Execute | Per-AC red/green/refactor with branch-coverage audit |
| `aiwfx-wrap-milestone` | Close | Verifies completion, finalizes spec, promotes to done |
| `aiwfx-wrap-epic` | Close | Closes the epic, harvests ADRs, merges to mainline |

The TDD axis on a milestone is a single field: `tdd: required | advisory | none`. The kernel's `acs-tdd-audit` enforces `met requires phase: done` as an error under `required` and a warning under `advisory`.

---

## 2. What works structurally

Worth naming explicitly, because these are the parts of the architecture that don't need to change for the system to support a wider range of disciplines.

**Just-in-time spec authoring.** `aiwfx-plan-milestones` deliberately does not front-load AC detail; full bodies fill at `aiwfx-start-milestone`. The skill explicitly calls front-loading detail an anti-pattern. This is a defense against spec rot — an inherent failure mode of "spec-driven AI" tooling that pre-fills 16 acceptance criteria for a 1-day task.

**Observable-behavior bar on ACs.** The milestone template explicitly rejects `"X is tested"`, `"refactor complete"`, and `"feature implemented"` as ACs. The bar is "observable behavior, not implementation detail." This is the right bar; the tensions identified below come from how the bar interacts with other parts of the system, not from the bar itself.

**Separation between epic Success criteria and tests.** `aiwfx-plan-epic` is explicit: Success criteria are observable outcomes "not tests." Most spec systems collapse these; aiwf does not.

**Append-only Work log.** Combined with the kernel's `aiwf history M-NNN/AC-<N>` trailer-derived timeline, this gives a reconstructible audit trail of mid-flight context. The agent cannot retroactively rewrite the timeline.

**Pasted validation output at wrap.** `aiwfx-wrap-milestone` requires test-suite results pasted into the milestone spec's `## Validation` section, not summarized. This is exactly the defense the LLM-research literature recommends against the "Verification Gap" — agents reporting success before the verification suite confirms it.

**Structured deferrals via gap entities.** Every deferred item gets a `G-NNN`. `aiwfx-wrap-milestone` blocks if a deferral has no gap reference. Deferrals don't evaporate.

**Reference-phrasing for list-derived counts.** `aiwfx-plan-epic` and `aiwfx-wrap-epic` instruct authors to write "every ADR listed in *ADRs ratified*" instead of "all 4 ADRs." Counts drift; references don't.

**Wrap-as-closure separated from release-as-shipping.** `aiwfx-wrap-epic` explicitly defines wrap as the planning unit's closure, not a tag-and-publish. Releases, changelogs, and version tags belong to a separate `aiwfx-release` skill.

**Forward-flowing DAG with explicit `depends_on`.** Sequence is reviewable and validatable; cycles are kernel-rejected.

**Commit and push human gates.** `aiwfx-wrap-milestone` (steps 8, 10) and `aiwfx-wrap-epic` (steps 7, 9, 10) require explicit human approval at each commit and push. Branch deletions on origin require approval per branch or batch.

**Coverage notes section** in the milestone template provides a documented escape hatch for genuinely unreachable branches. The shape of the escape hatch is right; its scope (covered in §3.3) is the limitation.

**`aiwfx-start-milestone` preflight.** Step 1 does real work: confirms ACs are concrete, runs build green, runs tests green, refuses to start on a broken baseline. The "stop and ask the user to refine vague ACs" beat catches a meaningful class of problems before commitment.

**Per-AC verbs preserve position-stability.** `aiwf add ac` allocates a position-stable id and scaffolds a body heading; the kernel's `acs-body-coherence` check surfaces drift between frontmatter and body. Hand-editing `acs[]` is an anti-pattern flagged in `aiwfx-start-milestone`.

---

## 3. Structural tensions in the current design

The system is **labeled outside-in / behavioral** in its language and **structured Detroit-classical** in its FSM and audit rules. Each subsection below identifies a specific divergence.

### 3.1. AC and TDD cycle are coupled in the FSM

The kernel encodes a 1:1 relationship between an acceptance criterion and a TDD cycle. Evidence:

- The milestone template's frontmatter seeds `tdd_phase: red` per AC when `tdd: required`.
- `wf-tdd-cycle` advances phase per AC: `aiwf promote M-NNN/AC-<N> --phase red | green | refactor | done`.
- The kernel's `acs-tdd-audit` checks that every AC at `status: met` has reached `tdd_phase: done`.
- `aiwfx-wrap-milestone` step 1 says: "confirm each AC has at least one test that exercises it green."

The structural consequence: each AC is, by FSM definition, the unit of TDD progression. A single AC verified by an acceptance test plus targeted unit tests cannot be expressed as such — it must be either one AC with one cycle (forcing the multi-test reality into a single phase track) or multiple ACs (fragmenting one observable behavior into FSM-bookkeeping units). One acceptance test that verifies multiple ACs has no expression at all.

This couples discipline choice to FSM shape: classical Detroit-style triangulation maps cleanly (one AC, one cycle, one unit test); outside-in BDD does not (one acceptance scenario verifies several behaviors); property-based testing does not (a property covers many implicit cases that don't shard into ACs); contract-first does not (the contract is a single artifact verifying the public surface).

### 3.2. The TDD axis expresses enforcement, not discipline

The `tdd:` frontmatter has one axis with three values: `required | advisory | none`. This answers "how strict is the audit" but does not answer "which model of TDD."

Consequences:

- An LLM agent reading `tdd: required` defaults to its prior, which from training-data exposure is roughly Kent-Beck classical. There is no place in the spec to declare "this milestone is outside-in" or "this is property-based" and have the cycle skill behave accordingly.
- `tdd: advisory` is in the enum but defined operationally only as "audit severity downgrades from error to warning." The `aiwfx-plan-milestones` skill does not mention `tdd:` at all in its workflow — the TDD axis is invisible during decomposition. Discipline (in any sense) is invisible at every step.
- `aiwfx-start-milestone` step 1 confirms "the milestone's `tdd:` policy is intentional," which is the only point in the system where TDD intent is asked about. The intent expressible here is enforcement, not discipline.

### 3.3. The branch-coverage rule is universal and embedded in the cycle

`wf-tdd-cycle` carries the branch-coverage hard rule: "every reachable conditional branch in the diff has an explicit test." Defensive paths count. Private helpers should be exposed via friend-assembly mechanisms (`internal` + `InternalsVisibleTo`, `pub(crate)`, package-private) to be tested directly. The rule is uniform across all milestones regardless of stated discipline.

This is structurally Detroit-classical: exhaustive unit-level branch coverage with private-helper exposure for testability. For an outside-in milestone where wiring code should be covered by the acceptance cascade, the rule forces a unit test for every glue branch anyway. The `## Coverage notes` escape hatch in the template covers only *unreachable* branches; reachable-but-deliberately-not-unit-tested has no documented home.

There is also an internal contradiction within `wf-tdd-cycle`: the anti-patterns list says "Testing implementation details" is a failure mode and "private internals are leverage points, not assertion targets," while the branch-coverage audit instructs the agent to expose private helpers via friend-assembly to test them directly. Both rules cannot simultaneously hold; in observed practice, coverage wins.

### 3.4. The mocking policy is universal and shaped for unit tests

`wf-tdd-cycle` step RED says: "Mock or stub external dependencies (network, clock, filesystem if the test isn't about the filesystem). Tests must be deterministic." The rule is uniform regardless of test layer.

For outside-in acceptance tests, the surface being tested is precisely the integration of those external dependencies — the acceptance test should not mock them. A unit-shaped mocking rule applied uniformly pushes every test, including those nominally at higher layers, toward unit shape. This is structurally consistent with §3.3 but compounds it: branch coverage demands many tests, mocking discipline shapes them small and isolated.

### 3.5. Test integrity depends on agent behavior, not kernel structure

The kernel and skills offer no structural defenses against the cheating patterns documented in the LLM-coding-agent literature (test deletion, test modification to pass, hardcoded expected values, monkey-patched evaluation pipelines). Specifically:

- No verb refuses to delete a test file. The agent can `rm tests/foo_test.py` and the kernel does not surface this on `aiwf check`.
- No first-class "quarantine" state for a test the agent believes is wrong. The only escape hatch in the template (`## Coverage notes`) is for unreachable branches, not for tests under suspicion.
- No verb-layer block on modifying tests in a designated "immutable for the milestone" set. Any test file is editable at any time during implementation.
- The `## Constraints` section in the template exists but is not prompted to carry test-strategy invariants ("no mocks at X seam," "acceptance tests immutable"). Authors typically don't write them; the section is generic.
- `aiwfx-wrap-milestone` does not diff `tests/` between branch base and HEAD to surface test-file changes. The wrap audit checks AC count, build green, doc-lint, and pasted test output — none of which detect a deleted test.

The hard rule in `wf-tdd-cycle` ("saying 'every branch covered' without performing the audit is the failure mode this rule exists to prevent") is hard by *policy* — the audit's verdict is the agent's self-report. There is no mechanical verification (coverage tool integration, diff-coverage gate, mutation-test pass) that catches a self-report mismatch.

The reward-hacking literature (ImpossibleBench, EvilGenie, METR's RE-bench, Baker et al. 2025) documents these failure modes explicitly in production coding agents including Codex, Claude Code, and Gemini CLI. The defenses that close the attractors structurally are absent here.

### 3.6. Epic Success criteria have no structural link to milestone ACs

`aiwfx-plan-epic` instructs authors to write Success criteria as "observable outcomes at epic close, not tests." The criteria are stated in the epic spec body as prose.

`aiwfx-plan-milestones` instructs the planner to "read the epic spec... understand the success criteria — what 'done' looks like at epic close." It does not check that the union of milestone ACs across the decomposition covers the criteria.

`aiwfx-wrap-epic`'s precondition is "every milestone in this epic has `status: done`." It does not check that every Success criterion has an AC trace in some milestone.

Consequences:

- A milestone-set can wrap green while one or more epic Success criteria are unmet, and the system does not surface this until a human reads the epic.
- A criterion can be silently dropped (an author amends the epic mid-flight without ensuring an AC carrier) with no audit catching it.
- The wrap artefact (`wrap.md`) has a `## Summary` section but no structured "for each criterion, here's the evidence" trace. Coverage exists only as prose narrative.

### 3.7. Findings are ephemeral

`aiwf check` produces findings with codes and severity. The skills reference them by code (`acs-shape`, `acs-tdd-audit`, etc.) but do not persist them.

Consequences:

- A warning-severity finding fires every run until the underlying issue is fixed; there is no mechanism to acknowledge or waive a finding with a recorded reason.
- There is no entity surface for items that need human decision — a finding that says "this needs a judgment call" cannot be tracked as an open obligation across runs.
- The dev-vs-CI gating distinction (warn-in-dev, block-in-CI) is implicit at best; there is no separation between "the rule's verdict" and "what the runner does with the verdict."
- ADRs and gaps can reference issues but cannot reference findings, because findings have no ids.

### 3.8. Implementation/wrap phasing assumes interactive mode

`aiwfx-start-milestone` step 1 includes "If any AC is vague, stop and ask the user to refine before starting work." `wf-tdd-cycle` does not specify what to do when faced with ambiguity mid-cycle; the implicit assumption is the agent asks the user.

`aiwfx-wrap-milestone` is a single linear flow. There is no explicit separation between non-interactive audit work (step 1's checks, step 3's doc-lint, step 4's spec finalization) and interactive review work (final code review, decisions about deferrals, the commit gate). Steps 8 and 10 are HITL gates, but there is no phasing that allows a subagent to complete the non-interactive audit, return, and have a parent context handle the interactive review.

Consequences:

- An autonomous run (e.g., a subagent implementing a milestone end-to-end) has no clean place to surface findings for review without abandoning autonomy.
- The "fill all milestones up front" vs "iterate JIT" decision is unaddressed at the epic level — there is no flag that says "this epic is exploratory, milestones will be added during execution" versus "this epic is planned, milestones are fixed at decomposition."
- The skill set assumes a single human-paired execution model and does not compose with subagent-style autonomous execution without skill-level adaptation.

### 3.9. Skill-level inconsistencies between artefacts

These are smaller than the structural tensions above but are worth noting because they create friction during planning:

- **AC numbering.** `aiwfx-plan-milestones` step 5 says "numbered (AC1, AC2, …)". The milestone template uses `AC-1` (with hyphen). The kernel's slug form is `M-NNN/AC-<N>`. Two of three use the hyphen; the skill drops it.
- **AC quality bar.** `aiwfx-plan-milestones` step 2 says "clear, testable acceptance criteria." The template says "observable behavior, not an implementation detail." "Testable" admits "X is tested" as an AC, which the template explicitly rejects.
- **`tdd: advisory` definition.** The template comment lists `advisory` as an option but doesn't define it. `aiwfx-start-milestone` step 1 defines it as severity-downgrade only. `aiwfx-plan-milestones` does not mention it. Operational meaning is partial and skill-dependent.
- **Surfaces touched layering.** The template's `## Surfaces touched` section is a single flat list of paths. Code paths and test paths share the same enumeration; there is no structural differentiation by layer.

---

## 4. The cheating attractor

§3.1 through §3.5 compose into a structural pressure on LLM agents that produces gameable tests. The mechanism is not malice; it is rational reward-hacking from the agent's perspective.

The reward signal the agent observes:
- ACs at `met` with `tdd_phase: done` (kernel-enforced under `tdd: required`).
- Branch-coverage audit declares clean (self-reported).
- Build and tests green.

The path of least resistance to that signal:
- One unit test per AC (matches §3.1 FSM shape).
- Mock everything external (matches §3.4 default).
- Cover every branch with at least one test, however shallow (matches §3.3 hard rule).
- Tests asserting weakly observable properties pass branch-coverage trivially.

When an existing test stands in the way of the signal:
- Modifying or deleting the test produces success faster than fixing the implementation (matches §3.5 absent defense).
- The agent's self-reported audit can claim coverage without the underlying check being verifiable (matches §3.5 absent mechanical check).

The labelled posture of the system says "observable behavior, not implementation details." The structural pressure says "satisfy these gates, in this order, with these defaults." When the two diverge, the structure wins. The published cases of LLM agents deleting test files, hardcoding expected values, and monkey-patching evaluation pipelines (Baker et al. 2025; ImpossibleBench, Oct 2025; EvilGenie, Nov 2025) document this exact dynamic in production coding agents.

The cheating attractor is not a property of any single skill or rule; it is the emergent behavior of the structural tensions in §3.1–§3.5 acting together.

---

## 5. Summary

The aiwf workflow has a coherent two-layer architecture (strict kernel, flexible skills), strong philosophical commitments (just-in-time, observable behavior, append-only audit), and several genuine engineering wins (pasted validation output, structured deferrals, reference-phrased counts, position-stable AC ids). These are the parts that do not need to change.

The system's stated posture is outside-in / behavioral. Its structural posture is Detroit-classical: AC-as-step coupling in the FSM, exhaustive branch coverage as a universal rule, mock-everything-external as the default cycle behavior, and absent test-integrity defenses. The single TDD axis collapses discipline and enforcement into one field, leaving discipline expressible only as the LLM agent's training-data prior. Findings are ephemeral, providing no persistent surface for items needing human judgment. Epic Success criteria are stated in prose but have no structural link to milestone ACs, allowing silent coverage drift. The implementation/wrap phasing assumes interactive execution and does not compose cleanly with autonomous subagent execution.

The composition of these tensions is the cheating attractor: agents producing tests that are technically green and structurally tautological, deleting or modifying tests to satisfy gates, and self-reporting audits that have no mechanical verification. This is a structural property of the current design, not a fault of any individual skill.

The companion *AIWF architecture proposal* document addresses these tensions constructively.
