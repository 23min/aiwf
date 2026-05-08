# FlowTime policy mining — overview

> Scratch material extracted from `flowtime-vnext` to feed the policies-as-primitive
> exploration. **Not** sanitized for publication: project name and entity ids appear
> verbatim. If anything from here graduates into framework docs, neutralize the
> `flowtime` / `aiwf` / `M-NNN` references first.

## What this is

A first-pass corpus of every "rule-bearing" artifact found in `flowtime-vnext` — all
the places that say "must / should / may / never" about how the codebase or how the
work itself must operate. Each item is a candidate policy. The point is not to claim
each one *is* a policy; the point is to see what shape the territory has when you
sweep one real, mid-size, mid-aged repo and try to lift it.

The mining pass deliberately **did not** invent new rules. Every item below is a
direct lift from a tracked file, with the source location cited. Where I had to
paraphrase to compress, I marked the item as paraphrased.

## Why FlowTime

Properties that make this a useful test repo for the policy exploration:

- **Polyglot stack** — .NET 9 (engine, services, Blazor UI), TypeScript+Svelte (modern UI),
  Rust (evaluation core), Python tooling. Forces the policy taxonomy to deal with
  multiple substrates at once, not one happy-path language.
- **Real planning surface** — uses `aiwf v3` (the *previous* generation of the framework
  this exploration sits inside), so the work-tracking layer is already structured into
  epics, milestones, decisions, gaps, contracts, ADRs. Workflow policies are explicit
  and visible, not buried in chat.
- **Multiple service surfaces** — Engine API (`:8081`), Sim API (`:8090`), Blazor UI
  (`:5219`), Svelte UI (`:5173`). Cross-surface contracts already exist; cross-surface
  policies emerge naturally.
- **Schema-driven contracts** — `docs/schemas/*.json` and `model.schema.yaml` already
  exist as canonical machine-readable contracts; we can see what *isn't* in schema.
- **Active epic with policy-shaped work** — E-25 Engine Truth Gate is, on its own
  reading, a policy-ratification epic (flow-authority policy + ADR + schema/compile/
  analyse enforcement points). M-066's spec is a real, in-flight worked example of
  the design-space's "rung-1 + rung-2 + rung-3" stacking.
- **Explicit shell guards** — `work/guards/*.sh` files contain literal grep-based
  rung-2 checks that "every deleted symbol stays deleted" — exactly the *deletion-
  stays-deleted* policy shape.
- **A skill that is itself a policy framework in miniature** —
  `.claude/skills/dead-code-audit/` is recipe-driven, polyglot, multi-rung, and
  produces structured findings. Worth studying as prior art the framework
  shouldn't reinvent.

## Method

1. Read top-level engineering files (`CLAUDE.md`, `.editorconfig`, `.gitignore`,
   `aiwf.yaml`, `STATUS.md`, `ROADMAP.md`).
2. Read CI (`.github/workflows/build.yml`).
3. Read the `.claude/` skill set (the embedded skills, not the aiwf-managed ones).
4. Read a representative slice of `docs/`: schemas index, ADR-shaped
   architecture docs (`nan-policy.md`, `run-provenance.md`).
5. Read a representative slice of `work/`: guards, decisions, gaps, an epic +
   milestone (E-25 / M-066).
6. Bucket every rule-shaped sentence into one of four categories.
7. Tag each with a candidate enforcement rung (0..5, per the design-space doc).
8. Note where the rule already has rung-2+ enforcement and where it sits at rung-1.

## Files in this directory

- `00-overview.md` — this file.
- `01-policies-general.md` — applies to any repo of this stack/scale (codebase,
  language, build, test, doc hygiene).
- `02-policies-project-specific.md` — only meaningful inside FlowTime (flow-authority,
  NaN policy, run-provenance, schema alignment, port topology, etc).
- `03-policies-workflow.md` — belongs in the aiwf / PM domain (TDD discipline, branch
  conventions, milestone wrap, dead-code audit, conventional commits, principal/
  agent/scope provenance).
- `04-policies-rest.md` — items that don't fit cleanly into the three buckets, plus
  observations about boundary cases.
- `05-skills-needed.md` — based on the stack and the kinds of work, sketches which
  skill / policy bundles would emerge naturally; ties back to the design-space §3
  categories.
- `06-cross-cuts.md` — patterns across all four buckets: rung distribution, what's
  enforced where, what's currently rung-1 prose that could move to rung-2, what's
  rung-2 with no rung-1 explanation.

## Counts at a glance

The category-by-category lists below total roughly **140 distinct rules** lifted
from the corpus, distributed approximately:

| Bucket | Count (~) | Avg rung today |
|---|---|---|
| General (engineering, language, test) | 50 | mostly 1 (some 2 via .editorconfig + CI) |
| Project-specific | 45 | mixed — schema rules are 3, NaN policy is 2-via-tests, most are 1 |
| Workflow / PM | 35 | mostly 0-1; aiwf provides some 2 (FSM checks, hash, refs-resolve) |
| Rest / boundary | 10 | n/a |

Numbers are rough; the per-category files have the actual entries.

## Caveat

A first pass; not exhaustive. Likely-missed surfaces: the `templates/*.yaml` template
schemas and their per-template validation rules; the `examples/*.yaml` model files
(which encode "valid model" by example); `tests/fixtures/` (golden inputs); the
Rust crate's `Cargo.toml`/`clippy` config; `ui/`'s `tsconfig.json`/`eslint`/`vitest`
config. Each is another seam where rules live.
