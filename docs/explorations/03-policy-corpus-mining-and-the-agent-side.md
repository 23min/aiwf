# Policy Corpus Mining and the Agent-Side Question

> **Status:** exploration — design-phase prep, not a proposal.
> **Audience:** anyone working on `01-policies-design-space.md` or `02-policy-substrates-and-execution.md` who wants concrete corpus evidence before the targeted design session, *and* anyone wondering how an AI agent actually does code-generation work in a repo where the rules are first-class objects.
> **Hypothesis (tentative):** (a) two real-repo policy corpora are enough to test the design-space's bucket model and substrate selection against contact with reality, but not enough to commit to a kernel shape — that needs a vertical slice and a git-log lifecycle survey; and (b) once a policy system exists, the agent's CLAUDE.md and skills do not shrink to nothing — they take on a new shape centered on a *generated policy digest*, with the verifier acting as backstop rather than primary teacher.
> **Tags:** #aiwf #policies #corpus #agent-context #exploration

---

## What this is, and what it isn't

A companion to [`01-policies-design-space.md`](01-policies-design-space.md) and
[`02-policy-substrates-and-execution.md`](02-policy-substrates-and-execution.md). Both
of those docs map the territory abstractly. This one tests the abstractions
against two real-repo corpora and works through one concrete question the
abstractions did not pin down: **how does an AI agent doing code generation
actually consume a policy system without bloating its context?**

It is design-phase prep — not the targeted design session itself, not an
implementation plan, not a kernel commitment. The corpora it draws on live in
`.scratch/` (gitignored, local-only). What survives into this doc is the
*lessons*, sanitized of project-specific identifiers where possible.

The shape:

1. What was mined, from where, with what method.
2. The convergent findings — what both corpora agree on.
3. The divergent findings — where the two repos differ and what that teaches.
4. The agent-side design question and its answer arc.
5. The honest assessment: what the corpus evidence is usable for *now*, what it
   is not yet usable for, and what would close the gap.
6. A vertical slice as the proposed next concrete step.

---

## 1. The two corpora

| Corpus | Repo | Stack | Raw extracted | Unique after de-dup | Mining depth |
|---|---|---|---|---|---|
| **A** | Liminara | Elixir/OTP umbrella + Python (via `:port`) + TypeScript + CUE schemas | ~625 | ~350-400 | Three parallel mining passes; all rules in CLAUDE.md, `.ai/`, `.ai-repo/`, governance, ADRs, every CUE schema, every reviewer rule; sample-only on the 21 framework skills. Output: 5 files in `.scratch/liminara/`. |
| **B** | FlowTime (`flowtime-vnext`) | .NET 9/C# + Svelte/TypeScript + Rust evaluation core + Python tooling | ~140 candidates | ~140 (single-pass, no de-dup pressure) | Single targeted pass over CLAUDE.md, `.editorconfig`, CI, `.claude/` skill set, `docs/architecture/` (NaN policy, run-provenance), `docs/schemas/`, an in-flight epic + milestone (E-25 / M-066), the deletion-stays-deleted shell guards in `work/guards/`. Output: 7 files in `.scratch/flowtime/`. |

Both repos use the same ai-workflow framework family (Liminara on the
`.ai/`-submodule generation; FlowTime on aiwf v3 with the rituals plugin), so
their *workflow-policy* surface overlaps deliberately. Their *engineering* and
*project-specific* policy surfaces are independent and built before the
policy-as-primitive question was on the table — which is exactly what makes
them useful evidence.

The two mining efforts used the same four-bucket frame:
*general engineering* | *project-specific* | *workflow / PM* | *rest*. The
Liminara pass surfaced a fifth bucket worth naming, *agent-behavior*, which the
FlowTime pass folded into workflow.

---

## 2. What the two corpora agree on

The convergent findings are the load-bearing ones. Where two different repos,
two different teams, two different stacks, and two different mining passes
arrive at the same observation, the observation is probably about the
*territory*, not about either corpus individually.

### 2.1 Rung distribution is roughly the same

