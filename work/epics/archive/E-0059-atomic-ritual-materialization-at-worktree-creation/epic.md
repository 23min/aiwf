---
id: E-0059
title: Atomic ritual materialization at worktree creation
status: done
---

# E-0059 — Atomic ritual materialization at worktree creation

## Goal

Make a freshly-cut git worktree carry the same materialized `.claude/skills/`,
`.claude/agents/`, `.claude/templates/`, and `.claude/aiwf-guidance.md` as the main
checkout — atomically at creation time via aiwf's own tooling, with a session-level
backstop that catches any worktree created outside that path — so ritual discipline
(TDD, vacuity, rethink, gate rules) is never silently absent just because work happens
to be isolated in a worktree.

## Context

Every ritual and instruction in this repo that creates a git worktree (`aiwfx-start-
milestone`'s cut-branch step, presumably `aiwfx-start-epic`, `wf-patch`'s worktree
setup, and the CLAUDE.md "Default to a worktree" / "Subagent worktree isolation"
instructions) stops at `git worktree add` and never runs `aiwf init`/`aiwf update`
afterward. Because `.claude/skills/`, `.claude/agents/`, `.claude/templates/`, and
`.claude/aiwf-guidance.md` are materialize-on-demand artifacts (ADR-0018) and are
gitignored, `git worktree add` never checks them out — every freshly-cut worktree
starts with none of them, and nothing mechanically detects it.

This directly contradicts "Framework correctness must not depend on LLM behavior":
skills are advisory, and a guarantee that depends on the LLM remembering to run
`aiwf update` is not a guarantee. Worktrees are the recommended default for nearly
all branch work here (ADR-0023) and the mandatory mechanism for subagent isolation,
so this sits on the critical path.

The blast radius varies by how a session starts. A session that begins in the main
checkout and later `cd`s into a worktree gets lucky: Claude Code resolves the
`CLAUDE.md` import tree once at session start, so the always-on guidance fragment
rides along even though the worktree's own copy doesn't exist — but the Skill tool's
live "available skills" list does not, so every on-demand ritual skill becomes
unreachable the moment the session moves into the worktree. A session or subagent
that starts fresh with the worktree as its initial directory (the normal case for a
dispatched subagent) gets neither guidance nor invocable rituals, with no error and
no warning — it silently degrades to writing code without the TDD/vacuity/rethink/
gate discipline the framework is built around.

This gap was discovered mid-milestone (M-0186, E-0045) after `wf-rethink` failed to
appear as an available skill; `aiwf doctor` confirmed 17+ missing verb skills and a
completely absent `.claude/skills/`, `.claude/agents/`, `.claude/templates/` despite
substantial work already having happened there under the assumption that ritual
discipline was live. It closes G-0374.

