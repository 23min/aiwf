---
id: M-0233
title: 'aiwf worktree add verb: atomic creation with ritual materialization'
status: done
parent: E-0059
tdd: required
acs:
    - id: AC-1
      title: aiwf worktree add creates worktree + materializes rituals atomically
      status: met
      tdd_phase: done
    - id: AC-2
      title: Explicit path honored verbatim; default resolves via worktree.dir
      status: met
      tdd_phase: done
    - id: AC-3
      title: Repo-escape rejection applies only to default path, not explicit path
      status: met
      tdd_phase: done
    - id: AC-4
      title: --print-path emits only the absolute path on success, nothing on failure
      status: met
      tdd_phase: done
    - id: AC-5
      title: git worktree add failures surface directly; never reports false success
      status: met
      tdd_phase: done
    - id: AC-6
      title: Flag completion and --help wired per completion-drift chokepoint
      status: met
      tdd_phase: done
---

## Goal

Add `aiwf worktree add`, a Cobra verb that performs `git worktree add` and `aiwf
init`/`aiwf update` materialization as a single atomic step, so a worktree created
through it always starts with `.claude/skills/`, `.claude/agents/`,
`.claude/templates/`, and `.claude/aiwf-guidance.md` already present.

## Context

G-0374 found that `git worktree add` never checks out aiwf's gitignored,
materialize-on-demand artifacts (ADR-0018), and nothing automates the follow-up
`aiwf init`/`update` step. This is the foundation milestone for E-0059: it lands
the verb itself, independent of rewiring any call site (M-0234) or the detection
backstop (M-0235). Builds on the existing `worktree.dir` config knob and
`config.WorktreeDir()` getter (M-0189/M-0190, E-0046).

## Acceptance criteria

### AC-1 — aiwf worktree add creates worktree + materializes rituals atomically

`aiwf worktree add <branch> [path]` creates a git worktree and materializes
rituals into it in one command; `aiwf doctor` run immediately after reports
`rituals: ok` with no intervening `aiwf update`.

### AC-2 — Explicit path honored verbatim; default resolves via worktree.dir

An explicit target path argument is honored verbatim (sibling directory, any
custom location); omitting it resolves to `<worktree.dir>/<branch-slug>` via
the existing `config.WorktreeDir()`.

### AC-3 — Repo-escape rejection applies only to default path, not explicit path

`worktree.dir`'s repo-escape rejection (M-0190/AC-4) applies only when
resolving the *default* path; an explicit caller-supplied path is never
subject to it, even one that points outside the repo.

### AC-4 — --print-path emits only the absolute path on success, nothing on failure

A `--print-path` flag prints only the resulting absolute path to stdout on
success (nothing else) and nothing to stdout on failure (nonzero exit) —
verified by a binary-level subprocess test that runs `cd "$(aiwf worktree add
... --print-path)" && pwd` in a real subshell, not just a Go-level
string-return unit test.

### AC-5 — git worktree add failures surface directly; never reports false success

A `git worktree add` failure (branch already checked out elsewhere, path
already exists, etc.) surfaces the underlying git error directly; the verb
never reports success on a failed creation.

### AC-6 — Flag completion and --help wired per completion-drift chokepoint

Flag completion and `--help` text are wired per the completion-drift
chokepoint (`cmd/aiwf/completion_drift_test.go`).

## Constraints

- Must not silently swallow `git worktree add` failures.
- `--print-path` output is composition-critical — stdout hygiene is tested at the
  binary/subprocess level, not just as a Go string-return unit test (per this
  repo's "test the seam, not just the layer" convention).
- No new entity kind or schema change; this is verb-only work within the existing
  kernel model.

## Out of scope

- Rewiring aiwf's own rituals or CLAUDE.md to call this verb (M-0234).
- The session-start detection backstop (M-0235).

## Dependencies

- M-0189's `worktree.dir` config knob and `config.WorktreeDir()` getter (already
  shipped, E-0046) — this milestone's default-path resolution builds on it.
- No prior milestone within this epic — this is the first.

## References

- G-0374 — the gap this epic (and this milestone) closes.
- ADR-0018 — materialize-on-demand model.
- ADR-0023 / E-0046 — in-repo worktree placement default; M-0189/M-0190 — the
  config knob and escape-rejection this milestone reuses and constrains.

## Work log

### AC-1 — aiwf worktree add creates worktree + materializes rituals atomically

Verb lands with `gitops.WorktreeAdd`/`WorktreeAddNewBranch` + an in-process
call to `initrepo.RefreshArtifacts` (the same pipeline `aiwf update` runs) ·
commit 4f577230 · tests 1/1

### AC-2 — Explicit path honored verbatim; default resolves via worktree.dir

Default-path branch routes through `config.WorktreeDir()`; explicit-path
branch never calls it · commit 4f577230 · tests 2/2

### AC-3 — Repo-escape rejection applies only to default path, not explicit path

Falls out of AC-2's branching directly — no additional code path, covered by
a dedicated test asserting a repo-escaping explicit path is honored ·
commit 4f577230 · tests 1/1

### AC-4 — --print-path emits only the absolute path on success, nothing on failure

Every error path in `Run` writes to stderr only; `--print-path` short-circuits
the success path before any ledger/JSON output · commit 4f577230 · tests 3/3
(binary-level subprocess tests, including a real `sh -c 'cd "$(...)" && pwd'`
composition)