| Rung | Liminara | FlowTime | Comment |
|---|---|---|---|
| 0 (LLM memory only / open question / non-goal) | ~0% | ~11% | FlowTime explicitly tagged open questions and deferred non-goals; Liminara folded these into rung 1. Methodology drift, not corpus drift. |
| 1 (markdown / prose) | ~75% | ~54% | Both rung-1-dominant. The lower FlowTime number reflects more aggressive elevation tagging (counting an item as rung 2 if any substrate exists, even if partial). |
| 2 (pattern lint / regex / CI / shell guard) | ~10% | ~25% | FlowTime's `.editorconfig` + Roslyn + CI + the literal `work/guards/*.sh` scripts give it a heavier rung-2 floor. Liminara concentrates rung-2 in formatters/linters only. |
| 3 (schema / type) | ~10% | ~9% | **Strikingly similar.** Both repos have a schema backbone (Liminara: 5 named CUE schemas; FlowTime: `model.schema.yaml` + `template.schema.json` + 4 others) and that backbone does the heavy mechanical lifting in both. |
| 4 (runtime / test) | ~5% | ~2% | Both small. The NaN policy in FlowTime + the property tests + golden fixtures in Liminara are the typical sites. |
| 5 (formal proof) | 0% | 0% | As the design-space §4 anticipated. |

The single most striking convergence: **the schema layer carries roughly 10% of
each corpus and is the strongest mechanical enforcement in both.** This is not a
coincidence. Closed-grammar contracts get rung-3 enforcement essentially for
free wherever a schema language exists. The substrate exploration's bet on CUE
as a top-layer index is supported by this — but importantly, *neither repo
needs* CUE specifically; both repos already get the same enforcement value out
of substrate-native schemas (CUE in Liminara, JSON Schema + YAML in FlowTime).
The lesson is *schemas as substrate*, not *CUE specifically*.

### 2.2 The MUST-at-rung-1 failure mode dominates both

Both corpora exhibit the failure mode the design-space §4 names directly:
*"most policy failures escape because the rung is too soft for the bindingness
claimed."* Concretely:

- *"Compatibility shims are banned by default"* — Liminara: rung 1 only. FlowTime
  has the same rule (`P-31`, `P-33`) at rung 1 only.
- *"NEVER commit or push without explicit human approval"* — both repos. Both
  rung 1. Mirrored multiple times in prose. No git hook gates it in either.
- *"Branch coverage required before declaring done — line-by-line audit"* — both
  repos. Both rung 1. Manual audit is the enforcement.
- *"Doc-tree taxonomy"* — Liminara has explicit bind-me/inform-me; FlowTime has
  a Truth Discipline section with truth classes. Both are rung 1; neither has a
  check that refuses a misfiled doc.

The MUST-heavy distribution combined with the rung-1 dominance is the
structural problem the design-space exploration identified, validated against
real corpora. The honest reading: **most of what gets called "MUST" in either
repo can deliver only "SHOULD" enforcement today.**

### 2.3 The four-bucket split survives, with one wrinkle

| Bucket | Liminara count (~) | FlowTime count (~) | Survives? |
|---|---|---|---|
| General engineering | 140 | 50 | Yes — same shape, bucket B is bigger in Liminara because the pass was deeper. |
| Project-specific | 180 | 51 | Yes — both repos' largest or near-largest bucket. |
| Workflow / PM (aiwf domain) | 70 | 41 | Yes — both repos repeat or specialize the same workflow rules. |
| Agent-behavior | 90 (named separately) | folded into workflow | Boundary case. |
| Rest | 25 | 33 | Yes — meta-policies cluster here in both. |

The wrinkle: **agent-behavior is a real, distinct fifth bucket**, but only
becomes visible at sufficient mining depth. Liminara's deeper pass surfaced it
as ~90 rules; FlowTime's lighter pass folded the same content into workflow.
Both reads are defensible. The implication for the design-space: when the
exploration eventually settles on a kind-set, a *role-and-skill-conduct*
category is worth holding open as a possible kind separate from workflow.

### 2.4 Both repos have a "two-population" character

Both corpora exhibit a clean split between:

- **Project-shape policies** — describe *the thing being built*. Liminara: truth
  model, doc-tree, contract matrix, the five CUE-schema invariants. FlowTime:
  schema-alignment, NaN policy, run-provenance, flow-authority. Authored by
  humans, ratified by humans, applied to AI + humans both.
