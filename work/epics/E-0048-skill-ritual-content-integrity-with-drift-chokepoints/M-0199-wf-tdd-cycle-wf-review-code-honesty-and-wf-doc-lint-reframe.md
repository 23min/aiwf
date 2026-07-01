---
id: M-0199
title: wf-tdd-cycle/wf-review-code honesty and wf-doc-lint reframe
status: in_progress
parent: E-0048
depends_on:
    - M-0196
tdd: advisory
acs:
    - id: AC-1
      title: wf-tdd-cycle records met after the coverage audit and vacuity check
      status: met
    - id: AC-2
      title: wf-tdd-cycle drops idempotent misuse and reframes force as sovereign
      status: open
    - id: AC-3
      title: branch-coverage audit framed as agent-performed in both skills
      status: open
    - id: AC-4
      title: wf-doc-lint separates repo-wide path-leak scan from four doc heuristics
      status: open
    - id: AC-5
      title: wf-doc-lint secret-scan advice is pre-push plus CI with current gitleaks
      status: open
---
## Goal

Three `wf-*` engineering rituals ship advice that is dishonest, self-contradictory,
or actively steers an implementing agent toward the wrong move. An AI assistant (or
human) reading them for authoritative guidance is misled:

- **`wf-tdd-cycle` narrates the "done" judgment before the evidence.** Its RECORD
  step promotes the AC to `met`, yet it is narrated *before* the branch-coverage
  audit and the vacuity check — the very evidence that substantiates "done." A
  judgment gate placed where the judge cannot yet see the evidence is vacuous, and
  invites "green tests, untested branch" closures.
- **`wf-tdd-cycle` + `wf-review-code` overstate branch coverage as mechanical.**
  Both call the branch-coverage audit a "hard rule," which reads as *tool-enforced
  at branch granularity*. It is an **agent-performed manual branch-walk**; a
  project's mechanical coverage gate is typically statement-level, so the manual
  walk is what actually supplies branch-level assurance.
- **`wf-tdd-cycle` steers the implementing agent toward `--force`.** Its RECORD
  step offers `--force --reason` as a routine way to record `met` ahead of `done`.
  But `--force` is a sovereign, human-only act, and recording `met` ahead of `done`
  bypasses the TDD audit that gives the phase ladder its meaning.
- **`wf-tdd-cycle` misuses "idempotent."** The RED step calls re-running the
  phase-seed "idempotent" while noting the FSM *refuses* `red → red`. A step the FSM
  refuses errors on re-run — the opposite of idempotent; it is redundant, and the
  honest instruction is to skip it.
- **`wf-doc-lint` ships a self-contradictory path-leak check.** It lists *five*
  checks while the Workflow, output template, and `description:` all say *four*;
  its anti-pattern ("block-on-zero is too strict") contradicts check #5's own "this
  deserves a real chokepoint"; and its docs-tree scope contradicts the repo-wide
  reach of path-leak scanning. It also recommends the scan as a *pre-commit* hook
  using the deprecated `gitleaks detect` subcommand — advice this repo has since
  decided against for itself (a secret is not exposed until push).

This milestone corrects all of it and pins each fix with a structural test under
`internal/policies/`. Because the edited files are ritual `SKILL.md`s under
`internal/skills/embedded-rituals/**`, those same tests satisfy the
`skill-edit-structural-test-backstop` (G-0220 / M-0196): every edited skill's path
is referenced by a test, so no fix ships to consumers without a mechanical backstop.

Sources: G-0309 (reorder), G-0297 (tdd-cycle / review-code honesty), G-0294
(doc-lint reframe). Parent epic E-0048.

