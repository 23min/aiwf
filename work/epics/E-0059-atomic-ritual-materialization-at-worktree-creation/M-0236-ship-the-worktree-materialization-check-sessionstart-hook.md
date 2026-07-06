---
id: M-0236
title: Ship the worktree-materialization-check SessionStart hook
status: in_progress
parent: E-0059
depends_on:
    - M-0235
tdd: required
acs:
    - id: AC-1
      title: Hook flags unmaterialized worktree rituals, nonzero exit with stderr
      status: met
      tdd_phase: done
    - id: AC-2
      title: Hook registered in the registry for both SessionStart and SubagentStart events
      status: met
      tdd_phase: done
    - id: AC-3
      title: Subprocess policy test pins exit code and stderr for both hook cases
      status: met
      tdd_phase: done
    - id: AC-4
      title: init/update materialize and wire the hook per consent; doctor reports its state
      status: met
      tdd_phase: done
---

## Goal

Ship the one concrete hook this epic exists to add: a `SessionStart` /
`SubagentStart` script that detects a session or subagent starting with cwd
inside a `.claude/worktrees/` checkout whose rituals aren't materialized,
and warns visibly without blocking — registered against the hook registry
the prior milestone builds.

## Context

M-0235 lands the generalized hook registry: the `aiwf.yaml` `hooks:` consent
schema, the settings.json writer, and the materialization category. This
milestone adds the first (and, for now, only) entry in that registry.

