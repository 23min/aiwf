---
id: G-0481
title: 'Narrow-ID eradication: block minting, purge examples, retire rewidth'
status: addressed
priority: high
addressed_by_commit:
    - db307fdc7
---
# Narrow-ID eradication: make minting impossible, purge example debris, retire `rewidth`

## Goal

Reach a **canonical-only kernel**: (1) no `aiwf` verb can create a below-canonical-width id, (2) no narrow id appears as an example in any shipped or normative surface, and (3) the `rewidth` migration verb is retired. ADR-0008 completed the *runtime* migration — input tolerance, allocator emit, drift check, and `prior_ids` resolution are all correct and load-bearing — but it deliberately deferred the example/prose cleanup, and one creation path and one guard blind spot were never closed. This gap is the audit and the umbrella for the follow-on epic.

The concrete cost: narrow ids modelled as "current" in the surfaces an LLM reads while operating aiwf degrade the assistant's behavior. Every migration target is a real tree that has already been rewidthed, so **a narrow id in an active tree is now a bug**, not a pre-migration state.

## Audit — creation paths (can a verb mint a narrow id?)

| Path | Emits | Verdict |
|---|---|---|
| `AllocateID` (`internal/entity/allocate.go:81`) | `%s%0*d` at `CanonicalPad=4` | canonical ✓ |
| `reallocate` | via `AllocateID` | canonical ✓ |
| `rewidth` | widens to canonical | canonical ✓ (slated for deletion) |
| **`import`, explicit-id entry for a new entity** (`internal/verb/import.go:133`) | **stores the manifest's raw id verbatim** | **NARROW MINT HOLE** |
| `import`, explicit-id entry on `--on-collision=update` (`:150`) | uses the existing entity's on-disk id | correct — must target the real file |
| `import`, auto-id entry (`:180`) | via `AllocateID` | canonical ✓ |
| hand-edit / `git mv` | n/a (not a verb) | outside allocator scope; caught post-hoc by the drift check |

`entity.Canonicalize` is applied to the reservation key and the collision lookup (`:114`, `:130`), so a manifest declaring `` `E-11` `` collision-checks canonically — and then stores `` `E-11` `` in frontmatter and builds `work/epics/E-11-<slug>/epic.md`. Closing it is a canonicalization at the point of assignment.

The fix is *not* an AST rule. Extending `mint_ids_via_allocate` to catch verbatim id writes has no structural signature to match on — there is no way to recognize "this string literal is an entity id" from syntax, so any such rule is a proxy that both misses and false-positives. The guarantee belongs in an **output-property invariant test**: drive every id-creating verb end to end and assert canonical width in both the stored `id:` and the resolved path.

## Audit — Tier A, shipped surfaces (highest priority)

Everything under `internal/skills/embedded{,-rituals,-guidance}/` materializes into consumer repos via `aiwf init` / `aiwf update`.

| Shape | Hits | Files |
|---|---|---|
| Narrow numeric (`` `M-001` ``, `` `E-01` ``) | 51 | 14 |
| Narrow placeholder (`` `E-NN` ``, `` `M-NNN` ``, `` `D-NNN` ``) | 152 | 24 |
| **Tier A total** | **203** | **28** (union) |
| (already-canonical `` `<prefix>-NNNN` `` forms, for contrast) | 213 | — |

The narrow-placeholder population is the larger half and the more damaging one: it concentrates in the ritual skills that *drive* planning (`aiwfx-start-epic` says `` `E-NN` `` 29 times, `aiwfx-wrap-epic` 27), so an assistant following the ritual learns the wrong id shape.

Two further defects the numeric grep does not surface, both violations at any width:

- **Real-id citations in shipped surfaces.** `aiwf-edit-body/SKILL.md` cites M-0058 and `aiwf-acknowledge/SKILL.md` cites E-0021. Both are `done` and archived — the exact rot the skills policy predicts. Widening them fixes nothing; they must go.
- **Entity templates seed the rejected shape.** `templates/milestone-spec.md` ships `` `M-NNN` ``, `` `E-NN` ``, `` `D-NNN` ``, `` `G-NNN` `` — and `internal/check/body_prose_id.go:10-11` names precisely that shape ("uppercase placeholder leaks") as a thing it rejects in entity bodies. The frontmatter and H1 occurrences are inert (`aiwf add` stamps the real id over them); the prose-guidance lines are author-facing text that can survive into a committed body, where the shipped check then fires on it.

**Sweep target is `` `<prefix>-NNNN` `` uniformly** — not canonicalized digits. Canonical-width fictional ids (`` `E-0001` ``) are real entities in most consumer trees, which is the collision the placeholder convention exists to prevent. The fictional worked-example transcripts in `aiwf-status` and `aiwf-list` lose some vividness under this; their titles carry the distinctions, and a guard with no transcript carve-out is worth more than the vividness across a ~200-site sweep.

**Allowlist — three files, all citing narrow ids as the subject of a rule rather than as examples of current shape:**

- `aiwf-check/SKILL.md`'s grammar table, which must cite `` `M-a` ``, `` `M-NNNN` ``, `` `M-1` ``, and `` `M-0001` `` to document the `body-prose-id` rule.
- `aiwfx-plan-epic/SKILL.md` (`:93`) and `aiwfx-plan-milestones/SKILL.md` (`:135`), which sanction `` `M-1` ``/`` `M-2` ``/`` `M-3` `` as conversational shorthand for not-yet-allocated milestones. The convention depends on narrow width being the marker that distinguishes a casual label from an allocated id, so these six hits are load-bearing for the rule they teach and survive the sweep unchanged.

## Audit — Tier B, repo-facing docs

