---
id: M-0073
title: aiwf-list skill, aiwf-status skill tightening
status: done
parent: E-0020
tdd: advisory
acs:
    - id: AC-1
      title: aiwf-list embedded skill exists with list-shaped description
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf-list skill body covers recipes, output, JSON, list-vs-status criteria
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf-status skill description tightened to narrative-snapshot phrasings
      status: met
      tdd_phase: done
    - id: AC-4
      title: aiwf-status skill body redirects to aiwf list for tree queries
      status: met
      tdd_phase: done
    - id: AC-5
      title: Both skills materialize via aiwf init and aiwf update
      status: met
      tdd_phase: done
---

# M-0073 — aiwf-list skill, aiwf-status skill tightening

## Goal

Route AI discovery to `aiwf list` as the hot path for tree queries by adding a focused `aiwf-list` embedded skill, and demote `aiwf-status` to its real role (human-curated narrative) by tightening its description to snapshot phrasings and adding an explicit redirect to list in its body.

## Context

The skill host (Claude Code) loads skills based on description-match scoring against the user's prompt; specificity beats generality. Today `aiwf-status` is the AI's primary read skill, which is wrong — status is human-curated narrative whose contents deliberately exclude done/closed entities. With the verb shipped in M-0072, the right division of labor is: list answers query-shaped prompts ("list all done milestones in E-0013"), status answers narrative-shaped prompts ("what's next?"). This milestone makes that division materialize through skill descriptions: list's description is dense with list-shaped natural-language phrasings; status's description tightens to narrative-snapshot phrasings; status's body adds an explicit redirect so an AI that lands on status for a query-shaped prompt recovers in one hop.

## Acceptance criteria

### AC-1 — aiwf-list embedded skill exists with list-shaped description

### AC-2 — aiwf-list skill body covers recipes, output, JSON, list-vs-status criteria

### AC-3 — aiwf-status skill description tightened to narrative-snapshot phrasings

### AC-4 — aiwf-status skill body redirects to aiwf list for tree queries

### AC-5 — Both skills materialize via aiwf init and aiwf update

## Constraints

- The `aiwf-list` skill description *must* enumerate list-shaped natural-language query phrasings the AI host can match against — examples include "list every milestone with status X", "find all entities matching Y", "filter by kind/parent", "show me all proposed ADRs". A description that just says "documents the list verb" fails the discoverability priority test.
- The `aiwf-status` description *must not* contain list-shaped phrasings after this milestone. Specifically: phrases like "list every X", "find all Y", "filter Z" — those belong to list. Status's description scope is narrative-snapshot only.
- Skills under `internal/skills/embedded/` are the *source*; `.claude/skills/aiwf-*/` are gitignored materializations. Edits land in the embedded source; AC-5 verifies the materialization path still works.
- Skills are advisory per the kernel principle ("framework correctness must not depend on LLM behavior") — this milestone does not change that. The mechanical drift guard against skills referencing missing verbs lands in M-0074.

## Design notes

- Skill split rationale (judgment): per the discoverability-priority lens, list and status answer different prompt shapes. Folding them under one skill (option 1) or one topical skill (option 2) dilutes the description that should be specific to list. Splitting (option 3) gives each skill a focused description tuned to its query shape. The judgment rule itself is captured in the ADR allocated in M-0074.
- `aiwf-list` skill body sections (suggested shape — refine at start-milestone):
  1. *What it does* — the verb's mechanical surface, in plain prose.
  2. *When to use* — list-shaped query phrasings the AI can match; explicit examples.
  3. *Recipes* — common filter combinations (`--kind milestone --status done --parent E-NN`, `--kind contract`, no-args summary, `--archived`).
  4. *Output* — text default, `--format=json` envelope shape.
  5. *List vs. status decision criteria* — when to reach for which.
- `aiwf-status` skill edits:
  - Description: drop list-shaped phrasings; keep "what's next?", "where are we?", "what's in flight?", "give me a summary".
  - Body: add a top-level note — *"For programmatic tree queries (every milestone with X, all entities by status Y, filter by parent), prefer `aiwf list` — that is the hot path. This skill covers narrative snapshots for human readers."*
- `tdd: advisory` reflects that skill content is prose; the materialization mechanics are already TDD-covered upstream (E-0003, E-0011). AC-5 is a stamp check (does the file appear at `.claude/skills/aiwf-list/SKILL.md` after `aiwf init`?).

## Surfaces touched

- `internal/skills/embedded/aiwf-list/SKILL.md` (new)
- `internal/skills/embedded/aiwf-status/SKILL.md` (description + body)

## Out of scope

