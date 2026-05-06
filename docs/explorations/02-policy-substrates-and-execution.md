# Policy Substrates and Execution: Design-Space Exploration

> **Status:** exploration
> **Audience:** anyone reading [`01-policies-design-space.md`](01-policies-design-space.md) and wondering what the policy primitive's *substrate* and *execution model* would actually look like in practice — what languages, what dispatch shape, what happens when the policy population grows to 1,000.
> **Hypothesis (tentative):** the field has already worked out most of this. RFC 2119 + EARS solves the prose-form layer; CUE works well as a unifying top-layer index that points at substrate-appropriate evaluators (CUE itself for static invariants, Rego for capability decisions, Go / language-native for code-as-policy, TLA+ for behavior, EARS-with-runner for prose-with-evidence); and the execution pattern (*index → select → cache-check → fan-out → collect → short-circuit*) is what every mature PaC and linting system converged on. The framework's contribution is consistency over substrates, not novel runtime invention.
> **Tags:** #aiwf #policies #substrate #execution #exploration

---

## What this is, and what it isn't

This is a companion exploration to [`01-policies-design-space.md`](01-policies-design-space.md), expanding the *Lateral options and prior art* section (§14 of the parent doc) into the substrate and execution territory specifically. It does not pick winners. It catalogues what exists, names what each thing buys, and sketches a concrete top-layer schema that ties them together.

This doc is descriptive. The targeted design session that produces `docs/design/policy-model.md` will be the one that selects.

The shape:

1. RFC 2119 — the prose-bindingness layer.
2. EARS — the prose-structuring layer.
3. RFC 2119 + EARS in combination.
4. CUE as a unifying top layer, with a concrete schema sketch.
5. The execution model — how a 1,000-policy population actually runs.
6. What this implies for the design session.
7. Open questions.

---

## 1. RFC 2119 — the prose-bindingness layer

The shortest, oldest piece of policy-language standardization in the field. **Scott Bradner wrote it for the IETF in March 1997 as RFC 2119, *Key words for use in RFCs to Indicate Requirement Levels***. Three pages. Updated in May 2017 by **RFC 8174** ("Ambiguity of Uppercase vs Lowercase in RFC 2119 Key Words"), which clarified that lowercase uses do *not* invoke the keyword definitions.

Together, the two RFCs define exactly fourteen terms — five capitalized keywords with five negations, plus four "lower-strength" pairs:

| Keyword | Meaning |
|---|---|
| **MUST** / **REQUIRED** / **SHALL** | Absolute requirement. |
| **MUST NOT** / **SHALL NOT** | Absolute prohibition. |
| **SHOULD** / **RECOMMENDED** | Strong, but exceptions exist; weighed against other valid reasons. |
| **SHOULD NOT** / **NOT RECOMMENDED** | Strong avoidance, exceptions exist. |
| **MAY** / **OPTIONAL** | Truly optional; both presence and absence are conformant. |

The capitalization is part of the spec. *"must"* in a sentence is regular English; *"MUST"* invokes the keyword's defined meaning. The trick is small but load-bearing: it makes prose mechanically scannable for normative content while staying readable to humans. It also encodes a *bindingness gradient* (MUST > SHOULD > MAY) that maps directly onto the parent exploration's §4 enforcement-spectrum and bindingness axis.

**What RFC 2119 buys for a policy framework:**

- **Mechanical parseability of prose.** A grep for `\b(MUST|MUST NOT|SHOULD|SHOULD NOT|MAY|REQUIRED|RECOMMENDED|SHALL)\b` enumerates the normative claims in a document. Counting MUSTs in an unreviewed policy gives a quick health signal. Hobbyist linters like `eslint-plugin-rfc-2119` exist.
- **Bindingness gradient already encoded.** The exploration's *advisory / warning / blocking* axis maps onto MAY / SHOULD / MUST without a custom vocabulary.
- **Industry recognition.** Anyone who has read more than a handful of standards documents knows the convention. New consumers don't have to learn aiwf's invented vocabulary. Used across IETF, W3C in spirit, ISO partially, FIDO Alliance, OAuth standards, OpenAPI.

