# AIWF TDD architecture proposal

**Status.** Working synthesis. Not a final landing — expect further iteration before the migration slices are filed as epics.

**Companion documents.**
- [`06-tdd-diagnostic.md`](06-tdd-diagnostic.md) — fact-grounded analysis of the current architecture's structural tensions. Motivates this synthesis. Independently readable; unchanged across revisions.
- [`docs/pocv3/design/agent-orchestration.md`](../pocv3/design/agent-orchestration.md) — substrate design for subagent execution, capability registry, pipeline schema, cycle envelope. Subsumed by Slices 6–7 of this synthesis.
- [`docs/adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md`](../adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md) — F-NNN as 7th entity kind. Implemented in Slice 2.
- [`docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md`](../adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — uniform archive convention. Lands with Slice 2.

**Audience.** AIWF maintainers and contributors evaluating the next set of milestones. Assumes familiarity with the existing kernel, the `aiwfx-*` and `wf-*` skill families, and CLAUDE.md's commitments.

---

## 1. Goals

The redesign targets six concrete outcomes:

1. **TDD discipline is decoupled from AC structure.** ACs describe observable behavior; their `met` status is gated by *evidence audits*, not by a coupled per-AC phase FSM. The kernel stops baking in Detroit-classical assumptions.

2. **Discipline (Detroit, London, property, contract-first, ad-hoc) is a skill concern, not a kernel enum.** A milestone declares its TDD posture (`required | advisory | none`); the cycle skill chooses the cycle shape. New disciplines are skill additions, not kernel changes.

3. **Cheating attractors are mechanically visible without hard gates.** AC-met evidence audits (cycle-commit presence, batch detection) produce findings. Findings persist as F-NNN entities; the human triages at wrap-time. Hard blocks remain at obvious chokepoints; softer surface for everyday discipline.

4. **HITL surface via persistent findings.** F-NNN entities are the persistent surface for items needing human attention. `aiwf check` produces and reconciles them; AC closure soft-blocks on open findings linked to the AC; the human resolves or waives at wrap.

5. **Subagent-friendly cycle execution arrives via the agent-orchestration substrate.** That work is sequenced after the kernel-side discipline shifts so it lands into a clean evidence-audit surface and addresses G-067's within-cycle drift directly.

6. **No new entity kinds beyond F-NNN. No new mandatory verb flags.** The kernel grows by exactly one kind. AC `met` keeps its current verb signature.

### Adherence to existing engineering principles

- **KISS / YAGNI.** Every addition closes a concrete tension the diagnostic identifies or the conversation surfaced. T-NNNN, discipline enum, cycle-skill family expansion, structured `success_criteria`, epic mode, mandatory test trailers are all deferred or rejected (§10).
- **Errors are findings, not parse failures.** F-NNN is the persistent counterpart to the existing `aiwf check` finding output.
- **Framework correctness must not depend on LLM behavior.** Evidence audits are kernel-side; cycle skills are advisory. Cheats are made visible by structural rules that walk commit history, not by trusting prose declarations or trailer counts.
- **Kernel functionality must be AI-discoverable.** Every new verb, finding code, trailer key ships with `--help` text and skill-level documentation alongside the implementation.

---

## 2. Design principles

**P1 — Kernel = small core of strict invariants. Skills = flexible LLM-facing layer.**
The kernel knows about entities, ids, FSM transitions, tree shape, audit registration, and trailer-derived history queries. Discipline rules, cycle shape, branch-coverage policy, mocking conventions, prompting heuristics live in skills.

**P2 — Acceptance criteria are behavioral commitments, not test-coupling units.**
ACs have a status FSM (`open → met | deferred | cancelled`). They do not carry phase information; their relationship to tests is implicit via the trailer history of commits in their lineage and explicit (when the practitioner chooses) via the `## Testing` body section. Many tests can verify one AC; one test can verify many ACs; the kernel does not constrain or model the relation.

**P3 — TDD discipline is a skill concern, declared as advisory metadata.**
The milestone's `tdd: required | advisory | none` field controls audit strictness. A free-form `tdd_style:` field (deferred to Slice 8) lets cycle skills dispatch on practitioner discipline (Detroit, London, property) without the kernel knowing. Disciplines are added as skills ship; no kernel enum.

**P4 — Test integrity is enforced via commit history, not entity tracking, not declared counts.**
The kernel never tries to identify "what is a test file." It walks commit trailers, applies evidence audits, and produces findings on the AC's lineage. Test-file conventions are a stack-and-project concern; aiwf is path-blind by default. Test execution is not declared to the kernel; its evidence is the practitioner's `## Testing` prose record and the human review at wrap.

**P5 — Findings persist as entities; severity is on the rule, gating is on the runner.**
`aiwf check` produces F-NNNN entities with `severity` (error/warning/info) and `requires_human_decision` (a property of the rule code). Whether a finding blocks merge or proceeds is the runner's responsibility — `aiwf check` reports, `aiwf check --strict` blocks. Per ADR-0003.

**P6 — Just-in-time authoring with explicit strategic/tactical split.**
Strategic metadata (TDD posture, milestone shape) is set at plan time and is stable. Tactical detail (test strategy prose, AC bodies, design notes) is filled at start time against current context. The skill family enforces this split.

**P7 — Hard gates are reserved for obvious chokepoints.**
The kernel uses warnings (with `--force --reason` overrides) for discipline cheats. Hard blocks at the verb layer are reserved for structural integrity (id allocation, FSM legality, schema validity). Discipline is *visible*, not enforced. The cheating attractor is closed by making cheats traceable, not by trying to block them.

**P8 — Wrap phases cleanly: audit → review → commit.**
The skill restructures around three phases: A (non-interactive audit), B (HITL on findings, conditional), C (commit + push). The kernel doesn't know about phases; it knows about findings, and the skill chooses how to phase its workflow around them.

**P9 — Provenance composes orthogonally with runner mode.**
Per CLAUDE.md commitment #9, principal × agent × scope tracks accountability. The runner mode flag (`--autonomous | --interactive`, set at invocation) controls HITL behavior in skills; scope (opened via `aiwf authorize`) controls kernel-level authority. Same composition rules as agent-orchestration §2 P9.

---

## 3. Entity model

### 3.1. Entity catalog

**Seven entities total: six preserved + F-NNN.**

| Entity | ID | Parent | Status FSM | Notes |
|---|---|---|---|---|
| Epic | `E-NN` | none | `proposed → active → done | cancelled` | Unchanged |
| Milestone | `M-NNN` | epic | `draft → in_progress → done | cancelled` | Adds `--tdd` chokepoint at create-time (Slice 1, E-16) |
| Acceptance Criterion | `M-NNN/AC-N` | milestone | `open → met | deferred | cancelled` | Compound id (preserved); **loses `tdd_phase`** (Slice 3) |
| ADR | `ADR-NNNN` | none | (existing) | Unchanged |
| Decision | `D-NNN` | none | (existing) | Unchanged |
| Gap | `G-NNN` | discovered-in milestone | (existing) | Unchanged |
| Contract | `C-NNN` | none | (existing) | Unchanged |
| **Finding** | **`F-NNNN`** | none | `open → resolved | waived | invalid` | **New (ADR-0003), Slice 2** |

No T-NNNN test artifact entity. No verification-source entity. No structured `success_criteria` array. No epic `mode:` field. (See §10 for the rationale on each deferral.)

### 3.2. Entity relationships

```
E-NN ─── parent of ──► M-NNN ─── parent of ──► M-NNN/AC-N
                            │
                            ├── depends_on ──► M-NNN
                            │
                            └── discovered_in ──► G-NNN

ADR-NNNN, D-NNN, C-NNN — no parent; reference subjects via explicit fields

F-NNNN ─── linked_acs ──► M-NNN/AC-N
       └── linked_entities ──► (any entity)
       └── waived_by ──► ADR-NNNN
```

The AC ↔ test relationship lives in:
- The trailer history of commits with `aiwf-entity: M-NNN/AC-N` (implicit, mechanical, walked by audits)
- The optional per-AC `## Testing` prose section (explicit, human-readable, not parsed by the kernel)

### 3.3. Frontmatter schemas

#### Milestone (revised)

```yaml
---
id: M-066
title: <imperative title>
parent: E-16
status: in_progress
tdd: required          # required | advisory | none — controls audit strictness
depends_on: [M-061]
acs:
  - id: AC-1
    title: <observable behavior>
    status: open       # open | met | deferred | cancelled
  - id: AC-2
    title: <observable behavior>
    status: open
---
```

Changes from current schema:
- AC frontmatter: **`tdd_phase` removed** (soft-deprecated; see §6 migration).
- Milestone frontmatter: **no other changes**. The `tdd:` field's three-value enum is unchanged.

What's *not* added:
- No `verified_by` on AC.
- No `tdd:` block (`discipline | enforcement | gate | branch_coverage`).
- No `tdd_style:` field (deferred to Slice 8 if real friction shows up).
- No `mode:` on epic.
- No `success_criteria:` array on epic.

#### Finding

```yaml
---
id: F-1023
title: AC-1 promoted met with no work commits in lineage
status: open                  # open | resolved | waived | invalid
code: ac-met-without-cycle-commits
linked_acs: [M-066/AC-1]
linked_entities: []
recorded_by: framework/aiwf-check
---
```

Per ADR-0003 §"Frontmatter".

### 3.4. Status FSMs

#### Acceptance Criterion (unchanged)

```
open ──► met
  │       │
  ├──► deferred       (work moved to a gap or future milestone)
  │       │
  ├──► cancelled      (no longer applies)
  │
  └──► (met can also move to deferred / cancelled if scope changes after the fact)
```

The "met requires `tdd_phase: done`" audit rule (`acs-tdd-audit`) is **removed in Slice 3**. Replaced by the AC cycle-evidence findings (§5).

#### Finding

Per ADR-0003. `open → resolved | waived | invalid`. All terminal. `waived` requires `--force --reason` and a waiving ADR; `invalid` requires `--reason`. Sovereignty is human-only by the existing kernel rule.

### 3.5. Verb surface

#### Changed verbs

```bash
aiwf promote M-NNN/AC-N met
  # No new flags. The verb gains the findings-block-met chokepoint:
  # if any open F-NNN has the AC in its linked_acs, the verb refuses.
  # Override via --force --reason (human-actor only by existing rule).
```

The verb's signature stays exactly as today. The new behavior is the chokepoint.

#### Soft-deprecated verbs

```bash
aiwf promote M-NNN/AC-N --phase red | green | refactor | done
  # Continues to work for ONE minor release after Slice 3 ships.
  # Each invocation prints a deprecation notice naming the new model.
  # Removed in the following minor release.
  # The wf-tdd-cycle skill stops calling it on Slice 3 ship.
```

#### New verbs (Slice 2)

Per ADR-0003, all promote / cancel / show / history work via the kernel's generic dispatch. The single new verb is the F-NNN allocator:

```bash
aiwf add finding --code <code> --linked-acs <ac-ids> --title "..." [--body-file <path>]
```

#### Removed concepts

- `tdd_phase` field on AC (parser tolerates it as inert frontmatter; ignored by all rules; new ACs don't write it).
- `acs-tdd-audit` rule ("AC met requires tdd_phase: done") — replaced by the cycle-evidence findings in §5.

### 3.6. Audit codes

Existing codes preserved where they still apply. New codes:

| Code | Severity | Trigger |
|---|---|---|
| `ac-met-without-cycle-commits` | warning under `tdd: required`; info under `advisory`; silent under `none` | `aiwf history M-NNN/AC-N` shows zero commits with the AC's trailer between the add and met commits |
| `ac-batch-promotion` | warning under `tdd: required`; info under `advisory`; silent under `none` | 2+ ACs in the same milestone hit `met` in the same commit SHA |
| `tests-deleted-in-milestone` | warning under any `tdd:` value | Diff between milestone branch base and HEAD shows test-file deletions without an associated F-NNN waiver. **Path detection is opt-in via `aiwf.yaml.tdd.test_globs:`; absent config = rule silently doesn't fire.** |
| `findings-block-met` | error (soft chokepoint at promote-time) | `aiwf promote M-NNN/AC-N met` attempted while open F-NNN findings reference the AC. Override via `--force --reason`. |
| `ac-has-open-findings` | warning (visibility surface) | Standing-rule version of `findings-block-met` — fires on every `aiwf check` for ACs with open findings, regardless of promote state. |
| `finding-resolved-without-fix-link` | warning (soft check) | Per ADR-0003 — F-NNN promoted resolved with no associated fix commit nearby. |

Removed:
- `acs-tdd-audit` (replaced by the two cycle-evidence findings above)

A future `tdd.strict: true` config (deferred, §10.6) can promote the warnings to errors.

---

## 4. The TDD posture model

### 4.1. The `tdd:` field

A milestone declares its TDD posture as one of three values:

- **`required`** — TDD discipline is structurally important. Evidence audits fire as warnings (escalatable to errors via future `tdd.strict: true`). The wrap skill prompts about missing `## Testing` sections.
- **`advisory`** — TDD discipline is the project's preference but not enforced. Evidence audits fire as info.
- **`none`** — No formal TDD. Evidence audits are silent. Used for documentation sweeps, exploratory work, refactors with no observable behavior change.

The field is a **strictness knob**, not a discipline selector. A milestone running outside-in or property-based testing under `tdd: required` is just as well-served as a Detroit-classical one — the audits care about *cycle evidence in commit history*, not *discipline shape* or *test counts*.

E-16 ships the `--tdd` chokepoint at create-time + `aiwf.yaml: tdd.default` config + `aiwf update` migration + `milestone-tdd-undeclared` defense-in-depth check. That work is unchanged by this synthesis; it forms Slice 1.

### 4.2. Discipline as a skill concern

The cycle skill (`wf-tdd-cycle` today; potentially `wf-bdd-cycle`, `wf-property-cycle` if they ship) chooses the cycle shape, mocking conventions, branch-coverage interpretation. It does not need a kernel enum to dispatch.

If projects start running multiple disciplines in the same repo, an optional `tdd_style:` free-form frontmatter field lets the cycle skill dispatch:

```yaml
tdd: required
tdd_style: outside-in   # or detroit, property, contract-first, ad-hoc — free-form
```

The kernel does not validate `tdd_style:`. The skill reads it and chooses behavior. When/if a style becomes load-bearing for some kernel rule, promote it to a closed-set then. **This is deferred to Slice 8 — ships only when a real milestone needs it.**

The internal contradiction in the current `wf-tdd-cycle` skill (the anti-patterns list says "don't test private internals" while the branch-coverage rule instructs the agent to expose privates via friend-assembly) gets resolved in the Slice 3 skill update, with a clear rule for when private-helper exposure is acceptable.

### 4.3. The kernel does not detect test files

The kernel does not try to identify "what is a test file." Test conventions vary too widely across stacks:

- Go puts unit tests next to source (`*_test.go`); integration tests sometimes in `tests/`.
- Rust puts unit tests inline (`#[cfg(test)] mod tests`); integration in `tests/`.
- TypeScript has `__tests__/`, `*.test.ts`, `*.spec.ts`, often co-located.
- .NET uses separate test projects (`Foo.Tests/`).
- Solo devs do whatever they did last time; polyglot repos mix conventions.

A configurable `tdd.test_globs:` would put the burden on every project to configure correctly, with audits silently lying when configuration drifts — the "framework correctness must not depend on remembering" failure mode.

The kernel's mechanical surface for test work is therefore narrow:
- **Cycle-commit presence** in the AC's trailer history (§5.1).
- **Batch-promotion detection** via SHA equality (§5.2).
- **Open findings** linked to the AC (§5.4).

These tell the kernel "work happened in this AC's lineage" and "ACs were promoted in batch" — both mechanically clean. They do not tell the kernel "test work happened" — that's a discipline question answered by the practitioner's `## Testing` prose section (§7) and the wrap skill's review prompt (§8).

The `tests-deleted-in-milestone` heuristic is the one place a path concept appears, gated behind opt-in `aiwf.yaml.tdd.test_globs:`. Absent configuration = the rule silently doesn't fire.

### 4.4. The `aiwf-tests:` trailer

An `aiwf-tests: pass=N fail=N skip=N` trailer keyword is recognized by the parser. The kernel does not write it; no audit consumes it. Cycle skills may write it directly via `git commit -m "... aiwf-tests: pass=N fail=N"` if they want a per-cycle test-count record — purely informational.

The governance HTML render's Tests tab may render `## Testing` section content rather than trailer counts. A future deprecation of the trailer keyword is possible if no consumer adopts it.

---

## 5. AC cycle-evidence audits

Two new audit codes from §3.6 are the synthesis's main structural contribution. Each fires under specific cycle-history shapes; together they make the across-AC batched-met cheat mechanically visible.

### 5.1. `ac-met-without-cycle-commits`

**Trigger.** `aiwf promote M-NNN/AC-N met` lands and `aiwf history M-NNN/AC-N` between the AC's add commit and the met commit shows zero work commits (only the add and the met).

**Failure mode it catches.** AC promoted met with no incremental work — the LLM (or human) wrote the AC body, then immediately promoted met without doing the implementation work the AC describes. Captures the "filled in the AC body, called it done" cheat.

**Override paths.**
- Genuine zero-work AC (e.g., a behavior already verified by an existing test from a prior milestone) — `aiwf promote M-NNN/AC-N met --force --reason "verified by tests/foo_test.go::TestAlreadyExists from M-061"`.
- A practitioner who legitimately ships impl + met in a single commit can carry the AC's trailer on that commit; the audit walks lineage including the met commit itself.

### 5.2. `ac-batch-promotion`

**Trigger.** Two or more ACs in the same milestone reach `met` in the same commit SHA.

**Failure mode it catches.** Hand-editing milestone frontmatter to flip multiple AC statuses in one commit. The standard `aiwf promote` verb produces one commit per call, so the only way to batch is hand-edit (which leaves the same SHA across all flipped ACs).

**Override paths.**
- A genuinely-batched promote (rare: e.g., AC re-frame mid-milestone where two ACs collapse into one) — `--force --reason "AC-2 absorbed into AC-1; both verified by same test"` on the batch commit.
- Otherwise: cancel the batch commit, re-promote each AC individually.

### 5.3. Severity matrix

| `tdd:` value | `ac-met-without-cycle-commits` | `ac-batch-promotion` |
|---|---|---|
| `required` | warning | warning |
| `advisory` | info | info |
| `none` | silent | silent |

(Future `tdd.strict: true` config can promote warnings to errors. Deferred §10.6.)

### 5.4. The findings-gated AC closure (soft chokepoint)

`aiwf promote M-NNN/AC-N met` reads the F-NNN tree:
- If any open F-NNN has the AC in `linked_acs`, the verb refuses with `findings-block-met`.
- Override via `--force --reason "<text>"` (human-actor only by existing rule).

This is a *soft chokepoint*: the rule is mechanical but the human can override. It's not a one-shot kernel block; it's a "pause and triage" surface.

The standing-check counterpart `ac-has-open-findings` (warning) fires on every `aiwf check` for ACs with open linked findings, regardless of promote state. Visibility on top of the chokepoint.

### 5.5. The layered surface for test-work evidence

The mechanical audits in §5.1–§5.2 catch zero-work and batched-work cheats. They do not catch "work happened but no tests" — distinguishing test work from other work would require the kernel to identify test files, which §4.3 rules out.

That gap is closed by a layered surface:

1. **Practitioner discipline + cycle skill prompts.** `wf-tdd-cycle` (and disciplinary siblings) prescribe red-first ordering as practice.
2. **`## Testing` per-AC body section** (§7). Practitioner writes test names / paths / approaches in prose; visible inline next to the AC.
3. **Wrap Phase A surfaces missing sections.** The wrap skill (§8) lists ACs that promoted met without a `## Testing` subsection for the human to consider during review.
4. **Wrap Phase B human judgment.** When the layered surfaces flag something, the human decides at wrap whether the AC was actually verified.
5. **Subagent isolation (Slice 6).** When the agent-orchestration substrate lands, cycle drift becomes structurally bounded by the subagent's lifetime — addressing G-067 directly.

The discipline question is a human question; the kernel makes it visible at the right moment.

---

## 6. Migration

The synthesis is **backward-compatible**. No existing tree breaks.

### 6.1. `tdd_phase` soft-deprecation

The field is **kept in the parser** as tolerated-but-ignored:
- Existing milestones with `tdd_phase: done` continue to validate (the parser reads the field but no rule consults it).
- The `acs-tdd-audit` rule is **removed** in Slice 3 (replaced by the cycle-evidence findings).
- New `aiwf add ac` does not seed `tdd_phase`; the field is omitted from new entries.
- A future `aiwf rewrite --strip-deprecated` can bulk-clean existing milestones; not on the critical path.

### 6.2. `aiwf promote --phase` soft-deprecation

The verb form continues to work for **one minor release** after Slice 3 ships:
- Each invocation prints a deprecation notice naming the new model:
  `WARNING: aiwf promote --phase is deprecated. Phase tracking is no longer kernel-side; see CHANGELOG.md vX.Y.Z. The verb is removed in vX.Y+1.0.`
- The `wf-tdd-cycle` skill update (Slice 3) stops calling `--phase`.
- The verb is removed in the following minor release.

This gives the rituals plugin time to ship its updated skill before the verb disappears.

### 6.3. Historical AC grandfather rule

ACs already at `met` with `tdd_phase: done` (e.g., E-07, E-14, E-21 milestones) are **not retroactively audited** by the new cycle-evidence findings. The audits walk the AC's lineage relative to its add commit; ACs that pre-date the rules are silently exempt.

This mirrors the G-055 grandfather pattern E-16 already uses for `milestone-tdd-undeclared`.

### 6.4. The `aiwf-tests:` trailer

Per §4.4, the trailer is now legacy:
- Parser continues to read it (no breaking change for existing commits).
- No kernel verb writes it.
- No audit consumes it.

Cycle skills may write it directly via `git commit -m "... aiwf-tests: pass=N fail=N"` if they want a record. The HTML render's Tests tab can either repurpose to render `## Testing` content or be dropped.

---

## 7. The `## Testing` section

A per-AC body subsection, free-form prose. The cycle skill writes notes here for human bookkeeping; the wrap skill prompts about its absence at audit time. Kernel does not parse.

Example:

```markdown
### AC-1 — POST /checkout with empty cart returns 422

The handler returns 422 when the cart is empty.

#### Testing
- `tests/acceptance/checkout_test.go::TestEmptyCart_Returns422` — covers AC-1 happy path
- `tests/acceptance/checkout_test.go::TestEmptyCart_ReturnsErrorMessage` — also covers AC-2 (cross-reference)
- Branch coverage: every reachable branch in `handler.Checkout` exercised; no `## Coverage notes` exception
```

The section is **the practitioner's primary evidence record**. It works for every discipline:

- **Detroit-classical:** `tests/cart_test.go::TestEmptyCart_Returns422`
- **Outside-in:** `tests/acceptance/checkout.feature::Scenario "empty cart"`
- **Property:** `property: commutative_addition (1000 generated cases)`
- **Contract-first:** `contracts/checkout/v2/valid/empty-cart.json accepts; invalid/missing-cart.json rejects`
- **Solo ad-hoc:** `manual: ran curl, got 422`

The section is **optional** at the kernel level. Cycle skill writes it when test work is non-trivial; skips when test inventory is one line. `aiwf check` does not enforce its presence; `acs-body-coherence` does not warn on its absence.

The wrap skill, however, *does* surface its absence: Phase A lists ACs that promoted met without a `## Testing` subsection for the human reviewer. This is a skill-level prompt, not a kernel finding — the practitioner can answer "covered by [other AC's test]," "covered by integration suite from M-061," "manual verification only," or write the section.

For milestones where one acceptance test verifies many ACs, the cycle skill may choose a milestone-level `## Testing` section instead, organized by AC. Both layouts are valid markdown.

The kernel knows nothing about which test files verify which ACs. Test-name lists in `## Testing` are documentation that travels with the AC; the audit surface is the trailer history (mechanical) + the wrap skill's prompts (process).

If projects later treat the test-name lists as references the kernel should validate (e.g., "this test deleted; F-NNN cites the ACs it verified per their `## Testing` section"), the section can be promoted to structured then. Not in scope now.

---

## 8. Wrap phasing (skill-only)

`aiwfx-wrap-milestone` reorganized into three phases:

```
Phase A — Wrap audit (non-interactive)
  - Run aiwf check; collect findings.
  - Run cycle-evidence audits (the two §5 codes); collect findings.
  - Run test-deletion heuristic (if test_globs configured).
  - Run finding-gated AC closure visibility (ac-has-open-findings).
  - Run doc-lint sweep.
  - List ACs at met without a `## Testing` subsection for human review.
  - Finalize wrap-side spec sections (Work log, Validation, Deferrals, Reviewer notes).
  - Compute the blocking-findings set.

Phase B — HITL review (conditional on findings or missing-section list)
  - Skipped entirely if no blocking findings exist AND no ACs lack a
    `## Testing` section.
  - Otherwise: walk each blocking finding with a human:
      - Resolve (with --by commit-sha | ADR-NNNN | D-NNN | G-NNN)
      - Waive (with --adr ADR-NNNN)
      - Defer (open gap, link)
    Then walk each AC missing `## Testing`:
      - Write the section, OR
      - Note "verified by <other-AC>" / "manual verification only" / etc.
        (Just prose; no kernel artifact.)
  - End condition: every blocking finding is at a terminal state and
    every met AC has a `## Testing` section or an explanatory note.

Phase C — Wrap commit (non-interactive aside from existing gates)
  - Stage milestone spec changes.
  - Show diff and proposed commit message.
  - Human commit gate (preserved from current wrap).
  - Commit.
  - Promote milestone to done.
  - Human push gate (preserved).
  - Push.
  - Update roadmap.
```

This is a **skill-only update** to `aiwfx-wrap-milestone`. The kernel doesn't know about phases; it knows about findings, and the skill chooses how to phase its workflow around them.

Subagent compatibility: a subagent running the milestone end-to-end runs Phase A and returns the findings list and the missing-section list with the staged-but-uncommitted state. The parent context runs Phase B (with HITL) and Phase C. The kernel's `findings-block-met` chokepoint ensures the next milestone cannot start while Phase B's blocking findings remain open. This composes cleanly with the agent-orchestration substrate (Slice 6) without further kernel work.

Phase A and C are agent-runnable end-to-end. Phase B is human-time when present and skipped entirely when not.

---

## 9. Migration slices

In order; each independently shippable; each closes a specific pain. CLAUDE.md updates ride with the slice that makes them true (Appendix A).

### Slice 1 — `--tdd` chokepoint (E-16, already drafted)

Per E-16 and its four milestones (M-062..M-065):
- `--tdd required | advisory | none` flag at create-time on `aiwf add milestone`
- `aiwf.yaml: tdd.default` schema field
- `aiwf init` seeds `tdd.default: required` with explanatory comment
- `aiwf update` migrates existing repos with loud output
- `milestone-tdd-undeclared` finding (warning; error under future `tdd.strict: true`)

Closes G-055. Independent of every other slice. Ships as drafted.

### Slice 2 — F-NNN entity + uniform archive convention

Per ADR-0003 + ADR-0004:
- F-NNN as 7th entity kind with status FSM (`open → resolved | waived | invalid`), frontmatter, body
- `aiwf add finding` verb
- `aiwf promote F-NNN <terminal>` reuses generic dispatch
- `waived` requires `--force --reason` and a waiving ADR (sovereignty rule); `invalid` requires `--reason`
- Uniform archive convention for terminal-status entities (all kinds): terminal-status promotion moves the file to `<kind>/archive/` in the same atomic commit
- `aiwf list` / `aiwf status` filter active by default; `--include-archived` reveals
- Soft check `finding-resolved-without-fix-link` (warning) per ADR-0003

### Slice 3 — AC model revision

The architectural shift:
- Soft-deprecate `tdd_phase` field on AC (parser tolerates; new ACs don't write)
- Soft-deprecate `aiwf promote --phase` verb form (one-release deprecation notice; removed in following release)
- Remove `acs-tdd-audit` rule
- `wf-tdd-cycle` skill update (rituals plugin): drop `--phase` calls; prose RED/GREEN/REFACTOR/DONE stays as practitioner advice; resolve the internal contradiction on private-helper exposure
- Backward-compatibility tested: old milestones validate unchanged

### Slice 4 — AC cycle-evidence audits + finding-gated closure + test-integrity heuristic

The new structural surface:
- `ac-met-without-cycle-commits` (warning under `tdd: required`)
- `ac-batch-promotion` (warning under `tdd: required`)
- `findings-block-met` soft chokepoint at met-promote (override via `--force --reason`)
- `ac-has-open-findings` standing-rule visibility
- `tests-deleted-in-milestone` heuristic check (opt-in via `aiwf.yaml.tdd.test_globs:`)

Depends on Slice 2 (F-NNN entity) and Slice 3 (AC model).

### Slice 5 — Wrap A/B/C as skill update

Skill-only restructure of `aiwfx-wrap-milestone`:
- Phase A (non-interactive audit, including the missing-`## Testing` list)
- Phase B (HITL on blocking findings + missing-section walkthrough, conditional)
- Phase C (commit + push, existing HITL gates preserved)

No kernel changes. Depends on Slice 4 (findings exist as the substrate Phase B operates on).

### Slice 6 — Agent-orchestration substrate

Per `docs/pocv3/design/agent-orchestration.md` §13 (its own sequencing):
- Capability registry (`aiwf.yaml.subagents.agents[]`)
- Sub-scope FSM extension + role-tagged actor + cycle trailer keys
- Pipeline schema parser + reconciliation check + `pipeline-*` finding codes
- Cycle envelope schema + forensic bundle layout + reap verbs
- Verb gate for subagent context (refuse `cancel`, `reallocate`, `authorize`, `--force`)
- Provenance check rule (`cycle-trailer-incomplete`)

Depends on Slices 1–5 (the kernel-side discipline shifts give substrate work a clean evidence-audit surface to land into).

This slice is the structural answer to G-067 (within-cycle drift). Bounded subagent context replaces the long-conversation-drift failure mode that motivated the diagnostic.

### Slice 7 — Pipeline schema reconciliation + export surface

Per agent-orchestration §5 + §10:
- Pipeline schema as `## Pipeline` block on epic body
- Reconciliation check rules (`pipeline-step-missing`, `pipeline-agent-mismatch`, etc.)
- `aiwf export-cycles` verb with stable schema
- Schema-stability drift tests as public-API discipline

Depends on Slice 6.

### Slice 8 — Cycle skill family + `tdd_style:` hint (opt-in)

Optional evolution; ships only when real friction surfaces:
- `tdd_style:` free-form frontmatter field on milestone
- Cycle skill dispatches on `tdd_style` (Detroit, London, property — kernel doesn't enumerate)
- New cycle skills (`wf-bdd-cycle`, `wf-property-cycle`) ship **only when a real milestone needs them**

No kernel changes. Independent timing; can land any time after Slice 4.

### Dependency summary

```
1 ──────────────────────────────────────────► 8 (any time after 4)
   │
   └──► 2 ──► 3 ──► 4 ──► 5 ──► 6 ──► 7
```

Slices 1–4 give most of the practical benefit (creation chokepoint, persistent findings, AC decoupling, evidence audits). Slice 5 bridges to subagent execution. Slices 6–7 are the substrate engineering. Slice 8 is opt-in style support.

---

## 10. What's deliberately deferred

Each item is implementable within the rest of the architecture without structural changes. Each is deferred until concrete friction surfaces.

### 10.1. T-NNNN test artifact entity

**Deferred.** The cycle-evidence findings (§5) cover the integrity concerns without a new entity kind. Standing-test health (a test going `red` between milestones) is a CI concern that doesn't need first-class kernel modeling. Tests are typically composed by exactly one milestone — the same compositional reason ACs are namespaced sub-elements rather than a kind of their own. Standing cross-milestone tests are the motivating exception, and not common enough yet to justify the entity surface.

**On-ramp.** Add `T-NNNN` as 8th kind with status FSM, `verifies:` field, integrity flags (`immutable_during_milestone`). Existing `## Testing` sections become candidate references. The data model accommodates it cleanly.

### 10.2. TDD discipline as a kernel enum

**Deferred.** Discipline is a skill concern. The single-axis `tdd:` field (E-16) is sufficient strictness control; the `tdd_style:` free-form hint (Slice 8) handles dispatch when needed.

**On-ramp.** Promote `tdd_style:` from free-form to closed-set (e.g., `detroit | outside-in | property | hybrid`). Add per-discipline cycle skills. Add a `discipline-shipped-mismatch` audit if drift becomes a real concern.

### 10.3. Cycle skill family expansion (`wf-bdd-cycle`, `wf-property-cycle`)

**Deferred.** Ship `wf-bdd-cycle` when a real outside-in milestone needs it. Ship `wf-property-cycle` when a real property milestone needs it. Don't pre-author skill content for hypothetical disciplines.

**On-ramp.** Each new cycle skill is a markdown file in the rituals plugin; no kernel changes required.

### 10.4. Structured `success_criteria` + `carrier_acs` on epic

**Deferred.** Epic Success criteria currently live as prose. No empirical "criterion silently dropped" failure has surfaced. Pipeline declarations (Slice 7) provide structured per-epic work-shape; if criterion-coverage becomes a separate concern, address then.

**On-ramp.** Add `success_criteria: [{id, title, carrier_acs}]` array to epic frontmatter. Add `epic-criterion-uncovered` audit. Wire `aiwfx-wrap-epic` to verify coverage.

### 10.5. Epic mode (`planned | exploratory`)

**Deferred.** Nothing currently gates on it.

**On-ramp.** Add `mode:` field to epic frontmatter. Wire epic-criterion-coverage audit timing per mode.

### 10.6. `tdd.strict: true` config to escalate evidence findings to errors

**Deferred.** Ship warnings only initially. Add `tdd.strict: true` if dogfooding shows people ignoring the warnings.

**On-ramp.** Add field to `aiwf.yaml` schema; the audit rules already conditionalize severity on a config lookup.

### 10.7. Stronger mechanical test-evidence audits

**Deferred.** The cycle-commit and batch-promotion audits, combined with the wrap-skill's missing-section prompt, are this proposal's coverage of test-work evidence. If practical use shows the human review at wrap is insufficient, two on-ramps are available without breaking the kernel surface:

- **`ac-met-without-testing-section` audit.** Extend the body parser to check for `#### Testing` subheadings under `### AC-N`. Fires as a warning when an AC at met under `tdd: required` has no section or an empty one. Spoofable in the limit (write `#### Testing\nTODO`) but raises the cost via prose commitment.
- **`--tests-output <path>` flag.** Writes a test report path on the met commit; kernel sanity-checks for existence and minimum size. Raises the lying-cost meaningfully but introduces stack-specific report-format assumptions.

Pick one (or neither) when the wrap-prompt's effectiveness is evaluated.

### 10.8. Stack-aware test-path detection

**Deferred.** No automatic detection. The `tests-deleted-in-milestone` heuristic is opt-in via `aiwf.yaml.tdd.test_globs:`; absent config = silent rule.

**On-ramp.** If projects find configuring globs annoying, an `aiwf init` stack-detection pass (per the `policy-model.md` §"Stack discovery" pattern) could populate sensible defaults from manifest files (`go.mod`, `Cargo.toml`, etc.).

### 10.9. Empty `acs[]` policy

**Deferred without commitment to a stance.** The current implementation tolerates empty `acs[]` on milestones; doc-sweep and refactor milestones use this. Whether to add a strict-mode requirement that every milestone has at least one AC is a future call; not on this synthesis's path.

---

## 11. Worked example — what implementing a milestone looks like

A small payments-team milestone implemented end-to-end. Discipline: Detroit-classical (the practitioner's choice; kernel agnostic).

**Plan-time.**

```bash
aiwf add milestone --epic E-08 --tdd required --title "Empty cart guard"
# Creates M-066 with tdd: required.

aiwf add ac M-066 --title "POST /checkout with empty cart returns 422"
aiwf add ac M-066 --title "POST /checkout with empty cart returns explanatory error message"
# Creates M-066/AC-1 and M-066/AC-2. No tdd_phase seeded (Slice 3 removed that).
```

**Implementation (one cycle per AC, AC-1 first).**

```bash
# RED: write the failing test.
# (Practitioner edits tests/acceptance/checkout_test.go to add TestEmptyCart_Returns422.)
git commit -m "test(checkout): add failing TestEmptyCart_Returns422

aiwf-entity: M-066/AC-1
aiwf-actor: human/peter
"

# GREEN: write the implementation.
# (Practitioner edits handler.go.)
git commit -m "feat(checkout): empty-cart guard returns 422

aiwf-entity: M-066/AC-1
aiwf-actor: human/peter
"

# REFACTOR (optional): clean up.
git commit -m "refactor(checkout): extract emptyCartError helper

aiwf-entity: M-066/AC-1
aiwf-actor: human/peter
"

# Edit AC-1's body to fill in the ## Testing subsection (one-line note or
# a list of test names, depending on what the practitioner finds useful).

# Promote to met.
aiwf promote M-066/AC-1 met
```

`aiwf history M-066/AC-1` now shows: add commit, three work commits with the AC trailer, met commit. The cycle-evidence audits all pass:
- `ac-met-without-cycle-commits`: 3 work commits in lineage; doesn't fire.
- `ac-batch-promotion`: only AC-1 in this commit; doesn't fire.

**AC-2 follows the same pattern.** `aiwf promote M-066/AC-2 met`.

**Wrap-time.**

```bash
# Phase A (non-interactive):
# - aiwf check produces zero findings.
# - All ACs at met. Build green.
# - Both ACs have ## Testing subsections; missing-section list is empty.
# - Spec sections finalized (Work log, Validation, Deferrals, Reviewer notes).
# - No blocking findings, no missing sections. Phase B skipped.

# Phase C (commit + push gates as today):
git commit -m "chore(M-066): wrap milestone — empty cart guard

aiwf-verb: promote
aiwf-entity: M-066
aiwf-to: done
aiwf-actor: human/peter
"
aiwf promote M-066 done
git push
```

Compare to the old flow: same number of git commits at the work level (the `aiwf promote --phase` verb stops contributing four ceremony commits per AC). No new flag friction at met-time.

**The cheat path, for contrast.** A practitioner trying to fake the discipline:

```bash
# Hand-edit milestone frontmatter to flip both ACs to met in one commit.
git commit -m "(intentionally batch-met both ACs)"

# The next aiwf check produces three findings:
# - ac-met-without-cycle-commits for AC-1: no work commits in lineage.
# - ac-met-without-cycle-commits for AC-2: no work commits in lineage.
# - ac-batch-promotion: 2 ACs in same SHA.
#
# Phase A also lists both ACs as missing ## Testing sections.
# aiwfx-wrap-milestone Phase B walks them with the human:
# either resolve via fix commits, waive with ADR, or defer to a gap.
```

The cheat doesn't get blocked, but it produces multiple open findings + a missing-section list that all have to be triaged before wrap. The discipline is *visible*, not enforced. Resolving each finding requires either (a) doing the actual work and pointing at the fix commit, (b) opening a gap to defer, or (c) waiving with an ADR explaining why the rule's verdict is wrong here.

---

## 12. Open questions

Items still open at synthesis time. Each needs to be pinned at implementation time but doesn't block this document.

1. **Test-globs default templates.** If §10.8 is added later, what's the shipped default per detected stack? The `policy-model.md`-style stack-discovery list is a good starting point but each language's conventions are debatable. Pick at on-ramp time.

2. **Hard-deprecation timeline for `aiwf promote --phase`.** One minor release of soft-deprecation, then removed. Specific version numbers picked at Slice 3 ship time.

3. **Whether `## Testing` per-AC subsection becomes structured later.** No current need. If projects start treating test-name lists as references the kernel should validate, promote then.

4. **Whether the wrap skill's missing-`## Testing` prompt should also be a `aiwf check` rule.** Currently scoped as skill-only (Phase A surfaces; kernel doesn't fire). If practitioners ignore the wrap prompt, promote to a check finding (`ac-met-without-testing-section`, §10.7's first on-ramp). Not in the synthesis; future judgment call.

5. **Disposition of the `aiwf-tests:` trailer keyword.** Currently legacy (parser reads, no writer, no audit). If no consumer surfaces, formal deprecation in a future release. If a cycle skill or external consumer adopts it for informational use, retain.

---

## 13. Summary

The aiwf TDD discipline architecture, post-synthesis:

- Acceptance criteria are **behavioral commitments** with a status FSM (`open → met | deferred | cancelled`). They do not carry phase information.
- Tests live in the project's stack-native conventions. The kernel never tries to identify "what is a test file."
- The TDD posture (`tdd: required | advisory | none`) is a **strictness knob** on the milestone.
- Discipline (Detroit, London, property, contract-first) is a **skill concern**, not a kernel enum.
- **Two cycle-evidence audits** walk commit-trailer history: AC met without work commits in lineage, and batched-met across ACs in one SHA. Both warnings under `tdd: required`; no hard blocks.
- **F-NNN findings persist** as the HITL surface; the human triages at wrap.
- **`## Testing` per-AC body section** is the practitioner's primary evidence record. Free-form prose, optional at the kernel level, prompted by the wrap skill.
- **Wrap A/B/C** is a skill-only restructure of `aiwfx-wrap-milestone`. Phase A includes the missing-section list; Phase B walks it with the human.
- **Agent-orchestration substrate** (subagent execution, capability registry, pipeline schema, cycle envelope) lands after Slices 1–5 to inherit a clean evidence-audit surface and address G-067's within-cycle drift directly.

Seven entity kinds (six preserved + F-NNN). Two new audit codes for AC cycle-evidence + one for test-integrity heuristic + one for finding-gated closure + one standing-rule visibility surface + one resolved-without-fix soft check. No new mandatory verb flags. `tdd_phase` field, `--phase` verb, and `acs-tdd-audit` rule soft-deprecated through one minor release; `aiwf-tests:` trailer remains as parser-recognized informational metadata.

The cheating attractor is closed by making cheats *visible*, not by adding hard gates. The kernel stays small; the skills carry the discipline; the human triages at wrap.

---

## Appendix A — CLAUDE.md update recommendations

CLAUDE.md updates ride with the slice that makes them true. Commitments are not amended speculatively.

### A.1. Slice 1 (E-16, `--tdd` chokepoint)

- Operational definition of `tdd: advisory` propagated to CLAUDE.md (currently only in `aiwfx-start-milestone`).
- Note that `tdd:` is now mandatory at create-time (with project-default fallback); existing milestones grandfathered.

### A.2. Slice 2 (F-NNN entity + archive)

- **Commitment #1 — entity count.** "Six entity kinds" → **"Seven entity kinds"** (add finding). Updated list: *"epic, milestone, ADR, gap, decision, contract, finding."*
- **Commitment #1 — id format list.** Add `F-NNNN` to the stable-id list.
- **New commitment** for findings-as-entities: *"`aiwf check` produces persistent finding entities (F-NNNN) with severity, status, and reconciliation across runs. Severity is a property of the rule code; gating is a property of the runner. Waivers require a waiving ADR; revocation requires a new ADR."*
- **New commitment** for uniform archive convention: *"Terminal-status promotion moves the file to `<kind>/archive/` in the same atomic commit. The id-resolver scans both directories. References stay valid; archived entries remain queryable via `aiwf history`."*

### A.3. Slice 3 (AC model revision)

- **Commitment #8 — audit rule.** "AC `met` requires `tdd_phase: done`" is **removed**. Replaced with: *"Acceptance criteria are namespaced sub-elements of milestones with a status FSM (`open → met | deferred | cancelled`); the relationship to tests is implicit via the trailer history of commits in the AC's lineage and explicit (when present) via the `## Testing` body section. The kernel does not track per-AC TDD phase. Cycle skills (advisory) drive red/green/refactor practice."*

### A.4. Slice 4 (cycle-evidence audits + finding-gated closure)

- **"What's enforced and where" table.** Add rows:
  - `ac-met-without-cycle-commits` → CI test (warning under `tdd: required`)
  - `ac-batch-promotion` → CI test (warning under `tdd: required`)
  - `tests-deleted-in-milestone` → CI test (opt-in via test_globs)
  - `findings-block-met` → kernel verb refusal at promote-time (override via `--force --reason`)
  - `ac-has-open-findings` → CI test (visibility)
- **New commitment** for cycle-evidence visibility: *"AC `met` is gated by trailer-derived evidence audits surfaced as findings. Hard blocks are reserved for structural integrity; discipline cheats are visible warnings, not refusals. The kernel does not identify test files; it walks trailer history and surfaces audit codes the practitioner reviews at wrap."*

### A.5. Slice 5 (wrap A/B/C)

- No CLAUDE.md changes (skill-only restructure; no kernel commitment shift).

### A.6. Slice 6 (agent-orchestration substrate)

- **Commitment #9 — provenance × runner mode.** Add the four composition rules from agent-orchestration §2 P9.
- New commitments per agent-orchestration §13's CLAUDE.md updates.

### A.7. Slices 7–8

- Per agent-orchestration §13 and Slice 8's opt-in nature; specific updates pinned at landing time.

### A.8. What does NOT change

For clarity, several commitments and sections remain untouched by this synthesis:

- Commitments #2 (stable ids), #3 (`aiwf check` as pre-push hook), #4 (`aiwf history` reads `git log`), #5 (marker-managed framework artifacts), #6 (layered location-of-truth), #7 (one commit per mutating verb).
- "What is *not* in scope" — no items added or removed. The synthesis stays within the existing scope.
- Engineering principles (KISS, YAGNI, no half-finished implementations, errors-as-findings, framework correctness independent of LLM behavior, AI-discoverability, auto-completion friendliness).
- Go conventions, test conventions, error handling, CLI conventions, commit conventions, release process, dependencies, naming, type design.
- "Designing a new verb" rule itself.
- Skills policy ADR-0006.
