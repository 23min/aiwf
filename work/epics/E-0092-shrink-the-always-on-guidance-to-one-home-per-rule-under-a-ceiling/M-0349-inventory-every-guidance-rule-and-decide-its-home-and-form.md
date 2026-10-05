---
id: M-0349
title: Inventory every guidance rule and decide its home and form
status: in_progress
parent: E-0092
depends_on:
    - M-0333
tdd: advisory
acs:
    - id: AC-1
      title: The inventory lists every rule in the guidance set once, with its current homes
      status: met
    - id: AC-2
      title: Every rule carries one disposition from the closed set
      status: met
    - id: AC-3
      title: Every conflict is decided by the maintainer and recorded
      status: open
    - id: AC-4
      title: The on-demand documents and the router are specified
      status: open
    - id: AC-5
      title: Each host's handwritten primed ceiling is set from the primed rules
      status: open
---
## Goal

List every guidance rule in this repository once, with where it lives today, and decide its home and form: primed or on demand, kept, merged, tightened, reduced to a pointer, or deleted as a copy. Settle every conflict with the maintainer.

## Closes

- (none)

## Context

M-0333's fence is in place. Nothing here changes guidance text: this milestone produces the table the later milestones carry out, and M-0334 records its baseline from that table before any decision is carried out. The inventory covers two homes: this repository's development guidance — the set M-0333 defines, the root `CLAUDE.md` and `AGENTS.md`, `.guidance/project.md` and what it routes to — and the shipped aiwf fragment (`internal/skills/embedded-guidance/aiwf-guidance.md`). The selected packs under `.guidance/packs/` are canonical homes for the rules they carry and are compared against, not inventoried.

## Acceptance criteria

### AC-1 — The inventory lists every rule in the guidance set once, with its current homes

The table in `## Inventory` below has one row per rule, across both homes. A row carries a label the removal commits cite, and names the rule, every place it is stated today, its words, its audience — operating aiwf in any repository, or developing aiwf — and whether it is a judgment rule or a check-backed one. **Pass criterion**: the inventory test derives every `##` and `###` section of the handwritten guidance set and every top-level rule of the shipped fragment, as they stand at the commit that freezes the guidance, and asserts each is cited as a home by at least one row, read through the loader; it passes.

### AC-2 — Every rule carries one disposition from the closed set

Each row carries a placement — primed or on demand — by E-0092's test, and one disposition: `keep`, `merge into <rule>`, `tighten`, `move on demand to <document>`, `move to the shipped fragment`, `move to the shipped skill <path>`, `move to repository guidance <document>`, `pointer to <policy or finding code>`, `delete as copy of <home>`, or `conflict`. A row for a rule in the shipped fragment takes only `keep`, `move to repository guidance <document>` or `conflict`; a tightening it could take is recorded in the row's notes as a suggestion for a later epic. A rule's final home matches its audience and placement: a development rule ends in `CLAUDE.md` when primed and in an on-demand document when not; an operating rule ends in the shipped fragment when primed and in the shipped skill for its task when not. **Pass criterion**: the inventory test rejects a row with no placement, no audience, no disposition, a disposition outside the set or outside what its home allows, a final home that disagrees with its audience or placement, or a shipped skill that does not exist; on the table it reports none.

### AC-3 — Every conflict is decided by the maintainer and recorded

A rule that contradicts another, in the guidance set or against the fragment, a pack or a skill, is presented to the maintainer one decision at a time. The decision and the rule it leaves standing are recorded in `## Decisions made during implementation`, and the row's disposition is replaced by the outcome. A decision with reach beyond this repository is recorded as a decision entity. **Pass criterion**: no row ends the milestone marked `conflict`.

### AC-4 — The on-demand documents and the router are specified

The on-demand documents are named, with their paths and the tasks each serves, and the `.guidance/project.md` router entries that reach them are drafted here. Every `move on demand to <document>` row names one of them, and every `move to repository guidance <document>` row names one of them or the root `CLAUDE.md`. **Pass criterion**: the check from AC-2 rejects a row whose destination is not in the named set.

### AC-5 — Each host's handwritten primed ceiling is set from the primed rules

Sum the words of the rules placed primed, after their recorded dispositions, for each host, and set that as the target ceiling M-0337 reaches. **Pass criterion**: the figures and the command that derives them from the table are recorded in Validation, and E-0092's open question on the ceiling is resolved by an edit that cites them.

## Constraints

- No guidance text changes in this milestone.
- Set `guidance.enabled: false` in `aiwf.yaml` at this milestone's start, so no `aiwf update` or `aiwf worktree add` refreshes the installed packs while the inventory is taken; it stays off until M-0338 completes.
- A rule in the shipped fragment keeps its wording in this epic; a clause moved into one of its rules is added, never in place of its text.
- The primed test is E-0092's: nearly every task, or harm before the agent would think to look.
- One home per rule, chosen by audience and placement: operating aiwf in any repository belongs to the shipped fragment when primed and to the shipped skill for its task when on demand; developing aiwf belongs to `CLAUDE.md` when primed and to `.guidance/project/` when on demand.
- Conflicts are the maintainer's decisions, presented one at a time, with the evidence from both homes.

## Design notes

- A near-duplicate is one row whose disposition is `merge into` the surviving statement; the survivor's row records the merge.
- A `tighten` row states what the tightened rule must still carry — the obligation and its one-line reason — so review of M-0335 can check the rewrite against it.
- A check-backed rule is `pointer to` only if its diagnostic already states the remedy or M-0337's audit will make it; otherwise it stays `keep` or `tighten`.
- A row's label is what a removal commit names in its disposition block, so the inventory test can match the two. Labels are short and not shaped like entity ids.
- **The inventory test.** One test file under `internal/policies/` parses this table once and carries every check against it: AC-1's coverage, AC-2's and AC-4's table checks, M-0336 AC-5 and AC-6's reconciliation of removal commits, and M-0337 AC-1's coverage of its audit table. Each rule it enforces has one fixture. The coverage checks read the guidance at the commit each table was taken from, so later milestones' edits leave them green. It runs with the policy suite, and E-0092's wrap deletes it once M-0338 records its last run.

