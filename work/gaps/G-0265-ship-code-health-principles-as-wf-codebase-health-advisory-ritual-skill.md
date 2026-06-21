---
id: G-0265
title: Ship code-health principles as wf-codebase-health advisory ritual skill
status: open
---
Downstream aiwf consumers get the kernel's mechanical guarantees and the
ritual skills on `aiwf init` / `aiwf update`, but they do **not** get the
code-health field guide this repo scores itself against (the rubric behind
the periodic health scorecards under `docs/pocv3/`). The rubric is valuable
and reusable, yet it currently lives only as an in-repo doc with no
distribution vehicle.

## Decision

Ship the rubric as a new advisory **ritual skill**, `wf-codebase-health`, in
the `wf-rituals` plugin, materialized into the consumer's `.claude/skills/`
on `init` / `update` like the other `wf-*` engineering skills.

The rubric has two faces, and the patch wires both — **priming primary**:

- **Prime ("do it this way")** — a minimal 5-force digest (D1, C1, C3,
  B1/B2, E1) primes *every turn* via the guidance fragment, and the
  `builder` / `planner` agents and `wf-tdd-cycle` point to the full skill at
  build/design time.
- **Score ("did we do it this way")** — `wf-review-code` and the `reviewer`
  agent point to the full skill for the whole-codebase review pass.

This places the principles in aiwf's **advisory** half (rituals + guidance),
not its **mechanical** half (check rules / hooks): the principles are
judgment forces, not pass/fail rules, and mechanizing them would prescribe
how consumers write code — which aiwf does not do.

## Vehicle — roads not taken

- **Inject the *full* rubric into the per-turn guidance fragment** — rejected;
  it would blow the fragment's line budget and dilute the operating rules.
  Only the minimal digest goes there.
- **A separate dedicated every-turn fragment** — deferred (YAGNI); a fenced
  section in the existing fragment primes every turn at far lower cost than a
  new artifact + wiring.
- **Opt-in materialization of a tracked doc** into the consumer's `docs/` —
  deferred (YAGNI); a future consent-gated knob if a consumer wants it
  committed for human readers.
- **A `check` rule or git hook over consumer code** — rejected; prescribes,
  cannot be language-agnostic, removes the freedom this design preserves.

Full rationale: ADR-0019.

## Scope of the closing patch

- `wf-codebase-health/SKILL.md` (embedded; auto-picked-up by the
  `embedded-rituals` tree embed), reframed to lead with the priming face.
- Minimal code-health digest appended to the guidance fragment
  (`aiwf-guidance.md`), within its line-budget guard, with its scope line
  reworded and a presence test.
- Prime-side pointers from the `builder` / `planner` agents and
  `wf-tdd-cycle`; score-side pointers from `wf-review-code` and the
  `reviewer` agent.
- A structural test asserting the skill materializes and carries the
  principle sections.
- A companion ADR (ADR-0019) recording the vehicle decision.

No `CLAUDE.md` edit is needed: it names the engineering skills collectively
as `wf-*` (no per-skill enumeration), so the new skill is already covered.
