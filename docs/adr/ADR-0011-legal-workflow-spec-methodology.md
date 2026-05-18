---
id: ADR-0011
title: Legal-workflow spec methodology
status: accepted
---

# ADR-0011 — Legal-workflow spec methodology

> **Date:** 2026-05-18 · **Decided by:** Peter Bruinsma (human/peter)

## Context

aiwf is a kernel that pins planning-tree invariants through a small set of mutating verbs. Each verb's pre/postconditions, each kind's FSM transitions, and the cross-verb sequencing rules that make up a "legal workflow" are currently expressed in three half-overlapping places:

- `internal/entity/transition.go` carries the per-kind FSM tables.
- `internal/policies/` carries mechanical invariants enforced as Go tests.
- `internal/checks/` carries the runtime check rules that surface as findings.

Plus prose-only sources: ADRs, `docs/pocv3/design/design-decisions.md`, `CLAUDE.md`, skills in `.claude/skills/` and the rituals plugin, and `aiwf <verb> --help` output. No source today expresses the **closed-set frontier between legal and illegal verb sequences** as a single machine-readable table. The verbs work, the FSM is right *as far as we know*, but there is no chokepoint that catches the failure mode "the implementation silently permits a workflow it shouldn't" or "the implementation silently rejects one it should permit."

E-0033 commits to closing that gap with a canonical spec and per-cell tests. Before any catalog or test-writing work begins, this ADR pins the **methodology** for how the spec is built and maintained — independently of what the spec ends up saying.

The prior epic E-0031 attempted the same closure with a prose-shaped catalog (`docs/pocv3/design/legal-workflows.md`) plus a hand-coded fuzz harness. It was cancelled (2026-05-18) because the "spec" couldn't drive mechanical execution: each downstream milestone (test harness, citation symmetry, fuzz harness) papered over the prose by hand-coding what should have been spec-driven. This ADR exists to prevent that failure mode from recurring.

**Alternatives considered:**

1. **Spec-from-impl extraction.** Auto-derive the legal-workflow table by introspecting `internal/entity/transition.go`, the Cobra verb tree, and the check-rule registry. Rejected: tests built against a spec that was *itself* extracted from the impl are tautological — they cannot find a divergence because there is no second source to disagree. Rubber-stamps bugs.
2. **Impl-from-spec generation.** Author the spec as the single source of truth; generate the FSM tables, verb validators, and check-rule entries from it. Rejected as premature surgery: the kernel's hand-written Go FSM and verb code is established and works; generating it would be a large refactor for marginal benefit. Worth revisiting only if a future epic demonstrates concrete drift that simpler methodology can't catch.
3. **Random / model-based fuzz testing only.** Generate random verb sequences and observe whether the binary's behavior matches a hidden internal model. Rejected as the *only* methodology: without an explicit model, divergences can't be classified, and the failure mode "we never enumerated this cell" is silent. Fuzz becomes informative once a canonical spec exists — a worthwhile follow-up after E-0033 wraps, not a substitute for it.
4. **Independent spec + agreement tests (adopted).** The spec is human-authored; tests cross-check the implementation against it cell-by-cell. The cross-check is the safeguard. Cost is duplication — the spec and the impl say the same thing twice — but that's the point; duplication is what makes the cross-check informative.

## Decision

The legal-workflow specification is built and maintained according to the following seven commitments.

### Independence

The spec is **human-authored**, not generated from the implementation. Spec-from-impl extraction is rejected (alternative 1 above): a test suite built against a spec that was itself derived from the binary cannot find a divergence between the two because there is no second source. The independence is the whole point.

This applies in both directions during construction. During the build (M-0121, M-0122, M-0123) and during maintenance (every PR thereafter), the spec is reasoned about as a separate artifact — it may agree with the impl, it may disagree, but it is never *defined by* the impl.

### Three-pass methodology

The spec is built in three sequential passes, owned by M-0121, M-0122, M-0123 respectively:

- **Pass A — Audit existing surfaces.** Walk every place the repo already states a legality rule (transition tables, policy tests, check rules, ADRs, design docs, `CLAUDE.md`, skills, `--help` text) and extract them into a draft catalog with citations.
- **Pass B — First-principles derivation.** Independently of Pass A's output, derive the legal-workflow surface from the entity model: lifecycles, ownership relations, cross-entity invariants. The independence is load-bearing — if Pass B is informed by Pass A, the cross-check loses value.
- **Pass C — Reconcile.** Walk both catalogs cell-by-cell. Agreement entries land in the spec. Audit-only entries, first-principles-only entries, and conflicts each surface as explicit decisions captured in their own decision entities. The output is the canonical Go table.

Per-PR maintenance after E-0033 wraps follows a lighter version of the same shape: when a verb, kind, or finding code is added or changed, the same PR updates the spec and at least one test exercising the new/changed cell.

### Canonical form

