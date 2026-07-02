---
id: M-0195
title: Strict skill-body id-reference discipline, check, and full sweep
status: done
parent: E-0048
depends_on:
    - M-0209
tdd: required
acs:
    - id: AC-1
      title: Skill-body id-reference check fires pre-push over the embedded skill tree
      status: met
      tdd_phase: done
    - id: AC-2
      title: Canonical placeholders, code, and ADR doc-link carve-out stay silent
      status: met
      tdd_phase: done
    - id: AC-3
      title: All placeholders normalized to canonical width
      status: met
      tdd_phase: done
    - id: AC-4
      title: 'Full sweep: id-reference check passes clean over the shipped skill tree'
      status: met
      tdd_phase: done
    - id: AC-5
      title: Standing rule documented in CLAUDE.md Skills policy
      status: met
      tdd_phase: done
---
## Goal

Make skill-body id-reference discipline strict and mechanical. Shipped skill
bodies (`internal/skills/embedded/**` verb skills and
`internal/skills/embedded-rituals/**` rituals) must cite no real entity id,
filesystem path, or inline lifecycle status: they ship to consumer repos where
aiwf's own ids are meaningless, and they rot as entities change status, archive,
or rewidth. Illustrative content uses canonical-shape placeholders
(`<prefix>-NNNN`) or shape-descriptions; a markdown link to a design or ADR doc
is the one carve-out.

A new pre-push `aiwf check` rule over the embedded skill tree makes the
discipline mechanical — the standing-rule prose is the convenient version, per
the kernel principle "framework correctness must not depend on LLM behavior".
The rule is the mirror image of the `body-prose-id` check (G-0184): there a real
id is required and a placeholder is the defect; here a real id is the defect and
the canonical placeholder is correct.

Sequence: normalize all placeholders to canonical width first (the precondition
that lets the check allow placeholders while flagging real ids), then sweep
every skill body, then land the check green over the swept tree. This is a
foundation milestone — the content milestones (verb-skill corrections, ritual
honesty, prose polish, planning-ritual body-fill) rebase onto the swept bodies,
so the id-hygiene rewrite lands once, first.

Source: G-0299. Parent epic E-0048; sequenced after foundation epic E-0050
(done), whose generalized declared-sequence gate governs this milestone's wrap.

## Acceptance criteria

### AC-1 — Skill-body id-reference check fires pre-push over the embedded skill tree

A new rule in `internal/check` scans every `SKILL.md` under
`internal/skills/embedded/**` and `internal/skills/embedded-rituals/**`, reuses
the existing CommonMark prose-mask so code spans, fenced blocks, and link
destinations are exempt by construction, and emits a finding for any
digit-bearing strict-form id token in skill-body prose (the per-kind canonical
shapes, bare and composite). A new finding code is registered in the closed-set
constants. The rule runs inside `aiwf check`, so the catch is pre-push and
in-context — the earliest tier its class allows per the epic's timeliness
criterion; the CI policy layer is a backstop, not the catch.

Evidence: a check-package test with a fixture skill body containing a real id
(e.g. `M-0001`) makes the finding fire; a clean body stays silent.

### AC-2 — Canonical placeholders, code, and ADR doc-link carve-out stay silent

The check's allow-set stays silent: letter-form placeholders (the canonical
`<prefix>-NNNN` token, whose suffix is the letter N, distinct from a
digit-bearing real id), id-shapes inside any code construct, and an id-shape
carried by any markdown link destination — `proseMask` exempts link
destinations regardless of path, which is the mechanism behind the design/ADR
doc-link carve-out (the author keeps the id in the destination and the visible
text descriptive). This is the deliberate inversion of the `body-prose-id`
check, which flags `M-NNNN` in entity bodies; in a shipped skill body the
placeholder is correct and the real id is the defect.

Evidence: a fixture body with a `G-NNNN` placeholder, a backticked real id, and
a `[doc-link](docs/adr/ADR-…md)` carve-out stays silent; the same body with a
bare inline real id makes the finding fire.

### AC-3 — All placeholders normalized to canonical width

Every placeholder across the embedded skill tree is normalized to the canonical
`<prefix>-NNNN` shape. Idiosyncratic forms (`G-XYZ`, `ADR-WXYZ`, the fabricated
`ADR-OPSPEC-01`), narrow legacy widths (`E-NN`, `M-NNN`, `D-NNN`, `ADR-NNN`),
and pseudo-arithmetic (`C-NNN+1`) are eliminated; where two placeholders
co-occur in one example they are made distinct. Normalization is the
precondition for the AC-1 check: it is what lets the rule allow placeholders
while flagging real ids. **Prose-scoped**, like the check: the test masks code
constructs, so narrow metavariables that survive only inside command-syntax
examples (`aiwf history M-NNN`) are deliberately out of scope — they are
syntactic illustration, not entity references, and the check exempts them.

Evidence: a test asserting no non-canonical placeholder shape remains in
shipped skill-body prose.

### AC-4 — Full sweep: id-reference check passes clean over the shipped skill tree

Every real entity id, id-bearing filesystem path, and inline lifecycle-status
assertion is removed from shipped skill-body **prose** (across the 34 `SKILL.md`
files in the two embedded trees); the AC-1 rule reports zero findings when run
over the real embedded skills. This assertion pins both the prose sweep (no real
refs remain) and normalization (only valid placeholders remain). Because a
leaked filesystem path or inline status almost always carries a real id, the
id-shape rule transitively catches the common leakage forms; the residual
path/status discipline beyond id-shapes lives in the AC-5 standing rule.

