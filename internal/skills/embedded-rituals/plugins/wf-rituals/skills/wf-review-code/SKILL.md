---
name: wf-review-code
description: Code review checklist with verdict (approve / request-changes / questions). Reviews a diff for correctness, AC coverage, branch-coverage discipline, conventions, and documentation hygiene. Findings are classified blocking vs. non-blocking with file:line references. Use when the user says "review this", "check my changes", or before a change is proposed for merge.
---

# wf-review-code

A structured review pass over a diff. Produces a verdict and a list of findings with file:line locations. The skill is a checklist, not a rewrite — it surfaces issues, the human decides what to do with them.

## When to use

- The user says "review this," "check my changes," "review my branch," "review the PR," or similar.
- A self-review pass is needed before declaring implementation complete — a calling ritual (a patch workflow, a milestone-wrap workflow, etc.) invokes this near the end of its sequence.
- A change is ready for review — in PR form, on a branch about to be merged, or in any other shape the consuming project's flow uses.

## Workflow

### 1. Understand the scope

- What change does this diff propose? Read the PR description, the commit message, the issue, the spec, or whatever the project uses to capture the goal.
- If the goal isn't stated anywhere, ask before reviewing — a review without a stated goal can only judge "does this look like code," not "does this do the right thing."
- If the change is part of a larger spec (acceptance criteria, contract requirements), open it now.

### 2. Walk the diff

- Look at every changed file. `git diff <base>..HEAD`, or the diff view in the project's review host (PR page, branch comparison, etc.).
- For each file: does the change serve the stated goal? Flag changes that don't.
- Flag *unrelated* changes ("while I was in there"). Suggest splitting them out unless they're trivially defensible.

### 3. Correctness

- Logic matches the requirement.
- Edge cases handled: null, empty, boundary, large input, concurrent input where relevant.
- Error handling is adequate. No silently-swallowed errors. No panics on user input.
- No off-by-one errors, race conditions, or resource leaks (file handles, network connections, goroutines, subscriptions, timers).
- The change can be undone — if the diff bakes in a one-way decision (irreversible migration, schema change, public-API breaking), that's worth flagging even when correct.

### 4. Constraints (project-stated invariants)

- If the project's spec lists invariants the diff must respect (banned patterns, shim policies, capability boundaries, removal triggers), check each.
- Any violation is **blocking** unless the diff also includes a ratifying decision record (ADR or equivalent) that re-opens the constraint.

### 5. Tests

- A test exists for the new or changed behavior — every acceptance criterion if AC-driven, otherwise every behavioral change.
- **Branch coverage:** every reachable conditional branch in the diff has at least one test exercising it. (See `wf-tdd-cycle` § "Branch-coverage audit" — same hard rule.)
- Tests are deterministic.
- Tests cover happy path *and* edge cases.
- No tests removed without an explicit reason.
- If the project's TDD discipline expects red-before-green evidence (commit history, work log, linked PRs or branches), the trail is present and consistent.

### 6. Conventions

- Naming follows project conventions (look at neighboring files to infer if no style guide exists).
- File placement follows project structure.
- No hardcoded values that should be configurable (URLs, paths, credentials, feature flags).
- No secrets, no PII, no real customer data in tests.

### 7. Documentation

- Public-API change → README, reference docs, or whatever the project uses to publish surface.
- Comments explain *why* where non-obvious — hidden constraint, subtle invariant, workaround for a specific bug. Comments that re-describe *what* the code does are noise; flag them.
- If the project keeps a tracking doc, work log, or change log alongside the diff, the entry is present and accurate.

### 8. Verdict

For each finding, classify:

- **Blocking** — must be fixed before the change merges. Correctness, constraint violations, missing tests for AC, security issues.
- **Track for later** — worth recording somewhere durable (the project's gap log, an issue, a ticket), not this change's scope. Note rough sizing if useful.
- **Non-issue** — acknowledged, no action.

Then give an overall:

- **Approve** — no blocking findings; non-blocking findings noted.
- **Request changes** — blocking findings exist; list them with file:line.
- **Questions** — review can't proceed without clarification; list the questions.

## Output format

```markdown
# Review — <one-line summary of the diff>

**Verdict:** approve | request-changes | questions

## Blocking findings
- `path/to/file.ts:42` — <what's wrong; what to do about it>

## Track for later
- `path/to/file.go:101` — <observation; sizing if non-trivial>

## Non-issues / acknowledged
- <thing the reviewer noticed but the author already addressed in the change>

## Overall
<one-paragraph assessment>
```

## Anti-patterns

- *Reviewing without reading the goal.* "It looks fine to me" without knowing what the change is for is no review.
- *Style-only reviews.* If every finding is about naming or formatting, the reviewer either skipped the substance or the diff is genuinely trivial — say so.
- *Bundling blocking and non-blocking together.* The author needs to know what to fix before the change merges vs. what to consider afterwards. Keep them separate.
- *Approving with unread tests.* A diff with new tests gets the tests reviewed, not just the implementation.
- *Vague findings.* "This could be cleaner" without a specific suggestion is a complaint, not a review item. Either propose a concrete change or move it to "non-issue."

## Constraints

- 🛑 Findings include `file:line` references. A finding without a location is unactionable.
- Branch-coverage hard rule applies even at review time. If the diff is missing branch-coverage discipline, that's blocking.
- The reviewer never edits the diff. The skill emits the report; the author makes the changes.
