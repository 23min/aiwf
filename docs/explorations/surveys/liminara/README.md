# `.scratch/liminara/` — policy-corpus mining

Output of mining `/Users/peterbru/Projects/liminara` for policy candidates, to feed the policy-substrate PoC. The repo is gitignored (see `.gitignore`); content here is local-only.

**Why this corpus exists:** to answer "what does a real body of rules / guards / decisions look like in a working repo, and what would it take to express them through the policy primitive sketched in `docs/explorations/policies-design-space.md` and `policy-substrates-and-execution.md`?"

**Final answer location:** [`40-categorized.md`](40-categorized.md) — start there.

---

## TL;DR

- **~625 raw policy candidates extracted** across three mining passes; ~350–400 unique after de-dup.
- **Bucket split** (the user-asked four buckets, plus a fifth that surfaced):
  - Bucket A — General engineering: ~140
  - Bucket B — Project-specific (Liminara): ~180
  - Bucket C — Workflow / PM (aiwf domain): ~70
  - Bucket D — Agent-behavior: ~90
  - Bucket E — Rest: ~25
- **Enforcement rung distribution:** ~75% rung-1 (prose only), ~10% rung-2 (lint/format), ~10% rung-3 (CUE schemas), ~5% rung-4 (tests). The CUE layer is the bright spot; everything else is prose-claimed-MUST sitting at rung 1.
- **Substrate fit:** roughly 30 rules are CUE-shaped (and 5 are already CUE), ~80 are EARS+runner shaped, ~40 are code-as-policy shaped, ~150 are pure-prose / judgment-driven. ~50 are descriptive, not normative.

## Top-line takeaway for the PoC

The corpus supports the substrate exploration's **Option α** (CUE for static-shape policies + code-as-policy escape hatch + RFC 2119 / EARS for the prose layer). It also supports the parent doc's claim that **the umbrella earns its keep most strongly for project-specific policies (Bucket B)**, where the framework's existing `.ai-repo/rules/` + `docs/governance/` + ADRs already form a proto-policy layer that lacks a unifying lifecycle / provenance frame.

The single most prevalent failure mode in the corpus is exactly the one §4 of the parent doc names: **MUST claims at rung 1 only.** The doc-tree taxonomy, contract matrix, shim ban, frontmatter requirements, and bundle-as-contract discipline all assert MUST in prose with no machine check.

A high-leverage PoC seven-policy slice (each is one runner + one EARS sentence, each has a partial implementation in the wild today):

1. **Bundle-as-contract** verifier (ADR + schema + fixtures + worked example + ref impl coexist; no TBD).
2. **Schema-evolution loop** — every committed historical fixture validates against current HEAD schema.
3. **Frontmatter required-keys** for `docs/architecture/` + `docs/history/`.
4. **Contract-matrix row-paths-resolve** — every row's live-source file exists.
5. **Compatibility-shim with named-trigger** — code-grepped removal comment cross-refs tracking doc.
6. **Two-pack-citation** rule for ADRs (Radar primary, admin-pack secondary).
7. **Branch-coverage audit before commit** — language-specific coverage runner.

## Files in this directory

| File | What it is | Size |
|---|---|---|
| [`00-survey.md`](00-survey.md) | Structural survey of the liminara repo: layout, AI scaffolding, enforcement, docs, tech stack, submodules, workflow layer | 477 lines |
| [`10-ai-scaffolding-policies.md`](10-ai-scaffolding-policies.md) | 338 policy candidates extracted from CLAUDE.md, `.ai/`, `.ai-repo/`, agent files, sample skills | 1139 lines |
| [`20-project-docs-policies.md`](20-project-docs-policies.md) | 287 policy candidates from `docs/governance/`, ADRs, CUE schemas, operational guides, roadmap, sampled epics + architecture | 416 lines |
| [`30-enforcement-mechanisms.md`](30-enforcement-mechanisms.md) | 15 enforcers across 4 rungs; what they enforce, how, with what blocking severity; gaps named | 315 lines |
| [`40-categorized.md`](40-categorized.md) | The synthesis: four-bucket split, cross-cuts by rung / substrate / bindingness, observations | ~480 lines |
| `README.md` | this file — fast-read summary | this file |

## Methodology notes

1. **Mining was structural, not exhaustive.** Every rule in CLAUDE.md, every rule in `.ai-repo/rules/`, every governance doc, every ADR's *Decision* section, every CUE schema's invariants. Sample-only for the 21 framework skills (4 representative ones) and the long-tail architecture docs.
2. **Candidate format was uniform across mining passes:** ID-shaped slug, one-sentence rule (paraphrase), source path, bindingness (RFC 2119 framing), audience, category guess. The categorization in `40-categorized.md` re-buckets these against the four-buckets-plus-one frame.
3. **Three mining agents ran in parallel** to keep the main session's context window clean. Each wrote its corpus to its own file. Synthesis happened in the main session.
4. **Read-only.** Nothing in `/Users/peterbru/Projects/liminara/` was modified.

## Sanitization needed before any of this lands in a public artifact

Every file in this directory cites `liminara` by name and references named entities specific to that project: Radar, admin-pack, VSME, ex_a2ui, dag-map, proliminal.net, the five named CUE schemas (op-execution-spec, wire-protocol, plan, manifest, replay-protocol). If a future PoC artifact wants to use any of this corpus as illustrative material, replace:

- `liminara` → `<consumer-repo>` or a fictitious name
- `Radar`, `admin-pack`, `VSME` → generic pack names
- The five CUE schemas → one or two illustrative schemas (e.g., `event-stream` and `pack-manifest`)
- Specific file paths under `runtime/`, `docs/decisions/`, `work/epics/` → generic equivalents

The *shape* of the corpus is what generalizes; the content is one specific repo's invariants and won't make sense outside it.