- **Engineering-discipline policies** — describe *how building is done*. TDD,
  commits, branch coverage, formatting, no-shims-without-trigger. Stable
  across years; transferable across repos with minor wording changes.

The split is not a mining artifact; it shows up in both repos with the same
shape. It maps cleanly onto the substrate exploration's Option α
(*CUE/schema-bodied for static-shape, code-as-policy for dynamic*): project-shape
policies fit CUE/schema; engineering-discipline policies fit code-as-policy or
runner-pointer.

### 2.5 Both repos already have proto-policy artifacts the framework should
absorb

Each corpus surfaced one or two artifacts that are *already shaped like* the
policy primitive being designed, just without the framework affordances:

| Repo | Proto-policy artifact | What it already does |
|---|---|---|
| Liminara | The five named CUE schemas + their valid/invalid fixture libraries + their per-schema ADRs | Rung-3 enforcement; explicit ratification (the ADR); schema-evolution loop (every fixture validates against current HEAD); supersession mechanics (schema versioning). The CUE schemas are *de facto* policy entities. |
| FlowTime | The `dead-code-audit` skill — recipe-driven, polyglot, soft-signal contract, structured findings, blind-spot sweep | A complete small-scale policy framework in miniature: per-substrate recipes; bootstrap-then-use two-step; tool-failure-is-a-finding meta-rule; structured finding classes. |
| FlowTime | The per-milestone grep guards in `work/guards/m-E19-0{2,3,4}-grep-guards.sh` | Rung-2 enforcement bound to a milestone identity; lifecycle is *forever after the milestone closes*; explicit allowlist mechanics; a literal "tool-failure-as-pass-is-worse-than-no-check" guard. |
| FlowTime | The NaN policy doc with three tiers + per-site enforcement table + how-to-add-a-new-site workflow | Gold-standard "policy entity" shape A: name, why, tiered severity, per-site table, lifecycle for adding new sites. |

The framework's job, partly, is to recognize these as the same kind of thing
and provide the consolidation. Neither repo's authors framed them as policies;
both repos' authors built them anyway, because the underlying need is real.

---

## 3. Where the two corpora differ, and what that teaches

The divergences are smaller than the convergences and mostly point at *what
substrate the team had available* rather than *what the team thought policy
was*.

### 3.1 Liminara concentrates enforcement in CUE; FlowTime spreads it across .editorconfig + CI + shell guards

Liminara has five CUE schemas doing most of the rung-3 work, plus formatters
and linters at rung 2, plus a single CI workflow (`wf-graph-ci`). Mechanical
surface is concentrated.

FlowTime has its model schema at rung 3 but spreads rung-2 work across:
- `.editorconfig` (Roslyn naming + analyzer severity tuning)
- CI yaml (8 named test jobs with hang timeouts)
- `work/guards/*.sh` (per-milestone grep guards)
- Test-as-policy patterns (the NaN policy site tests; the `Survey_Templates_For_Warnings` baseline canary)
- `aiwf check` (FSM + acyclic + refs-resolve)

**The lesson: a policy primitive must compose cheaply with an arbitrary number
of substrate types per consumer.** Liminara needs CUE-shaped enforcers;
FlowTime needs Roslyn-config-shaped, CI-yaml-shaped, shell-script-shaped, and
test-shaped enforcers. The substrate doc's `enforces[]` list is right to be
1..N rather than 1..1.

### 3.2 FlowTime has explicit per-milestone-bound enforcement; Liminara does not

The grep-guards in `work/guards/m-E19-0{2,3,4}-grep-guards.sh` are *named after
the milestone that produced them* and live forever after the milestone closes.
This is a real pattern — the milestone authored the guard as part of its
deletion work, and the guard is the durable artifact that prevents the
deletion from being silently undone in a future change.

Liminara has nothing equivalent. Its enforcement is centralized (the CUE
schemas, the CI workflow) and milestone-independent.

**The lesson: the policy primitive should support *milestone-bound
enforcement* as a first-class shape.** Most policies are repo-bound; some are
milestone-bound. The framework's verb-set should let a milestone produce a
permanent post-close enforcement artifact without that artifact becoming
orphaned from its origin.

### 3.3 Liminara has an in-flight policy-shaped surface (the bind-me / inform-me doc taxonomy); FlowTime has an in-flight policy-ratification *milestone* (E-25 / M-066)

