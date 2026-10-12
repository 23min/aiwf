---
id: M-0337
title: Audit finding messages and check that guidance paths resolve
status: in_progress
parent: E-0092
depends_on:
    - M-0335
tdd: advisory
acs:
    - id: AC-1
      title: The audit table lists every policy and finding code a chokepointed section names
      status: met
      tdd_phase: done
    - id: AC-2
      title: Every message the audit marks as not stating the fix is corrected and pinned
      status: open
      tdd_phase: done
    - id: AC-3
      title: Every guidance enforcement pointer resolves to a policy or finding code
      status: cancelled
    - id: AC-4
      title: Every repository path cited by development guidance exists
      status: open
    - id: AC-5
      title: The ceiling constant reaches its target
      status: open
---

## Goal

Audit the message of every policy and finding code the development guidance names, correct each one that does not say what to do, and check that every repository path the guidance cites exists; close G-0436.

## Closes

- G-0436 — stale `cmd/aiwf/` paths cited by guidance; both citations it names read correctly in the tree, and AC-4 is the check that keeps development guidance's paths resolving.

## Context

The development-guidance set is the one M-0333 defines: both root entry points, `.guidance/project.md` and the on-demand documents it routes to. Generated operating and language blocks are judged through their owning sources; expected rendered copies are not independent rule restatements. E-0092 defines the primed/on-demand split and the two ceilings.

Much of the development guidance documents rules that a policy, a check rule, or a hook enforces. The check does the work; the prose spares one failed-check round trip, which the finding message spares equally once it names the fix. The audit decides, per message, whether it already does, and the messages that do not are corrected.

## Acceptance criteria

### AC-1 — The audit table lists every policy and finding code a chokepointed section names