**What it doesn't buy:**

- **Subject of the requirement.** RFC 2119 says *"the implementation MUST do X"* — it leaves entirely open *what X is*, *who is bound*, *under what conditions*, *what counts as evidence of compliance*. Subject-naming and evidence-binding are exactly what EARS adds.
- **Composition over multiple requirements.** RFC 2119 keywords are sentence-level; they don't say how multiple sentences relate (and / or / unless / superseded-by).

So RFC 2119 alone is necessary-but-insufficient. It's the *bindingness annotation* on top of *some other structuring scheme*.

---

## 2. EARS — the prose-structuring layer

**EARS** (*Easy Approach to Requirements Syntax*) was developed by **Alistair Mavin and colleagues at Rolls-Royce around 2009**, originally for civil aircraft engine requirements; the canonical paper is *"Easy Approach to Requirements Syntax (EARS)"* at the 17th IEEE International Requirements Engineering Conference (RE'09). It has spread far beyond aerospace — automotive (ISO 26262 work), medical devices (FDA submissions), high-assurance more generally.

EARS defines exactly five sentence templates. *Every requirement, no matter how complex, fits one of the five patterns* (or a small composition of them).

| Pattern | Shape | Example |
|---|---|---|
| **Ubiquitous** | always-true requirement, no precondition | *The system **shall** record every entity-touching commit with structured trailers.* |
| **Event-driven** | triggers on a specific event | *When a milestone is promoted to `done`, the system **shall** verify all acceptance criteria are ratified.* |
| **State-driven** | holds while a state condition is active | *While an ADR is in `proposed` status for more than 90 days, the system **shall** surface a review reminder.* |
| **Unwanted-behavior** | failure or prohibition | *If a verification runner returns no output, then the system **shall** treat the policy as unverified, not as passing.* |
| **Optional-feature** | applies only when an optional capability is present | *Where the consumer has enabled the brand-voice contract, the system **shall** validate generated copy against it.* |

EARS authors typically write requirements in this order: ubiquitous first (always-true), then event-driven (triggers), then state-driven (conditions), then unwanted-behavior (failure modes), then optional-feature (configurable extras).

**The genuinely useful properties:**

- **The pattern *is* the parser.** *"When X, the system **shall** Y"* trivially extracts to `{trigger: X, requirement: Y, bindingness: shall}`. Five regexes covers the lot.
- **Unambiguous composition.** Requirements compose by *trigger overlap*, *state intersection*, *event ordering*. EARS authors do the disambiguation at write-time; readers don't have to.
- **Counterexample-friendly.** When a requirement is violated, the violation report names the pattern: *"State-driven requirement R-12 violated: while ADR-7 was in `proposed`, no review reminder was issued for 91 days."* Same shape as model-checker output.
- **Naturally scoped.** Each requirement names its trigger / state / event explicitly. No global state to reason about; the requirement is self-contained.
- **Tooling exists.** *EARS Toolkit* from QRA Corp (commercial), open-source `ears-grammar` repos on GitHub, IBM Engineering Requirements Management DOORS support, several VS Code extensions. Less mature than OPA's tooling, but real.

**Where EARS strains:**

- **Cross-cutting policies.** *"Every public function has a docstring"* fits *Ubiquitous*, but the *evidence* is "for every function in the codebase." That's a quantifier; EARS handles it fine prose-wise but the runner has to enumerate.
- **Policies about *policies*** — supersession, waiver lifecycle. EARS works on the system; meta-rules need a separate layer.
- **Requirements that are really algorithms.** *"The cache eviction policy SHALL be LRU with a 1-hour TTL"* is more spec-shaped than EARS-shaped. Use TLA+ for that, EARS for the rest.

---

## 3. RFC 2119 + EARS in combination

The combination is what most regulated industries actually use today. Aerospace, automotive, medical, certain financial systems. The pattern: every requirement opens with one of the EARS keywords (*The system, When X, While X, If X then, Where X*) and binds with one of the RFC 2119 keywords (*shall / should / may*). It's three lines per requirement on average. It's deeply unflashy. It works.

A worked example, in EARS+RFC-2119 form, for a few §3 territory items from the parent exploration:

```
P-001 (ubiquitous, MUST):
  The system MUST record every entity-touching commit with structured trailers.

P-002 (event-driven, MUST):
  When a milestone is promoted to `done`, the system MUST verify that all
  acceptance criteria are ratified.

P-003 (state-driven, SHOULD):
  While an ADR is in `proposed` status for more than 90 days, the system
  SHOULD surface a review reminder.

P-004 (unwanted-behavior, MUST):
  If a verification runner returns no output, the system MUST treat the
  policy as unverified, not as passing.

P-005 (optional-feature, MAY):
  Where the consumer has enabled the brand-voice contract, the system
  MAY validate generated copy against it on each commit.
```

**Why this is worth adopting now, not later:**

- Costs nothing in implementation.
- Mechanically parseable today (five EARS regexes + the RFC 2119 keyword set).
- Industry-recognizable; reviewers from regulated environments parse this on sight.
- Gives the framework a cheap *prose-bindingness annotation* to sort policies by enforceability.
- Forward-compatible with any structural-shape decision the design session lands on — if policy bodies are CUE, they reference the EARS prose; if policy bodies are markdown, the EARS prose *is* the body.

**Load-bearing observation:** the prose-form layer of policy is a solved problem. RFC 2119 + EARS is the answer. Adopt it, get parseability for free, get industry recognition for free, ship.

---

## 4. CUE as a unifying top layer

CUE (Marcel van Lohuizen, Google, successor to BCL/GCL) sits at an unusual point: more expressive than JSON Schema, less heavyweight than OPA, declarative throughout, and explicitly designed for *unifying configuration and constraint*. The parent exploration's §14.4 covered what CUE buys and doesn't. This section pushes further: **does CUE work as a unifying top-layer framework**, with substrate-specific bodies linkable as parameterized evaluators?

The conditional answer is **yes**.

### 4.1 Why CUE is unusually well-suited as the top layer

- **Lattice-based unification is the killer feature.** Every CUE value sits in a lattice ordered from `_` (any) at top to `_|_` (bottom, contradiction) at bottom. Combining two CUE values is *unification*: the most specific value consistent with both. Combining three is unification of three. Combining a thousand: the same operation, idempotent and commutative. This is exactly the algebra a policy framework needs — many sources of policy combine into one consistent set, with conflicts surfacing as contradictions (not as silent overrides).
- **Schemas, defaults, and constraints in one substrate.** CUE doesn't separate "what the data must look like" from "what defaults apply" from "what cross-field invariants hold." All three live in the same constraint expression. For policies this means a single CUE file says *what shape a policy entity has*, *what the default values are*, and *what extra constraints any consumer's policy file must satisfy*.
- **Genuine first-class disjunctions.** A field can be `string | int | null`. The unification engine handles disjunction correctly. This matters because policy bodies in different domains (security, performance, naming, behavior) want different field shapes, and a single top-layer schema needs to express *"policy body is one of N shapes, depending on subject."*
- **Tooling is real.** `cue vet`, `cue eval`, `cue trim`, `cue export` to JSON / YAML / OpenAPI. Editor support across major IDEs. Adopted by Istio for configuration, Dagger for pipelines, Google internally as the BCL/GCL successor, HashiCorp's Waypoint internally. Not a hobbyist tool.
- **Composes with everything else.** A CUE schema can describe *the metadata of a policy entity that points at a Rego policy*, or *at a TLA+ spec*, or *at a Go validator function*, or *at an EARS-form requirement document*. CUE doesn't replace those substrates; it canonicalizes the *index over them*.

### 4.2 A concrete top-layer schema sketch

```cue
// schemas/policy.cue — the top-layer schema (illustrative)
#Policy: {
    id:          string & =~"^P-[0-9]+$"
    status:      "proposed" | "accepted" | "in-effect" | "waived" | "superseded" | "retired"
    subject:     "engineering" | "security" | "performance" | "naming" |
                 "citation" | "process" | "capability" | "governance" | string
    bindingness: "MUST" | "SHOULD" | "MAY"            // RFC 2119
    audience:    "project-engineering" | "framework-internal"
    title:       string
    rationale:   string

    // Body shape varies by subject; the discriminator drives a disjunction.
    body: #ProseBody | #EARSBody | #CUEBody | #TLABody | #CodeBody | #RegoBody

    // Enforcement is a list of pointers — runners that evaluate this policy.
    enforces: [...#Enforcer]

    supersedes: [...string]
    waivers:    [...#Waiver]
}

#ProseBody: { kind: "prose", content: string }
#EARSBody:  { kind: "ears",  requirements: [...#EARSRequirement] }
#CUEBody:   { kind: "cue",   path: string, package?: string }
#TLABody:   { kind: "tla",   path: string, spec_name: string }
#CodeBody:  { kind: "code",  language: "go" | "python" | "...", path: string, function: string }
#RegoBody:  { kind: "rego",  path: string, query: string }

#Enforcer: {
    kind:     "cue-vet" | "go-test" | "command" | "benchmark" | "mcp-call" | "rego-eval"
    target:   string
    timeout?: string
    severity: "blocking" | "warning" | "advisory"
}

#EARSRequirement: {
    pattern:     "ubiquitous" | "event-driven" | "state-driven" |
                 "unwanted-behavior" | "optional-feature"
    trigger?:    string                                // for event-driven
    state?:      string                                // for state-driven
    feature?:    string                                // for optional-feature
    bindingness: "shall" | "should" | "may"
    requirement: string
}
```

Every consumer's policy file unifies against this schema. `cue vet` validates conformance. The framework reads the unified result; it knows how to dispatch to enforcers based on `body.kind` and the `enforces[]` list.

**The top layer is CUE, *and* every body shape is a structured pointer to a substrate that fits the policy's subject.** This is the "language linkable to evaluators with parameters" shape made concrete: the CUE schema is the language; the `body` field's discriminator drives substrate selection; the `enforces[]` list is the parameter set; runners receive the parameters and produce findings.

### 4.3 The limits to be honest about

- **CUE doesn't run the enforcers.** It validates the *index* — that the policy entity is well-formed, that pointer targets exist, that bindingness is one of the allowed values, that disjunctions are honored. It does *not* execute Rego, run benchmarks, evaluate TLA+, or call MCP servers. Something else dispatches.
- **CUE is bad at imperative.** A policy that says *"when state X, do Y"* expresses fine in EARS prose, but CUE can't execute the Y. The runner does.
- **CUE is bad at temporal logic.** *"This sequence of events must happen in this order"* is a TLA+ shape, not a CUE shape. CUE records the pointer to the TLA+ spec; doesn't replicate it.
- **Performance.** CUE evaluation is fast for thousands of constraints; *not* fast for millions. Plausibility check: a project with 1,000 policies is large. CUE handles 1,000 trivially.

---

## 5. The execution model — how 1,000 policies actually run

The naïve answer is a for-loop over 1..1000. In practice nobody does it that way past a few dozen policies, because the field has worked out better patterns over twenty years of PaC and CI experience. Each pattern below is the convergent answer multiple tools landed on independently.

### 5.1 Indexing and selection — the foundation

You don't run all 1,000 policies on every event. You *select* which policies apply to *this* event, and run only those.

- **Subject-indexed.** Each policy declares its subject (`engineering` / `security` / `naming` / `citation` / ...). Each enforcement event declares its scope (a commit, a milestone promotion, a verb invocation). Map subject → scope; run the intersection.
- **Trigger-indexed (EARS-style).** Each EARS *event-driven* requirement declares its trigger; the framework evaluates the requirement only when that trigger fires. The runtime equivalent of the EARS structure.
- **Path-indexed.** Policies that constrain *files matching pattern X* register against a glob. A commit touching `internal/eventlog.go` activates only the policies whose globs match. This is how pre-commit, lefthook, and husky all work.
- **Layer-indexed.** Per ThoughtWorks Vol 34's harness-engineering frame: feedforward controls run at compose-time (skill loading, instruction context); feedback controls run at write-time, commit-time, push-time, CI-time. A policy declares its layer; only policies for *this* layer run on *this* event.

OPA's *bundles* model is the canonical PaC version: policies live in named bundles, services request only bundles relevant to their decision points, the evaluator loads only those. Cedar's *policy stores* are the same idea with explicit scoping.

For aiwf the natural shape: `aiwf policy verify` takes a `--scope` flag. Default scope is the working-tree changes; `--scope ci` is the full sweep; `--scope pre-commit` is the small fast subset. Each policy declares which scopes it participates in.

### 5.2 Compilation and caching

You don't re-parse and re-evaluate from source on every run. You compile once and cache.

- **OPA bundles** are tarballs of compiled Rego, served from a bundle service (or a static URL); evaluators pull and cache locally.
- **Cedar** has compiled policy stores; evaluation is over the compiled form, not the source.
- **CUE** has a compiled cache; skipping the parse phase saves an order of magnitude on large schemas.
- **Linters** (golangci-lint is the canonical example): each linter is a Go function; the framework loads them once into one process and runs them across the file set in one pass with shared AST cache. Running 30 linters across 100,000 lines of Go takes 5 seconds; running them as 30 separate processes would take 30× more.

For aiwf: policies that compile to CUE schemas, Rego modules, or Go validator functions all benefit from one-process loading. The dispatcher loads the policy index, instantiates each enforcer once, then evaluates against the relevant input.

### 5.3 Parallel evaluation

Most policies are independent. Run them concurrently.

- OPA evaluates queries in parallel by default for independent policies.
- Cedar's evaluator parallelizes across policies in a store.
- Linters in golangci-lint run in goroutines; output is collected and merged.
- Dagger pipelines explicitly express policy graphs as DAGs; independent nodes run in parallel.

For aiwf: a policy run is a fan-out / fan-in. Fan out across enforcers; collect findings; merge into a single report. Worker pool sized to CPU count; finished policies don't block running ones.

### 5.4 Short-circuiting and early termination

Some policies are gates; others are advisory. Gates that fail can short-circuit the rest of CI; advisory findings always run to completion.

- *Pre-commit hooks*: failing a hook can either *block the commit* (default) or *allow with warning*. The pre-commit framework supports both.
- *CI pipelines*: failed *blocking* checks stop the pipeline; failed *non-blocking* checks emit warnings and continue.
- *Cedar*: deny-overrides means once a deny is found, evaluation can stop early for the decision (subject to audit-completeness policies).

For aiwf: a policy's `severity` (blocking / warning / advisory) drives short-circuit behavior. Blocking findings can stop the run after the current batch finishes (so dependent findings still surface); advisory findings always run to completion.

### 5.5 Incremental evaluation

You don't re-evaluate policies whose inputs haven't changed.

- *Bazel* and the build-system tradition: every policy declares its input dependencies; the framework hashes the inputs; if the hash matches the last run's, the result is cached.
- *Pre-commit* with `--all-files` is the full sweep; default is "files touched by this commit only" — only running policies whose path-index matches the changed paths.
- *golangci-lint* has `--new` mode: only report findings newer than a baseline.
- *OPA* with partial evaluation: for queries with mostly-stable inputs, OPA pre-evaluates the stable part once and re-evaluates only the dynamic part per request.

For aiwf: a policy verification cache keyed on (policy version × input hash) lets a CI run skip verifications whose inputs haven't changed since the last run. For a 1,000-policy repo, the typical pre-push run touches a fraction of the policies because most inputs haven't moved.

### 5.6 Sharded / batched evaluation

When a single repo can't fit all policies' inputs in memory, shard.

- *OPA* in distributed mode runs as a sidecar per service; each evaluator only sees its slice.
- *Conftest* (OPA over file inputs) shards by directory, by file pattern, by manifest type.
- *Bazel* builds shard across workers; policy checks ride the same sharding.

For aiwf-scale (one repo, hundreds of policies, thousands of files), this isn't yet relevant. It becomes relevant if cross-repo registries or organization-wide policy stores enter the picture (the parent exploration's §12 cross-project portability).