Liminara's ADR-0003 (doc-tree taxonomy) is itself a meta-policy: it says
*"these directories gate work; these directories inform work"*. This is
governance content treated as architectural decision.

FlowTime's M-066 is a worked example of the design-space's flow exactly: a
milestone whose entire deliverable is a policy ratification (the flow-authority
ADR) plus a doc sweep classifying every document in the repo against the new
policy plus an explicit naming of the schema/compile/analyse enforcement points
that the *next* milestone (M-069) will implement. **It is the first real
in-flight policy-ratification work in either corpus, and its structure tracks
the design-space's lifecycle (§5) and shape catalogue (§14) one-to-one.**

The lesson: policy-ratification work is a recurring structural pattern, and
the framework can ship workflow templates for it. The design session can use
M-066's structure as a literal worked example of "what a policy-ratification
milestone looks like in a real repo."

### 3.4 Liminara explicitly names truth-precedence as a register split (bind-me / inform-me); FlowTime names it as a four-class precedence ordering (code+tests > decisions/ADRs > epic specs > arch docs > history)

Same governance content, two different formal shapes. Liminara's split is a
*binary classification per directory*; FlowTime's is a *total order per source
type*. Both work; both are rung 1.

**The lesson: there is no settled vocabulary for the truth-precedence layer.**
The framework's policy primitive should accommodate both shapes (and probably
others) — the conflict-resolution rule is a *property* of a policy entity, not
a fixed framework axiom.

---

## 4. The agent-side question

The mining produced the corpus. The corpus made one further question
unavoidable: **with a working policy system, how does an AI agent doing code
generation actually consume the policies without bloating context?**

This question is not addressed by either the design-space or the substrates
doc; both treat policies as objects to be designed, not as objects to be
*consulted by an agent at compose time*. The corpus mining made it concrete:
a single repo can have 140-400 policies. The agent cannot carry all of them in
context; it cannot even carry the *headlines* trivially without spending a
significant fraction of its context budget.

### 4.1 The wrong answer: lazy-load on demand

The first instinct is to make the agent query the policy system lazily —
"what policies apply to a change in `src/X/`?" — fetching only what's needed
for the current work. This sounds attractive: minimal context, just-in-time
retrieval, the verifier as a backstop catching anything missed.

It is wrong, for the reason a careless-driver analogy makes vivid: **the agent
cannot violate a rule and then plead "I didn't know there was a sign."** A
lazy-load architecture turns every preventable violation into a round trip
(write code → verify → finding fires → fetch policy → fix code → re-verify),
and worse, it makes the agent look careless to its user. The signs are up;
ignorance is not a defense.

The deeper problem: **the agent doesn't reliably know what to query for.** Most
preventable violations come from rules the agent didn't know existed — the
camelCase-not-underscore rule, the no-snake_case-in-JSON rule, the
forward-only-migration rule. The agent cannot ask "which rules might bite me
on this change?" without already knowing the rules' names. Lazy-loading
solves the wrong problem.

### 4.2 The right answer: a generated digest

The agent should always carry a *digest* of all active policies — enough to
be law-abiding by default — and the verifier should be the *backstop* that
catches what the digest didn't anticipate, not the *primary teacher*.

Per policy, in CLAUDE.md / skill context, the digest entry is one to three
lines:

- **Id + headline** — `P-2: JSON payloads and schemas use camelCase; never snake_case.`
- **Trigger or scope** — `(touches: any *.cs JSON serialization, any docs/schemas/*)`
- **Severity** — `blocking / warning / soft-signal.`

Optionally a fourth line for policies whose *why* changes how an edge case is
judged — a one-line pointer to the rationale, not the rationale itself. The
full body lives in the policy store and is fetched when (a) the headline is
ambiguous for the case in front of the agent, or (b) a finding fires and the
agent needs the *why* to remediate well.

For a corpus the size of FlowTime's, a three-line digest is roughly 400-500
lines of additional context. For Liminara's denser corpus, it would be
600-800 lines. That cost is real. It is also the *price of being law-abiding
by default*, and it is significantly less than the current pattern of carrying
full doc bodies plus inlined CLAUDE.md prose plus defensive ad-hoc reads.

