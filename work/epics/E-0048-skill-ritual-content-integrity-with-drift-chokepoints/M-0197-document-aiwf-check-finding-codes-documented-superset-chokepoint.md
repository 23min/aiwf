---
id: M-0197
title: Document aiwf-check finding codes + documented-superset chokepoint
status: done
parent: E-0048
depends_on:
    - M-0196
tdd: required
acs:
    - id: AC-1
      title: Every emitted finding code is documented in the aiwf-check skill
      status: met
      tdd_phase: done
    - id: AC-2
      title: A documented-superset chokepoint binds the skill to the emission sites
      status: met
      tdd_phase: done
---
## Goal

The `aiwf-check` skill exists to document the finding codes `aiwf check` emits —
it is the channel an AI assistant or operator consults to interpret a finding it
did not recognize ("what does this code mean, how do I fix it"). Its documented
set had drifted: a dozen check-layer codes were undocumented, including the
high-stakes branch-choreography findings `isolation-escape` (and its three
subcodes) and `promote-on-wrong-branch`, exactly the codes whose interpretation
guidance matters most. Nothing mechanical bound the skill's documented set to the
emission sites, so a new finding code could ship undocumented and no chokepoint
noticed.

This milestone (a) documents every currently-undocumented check-layer code in the
skill, and (b) adds a **documented-superset chokepoint**: a Go policy test that
enumerates the emitted finding-code set and fails if the skill omits any of it.
The enumerator is the one `PolicyFindingCodesHaveHints` already uses to walk
`Finding{}` emission sites and resolve `Code*` constants — extracted to a shared
helper (`emittedFindingCodeSites`), so "what is emitted" has a single source of
truth and the two chokepoints cannot drift. That shared enumerator is hardened to
also resolve the typed `codespkg.Code{ID: …}` descriptors (referenced as
`Code: CodeXxx.ID` selectors, same- and cross-package), which the prior version
silently skipped — this is why the branch-choreography findings were invisible to
it.

The chokepoint lives as a Go policy test (CI tier), not an `aiwf check` finding,
because it enumerates Go `Code*` declarations by AST — meaningless in a consumer
tree where `internal/check/` is absent and the skill is materialized rather than
authored. It scopes to check-layer emission sites (`internal/check/`,
`internal/cli/check/`); verb-layer codes (e.g. `import-collision` from
`aiwf import`) ride the shared emitted set for the hint chokepoint but are out of
the aiwf-check skill's scope. The drift is one-directional (omissions only); the
check enforces documented ⊇ emitted, with a rationale-annotated opt-out for the
synthetic test-fixture codes.

Source: G-0283. Parent epic E-0048.

## Acceptance criteria

### AC-1 — Every emitted finding code is documented in the aiwf-check skill

The `aiwf-check` skill body (`internal/skills/embedded/aiwf-check/SKILL.md`)
carries a meaning + remediation entry, in the correct severity table, for every
finding code the check layer emits. The twelve check-layer codes undocumented at
the milestone's start are each added: `acs-tdd-audit`, `acs-title-prose`,
`body-prose-id`, `id-path-consistent`, `milestone-done-incomplete-acs`,
`skill-body-id`, `git-config-core-worktree-misset`, `isolation-escape`,
`isolation-escape-shallow-clone`, `isolation-escape-oracle-failure`,
`isolation-escape-orphaned-ai-commit`, and `promote-on-wrong-branch`. (The
foundation codes the planning estimate also listed — `acs-shape`,
`acs-body-coherence`, `refs-resolve`, `fsm-history-consistent` — were already
documented via their subcode rows; three verb-layer codes surfaced by
`aiwf import` / `aiwf add`, not `aiwf check`, are deliberately out of scope.)

Mechanical evidence: the AC-2 chokepoint — which asserts the skill's documented
set is a superset of the emitted check-layer set — passes. Per CLAUDE.md "AC
promotion requires mechanical evidence", the AC-2 guard is the structural
assertion that fails if any code named here loses its skill entry; there is no
separate substring test for AC-1.

### AC-2 — A documented-superset chokepoint binds the skill to the emission sites

A Go policy test under `internal/policies/` enumerates the emitted finding-code
set and fails if any emitted code is absent from the aiwf-check skill's
documented set. The emitted set is the union of the `Code*` string constants and
the typed `codespkg.Code{ID: …}` descriptors used at `Finding{}` construction
sites across `internal/check/` and `internal/cli/check/`, enumerated through the
**same** walker `PolicyFindingCodesHaveHints` uses (extracted to a shared helper
so the two chokepoints read one source of truth). A rationale-annotated opt-out
list carves out the synthetic test-fixture codes (`a-err`, `z-warn`) that are
never surfaced to a user. Adding a new emitted code without documenting it in the
skill reddens the gate.