The spec lives as **Go data structures** under `internal/workflows/spec/` (exact package name and schema are settled in M-0123, when the catalogs are in front of us). The catalogs from Pass A and Pass B are **working artifacts** — markdown documents under `docs/pocv3/design/` — but they are not the spec. Once Pass C produces the Go table, the markdown catalogs become evidence of how the spec was constructed, not the source of truth for what it says.

Go was chosen over markdown / YAML / DSL for three reasons. First, the test harness consumes Go: every cell-coverage test reads `Rules()` and exercises the binary against an entry, so the spec must already be in the same language as the tests. Second, AI-discoverability prefers typed Go data over markdown for closed-set tables: tab-completion, godoc, refactor tooling, and the existing `internal/policies/` test pattern all expect Go. Third, a single source of truth simplifies the drift policy below.

### Cell-coverage commitment

Every entry in the spec table has **paired positive and negative tests** under `internal/policies/`. Positive tests (M-0124's scope) build a fixture tree that satisfies the cell's preconditions, exercise the verb on a real `aiwf` binary, and assert the expected outcome and post-state. Negative tests (M-0125's scope) construct fixture trees that violate the cell's preconditions and assert the expected rejection (exit code + finding code or named error).

A meta-test (also under `internal/policies/`) walks the spec table and asserts every cell has at least one matching test. Missing coverage fails CI. The coverage commitment is closed-loop: adding a spec entry without a paired test fails the meta-test; adding a test without a spec entry is mechanically allowed but reviewer-flagged (it's evidence of either a missing spec entry or a redundant test).

### Drift policy

The spec is **closed-set against the implementation**. A drift test under `internal/policies/` asserts:

- Every (kind, state) pair the FSM tables in `internal/entity/transition.go` recognize has at least one corresponding rule in the spec.
- Every top-level Cobra verb is referenced by at least one rule.
- Every `aiwf check` finding code that pertains to verb-sequence legality is referenced by at least one illegal-cell rule (advisory-only codes are listed in an explicit exemption block with rationales).

Failure of any of these is a hard CI block. The impl cannot grow a new verb, kind, state, or workflow-legality finding without the spec growing in the same PR. This is the load-bearing maintainability property: without it, the spec rots and the cross-check progressively loses value.

### Scope

This methodology covers **kernel-verb workflows** at three layers: per-entity FSM transitions, per-verb pre/post conditions beyond FSM (cross-entity invariants), and cross-verb sequence legality. Out of scope:

- **Branch choreography** — which branch each verb is legal on. That layer is E-0030's scope; it deals with git state, not entity state, and its test fixture shape is different.
- **Rituals-plugin orchestration** — skills like `aiwfx-start-milestone` compose kernel verbs into named sequences. Per the kernel principle that *"framework correctness must not depend on the LLM's behavior,"* only the kernel can be mechanically verified. Skill-level coverage remains advisory.
- **Random / model-based fuzz testing.** A worthwhile safety net beyond the spec table but a separate later concern.

### Future-change handling

When the implementation gains a new verb, kind, state, finding code, or workflow constraint, the **same PR** must update the spec and add at least one test exercising the new/changed cell. The drift policy enforces this mechanically: a PR that touches the FSM, the Cobra tree, or the check-rule registry without a corresponding spec update fails CI.

This is the maintainability commitment. Without it, the methodology degrades over time as small changes slip past the spec one at a time. The chokepoint is at the *moment of impl change*, not at periodic audit time.

## Consequences

- E-0033 produces a Go spec table that downstream epics can depend on. Subsequent kernel work referring to "legal workflows" can cite specific spec cells rather than re-litigating the legality question.
- Every future kernel PR that touches verb behavior, FSM transitions, or workflow-legality findings carries spec changes and at least one cell test. Estimated cost: a small constant overhead per PR (~10–30 lines), in exchange for the cross-check guarantee.
- The two markdown catalogs from Pass A and Pass B become permanent evidence artifacts. They are not deleted at wrap; they are referenced from the spec entries they motivated.
- The methodology does not commit to a specific Go schema for the spec table — that's M-0123's design decision, made once the catalog content is concrete. This ADR is intentionally one level higher: it pins the shape of the process, not the shape of the data.
- The kernel acquires a new chokepoint (the drift policy) that future contributors — human or AI — must learn. Discoverability is via `internal/policies/` (the existing pattern) and via this ADR.

## References

- [E-0033 — Pin legal kernel-verb workflows mechanically](../../work/epics/E-0033-pin-legal-kernel-verb-workflows-mechanically/epic.md), the epic this ADR ratifies methodology for.
- ADR-0010 — Branch model. Defines the branching surface that E-0030 builds on; this ADR's scope explicitly excludes layer 4 (branch choreography) for that reason.
- [`CLAUDE.md`](../../CLAUDE.md), §"Engineering principles" — *"framework correctness must not depend on the LLM's behavior"* is the upstream principle that motivates having a mechanical spec at all.
- The cancelled E-0031 (`work/epics/E-0031-pin-legal-workflows-composition-and-branch-choreography-mechanically/epic.md`) — its failure mode is what this methodology exists to prevent.