## Surfaces touched

- This milestone's body: `## Inventory`, `## Decisions made during implementation`, `## Validation`.
- E-0092's open questions.
- The inventory test under `internal/policies/`.

## Out of scope

- Any change to guidance text (M-0336, M-0335, M-0337).
- Rewording the shipped fragment; tightening suggestions are recorded for a later epic.
- The content of external language packs.

## Dependencies

- M-0333 — the guidance set and the measure

## Coverage notes

- (none)

## References

- ADR-0054 — external language-content ownership
- D-0102 — the evidence rule over the development-guidance set
- G-0676 — how the guidance grew

## On-demand documents

Each document lives under `.guidance/project/` and serves the tasks named beside it. The router entries below replace `.guidance/project.md`'s current text in M-0335.

```markdown
# Project guidance

`CLAUDE.md` holds the rules for every task in this repository, and they take
precedence over the packs. Before the first edit a task governs, read the
document that matches it:

- [Testing](project/testing.md) — writing or changing Go tests, the coverage
  gates, running tests on macOS.
- [Verb design](project/verb-design.md) — adding or changing a verb, a check
  rule or the loader.
- [Go conventions](project/go-conventions.md) — Go code in this repository;
  where it differs from the Go pack, logging included, it overrides the pack.
- [Environment](project/environment.md) — setup, the devcontainer, subagent
  worktrees, diagnosing with a binary built from a worktree.
- [Shipped content](project/shipped-content.md) — editing embedded skills,
  rituals, templates or the guidance fragment.
- [Stress harness](project/stress-harness.md) — stress scenarios and their
  lanes.
- [Writing docs](project/writing-docs.md) — documents under `docs/` and ADRs.
- [Release](project/release.md) — cutting a release.
- [Enforcement](project/enforcement.md) — where a rule is enforced, from
  pre-commit to the release tag.
```

## Inventory

Words are whitespace-separated words of the rule's text at the commit that freezes the guidance, as the ceiling policy counts them. `F` in a home stands for `internal/skills/embedded-guidance/aiwf-guidance.md`, the shipped fragment, and the text after `›` is the rule's bold lead-in. A `CLAUDE.md` home with no section is its preamble.

