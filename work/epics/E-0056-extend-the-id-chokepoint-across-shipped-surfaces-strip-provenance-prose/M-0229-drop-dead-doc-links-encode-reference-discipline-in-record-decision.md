---
id: M-0229
title: Drop dead doc-links; encode reference discipline in record-decision
status: in_progress
parent: E-0056
depends_on:
    - M-0227
tdd: advisory
acs:
    - id: AC-1
      title: aiwfx-record-decision encodes the self-contained reference rule
      status: met
      tdd_phase: done
    - id: AC-2
      title: No shipped-skill markdown link targets a repo-relative path
      status: met
      tdd_phase: done
    - id: AC-3
      title: Discoverability tests pin behavior, not the removed ADR ids
      status: met
      tdd_phase: done
---
## Goal

The rule for how a behavioral skill references a decision — so a reference never
becomes a dead link in a consumer's materialized `.claude/` — is encoded in the
`aiwfx-record-decision` skill, the ritual that authors decisions. The existing
shipped-skill doc-links that violate it are removed and reworded to self-contained
imperative instruction. No decision entity is created; the behavior lives in the
skill body.

## Approach

The shipped skills carry markdown doc-links that are dead twice over (`G-0315`):
they use `../` depths that do not resolve even in this repo, and they target a
`docs/` (or `internal/`) tree that a consumer's materialized `.claude/` does not
contain. Fixing the depth alone still leaves the consumer a dead link, so the
resolution is to drop the non-shipping references and state the behavior directly.

Scope is **all shipped skills**, not only the two ritual skills named in the
sketch: the same defect sits in three verb skills (`aiwf-render`,
`aiwf-authorize`, `aiwf-archive`), and `aiwf-archive`'s "cite `ADR-0004` by id"
discoverability AC is in direct tension with the broadened "cite no real entity
id" principle `M-0228` just shipped — so the whole class is closed here, per the
epic's "across shipped surfaces" framing.

- **Encode the reference behavior in the `aiwfx-record-decision` skill body.** A
  behavioral skill states its behavioral fact directly and self-contained; it
  does not embed a link to a decision record or design doc under `docs/` (or
  another non-shipping repo path). A decision's rationale lives in its own entry,
  authored via this skill — not in a link from a behavioral skill. This is skill
  guidance, **not** a decision entity — no ADR or `D-` entity is created for it.
- **Remove the dead links** across every shipped skill (`aiwfx-*` / `wf-*`
  rituals and the `aiwf-*` verb skills), rewording to self-contained imperative
  instruction that conveys the same behavioral fact, and convert the two
  placeholder `` `work/epics/E-NNNN-<slug>/M-NNNN-<slug>.md` `` links to plain
  code-spans. Reconcile the prior discoverability tests that required an ADR
  reference so they no longer mandate a now-removed link.

## Acceptance criteria — all three mechanizable

Unlike `M-0228` (one met-with-a-test AC plus a deliberate review backstop), every
AC here has a stable machine shape: a section-scoped structural assertion (AC-1),
a link-shape absence guard (AC-2), and the reconciled discoverability tests
(AC-3). No review-only backstop item is required.

### AC-1 — aiwfx-record-decision encodes the self-contained reference rule

The `aiwfx-record-decision` skill body carries a `## Referencing a decision`
section stating the rule: a behavioral skill states its behavioral fact directly
and self-contained; it does not embed a link to a decision record or design doc
under `docs/` (or another non-shipping repo path); a decision's rationale lives
in its own entry, authored via this skill. Mechanical evidence: a structural test
in `internal/policies/` that walks the skill's heading hierarchy to the named
section and asserts each rule marker (*self-contained*, *does not embed a link*,
*docs/*, *rationale*, *own entry*) is present *within that section* — a
section-scoped assertion, not a bare whole-file grep, mirroring
`m0228_skills_policy_broadened_principle.go`. The markers are absent before the
edit, so the test is red until the section lands. The test references the skill's
embedded path, satisfying the `skill-edit-structural-test-backstop`.

### AC-2 — No shipped-skill markdown link targets a repo-relative path

A CI policy test (`internal/policies/`, inert in a consumer) walks every shipped
skill `.md` under `internal/skills/embedded/**` and
`internal/skills/embedded-rituals/**` and asserts every markdown link destination
is an external `http(s)://` URL or a same-file `#anchor` — no repo-relative
destination survives. The predicate is universal, not a per-tree allowlist: since
no shipped skill has a single legitimate repo-relative link, any such link is
dead in a consumer regardless of which tree it targets (`docs/`, `internal/`,
`work/`, anything). The test is red now (13 repo-relative links: 11 escaping into
`docs/`/`internal/` plus 2 placeholder `` `work/epics/...` `` links) and green
after the reword; it stays meaningful indefinitely — any future embedded-skill
edit that adds a repo-relative link trips it.

### AC-3 — Discoverability tests pin behavior, not the removed ADR ids

The prior discoverability tests that required an ADR reference are reconciled so
they pin the behavioral fact rather than the removed id: `M-0104/AC-2` (which
required `ADR-0010` in `aiwfx-start-epic` `## Workflow`), `M-0190/AC-1` and
`M-0190/AC-3` (which required `ADR-0023` in the worktree guidance), and
`aiwf-archive`'s "cite `ADR-0004` by id" AC. Where a sibling marker already pins
the behavior (the worktree tests already assert *sandbox* / *devcontainer* /
*in-repo* / *worktree.dir* / *rebuild*), the id marker is dropped. Where the id
was the sole pin, a replacement assertion on the behavioral fact is added (the
sovereign-acts-before-branch-cut sequencing; the archive convention). Mechanical
evidence: the reconciled tests themselves — still red if the behavioral fact goes
missing, green with the ids gone.

## Work log

<!-- One entry per AC; append-only. Populated through implementation. -->

## Decisions made during implementation

- Scope widened from the sketch's two ritual skills to **all shipped skills**
  (adding the three verb-skill links) at start-milestone — same defect class, and
  it resolves the `aiwf-archive` `ADR-0004`-citation tension with `M-0228`. An
  operator scope call recorded here; no `D-` entity required.

## Validation

<!-- Pasted at wrap: test-suite, build, lint, coverage-gate, aiwf check. -->

## Deferrals

- (none)

## Reviewer notes

<!-- Trade-offs and deliberate omissions; filled at wrap. -->