- Skills-coverage policy enforcement. M-0074 owns that.
- Any change to other embedded skills (`aiwf-add`, `aiwf-promote`, `aiwf-contract`, etc.).
- A new `aiwf-show` skill — explicitly deferred and recorded as a follow-up gap in M-0074.
- The judgment-rule ADR — allocated and authored in M-0074, not here.

## Dependencies

- M-0072 (verb must exist before its skill is meaningful; the skill body's recipes invoke `aiwf list` and would fail at materialization-time validation if the verb were missing).

## Coverage notes

- This milestone is `tdd: advisory` — its deliverables are prose (skill bodies)
  and a one-time materialization check, not testable code. No new code coverage
  to report.
- Materialization is guarded by `internal/skills/skills_test.go`'s
  `TestList_AllShippedSkillsPresent` (updated to expect 13 skills, including
  `aiwf-list`) and `TestList_ContentNonEmptyAndYAMLFrontmatter` (every embedded
  SKILL.md starts with `---` and the `name:` field matches its directory). Both
  pass on the post-milestone tree.
- The materialization pipeline itself is covered by `internal/initrepo/...`
  and `internal/skills/...` tests upstream of this milestone (E-0003, E-0011);
  M-0073 ships content, not engine changes, and inherits that coverage.

## References

- E-0020 epic spec (this milestone's parent).
- `internal/skills/embedded/aiwf-status/SKILL.md` — current state; this milestone tightens it.
- `internal/skills/embedded/aiwf-contract/SKILL.md` — precedent for topical multi-verb skills (the one that does NOT split despite covering many verbs, because contract has uniform discovery shape).
- `aiwf init` / `aiwf update` flows — the materialization path AC-5 verifies.

---

## Work log

(filled during implementation)

## Decisions made during implementation

- **AC-3 minimal tightening (disclosed):** the spec called for tightening `aiwf-status`'s description to drop list-shaped phrasings like "list every X", "find all Y", "filter Z". Audit found the original description never contained any such phrasings — its phrasings were already narrative-only ("what's next?", "where are we?", "what are we working on?", "current status?", "what's in flight?"). The AC's contract was therefore satisfied at start-time. Implementation went further: added a "narrative-shaped" framing word, added "give me a summary" to the example list, and added an explicit "for those, use `aiwf list`" redirect — bonus discoverability work, not the AC's literal contract. Logging here so the closure doesn't read as having corrected drift that wasn't there.

## Validation

```
$ aiwf doctor 2>&1 | head -5
binary:    (devel) (working-tree build)
config:    ok
actor:     human/peter (from git config user.email)
skills:    ok (13 skills, byte-equal to embed)
ids:       ok (no collisions)

$ ls .claude/skills/ | grep -E "list|status"
aiwf-list
aiwf-status

$ head -3 .claude/skills/aiwf-list/SKILL.md
---
name: aiwf-list
description: Use to filter the planning tree by kind, status, parent, or archive flag — answers prompts like "list every milestone with status X", ...

$ head -4 .claude/skills/aiwf-status/SKILL.md
---
name: aiwf-status
description: Use for narrative-shaped state questions — "what's next?", "where are we?", ...
---

$ go test ./internal/skills/...
ok  	github.com/23min/ai-workflow-v2/internal/skills	0.371s
```

Both skills materialize from `internal/skills/embedded/` to
`.claude/skills/aiwf-*/SKILL.md` byte-equal; doctor reports 13 skills
total (one more than pre-milestone, accounting for the new `aiwf-list`).

## Deferrals

- (none)

## Reviewer notes

- **Read AC-3's "minimal tightening" disclosure** (in *Decisions made during
  implementation*, above). It documents that the original `aiwf-status`
  description was already narrative-only — there were no list-shaped phrasings
  to drop. The work landed (added narrative framing + redirect) but the AC's
  literal contract was satisfied at start-time. Honest log rather than a
  re-revert.
- **The split-skill design follows ADR-0006's "discoverability-priority split"
  case.** `aiwf-list` and `aiwf-status` cover closely related read verbs over
  the same planning tree but are split because their description-match prompt
  shapes diverge: list answers structured-filter prompts, status answers
  narrative-state prompts. Bundling them would dilute either description.
- **The redirect note in `aiwf-status`'s body is the only structural cue
  pointing AI assistants from status to list.** If the host's
  description-match scoring still routes a filter-shaped prompt to status,
  the body's bold-paragraph redirect is what catches the recovery in one hop.
- **`tdd: advisory` does not mean "skip TDD phase tracking".** All five ACs
  are stamped `tdd_phase: done` post-wrap (the kernel's `acs-tdd-audit`
  rule warns when `met` lands without phase, even under advisory). Earlier
  closure left phase absent and was caught at the second-audit pass.
