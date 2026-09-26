---
id: E-0096
title: One repo-only gate for everything aiwf ships to consumers
status: proposed
---
## Goal

Nothing aiwf ships to a consumer names aiwf's own entity ids, source paths, design
documents or development history, and one check that exists only in this repository
proves it on every push. "Is shipped content clean?" becomes a gate result rather
than a review.

## Context

The rule has four clauses, stated in `CLAUDE.md` §"Skills policy": a shipped surface
cites no real entity id, no filesystem path of this repo, no inline lifecycle status,
and carries no development history or rationale. Enforcement is split and partial:

- `skill-body-id` and `skill-body-claude-md-section` live in `internal/check`, which
  is compiled into the shipped binary (`go list -deps ./cmd/aiwf` lists it). Both are
  inert in a consumer repo, and the shipped `aiwf-check` skill documents them to
  consumers, naming this repo's `internal/skills/embedded…` trees.
- `PolicyCLITextCarriesNoInternalIDs` (`internal/policies/cli_text_internal_ids.go`)
  is repo-only and reads the whitespace-bearing Go string literals in the
  operator-text packages.
- Nothing checks paths or history in the embedded markdown (G-0548), and nothing
  reads the render stylesheet or the recipe tree.

Each clean-up so far scoped "shipped" by a list and left whatever sat outside it.
G-0538 records the first round: E-0078 purged its list and the Go-assembled text
survived. The Go-text patch for G-0538 left the skill prose and the stylesheet.

Measured on 2026-09-26 at `fde32ce8a` in the devcontainer, with a binary built from
that commit:

- All 47 `--help` screens: no real id, source path, `make` target or repository URL.
- The files under the seven embed roots (79 files): 14 lines citing this repo's
  source paths across six files, three citing design documents under `docs/`, ten
  lines of milestone ids in `internal/htmlrender/embedded/style.css`, and four
  self-references ("this repo's", "in the aiwf repo").
- `aiwf render --format=html` copies that stylesheet verbatim into the consumer's
  `site/assets/style.css`.

`internal/policies` is not in the binary. The repository's own pre-push hook
(`scripts/git-hooks/pre-push`, chained as `pre-push.local`) already runs one policy
test at push time.

## Scope

- A repo-only gate over everything aiwf ships, whose scanned set is derived rather
  than listed: every file under a `go:embed` directive in `internal/` and `cmd/`,
  plus the operator-text literals the CLI-text policy reads. It reports real ids,
  paths into this tree, design-document paths, and self-referential wording.
- Wiring the gate into the repository's pre-push hook and into CI.
- Cleaning every leak the gate reports, starting with the measured list above.
- Moving `skill-body-id` and `skill-body-claude-md-section` out of `internal/check`
  into the gate, and removing their rows from the shipped `aiwf-check` skill.

## Out of scope

- Enforcing source discipline in a consumer's own code (G-0526): a different
  audience and a different seam.
- The rendered HTML's entity content: in a consumer repo it renders the consumer's
  own tree.
- The `plugin.json` manifests naming the archived upstream rituals repository: no
  materializer reads them, so they do not reach a consumer.

## Constraints

- Nothing that exists only for this repository compiles into the shipped binary: the
  gate lives in `internal/policies` or a test-only package, never in `internal/check`.
- The gate fires no later than `skill-body-id` does today: at push, through the
  repository's pre-push hook, and in CI.
- The scanned set comes from the `go:embed` directives, so a new embed root is covered
  with no edit to the gate.
- A path is this repo's only when its first two segments exist in the tree, the
  CLI-text policy's rule, so an areas example such as `internal/billing` passes.
  `docs/adr/` stays allowed, since aiwf writes a consumer's ADRs there.
- Illustrations use the canonical `<prefix>-NNNN` placeholder. A deliberately
  exhibited bad shape carries a marked exemption (the classes G-0514 lists).
- A skill that must point at a consumer's own documents names them by shape, not by
  path (G-0587).
- `internal/check/hint.go` belongs to E-0092 until that epic lands; any change there
  waits for it.

## Success criteria

- [ ] No shipped-surface rule that exists only for this repository is in the
      binary: `skill-body-id` and `skill-body-claude-md-section` are gone from
      `internal/check` and from the shipped `aiwf-check` skill.
- [ ] A new file under any embed root that names a real aiwf id, a path into this
      tree or a design document under `docs/` fails the push, with no change to the
      gate.
- [ ] Every leak in the measured list under *Context* is gone, and the gate reports
      none on the tree.
- [ ] G-0538 and G-0548 are addressed.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| How much of the history clause a phrase list can carry, and what stays at review | no | Decided while building the gate |
| Whether the gate absorbs the CLI-text policy or sits beside it | no | Decided while building the gate |
| Whether G-0514's misclassifications are fixed as part of the move | no | Decided in the move |
| When the ids in `internal/check/hint.go` can be cleaned | yes, for closing G-0538 | After E-0092 lands |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| A derived scan set takes in data files (JSON examples, CSS) where an id-shaped value is legitimate | med | Per-file-type reading and marked exemptions, each with a reason |
| The move collides with E-0092's work in `internal/check` | med | Sequence the move after E-0092 lands |
| Skill text that explains a rule by naming this repo's source needs rewording rather than deleting, and the rewording can lose meaning | low | Each rewording reviewed against the rule it explains |

## Milestones

Allocated by `aiwfx-plan-milestones`; the candidates, in order:

- Build the derived repo-only gate and wire it into the pre-push hook and CI · depends on: —
- Clean every leak the gate reports · depends on: the gate
- Move `skill-body-id` and `skill-body-claude-md-section` out of `aiwf check` and clean the ids in `hint.go` · depends on: the gate, and E-0092 landing

## References

- G-0538 — operator-facing output cites internal ids and repo paths
- G-0548 — shipped surfaces cite this repo's filesystem paths
- G-0587 — a shipped skill cannot name the docs corpus a review must read
- G-0514 — `skill-body-id` misclassifies metavariables and non-id acronyms
- G-0526 — consumer source discipline ships as prose
- E-0078 — the earlier narrow-id purge of shipped surfaces
- `internal/policies/cli_text_internal_ids.go`, `internal/check/skill_body_id.go`,
  `scripts/git-hooks/pre-push`