| Label | Rule | Current homes | Words | Audience | Kind | Placement | Disposition | Notes |
|---|---|---|---|---|---|---|---|---|
| repo-orientation | What this repository is; design-decisions.md holds the commitments and scope limits; gaps are entities | `CLAUDE.md` | 62 | development | judgment | primed | keep |  |
| kiss | Prefer the boring solution | `CLAUDE.md › Engineering principles` | 29 | development | judgment | on demand | delete as copy of `.guidance/packs/code-health/guide.md` | The pack's Design discipline states it. |
| yagni | No speculative interfaces or knobs | `CLAUDE.md › Engineering principles` | 29 | development | judgment | on demand | delete as copy of `.guidance/packs/code-health/guide.md` |  |
| no-half-finished | A feature lands tested; no stubs or TODOs in shipped code | `CLAUDE.md › Engineering principles` | 19 | development | judgment | on demand | delete as copy of `.guidance/packs/code-health/guide.md` |  |
| errors-are-findings | `aiwf check` loads inconsistent state and reports it, never refusing to start | `CLAUDE.md › Engineering principles` | 27 | development | judgment | on demand | move on demand to `.guidance/project/verb-design.md` | Applies when writing loader or check code. |
| llm-independent-correctness | A guarantee must not depend on an LLM invoking a skill; hooks and `aiwf check` are authoritative | `CLAUDE.md › Engineering principles` | 36 | development | judgment | primed | keep | Shapes every feature's design. |
| ai-discoverable | Every kernel capability is reachable through help, skills or design docs | `CLAUDE.md › Engineering principles` | 60 | development | check-backed | on demand | move on demand to `.guidance/project/verb-design.md` | The discoverability policies fail on an undocumented finding code or config field. |
| completion-friendly | Every verb, flag and closed-set value tab-completes | `CLAUDE.md › Engineering principles` | 39 | development | check-backed | on demand | move on demand to `.guidance/project/verb-design.md` | The completion drift test fails on a miss. It lives at `internal/cli/integration/completion_drift_test.go`; the moved text names that path. |
| provenance-principle | Provenance is principal × agent × scope | `CLAUDE.md › Engineering principles` | 62 | development | judgment | primed | merge into commit-provenance | The same commitment, stated twice. |
| no-fake-ids | No malformed or fabricated id-shaped tokens in committed prose | `CLAUDE.md › Engineering principles`; `F › Never write a fake or narrow id-shaped token` | 66 | operating | check-backed | primed | delete as copy of `internal/skills/embedded-guidance/aiwf-guidance.md` | The fragment states the entity-file rule and points at the `aiwf-check` skill's per-file table, which stands over this passage's broader wording (a decision below); the planning shorthand and pseudo-sequence labels are stated in the planning ritual skills. `body-prose-id` enforces it in entity bodies. |
| one-decision | Present three or more decisions one at a time | `CLAUDE.md › Working with the user`; `F › Decide one thing at a time` | 31 | operating | judgment | primed | delete as copy of `internal/skills/embedded-guidance/aiwf-guidance.md` | The fragment states the rule; the clauses it lacks are the one-decision-extras row. |
| one-decision-extras | "Q&A" or "interview me" triggers the rule; the options include "something else"; wait for the pick, confirm it in one line, move on | `CLAUDE.md › Working with the user` | 19 | operating | judgment | primed | move to the shipped fragment | Added to the fragment's "Decide one thing at a time" rule, which keeps its wording. |
| never-pause | Never suggest the human pause | `CLAUDE.md › Working with the user`; `F › Never suggest the human pause` | 47 | operating | judgment | primed | delete as copy of `internal/skills/embedded-guidance/aiwf-guidance.md` |  |
| gate-per-mutation | Every mutation is its own gate; the declared-sequence exception; outward actions never batch | `CLAUDE.md › Working with the user`; `F › Each mutating action is its own approval gate` | 290 | operating | judgment | primed | delete as copy of `internal/skills/embedded-guidance/aiwf-guidance.md` | The fragment states the gate, the declared-sequence exception and live TDD promotes; each wrap ritual states its own order and reason, which stand over this passage's "promote lands last" (a decision below); the cited gaps hold their own reasoning. |
| declared-gate-abort | Any deviation — a conflict, a finding, unexpected dirty state, an action not on the list — aborts a declared-sequence gate and re-gates | `CLAUDE.md › Working with the user` | 16 | operating | judgment | primed | move to the shipped fragment | The wrap rituals and `wf-patch` say it for their own gates; added to the fragment's gate rule, it covers every declared-sequence gate. |
| remote-delete-gate | A remote-branch delete is outward and never batches | `CLAUDE.md › Working with the user` | 2 | operating | judgment | primed | move to the shipped fragment | Added to the outward actions the fragment's gate rule lists; the wrap rituals already gate it. |
| finish-in-context | Fix a closely related issue inline | `CLAUDE.md › Working with the user`; `F › Fix closely-related issues in place` | 45 | operating | judgment | primed | delete as copy of `internal/skills/embedded-guidance/aiwf-guidance.md` | The fragment's cheap-fix rule states it; the surface-what-you-noticed row carries the clause it lacks. |
| surface-what-you-noticed | Surface what you noticed plainly; don't silently defer | `CLAUDE.md › Working with the user` | 8 | operating | judgment | primed | move to the shipped fragment | Added to the fragment's cheap-fix rule. |
| adr-no-schedule | An ADR records the choice, not when to act on it | `CLAUDE.md › Authoring an ADR` | 67 | operating | judgment | on demand | delete as copy of `internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/adr.md` | The template's "Record what was chosen, never when to act on it" states it; the `aiwf-promote` skill carries `--force --reason` for an exceptional transition. |
| adr-no-status-pins | Only the FSM and `aiwf promote` constrain ADR status; no per-ADR test pins | `CLAUDE.md › Authoring an ADR` | 27 | development | judgment | on demand | move on demand to `.guidance/project/writing-docs.md` |  |
| state-conclusion | State the conclusion, not the drafting history, in docs, bodies and comments | `CLAUDE.md › Writing docs, entity bodies, and code comments — state the conclusion, not the drafting history`; `F › State the conclusion, not the drafting history` | 254 | operating | judgment | primed | delete as copy of `internal/skills/embedded-guidance/aiwf-guidance.md` | The fragment states the rule; which parts are gated is the enforcement-map row. |
| doc-tiers | Every docs subtree carries an authority tier | `CLAUDE.md › Documentation hierarchy` | 223 | development | judgment | on demand | move on demand to `.guidance/project/writing-docs.md` |  |
| commitments-intro | Changes preserve the load-bearing commitments; one that does not is a kernel-level decision | `CLAUDE.md › What aiwf commits to` | 29 | development | judgment | primed | keep |  |
| commit-six-kinds | Six hardcoded entity kinds with closed status sets | `CLAUDE.md › What aiwf commits to`; `CLAUDE.md › Go conventions › Type design` | 31 | development | judgment | primed | tighten | Must carry: six kinds, closed status sets, one transition function each, hardcoded rather than YAML until a consumer needs custom vocabulary. Absorbs the kinds-hardcoded row. |
| commit-stable-ids | Stable canonical-width ids that survive rename, cancel and collision | `CLAUDE.md › What aiwf commits to` | 109 | development | check-backed | primed | tighten | Must carry: the id is the primary key and the slug display, so rename preserves the id; canonical four-digit width with permanent narrow-width tolerance, since no verb widens an id in place; a narrow id in the active tree is an error; removal is a terminal status; collisions resolve by `aiwf reallocate`. The tolerance's reasoning is in the `aiwf-check` skill, not ADR-0008. |
| commit-prepush-check | `aiwf check` runs as the pre-push hook | `CLAUDE.md › What aiwf commits to` | 18 | development | judgment | primed | keep |  |
| commit-history-from-git | `aiwf history` reads `git log` trailers; no event log | `CLAUDE.md › What aiwf commits to` | 20 | development | judgment | primed | keep |  |
| commit-selected-hosts | Host artifacts are selected per host and refreshed explicitly | `CLAUDE.md › What aiwf commits to` | 73 | development | judgment | primed | tighten | Must carry: host selection by `aiwf init`/`update`, preserved user content, host-independent git hooks; the paths stay in the README's host setup guide. |
| commit-location-of-truth | Binary external; planning state in the repo; adapters materialized and gitignored | `CLAUDE.md › What aiwf commits to` | 24 | development | judgment | primed | keep |  |
| commit-one-commit-per-verb | One mutation, exactly one commit; a NoOp, none | `CLAUDE.md › What aiwf commits to` | 66 | development | judgment | primed | tighten | Must carry: one mutation makes exactly one commit and a NoOp none; a failed mutation aborts before the commit; a request already satisfied is a NoOp at exit 0; `metadata.commit_sha` is present only on a mutation. ADR-0036 keeps the same-status transition case. |
| commit-acs-tdd | ACs are milestone sub-elements; TDD is opt-in per milestone | `CLAUDE.md › What aiwf commits to` | 38 | development | check-backed | primed | keep |  |
| commit-provenance | Principal × agent × scope provenance | `CLAUDE.md › What aiwf commits to`; `CLAUDE.md › Engineering principles` | 58 | development | judgment | primed | tighten | Must carry, once for both rows: the human principal, the possibly non-human actor, authorize scopes, `--force` human-only, identity from git config; the model doc keeps the rest. |
| commit-archive | Terminal entities move to per-parent archive directories by `aiwf archive` | `CLAUDE.md › What aiwf commits to` | 76 | development | check-backed | primed | tighten | Must carry: terminal entities move to a per-parent `archive/` directory; promote and cancel flip status only; `aiwf archive` sweeps in one commit; the loader resolves archived ids; no reversal. ADR-0004 keeps the rest; `archive-sweep-pending` reports drift. |
| not-in-scope | What aiwf deliberately excludes | `CLAUDE.md › What is *not* in scope` | 74 | development | judgment | primed | keep | design-decisions.md's list differs, so this is not a copy. |
| validate-commands | The test, lint and build commands, and what `make check-fast` and `make ci` bundle | `CLAUDE.md › How to validate changes` | 66 | development | judgment | primed | tighten | Must carry: the test, lint and build commands and what `make check-fast` and `make ci` bundle. Its claim that CI runs the full gate on every push is untrue while G-0715 is open; the tightened text says which pushes CI runs on. |
| profile-gates | The profile-driven gates run off the `test-cov` profile and diff the working tree | `CLAUDE.md › How to validate changes` | 176 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` |  |
| validation-cadence | `make check-fast` in the inner loop, `make ci` at integration boundaries, skip redundant runs, never `--no-verify` a push | `CLAUDE.md › How to validate changes` | 165 | development | judgment | primed | tighten | Must carry: `make check-fast` in the inner loop, a milestone wrap into its epic included; `make ci` at the integration boundaries; skip a run when no build input changed, planning commits included; the wrap merge's `--no-verify`; never `--no-verify` a push, which skips the gitleaks scan. The claims that CI runs on every push and on an epic-branch push are untrue while G-0715 is open; the tightened text says which pushes CI runs on. |
| stress-purpose | `cmd/stresstest` is a dev-only correctness harness, run on demand | `CLAUDE.md › Stress-test harness` | 65 | development | judgment | on demand | move on demand to `.guidance/project/stress-harness.md` |  |
| stress-lanes | Hermetic scenarios run untagged on every push; real-subprocess ones carry the `stress` tag | `CLAUDE.md › Stress-test harness` | 471 | development | judgment | on demand | move on demand to `.guidance/project/stress-harness.md` |  |
| stress-lane-census | A `stress`-tagged test must start a process, wait on a clock or run a goroutine | `CLAUDE.md › Stress-test harness` | 101 | development | check-backed | on demand | move on demand to `.guidance/project/stress-harness.md` | `PolicyStressLaneCensus`. |
| stress-commands | The harness commands and `run` flags | `CLAUDE.md › Stress-test harness` | 152 | development | judgment | on demand | move on demand to `.guidance/project/stress-harness.md` |  |
| operator-setup | Run `aiwf init` or `aiwf update` after cloning, then `aiwf doctor` | `CLAUDE.md › Operator setup` | 37 | development | judgment | on demand | move on demand to `.guidance/project/environment.md` |  |
| settings-consent | Claude settings writes need per-invocation consent | `CLAUDE.md › Operator setup` | 20 | development | judgment | on demand | move on demand to `.guidance/project/environment.md` |  |
| devcontainer | What the devcontainer provides and where aiwf's additions live | `CLAUDE.md › Operator setup › Devcontainer` | 93 | development | judgment | on demand | move on demand to `.guidance/project/environment.md` |  |
| trunk-based | Maintainers commit directly to trunk without PRs; contributors use PRs; Conventional Commits on both paths | `CLAUDE.md › Working in this repo` | 54 | development | judgment | primed | tighten | Must carry: planning-state verbs, release commits and doc edits may commit directly to `main`; every code change goes on a branch — `wf-patch` for a small fix, a ritual branch for the rest — in a worktree, merged into `main` locally without a PR; outside contributors use PRs; Conventional Commits on both paths; `aiwf check` runs shape-only at pre-commit and in full at pre-push. The smaller-change line is a copy of the code-health pack. |
| derived-artifacts | Regenerate derived artifacts, never hand-edit them; the embedded ritual snapshot is hand-edited | `CLAUDE.md › Working in this repo` | 90 | development | judgment | primed | tighten | Must carry: `ROADMAP.md` by `aiwf render roadmap`, `STATUS.md` by the post-commit hook, `site/` by `aiwf render --format=html`, host artifacts by `aiwf update`, and the ritual snapshot as the hand-edited source. |
| ac-evidence | An AC claims a standing property or an observation; the discriminator is re-derivation; never a proxy | `CLAUDE.md › AC promotion requires mechanical evidence`; `F › Promote an AC to met only with mechanical evidence`; `internal/skills/embedded/aiwf-promote/SKILL.md` | 356 | operating | judgment | primed | delete as copy of `internal/skills/embedded-guidance/aiwf-guidance.md` | The fragment states the rule; the `aiwf-promote` skill carries both kinds of claim, the discriminator and the proxy test. This repository's evidence forms are the ac-evidence-repo row. |
| ac-evidence-repo | In this repository, a doc-shaped AC's evidence is a structural assertion scoped to a named section, and over a shipped surface or development guidance a relationship check | `CLAUDE.md › AC promotion requires mechanical evidence` | 51 | development | check-backed | on demand | move on demand to `.guidance/project/testing.md` | `shipped-prose-assertion` and `guidance-readers` hold the two exceptions. |
| ac-evidence-tdd-none | `tdd: none` never waives the test obligation | `CLAUDE.md › AC promotion requires mechanical evidence` | 21 | operating | judgment | on demand | move to the shipped skill `internal/skills/embedded/aiwf-promote/SKILL.md` | No shipped surface states it. |
| ac-evidence-binds-at-promote | The rule binds at the promote command and is held at review, not by the verb | `CLAUDE.md › AC promotion requires mechanical evidence` | 23 | operating | judgment | on demand | move to the shipped skill `internal/skills/embedded/aiwf-promote/SKILL.md` | No shipped surface states it. |
| worktree-default | Branch work goes in an in-repo worktree made with `aiwf worktree add` | `CLAUDE.md › Default to a worktree for any branch work in this repo` | 80 | development | judgment | primed | tighten | Must carry: the in-repo default under `.claude/worktrees/<branch>/` (ADR-0023), `aiwf worktree add` as the verb, that it covers the session's own work, and that only an explicit instruction keeps work in the main checkout. |
| subagent-isolation | The parent creates a subagent's worktree; never the `isolation` argument | `CLAUDE.md › Subagent worktree isolation` | 158 | development | check-backed | on demand | move on demand to `.guidance/project/environment.md` | `.claude/hooks/validate-agent-isolation.sh` denies the argument. |
| binary-discipline | Diagnose a worktree with a binary built from it | `CLAUDE.md › Worktree binary discipline` | 98 | development | judgment | on demand | move on demand to `.guidance/project/environment.md` |  |
| ritual-authoring | Rituals are authored in the embedded snapshot | `CLAUDE.md › Ritual content authoring` | 85 | development | judgment | on demand | move on demand to `.guidance/project/shipped-content.md` |  |
| ritual-provenance | An embedded `SKILL.md` edit rides a commit naming its entity | `CLAUDE.md › Ritual content authoring` | 253 | development | check-backed | on demand | move on demand to `.guidance/project/shipped-content.md` | `skill-edit-provenance-backstop` and the `commit-msg` hook. |
| audience-split | Operating rules ship in the fragment; development rules stay here; hybrids split | `CLAUDE.md › Consumer-operating guidance vs repo-development guidance` | 207 | development | judgment | on demand | move on demand to `.guidance/project/shipped-content.md` |  |
| collision-stale-branch | Before reallocating on a trunk collision, rule out a rename on trunk the branch has not merged | `CLAUDE.md › Id-collision resolution at merge time` | 168 | operating | judgment | on demand | move to the shipped skill `internal/skills/embedded/aiwf-reallocate/SKILL.md` | No shipped surface states it; the `aiwf-check` skill's `ids-unique/trunk-collision` row describes the finding but not this check. |
| id-allocation | Allocate ids on the working branch and push promptly; resolve a collision with `aiwf reallocate`, not `git mv` | `CLAUDE.md › Id-collision resolution at merge time`; `F › Allocate ids on your working branch and push promptly` | 178 | operating | check-backed | primed | delete as copy of `internal/skills/embedded-guidance/aiwf-guidance.md` | The fragment states the rule; the `aiwf-add` skill carries the multi-clone fetch, the unpushed-peer and unmerged-branch caveats and the all-refs scan; `id-rename-untrailered` reports a rename without the verb. `CLAUDE.md`'s claim that the verb rewrites every cross-reference is untrue: it rewrites frontmatter references and reports body-prose ones, as the `aiwf-reallocate` skill says. |
| go-intro | Repo-wide principles apply on top of the Go conventions | `CLAUDE.md › Go conventions` | 14 | development | judgment | on demand | delete as copy of `.guidance/packs/code-health/guide.md` | The pack states KISS, YAGNI and no half-finished work; errors-as-findings is its own row. |
| gofumpt | `gofumpt` formats via `golangci-lint` | `CLAUDE.md › Go conventions › Formatting and linting` | 11 | development | judgment | on demand | delete as copy of `.guidance/packs/go/cobra/guide.md` |  |
| golangci-lint | `golangci-lint` is the only linter; no bare `//nolint` | `CLAUDE.md › Go conventions › Formatting and linting` | 31 | development | check-backed | on demand | move on demand to `.guidance/project/go-conventions.md` | The pack has the linter and the `//nolint` rule; the enabled set is `.golangci.yml`'s, and "the only linter" is aiwf's. |
| no-blank-alias | Never `var _ = x` to silence `unused` | `CLAUDE.md › Go conventions › Formatting and linting` | 55 | development | judgment | on demand | move on demand to `.guidance/project/go-conventions.md` |  |
| stdlib-testing | stdlib `testing` and `go-cmp`, no testify | `CLAUDE.md › Go conventions › Testing` | 10 | development | judgment | on demand | delete as copy of `.guidance/packs/go/cobra/guide.md` |  |
| table-driven | Table-driven when two or more cases share a function; single-case tests stay flat | `CLAUDE.md › Go conventions › Testing` | 15 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` | The pack lacks the single-case rule. |
| golden-files | Golden files under `testdata/`, synthetic content only | `CLAUDE.md › Go conventions › Testing` | 9 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` | The pack lacks the synthetic-content rule. |
| race-detector | The race detector on every CI run | `CLAUDE.md › Go conventions › Testing` | 11 | development | judgment | on demand | delete as copy of `.guidance/packs/go/cobra/guide.md` |  |
| tests-in-devcontainer | Tests run unwrapped in the devcontainer | `CLAUDE.md › Go conventions › Testing › Running tests in the devcontainer (primary)` | 35 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` |  |
| tests-on-macos | On macOS, tests route through the signing wrapper | `CLAUDE.md › Go conventions › Testing › Running tests on macOS host (fallback)` | 128 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` |  |
| test-the-seam | Cover the helper and its integration seam | `CLAUDE.md › Go conventions › Test design rules` | 49 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` |  |
| contract-tests | Pin the question asked of an upstream-cached system | `CLAUDE.md › Go conventions › Test design rules` | 32 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` |  |
| spec-sourced-inputs | Cover an upstream-defined input space from its spec | `CLAUDE.md › Go conventions › Test design rules` | 26 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` |  |
| structural-assertions | Substring assertions are not structural; over shipped prose and development guidance, only relationship checks | `CLAUDE.md › Go conventions › Test design rules` | 225 | development | check-backed | on demand | move on demand to `.guidance/project/testing.md` | `shipped-prose-assertion` and `guidance-readers`. |
| render-verified | Rendered output is human-verified before an iteration closes | `CLAUDE.md › Go conventions › Test design rules` | 48 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` |  |
| every-branch | Each reachable branch has a test or a `//coverage:ignore` | `CLAUDE.md › Go conventions › Test design rules` | 36 | development | check-backed | on demand | move on demand to `.guidance/project/testing.md` | The diff-scoped coverage gate. |
| root-cause | Root-cause a test failure; don't paper over it | `CLAUDE.md › Go conventions › Test design rules` | 51 | development | judgment | on demand | move on demand to `.guidance/project/testing.md` |  |
| loader-paths | Policy tests resolve entity files through the loader | `CLAUDE.md › Go conventions › Test design rules` | 38 | development | check-backed | on demand | move on demand to `.guidance/project/testing.md` | `PolicyNoHardcodedEntityPaths`. |
| test-discipline | Parallel by default: `setup_test.go`, serial skip-list, `sync.Once` fixtures, the `-parallel 8` cap, hardened git env, presence chokepoint | `CLAUDE.md › Go conventions › Test discipline` | 150 | development | check-backed | on demand | move on demand to `.guidance/project/testing.md` | Policies hold the `setup_test.go` presence, the `-parallel 8` cap and the hardened git environment; the serial skip-list and `sync.Once` are held at review. Absorbs the t-parallel row. |
| t-parallel | `t.Parallel()` first line | `CLAUDE.md › Go conventions › Test discipline` | 13 | development | judgment | on demand | merge into test-discipline | The pack has the first-line rule; nested table subtests are aiwf's. |
| write-executable | Write executable stand-ins with `testsupport.WriteExecutable` | `CLAUDE.md › Go conventions › Test discipline` | 104 | development | check-backed | on demand | move on demand to `.guidance/project/testing.md` | `test-executable-write`. |
| coverage | The coverage target, the diff-scoped gate, the firing-fixture meta-gate, and fuzz, property and mutation testing | `CLAUDE.md › Go conventions › Coverage` | 235 | development | check-backed | on demand | move on demand to `.guidance/project/testing.md` |  |
| error-wrapping | Wrap across boundaries; compare with `errors.Is`/`As` | `CLAUDE.md › Go conventions › Error handling` | 18 | development | judgment | on demand | delete as copy of `.guidance/packs/go/cobra/guide.md` |  |
| sentinel-errors | Sentinel errors for stable conditions, typed errors carrying data | `CLAUDE.md › Go conventions › Error handling` | 13 | development | judgment | on demand | delete as copy of `.guidance/packs/go/cobra/guide.md` |  |
| no-panic-in-library | Library code never panics or exits | `CLAUDE.md › Go conventions › Error handling` | 11 | development | judgment | on demand | delete as copy of `.guidance/packs/go/cobra/guide.md` |  |
| context-first | `context.Context` first on IO-touching functions | `CLAUDE.md › Go conventions › Concurrency` | 22 | development | judgment | on demand | move on demand to `.guidance/project/go-conventions.md` | The pack has the first-argument and cancellation-only rule; the `WithValue` exception is aiwf's. |
| no-mutex-across-io | Never hold a mutex across an IO call | `CLAUDE.md › Go conventions › Concurrency` | 9 | development | judgment | on demand | delete as copy of `.guidance/packs/go/cobra/guide.md` |  |
| cobra-wiring | Each verb is a `cobra.Command` with completion wired | `CLAUDE.md › Go conventions › CLI conventions` | 30 | development | check-backed | on demand | move on demand to `.guidance/project/go-conventions.md` | The completion drift test, at `internal/cli/integration/completion_drift_test.go`; the moved text names that path. |
| exit-codes | Exit codes 0–3, shuttled through `*exitError` | `CLAUDE.md › Go conventions › CLI conventions` | 20 | development | judgment | on demand | move on demand to `.guidance/project/go-conventions.md` | The pack has the codes; the `*exitError` route is aiwf's. |
| output-and-logging | JSON envelope; diagnostic logging opt-in, default-off, to a file, never stderr unless configured | `CLAUDE.md › Go conventions › CLI conventions` | 110 | development | check-backed | on demand | move on demand to `.guidance/project/go-conventions.md` | A project override of the Go pack, which sends `log/slog` to stderr; the router's Go entry names this document as overriding the pack. |
| cli-flags | Standard flags; config by flags, env or `aiwf.yaml` | `CLAUDE.md › Go conventions › CLI conventions` | 18 | development | judgment | on demand | move on demand to `.guidance/project/go-conventions.md` |  |
| no-package-state | No package-level mutable state | `CLAUDE.md › Go conventions › CLI conventions` | 23 | development | judgment | on demand | delete as copy of `.guidance/packs/go/cobra/guide.md` |  |
| commit-trailers | Mutating verbs write `aiwf-verb`/`aiwf-entity`/`aiwf-actor` trailers | `CLAUDE.md › Go conventions › Commit conventions` | 19 | development | judgment | on demand | move on demand to `.guidance/project/verb-design.md` |  |
| conventional-commits | Conventional Commits subjects that say why; one logical change per commit | `CLAUDE.md › Go conventions › Commit conventions` | 18 | development | judgment | primed | keep |  |
| no-coauthor | No `Co-Authored-By:` for a non-human agent, overriding a harness's default | `CLAUDE.md › Go conventions › Commit conventions` | 158 | development | check-backed | primed | tighten | Must carry: the ban, that it overrides a harness attribution default, that a human co-author is unaffected, and the enforcement pointer; D-0096 keeps the reasoning. Primed because a harness asks for the line on every commit. |
| release-process | Tag releases on `main` after a release commit edits the changelog; the changelog audit | `CLAUDE.md › Go conventions › Release process` | 238 | development | check-backed | on demand | move on demand to `.guidance/project/release.md` | `changelog-check.yml`. |
| dependencies | Minimal dependencies; static binaries; the `go.mod` floor versus the CI toolchain | `CLAUDE.md › Go conventions › Dependencies` | 46 | development | judgment | on demand | move on demand to `.guidance/project/go-conventions.md` | The pack has minimal deps and `CGO_ENABLED=0`; the floor rule is aiwf's. |
| naming | Short singular package names, no stuttering, doc comments, capitalized acronyms | `CLAUDE.md › Go conventions › Naming` | 28 | development | judgment | on demand | move on demand to `.guidance/project/go-conventions.md` |  |
| closed-set-enums | Closed-set enums ship only values with a call site | `CLAUDE.md › Go conventions › Type design` | 16 | development | judgment | on demand | delete as copy of `.guidance/packs/go/cobra/guide.md` |  |
| kinds-hardcoded | The six kinds and status sets are hardcoded in Go | `CLAUDE.md › Go conventions › Type design` | 23 | development | judgment | primed | merge into commit-six-kinds |  |
| title-cap | Title and slug share `entities.title_max_length`; hard reject; retitle re-derives a tracking slug | `CLAUDE.md › Go conventions › Type design` | 94 | development | judgment | on demand | move on demand to `.guidance/project/verb-design.md` |  |
| verb-undo | A verb's design names the verb that undoes it | `CLAUDE.md › Go conventions › Designing a new verb` | 58 | development | judgment | on demand | move on demand to `.guidance/project/verb-design.md` |  |
| same-state-convergence | Resolve, then converge: R1, R2, multi-surface NoOps, the precondition seams, canonical ids, composite ids | `CLAUDE.md › Go conventions › Designing a new verb › Same-state convergence — resolve, then converge` | 817 | development | check-backed | on demand | move on demand to `.guidance/project/verb-design.md` | `PolicyClaimGuardPresence` and the NoOp invariant cover parts. |
| skills-policy | Every verb is reachable through a skill or an allowlist entry | `CLAUDE.md › Go conventions › Skills policy` | 64 | development | check-backed | on demand | move on demand to `.guidance/project/shipped-content.md` | `skill_coverage.go`. |
| shipped-surfaces | Shipped surfaces carry no real ids, paths, statuses or history | `CLAUDE.md › Go conventions › Skills policy` | 269 | development | check-backed | on demand | move on demand to `.guidance/project/shipped-content.md` | `skill-body-id` covers the id subset. |
| enforcement-map | Where each rule is enforced, from pre-commit to the release tag | `CLAUDE.md › Go conventions › What's enforced and where` | 712 | development | check-backed | on demand | move on demand to `.guidance/project/enforcement.md` |  |
| codex-read-claude | Codex reads `CLAUDE.md` in full before changing the repository | `AGENTS.md` | 40 | development | judgment | primed | tighten | Must carry the required read; once `CLAUDE.md` holds only primed rules, the read covers only them (M-0335 AC-1). |
| codex-host-guidance | Codex follows its own rendered guidance and skills, not Claude's | `AGENTS.md` | 65 | development | judgment | primed | keep |  |
| codex-explicit-read | A Markdown link is not an automatic Codex import | `AGENTS.md` | 18 | development | judgment | primed | keep |  |
| router-precedence | Read `CLAUDE.md` in full; it takes precedence over the packs | `.guidance/project.md` | 42 | development | judgment | primed | tighten | Becomes the task router drafted below; must carry that `CLAUDE.md`'s primed rules take precedence over the packs. |
| frag-intro | The rules have no chokepoint except code-health priming | `F`; `internal/skills/embedded-guidance/claude/skill-invocation.md` | 38 | operating | judgment | primed | keep |  |
| frag-codex-skill-invocation | Codex loads and follows a named skill's `SKILL.md` before its steps; `/skills` and `$skill-name` are prompt controls, not shell commands | `internal/skills/embedded-guidance/codex/skill-invocation.md` | 46 | operating | judgment | primed | keep | Spliced into the fragment's opening for Codex. |
| frag-bless-mode | Body edits keep a review-before-commit window | `F › Body edits keep a review-before-commit window` | 108 | operating | judgment | primed | keep |  |
| frag-entity-files | Entity files are verb- and template-managed | `F › Entity files are verb- and template-managed` | 93 | operating | check-backed | primed | keep |  |
| frag-test-pins-rule | A test pins a rule, not an input | `F › A test pins a rule, not an input` | 77 | operating | judgment | primed | keep |  |
| frag-gap-names-where | A gap names what is wrong and where | `F › A gap names what is wrong and where` | 82 | operating | judgment | primed | keep |  |
| frag-derive-facts | Keep the reasoning; derive the facts | `F › Keep the reasoning; derive the facts` | 163 | operating | judgment | primed | keep |  |
| frag-measurement | Nothing is settled except by measurement | `F › Nothing is settled except by measurement` | 96 | operating | judgment | primed | keep |  |
| frag-re-derive | On revising, re-derive before you rewrite | `F › On revising, re-derive before you rewrite` | 252 | operating | judgment | primed | keep |  |
| frag-code-health-intro | Hold the highest-leverage forces while writing code; the full rubric is a skill | `F › Code-health priming — for the code you write here` | 22 | operating | judgment | primed | keep |  |
| frag-d1 | D1 — pin behavior, not implementation | `F › D1 — pin behavior, not implementation` | 20 | operating | judgment | primed | keep | Overlaps the code-health pack; the fragment ships to repositories without it. |
| frag-d5 | D5 — findings become checks | `F › D5 — findings become checks` | 74 | operating | judgment | primed | keep |  |
| frag-c1 | C1 — single source of truth | `F › C1 — single source of truth` | 19 | operating | judgment | primed | keep | Overlaps the code-health pack. |
| frag-c3 | C3 — atomic writes | `F › C3 — atomic writes` | 15 | operating | judgment | primed | keep | Overlaps the code-health pack. |
| frag-b1b2 | B1/B2 — typed interfaces and validated schemas at boundaries | `F › B1/B2 — typed interfaces and validated schemas at boundaries` | 16 | operating | judgment | primed | keep | Overlaps the code-health pack. |
| frag-e1 | E1 — structured logs | `F › E1 — structured logs` | 14 | operating | judgment | primed | keep | Overlaps the code-health pack. |
| frag-h1 | H1 — reuse over duplication | `F › H1 — reuse over duplication` | 46 | operating | judgment | primed | keep |  |
| frag-h3 | H3 — additions carry | `F › H3 — additions carry` | 68 | operating | judgment | primed | keep |  |

