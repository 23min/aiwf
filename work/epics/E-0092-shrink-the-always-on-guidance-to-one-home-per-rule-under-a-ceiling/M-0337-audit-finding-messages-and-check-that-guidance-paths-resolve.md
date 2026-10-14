---
id: M-0337
title: Audit finding messages and check that guidance paths resolve
status: done
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
      status: met
      tdd_phase: done
    - id: AC-3
      title: Every guidance enforcement pointer resolves to a policy or finding code
      status: cancelled
    - id: AC-4
      title: Every repository path cited by development guidance exists
      status: met
      tdd_phase: done
    - id: AC-5
      title: The ceiling constant reaches its target
      status: met
---

## Goal

Audit the message of every policy and finding code the development guidance names, correct each one that does not say what to do, and check that every repository path the guidance cites exists; close G-0436.

## Closes

- G-0436 — stale `cmd/aiwf/` paths cited by guidance and `docs/design/id-allocation.md`; both citations it names now resolve, the second with the history functions it names corrected. AC-4's check keeps the development guidance's paths resolving; the design document is outside it.

## Context

The development-guidance set is the one M-0333 defines: both root entry points, `.guidance/project.md` and the on-demand documents it routes to. Generated operating and language blocks are judged through their owning sources; expected rendered copies are not independent rule restatements. E-0092 defines the primed/on-demand split and the two ceilings.

Much of the development guidance documents rules that a policy, a check rule, or a hook enforces. The check does the work; the prose spares the investigation a failed check would otherwise need, and the finding message spares it equally once it names the fix. The audit decides, per message, whether it already does, and the messages that do not are corrected.

## Acceptance criteria

### AC-1 — The audit table lists every policy and finding code a chokepointed section names