### 5.7 The typical CI-time pattern, end-to-end

Putting it together — what a 1,000-policy run actually looks like in CI today, in the OPA / Cedar / linter tradition:

1. **Boot once.** Load the compiled policy bundle, instantiate enforcers. Sub-second.
2. **Index the input.** Compute which policies apply to this run (scope, paths, triggers).
3. **Cache lookup.** For each selected policy, compute the input hash; skip if cached.
4. **Fan out.** Run the remaining policies concurrently across a worker pool.
5. **Collect findings.** Merge into a single report with severity-sorted output and counterexample traces.
6. **Short-circuit.** If any blocking finding fails, mark the run as failed; continue running advisory checks for visibility.
7. **Cache the results.** Hash-keyed by policy version × input hash.

For 1,000 policies on a typical commit, the realistic distribution: 200–400 selected (after scope/path filtering), 50–100 actually evaluated (after cache hits), parallelized on 8 cores, taking 5–30 seconds total. This is the steady-state target.

Cold-cache CI runs touching everything: 5–10 minutes for 1,000 policies in this shape, depending on the heaviest enforcer (mutation testing, fuzz, full-corpus security scans dominate; CUE schemas, EARS pattern matches, simple validators are negligible).

### 5.8 Translation into aiwf-shaped commitments

