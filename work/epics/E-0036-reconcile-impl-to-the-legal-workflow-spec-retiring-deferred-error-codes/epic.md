---
id: E-0036
title: Reconcile impl to the legal-workflow spec, retiring deferred error codes
status: active
---

## Goal

Make E-0033's legal-workflow spec a *fully* verified source of truth by reconciling the kernel impl to it. Concretely: retire the `deferredImplErrorCodes` IOU list so every illegal cell the spec names actually **fails-verified** through the binary, and every legality-pertinent finding code is provably referenced by a spec rule (the bidirectional-completeness guarantee). The enabling deliverable is a **typed `CodedError` pattern** that lets verb-time refusals carry a first-class, structured error code — `errors.As`-able for the JSON envelope and visible to the AC-5 spec↔impl scanner, mirroring the existing `check.Finding{Code}` shape.

## Context

E-0033 produced `internal/workflows/spec/rules.go` — a declarative table of every `(kind, state, verb)` cell marked Legal/Illegal, each illegal cell naming an `ExpectedErrorCode`, pinned by three drift policies (AC-5 spec↔impl, M-0124 positive driver, M-0125 negative driver). The spec deliberately ran *ahead* of the impl: it names error codes the verbs don't emit yet, parked in the `deferredImplErrorCodes` allowlist (five codes). The allowlist is honest bookkeeping, but as long as it exists the spec's "verified source of truth for legal/illegal kernel workflows" claim carries five asterisks — five boundaries the spec asserts but the engine doesn't enforce as recognizable structured data.

The root cause is uniform: verbs emit refusals as `fmt.Errorf` *prose*, while the AC-5 scanner only recognizes a code as impl-resolved when it appears as a `Code: "..."` composite literal. No typed coded-error pattern exists in the codebase today (`ExpectedErrorCode` is a spec-side field only). Two adjacent loose ends ride along: AC-5's *fourth* drift arm — every legality-pertinent impl code referenced by ≥1 spec rule — was deferred (G-0145), and one finding code's name (`gap-resolved-has-resolver`) drifted from the current gap FSM vocabulary (`addressed`/`wontfix`) (G-0144). This epic is the impl-reconcile follow-up to E-0033; it builds directly on that spec and its drift policies.

## Scope

The **reviewed full reconcile of the legality surface** — "reviewed" because the act of reconciling each cell is also the act of *validating* its decision; a cell that fails review is transformed or removed, not blind-emitted (precedent: G-0140, carved out below for exactly this reason).

| Gap | What it lands | Decision |
|-----|---------------|----------|
| G-0142 | Typed structured `fsm-transition-illegal` error from `entity.ValidateTransition` — the **foundation pilot** for the `CodedError` pattern (most-referenced code: every terminal cell) | — |
| G-0139 | Verb-time `cancel` refusal on non-terminal children/ACs; emits `epic-cancel-non-terminal-children` / `milestone-cancel-non-terminal-acs` via the new pattern; un-skips the M-0124/M-0125 driver cells | D-0003, D-0004 |
| G-0141 | `authorize-kind-not-allowed` Phase 2 — behavior already shipped (M-0125, `d5abcf51`); only structured-code emission via the new pattern remains | D-0007 |
| G-0145 | Legality-pertinent finding-code classifier — closes AC-5's deferred fourth arm; **the keystone of the bidirectional-completeness goal** | — |
| G-0144 | Rename `gap-resolved-has-resolver` to match the current gap FSM, atomically across impl, spec, hint table, and fixtures (gated on a small pre-decision) | new D-NNNN |
| G-0143 | Scope three-edge reachability (`parent` fwd + composite-id + `discovered_in` reverse) with verb-time out-of-scope refusal — **greenfield: no reachability enforcement exists today** (`internal/scope/` is FSM-only) | D-0006 |

Plus the **foundation**: the typed `CodedError` pattern itself, and the AC-5 scanner extension that recognizes it.

## Out of scope

