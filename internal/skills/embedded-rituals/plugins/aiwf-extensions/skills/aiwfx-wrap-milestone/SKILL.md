---
name: aiwfx-wrap-milestone
description: Closes an aiwf milestone — verifies all ACs met, runs scoped doc-lint, finalizes the milestone spec's wrap-side sections, promotes status to done, prepares the wrap commit. Use when the user says "wrap M-NNNN" or "finish the cache milestone" and self-review per `aiwfx-start-milestone` has passed. Commit and push require explicit human approval.
---

# aiwfx-wrap-milestone

Closes a milestone. Verifies completeness, finalizes the milestone spec's wrap-side sections, promotes the milestone to `done`, prepares the single wrap commit.

## When to use

The milestone's implementation is complete and self-reviewed (`aiwfx-start-milestone` step 7 ran clean). The user says: *"wrap M-NNNN"*, *"finish M-0007"*, *"close out the cache milestone"*.

If the milestone isn't actually done — failing tests, unmet ACs, broken build — stop and report. Don't paper over.

## Workflow

### 1. Verify completion

- Re-read the milestone spec. Walk every AC in frontmatter `acs[]`. Confirm each has at least one test that exercises it green.
- Run `aiwf show M-NNNN`; confirm every AC's `status` is terminal (`met`, `deferred`, or `cancelled`) — none `open`. Under `tdd: required`, also confirm every `met` AC has `tdd_phase: done` (the kernel's `acs-tdd-audit` will surface it otherwise).
- Run `aiwf check`. **Zero error-severity findings on the milestone.** The relevant codes: `acs-shape`, `acs-tdd-audit`, `milestone-done-incomplete-acs`, `acs-body-coherence`. Warnings (e.g. `acs-body-coherence`) are advisory but worth resolving before wrap.
- Run the full test suite. **All pass.**
- Run the project's build. **Green.**
- Run the project's full lint gate — the same linter set CI runs on push (e.g. a `make ci` target), not a subset like `go vet` alone. **Clean.** Unpushed branches accumulate lint debt invisibly; the wrap is the cheap moment to catch it.

If anything is red, stop and report. Wrap does not paper over failure.

### 2. Independent two-lens review — before the wrap

This gates milestone *closure*, not the per-commit work: the implementation commits are already in, but the milestone is not yet wrapped, so there is still a chance to fix things *inside* the milestone. Findings become corrective commits on the milestone branch — before any AC flips to `met` and before the commit gate (step 8). The review feeds the human gate; it does not replace it.

Dispatch a **fresh-context reviewer** (a subagent with no authorship attachment) over the milestone's full change-set (`git diff <base>..HEAD`), briefed adversarially per `wf-review-code` §"Independence" (enumerate the load-bearing claims, instruct *verify by measuring not reasoning*, name the risk areas). Run two lenses:

- **Code-quality** (`wf-review-code`): correctness, AC coverage, branch-coverage discipline, conventions, docs. For a large milestone, *slice the review by concern or file group* — one agent over thousands of lines goes shallow, the exact failure independence is meant to avoid.
- **Design-quality** (`wf-rethink`): run on the design unit(s) the milestone introduced — those matching the `wf-rethink` trigger (a new module/package boundary, core abstraction, or data model; see `wf-rethink` §"The non-trivial-design trigger"). `wf-rethink` is per-unit by rule ("never run it over the whole codebase at once"), so **name the unit(s)** rather than pointing it at the whole diff. If the milestone introduced no such surface — only mechanical or local change — there is nothing to rethink; say so and move on.

