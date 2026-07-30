---
id: G-0481
title: 'Narrow-ID eradication: block minting, purge examples, retire rewidth'
status: open
priority: high
---
# Narrow-ID eradication: make minting impossible, purge example debris, retire `rewidth`

## Goal

Reach a **canonical-only kernel**: (1) no `aiwf` verb can create a below-canonical-width id, (2) no narrow id appears as an example in any shipped or normative surface, and (3) the `rewidth` migration verb is retired. ADR-0008 completed the *runtime* migration (own tree is 100% canonical; input tolerance, allocator emit, drift check, and `prior_ids` resolution are all correct and load-bearing) — but it deliberately deferred the example/prose cleanup, and one creation path and one policy blind spot were never closed. This gap is the comprehensive audit and the umbrella for the follow-on work (epic-sized).

Narrow ids modelled as "current" in the surfaces an LLM reads while operating aiwf degrade the assistant's behavior — the concrete cost that motivates this.

## Audit — creation paths (can a verb mint a narrow id?)

| Path | Emits | Verdict |
|---|---|---|
| `AllocateID` (`internal/entity/allocate.go:81`) | `%s%0*d` at `CanonicalPad=4` | canonical ✓ |
| `reallocate` | via `AllocateID` | canonical ✓ |
| `rewidth` | widens to canonical | canonical ✓ (but slated for retirement) |
| **`import` explicit-id entry** (`internal/verb/import.go:261`, path at `:292–310`) | **stores `fm["id"] = pe.id` and builds the filename from `pe.id` verbatim** | **NARROW MINT HOLE** |
| `import` auto-id entry (`:182`) | via `AllocateID` | canonical ✓ |
| hand-edit / `git mv` | n/a (not a verb) | out of allocator scope; caught post-hoc by the drift check |

The one live hole: a manifest declaring a narrow id (`import-format.md` even shows `` `E-11` ``, `` `M-001` ``) creates a narrow file **and** narrow frontmatter. It slips past `mint_ids_via_allocate` because that policy only forbids `%0*d`-style minting outside `AllocateID`; a verbatim string write is invisible to it. Closing this — canonicalize the stored id and path on import — plus extending the policy to cover verbatim id writes, plus a creation-time invariant test, is what makes minting *mechanically impossible* rather than merely conventional.

## Audit — example / prose debris surfaces

Counts are approximate; the point is scope, not the exact integer.

| Tier | Surface | Narrow example ids | Ships to consumers | Priority |
|---|---|---|---|---|
| A | Embedded verb skills (`internal/skills/embedded/aiwf-{status,list,add,promote,show,rename,render,edit-body,retitle}/SKILL.md`) | ~40 across ~10 files | **yes** | highest |
| B | Human/normative docs — `README.md` (~19), `docs/workflows.md` (~86), `docs/overview.md` (~12), `docs/design/**` (~60), `docs/architecture.md` | ~180 | repo-facing | high |
| C | Code-comment illustrative examples (`htmlrender/paths.go`, `roadmap.go`, `cli/list/list.go`, `verb/reallocate.go`, …) | ~2 dozen + real-entity self-references | no | low |
| D | Test fixtures — raw-frontmatter `id:` blobs (heaviest in `check`/`verb` pkgs), 18 `internal/check/testdata/` trees, `m0089` goldens | ~190 `id:` values + goldens | no | medium |
| — | Gray zone — `docs/research/**` (~135), `docs/explorations/**` (~193): historical/exploratory but **not** under `archive/`, so not formally frozen | ~330 | no | low |
| — | Leave frozen — `docs/archive/**`, `CHANGELOG.md`, ADR-0008, `rewidth`/canonicalization implementation comments | — | — | keep |

**Keep (intentional teaching cases, must survive any sweep):** `aiwf-check/SKILL.md` (explains the narrow-`` `M-1` `` rejection); `aiwfx-plan-epic`/`aiwfx-plan-milestones` (sanction `` `M-1` ``/`` `M-2` `` as conversational planning shorthand); `CLAUDE.md`'s planning-shorthand and canonicalization clauses.

## Audit — why the debris persisted (guard blind spots)

Each existing guard has a hole the debris lives in — a one-time sweep will rot again unless these close:

- **`skill-body-id` / `body-prose-id` are prose-scoped and mask code spans / fenced blocks.** Nearly every narrow id in the shipped skills sits in an inline-code command or a fenced transcript/JSON block — so the check by design never sees it. This is why skill *examples* escaped.
- **`narrow_id_sweep_test.go` (M-081) matches only quote-adjacent literals** (`ByID("E-14")`) and **exempts `testdata/` outright** — so the mid-string fixture `id:` values and golden trees are invisible.
- **Active docs and code comments have no narrow-id shape check at all.**
- **`mint_ids_via_allocate` misses the import verbatim-write path** (above).

## Audit — `rewidth` deprecation blast radius

`rewidth` has no dedicated skill (good). Retiring it touches:

- **`internal/check/entity_id_narrow_width.go:102`** — the drift finding's remediation literally says "run `aiwf rewidth` to complete the migration." This is the crux: the finding *exists to point at `rewidth`*. Retiring the verb requires a replacement remediation — which forces a policy decision (below).
- Command registration (`internal/cli/root.go`), the `aiwf-check` skill mention, `CLAUDE.md` commitment #2, and the ADR-0008 clauses that specify the verb.

## Load-bearing decision — the fate of narrow *read* tolerance

The three goals fully apply to **writes and examples**, but **narrow *read* tolerance likely cannot ever be fully removed**: git history contains narrow commit trailers (`aiwf-entity: E-22`), `prior_ids` lineage, and real code comments that reference pre-migration entities — `aiwf history` on a narrow id must keep resolving unless git history is rewritten. So the realistic end-state is:

**strict write (minting impossible, incl. import) · all examples purged · `rewidth` retired after a legacy cutoff · a minimal, permanent narrow-*read* tolerance retained for historical resolution.**

Retiring `rewidth` therefore requires deciding the legacy-consumer story: either (a) declare narrow trees unsupported past a named version and reword the drift finding to "migrate before upgrading to vX," or (b) keep the drift finding as an advisory-only note with no verb to point at. This decision gates phase 3 and should be recorded (ADR superseding the relevant ADR-0008 clauses).

## Proposed phased approach (epic candidate)

1. **Make minting impossible.** Canonicalize import's stored id + filename; extend `mint_ids_via_allocate` (or a sibling policy) to catch verbatim id writes; add a property/invariant test asserting every id-writing verb emits canonical width. Fixes the `import` hole and turns the convention into a guarantee.
2. **Purge examples + install a regression guard.** Sweep Tiers A+B to canonical / `<prefix>-NNNN` placeholder form; add a doc/skill narrow-id-*shape* lint over embedded surfaces + active docs (allowlisting the teaching cases) so the debris cannot return. Then Tiers C+D (comments, fixtures — the fixture cleanup can ride a `rewidth`-on-`testdata` pass or fixture-builder change).
3. **Retire `rewidth`.** Record the legacy-cutoff decision, reword the drift-finding remediation, remove/deprecate the command, update `CLAUDE.md` #2 and supersede the affected ADR-0008 clauses. Keep the minimal narrow-read tolerance.

Related: **G-0454** (unify the id-shape parsers) is adjacent convergent-duplication cleanup that phase 1 can fold in. This gap subsumes the narrow-example portion the earlier whiteboard tracked informally.
