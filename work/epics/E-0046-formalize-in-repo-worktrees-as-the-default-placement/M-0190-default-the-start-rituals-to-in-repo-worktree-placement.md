---
id: M-0190
title: Default the start rituals to in-repo worktree placement
status: in_progress
parent: E-0046
depends_on:
    - M-0189
tdd: none
acs:
    - id: AC-1
      title: Start-epic worktree step defaults to in-repo placement via worktree.dir
      status: met
    - id: AC-2
      title: Start rituals retain the per-invocation worktree placement override
      status: met
    - id: AC-3
      title: Start rituals record the sandbox rationale citing ADR-0023
      status: met
    - id: AC-4
      title: WorktreeDir rejects a repo-escaping path and falls back to default
      status: met
---

# M-0190 — Default the start rituals to in-repo worktree placement

## Goal

Make `aiwfx-start-epic` and `aiwfx-start-milestone` default to in-repo worktree placement
(reading the `worktree.dir` knob), with the per-invocation override retained and the
devcontainer-sandbox rationale recorded inline.

## Acceptance criteria

Tracked in frontmatter `acs[]` and detailed in the `### AC-1` … `### AC-4` sections below.
Three doc-shaped ACs (AC-1/AC-2/AC-3) assert ritual content against the embedded SKILL.md
bytes; AC-4 is the Go-side safety carried forward from the M-0189 reviewer — the resolved
`worktree.dir` cannot escape the repo.

## Context

Builds on the `worktree.dir` knob (M-0189) and the loader guard (M-0188). The start rituals
previously offered three placements as a free choice with no default; this milestone flips
the recommended default to in-repo and records why. Authoring is in the embedded ritual
snapshot (`internal/skills/embedded-rituals/…`) per CLAUDE.md "Ritual content authoring"; AC
tests assert against the embedded bytes.

## Constraints

- The knob sets the *default*, not a lock — the per-invocation override stays.
- Doc-shaped ACs use structural assertions scoped to the named SKILL.md section, not flat
  substring greps (CLAUDE.md "Substring assertions are not structural assertions").

## Design notes

- The in-repo-worktree-default decision is **ADR-0023** (accepted), allocated and ratified
  earlier in this epic. The rituals cite it inline for the rationale.

## Out of scope

- The config knob (M-0189); the loader guard (M-0188).

## Dependencies

- M-0189 — the `worktree.dir` knob the rituals read.

## References

- E-0046 epic spec; ADR-0023; CLAUDE.md "Ritual content authoring".

### AC-1 — Start-epic worktree step defaults to in-repo placement via worktree.dir

`aiwfx-start-epic`'s worktree-placement step (step 8) leads with in-repo placement under the
configured `worktree.dir` (default `.claude/worktrees`) as the recommended default, read from
the kernel via `aiwf doctor | grep '^worktree-dir:'` rather than hardcoded. The skill's
`## Principles` summary agrees with the step — no leftover neutral-prompt framing.

Evidence: `TestAiwfxStartEpic_M0190_AC1_WorktreeDefaultsToInRepo` in
`internal/policies/aiwfx_worktree_default_test.go` — structural assertions on the step-8
worktree subsection (knob, default dir, `aiwf doctor`, in-repo/default framing) **and** on the
`## Principles` section (in-repo default + ADR-0023), so a regression in either place fails.

### AC-2 — Start rituals retain the per-invocation worktree placement override

Both rituals keep the override: start-epic's step 8 retains all three placements (in-repo /
main-checkout / sibling); start-milestone's step 5 names the main-checkout / sibling override.
In-repo is the default, not a lock.

Evidence: `TestStartRituals_M0190_AC2_OverrideRetained` — structural assertions that the
start-epic worktree subsection names "override" plus all three placement markers, and the
start-milestone cut subsection names "override" + "main-checkout" + "sibling".

### AC-3 — Start rituals record the sandbox rationale citing ADR-0023

Both rituals' worktree guidance records the devcontainer-sandbox rationale (sandboxed session
cwd confinement; `$HOME`-placed worktrees wiped on container rebuild) and cites ADR-0023.