Out of scope: the deeper `wf-vacuity` / over-gating question (owned by G-0295, the
over-gating complement to G-0309's under-gating); the standalone path-leak tool's
own rules (the consumer owns their `.gitleaks.toml`). These are content-and-honesty
corrections to shipped ritual prose, not new mechanical gates.

## Acceptance criteria

### AC-1 — wf-tdd-cycle records met after the coverage audit and vacuity check

The `wf-tdd-cycle` skill narrates the branch-coverage audit and the vacuity check
*before* the RECORD step that promotes the acceptance criterion to `met`. The "done"
judgment — the HITL/agent moment where someone can still act — sits after the
evidence, not before it (G-0309).

Test: a structural test locates the "Branch-coverage audit", "Vacuity check", and
"RECORD" section headings in the body and asserts the RECORD heading appears after
both the audit and the vacuity headings (positional, not a flat substring match).

### AC-2 — wf-tdd-cycle drops idempotent misuse and reframes force as sovereign

Two honesty corrections in `wf-tdd-cycle` (G-0297):

- The RED phase-seed no longer calls re-running the `--phase red` promote
  "idempotent"; it names it **redundant** — the FSM refuses `red → red`, so the step
  is *skipped* when the AC was already seeded at `red`.
- The RECORD step reframes the `--force met` escape hatch as a **human-only
  sovereign** act that **bypasses the TDD audit** — not a routine move the
  implementing agent (often non-human, which the kernel refuses) reaches for.

Test: a structural test asserts the RED phase-seed prose contains "redundant" and no
longer contains "idempotent"; and that the `--force` passage frames it as
human-only / sovereign and states it bypasses the audit.

### AC-3 — branch-coverage audit framed as agent-performed in both skills

Both `wf-tdd-cycle` and `wf-review-code` state that the branch-coverage audit is an
**agent-performed manual branch-walk**, and that where a project's mechanical
coverage gate is **statement-level**, the manual walk is what supplies the
branch-level assurance. This removes the "hard rule ⇒ tool-enforced at branch
granularity" false implication while keeping the audit a hard *discipline* rule
(G-0297).

Test: a single structural test reads both skills (referencing both paths) and, for
each, asserts the branch-coverage section carries the agent-performed / manual-walk
framing and the statement-vs-branch mechanical-gate distinction.

### AC-4 — wf-doc-lint separates repo-wide path-leak scan from four doc heuristics

The `wf-doc-lint` skill presents exactly **four** doc-heuristic checks under "What it
checks" (code-reference drift, removed-feature docs, orphan documents, documentation
TODOs), and moves the path-leak / secret scan into a distinct "Related: repo-wide …"
section that is explicitly **not** one of the four heuristics. This resolves the
count drift (four heuristics, four everywhere), the anti-pattern contradiction (the
"block-on-zero is too strict" caution scopes to the four advisory heuristics; the
deterministic standalone tool legitimately gates), and the scope mismatch (the four
heuristics are docs-scoped; the standalone scan is repo-wide) (G-0294 facets 3/4/5).

Test: a structural test asserts the "What it checks" section contains exactly four
numbered `###` sub-headings and no fifth; asserts a separate repo-wide secret /
path-leak section exists outside "What it checks"; and asserts the block-on-zero
anti-pattern text scopes itself to the doc heuristics rather than contradicting the
standalone gate.

### AC-5 — wf-doc-lint secret-scan advice is pre-push plus CI with current gitleaks

Within the reframed standalone-scan section, `wf-doc-lint` recommends wiring the
secret / path-leak scan as a **pre-push hook plus a CI job** (the push is the trust
boundary; pre-commit merely taxes latency without being the boundary), and uses the
current **`gitleaks git`** (history) / **`gitleaks dir`** (filesystem) subcommands.
The deprecated `gitleaks detect` invocation and the pre-commit-hook recommendation
are gone (G-0294 facets 1/2).

Test: a structural test asserts the section mentions a pre-push hook and a CI job,
mentions `gitleaks git` and/or `gitleaks dir`, and contains neither `gitleaks detect`
nor a recommendation to run the scan as a pre-commit hook.

## Work log

tdd: advisory — no per-AC phase timeline; this log records the final outcome per AC.

### AC-1 — wf-tdd-cycle records met after the coverage audit and vacuity check

### AC-2 — wf-tdd-cycle drops idempotent misuse and reframes force as sovereign

### AC-3 — branch-coverage audit framed as agent-performed in both skills

### AC-4 — wf-doc-lint separates repo-wide path-leak scan from four doc heuristics

### AC-5 — wf-doc-lint secret-scan advice is pre-push plus CI with current gitleaks

## Decisions made during implementation

## Validation

## Deferrals

## Reviewer notes