Handle the verdict: fix every blocking finding as a corrective commit on the milestone branch; re-verify (re-run step 1's gates) if code changed; confirm judgment-level fixes by re-dispatching a fresh reviewer *scoped to the changed surface* (mechanical fixes can be confirmed mechanically — re-run the gate or scan). Record the review outcome under the spec's `## Reviewer notes` (step 4).

Then the residual self-checks — cheap, and *not* a substitute for the independent pass above:

- Skim for `TODO` / `FIXME` left behind. If they're intentional, document them in the milestone spec's `## Reviewer notes` section. If they're unintentional, fix or open as gaps (`aiwf add gap --title "..." --discovered-in M-NNNN`).
- Skim for debug code, commented-out blocks, scratch logging. Remove.
- Confirm public-API or schema changes are reflected in README, inline docs, or wherever the project publishes its surface.

### 3. Doc-lint sweep (scoped)

Invoke `wf-doc-lint` against the milestone's change-set (every file the milestone branch touched since diverging from its base). Surface the report inline.

If the report is clean, note "doc-lint: clean" and continue. If findings:

- **Broken code references** — fix in this milestone, or open a gap.
- **Removed-feature docs** — same.
- **Orphan files / TODOs** — record under the spec's `## Reviewer notes` for the reviewer to consider; don't block wrap.

`wf-doc-lint` reports only — it does not rewrite prose. Any prose changes happen here as deliberate edits.

### 4. Finalize the milestone spec's wrap-side sections

The v1 separate tracking doc is gone. The milestone spec itself carries the wrap-side sections; finalize them in place:

- `## Work log` — confirm one entry per AC with the final outcome and commit SHA. The phase timeline is in `aiwf history M-NNNN/AC-<N>`; don't duplicate dates here.
- `## Decisions made during implementation` — confirm every mid-flight decision is captured (each should already have an `ADR-NNNN` or `D-NNN` from `aiwfx-record-decision` invocations during work).
- `## Validation` — paste the test-suite and build results.
- `## Deferrals` — list any work this milestone deliberately punted; for each, **open a gap entity** so it survives:

  ```bash
  aiwf add gap --title "<deferred-work>" --discovered-in M-NNNN
  ```

  Then mirror the resulting `G-NNN` id here. Deferred ACs (status `deferred`) get a one-line note pointing at the receiving milestone or gap.
- `## Reviewer notes` — trade-offs, deliberate omissions, places where the obvious approach was rejected. The reviewer agent reads this first.

For ACs that were `cancelled` mid-implementation, link to the `D-NNN` decision (or the conversation context) explaining why under the cancelled AC's body section. The kernel only guards the structural state (`status: cancelled`, position-stable in `acs[]`); the why is the human's narrative.

### 5. Promote the milestone status

```bash
aiwf promote M-NNNN done
```

aiwf validates `in_progress → done`, rewrites frontmatter, commits with `aiwf-verb: promote` trailers. The promote commit is *separate* from the implementation commits — it captures the moment of closure.

### 6. Update the roadmap

```bash
aiwf render roadmap --write
```

The roadmap reflects the milestone's new status without hand-edits.

### 7. Stage all changes and prepare the wrap commit

The milestone spec carries all the wrap-side prose now (Work log, Validation, Deferrals, Reviewer notes). Stage it:

```bash
git add work/epics/E-NNNN-<slug>/M-NNNN-<slug>.md
git status
git diff --staged --stat
```

Draft a conventional commit message: `feat(<scope>): <one-line summary> (M-NNNN)`.

### 8. 🛑 Commit gate

Show the user:
- `git diff --staged --stat`
- The proposed commit message.
- A summary of what landed: AC count green, doc-lint summary, deferrals opened (with gap ids).

**Stop and wait for explicit "commit" approval.**

### 9. After commit approval

The wrap commit touches a milestone entity file (`work/epics/E-NNNN-<slug>/M-NNNN-<slug>.md`), so it carries the three required trailers — `aiwf-verb: wrap-milestone`, `aiwf-entity: M-NNNN`, `aiwf-actor: human/<id>`. Skipping any one of them trips the kernel's `provenance-untrailered-entity-commit` finding on the file touch. Parallel shape to `aiwfx-wrap-epic`'s trailered merge + wrap-artefact commits:

```bash
git commit -m "<approved-message>" \
  --trailer "aiwf-verb: wrap-milestone" \
  --trailer "aiwf-entity: M-NNNN" \
  --trailer "aiwf-actor: human/<id>"
```

The trailer keys are quoted from CLAUDE.md §"Commit conventions" verbatim — variant casings (e.g. `Aiwf-Verb`) fail the kernel's trailer-keys policy.

### 10. 🛑 Push gate

Confirm with the user before pushing. Then:

```bash
git push -u origin milestone/M-NNNN-<slug>
```

Open the PR if the project's flow is PR-driven. Reference the milestone id in the PR title.

### 11. After merge

If the project uses an epic-integration branch, merge the milestone branch into the epic branch following the same pattern as `aiwfx-wrap-epic`'s epic-into-trunk merge: stage the merge **without committing** so the merge commit's trailer set can be attached explicitly.

```bash
git checkout epic/E-NNNN-<slug>
git merge --no-ff --no-commit milestone/M-NNNN-<slug>
```

`--no-ff` preserves the milestone as a single merge commit (rather than fast-forwarding individual milestone commits into the epic). `--no-commit` leaves the merge staged so the commit-emitting step is the one carrying trailers — without it, git produces an untrailered merge commit and the kernel's `trailer-verb-unknown` warning fires (the operator's hand-typed `aiwf-verb: merge` is a fabrication; `merge` is a git concept, not a recognized ritual or kernel verb).