### 4.3 Three layers of digest, composed

The digest is not one thing — it is three composed scopes:

1. **Repo digest** — every active policy in the repo, one to three lines each,
   grouped by subject. Generated by `aiwf policy digest --write` whenever the
   policy set changes (same shape as `aiwf render roadmap` regenerates
   `STATUS.md` today). Lives at `.aiwf/policy-digest.md`. Loaded into CLAUDE.md
   context via reference. This is the agent's copy of the highway code.

2. **Skill-scoped digests** — embedded in each skill, carrying only the
   policies most relevant to the skill's operation. A skill like
   `dead-code-audit` carries the digest for soft-signal contract,
   finding-classes, blind-spot families — perhaps 10 policies, not 140. Same
   shape, narrower scope.

3. **Milestone-scoped digest** — surfaced on session start. The active milestone
   already declares which policies its work touches; the digest of *those* —
   typically 5-15 policies — gets prepended to working context. The agent's
   "current driving conditions."

The three layers compose: repo digest = baseline; skill digest = operating
mode; milestone digest = current task. The agent always has all three. None of
them require human authoring beyond the policy entries themselves; all three
are generated.

### 4.4 What stays in CLAUDE.md alongside the digest

CLAUDE.md doesn't shrink to nothing, but its role changes. With the digest
absorbing the *what*, what remains is roughly:

- **The digest itself** (or a reference to it) — 400-800 lines depending on
  corpus size.
- **Routing** — how to fetch full policy bodies on demand; how to ask the
  verifier proactively before commit.
- **Truth precedence and conflict resolution** — the governance layer (this is
  exactly what FlowTime's Truth Discipline section is, and what Liminara's
  ADR-0003 is). Not absorbable into per-policy entries because it is *meta*.
- **The hard rules that bind agent behavior itself** — never commit without
  approval, never modify generated files. Small and irreducible.
- **Stack orientation** — repo shape, ports, build commands. Brief, factual,
  navigational.
- **Escalation playbook** — what to do when a finding fires (blocking → fix;
  soft-signal → surface; tool-failure → treat as finding, not pass). Generic
  across repos; the framework should ship it.

So FlowTime's current ~250-line CLAUDE.md becomes maybe 600-800 lines, *up*
not down. The cost is real; the benefit is that the agent now actually knows
the rules instead of pretending to. **The way to make the cost tolerable: the
digest is generated, standardized in shape, and held to one-to-three lines per
entry by mechanical regeneration.** Hand-authored CLAUDE.md grew because each
new rule got a paragraph; a generated digest holds the line.

### 4.5 The verifier's role with this in place

The verifier no longer teaches the agent; it **checks the agent's work against
what the agent should already have known**. Findings come in three flavors:

1. **"You knew, you slipped."** The policy was in the digest; the agent
   violated it anyway. Direct fix; the agent should not need the *why*.
2. **"You didn't know because the digest doesn't cover this edge."** The
   headline was too compressed; the body would have caught it. Two responses:
   the agent fetches the body and remediates; *and* a follow-up note marks the
   digest line as "needs expansion" so the next regeneration carries more.
3. **"Nobody could have known — the policies disagreed."** A real conflict.
   Surface to the human; do not auto-resolve. This is the truth-precedence case.

The first category should dominate by far in steady state. The second is the
natural feedback loop that improves the digest over time. The third is rare
and important.

### 4.6 What the framework needs to ship to make the agent-side work

Three additions on top of the policy primitive:

1. **`aiwf policy digest`** — generates the repo / skill / milestone digests in
   a standardized shape from the policy store. Idempotent, regeneratable,
   hash-verifiable.
2. **A digest-shape standard** — the one-to-three-line entry format, so the
   agent reads it the same way across repos.
3. **A digest-coverage feedback loop** — when a finding fires for a rule whose
   digest entry was too compressed, the regeneration considers expanding the
   line. Same provenance shape as ratification, applied to digest quality.

---

## 5. Autonomous runs and policy

The agent-side discussion in §4 implicitly assumes a human-in-the-loop session:
the user is present, can answer escalation prompts, can be the ambient
feedback channel. But the framework is increasingly used for *autonomous*
runs — "go run M-NN to completion" — where the agent and several spawned
subagents work for hours or days with the user not in the loop turn-by-turn.
This changes what the policy system has to do, and adds two design surfaces
the digest model alone does not cover.

