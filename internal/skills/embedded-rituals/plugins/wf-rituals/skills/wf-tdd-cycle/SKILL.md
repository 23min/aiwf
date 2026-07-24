---
name: wf-tdd-cycle
description: Red/green/refactor for a single acceptance criterion or feature unit, ending with a hard-rule branch-coverage audit and a required vacuity check (does a covered assertion actually catch the bug). Write a failing test, write the minimum code to pass, refactor, then walk every reachable conditional branch and confirm an explicit test exercises it. Use during milestone implementation and inside `wf-patch` when the change touches logic.
---

# wf-tdd-cycle

A single iteration of test-first development for one acceptance criterion or one focused feature unit. Ends with a branch-coverage audit that is a **hard rule**, not a guideline — then a required `wf-vacuity` check on whether those now-covered assertions can actually fail.

## When to use

- The user is implementing one acceptance criterion of an in-progress milestone.
- The user is on a `wf-patch` branch and the change touches logic (not pure config / dependency bumps).
- Any other moment where a unit of behavior change wants a test before it has code.

If you find yourself running `wf-tdd-cycle` for a config nudge, you don't need it.

## The cycle

### RED — Write the failing test first

- Write the test(s) that describe the expected behavior. Test names follow the project's convention; if there is none, prefer `MethodName_Scenario_ExpectedResult` (or the language-idiomatic equivalent).
- Use the project's test framework. Don't introduce a new one mid-cycle.
- Mock or stub external dependencies (network, clock, filesystem if the test isn't about the filesystem). Tests must be deterministic.
- Run the test and watch it fail. Red has two parts, and they can arrive at different moments:
    - **Test-first (the ordering).** You have written the test and *not* the implementation. If the test won't even compile — or, in a dynamic language, won't import — because the symbol under test doesn't exist yet, that already counts: per the Three Laws of TDD, *not compiling is failing*. Nothing but the test exists, which is the strongest test-first evidence there is.
    - **The right reason (the behavior).** Once the test does compile, the *assertion* is what fails — the behavior is missing — not a typo, an unrelated import error, or a broken fixture. A failure that isn't about the behavior means the test is broken, not red; fix it.
- If the project uses aiwf and the milestone is `tdd: required`, advance the AC's TDD phase to `red` — a live, mandatory step, run the moment the failing test is written and shown to fail, **before you touch the implementation**:

  ```bash
  aiwf promote M-NNN/AC-<N> --phase red
  ```

  This `"" → red` promote is the event that records "a failing test now exists," so it must fire live, as it happens — the `aiwf history` timeline is what shows the test came before the code. A freshly-added AC rests at the pre-cycle empty phase, so the transition is always available; never skip it, defer it, or back-stamp it later.
- When the project declares test-path globs, this promote is **gated** on exactly that test-first shape by the red/green diff-shape gate (below): it refuses unless *only* test-path files are dirty. That is why you promote `red` the moment the test is written — before any implementation, a compile stub included. See the gate section for the new-symbol / compile-stub case.

### GREEN — Make it pass with the minimum code

- Write the smallest code change that turns the failing test green.
- Don't add features the test doesn't require. If you find yourself thinking "while I'm here…", stop — that's the next cycle.
- Run the full test suite. Confirm the new test passes *and* nothing else broke.
- If the project uses aiwf and the milestone is `tdd: required`:

  ```bash
  aiwf promote M-NNN/AC-<N> --phase green
  ```

- When the project declares test-path globs, this promote is **gated** by the red/green diff-shape gate (below): it refuses unless an implementation (non-test) file is dirty. Write the implementation before promoting to green.

### REFACTOR — Clean up

- Remove duplication introduced by the green step.
- Improve names that became wrong as the code grew.
- Extract methods or types if shape demands it.
- When the refactor reshapes structure — a new type, boundary, or module — consult `wf-codebase-health` for the forces that should guide the shape (cohesion, coupling, single-source-of-truth).
- Run tests after every meaningful refactor. Stay green.
- If the project uses aiwf and the milestone is `tdd: required` and the refactor was non-trivial:

  ```bash
  aiwf promote M-NNN/AC-<N> --phase refactor
  ```

  This step is optional — `green → done` is legal under the FSM. Use it when the refactor pass meaningfully reshaped the code.

### The red/green diff-shape gate

When a project declares test-path globs (the `tdd.test_paths` config), the `--phase red` and `--phase green` promotes enforce red-first ordering mechanically by inspecting the working tree — no extra command, trailer, or flag to remember:

- **`--phase red`** wants *only* test-path files dirty. It refuses when any non-test (implementation) path is already dirty — code before test, naming the offending paths — and refuses when nothing is dirty at all (a red phase with no failing test). Write the failing test first.
- **`--phase green`** wants an implementation (non-test) path dirty — the code that turns the test green. It refuses when no such path is dirty.

The gate reads the current working-tree diff only; it keeps no red-time snapshot, and it excludes planning and documentation files from the inspected set. Both refusals are overridable with `--force --reason "<why>"`, a human-only act. A project that declares no test-path globs is unaffected — the gate is opt-in.

**New symbols and compile stubs.** In a typed language a test for a brand-new symbol can't compile until a minimal declaration exists — and that declaration is a non-test change the red gate rejects. There is no real conflict once you promote at the right moment: *the test failing to compile is the test-first red* (only the test is dirty), so run `--phase red` **then**, before adding the stub. Add the minimal stub afterward to watch the assertion fail for the right reason — you are already moving toward green, past the red gate. Reserve `--force --reason` for genuine exceptions, not this everyday path.

## Branch-coverage audit (HARD RULE — runs before declaring done)

Before declaring this cycle complete, you walk every reachable conditional branch in the diff and confirm an explicit test exercises each side. **Saying "every branch covered" without performing the audit is the failure mode this rule exists to prevent.**

This audit is **agent-performed** — a manual branch-walk, not a tool invocation. A project's mechanical coverage gate is typically **statement**-level: it records that a basic block *ran*, not which arm of an `if`/`switch` was taken. So "hard rule" here means *you must perform this walk*, not *a tool enforces it at branch granularity* — where the mechanical gate stops at statements, this manual walk is what supplies the branch-level assurance. Don't read "hard rule" as "something else will catch me if I skip it."

### How to audit

- Open each new or changed source file.
- Walk it line by line.
- For every `if`/`else`/`switch`/`case`/`catch`/`?:`/early-return/short-circuit, identify which test exercises each side.
- If a branch has no test, write one. **Defensive paths count** — if a guard, an exception catch, or a malformed-input handler ships, it gets a test.
- If a helper is private and the branch is hard to reach via the public API, expose it to tests using the language's friend-assembly or package-private mechanism (C# `internal` + `InternalsVisibleTo`, Rust `pub(crate)`, Python `_internal` + explicit import, Java/Go package-private). Then write a direct test.
- Genuinely unreachable branches (e.g., a defensive `null` check on a value the type system guarantees non-null) are documented where the project records such things. Include the reason.

### What "reachable" means

A branch is reachable if any caller of the function — direct or transitive — can produce inputs that select it. The compiler can't prove unreachability for most defensive code. **Default to reachable; require a written reason to call something unreachable.**

## Vacuity check (required invocation — runs right after the branch-coverage audit)

Branch coverage proves each line *ran*; it does not prove an assertion would *catch a bug* on that line. Immediately after the branch-coverage audit, **invoke `wf-vacuity`** on the unit just built — the invocation is required, not optional. The same agent wrote the implementation and its tests and is graded on them passing, so the suite is suspect by construction; `wf-vacuity` is the adversarial sufficiency check coverage can't give you.

- **Defer probe 1 to a mechanical mutation tool where one is wired up** — a `mutate-hunt` / gremlins workflow, Stryker, mutmut, PIT. It is the stronger signal; the manual mutation probe is the stop-gap for units, languages, or repos without one.
- **Scope is the unit just built**, never the whole tree (per the skill).
- **No shared-tree mutation during the probe.** This check typically runs while the AC's implementation is still staged or otherwise uncommitted in the same working tree. The mutation probe's revert must not touch git state: no `git stash`, no `git checkout`/`git restore` on the file under test — those can silently desync the index from what a pending commit is about to land. Capture the pre-mutation content directly (read the file, or `git show HEAD:<path>`) and write it back byte-for-byte; if `wf-vacuity` runs via a dispatched reviewer rather than inline, isolate that reviewer in its own worktree instead of the shared checkout.
- **The invocation is mandatory; the output is advisory.** A surviving mutant or a weak-assertion finding routes back into this cycle as a fresh RED → GREEN — strengthen the assertion, then confirm it goes red on the mutant — or, if you judge it out of scope, it is surfaced at the calling skill's commit gate for the human to weigh. It is *not* an automatic block: the assertion-shape probe is LLM-judged and cannot be a hard gate (a gating mutation-testing step is a separate, mechanical concern).

Skipping the invocation because "the tests look fine" is the failure mode this step exists to prevent — the same shape as declaring branch coverage without performing the audit.

## RECORD — record progress (after the evidence)

The AC is promoted to `met` only *after* the branch-coverage audit and the vacuity check above have run. `met` is the "this AC is done" judgment; it sits after the evidence that substantiates it, never before — a judgment recorded before its evidence is a vacuous gate.

- If the project uses aiwf and this cycle is driving a milestone AC:
    - Advance the AC's `tdd_phase` to `done`:

      ```bash
      aiwf promote M-NNN/AC-<N> --phase done
      ```

      Under `tdd: required`, the kernel's `acs-tdd-audit` refuses `met` while `tdd_phase` is not `done` — and **`--force` does not get you around it.** Force relaxes only the status/phase FSM *transition* check; the audit runs as a projection finding **regardless of `--force`**, so there is no `--force met` shortcut. `--force` itself is a **sovereign, human-only** act (the kernel refuses a non-human `--force` actor) — if you think an exception genuinely needs it, the honest lever is fixing the *phase* (or reconsidering the milestone's `tdd:` setting), not forcing the *status*; surface that to the human rather than reaching for `--force met` yourself.
    - Stop here — this cycle's job ends at `phase: done`. Promoting the AC to `met`, committing the implementation, and appending the Work log entry belong to the *calling* milestone ritual (e.g. `aiwfx-start-milestone` step 6), never to this cycle: the Work log's `commit <SHA>` citation needs the implementation commit's SHA, which does not exist until this cycle returns control and that commit lands. Doing `met` + Work log here, before that commit exists, is the exact "SHA doesn't exist yet" ordering bug the milestone-level commit-per-AC model exists to avoid.
    - The kernel records the phase + status timeline via `aiwf history M-NNN/AC-<N>` automatically — no need to duplicate dates and SHAs in the work log.
