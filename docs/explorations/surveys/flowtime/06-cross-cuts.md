# Cross-cuts — patterns across the four buckets

> Not new policies — patterns the four bucket files reveal when read together. The
> things to bring back to the design-space exploration.

---

## 1. Where policy actually lives in this real repo

Tally of policy locations across the corpus:

| Location | # candidate policies | Avg rung |
|---|---|---|
| `CLAUDE.md` (single file) | ~50 | 1 |
| `.editorconfig` | ~10 | 2 |
| `.gitignore` (with comments) | ~6 | 2 |
| `.github/workflows/*.yml` (CI) | ~10 | 2 |
| `.claude/skills/*/SKILL.md` | ~20 (counting dead-code-audit's internal rules) | 1-2 |
| `docs/architecture/*.md` (NaN policy, run-provenance, etc.) | ~25 | 1-2 |
| `docs/schemas/*` | ~15 | 3 |
| `work/decisions/*.md`, `work/gaps/*.md`, `work/epics/*.md` | ~20 (the in-flight policy work itself) | varies |
| `work/guards/*.sh` | ~15 (per-deletion guards) | 2 |
| `aiwf.yaml` | 1 (version pin) | 2 |
| Source code asserts, throws, validators | many | 2-3 |
| Tests as enforcement | many | 2 |

**Observation:** policy is everywhere, in many shapes, with no unifying surface.
CLAUDE.md is the one document that *attempts* to be the umbrella, and it has the
weight to show for it (160 lines, dense). The framework's policy primitive should
*reduce* what CLAUDE.md has to carry, by giving the other items proper homes.

---

## 2. The rung distribution

Summed across all four bucket files (~140 entries):

| Rung | Count | % | Where it shows up |
|---|---|---|---|
| 0 (LLM memory only / open question / non-goal) | ~15 | 11% | Open questions, deferred decisions, deliberate non-goals |
| 1 (markdown reminder) | ~75 | 54% | CLAUDE.md, skill bodies, doc prose, ADR rationale |
| 2 (pattern lint / regex / AST / glob / shell guard / CI rule) | ~35 | 25% | `.editorconfig`, `.gitignore`, CI yaml, `work/guards/`, integration tests |
| 3 (schema / type) | ~12 | 9% | Model schema, template schema, type system |
| 4 (runtime contract / assertion) | ~3 | 2% | NaN policy invariants, `Pmf` exception |
| 5 (formal proof) | 0 | 0% | — |

**Observations:**

- **Rung 1 is 54% of the corpus.** Most policies live as prose; the prose is the
  *enforcement* until proven otherwise. This validates the design-space §4
  observation that rung 1 is the dominant category in the wild.
- **Rung 2 is large and well-developed (25%) when the substrate exists.** Roslyn
  analyzers, gitignore, CI gates, shell guards. The .editorconfig + CI combo is
  doing most of the rung-2 work.
- **Rung 3 (schema/type) is the strongest enforcement** for everything that fits
  in a closed grammar (model shape, template shape, manifest shape).
- **Rung 4 is rare and used for "this would crash badly otherwise"** sites — NaN
  cascade prevention, invalid-PMF construction.
- **Rung 5 (formal proof) is absent**, as the design-space doc anticipated.

The rung distribution tracks the design-space's enforcement spectrum exactly. The
honest read: **invest in rung-2 substrate (the linter/CI/grep-guard layer) and the
rung-1 prose layer in tandem; rung 3+ comes free where the data shape allows it.**

---

## 3. The five recurring shapes

Listed in [04-policies-rest.md §G] with names; restated here as a checklist for
the design session:

- **Shape α — Comment-as-policy in `.gitignore`** (rung-2 substrate + rung-1 explanation in a comment).
- **Shape β — Footprint analysis as policy evidence** (the *evidence* that supported ratification, preserved with the policy).
- **Shape γ — Doc sweep as conflict surfacing** (given a new policy, walk the repo for contradictions).
- **Shape δ — Per-milestone grep guards, named after the milestone** (deletion-stays-deleted; lifecycle bound to a milestone but enforcement lives forever).
- **Shape ε — Defense-in-depth via gitignore** (every repo declares its own producer/consumer asymmetry).
- **Shape ζ — Soft-signal contract** (a category of policy that surfaces but never blocks).
- **Shape η — Two-step skill: bootstrap + use** (configuration is a separate, reviewable step).

Each is a *pattern that recurs across the corpus*; the framework's policy primitive
should be expressive enough to capture each of them as a *type* without making the
schema huge.

---

## 4. The Truth Discipline section is the policy frontier

[01-policies-general.md §D §G-30..G-35] and [03-policies-workflow.md §B W-9..W-11]
both reach into the same CLAUDE.md section. It is the densest, most policy-shaped
prose block in the entire repo:

- It defines a **precedence order** (rung-1 conflict resolution).
- It defines **truth classes** (rung-1 categorization of where rules live).
- It enumerates **specific guards** (rung-1 prohibitions on common mistakes).

The Truth Discipline section is, all by itself, a *miniature governance document*.
It does not call itself one. **If the framework's policy primitive earns its keep
anywhere, it is in absorbing this section into a structured form** — precedence as
data, truth classes as labels on policy entities, guards as anti-pattern entries.

The exercise validates the design-space's "scope note: policy vs. governance"
distinction. Most of Truth Discipline is governance, not policy.

---

## 5. The framework's existing primitives cover most of the workflow bucket

[03-policies-workflow.md] has ~40 entries. Of them:

- ~15 are already enforced by aiwf today (FSMs, trailers, audit verbs, render).
- ~10 are FlowTime-specific extensions of generic patterns (TDD phase tracking; per-AC verbs).
- ~15 are still rung-1 prose because they are LLM-honor rules ("never commit without approval," conventional commit content, branch coverage audit).

The framework's *workflow* coverage is real. The framework's *engineering policy*
coverage is small (the meta-policies package + a few rules); the *project-specific
policy* coverage is essentially zero (consumer repos invent their own).

The honest sequencing implication: **the framework's next big move is not a deeper
workflow story but a substrate for engineering and project policies** — the §A-§C
of [05-skills-needed.md].

---

## 6. The dead-code-audit skill is a working blueprint

It is, on close reading, a complete small-scale policy framework:

- Recipe-driven (one config per substrate).
- Bootstrap step that *configures* the framework before use.
- Soft-signal contract.
- Per-stack tools (Roslynator, knip, vulture, etc.).
- Structured findings with classes (confirmed-dead, tool-flagged-but-live, intentional, needs-judgement).
- Blind-spot sweep that the LLM does *because* the tool can't.
- Anti-patterns explicitly named.
- Output is overwritten on every run; findings worth keeping graduate to gaps.

If the framework's policy primitive is hard to imagine concretely, the
dead-code-audit skill is what it would look like — generalized from one subject
("dead code") to many ("this is a policy of subject X, evaluated by tool Y, with
soft-signal vs blocking severity, with per-substrate recipes").

The team built one; the framework should generalize the pattern.

---

## 7. Open question: what does the framework do about open questions?

The corpus has at least three explicit open policy questions sitting unresolved:

1. **P-8 / R-12** — "Is the snake_case telemetry-manifest deliberate, or drift?" — flagged in dead-code report; no path to ratification.
2. **R-13** — "When does the class-2 capacity-aware allocator ship?" — deferred-follow-up gap (G-NNN to be filed in M-066).
3. **W-22 / W-23** — "Pre-aiwf v1 *-tracking.md / *-log.md files in archived epics — what to do with them?" — G-035 tracks but doesn't resolve.

Each is a **policy-shaped question without a policy entity to live in**. The
framework should distinguish between:
- A **question** (something that could go either way; needs deliberation).
- A **proposal** (a candidate answer, not yet ratified).
- A **policy** (ratified, in effect).

Today the corpus uses *gaps* for questions, *milestones* for the work to answer
them, and *ADRs* / *decisions* / *prose* for the answers. That works, but it
splits the lifecycle across three entity kinds. **The §13 design-session question
"is policy really an umbrella?" is partly answered by this**: yes, because three
existing kinds together still leave gaps (the open-question stage is poorly modeled).

---

## 8. The policy that everything else hangs off

If forced to pick one policy from the entire corpus that, if solved well, would
make the most others easier to solve, it is:

> **Every policy carries its *why* and its enforcement substrate together.**

The corpus has many policies whose prose explains *why* but doesn't say *how it's
checked*; many lints / shell guards / schema rules that *check* but don't say
*why*; and many entries that have neither. When a policy goes wrong (the regex
breaks, the warning is ignored, the rule is forgotten), the recovery cost is
proportional to how disconnected the *why* and the *how* were when the policy was
written.

The framework's policy primitive's single highest-value commitment is to bind
*why* and *how* into one structured artifact. Everything else in the design-space
exploration follows from that.

---

## 9. Sanitization notes (for later)

If anything from this scratch graduates into framework-tracked content:

- Replace `flowtime` / `flowtime-vnext` / `FlowTime.*` with a generic project name.
- Replace `aiwf v3` with the framework version current at writing.
- Strip per-milestone ids (`M-066`, `E-25`, `G-033`, `D-053`, etc.); replace with
  abstract identifiers.
- The NaN policy and run-provenance examples are technical and may be useful as
  worked examples; cite the source if used externally.
- The grep-guards script body is generic enough to use as an example as-is, with
  the project-specific symbol names redacted.
- The dead-code-audit skill is already framework-shaped; it can be referenced
  directly if its license allows.
