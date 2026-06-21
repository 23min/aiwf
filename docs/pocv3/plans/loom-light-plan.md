---
title: "loom-light × aiwf — a verification value-gate"
status: proposal
audience: aiwf maintainers (you), to seed spec'ing
one-liner: >
  Add a mechanical value-gate over code correctness to aiwf by composing an
  existing verifier through a small, standalone engine ("loom-light"); aiwf
  consumes it as a contract kind. The differentiator is not "runs a verifier"
  but catching weak, LLM-authored claims that pass anyway.
---

# loom-light × aiwf — a verification value-gate

## 0. How to use this document

This is a **design proposal**, not a finished spec. It is written to be
*decomposed* into aiwf entities: the §4 milestone sketch maps to candidate
epics/milestones, §5 to ADRs, §6 to gaps. Where it describes aiwf internals it
is describing what already exists (the contract engine, the pre-push chokepoint,
findings); where it proposes new surface (flags, recipes, a `mode: verify`) it
says **proposed** so you don't mistake a suggestion for a fact.

It carries the deliberately **balanced** framing from the analysis that produced
it: every section names what this is *not*, and §6 is a real risk register, not
a formality. The narrow-adoption reality is stated up front rather than buried,
because a proposal you can spec from needs honest decision points more than it
needs enthusiasm.

**The recommended path, in one paragraph:** factor the verification capability
as a shared engine (a Go library) with two thin consumers — a minimal standalone
`loom` CLI for people who want verification with no workflow, and an aiwf
*contract kind* for people who also want governance. Ship v0 against **one**
verifier (Dafny), using its **native** annotation syntax (defer the `.lm`
surface). Make the headline feature the **vacuity check**, not the gate. Keep it
**opt-in per component**. Treat the `.lm` claim language as a separable, later
front-end, not a precondition.

---

## 1. Why

### 1.1 The gate-theory argument (aiwf's own terms)

aiwf executed a deliberate **philosophy walk-back**: from mechanical enforcement
of process (TDD-style discipline gates) to evidence-and-judgment with persistent
findings and human triage at wrap. The reason was the cheating attractor —
mechanical *process* gates over LLM behaviour are gameable, because the agent
controls the process signals the gate reads.

That walk-back was correct, and it left a hole. The robust complement to a
gameable process-gate is a **value-gate**: a property that is true or false *in
the values*, with no "did you follow the process?" question hovering above it.
`for-all L T, sum(L before T) = sum(L after T)` either holds for all values or it
does not; process-faking cannot make it hold. **aiwf currently has no value-gate
over code behaviour.** Its `check` kernel is a value-gate over the *planning
tree*; its contracts are value-gates over *data shape* (CUE / JSON Schema). The
behaviour of the code the planning tree describes is gated only by
evidence-and-triage. loom-light supplies the missing layer, at the layer where
it is robust (value) rather than the layer aiwf abandoned (process).

This is **not** a re-litigation of the philosophy walk-back. It is that
walk-back's logic completed: process-gate removed because gameable → value-gate
added because robust.

### 1.2 The field context (brief, and why it matters)

The verification bottleneck under agentic coding is now a mainstream concern: a
major quant shop publicly reversed 25 years of formal-methods skepticism on
exactly this basis, and spec-driven development tools became an industry-standard
category in 2025–26. Two things follow that the design must respect:

- **The workflow half is crowded and adopted.** SDD tooling (markdown-spec-in-git,
  plan→tasks→implement) has large, well-resourced incumbents. aiwf's *core*
  competes there. The verification value-gate is the part that is *not* in those
  tools — they gate by review and tests, and their acknowledged unsolved problem
  is spec↔code drift, which a re-checked mechanical gate addresses.
- **The verification half is barely adopted and mostly academic.** LLM+formal-
  verification systems are research prototypes; production formal methods are
  expert-driven with hand-written specs. So the opportunity is a *bridge* nobody
  adopted occupies — with the caveat in §6 that the bridge serves a narrow segment.

### 1.3 Why aiwf is the right host

Four properties aiwf already has are precisely what a verification gate needs:

1. **A validator-agnostic engine.** `contractverify` already runs a user-declared
   command, reads exit 0 / non-zero as accept / reject, and captures the tool's
   output into findings. From aiwf's view, a verifier is *just another validator
   command*. The integration seam exists.
