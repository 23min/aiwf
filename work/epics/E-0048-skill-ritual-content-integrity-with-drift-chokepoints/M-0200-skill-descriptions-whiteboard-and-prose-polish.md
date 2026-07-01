---
id: M-0200
title: Skill descriptions, whiteboard, and prose polish
status: in_progress
parent: E-0048
depends_on:
    - M-0196
tdd: advisory
acs:
    - id: AC-1
      title: Completion-boundary forks in plan-epic and wrap-epic shed pause vocabulary
      status: met
    - id: AC-2
      title: whiteboard description states it writes the gitignored WHITEBOARD.md cache
      status: open
    - id: AC-3
      title: wf-codebase-health description leads with its aiwf-ritual identity
      status: open
    - id: AC-4
      title: aiwf-retitle description drops the rename-the-title aiwf-rename collision
      status: open
---
## Goal

Correct the five low-severity prose and description defects catalogued in G-0298,
and pin each fix with a structural test so a future re-drift is caught at CI. The
defects span five skills:

- **`aiwfx-plan-epic` and `aiwfx-wrap-epic`** carry prohibited "pause" / "stop
  here" vocabulary at their completion-boundary forks. The forks are legitimate —
  they sit at genuine "the work is done, the next step is the user's call" moments —
  but the literal vocabulary collides with the standing rule ("never suggest the
  user pause") that these materialized skills are read against every session. Keep
  the forks; reframe the vocabulary as completion.
- **`aiwfx-whiteboard`'s description contradicts its body.** The frontmatter
  `description:` ends "Read-only; no commit; no persisted artefact," but the body
  writes a gitignored `WHITEBOARD.md` cache and an anti-pattern explicitly blesses
  it. An LLM trusting the description skips the cache the body requires. A stale
  cross-reference ("see *Output cache* below") also points the wrong way — the
  section is above.
- **`wf-codebase-health`'s description is a near-twin of the global `code-health`
  skill.** The shared generic opening sentence makes skill selection a coin-flip
  whenever both are active; the `wf-review-code` differentiator is buried at the end.
- **`aiwf-retitle`'s description lists "rename the title,"** colliding with
  `aiwf-rename`'s primary "rename" trigger. The body disambiguates, but the
  description is the selection surface.

No kernel-code change: this milestone edits skill prose only. Four of the five
skills are rituals under `internal/skills/embedded-rituals/**`, so their edits trip
the M-0196 skill-edit→structural-test backstop; the fifth (`aiwf-retitle`) is a verb
skill under `internal/skills/embedded/`, pinned by the same evidence discipline
even though the backstop does not require it.

## Acceptance criteria

### AC-1 — Completion-boundary forks in plan-epic and wrap-epic shed pause vocabulary

`aiwfx-plan-epic`'s "Closing the planning session" fork and `aiwfx-wrap-epic`'s
"Next step" fork keep their structure but drop the prohibited "pause" / "stop here"
vocabulary, reframed as completion ("merge to main if planning is complete for now";
"for whatever's next"). **Test:** a structural assertion that neither skill body
contains `pause` or `stop here` (case-insensitive), and that each carries its
completion-reframe token, so a future edit that reintroduces the prohibited
vocabulary reddens.

### AC-2 — whiteboard description states it writes the gitignored WHITEBOARD.md cache

`aiwfx-whiteboard`'s frontmatter `description:` states it writes a gitignored
`WHITEBOARD.md` cache (replacing "no persisted artefact"), and the body's
cache cross-reference reads "*Output cache* above" (the section is above the
reference). The ≥5 spec-listed query phrasings the existing description-density
test guards are preserved. **Test:** the description contains `WHITEBOARD.md` and
no longer says "no persisted artefact"; the body carries "*Output cache* above"
and not "*Output cache* below".

### AC-3 — wf-codebase-health description leads with its aiwf-ritual identity

`wf-codebase-health`'s `description:` opens with its aiwf-ritual identity — the
per-repo / whole-codebase companion to `wf-review-code`'s per-diff gate — rather
than the generic "field guide of code-health principles" sentence it shares with
the global `code-health` skill. **Test:** the description's opening names
`wf-review-code` (the differentiator) and does not begin with the colliding generic
sentence, so a future edit that reinstates the twin opener reddens.

### AC-4 — aiwf-retitle description drops the rename-the-title aiwf-rename collision

`aiwf-retitle`'s `description:` no longer lists the "rename the title" trigger that
collides with `aiwf-rename`'s primary "rename" trigger; the softened
change/correct-title framing stays. **Test:** the description omits "rename the
title" and retains a change/correct-title trigger, so the collision cannot silently
return.