The guarantee is **prose-scoped**: the check (and this test) mask code
constructs and link destinations, because code carries both leakage *and*
legitimate syntax-teaching format-examples (e.g. `body-prose-id`'s own `M-0001`
hint) the masker cannot tell apart. A real id used as a *reference* inside a
code example is an authoring discipline caught at review, not mechanically — the
independent wrap review caught two such cases (`G-0018`, `G-0071` inside
backticks) and the sweep cleaned them to placeholders.

Evidence: a test that runs the AC-1 rule over the real embedded skill tree and
asserts zero findings.

### AC-5 — Standing rule documented in CLAUDE.md Skills policy

The strict discipline is written into CLAUDE.md's Skills-policy section: shipped
skill bodies cite no real entity id, filesystem path, or inline lifecycle
status; illustrative content uses canonical `<prefix>-NNNN` placeholders or
shape-descriptions; a markdown link to a design or ADR doc is the one carve-out.
The prose names the mechanical chokepoint (the AC-1 check) as the guarantee and
itself as the convenient version.

Evidence: a section-scoped structural assertion that the rule text appears under
the Skills-policy heading — not a flat file-wide grep — per CLAUDE.md
§"Substring assertions are not structural assertions".

## Work log

- **AC-1 / AC-2** — `internal/check/skill_body_id.go`: `ScanSkillBodyID` byte
  scanner + `skillBodyIDReference` tree-walk (inert in consumer repos), wired
  into `check.Run`; `CodeSkillBodyID` constant + `hint.go` entry. Tests:
  `skill_body_id_test.go` (11 scanner cases + dedupe + a `check.Run` seam test).
- **AC-3 / AC-4** — full sweep of 23 `SKILL.md` bodies (≈117 real-id citations
  removed, placeholders normalized); `skill_body_id_realtree_test.go` asserts the
  real tree is clean (AC-4) and prose placeholders are canonical (AC-3). The
  high-volume sweep ran via a Sonnet builder subagent, reviewed for prose quality.
- **AC-5** — CLAUDE.md Skills-policy standing rule + enforcement-table row;
  `internal/policies/m0195_skill_body_discipline_test.go` (section-scoped).

Phase + status timeline per `aiwf history M-0195/AC-N`. Implementation lands in
the single wrap commit (current bundle-at-wrap model).

## Decisions made during implementation

- **Check lives in `internal/check`, not `internal/policies`** — resolves the
  epic's open question toward the pre-push (in-context) tier per the
  chokepoint-timeliness criterion; inert in consumer repos (skill-source tree
  absent).
- **The guarantee is prose-scoped.** The check reuses `body-prose-id`'s
  `proseMask`, so code constructs and link destinations are exempt — necessary
  because code carries both real-id leakage *and* legitimate syntax-teaching
  format-examples (e.g. `body-prose-id`'s own `M-0001` hint) the masker cannot
  distinguish. Real ids inside code examples are an authoring discipline caught
  at review, not mechanically.
- **ADR citations preserved as doc-links; provenance/exemplar ids removed.** The
  sweep collided with prior discoverability ACs — ADR references kept via the
  doc-link carve-out (id in destination), milestone/gap provenance + exemplar ids
  dropped. The whiteboard tier rubric was reworked to archetype-lead (operator
  decision).
- **Streamlined TDD promote cadence** adopted by operator direction (the
  per-phase HITL gate is low-value for local, mechanically-grounded transitions);
  a config knob is tracked in `G-0314` (filed on `main`).

## Validation

- `go test ./internal/check ./internal/skills ./internal/policies` — all green
  (policies full suite ≈77s).
- `go build ./...`, `go vet ./internal/...` — clean. `golangci-lint` on changed
  packages — 0 issues.
- `aiwf check` (worktree diag binary) — 0 errors; `skill-body-id` reports 0 over
  the swept tree.
- Diff-scoped coverage: `ScanSkillBodyID` 100%, `skillBodyIDReference` 95.8% (one
  `//coverage:ignore` TOCTOU guard).
- Independent two-lens review (fresh-context reviewer): REQUEST-CHANGES → 1
  blocking (B1: two real ids in code constructs + overstated prose) resolved
  (content swept to placeholders + prose made prose-scoped-honest); 2 non-blocking
  — N2 fixed (AC-2 prose), N1 recorded below.

## Deferrals

- **`G-0315`** (`--discovered-in M-0195`) — ritual-skill ADR doc-links have broken
  relative depth and are dead in consumer repos; questions the doc-link carve-out
  for shipped skills. Filed on `main`.
- **N1 — pre-push false-negative** — a real bare id glued to a malformed `/AC-x`
  or `_` tail (`M-0001/AC-foo`, `M-0001_x`) escapes the pre-push check (the
  combined token matches neither strict pattern). The CI-tier AC-3 real-tree test
  *does* flag every such shape, so the tree stays safe; only the in-context
  pre-push catch has the hole. Low priority — recorded here; file a gap if it
  recurs.

## Reviewer notes

- `tdd: required`; every AC carries a structural test (red→green verified) per
  the mechanical-evidence rule.
- The guarantee is **prose-scoped** (see Decisions) — a deliberate, documented
  limitation, not an oversight. AC-3 likewise: command-syntax metavariables
  (`aiwf history M-NNN`) in code blocks are out of scope.
- The sweep rippled into 7 prior ACs' tests; each was legitimately re-pointed
  (the independent review verified none were gutted to a tautology), not weakened.
- N1 is an accepted check limitation (the CI tier covers it).