- **G-0140 / D-0005 (`--evidence` flag).** Carved out by operator decision. D-0005 mechanizes a *process/authorship* gate, not a state-machine legality rule; its illegal condition (an AC reaches `met` without a resolvable test symbol) is a **gameable proxy** — the kernel can verify the symbol exists, not that it tests the claim (D-0005's own body admits this). It therefore cannot deliver "fails-verified," only the appearance of it, which is the cheating-attractor [`docs/research/14-two-walkbacks-substrate-and-philosophy.md`](../../../docs/research/14-two-walkbacks-substrate-and-philosophy.md) identifies. D-0005 gets reconsidered against that walk-back in a separate conversation — likely transforming into a *finding* (`ac-met-without-recorded-evidence`) rather than a verb-time hard-reject. `ac-evidence-missing` is the one entry that **stays** in `deferredImplErrorCodes` at this epic's close.
- **Converting non-legality verb errors to `CodedError`.** YAGNI — only spec-referenced (legality-pertinent) errors need first-class codes. The other ~30 verb errors stay `fmt.Errorf` until something references them.
- **New spec-schema expressivity.** M-0123's preconditions-only expressivity holds; if reconciliation surfaces a rule that can't be a state predicate, that's a decision against the spec, not a second schema.
- **Branch choreography (layer 4).** E-0030's surface.

## Constraints

- **The `CodedError` pattern is option (a)**: the code is carried *as data* (an `errors.As`-able typed error exposing the code), mirroring `check.Finding{Code}` so there is one mental model, not two parallel sources of truth. (b) constant-only and (c) permissive-scanner were rejected — both leave the code trapped in prose.
- **Reviewed reconcile, not blind reconcile.** Re-confirm each cell's decision still holds before emitting its code. Blind-emitting cements spec mistakes; the reviewed pass is what caught G-0140.
- **AC promotion requires mechanical evidence** (CLAUDE.md) — every AC in this epic promotes to `met` only behind a Go test under `internal/policies/` or a kernel finding-rule that fails if the AC's claim breaks. Applies even though some milestones may be `tdd: none`.
- **No `//nolint` without rationale; statically-linked, `CGO_ENABLED=0`; race detector on CI.** Standard repo gates.

## Success criteria

Observable at epic close (not tests):

- `deferredImplErrorCodes` contains **only** `ac-evidence-missing`; every other entry is retired and its code appears as a structured `Code` in non-test `internal/` source.
- Every illegal cell whose `ExpectedErrorCode` is listed in the *Scope* table fails-verified through the binary: the corresponding M-0124/M-0125 driver cells are un-skipped and their `ac2KnownImplGaps` entries removed.
- A typed `CodedError` carries the structured code for every legality-pertinent verb refusal, and the AC-5 scanner recognizes it as impl-resolved.
- AC-5's fourth arm is live: every legality-pertinent finding code is provably referenced by ≥1 spec rule (G-0145).
- `gap-resolved-has-resolver` is renamed to match the gap FSM, atomically across impl, spec, hint table, and fixtures.
- `aiwf authorize` and `aiwf cancel` refuse the spec's illegal cases at verb-time with the structured codes; `aiwf authorize` enforces D-0006's three-edge scope reachability for authorized-agent actions.
- Every gap listed in the *Scope* table is `addressed`; every decision listed in *Decisions implemented* is `accepted`.

## Decisions implemented

Ratify at epic start (promote `proposed → accepted`; independent of impl sequencing per CLAUDE.md decision philosophy): **D-0002** (contract `accepted→rejected`, codifies the already-wired edge), **D-0003**, **D-0004**, **D-0006**, **D-0007**. A new **D-NNNN** is authored for the G-0144 rename (downstream JSON-consumer caveat). **D-0005 stays `proposed`** — carved out (see *Out of scope*).

## Open questions

1. **Does G-0143 split into its own epic?** It is the only greenfield item and the heaviest (no reachability enforcement exists today). *Resolution:* size it during `aiwfx-plan-milestones`; split if it exceeds ~one milestone of impl or if its verb-time-refusal design warrants its own ADR. *Lean:* keep in, sequence last.
2. **`CodedError` shape** — single struct with a `Code` field, or an `interface { Code() string }` with small concrete types per code family? *Resolution:* settled in the foundation milestone's ADR. *Lean:* interface (matches G-0142's proposed `FSMTransitionError{}.Code()`).
3. **G-0144 rename target** — `gap-addressed-has-resolver` or other? *Resolution:* settled in the pre-rename D-NNNN.
4. **G-0145 classifier mechanism** — structural `Class` field on `Finding` vs. a hand-maintained allowlist. *Resolution:* settled in G-0145's milestone. *Lean:* `Class` field (structural property of the code, not of the drift policy).

## Risks

- **Reconcile cements spec mistakes if done blind.** Mitigated by the reviewed-reconcile constraint; G-0140 already demonstrates the catch working.
- **`CodedError` refactor scope-creeps to all verb errors.** Mitigated by the out-of-scope line limiting conversion to legality-pertinent errors.
- **G-0143 greenfield work balloons.** Mitigated by open question 1 — split to its own epic if it exceeds scope.

## Milestones

Decomposed and allocated via `aiwfx-plan-milestones`. G-0141 folded into the foundation milestone (its remaining work is the same transformation as the G-0142 pilot — convert an existing `fmt.Errorf` legality error to `CodedError`). G-0143 kept as one milestone here, not split to its own epic (D-0006 supplies the design, so no new ADR); revisit the split at `aiwfx-start-milestone` only if it exceeds ~one milestone of impl.

| Id | Title | Closes | Depends on |
|----|-------|--------|-----------|
| [M-0138](M-0138-introduce-typed-codederror-convert-existing-unstructured-legality-errors.md) | Introduce typed `CodedError`; convert existing unstructured legality errors | G-0142, G-0141 | — |
| [M-0140](M-0140-classify-legality-finding-codes-close-ac-5-bidirectional-arm.md) | Classify legality finding codes; close AC-5 bidirectional arm | G-0145 | M-0138 |
| [M-0139](M-0139-refuse-cancel-of-parents-with-non-terminal-children-acs-via-coded-errors.md) | Refuse `cancel` of parents with non-terminal children/ACs via coded errors | G-0139 | M-0138 |
| [M-0142](M-0142-rename-gap-resolved-has-resolver-to-match-the-gap-fsm-vocabulary.md) | Rename `gap-resolved-has-resolver` to match the gap FSM vocabulary | G-0144 | — |
| [M-0141](M-0141-enforce-three-edge-scope-reachability-at-verb-time.md) | Enforce three-edge scope reachability at verb-time | G-0143 | M-0138 |

**M-0138 is the keystone** — M-0139/0140/0141 emit their codes through its pattern. Recommended execution order: **M-0138 → M-0140** (stand up the legality-classifier *chokepoint* early, so later code-adding milestones must satisfy it) **→ M-0139 → M-0141** (greenfield, heaviest, last). **M-0142** is independent — slot it after M-0140 so the rename updates the classified set in one pass. Only the M-0138 edges are hard; the rest is soft ordering.

## What this epic deliberately does *not* do

- It does not implement D-0005's `--evidence` gate (carved out; gameable proxy, revisited separately).
- It does not convert non-legality verb errors to `CodedError`.
- It does not grow the spec's schema expressivity.
- It does not promote itself to `active` or break itself into milestones — that's `aiwfx-start-epic` + `aiwfx-plan-milestones`.