A pragmatic translation of the above into framework-shaped commitments the design session can settle:

- **Policy entities declare a `scope`, a `subject`, an optional `path-glob`, and a list of `enforces[]` runners.** This is the indexable surface.
- **`aiwf policy verify` takes `--scope` and `--changed-files`.** Defaults select the smallest viable set for the event type (commit, push, CI).
- **A policy run is a fan-out across enforcers, with a worker pool.** The dispatcher is part of the framework; the runners are external (CUE, Rego, Go funcs, commands, MCP calls).
- **Findings are cached keyed on policy version × input hash.** Cache lives in the consumer repo's `.aiwf/` cache directory.
- **Severity drives short-circuit.** Blocking findings fail the run; advisory findings complete and report.
- **Output is structured.** JSON envelope with one entry per finding, including counterexample traces where the runner produces them. Renderers shape it for humans.

This is mostly *coordination infrastructure*, not novel research. It is what every mature PaC and linting system has converged on. The framework's contribution is consistency — same structure across substrates (CUE, Rego, Go, TLA+, EARS-with-runner), same finding format, same caching surface.

---

## 6. What this implies for the design session

A few moves the design session can make that the parent exploration's §13 open questions did not fully spell out:

1. **Adopt RFC 2119 + EARS as the prose-form layer immediately.** No-cost adoption; mechanically parseable; industry-recognizable. Does not depend on any other form decision. The framework's prose convention (in skill content, ADR rationale, policy bodies) can move to RFC 2119 + EARS regardless of where the rest of the policy form lands.
2. **Pick a body-language first; the entity surface follows.** If CUE carries two-thirds of the territory, the entity surface is "policy with a CUE body OR an EARS+runner pointer OR a code-as-policy pointer" — a much smaller schema than "policy with a free-form prose body and an unspecified enforcement layer."
3. **Take CUE seriously for the top layer, not just for static-shape bodies.** §4 sketched the top-layer schema; the design session can adopt it (or a refined version) as the *index over policies*, regardless of which body shapes ship first.
4. **Borrow OPA's bundle-versioning posture.** A policy applies in versions; in-flight work is pinned to the version under which it started. Waivers granted under v3 must keep working when v4 lands, until v4's grace period expires.
5. **Borrow Cedar's policy-conflict-analysis posture without adopting Cedar itself.** Two policies that govern the same subject must declare their precedence, or the framework surfaces a finding. One paragraph rule, not a feature.
6. **Borrow formal-methods counterexample reporting.** Every finding includes the specific trace that produced it. The reader doesn't reverse-engineer the failure.
7. **Adopt the *index → select → cache-check → fan-out → collect → short-circuit* execution shape.** This is the converged pattern; do not invent a new one.
8. **Plan the cache from day one.** Without caching, even hundreds of policies bog the inner loop. With caching, thousands are tractable. The cache directory is `.aiwf/cache/` (gitignored); the cache key is (policy version × input hash).