- If the project doesn't use aiwf:
    - Mark the acceptance criterion done in whatever the project uses to track AC progress (an issue, a checklist).
    - Note any decisions or deviations made mid-cycle.
- Note any decisions or deviations made mid-cycle (regardless of project framework).
- If the project has no AC-tracking habit, skip — don't invent one.

## Anti-patterns

- *Writing code before the test.* The test that comes after the code is verification, not specification.
- *Writing tests that can't fail.* If you can comment out the assertion and the test still passes, the test is broken.
- *Skipping the refactor step.* Green-then-rush is how the codebase rots.
- *Testing implementation details.* Tests should describe behavior; private internals are leverage points, not assertion targets.
- *Tests that depend on execution order.* Each test owns its state.
- *Declaring "every branch covered" without performing the audit.* See the hard rule above.

## Test-quality checklist

Before declaring done, every new test passes:

- Deterministic (no randomness, no real clock, no real network).
- Independent (no shared mutable state with other tests).
- Covers edge cases (null, empty, boundary values, error paths).
- Named so the reader knows what's being tested without reading the body.

## Constraints

- 🛑 **Branch-coverage audit is a hard rule.** It runs before the commit-approval prompt of whatever calling skill invoked this cycle, not after a human asks. If the calling skill (e.g., `wf-patch`) is about to ask "commit?", the audit must already be complete.
- 🛑 **`wf-vacuity` invocation is required; its output is advisory.** The vacuity check is invoked on the unit after the branch-coverage audit and before the cycle is declared done — the invocation is mandatory. Its findings inform (strengthen a weak assertion, or surface a survivor at the commit gate); they do not mechanically block.
- Tests must be deterministic. No flakes shipped.
- The cycle ends green. Never leave a branch with a red test you'll "fix in the next cycle."