2. **A chokepoint.** The pre-push hook is what makes any gate real. A standalone
   tool would have to reinvent this; aiwf has it.
3. **A containment loop.** Value-gates have one residual attack surface —
   *claim-weakening* (the LLM authors a vacuous or over-narrowed claim that the
   verifier discharges trivially). aiwf's findings + wrap-time human triage is
   exactly the apparatus to contain what cannot be mechanically closed. This is
   the deepest reason aiwf is the right home, not just a convenient one.
4. **Provenance and lifecycle.** A verification claim becomes a first-class entity
   with a status set, attribution, and git-backed history — turning "we ran the
   verifier" into "we have an auditable record of every claim, who authored it,
   and which component it gates." That is the value for the regulated segment.

### 1.4 The actual differentiator

Running a verifier is a one-line CI step the verification crowd already has.
The thing worth building is **vacuity / gaming detection**: catching the weak,
LLM-authored claims that pass anyway. This is the contribution that is *yours*
and is currently under-served — most LLM-spec literature frames weak specs as
*capability* failures (the model tried and couldn't), whereas the live problem
under optimization pressure is *endogenous* weakening (the same agent authors the
claim and is graded on passing it). The vacuity check is the front door of
loom-light, not an add-on.

### 1.5 Scope of value (honest)

This pays off on code with **crisp invariants** — ledgers, parsers, protocols,
state machines, allocators, authorization logic. It is high-cost-low-yield on
glue, IO, and UI. That is the 25-year formal-methods objection, and easy plumbing
does not dissolve it. The gate must therefore be **opt-in per component**, used
where invariants are sharp, with everything else left to evidence-and-triage as
aiwf already does.

---

## 2. What

### 2.1 loom-light, defined

A minimal capability with a single pipeline:

```
claims + code  →  lower to verifier  →  run verifier  →  lift results
                                                        →  vacuity-check (mutate claims, re-run)
                                                        →  emit findings
```

Zero workflow footprint. Its only hard dependency is the verifier itself.

### 2.2 The two-consumer architecture (the key shape decision)

```
            ┌──────────────────────────┐
            │   loom engine (library)  │   verifier-runner + lifter + vacuity check
            └──────────────────────────┘
                  ▲                ▲
                  │                │
   ┌──────────────────┐   ┌────────────────────────────┐
   │  `loom` CLI       │   │  aiwf contract kind         │
   │  (no workflow)    │   │  (binding + lifecycle +     │
   │  for the          │   │   provenance + chokepoint + │
   │  verification     │   │   triage)                   │
   │  crowd            │   │  for the governance segment │
   └──────────────────┘   └────────────────────────────┘
```

One engine; two thin consumers. This is **compose-don't-absorb applied to aiwf's
own relationship with loom**, which is aiwf's stated philosophy. It exists because
the workflow apparatus is a *tax* on the pure verification crowd (they did not
come for epics and scope-FSMs) but an *asset* to the governance segment. The
standalone CLI removes the self-inflicted barrier; the contract kind preserves the
governance value. (Decision in §5.)

### 2.3 The `.lm` claim surface — optional, later, containment-shaped

The bespoke claim language is positioned as a **front-end that lowers to a
verifier** — the intermediate-verification-language pattern (cf. Why3/WhyML,
Boogie, Viper). It is **not** required for v0: v0 uses the verifier's native
annotation syntax (inline Dafny), and `.lm` is a separable layer added only if
and when it earns its place.

When/if built, position it not as "a translator" (that category is decades old)
but as a **claim surface designed for containment**: a grammar where common
non-trivial claims (conservation invariants, bounded `∀/⇒`) are the natural thing
to write, and where vacuous shapes (`∀x. true`, over-narrowed antecedents) are
conspicuous or awkward. That makes the language do gate-theory work. Honest limit:
a grammar raises the bar and surfaces patterns; it does not eliminate gaming —
containment, not solution.

### 2.4 Non-goals (explicit)

- **Not a verifier.** Compose Dafny/Z3; never reimplement SMT or proof search.
- **Not a proof assistant.** Hard proofs needing human proof-engineering are out
  of scope; v0 targets properties a backend can discharge largely automatically.
- **Not multi-backend in v0.** One verifier. Abstraction over verifiers is a
  later question, not a v0 feature.
- **Not a platform.** No LSP, no package manager, no build system. The language,
  if built, is surface + lowering + lift-back, full stop.
- **Not a replacement for tests or review.** It is one gate among several, for the
  subset of code with crisp invariants.
- **Not mandatory.** Opt-in per component. A repo with zero verification contracts
  behaves exactly as aiwf does today.

---

## 3. How — integration design

### 3.1 Frame: an "Iteration I-Loom" in the shape of I1 (Contracts)

You have already shipped the template: Iteration I1 added contract bindings,
`contractverify`/`contractcheck`, recipes, and pre-push integration. I-Loom is the
same shape — a new contract *kind* whose validator is a verifier and whose verdict
is a proof outcome rather than a fixture pass/fail.

### 3.2 Data-model mapping (needs an ADR — see §5)

aiwf contracts are built around (validator, schema, valid/invalid **fixtures**)
with a verify pass (current fixtures) and an evolve pass (historical fixtures vs
HEAD schema). Verification has no "fixtures" in the schema sense. Proposed mapping:

| aiwf contract concept | verification meaning (proposed) |
|---|---|
| validator command | the verifier invocation (`dafny verify …`) |
| schema | the claim(s) — inline in code for v0, `.lm` later |
| `valid/` fixtures | the real source units that **must verify** |
| `invalid/` fixtures | known-bad implementations the claim **must reject** (regression guard) |
| verify pass | run the verifier on the bound units; all must discharge |
| evolve pass | re-run prior known-bad cases against current claims (drift guard) |
| reclassification | verifier-not-found / toolchain error collapses to one finding |

The `invalid/` reinterpretation is the elegant fit: it turns the existing
fixture dichotomy into a verification regression suite (the claim must keep
rejecting the bad implementations).

### 3.3 Consumer UX (grounded in existing aiwf verbs; new surface marked *proposed*)

1. Consumer has already run `aiwf init` (pre-push hook installed).
2. They write code carrying claims. **v0:** inline native verifier syntax
   (Dafny pre/postconditions, invariants), authored by them or the LLM host.
3. *(proposed)* `aiwf contract recipe dafny` scaffolds a verifier binding;
   `aiwf add contract --verifier dafny --fixtures src/ledger.dfy`.
4. `aiwf contract bind C-0001 M-0007` ties the claim to a milestone.
5. `aiwf check` (and the pre-push hook) runs the verifier on bound units.
   Failure blocks the push and surfaces as a finding:
   `src/ledger.dfy:42: error verify: postcondition may not hold — hint: …`.
6. The contract is a first-class entity (proposed→accepted lifecycle), so the
   *claim* is versioned, attributed, and visible in `history` / `render` /
   `status`.
7. A passing-but-weak claim is caught by the vacuity check (§3.5) as a finding,
   and routed to wrap-time triage — the containment loop.

The downstream mental model is unchanged: **verification is a contract kind that
runs a verifier.** Users already understand contracts, bindings, `check`, and the
gate.

### 3.4 Output → findings mapping

- **v0:** opaque passthrough. The engine already captures stdout/stderr; surface
  the verifier's raw output in finding details. Zero parsing, works day one.
- **later:** a per-verifier output parser that lifts counterexamples and
  "postcondition may not hold" locations into structured `file:line` findings.

### 3.5 The vacuity check (the differentiator) — must run at the *lowered* level

Mutate the claims, re-run the verifier, measure the **kill rate**: a strong claim
rejects most mutants (high kill rate); a vacuous/over-narrowed claim still
verifies after mutation (low kill rate) → emit a "weak claim" finding for triage.

Critical design constraint, especially once `.lm` exists: **mutate and measure on
the lowered form (what the verifier actually checks), then lift the finding back
to the surface the human reads.** If you mutate the surface syntax, weakness
introduced by the lowering itself slips through. In aiwf terms, the vacuity check
is *itself another validator* in the contract pipeline.

### 3.6 Severity and the soft/hard gate

Verifiers time out and wobble (Z3 resource limits). A flaky hard-gate that blocks
pushes is a UX hazard. Map verification outcomes onto aiwf's existing finding
**severities**: a "hard" mode treats failure/timeout as error (blocks push); a
"soft" mode treats them as warning (surfaces, does not block). This rides existing
machinery and is cheap to express. *(proposed: `severity: hard|soft` per binding.)*

### 3.7 Dependency handling

The verifier is a heavyweight external dependency (Dafny + Z3 + .NET) that the
*consumer* must install, which dents aiwf's "one binary, zero deps" property. It
is consistent with the existing external-validator model (CUE is also external),
but heavier. Requirements:

- "validator not found" must degrade to a **clear finding**, never a confusing
  failure or a silent skip.
- Document the env/CI requirement explicitly in the recipe and skill.

### 3.8 The containment payoff

When the verifier passes but the claim is weak, the vacuity finding plus
wrap-time triage is where it is caught — *by humans, deliberately, with a record*.
This is the spec-authorship attractor handled exactly where aiwf is already built
to handle residual, non-mechanical risk. The gate is mechanical; its *meaning* is
treated as provisional and routed to judgment. That stance is the whole gate-theory
paying off inside your own tool.

---

## 4. Candidate milestone sketch (to seed aiwf entities)

Proposed I-Loom breakdown, in the I1 mould. Treat as candidates, not a fixed plan.

- **M — ADR: verification contract data-model mapping** (verify/evolve/fixtures
  semantics; §3.2). *Decision-bearing; do first.*
- **M — ADR: engine factoring and standalone boundary** (library vs subprocess;
  the `loom` CLI; §5). *Decision-bearing; do first.*
- **M — Engine core: verifier-runner.** Largely reuse `contractverify`'s
  validator-agnostic execution + output capture; add proof-outcome semantics.
- **M — `dafny` / `verifier` recipe** alongside CUE and JSON-Schema recipes.
- **M — contract `mode: verify` + `--verifier` binding** in `aiwf.yaml` /
  `aiwfyaml`; `contractcheck` correspondence rules extended for the new kind.
- **M — output→findings (opaque passthrough)**; structured parser deferred.
- **M — vacuity check (mutation on lowered claims) + kill-rate finding** (§3.5).
  *This is the differentiator; sequence it early, not last.*
- **M — soft/hard severity + timeout config** (§3.6).
- **M — skill update**: teach the LLM host how to author claims and read
  verification findings (mirrors the embedded `aiwf-contract` skill).
- **M — (later/optional) `.lm` surface + lowering + lift-back** (§2.3). Separable
  layer; do only if v0 earns it.
- **M — (early validation) the endogenous-gaming experiment** (§7).

---

## 5. Decisions to make first (ADR seeds)

These genuinely fork the design; resolve before building.

1. **Standalone-first vs aiwf-bundled.** Recommended: standalone engine + `loom`
   CLI, with aiwf as a consumer. Trades more surface (a solo-maintenance cost) for
   reachability by the verification crowd. If the goal is *only* to complete aiwf
   for your own use, a bundled contract kind is simpler. **This choice depends on
   your goal, which is still unpinned — pin it here.**
2. **Subprocess vs library link** between aiwf and the engine. Subprocess = clean
   boundary, easy independent release, process overhead. Library = tighter, faster,
   couples release cycles. (Both are Go, so a library is feasible.)
3. **Which verifier first.** Dafny (recommended: gentler tooling, fastest to a
   result) vs F\* (closer to the umbrella's refinement-type ambition, harder).
4. **`.lm` now or later.** Recommended: **later.** v0 in native syntax; build the
   language only once the gate and the vacuity check are proven and you have a
   reason. Building the language first optimizes the part not in doubt.
5. **Vacuity-check level.** Confirm: mutate/measure on lowered form, lift to
   surface (§3.5). This is non-negotiable once `.lm` exists; decide it now so the
   engine is built for it.

---

## 6. Risks and open questions (gap seeds)

- **Narrow adoption ceiling.** The verification crowd is small and already served
  by experts; the SDD workflow crowd does not want formal verification for most
  code. Good packaging *removes friction*, it does not *create demand*. The
  realistic target is high-assurance / regulated / crisp-invariant work.
- **Workflow coupling hinders the pure crowd.** The standalone `loom` CLI exists
  to mitigate this; if it is not actually decoupled, the hindrance returns.
- **The expensive part is downstream.** Easy aiwf wiring ≠ easy verification for
  users — they still must *write verifiable claims*, which most repos won't for
  most code.
- **Toolchain dependency.** Dafny + Z3 + .NET in consumer env/CI dents the
  zero-deps property; "not found" must degrade gracefully.
- **Translation layer as a new attack surface.** Once `.lm` lowers to a verifier,
  a claim can read strong on the surface and discharge vacuous below — or the
  lowering can be subtly wrong. The translator becomes something you must trust:
  "verifying the verifier" recursing onto your own lowering pass.
- **Vacuity detection raises the bar; it does not eliminate gaming.** A determined
  author narrows an antecedent in any grammar. Containment, not solution.
- **Solo maintenance.** Standalone CLI + aiwf integration + shared engine is more
  surface than one bundled feature. Real cost; weigh against decision #1.
- **Determinism / flakiness of verifiers as gates** (Z3 resource limits) — see the
  soft/hard severity mitigation, but it remains a live UX concern.
- **Open:** does the `invalid/`-as-regression-suite mapping cover the cases you
  care about, or do you need a distinct sub-model? (Drives the §3.2 ADR.)

---

## 7. The validation experiment (recommended early)

The integration is also the natural home for the experiment that both validates
the differentiator and is the publishable contribution. Design it to demonstrate
**endogenous** weakening, not capability failure (they look identical in the
output), via a contrast condition:

1. Ask the model to author a claim for a component **when it is only specifying**
   (disinterested) → record claim strength (mutation kill rate).
2. Ask the **same** model to author the claim **when it is also graded on making
   its implementation pass** (incentivized) → record claim strength.
3. **The result is the gap.** Strong-when-disinterested + weak-when-incentivized
   demonstrates the thing the accuracy-framed literature structurally cannot see,
   and the vacuity check is what catches it. If the gap does not appear, that is
   also a real, worth-knowing result — and it tells you the differentiator is
   weaker than hoped before you over-invest.

Run this on the ledger / conservation example first; it is small and has a
non-trivial invariant.

---

## Appendix A — Mapping to aiwf concepts

| loom-light concept | aiwf mechanism it reuses |
|---|---|
| run the gate | `contractverify` validator-agnostic execution |
| make the gate real | pre-push hook chokepoint |
| report results | findings (`path:line: severity code: message — hint`) |
| track the claim | contract entity + lifecycle (proposed→accepted→…) |
| who/when/why | provenance trailers, `history`, scope/authorize |
| catch weak-but-passing | findings + wrap-time triage (the containment loop) |
| bidirectional consistency precedent | the kernel's spec↔impl scanner (AC-5, "fails-verified") |

## Appendix B — Prior art to position against (not to reinvent)

- **Intermediate verification languages / verifier front-ends:** Why3 (WhyML →
  multiple provers), Boogie (Dafny's own IVL → Z3), Viper. Establishes that "a
  language that lowers to a prover" is a sound, populated pattern — your angle is
  the *containment-shaped* surface and LLM-authoring, not the translation itself.
- **LLM + formal-spec quality:** SpecGen, MutDafny (mutation on Dafny specs),
  Laurel, user-intent-formalization work, "weak postcondition" studies. Your
  vacuity check overlaps; position the *endogenous-gaming* framing as the delta.
- **Verified-codegen / contract-as-intermediate-artifact:** VibeContract,
  Contract-Coding, BRIDGE (code/spec/theorem, code-first). Adjacent architectures;
  cite them so the work reads as informed rather than reinvented.
- **Spec-driven workflow (adopted):** GitHub Spec Kit, Kiro, BMAD. These are
  aiwf's category on the workflow axis; loom-light is the gate they lack.

## Appendix C — On the name

"Loom" survives as the name of the verification capability — the standalone tool
and, if built, the claim surface. It is a better fate for the name than a
greenfield compiler: it attaches to the part that is genuinely distinctive (the
gate + the vacuity check), and it lets the bespoke-language instinct live on as a
bounded, well-positioned front-end rather than a multi-year platform.
