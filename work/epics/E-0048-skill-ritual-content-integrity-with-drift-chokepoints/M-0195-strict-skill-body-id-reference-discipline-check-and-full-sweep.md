---
id: M-0195
title: Strict skill-body id-reference discipline, check, and full sweep
status: in_progress
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
      status: open
      tdd_phase: done
    - id: AC-5
      title: Standing rule documented in CLAUDE.md Skills policy
      status: open
      tdd_phase: red
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
carried by a markdown link that resolves under `docs/**` (the design/ADR
doc-link carve-out). This is the deliberate inversion of the `body-prose-id`
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
while flagging real ids.

Evidence: a test asserting no non-canonical placeholder shape remains in any
shipped skill body.

### AC-4 — Full sweep: id-reference check passes clean over the shipped skill tree

Every real entity id, id-bearing filesystem path, and inline lifecycle-status
assertion is removed from all shipped skill bodies (the 34 `SKILL.md` files
across the two embedded trees); the AC-1 rule reports zero findings when run
over the real embedded skills. This single assertion pins both the sweep (no
real refs remain) and normalization (only valid placeholders remain). Because a
leaked filesystem path or inline status almost always carries a real id, the
id-shape rule transitively catches the common leakage forms; the residual
path/status discipline beyond id-shapes lives in the AC-5 standing rule.

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