In scope: `README.md` (19 hits) and `docs/workflows.md` (85) — 104 total. These are the docs an assistant reads to learn the workflow, and their narrow ids are teaching examples whose fix is the placeholder form.

Deferred to a separate gap (71 hits): `docs/design/**` (56, 5 files), `docs/overview.md` (12), `docs/architecture.md` (3). These are largely citations of entities that were genuinely real at narrow width, so the correct fix is widening to the real canonical id — a different and lower-value edit than the teaching-example cleanup, and one that would bloat the width lint's allowlist.

## Audit — out of scope

- **Test fixtures.** A large share of the narrow `id:` values under `internal/check/testdata/` and the `check`/`verb` packages *are* the narrow-read-tolerance suite — coverage of a property the kernel keeps permanently. `narrow_id_sweep_test.go`'s allowlist already records which files are load-bearing that way. Canonicalize only incidental uses, opportunistically.
- **Code comments** (`htmlrender/paths.go`, `roadmap.go`, `cli/list/list.go`, …) — no consumer reach, low payoff.
- **Frozen by convention:** `docs/archive/**`, `docs/research/**`, `docs/explorations/**`, `CHANGELOG.md`, ADR-0008 itself.

## Audit — why the debris persisted

`skill-body-id` (`internal/check/skill_body_id.go`) is already the right rule: it scans all three embedded trees whole-file including frontmatter, covers templates and agent cards, runs at error severity pre-push, is inert in consumer repos, and already holds the inverted polarity these surfaces need (`:8-10` — a real digit-bearing id is the defect, a canonical letter-N placeholder is correct). All 203 hits survive it for two reasons, and closing them is the whole guard:

- **It masks code constructs.** `ScanSkillBodyID` runs `proseMask` (`:78`), blanking code spans and fenced blocks — where every one of the 51 numeric hits lives. It needs a mask that does not exempt them, distinct from `proseMask`, which `body-prose-id` shares and legitimately needs unchanged.
- **It declines width.** `:65` states that "canonical letter-N placeholders … are not this rule's concern (placeholder normalization is policed separately)." No such policing exists anywhere in `internal/check` or `internal/policies`; nothing validates that a letter-N placeholder carries four N's. That is why the 152 narrow placeholders accumulated under an otherwise-working rule.

A polarity trap for whoever implements it: `body-prose-id` rejects `` `M-NNNN` `` as a leak in entity bodies while `skill-body-id` *requires* it in shipped surfaces. Opposite rules for opposite domains, both currently correct.

The `:65` case is an instance of E-0076's pattern — a rule stated in an authoritative surface with no detector behind it, which reads as enforced and so stops the next reader from looking. That epic addresses three unrelated instances and does not cover this one; the shared shape is worth seeing from both sides.

Two lesser blind spots: `narrow_id_sweep_test.go` matches only quote-adjacent literals and exempts `testdata/` outright; active docs have no narrow-id shape check at all (Tier B needs one, and it is genuinely width-shaped — a different rule over a different corpus, where real ids are legitimate).

## `rewidth` retirement is a net deletion

Every tree the verb existed to migrate has been migrated, so a narrow id in an active tree is a defect rather than an unfinished migration. That collapses the drift check rather than complicating it: `entityIDNarrowWidth`'s mixed/uniform classifier exists only to stay silent on a uniform-narrow tree — the pre-migration state — which no longer occurs. The rule becomes "any narrow id in the active tree fires," at error severity, remediated by undoing the hand-edit or `git mv` that produced it (`reallocate` would assign a different number, not widen the same one).

Retirement deletes `internal/verb/rewidth.go`, `internal/cli/rewidth/`, their tests, the three registration sites in `internal/cli/root.go` (`:44`, `:185`, `:259`), and `padToCanonical` — a fourth independent id-shape parser, outside the three in `internal/entity` that G-0454 tracks, so it is one fewer site an id-grammar change must touch without altering that gap's scope. Its `mint_ids_via_allocate` allowlist entry goes with the file. What it adds is a superseding ADR over the ADR-0008 clauses that specify the verb, plus an edit to CLAUDE.md's commitment #2 and the `aiwf-check` skill's mention. `this_repo_drift_check_clean_test.go` keeps passing against a stronger assertion.

## Narrow read tolerance is permanent

`rewidth` skips archive subtrees by design (`internal/verb/rewidth.go:29`), so every migrated repo that archived before migrating still holds narrow ids under `<kind>/archive/` — and once the verb is deleted, nothing can ever widen them. The drift check's archive exclusion is therefore permanent, and read tolerance (`Canonicalize`, `IDGrepAlternation`, `prior_ids` resolution, narrow commit trailers) is load-bearing for resolving **live cross-references into archived entities**, not merely for `aiwf history` on old trailers. It is a standing property of the input space, not a legacy concession.

## Phased approach

**Phase 1 — close the mint hole. Standalone patch, not part of the epic.** Canonicalize the id at `internal/verb/import.go:133`; add the end-to-end canonical-width invariant test over every id-creating verb. Independent of everything below.

**The epic — four milestones, the first two strictly ordered, the last two independent of both.**

1. **Close the two `skill-body-id` holes**: a non-code-masking mask, and the placeholder-width check `:65` already promises. Land at warning severity so the sweep gets a worklist instead of a blocked push.
2. **Tier A sweep** — 203 sites to `` `<prefix>-NNNN` ``, the two real-id citations removed, the templates fixed; flip the rule to error at the end.
3. **Tier B** — the width-shaped lint over `README.md` and `docs/workflows.md`, plus its 104-site sweep. File the deferred-residue gap for the remaining 71.
4. **Retire `rewidth`** — the deletions above, the collapsed drift rule at error severity, the superseding ADR, CLAUDE.md #2.