This body's audit table lists every policy id and finding code named in the development-guidance set at the milestone's start, each with its current message and a verdict: states the fix, or does not. **Pass criterion**: the inventory test (M-0349's design notes) derives the set of ids and codes from those guidance files as they stand at the milestone's start and asserts each has a row in this milestone's table, read through the loader; it passes. **Code references**: the policy-id literals under `internal/policies/`, the kernel's code set under `internal/check/`.

### AC-2 — Every message the audit marks as not stating the fix is corrected and pinned

Every message the table marks as not stating the fix is changed to state it, and a test pins the changed message. **Pass criterion**: each such message has a test asserting its remediation clause, listed in this body at wrap; no check's condition changes, only its text. **Edge cases**: a message shared by several sites is changed once at its source.

### AC-3 — Every guidance enforcement pointer resolves to a policy or finding code

Cancelled. No guidance section is cut to a pointer, so the check would guard only the pointers already written: the development guidance holds one of the form "enforced by `<id>`", and it names a policy that exists. AC-4's check covers the pointers written as paths. A pointer naming a policy id, a finding code or a Go function is not checked.

### AC-4 — Every repository path cited by development guidance exists

Every backticked repository-relative path cited in development guidance exists. **Pass criterion**: a relationship check reports a cited path absent from the tree; on the tree it reports none. **Edge cases**: a glob-shaped citation is matched as a glob; a placeholder in angle brackets is not a path; a path under a gitignored directory is checked on disk. **Code references**: a new policy under `internal/policies/`; G-0436's two stale citations are the fixture.

### AC-5 — The ceiling constant reaches its target

Both hosts' handwritten primed words, as defined by E-0092 and measured by M-0333's policy, pass at the ceiling M-0349 set for each host. Record each command and result, with the fragment's words reported separately. Personal/global words and task-loaded text are observed rather than measured here: M-0334 AC-3 records both at the baseline, and M-0338 AC-2 compares the task-loaded text after. A document required in full before any task counts as primed and cannot be excluded to meet the target.

## Constraints

- Message edits change text only; a condition change is a kernel change and out of scope.
- One row appended to the iteration log when this lands.

## Design notes

- The audit table's verdicts are what a later cut of a check-backed section starts from: a section can become a pointer once its message states the fix, which E-0092 sets as the precondition.

## Surfaces touched

- messages under `internal/policies/` and `internal/check/`, with the tests that pin them
- a new policy under `internal/policies/` for the paths development guidance cites
- the inventory test, for AC-1's coverage of the audit table

## Out of scope

- Any change to what a check enforces.
- The fragment.
- Cutting a check-backed section to a pointer: M-0349's inventory decides `pointer to` for no row, and its test refuses a removal that does not carry its row's decision.

## Dependencies

- M-0335 — the routed guidance the audit and the path check read
- M-0349 — the ceiling AC-5 measures against

## Coverage notes

- (none)

## References

- G-0436
- G-0676 — how root guidance grows

## Audit

Each row is a policy or finding code the handwritten development guidance named at `65d846222`, the commit that started this milestone, by its id, by the policy file, or by its `Policy` function. A message states the fix when it names the action to take; one that only describes the defect does not, however plain the remedy. A finding code's message is its check's text with the hint `aiwf check` appends, and the hint is what names the action. Message quotes the remedy clause of a check that states the fix, and each message naming no action, at its source line, of one that does not. The quotes are as they stood at that commit.

| Check | Kind | Message | Verdict |
|---|---|---|---|
| `acs-tdd-audit` | finding | hint: "advance the AC's tdd_phase to `done` via `aiwf promote <id>/AC-N --phase done`, or relax the milestone's tdd: setting" | states the fix |
| `body-prose-id` | finding | hint, per subcode: "fix it with `aiwf edit-body <id>` …" | states the fix |
| `branch-coverage-audit` | policy | "Add a test that reaches it, or annotate the line `//coverage:ignore <reason>` if it is genuinely unreachable" | states the fix |
| `claim-guard-presence` | policy | `claim_guard_presence.go:100`: "… records claim scope %q, but no package-level function by that name was found under internal/verb — the row is either stale … or it names a shape this scan does not see, such as a method; either way this policy cannot vouch for it"; `:191`: "%s and %s record the same exemption reason — an exempt route carries its own specific reason, not a category label shared across routes …" | does not |
| `coauthor-trailer-ban` | policy | "rewrite the message to drop the line, or remove the address from that list" | states the fix |
| `comment-history-attrition` | policy | "state it in the present tense as current truth about the input space; otherwise drop the clause" | states the fix |
| `entity-id-narrow-width` | finding | hint: "undo the change that produced it (`git checkout -- <path>` if uncommitted, `git revert <sha>` if it landed)" | states the fix |
| `firing-fixture-presence` | policy | "Add a firing test beside the policy that drives it to return >=1 violation; if it is a structure-auditor …, grandfather it in grandfatherDark" | states the fix |
| `fsm-invariants` | policy | `fsm_invariants.go:50`: "kind %q is in AllKinds() but has no AllowedStatuses; FSM unwired"; `:72`: "kind %q has no non-terminal statuses; FSM is unwired or every state is terminal"; `:88`: "kind %q: transition %q → %q targets a status not in AllowedStatuses"; `:117`: "kind %q at %q: CancelTarget %q not in AllowedStatuses"; `:124`: "… CancelTarget %q has outgoing transitions %v; must be terminal"; `:148`: "kind %q: FSM contains a cycle: %v"; `:189`: "composite FSM %q contains a cycle: %v" | does not |
| `git-test-env-harden` | policy | `git_test_env_harden.go:111`: "exec-bearing test package has no setup_test.go to host the testsupport.HardenGitTestEnv() call"; `:126`: "test package shells out to a subprocess but its TestMain does not call testsupport.HardenGitTestEnv(); ambient git-locator env vars leak …" | does not |
| `guidance-readers` | policy | "list it with that reason; if it pins a phrase, state the claim as a relationship check or record it as an observation"; "delete the entry" | states the fix |
| `logging-chokepoint` | policy | "route through cliutil's text-output wrapper set … or allowlist the file with a rationale" | states the fix |
| `m0211-guidance-operating-anchors` | policy | `m0211_guidance_operating_anchors.go:46`: "shippable guidance source is unreadable — consumer-operating rules cannot ship from it: %v" | does not |
| `no-hardcoded-entity-paths` | policy | "Resolve the entity via tree.Load(ctx, root) + Tree.ByID + entity.Path instead" | states the fix |
| `race-parallel-cap` | policy | `race_parallel_cap.go:71`: "expected a `go test ... -race ...` line; none found" | does not |
| `shipped-prose-assertion` | policy | "D-0070 retires this class: delete the assertion" | states the fix |
| `skill-body-id` | finding | hint: "write the canonical `<prefix>-NNNN` placeholder or a shape-description" | states the fix |
| `skill-coverage` | policy | `skill_coverage.go:120`: "skill is missing a `name:` frontmatter field"; `:126`: "skill `name: %s` does not match its directory %q"; `:132`: "skill name %q does not match the `%s<topic>` convention"; `:139`: "skill is missing a `description:` frontmatter field — the host's match-scoring depends on it"; `:232`: "skill body references `aiwf %s` but %q does not resolve to a registered command path …" | does not |
| `skill-edit-provenance-backstop` | policy | "re-commit naming the epic, milestone, gap or decision that owns it"; "name an entity that exists or create one first" | states the fix |
| `stress-lane-census` | policy | "Move it to an untagged sibling file, or, if it drives real work through a helper, name that helper in slowWorkIdents" | states the fix |
| `test-executable-write` | policy | "Use testsupport.WriteExecutable. When the mode itself is the subject, annotate with //exec:ok <reason>" | states the fix |
| `test-setup-presence` | policy | `test_setup_presence.go:80`: "test-bearing package missing setup_test.go (see .guidance/project/testing.md *Test discipline*; …)"; `:97`: "setup_test.go failed to parse: " followed by the parser's error; `:106`: "setup_test.go missing top-level `func TestMain(m *testing.M)` declaration (see .guidance/project/testing.md *Test discipline*)" | does not |
| `trailer-verb-unknown` | finding | hint: "`git commit --amend` and drop the trailer … if it's a new ritual verb, add it to the ritualVerbs allowlist" | states the fix |
| `verb-result-noop-invariant` | policy | "repoint it at the verbs' new location"; "add a same-state test asserting the verb converges to a NoOp, or add an allowlist entry" | states the fix |

## Release note

## Decisions made during implementation

- No guidance section is cut to a pointer. M-0349's inventory decides `pointer to` for no row, its test refuses a removal that does not carry its row's decision, and the primed load already meets both targets, so a cut would override recorded decisions to save no primed words.
- AC-3 is cancelled, for the reason under its heading.
- The milestone runs `tdd: advisory`: AC-5 is met by the ceiling test, which already passes, so it has no red phase; AC-1, AC-2 and AC-4 record their phases as their tests are written.

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