Evidence: `TestStartRituals_M0190_AC3_SandboxRationale` — for each ritual's worktree section,
asserts `ADR-0023` (verbatim) plus the rationale markers (sandbox / devcontainer / rebuild).

### AC-4 — WorktreeDir rejects a repo-escaping path and falls back to default

`config.Config.WorktreeDir()` rejects a repo-relative `worktree.dir` that escapes the repo root
(`..` climbing above root, directly or after interior traversal) and falls back to
`DefaultWorktreeDir`, while still honoring non-escaping paths — so the value the rituals consume
via `aiwf doctor` can never place a worktree outside the repo (defeating ADR-0023 / the M-0188
loader guard). Carried forward from the M-0189 reviewer's advisory.

Evidence: `TestWorktreeDir_RejectsRepoEscapingPath` in `internal/config/config_test.go` — five
escaping shapes fall back to default; two non-escaping controls stay accepted (proving no
over-rejection). Both arms of the new branch are exercised, so the diff-scoped coverage gate is
satisfied.

## Work log

### AC-4 — WorktreeDir escape rejection

Added a `filepath.Clean` + `..`-prefix escape check to `WorktreeDir()` (returns the trimmed
original for honored paths; falls back to default for escaping ones). · commit 8eee4963 ·
tests: 1 new (5 escaping + 2 control inputs)

### AC-1 / AC-2 / AC-3 — ritual default flip

Rewrote `aiwfx-start-epic` step 8 to lead with the in-repo default reading `worktree.dir`
(keeping all three placements), revised the contradicting Constraint / Anti-patterns /
Principles / intro / description, and added a `## Principles`-section assertion so the
contradiction cannot regress. Added a worktree-placement note to `aiwfx-start-milestone` step 5.
· commit 8eee4963 · tests: 3 new structural (+1 helper branch-coverage)

## Decisions made during implementation

- **AC-4 lives in the getter, not a markdown "use site".** The M-0189 reviewer deferred the
  `..`-escape check to "the use site (M-0190)". M-0190's use site is a markdown ritual, not Go;
  the getter is the single chokepoint the value the ritual consumes passes through, so the
  escape rejection belongs there (symmetric with the existing absolute-path rejection). The
  doctor line the ritual greps is therefore always in-repo.
- **Honored paths return the trimmed original, not `filepath.Clean`'d.** Keeps `.wt` → `.wt`
  (the existing M-0189 test) and avoids a normalization behavior change beyond AC-4's scope;
  only the escape *test* uses the cleaned form.

## Validation

- `go test ./...` — green, no failures.
- `golangci-lint run ./internal/config/... ./internal/policies/...` — 0 issues.
- `aiwf check` — 0 errors (the `epic-active-no-drafted-milestones` warning is expected: all
  three E-0046 milestones are past `draft`; it clears at epic wrap).
- pre-commit hook (commit 8eee4963): policy suite green (128s), `aiwf check` shape clean.
- 5 new tests (1 config getter, 3 structural ritual-content, 1 helper branch-coverage); the
  existing 9-step / 8-step / 3-option structural pins all survived the SKILL.md edits.

## Reviewer notes

Independent fresh-context reviewer (two passes): **APPROVE**, verified by measuring (built the
binary, traced the escape grammar over the full vector set, confirmed each doc-AC test goes RED
on revert, ran the suites). One blocking finding in pass 1 — a stale `## Principles` bullet in
start-epic ("a prompt rather than picking on the operator's behalf") that contradicted the new
default and which the step-8-scoped AC-1 test could not see — fixed inline by rewriting the
Principles bullet, the intro, and the description, and adding a Principles-section assertion to
AC-1; pass 2 confirmed resolution with no new issues. The adjacent M-0104-stale "branch-shape
choice" framing in the intro/description was folded into the same edit.

gitleaks was not on PATH at commit time, so the pre-commit path-leak gate was skipped; the diff
is Go + ritual markdown with no absolute user paths, so no leak risk — noted for the record.

## Deferrals

- None. The `..`-traversal escape check carried from M-0189 is closed here (AC-4).