### 5.1 The verifier becomes the agent's continuous-feedback channel

In a human-in-the-loop session, the human is the ambient feedback channel —
the agent does something off-target, the human notices and redirects.
**In an autonomous run, the verifier is the only feedback channel.** The
policy system has to be designed for this: verifier latency, finding-format
clarity, and the "fix → re-verify → confirm clean" loop all matter
materially more than they would for a periodic CI gate.

The verb shape:

```
aiwf policy verify --scope working-tree --quick
aiwf policy verify --scope changed-since-last-call
aiwf policy verify --scope milestone --full
```

The agent runs `verify` proactively — after any non-trivial change, before
any commit, before any handoff to a subagent or a wrap step. Findings come
back structured; the agent acts on them; the agent re-runs verify; the loop
tightens.

The kernel rule that follows: **`verify` is a kernel verb with a structured
output envelope, not just a CI step.** Per-call latency budget is in the
seconds, not minutes — which forces the [`02-policy-substrates-and-execution.md`](02-policy-substrates-and-execution.md)
§5 incremental-evaluation work (caching, scope-narrowing, fan-out) to be
real rather than aspirational. This is the autonomous-run design pressure on
the substrate layer.

### 5.2 Subagents consult the policy engine on their own

A subagent that the parent spawned to do a research pass, write tests, or
audit a code area should have the same digest access and the same `verify`
access as the parent. **Policy is repo-state, not session-state**, and a
subagent is just another agent operating on the same repo.

The cleanest model: the digest gets passed *as part of the subagent prompt*
(small enough by design, per §4.3), the verifier is callable as a tool inside
the subagent, and findings the subagent surfaces flow up to the parent in the
standard report channel. **No new kernel work** — the digest and the verifier
are already designed to be agent-shaped.

A worth-naming caveat: **subagent context bloat is real**. A subagent doing a
tightly-scoped task (find-the-symbol, write-this-test) doesn't need the whole
digest; it needs the slice for *its* scope. The skill that spawns the
subagent should *narrow* the digest to the subagent's task — same shape as
the milestone-scoped digest in §4.3, applied one level deeper.

The discipline rule that follows (skill-author convention, not kernel):
**spawning skills narrow the digest to the subagent's scope before passing
it.** A subagent prompt that blindly forwards the entire repo digest is wrong
in the same way a CLAUDE.md that loads every doc-body up-front is wrong — it
mistakes "carry everything" for "be informed."

### 5.3 The `wf-policy-sweep` skill convention

The natural skill the policy system makes possible — and the one autonomous
runs need most — is a *sweep at handoff boundaries*. Probably 30-50 lines,
KISS:

```
/wf-policy-sweep
```