### AC-5 — git worktree add failures surface directly; never reports false success

`gitops.WorktreeAdd`/`WorktreeAddNewBranch` wrap git's combined output into the
returned error, which `Run` writes verbatim to stderr · commit 4f577230 ·
tests 2/2

### AC-6 — Flag completion and --help wired per completion-drift chokepoint

`--base` and the `<branch>`/`[path]` positionals added to
`completion_drift_test.go`'s opt-out lists with rationale; new `aiwf-worktree`
embedded skill added for `skill_coverage.go` and the M-0123 legality-verb
allowlist · commit 4f577230 · tests 3/3 (existing repo-wide policy tests:
`TestPolicy_FlagsHaveCompletion`, `TestPolicy_PositionalsHaveCompletion`,
`TestPolicy_SkillCoverageMatchesVerbs`)

### JSON envelope ordering + coverage gaps

The diff-scoped branch-coverage gate exposed two real problems: the
`--format=json` success path printed the materialization ledger before the
JSON envelope (violating D-0013's single-clean-envelope contract — the
unit test written to close the coverage gap caught it directly), and several
branches (the Cobra `RunE` dispatch itself, the hook-conflict path, and the
JSON/print-path success paths) were only reachable through the binary-level
subprocess tests, whose execution the `go test` coverage instrumentation
cannot see across a process boundary. Fixed the ordering bug, added
Cobra-dispatch/hook-conflict/unit-level print-path and JSON tests, and
`//coverage:ignore`'d the remaining defensive error branches consistent with
this repo's existing `ResolveRoot`/`AcquireRepoLock`/`filepath.Abs`
conventions · commit b7e191d4 · tests 6/6

### Independent review fix-up

The code-quality review (verdict: request-changes) found two blocking gaps:
the existing-branch-reuse dispatch arm had no test driving it through to a
successful full pipeline, and a failure after `git worktree add` succeeded
left the just-created worktree (and branch, if new) registered on disk with
no cleanup — contradicting the verb's atomic-step framing. Added
`gitops.WorktreeRemove`/`DeleteBranch` and a rollback path in `Run` that
undoes worktree + branch creation on any failure after the git step
succeeds; added the missing success-path test; also fixed the non-blocking
`--print-path` + `--format=json` precedence gap flagged as track-for-later
rather than deferring it. The design-quality review (`wf-rethink`) returned
keep-as-is with no blocking findings · commit 7023cf6a · tests 17/17 (11
new/updated in `internal/cli/worktree`, 3 new in `internal/gitops`)

## Decisions made during implementation

None — all decisions in this milestone (branch-exists detection to choose
between reusing vs. creating a branch, `--base` rejected as a usage error
against an existing branch, mapping git-level failures to `ExitInternal`,
shipping a dedicated skill rather than an allowlist entry, rolling back
worktree + branch creation on any post-creation failure) followed existing
codebase precedent (`FinishVerb`'s error-code convention, the
`rev-parse --verify --quiet` existence-probe idiom already used by
`StashTopRef`/`ReadFromHEAD`, and the `move`/`rewidth` vs. `aiwf-add`-style
skill-coverage split) or were direct fixes to review findings, rather than
introducing a new cross-cutting decision.

## Validation

- `make check-fast` (go vet + golangci-lint + full race test suite): clean,
  64 packages `ok`, 0 failures.
- `make coverage-gate` (diff-scoped branch-coverage audit + firing-fixture
  meta-gate): clean.
- Manual binary verification against a built `aiwf` binary (not inferred
  from source): `--print-path` success/failure stdout hygiene, a real
  `cd "$(...)" && pwd` subshell composition, `--format=json` single-envelope
  output, explicit-path repo-escape bypass, existing-branch reuse, and
  post-creation rollback (orphaned worktree + branch confirmed removed via
  `git worktree list` / `git branch`).

## Deferrals

- (none)

## Reviewer notes

Independent two-lens review before wrap (fresh-context subagents, no
authorship attachment):

- **Code-quality** (`wf-review-code`): verdict *request-changes* on first
  pass. Verified all 6 ACs' literal claims by running the built binary
  directly (not just reading the tests), then found the two blocking gaps
  described above under Work log / Decisions. Both fixed and independently
  re-verified — gates re-run clean, and the reviewer's own repro steps
  (missing `aiwf.yaml`, existing-branch reuse) re-run manually against the
  built binary to confirm the fix, not just re-read.
- **Design-quality** (`wf-rethink` on the `internal/cli/worktree` package):
  verdict *keep as-is*, no blocking findings. The branch-exists-then-choose-
  variant design, the `gitops` layering of the new git-wrapper functions,
  and the inline output-mode branching all matched established sibling-verb
  precedent (`update.Run`, `render.go`) rather than introducing a new
  pattern. Confirmed M-0234 (this epic's next milestone) only needs the CLI
  invocation string, not in-process Go reuse, so this verb's public shape
  imposes no friction there.
- `wf-doc-lint` (scoped to this milestone's change-set): clean, 0 findings.
  Design docs still describing raw `git worktree add` (ADR-0009,
  `agent-orchestration.md`) are pre-existing and explicitly out of this
  milestone's scope — M-0234 rewires them.