---

## Release note

## Decisions made during implementation

- On-demand development guidance lives under `.guidance/project/`, beside the router; `docs/` is for documentation.
- The working-in-this-repo rule: planning-state verbs, release commits and doc edits may commit directly to `main`; every code change goes on a branch, `wf-patch` for a small fix and a ritual branch for the rest, merged into `main` locally without a PR. Measured over `origin/main`'s first-parent history from 2026-08-05T16:43:08Z to 2026-10-04T16:43:08Z (`git log origin/main --first-parent --merges|--no-merges --since … --until …`, tip 2026-10-01): 100 merges and 789 direct commits, of which 712 carry an `aiwf-verb` trailer and 13 touch `*.go` files. Those 13 direct code commits are what the rule now routes through `wf-patch`.
- aiwf's diagnostic-logging rule, opt-in and default-off to a file, stays as a project override of the Go pack, which sends `log/slog` to stderr; it moves on demand to the Go conventions document, which the router names as overriding the pack.
- The wrap rituals' own orders stand: the milestone wrap merges and then promotes, and the epic wrap promotes as its last verb-driven commit and merges after, since the merge is a plain commit that carries no scope trailer. `CLAUDE.md`'s "promote lands last so a delegated scope stays live for the merge commit" goes with the gate-per-mutation copy.
- The id-token rule is per file, as the `aiwf-check` skill's table states it: an entity body cites real ids at canonical width and rejects placeholders, and a doc may use a `<prefix>-NNNN` placeholder for an invented example. `CLAUDE.md`'s ban on placeholders in all committed prose goes with the no-fake-ids copy.
- Clauses of `CLAUDE.md`'s copies of fragment rules that no shipped surface states move into the fragment, added to the rule they belong to: the Q&A trigger, the "something else" option and confirm-in-one-line for one decision at a time; abort-on-deviation for every declared-sequence gate; remote-branch deletes among the outward actions; and surfacing what you noticed. They add about 50 always-on words for consumers.
- Operating guidance follows the same placement test as development guidance: a primed operating rule lives in the shipped fragment, an on-demand one in the shipped skill for its task. The disposition `move to the shipped skill <path>` carries it, and E-0092's scope says so.

## Validation

Environment: the aiwf devcontainer, Linux, `go1.25.12 linux/amd64`, at commit `9b18d4819` on `milestone/M-0349-inventory-every-guidance-rule-and-decide-its-home-and-form`. The guidance was frozen at `3fcc69611`, the commit that set `guidance.enabled: false`.

- `go test -count=1 -run 'TestE0092Inventory|TestInventory' -v ./internal/policies/`, the inventory test. Expected exit 0. Observed `ok`. The live test passes, which is AC-1's coverage of every section and fragment rule as frozen, and AC-2's and AC-4's row rules. Each rule's fixture passes too.
- AC-5's targets, from the same run's log line. Expected: a handwritten primed target for each host, derived from the table. Observed: `M-0349 AC-5 handwritten primed targets: claude-code 1629, codex 1758`. Each target counts every development rule that ends primed, with a tightened rule at its current size; the router at its drafted size; and the H1 and section headings of the rules that stay. Codex's target adds the `AGENTS.md` preamble. The ceiling policy measures 9647 and 9776 words today (`go test -count=1 -run TestPolicy_GuidanceCeiling -v ./internal/policies/`), and M-0337 lowers the ceilings to these targets.

## Deferrals

- (none)

## Reviewer notes

- (none)