This body's audit table lists every policy id and finding code named in the development-guidance set at the milestone's start, each with its current message and a verdict: states the fix, or does not. **Pass criterion**: the inventory test (M-0349's design notes) derives the set of ids and codes from those guidance files as they stand at the milestone's start and asserts each has a row in this milestone's table, read through the loader; it passes. **Code references**: the policy-id literals under `internal/policies/`, the kernel's code set under `internal/check/`.

### AC-2 — Every message the audit marks as not stating the fix is corrected and pinned

Every message the table marks as not stating the fix is changed to state it, and a test pins the changed message. **Pass criterion**: each such message has a test asserting its remediation clause, listed in this body at wrap; no check's condition changes, only its text. **Edge cases**: a message shared by several sites is changed once at its source.

### AC-3 — Every guidance enforcement pointer resolves to a policy or finding code

Cancelled. No guidance section is cut to a pointer, so the check would guard only the pointers already written: the development guidance holds one of the form "enforced by `<id>`", and it names a policy that exists. AC-4's check covers the pointers written as paths. A pointer naming a policy id, a finding code or a Go function is not checked.

### AC-4 — Every repository path cited by development guidance exists

Every backticked repository-relative path cited in development guidance exists. **Pass criterion**: a relationship check reports a cited path absent from the tree; on the tree it reports none. **Edge cases**: a glob-shaped citation is matched as a glob; a placeholder in angle brackets is not a path; a path git ignores is not judged, since only a command such as `aiwf update` or a build creates it and a fresh checkout lacks it. **Code references**: a new policy under `internal/policies/`; G-0436's two stale citations are the fixture.

### AC-5 — The ceiling constant reaches its target

Both hosts' handwritten primed words, as defined by E-0092 and measured by M-0333's policy, pass at the ceiling M-0349 set for each host. Record each command and result, with the fragment's words reported separately. Personal/global words and task-loaded text are observed rather than measured here: M-0334 AC-3 records both at the baseline, and M-0338 AC-2 compares the task-loaded text after. A document required in full before any task counts as primed and cannot be excluded to meet the target.

## Constraints

- Message edits change text only; a condition change is a kernel change and out of scope.
- One row appended to the iteration log when this lands.

## Design notes

- E-0092 lets a check-backed section become a pointer once its check's message states the fix; with AC-2's corrections, every check in the audit table meets that precondition.

## Surfaces touched

- messages under `internal/policies/`, with the tests that pin them
- `fsm-invariants`, restructured so a test reaches each of its messages, and the firing-fixture grandfather list it leaves
- a new policy under `internal/policies/` for the paths development guidance cites, `projectguidance.HandwrittenLines`, which it reads the host files through, and its entry in the guidance-reader list
- `docs/design/id-allocation.md`'s citation of the history functions
- the inventory test, for AC-1's coverage of the audit table and AC-5's ceilings, and the audit test's entry in the guidance-reader list

## Out of scope

- Any change to what a check enforces.
- The fragment.
- Cutting a check-backed section to a pointer.

## Dependencies

- M-0335 — the routed guidance the audit and the path check read
- M-0349 — the ceiling AC-5 measures against

## Coverage notes

- (none)

## References

- G-0436
- G-0676 — how root guidance grows
- G-0726 — the go workflow's path filters, which leave out files the cited-path check requires to exist
- G-0444 — retired history function names; `docs/design/id-allocation.md:164` is corrected here, and the comments and test messages it records remain

## Audit

Each row is a policy or finding code the handwritten development guidance named at `65d846222`, the commit that started this milestone, by its id, by the policy file, or by its `Policy` function. A message states the fix when it names the action to take; one that only describes the defect does not, however plain the remedy. A finding code's message is its check's text with the hint `aiwf check` appends, and the hint is what names the action. Message quotes the remedy clause of a check that states the fix, and each message naming no action, at its source line, of one that does not. The quotes are as they stood at that commit.

| Check | Kind | Message | Verdict |
|---|---|---|---|
| `acs-tdd-audit` | finding | hint: "advance the AC's tdd_phase to `done` via `aiwf promote <id>/AC-N --phase done`, or relax the milestone's tdd: setting" | states the fix |
| `body-prose-id` | finding | each subcode's hint names a command, e.g. "fix it with `aiwf edit-body <id>` …" or "run `aiwf check` to settle it" | states the fix |
| `branch-coverage-audit` | policy | "Add a test that reaches it, or annotate the line `//coverage:ignore <reason>` if it is genuinely unreachable" | states the fix |
| `claim-guard-presence` | policy | `claim_guard_presence.go:100`: "… records claim scope %q, but no package-level function by that name was found under internal/verb — the row is either stale … or it names a shape this scan does not see, such as a method; either way this policy cannot vouch for it"; `:191`: "%s and %s record the same exemption reason — an exempt route carries its own specific reason, not a category label shared across routes …"; `:323`: "%s is not declared under internal/verb — every %s finding is phrased against that name, so this policy cannot answer anything until the anchor is repointed …" | does not |
| `coauthor-trailer-ban` | policy | "rewrite the message to drop the line, or remove the address from that list" | states the fix |
| `comment-history-attrition` | policy | "state it in the present tense as current truth about the input space; otherwise drop the clause" | states the fix |
| `entity-id-narrow-width` | finding | hint: "undo the change that produced it (`git checkout -- <path>` if uncommitted, `git revert <sha>` if it landed)" | states the fix |
| `firing-fixture-presence` | policy | "Add a firing test beside the policy that drives it to return >=1 violation; if it is a structure-auditor …, grandfather it in grandfatherDark" | states the fix |
| `fsm-invariants` | policy | `fsm_invariants.go:50`: "kind %q is in AllKinds() but has no AllowedStatuses; FSM unwired"; `:72`: "kind %q has no non-terminal statuses; FSM is unwired or every state is terminal"; `:88`: "kind %q: transition %q → %q targets a status not in AllowedStatuses"; `:117`: "kind %q at %q: CancelTarget %q not in AllowedStatuses"; `:124`: "… CancelTarget %q has outgoing transitions %v; must be terminal"; `:148`: "kind %q: FSM contains a cycle: %v"; `:189`: "composite FSM %q contains a cycle: %v" | does not |
| `git-test-env-harden` | policy | `git_test_env_harden.go:111`: "exec-bearing test package has no setup_test.go to host the testsupport.HardenGitTestEnv() call …"; `:126`: "test package shells out to a subprocess but its TestMain does not call testsupport.HardenGitTestEnv(); ambient git-locator env vars leak …" | does not |
| `guidance-readers` | policy | "list it with that reason; if it pins a phrase, state the claim as a relationship check or record it as an observation"; "delete the entry" | states the fix |
| `logging-chokepoint` | policy | "route through cliutil's text-output wrapper set … or allowlist the file with a rationale" | states the fix |
| `m0211-guidance-operating-anchors` | policy | `m0211_guidance_operating_anchors.go:46`: "shippable guidance source is unreadable — consumer-operating rules cannot ship from it: %v" | does not |
| `no-hardcoded-entity-paths` | policy | "Resolve the entity via tree.Load(ctx, root) + Tree.ByID + entity.Path instead" | states the fix |
| `race-parallel-cap` | policy | `race_parallel_cap.go:63`: "race-mode go test invocation must carry `-parallel 8` …"; `:71`: "expected a `go test ... -race ...` line; none found …" | does not |
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

Nothing a consumer sees changes: the policies whose messages changed, and the new path check, run only in this repository's test suite. In this repository, every policy and finding code the development guidance names says what to do when it fires, and a policy reports a repository path the development guidance cites that does not exist.

## Decisions made during implementation

- No guidance section is cut to a pointer. M-0349's inventory decides `pointer to` for no row, its test refuses a removal that does not carry its row's decision, and the primed load already meets both targets, so a cut would override recorded decisions with no target left to meet.
- AC-1's table counts a policy named by its file or `Policy` function as named, not only one named by its id; checks named only as tests are not policies or finding codes, and are left out.
- AC-4 reads a backticked span as a cited path when it has no whitespace or angle-bracket placeholder, holds a slash, and starts with an entry at the repository root; a bare file name, a path inside a command, and a path into a top-level directory the checkout does not hold are not judged. The root-entry anchor is what keeps `origin/main`, `github.com/…` and Go import paths out.
- AC-4's edge case for a path git ignores replaces checking it on disk: CI's test job creates none of the materialized host files or build outputs the guidance cites, so a check on disk fails there.
- The milestone runs `tdd: advisory`: AC-5's ceilings already met their targets when it started, so it has no red phase.
- `fsm-invariants` checks each kind through `kindFSMViolations` over a view of its status machine, so a test reaches every message over a synthetic machine, and the policy leaves the firing-fixture grandfather list, which held it as a structure-auditor no fixture could reach.

## Validation

Gates, run at `322887029`, the last code commit, in the aiwf devcontainer (Linux x86_64, go1.25.12):

- `make ci` exits 0: vet, lint, the race-and-coverage run at 92.1% of statements, the profile-driven gates `make coverage-gate-only` runs, and the 29-step self-check.
- `aiwf check --since origin/epic/E-0092-shrink-the-always-on-guidance-to-one-home-per-rule-under-a-ceiling` reports 0 errors; its one warning on this milestone is `acs-tdd-audit` on AC-5, which carries no TDD phase.

### AC-1 — the audit table

`TestM0337Audit` reads the handwritten development guidance at `65d846222`, failing if that commit cannot be read or its guidance names nothing, maps it through the policy files, `Policy` functions and finding codes at HEAD, and fails for any without a row under `## Audit`, or a row whose verdict is outside the closed set; `TestNamedChecks` and `TestAuditCoverageFindings` pin how a check is named and the table's rules.

### AC-2 — the pinned messages

Each message the audit marks as not stating the fix now names the action, and an assertion on the message's remedy clause fails without it:

- `claim-guard-presence`: a row naming no package-level function, two routes sharing an exemption reason, and a guard wrapper missing from `internal/verb` (`TestCheckClaimGuardPresence_RowTheScanCannotSeeFires`, `TestValidateExemptClaimReasons_SharedReasonFires`, `TestPolicyClaimGuardPresence_MissingAnchorFiresOnce`).
- `fsm-invariants`: each of the six ways a kind's status machine is miswired, and a cycle in a composite machine (`TestKindFSMViolations_MessagesStateTheFix`, over synthetic machines through `kindFSMViolations` and `fsmDAGViolations`); `TestKernelFSMView_ReadsTheEntityTables` holds the view the policy checks for each kind to `entity`'s exported API.
- `git-test-env-harden`: a package with no `setup_test.go`, and a `TestMain` that does not harden (`TestPolicyGitTestEnvHardened_Branches`).
- `m0211-guidance-operating-anchors`: an unreadable guidance source (`TestPolicyM0211GuidanceOperatingAnchors_UnreadableSourceStatesTheFix`).
- `race-parallel-cap`: a race-mode invocation without the cap, and a target with no race-mode invocation (`TestPolicyRaceParallelCap_MessagesStateTheFix`).
- `skill-coverage`: a missing `name:`, a missing `description:`, a name that differs from its directory, a name off the prefix convention, and an unresolved `aiwf` reference (`TestCheckSkillFrontmatter_FiresOnEmptyName`, `…_FiresOnEmptyDescription`, `…_FiresOnNameDirMismatch`, `…_FiresOnAiwfPrefixMissing`, `TestCheckSkillBodyMentionsResolve_FiresOnUnknownVerb`).
- `test-setup-presence`: a missing `setup_test.go`, one that does not parse, and one with no `TestMain` (`TestPolicyTestSetupPresence_MessagesStateTheFix`).

No check's condition changed.

### AC-4 — cited paths

`TestPolicy_GuidancePathsResolve` holds the live guidance to `guidance-paths-resolve`; `TestCitedPath`, `TestPathResolves` and `TestPolicyGuidancePathsResolve_ReportsStalePaths` pin which spans are paths, how globs and brace groups resolve, that G-0436's two stale citations are reported at the line a reader finds them on below a managed block, and that a path git ignores, as a file or under a pattern naming only directories, is not judged.

### AC-5 — the ceilings

Command: `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/`, run at `ad4ba93ed` in the aiwf devcontainer (Linux x86_64, go1.25.12). Expected: each host's handwritten primed load at or under the target M-0349 set, which `go test -count=1 -run 'TestE0092Inventory$' -v ./internal/policies/` logs from the inventory.

| Host | Handwritten primed | Ceiling constant | M-0349 target | aiwf-generated, reported separately |
|---|---|---|---|---|
| Claude Code | 1,275 | 1,275 | 1,629 | 2,460 |
| Codex | 1,415 | 1,415 | 1,758 | 2,497 |

Observed: the test passes, and each ceiling constant sits below its host's target. `TestE0092Inventory` holds each ceiling constant at or under the target it derives from the inventory, so the load, which the ceiling policy holds at or under the constant, stays at or under the target until E-0092's wrap deletes the inventory test. Codex's handwritten load includes `CLAUDE.md`, which its preamble requires in full before any task; the test lists it among Codex's required reads. The aiwf-generated column counts the managed blocks in each entry point, with the shipped fragment Claude Code imports or Codex carries inline, and `.guidance/index.md`, which aiwf generates.

## Deferrals

- (none)

## Reviewer notes

What this milestone obliges later changes to do, counted from `git diff 090ea06b3..HEAD -- internal/` (new `Test` functions and `Policy:` ids) and `scripts/growth-report.py`'s policy-id count:

- `guidance-paths-resolve`: a change that moves or deletes a file the development guidance cites fixes the citation in the same change. A ban, owned by the policy suite, and nothing retires it; G-0726 records that CI's path filters can skip it.
- The remedy-clause assertions on the seven audited policies: rewording a remedy means editing its test. Owned by each policy's tests; retired with the message.
- `fsm-invariants` under the firing-fixture gate, with `TestKernelFSMView_ReadsTheEntityTables` holding its view to `entity`'s exported API: a new drift mode needs a synthetic fixture. Owned by `firing-fixture-presence`; retired with the policy.
- `TestHandwrittenLines` holds the line-keeping reader to `HandwrittenText`'s words. Owned by `projectguidance`; retired with the helper.
- `TestM0337Audit` and its rule tests keep the Audit table readable, need `65d846222` reachable, and need a row for any policy id later added to a policy file the guidance named, since they map that guidance through HEAD's policy files. `ceilingTargetFindings` keeps each ceiling constant at or under M-0349's target. Owned by E-0092; deleted with `e0092_inventory_test.go` at E-0092's wrap, with the audit test's guidance-reader entry, after which nothing holds the ceiling constants to the targets.

Accepted as they stand:

- Deleting `PolicyFSMInvariants`' per-kind loop fails no test: over correct tables the policy reports nothing, so a skipped check and a clean one look alike.
- With git unable to start, `guidance-paths-resolve` reports a missing path rather than an error; both fail the suite, and pinning it needs a test that changes `PATH`.
- `git check-ignore` also honours a developer's personal ignore files, so a local run can be more lenient than CI.
- `**` matches one directory level, `path:line` and `path#anchor` citations are read as paths, and a brace citation is asked of `git check-ignore` as one string; no citation uses any of these, and each would be reported rather than passed.
- `guidance-paths-resolve` and `devcontainer-paths-resolve` both report a stale backticked `.devcontainer/` path in the development guidance; each covers citations the other does not, so both stay.
- `gitIgnores` resembles inline code in `internal/cli/render/render.go`; with two copies there is no shared helper.
- Remedy assertions pin each clause's leading phrase, not every word of it.
- `fsmView.isAllowed` stays, so the restructure reads as a one-for-one substitution of `entity` calls.
- Removing the `continue` after an undeclared cancel target adds a second, still-true finding rather than a wrong one.
- The changed messages name files under `internal/entity/` and `internal/policies/` that nothing re-checks, and `test-setup-presence`'s messages state the `TestMain` recipe rather than point at `.guidance/project/testing.md`.
- E-0092's success criterion that enforcement pointers resolve is held by a check only for pointers written as paths; it is judged at E-0092's wrap.

Verdict: the review ended on a scoped confirmation rather than a clean full pass. The last full pass over the change set requested changes; a fresh reviewer then confirmed everything changed after it. The corrections that confirmation asked for, at `322887029` and in G-0444's body, were not re-read by a reviewer.