Resolve the operator identity from `git config user.email` (per CLAUDE.md *Provenance model* §"Identity is runtime-derived"); do not hardcode `<id>`. Then commit with the three required trailers and a Conventional Commits subject:

```bash
git commit -m "chore(milestone): wrap M-NNNN — <milestone title>" \
  --trailer "aiwf-verb: wrap-milestone" \
  --trailer "aiwf-entity: M-NNNN" \
  --trailer "aiwf-actor: human/<id>"
```

The trailer keys are quoted from CLAUDE.md §"Commit conventions" verbatim — `aiwf-verb`, `aiwf-entity`, `aiwf-actor`. Variant casings (e.g. `Aiwf-Verb`) fail the kernel's trailer-keys policy. The `aiwf-verb: wrap-milestone` value names the ritual that produced the commit; the kernel's `trailer-verb-unknown` rule recognizes it via the ritualVerbs allowlist (sourced from the embedded ritual snapshot per G-0190), mirroring `aiwfx-wrap-epic`'s `aiwf-verb: wrap-epic` trailer at the equivalent step.

**Why an `aiwf-verb` trailer on a `git merge` commit.** The merge IS a kernel-meaningful structural transition (the milestone's work joins the epic's history); `aiwf-verb: wrap-milestone` records the *ritual* that produced it, not the underlying git operation. **Do NOT** write `aiwf-verb: merge` — `merge` is neither a Cobra verb nor an allowlisted ritual value; the `commit-msg` git hook materialized by `aiwf init` / `aiwf update` (per G-0218's primary chokepoint) refuses the commit at message-composition time with a named-value error pointing at the canonical `aiwf-verb: wrap-milestone` shape. Historical commits authored before the hook landed are still surfaced by the `trailer-verb-unknown` rule at pre-push, with two cleanup paths (`aiwf acknowledge illegal <sha>` or push the warning forward, since amend is blocked by the trunk-aware push model).

Record the resulting merge commit SHA wherever the project tracks merge history (the milestone's `## Work log` section is the natural place).

Then:

- Delete the milestone branch on origin.
- Run `aiwf render roadmap --write` once more if the merge introduced any state aiwf would notice.

## Constraints

- 🛑 **Never commit or push without explicit human approval** (steps 8, 10).
- All ACs must be green before wrap proceeds. Wrap does not bury failure.
- Branch-coverage hard rule applies — re-run the audit if any code changed since `aiwfx-start-milestone`'s self-review.
- Deferrals must be captured as gaps. Don't leave deferred work as a `## Deferrals` bullet that nothing else points at.

## Anti-patterns

- *Wrapping with red tests.* Either fix the tests, escalate the AC failure, or cancel the milestone (`aiwf cancel M-NNNN`). Don't wrap broken work as done.
- *Wrapping with open ACs.* The kernel's `milestone-done-incomplete-acs` finding will fire — `--force` lands the verb but leaves the standing check red. Resolve every AC to a terminal state (`met`/`deferred`/`cancelled`) before wrap.
- *Silent deferrals.* Every "we'll do that later" gets a gap entity.
- *Skipping doc-lint.* Doc drift compounds; the milestone wrap is the cheap moment to catch it.
- *Slipping unrelated code into the wrap commit.* If the change isn't part of this milestone, it's a separate `wf-patch`.

## Next step

If this is the last milestone in the epic: → `aiwfx-wrap-epic E-NNNN`.
Otherwise: → `aiwfx-start-milestone <next-M>`.