Spike finding (resolved this session, against the official Claude Code
hooks documentation): `SessionStart` and `SubagentStart` cannot block or
abort — their only user-visible channel is a nonzero exit code, whose
stderr renders as a harness-level "hook error notice" directly to the
human, without the model mediating it, while the session or subagent
proceeds regardless. That is exactly the advisory, never-hard-refuse shape
this epic's own risk mitigation calls for, and it sidesteps relying on the
LLM to relay a silently-injected `additionalContext` string — the harness
renders the notice itself.

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac M-0236 --title "..."`.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — The hook script checks whether cwd is under
  `.claude/worktrees/`; if so, it checks ritual materialization (reusing
  `aiwf doctor`'s existing rituals-presence check rather than
  reimplementing it) and exits nonzero with a clear, actionable stderr
  message when rituals are missing or stale. It exits 0 silently for the
  main checkout, or a worktree whose rituals are fully materialized.
- **AC-2 candidate** — The script is registered in the hook registry under
  both `SessionStart` and `SubagentStart`, so it fires for an interactively
  started session and for a dispatched subagent alike.
- **AC-3 candidate** — Contract pinned by a subprocess-level policy test
  mirroring `TestAgentIsolationHook_*` (`internal/policies/`), asserting
  exit code and stderr content for both the stale/missing case and the
  healthy case.
- **AC-4 candidate** — `aiwf init` / `aiwf update` materialize the script and
  wire both settings-json event arrays once the operator has consented via
  M-0235's registry mechanism; `aiwf doctor` reports the hook's
  materialized/wired state.

### AC-1 — Hook flags unmaterialized worktree rituals, nonzero exit with stderr

### AC-2 — Hook registered in the registry for both SessionStart and SubagentStart events

### AC-3 — Subprocess policy test pins exit code and stderr for both hook cases

### AC-4 — init/update materialize and wire the hook per consent; doctor reports its state

## Constraints

- Detection is harness-executed only — no skill-instruction fallback; the
  whole point is removing the LLM-memory dependency.
- Advisory only — never hard-refuses, per the epic's own risk mitigation
  against false positives on an intentionally bare worktree.
- Ships complete, standalone `--help` documentation for this concrete
  hook: replace M-0235's placeholder `aiwf init --enable-hook <hook-name>`
  / `aiwf update --enable-hook <hook-name>` Example lines with this hook's
  real name, and state plainly what enabling it does. `--help` is the
  shippable discovery channel for `init`/`update` (both allowlisted
  no-skill ops verbs per ADR-0006) — no CLAUDE.md mention needed, and no
  reference to any sibling consent mechanism (ADR-0015/ADR-0018).
- The settings.json command this milestone wires via `WireHookSettings`
  must be exactly `<Target.HooksDir>/<hook-name>` (e.g.
  `.claude/hooks/<hook-name>`) — `aiwf doctor`'s hook-drift check
  (`skills.HookDrift`, M-0235/AC-5) detects "wired" by matching that exact
  string against every command in `settings.json`'s `hooks:` key. A
  different command shape (an env-var prefix, an absolute path) silently
  breaks the drift report without erroring — `HookDrift`'s derivation must
  be revisited if this milestone needs a different convention.
- `aiwf doctor`'s "wired-but-stale" hook report (`skills.HookDrift`,
  M-0235/AC-5) checks script presence only (`os.Stat`), not content — it
  cannot detect an on-disk script whose bytes no longer match the shipped
  `HookDef.Content`. Confirm presence-only is the intended scope for this
  hook, or extend `HookDrift` with a content comparison if staleness needs
  to mean "outdated bytes," not just "decision drift."

## Out of scope

- The hook registry mechanism itself, `aiwf.yaml`'s `hooks:` schema, and the
  settings.json writer — M-0235.
- Migrating `.claude/hooks/validate-agent-isolation.sh` (G-0099) into the
  registry — a follow-up gap.

## Dependencies

- M-0235 (hook registry) — this milestone registers its one concrete hook
  against that infrastructure and cannot start before it lands.

## References

- G-0374 — the gap this epic closes.
- G-0099 — the sibling isolation-guard hook; its migration into this
  registry is a follow-up gap, not this milestone's work.

## Work log

### AC-1 — Hook flags unmaterialized worktree rituals, nonzero exit with stderr

Added `aiwf doctor --check-rituals [--root <path>]`: a terse,
exit-code-meaningful check reusing `skills.MaterializedRituals` rather
than reimplementing materialization detection — silent + exit 0 when
every ritual artifact is present, a single actionable stderr line +
exit 1 otherwise. The plain `aiwf doctor` report keeps rituals
advisory-only (unchanged), since `--check-rituals` is a distinct mode
built for automation, not a change to the default report's exit-code
contract.

Added the hook script itself (`internal/skills/embedded-hooks/
worktree-rituals-check.sh`, go:embed'd as `skills.WorktreeRitualsCheckScript`):
gates on cwd being inside a `.claude/worktrees/` checkout, resolves the
worktree's own root via `git rev-parse --show-toplevel` (robust to
branch names containing `/`, unlike string-parsing the path), and
`exec`s into `aiwf doctor --check-rituals --root <root>` rather than
reimplementing detection in shell. Not yet registered into
`skills.ShippedHooks` or wired to any settings.json event — that's
AC-2's job.

`wf-vacuity` (6 mutations) found one real surviving mutant: the
script-gate test's `strings.Contains(script, ".claude/worktrees/")`
assertion matched the file's own header comment, not the functional
`case` line, so a broken gate pattern went undetected. Strengthened to
match the actual case-pattern token (`*/.claude/worktrees/*)`); the
same mutation now fails as expected. 0 surviving mutants after the fix.

Branch-coverage audit: both directions of `checkRitualsResult`'s
ok/not-ok split, both of `RunCheckRituals`'s exit-code arms (Go-level
and via the actual Cobra `--check-rituals` flag wiring), and the
script's own gate/delegation assertions are each covered by a
dedicated test. The two defensive error arms
(`cliutil.ResolveRoot`'s error path, `skills.MaterializedRituals`'s
error path) are unreachable at runtime — `//coverage:ignore`d, mirroring
the identical pattern already used elsewhere in this file.

commit `05d50627` · tests 8/8

### AC-2 — Hook registered in the registry for both SessionStart and SubagentStart events

Added `HookDef.Events []string` (the settings.json event arrays a hook
wires into once enabled — read by a future `WireHookSettings` caller,
AC-4's job, rather than each call site hardcoding the event list) and
populated `skills.ShippedHooks` with the one entry AC-1 shipped the
script for, registered under both `SessionStart` and `SubagentStart`.

This flips three tests that had explicitly pinned "the registry is
empty until M-0236" as their expected state (per their own doc
comments naming this milestone as the trigger) — updated to assert the
real registry contents instead of rewriting around them.

`wf-vacuity` (2 mutations: dropping `SubagentStart` from Events;
appending a bogus second registry entry) — both caught, 0 surviving
mutants. No new conditional branches in this diff (a struct field plus
a registry literal) — nothing for the branch-coverage audit to walk.

commit `f92c261c` · tests 3/3

### AC-3 — Subprocess policy test pins exit code and stderr for both hook cases

Added `internal/policies/worktree_rituals_check_hook_test.go`, mirroring
`TestAgentIsolationHook_*`'s subprocess-level shape: builds the real
`aiwf` binary once (`sync.Once`, `os.MkdirTemp` rather than
`t.TempDir()` since the fixture must outlive whichever test builds it
first), writes the embedded hook script's actual bytes
(`skills.WorktreeRitualsCheckScript`) to an executable file, and execs
it with `Dir` set to a real git-repo fixture. Covers three cases:
not-a-worktree (exit 0, silent — the primary, most-common invocation),
healthy worktree (fully materialized via `initrepo.Init`, exit 0,
silent), and stale/missing worktree (bare git repo, no materialization
run, nonzero exit + actionable stderr).

`wf-vacuity` found one real surviving mutant: an invalid
`git rev-parse --show-toplevel` flag doesn't error (git silently
echoes the bogus flag back as a relative-path string rather than
failing), so `cliutil.ResolveRoot` resolves it relative to cwd —
landing a nonzero exit and an "aiwf update" message from the WRONG
root, one that happens to be prefixed by the real worktree path. The
stale-case assertion couldn't tell "correctly detected missing
rituals" from "resolved a garbage nested path that also reports 0
artifacts present." Strengthened to pin the message's root segment
exactly via the format string's own `" —"` delimiter
(`under <resolved-path> —`), which only the correct root satisfies.
Re-verified: real script passes, mutation now caught, implementation
confirmed byte-identical after revert. 0 surviving mutants after the
fix.

A synthetic fixture path segment (`M-9999-test`) initially tripped
this repo's own `no-hardcoded-entity-paths` policy (fires on any
literal matching the `M-\d+-` entity-slug shape regardless of whether
it names a real entity) — renamed to `milestone-fixture-branch` to
avoid the false collision.

commit `30e8d5b3` · tests 3/3

### AC-4 — init/update materialize and wire the hook per consent; doctor reports its state

M-0235 built the primitives (`MaterializeHooks`, `WireHookSettings`,
`HookDrift`) but never called them from `aiwf init`/`update`'s actual
`Run` flow — confirmed by grep before writing any code: a consented
hook's decision landed in aiwf.yaml with no script on disk and nothing
wired into settings.json. Added `cliutil.SyncHookMaterialization`, a
shared call site both verbs invoke right after their own
consent-gating step: reads decisions back fresh from aiwf.yaml
(avoiding a signature change to the existing gate functions), then
materializes each hook's script and wires or unwires its settings.json
entries per its current decision. `aiwf doctor`'s existing
`appendHookMaterializationReport` (M-0235) needed no changes — it
already reports real state once init/update actually produce one;
confirmed via manual end-to-end smoke test (`aiwf init --enable-hook`
→ `aiwf doctor` → `hooks: ok (1 hooks synced)`, script present and
executable, settings.json wired under both events).

Also added `skills.UnwireHookSettings` (with `removeCommandFromEntries`)
— `WireHookSettings` was append-only with no removal counterpart, so
ADR-0032's "remove both when false" promise had no settings-side
implementation. This was surfaced as a decision point before coding
(user chose to close the gap now rather than defer it as a gap, since
`HookDrift`'s "wired-but-stale" remedy message would otherwise be false
advertising the first time a hook is actually flipped from enabled to
declined) — resolved by adding the primitive and wiring it into
`SyncHookMaterialization`'s declined-hook branch alongside
`MaterializeHooks`'s existing script-removal half.

`wf-vacuity` (4 mutations across the wiring: swap the enabled/declined
branch; drop `removeCommandFromEntries`'s empty-group check; drop the
`SyncHookMaterialization` call from `initcmd.go`; drop it from
`update.go`) found 2 real surviving mutants:
- The empty-group-check drop wasn't caught because the existing test
  only asserted the flattened command list was empty, not that the
  emptied matcher-group object itself was removed from the raw JSON —
  a lingering empty group reports zero commands too. Strengthened with
  a raw matcher-group-count assertion.
- Dropping `update.go`'s wiring call still passed
  `TestRun_HookRemovedWhenFlippedFromEnabledToDeclined` because the
  priming (enable) call's own effect was never verified before
  asserting the decline call removed it — with the wiring gone,
  nothing was ever created, so "removed" held vacuously. Strengthened
  to assert the primed state exists before flipping to declined.
Both re-verified: real code passes, mutation now caught, implementation
byte-identical after revert. 0 surviving mutants after the fixes; the
other 2 mutations (branch-swap, initcmd.go call-drop) were caught
cleanly on the first pass.

Branch-coverage audit: `UnwireHookSettings`'s not-wired/missing-file/
malformed-JSON/malformed-hooks-key/read-error arms, `removeCommandFromEntries`'s
keep-whole/drop-whole/partial-within-one-group arms, and
`SyncHookMaterialization`'s empty-registry/undecided/enabled/declined/
each-of-its-three-callee-errors arms are each covered by a dedicated
test (mirroring `WireHookSettings`'s and `gateAndSyncHookDecisions`'s
own precedent fixture shapes where the error is genuinely reachable).
The four defensive backup/marshal-error arms in `UnwireHookSettings`
are `//coverage:ignore`d, mirroring `WireHookSettings`'s identical,
equally-untestable-without-a-disk-fault shape.

commit `8a7b81fd` · tests 21/21