Test: (1) a firing fixture — drive the policy against a documented set with one
emitted code removed and assert exactly one violation naming that code (this also
lights the policy's `Violation` construction line for the G-0259 firing-fixture
meta-gate, so no new `grandfatherDark` entry is owed); (2) a seam test that the
shared enumerator, run over the real check packages, includes a typed-descriptor
code (`isolation-escape`) in its emitted set — proving the `Code: CodeXxx.ID`
selector resolution added here actually reaches the branch-choreography findings,
the gap that kept them undocumented; (3) `PolicyFindingCodesHaveHints` continues
to pass over the live tree after the enumerator refactor — no regression to the
hint chokepoint that shares the walker.

## Work log

Per-AC phase timeline lives in `aiwf history M-0197/AC-<N>`; this log records the final outcome only.

### AC-1 — Every emitted finding code is documented in the aiwf-check skill
Added 12 check-layer code rows to the aiwf-check skill (6 error-table, 6 warning-table), each placed by its source `Severity:`. Evidence: `TestPolicy_FindingCodesDocumentedInSkill` green over the live tree. commit 9dc23a7b.

### AC-2 — A documented-superset chokepoint binds the skill to the emission sites
New `PolicyFindingCodesDocumentedInSkill` + shared `emittedFindingCodeSites` (extracted from `PolicyFindingCodesHaveHints`) with `.ID` selector resolution (same/cross-package) + `codespkg.Code{ID:…}` descriptor loading; added 4 hints so the hint chokepoint stays green. Firing / seam / discrimination / dedup / opt-out / enumerator-edge tests. commit 9dc23a7b.

## Decisions made during implementation

- **Check-layer scope, not all `Finding{}` sites.** The shared enumerator walks every emission site (the hint chokepoint requires each finding a hint); the doc chokepoint filters to `internal/check/` + `internal/cli/check/`, because the aiwf-check skill documents `aiwf check` findings, not verb-layer ones (`import-collision`, `import-duplicate-id`, `slug-dropped-chars`).
- **Document dynamically-subcoded codes as bare rows.** `body-prose-id` / `skill-body-id` emit a runtime `Subcode` variable the AST can't resolve, so the enumerator sees them bare; documented as bare code-family rows alongside the existing subcode rows.
- **Shared enumerator forced 4 hint additions.** Resolving `.ID` selectors made 4 branch-choreography codes newly visible to `PolicyFindingCodesHaveHints`; added their hints to keep it green — a latent remediation gap the `.ID`-blindness had hidden.

No `aiwfx-record-decision` ADRs were needed — these are scoping choices within the milestone, not cross-cutting architectural decisions.

## Validation

- `go test ./internal/policies/ ./internal/check/ ./internal/cli/check/` — green.
- `make check-fast` (build / vet / lint / full suite) — green.
- Diff-scoped coverage gate — clean on all changed lines (the 7 flagged lines covered by added tests).
- Firing-fixture meta-gate (G-0259) — the new policy's `Violation` construction line is covered; no new `grandfatherDark` entry.
- `skill-body-id` realtree — green (no real-id leak; `ADR-0010` / `M-0125` / `G-0155` reworded out of the shipped skill rows).
- Independent adversarial reviewer — [verdict recorded in Reviewer notes].

## Deferrals

- **G-0331** — `aiwfx-plan-epic` + `aiwfx-record-decision` lack a structural-test backstop (M-0195 residue, surfaced during M-0197's `make coverage-gate` run). Owned by M-0201 (recorded in the epic body's milestone→gap plan list); the skill-edit backstop ratchet forces its two tests when M-0201 edits those skills.

## Reviewer notes

- **Independent adversarial reviewer: APPROVE, no blocking findings.** All 8 load-bearing claims verified by measurement, not reasoning. Highlights: a delete-a-row experiment reddened the live-tree policy naming the exact same- and cross-package `.ID` emission sites (proving the resolver reaches the branch-choreography findings); all 12 severity placements cross-checked against each emission site's `Severity:` (zero misplacements); the discrimination/dedup/edge tests confirmed genuine (a gutted policy fails them); branch-coverage audit + firing-fixture meta-gate both pass over the live tree.
- **Track-for-later (non-blocking), with a bound the reviewer did not have:** cross-package `.ID` resolution keys on the bare descriptor name (`codeConsts[x.Sel.Name]`). This is *not* a live collision risk today: `loadCheckCodeConstants` scans only `internal/check/` — a single Go package — so its `Code*` names are unique by language rule, and the cross-package references from `internal/cli/check/` all resolve against that one namespace. The fragility would appear only if the scan were widened to multiple packages defining same-named descriptors; worth a guard or a one-line note if that ever happens.
- **12 vs 14 codes:** the planning estimate listed 14; the AST enumerator's actual undocumented check-layer set is 12 — 4 planning-listed codes (`acs-shape`, `acs-body-coherence`, `refs-resolve`, `fsm-history-consistent`) were already documented via subcode rows, and 3 verb-layer codes were scoped out. The Goal/AC-1 prose was corrected at wrap to match.
- The bare `body-prose-id` row coexists with the existing `body-prose-id/malformed-shape` subcode row — intentional (family overview + specifics); the same holds for any dynamically-subcoded code.
