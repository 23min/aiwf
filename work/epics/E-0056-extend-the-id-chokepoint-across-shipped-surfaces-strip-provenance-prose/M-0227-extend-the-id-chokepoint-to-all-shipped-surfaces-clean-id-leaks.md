---
id: M-0227
title: Extend the id chokepoint to all shipped surfaces; clean id leaks
status: in_progress
parent: E-0056
tdd: required
acs:
    - id: AC-1
      title: Broadened markdown scan fires on real ids; placeholder silent
      status: met
      tdd_phase: done
    - id: AC-2
      title: Statusline comment scan fires on real ids; shell code exempt
      status: met
      tdd_phase: done
    - id: AC-3
      title: Code, fenced, and link-destination carve-outs preserved
      status: met
      tdd_phase: done
    - id: AC-4
      title: Whole shipped tree green under the broadened check
      status: met
      tdd_phase: done
---
## Goal

The id chokepoint flags a real aiwf-internal id in any shipped surface — the
`description:` frontmatter, entity templates, role-agent cards, the guidance
fragment, and the statusline's comments — not just `SKILL.md` bodies. Every
existing id leak the broadened check would fire on is removed in the same
change, so the check is green and the leak class is mechanically closed.

## Approach

Broaden the scan in `internal/check/skill_body_id.go`:

- Include the `description:` field — parse it out of the frontmatter and scan it
  with the same masked-prose pass used on the body.
- Walk every materialized `*.md` under `embedded` / `embedded-rituals` (drop the
  `SKILL.md`-only filter), covering entity templates and role-agent cards.
- Add `internal/skills/embedded-guidance/` to the scanned roots.
- Add a comment-scoped scan for `internal/skills/embedded-statusline/*.sh` — the
  markdown `proseMask` does not apply to shell, so scan `#` comment text for
  strict id-shapes, leaving shell code exempt.

Keep code spans and link destinations exempt (unchanged carve-out). Then clean
the leaks the broadened check now fires on: rewrite the statusline comments to
drop the id/provenance tags, the `aiwfx-start-epic` description to drop the
`ADR-0023` / `E-03` references, and the `epic-spec.md` template's `E-0002`
example to a placeholder shape.

Implementation note: the scan is whole-file per `*.md` (not a separate
`description:` extraction) — one masked-prose pass over the whole file catches
the `description:` field, a template's frontmatter comment, and the body
uniformly, and needs no body-relative line-offset math.

## Acceptance criteria

Formalized at start-milestone into the four ACs below. The original sketch's
first criterion (a firing fixture per newly-covered surface) split into **AC-1**
(the `*.md` surfaces) and **AC-2** (the statusline shell scan) because they are
distinct code paths; the exemption criterion became **AC-3** and the
clean-tree criterion became **AC-4**.

### AC-1 — Broadened markdown scan fires on real ids; placeholder silent

A real (digit-bearing) aiwf id planted in a `description:` frontmatter field, an
entity template, a role-agent card, or the guidance fragment produces a
`skill-body-id` finding; a canonical letter-N placeholder in the same position is
silent. Mechanical evidence: `TestSkillBodyIDReference_BroadenedSurfaces` (four
surfaces × fires/silent, driven through `check.Run`, asserting the file-relative
line so the whole-file scan carries no offset regression) and
`TestSkillBodyIDReference_SkipsNonMarkdown` (both arms of the `*.md` filter) in
`internal/check/`.

### AC-2 — Statusline comment scan fires on real ids; shell code exempt

A real id in a `#` comment of a shipped `embedded-statusline/*.sh` file produces
a finding; a placeholder is silent, and a real id in shell *code* — a string
literal, a `${x#…}` parameter expansion, `$#` — is exempt (the shell analogue of
the code-span carve-out). Mechanical evidence: `TestShellCommentMask` (every arm
of the comment-detection rule), `TestShellCommentMask_PreservesShape` (the
same-length, newline-preserving mask contract), and
`TestStatuslineCommentIDReference_Seam` (through `check.Run`) in `internal/check/`.

### AC-3 — Code, fenced, and link-destination carve-outs preserved

Broadening the scanned surface does not defeat `proseMask`: a real id inside an
inline code span, a fenced block, or an ADR doc-link destination — including
inside the newly-scanned `description:` field — produces no finding. Mechanical
evidence: `TestSkillBodyIDReference_CarveOutsPreserved` in `internal/check/`, a
regression lock that goes red only if a future mask change breaks a carve-out.

### AC-4 — Whole shipped tree green under the broadened check

The full shipped tree carries no real-id leak under the broadened check —
`check.Run` over the repo root yields zero `skill-body-id` findings — after every
leak the audit named (the two descriptions, the template example, and the
statusline comments) is cleaned. The finding's human-readable text is generalized
from "skill body" to "shipped surface" to match the broadened scope, the finding
code unchanged. Mechanical evidence: `TestSkillBodyID_WholeShippedTreeClean` in
`internal/check/`, which drives the production `*.md` and statusline walkers over
the real tree.

## Work log

- **AC-1** — `81e039f0` (red fixtures) + `a4226d0a` (whole-file Design B `*.md`
  scan + coupled cleanup of the `aiwf-list` / `aiwfx-start-epic` descriptions and
  the `epic-spec` template). Met.
- **AC-2** — `4093e33f` (seam test) + `4bbc3e74` (`shellCommentMask` + walker +
  statusline comment cleanup) + `b314d94f` (metacharacter-boundary fix from the
  wrap review). Met.
- **AC-3** — `f7cf3445` (carve-out regression lock). Met.
- **AC-4** — `8fca6f55` (whole-tree seam test) + `6ebc1f46` (finding-text
  generalization + `aiwf-check` doc row). Met.
- Formalized AC bodies filled in `b531d17f`.

## Validation

- `go test ./...` — all packages pass; `go build ./...` clean.
- `golangci-lint run` — 0 issues; `go vet` clean; `gofumpt` clean.
- `aiwf check` (worktree binary, real tree) — 0 `skill-body-id` findings, 0
  errors overall.
- Branch coverage: `scanMaskedForRealIDs` / `shellCommentMask` / `shellWordBoundary`
  / `ScanSkillBodyID` at 100%; the two tree walkers' only uncovered lines are their
  `//coverage:ignore` TOCTOU read guards.

## Reviewer notes

- Independent fresh-context review (code-quality lens): **approve**. Every AC claim
  verified by measurement — plant-and-check a real id into a real shipped surface,
  adversarial white-box probing of `shellCommentMask`, coverage inspection.
- One non-blocking finding fixed in-context (`b314d94f`): `shellCommentMask` missed
  comments after a shell word-boundary metacharacter (`;#`, `)#`, …) — a silent
  false-negative in a leak detector. Now recognized via `shellWordBoundary`.
- **Non-goal (not a deferral):** the placeholder-canonicality axis is deliberately
  not extended to the newly-scanned surfaces — agent cards and templates use
  narrow-N placeholders (`E-NN`, `M-NNN`) a canonicality sweep would flag. This
  milestone pins real-id *firing* only.
- `Field: "body"` is hardcoded on findings even for `description:` / statusline
  hits — harmless (path + line + message carry the locator), consistent with the
  pre-existing rule.
- The `CLAUDE.md` §"Skills policy" paragraph still under-describes the broadened
  check (missing `-guidance`, the statusline scan, the whole-file surfaces). That
  broadening is milestone `M-0228`'s declared scope, not a deferral of this one.

## Deferrals

None. No AC was deferred or cancelled; no work was punted to a gap.
