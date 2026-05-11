# Epic wrap — E-0028

**Date:** 2026-05-11
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0028-start-epic-ritual
**Merge commit:** `81e8853`

## Milestones delivered

- **M-0094** — Add aiwf check finding `epic-active-no-drafted-milestones` (merged 62ec2c2)
- **M-0095** — Enforce human-only actor on `aiwf promote E-NN active` (merged 50e85fc)
- **M-0096** — Ship aiwfx-start-epic skill with worktree and branch preflight prompts (merged 36027b2)
- **M-0097** — Close M-0094/95/96 verification seams: M-0095 automation audit chokepoint and AC-5 drift comparator (merged 8281f8f)

## Summary

E-0028 shipped the start-epic ritual proper: a rituals-plugin skill (`aiwfx-start-epic`) orchestrating epic activation through a 10-step workflow — preflight reads, drafted-milestone check, `aiwf check` cleanliness, worktree-placement Q&A, branch-shape Q&A (placeholder pending G-0059), delegation prompt, sovereign promotion, optional `aiwf authorize`, and hand-off — backed by two new kernel chokepoints: `epic-active-no-drafted-milestones` (warns when an `active` epic has no drafted milestones) and a sovereign-act refusal on `aiwf promote E-NN active` for non-`human/` actors (mirroring the existing `--force --reason` override pattern). M-0097 added a fourth milestone post-hoc to close two verification seams from M-0094/95/96 surfaced during a confidence audit: a static CI/script chokepoint test for unforced epic-activation invocations, and a synthetic two-arm test for the M-0096/AC-5 drift comparator that previously relied on rare production drift to exercise its failure path. Closes G-0063 start-side per the epic's stated scope; the wrap-side concerns (scope-end-before-`done`, human-only on `done`, `aiwfx-wrap-epic` update) are deliberately deferred to G-0111 below.

## ADRs ratified

- none. The epic spec's *ADRs produced* section anticipated none; nothing surfaced during implementation that warranted ADR-shaped documentation. The skill design lives in its own SKILL.md (canonical at `internal/policies/testdata/aiwfx-start-epic/SKILL.md`, copied to rituals-repo at SHA `87fc790`); the two kernel rules are documented via `--help` text and `aiwf check` finding hints.

## Decisions captured

- **Reading A for M-0094** — `epic-active-no-drafted-milestones` fires whenever an `active` epic has zero `draft` milestones (the strict-literal reading). Recorded in M-0094's *Reviewer notes* with the rationale for rejecting reading B ("no forward motion") and reading C ("activation-moment only"). No D-NNNN entity opened — the choice is captured in the milestone spec.
- **Pre-wrap drift-check skip semantics** — M-0096/AC-5 skips on three "absent" states (manifest missing, plugin not installed, skill not materialised in cache) and fails only on actual drift. Diverges slightly from M-0090's precedent (which fails on "not materialised") for milestone-scope clarity. Recorded in M-0096's *Decisions made during implementation*.
- **Manual mutation review** — M-0097's operator-task self-review used manual branch-walking against named tests in place of `mutate-hunt`, because `gremlins --diff <ref>` silently SKIPs new-file mutants in this worktree configuration. Documented in M-0097's *Validation*; filed as G-0110.

## Follow-ups carried forward

- **G-0110** — `gremlins --diff <ref>` filter excludes new files entirely; future milestone self-reviews need either a workaround (full-package run + grep-filter) or a fix in `mutate-hunt.yml` / gremlins upstream.
- **G-0111** *(to be filed at wrap)* — Wrap-side concerns deferred from E-0028's scope: `aiwf promote E-NN done` auto-end-scope behavior change, human-only enforcement on `done`, `aiwfx-wrap-epic` update for the new wrap-timing, ADR for the scope-end-before-`done` decision. The skill ships with the current wrap-side behavior assumed; a follow-up epic will adjust both verb and ritual together.
- **`/aiwfx-start-epic` end-to-end verification** — known post-merge step. Requires `/reload-plugins` after the rituals-repo SHA `87fc790` is in the marketplace cache. Not a gap; natural operator activity post-epic-wrap.

## Doc findings

Doc-lint sweeps were run per-milestone (M-0094, M-0096, M-0097) scoped to each change-set; all came back clean. No epic-level findings.

## Handoff

What is ready for the next epic:

- The start-epic ritual is invocable as `/aiwfx-start-epic E-NN` after `/reload-plugins` (rituals-repo `87fc790` carries the skill; marketplace cache picks it up on plugin reload).
- The kernel chokepoints (M-0094, M-0095, M-0097) are mechanically enforced in CI — no honor-system aspects.
- The drift-check pattern (fixture in policies/testdata + drift test in policies/ + wrap-time copy to rituals-repo) is now precedented twice (M-0090, M-0096); future cross-repo SKILL.md milestones follow it without re-deriving.

What is deliberately left open:

- The wrap-side ritual's behavior on epic completion (deferred to G-0111 per epic scope).
- `G-0059` (branch-model gap) — the skill's branch prompt is a placeholder; resolution tightens the prompt's default.
- The sovereign-act rule's generalization to other kinds (`contract → active`, `ADR → accepted`) — open question, out of E-0028's scope.

## Validation snapshot

- Final `aiwf check` on the epic branch: 0 errors, 6 advisory warnings (4× `terminal-entity-not-archived` for M-0094/95/96/97 awaiting sweep, 1× `archive-sweep-pending`, 1× `provenance-untrailered-scope-undefined` for no-upstream worktree).
- Final `go test -race -count=1 ./...`: 25 packages green, 0 FAIL lines.
- Final `golangci-lint run ./...` per-package: 0 issues.
- Rituals-repo coupling: `87fc790` on `ai-workflow-rituals` `main` (local-only; push is a separate gate).