What it does:
1. Runs `aiwf policy verify --scope working-tree --full`.
2. Classifies each finding: `block` / `warn` / `soft-signal` (per the
   policy's severity).
3. For *blocking* findings: refuses to hand off until cleared (fix or
   scoped waiver, the latter requires user consent).
4. For *warnings*: prompts the agent to acknowledge — fix or note in the
   handoff message.
5. For *soft-signal*: includes in the handoff report; does not block.

Where it runs:
- Before any non-trivial commit by an autonomous agent.
- Before a subagent reports back to its parent.
- Before `wrap-milestone` runs.
- Before opening a PR.
- On user request when they want a confidence check.

The shape generalizes the FlowTime `dead-code-audit` skill's *soft-signal
contract* and applies it to *all* policies, with severity gating the blocking
behavior. It is the natural skill the policy system makes possible, in the
same way `wrap-milestone` is the natural skill the entity model makes
possible.

The convention rule that follows (framework-shipped, consumer can override):
**`wf-policy-sweep` is the default tail of every meaningful step.** Skills
that don't run a sweep before their handoff should justify the omission, not
the inclusion. The framework ships it as a built-in skill, not a recipe; the
wrap rituals chain it by default.

### 5.4 The boundary moments — when the digest re-reads

In a long autonomous run, the agent may stay in context for many turns; its
view of what scope it's in can drift. The digest model handles this with one
simple discipline: **the milestone-scoped digest (§4.3) re-reads on internal
boundaries** — finishing an AC, switching from build to wrap, handing off to
a subagent. The boundary is the moment the agent might forget what scope it's
in; the re-read is cheap because the digest is small and stable by design.

This is partly why the editorial filter in [`04-policy-system-ux-mining-and-compression.md`](04-policy-system-ux-mining-and-compression.md)
§4 matters so much: a 3000-token digest is re-readable on every boundary; a
30000-token one is not. **Autonomous runs are the design pressure that keeps
the editorial filter honest.**

### 5.5 What this leaves to verify

The autonomous-run discussion adds three commitments not in §4:

- The verifier is a kernel verb with a structured envelope and a per-call
  latency budget in the seconds.
- Subagent skills narrow the digest before forwarding (discipline, not
  kernel).
- `wf-policy-sweep` is a framework-shipped built-in, chained by the wrap
  rituals by default.

Each is small and consistent with the §4 architecture. The pattern: the
framework ships the substrate (the verbs, the digest format, the sweep
skill); the consumer wires it into their own ritual chain.

---

## 6. Honest assessment: what's usable, what isn't

The two corpora produce different kinds of evidence. It is worth being clear
about which kind is ready to act on.

### 6.1 Directly usable now

- **The taxonomy and bucket model.** Four buckets (or five with agent-behavior
  named separately) survives second-corpus contact. The design session can
  build the policy primitive's schema against this bucket model and have
  defensible evidence the buckets are real.
- **The substrate selection.** Both corpora support the substrate doc's
  Option α (CUE/schema for static-shape + code-as-policy for dynamic + EARS
  prose for the human-facing layer). Cedar is unneeded; TLA+ is unneeded;
  Rego appears in <5 candidates per corpus. **The framework should commit to
  Option α as the v1 substrate posture, on this evidence, before the design
  session.**
- **The recurring shape catalogue.** Comment-as-policy in gitignore,
  doc-sweep-as-conflict-surfacing, per-milestone grep guards, two-step
  bootstrap+use skills, soft-signal contract, defense-in-depth via gitignore,
  shim-ban-with-named-trigger — all appear across both corpora and constrain
  the policy primitive's expressive surface.
- **The agent-side digest architecture.** The lazy-load → digest redesign is
  not corpus-derived per se, but it is forced by the corpus sizes (140-400
  active policies per repo). The design session can adopt the three-layer
  digest model as a working assumption.
- **Three to five worked policies fully ready to lift.** Specifically: the
  NaN policy (FlowTime), the schema-unification post-E-24 (FlowTime), the
  deletion-stays-deleted shell guards (FlowTime), the bind-me / inform-me
  taxonomy (Liminara), the bundle-as-contract discipline (Liminara). Each is
  mature enough to be a *first real policy* the system is built against.

### 6.2 Not yet usable

- **Most of the 540 combined entries are headlines, not specifications.** Each
  entry has a citation and a rung tag; few have the actual enforcement
  artifact extracted (the Roslyn diagnostic config, the schema fragment, the
  shell-guard regex, the test assertion), the deliberately-broken fixtures
  that prove the gate fires, the waiver / supersession history from git, or
  the cross-policy implications. The gap between "lifted" and "implementable"
  is real.
- **Rung tags are best-effort, not verified.** Items tagged "rung 2" may be
  rung 0 in practice (substrate exists but is misconfigured, never invoked in
  CI, or has known holes — the dead-code report itself flagged
  `binMinutes` in `ApiIntegrationTests.cs:93` as exactly such a hole in
  FlowTime). Verification needs running the substrates against contrived
  violations.
- **Lifecycle data is missing.** Both corpora are *snapshots*. The
  design-space §5 lifecycle (proposed → accepted → in-effect → waived →
  superseded → retired) needs *historical* evidence. `git log` would carry
  the supersession chains, the waiver patterns, the times rules changed. No
  pass extracted this.
- **Cross-repo generalization claims rest on two repos.** "The general bundle
  transfers cleanly" and "the two-population split is universal" hold for
  Liminara + FlowTime; they would benefit from at least one more
  comparison corpus before becoming a kernel commitment.

### 6.3 What would close the gap

Three follow-up surveys, in order of payoff:

1. **A vertical slice.** Pick one of the five worked policies above. Express
   it fully in the new policy primitive's shape — body + frontmatter + enforcer
   pointers + fixtures + lifecycle metadata. Run the verifier against it.
   Generate the digest entry from it. **End-to-end evidence the primitive
   holds, on one real policy.** Then repeat with a second worked policy and
   see what the primitive lacks. After three or four iterations, the primitive
   is real and the rest of the 540 entries become a backlog to migrate against
   a stable target. Days of work, not weeks.
2. **A `git log` lifecycle survey on the worked-policy slice.** For each of
   the three or four policies in the vertical slice, walk their commit history.
   When were they introduced? Renamed? Superseded? Waived? **This closes the
   lifecycle modeling question without surveying the whole corpus.** Hours of
   work, not days.
3. **A third corpus, only if needed.** If the first two follow-ups land
   without surprise, the policy primitive's shape is well-evidenced. A third
   corpus would test the *generalization* claims — but only worth doing if
   the design session arrives at a kernel-shape decision that hangs on
   them. Defer until then.

### 6.4 The honest framing

What the corpora produced is **good ground for a design session** — the
session that produces `docs/design/policy-model.md` from the design-space's
open questions. The corpora give the session concrete artifacts to test the
primitive against, four-to-five worked policies as test cases, and a
defensible substrate posture (Option α) to size the implementation against.

It is **not yet ground for an implementation session.** That session needs the
vertical slice and the git-log lifecycle survey before there is enough
evidence to commit to fields, runner-pointer shapes, and digest formats with
confidence.

---

## 7. The proposed next concrete step

If the targeted design session is some way off, the highest-value
intermediate step is the vertical slice: **pick the NaN policy from FlowTime,
express it fully in a candidate policy-primitive shape, verify the shape
against contact with the real artifact.**

Why the NaN policy specifically:
- It is mature, fully developed, and *named* — three explicit tiers, per-site
  enforcement table, named exception class for the PMF case.
- It has rung-2 enforcement (per-tier tests).
- It has rung-1 rationale (the doc).
- It has a "how to add a new site" workflow already written.
- It is the cleanest existing example of a "shape A" policy entity in either
  corpus.
- Expressing it would force the primitive to handle: tiered severity,
  per-site enforcement tables, a process-policy embedded in an
  engineering-policy, the rationale-and-runner binding, and the
  add-a-new-site lifecycle.
- The same exercise repeated on the bundle-as-contract policy from Liminara
  would force a different mix (CUE-bodied; ADR + schema + fixtures + worked
  example + ref impl) and immediately reveal whether the primitive accommodates
  both.

After the vertical slice, the design session has *one fully-worked policy* in
the new shape to argue from. The conversation can move from "what should the
primitive look like?" to "this is what the primitive looks like for X; what
does it lack for Y?" That is a much more productive conversation than the
abstract one.

---

## 8. What this leaves to the targeted design session

Most of `01-policies-design-space.md` §13's open questions remain open. This
exploration's contribution is bounded:

- **Settled, on the evidence:** the four-bucket-plus-one model is real; the
  schema-as-rung-3-backbone pattern is universal; Option α is the right v1
  substrate posture; the agent-side needs a generated digest, not lazy
  loading.
- **Constrained, but not settled:** the policy entity's required fields
  (informed by the seven recurring shapes); the `enforces[]` cardinality
  (must be 1..N); the lifecycle state set (informed by, but not yet derived
  from, git-log data the surveys did not collect); the precedence and
  conflict-resolution shape (varies by repo — must be a property of policy
  entities, not a framework axiom).
- **Open:** everything else. Particularly the form question (single kind vs
  discriminator on contract vs new ADR-superset), the cross-project
  portability mechanism, governance scope, and the supersession-graph
  semantics.

The vertical slice is what bridges from this exploration to the targeted
session. Without it, the session re-litigates the substrate question from
abstract evidence; with it, the session starts from "here is one policy
expressed end-to-end in the primitive — what changes?"

---

*This document is intentionally exploratory. The corpora it draws on live in
`.scratch/liminara/` and `.scratch/flowtime/`, gitignored and local-only. If
anything from those scratch directories graduates into framework-tracked
content, sanitize the project-specific identifiers first.*