E-0046 ("Formalize in-repo worktrees as the default placement," done) explicitly
scoped this out of its own work — it fixed *where* a worktree goes, not *what's
inside it* once created. G-0099 ("Worktree isolation must be a parent-side
precondition") is the closest sibling concern, but its resolution is pinned to
ADR-0009 and gated behind E-0019, which is currently deferred pending a substrate
rewrite — no live epic currently touches the same ritual call sites this epic needs
to edit.

## Scope

### In scope

- **`aiwf worktree add` verb.** A new Cobra command that performs `git worktree add`
  and `aiwf init`/`aiwf update` materialization as one atomic step — a drop-in
  replacement for the raw two-command sequence regardless of placement. Takes an
  optional explicit target path (mirroring plain `git worktree add <path> <branch>`),
  so a sibling directory or any custom location works with no new config; when the
  path is omitted, it resolves to `<worktree.dir>/<branch-slug>`, honoring the
  existing `worktree.dir` config knob (M-0189) for the in-repo default. Prints the
  resulting absolute path (plain-output line and JSON `result.path`) plus a
  `--print-path` mode that emits only the path, for `cd "$(aiwf worktree add
  <branch> --print-path)"` composition. Ships with `--help`, flag completion, and
  either a dedicated skill or an ADR-0006 allowlist entry.
- **Rewire aiwf's own call sites** to the new verb instead of raw `git worktree add`:
  `aiwfx-start-milestone`'s cut-branch step, `wf-patch`'s worktree setup, and (pending
  the open question below) `aiwfx-start-epic`. Each `SKILL.md` edit lands with its
  required referencing structural test per the `skill-edit-structural-test-backstop`
  policy.
- **Update the CLAUDE.md "Default to a worktree" and "Subagent worktree isolation"**
  sections to instruct the new verb instead of the raw two-command sequence. The
  subagent-dispatch procedure is otherwise unchanged: the parent still passes the
  absolute worktree path into the subagent's prompt rather than relying on `cd`,
  since the new verb has no more ability to change a subagent's cwd than the raw
  command did.
- **Materialized hook registry with persisted consent.** `aiwf.yaml` gains a
  `hooks:` table so any Claude Code hook aiwf ships can be materialized and
  wired into a consumer's `.claude/settings.json`, gated by a per-hook
  decision that persists once made (ADR-0032) — replacing the ad-hoc,
  per-feature consent-flag model. The first (and, for now, only) hook this
  registry ships is the session-level backstop: it checks whether cwd is
  under `.claude/worktrees/` with rituals absent or stale, and warns via a
  harness-rendered notice before the session or subagent proceeds — the
  mechanism that catches any worktree created outside the wrapper (a bare
  `git worktree add`, or a path this epic doesn't rewire).

### Out of scope

- **Isolation-as-precondition / `isolation-escape` finding** (G-0099, ADR-0009) — a
  related but distinct concern (did the work actually land in the worktree, not
  whether the worktree had rituals materialized); stays with E-0019 when that
  substrate unfreezes.
- **`aiwfx-start-epic`'s and `aiwfx-start-milestone`'s three-way placement choice
  itself** (in-repo / sibling / main-checkout) — already settled by ADR-0023 /
  E-0046; this epic only changes *how* the chosen placement gets materialized, not
  which placement is offered.
- **Retrofitting already-existing bare worktrees.** This epic fixes creation going
  forward; an operator with a stale worktree still runs `aiwf update` by hand (or the
  new backstop hook flags it next session).

## Constraints

- The wrapper verb must surface `git worktree add` failures directly — no silent
  swallow-and-continue; a failed worktree creation must not report success.
- `worktree.dir`'s repo-escape rejection (`WorktreeDir()`, M-0190/AC-4) governs only
  the *config-driven default* — it must never apply to an explicit path the caller
  passes. A deliberate sibling-directory placement is a legitimate override, not a
  misconfiguration, and must not be silently redirected back in-repo.
- **`aiwf worktree add` cannot change the invoking shell's or session's cwd** — a
  child process cannot `chdir()` its parent, no CLI can do this. Its contract ends
  at creating the worktree, materializing it, and printing the path; the caller
  performs the `cd` itself. Not a limitation to work around — a fixed process-model
  fact the design (and the skill/CLAUDE.md text describing it) must state plainly
  rather than imply otherwise.
- **`--print-path` is composition-critical and must be tested as such, not eyeballed.**
  On success it emits the absolute path to stdout and nothing else — no progress
  line, no trailing garbage; on any failure it emits nothing to stdout and exits
  nonzero, so `cd "$(aiwf worktree add ... --print-path)"` fails loudly rather than
  landing somewhere wrong. Per this repo's "test the seam, not just the layer"
  convention, this needs a binary-level subprocess test that actually runs `cd
  "$(...)" && pwd` in a real subshell and asserts the landed directory — a Go-level
  unit test of the returned string doesn't exercise the shell-composition seam this
  flag exists for.
- The backstop is a harness-executed hook, not a skill instruction — the whole point
  is removing the LLM-memory dependency, so the check itself cannot be another piece
  of advisory prose.
- No new entity kind or schema change. This is verb + hook + ritual-content work
  within the existing kernel model.
- Rewritten `SKILL.md` bodies follow the existing shipped-surface rule: no real
  entity ids, filesystem paths, or inline lifecycle status — imperative,
  consumer-scoped instruction only.

## Success criteria

- [ ] A worktree created via any of aiwf's own rituals (`aiwfx-start-milestone`,
  `wf-patch`, and `aiwfx-start-epic` if in scope per the open question below) has
  `.claude/skills/`, `.claude/agents/`, `.claude/templates/`, and
  `.claude/aiwf-guidance.md` materialized at creation time, with no separate manual
  step.
- [ ] `aiwf doctor` run immediately after any such worktree is created reports
  `rituals: ok` without an intervening `aiwf update`.
- [ ] A session or subagent that starts fresh with cwd inside an un-materialized
  `.claude/worktrees/` checkout (created outside the wrapper) receives a visible
  warning before proceeding, rather than silently degrading.

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Rewiring multiple `SKILL.md` bodies (each needing its own structural test) balloons one milestone's scope. | med | Give the rewiring its own milestone, sized per call site, separate from the new-verb milestone. |
| The backstop hook false-positives on an intentionally bare worktree (e.g. a throwaway checkout for unrelated inspection). | low | Scope detection strictly to `.claude/worktrees/` (the aiwf-owned convention per ADR-0023); default the hook to advisory/warn, not hard-refuse. |
| `SessionStart`/`SubagentStart` hooks do not support blocking. | med | Confirmed unsupported against the official Claude Code hooks documentation — mitigated by the harness-rendered exit-code/stderr notice, visible to the human without depending on blocking or LLM mediation (ADR-0032, M-0236). |
| A stray stdout write (a progress ping, a future debug print) silently corrupts the `--print-path` / `cd "$(...)"` composition — looks fine in a manual check, breaks only under real shell composition. | med | Dedicated binary-level subprocess test asserting `cd "$(...)" && pwd` lands correctly on success and fails loudly on error; not satisfied by a Go-level string-return test. |

## Milestones

- [M-0233](work/epics/E-0059-atomic-ritual-materialization-at-worktree-creation/M-0233-aiwf-worktree-add-verb-atomic-creation-with-ritual-materialization.md) — `aiwf worktree add`: atomic git-worktree-add + init/update materialization, with completion and tests. · depends on: —
- [M-0234](work/epics/E-0059-atomic-ritual-materialization-at-worktree-creation/M-0234-rewire-aiwf-rituals-and-claude-md-to-use-aiwf-worktree-add.md) — Rewire aiwf's own rituals and CLAUDE.md to the new verb. · depends on: `M-0233`
- [M-0235](work/epics/E-0059-atomic-ritual-materialization-at-worktree-creation/M-0235-generalized-hook-registry-aiwf-yaml-declared-persisted-consent.md) — Generalized hook registry: aiwf.yaml-declared, persisted per-hook consent. · depends on: —
- [M-0236](work/epics/E-0059-atomic-ritual-materialization-at-worktree-creation/M-0236-ship-the-worktree-materialization-check-sessionstart-hook.md) — Ship the worktree-materialization-check SessionStart/SubagentStart hook. · depends on: `M-0235`

## ADRs produced (optional)

- [ADR-0032](docs/adr/ADR-0032-materialized-hook-consent-persisted-per-hook-aiwf-yaml-registry.md) — Materialized hook consent: persisted per-hook aiwf.yaml registry, the
  third instance of the risk-calibrated-consent family (ADR-0015, ADR-0018).

## References

- G-0374 — the gap this epic closes.
- ADR-0018 — materialize-on-demand model for skills/agents/templates/guidance.
- ADR-0023 / E-0046 — in-repo worktree placement default (precedent that explicitly
  scoped this concern out).
- G-0099 / ADR-0009 — adjacent isolation-as-precondition concern, deferred behind
  E-0019.
- `.claude/hooks/validate-agent-isolation.sh` — existing PreToolUse hook pattern the
  backstop milestone follows.