---

## 7. Open questions

1. **Does the RFC 2119 + EARS prose adoption need a CI-side validator from day one, or is convention enough?** A regex-based EARS-conformance check is a one-day implementation; whether the framework ships one immediately or treats EARS as a working norm is a small design choice.
2. **What's the smallest set of body-shape types worth supporting in the v1 schema?** Six (prose, EARS, CUE, TLA, code, Rego) is the maximal sketch. Two (prose-with-EARS + CUE) is a plausible v1; everything else is opt-in extension. Trade-off is between expressiveness and surface area.
3. **Is the `enforces[]` list 1..N, or 1..1?** Some policies have one obvious enforcer; some have several (a security policy might have a static check *and* a runtime assertion). Allowing N is more general; fixing 1 simplifies dispatch. Worth choosing deliberately.
4. **Does the framework own the CUE evaluator, or shell out to `cue vet`?** Embedding is faster (no process startup) but ties the framework to the CUE Go module. Shelling out is simpler but slower per call. Cache largely makes this moot, but the choice affects portability.
5. **Where does the policy-version pin live for in-flight work?** A trailer on the originating commit? A field on the milestone? A separate registry? OPA's bundle version + policy hash is one shape; aiwf may do better with milestone-attached pins.
6. **How does the framework handle a policy whose runner does not exist on this machine?** Skip with a finding? Fail closed? Warn? Different consumers want different defaults; this is a config decision.
7. **Does the cache need invalidation beyond input hash?** Time-based? Tool-version-based? A consumer who updated `gofumpt` to a new version wants the affected policies re-evaluated even if file inputs are unchanged. This is the standard build-system dependency-tracking problem; CUE's lattice doesn't solve it; the cache key has to include enforcer-binary version.

---

## 8. A frame to hold while iterating

Treat this exploration as the **menu**, not the meal. The PaC field has worked out conflict resolution, versioning, test-fixture discipline, and decidability trade-offs. CUE has worked out lattice-based composition. Formal methods has worked out invariants and counterexamples. EARS + RFC 2119 has worked out the prose-form layer. The design session's work is **selection and composition**, not invention.

The strongest single move toward determinism in this space is the combination: **CUE for the top-layer schema and static-shape bodies, code-as-policy for dynamic ones, RFC 2119 + EARS for the prose layer, counterexample-shaped findings, conflict-resolution declared per policy, the index → select → cache-check → fan-out → collect → short-circuit execution pattern.** That set is well-trodden, mechanically supported, and small enough to ship.

Whether the framework adopts this combination, a subset, or a different selection is the design session's decision. This document's claim is only that the menu is largely already on the table; the work in front of the framework is choosing what to put on the plate.

---

*This document is intentionally exploratory. The next step, when the time is right, is the same targeted session [`01-policies-design-space.md`](01-policies-design-space.md) §13 calls for — picking positions on substrate and execution choices and producing a defended `docs/design/policy-model.md`. Both this exploration and the parent feed that session.*
