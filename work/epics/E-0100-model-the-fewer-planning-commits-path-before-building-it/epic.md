---
id: E-0100
title: Model the fewer-planning-commits path before building it
status: proposed
---

## Goal

Write the behavior ADR-0056, ADR-0057 and ADR-0058 ask for as ratified verifier-kit spines before
any of them is built, so that each implementation epic starts from a spec and is done when
`spine conform` passes against the binary. Played against today's binary, each spine lists the
work its ADR leaves and finds defects.

## Context

- ADR-0058's amend writer is what reduces planning commits. Its "Amend is written only where the
  rest is in place" bullet lets no release write an amended commit unless that release also
  writes ADR-0056's scope ids and ADR-0057's operation ids, and the release before it reads
  commits that carry several operations. ADR-0056 is accepted; ADR-0057 and ADR-0058 are
  proposed.
- The pilot in `docs/research/fewer-planning-commits/verifier-kit-pilot/` wrote two aiwf
  machines as spines. Its scope spine reproduces G-0751 and G-0752, but it was written knowing
  both defects and does not cover G-0755. Its milestone machine went through chapter, two
  readings and answers, and `spine conform` then found G-0769 and G-0770.
- verifier-kit v0.3.3 adds an audit for an existing system (its guide, step 3): a docs reader, an
  opt-in history reader that reads commits and runs code, and a spec partner who asks about each
  conflict between them and records the answers. Docs an answer makes wrong are filed, and
  corrected after the spine is ratified.

## Scope

- One milestone per machine, in the order the build path needs them:
  - the scope machine (ADR-0056): one agent's delegation, adding pause, resume and a copied
    opening commit (G-0755) to what the pilot's scope spine covers;
  - operation identity (ADR-0057): an operation recorded once, recorded again by a cherry-pick or
    a merge, and read by `aiwf history` and `aiwf check`;
  - verb commit output (ADR-0058): amend, push, a lost push reply followed by `pull --rebase`, a
    branch cut, a merge, and `aiwf check` reading the result after each.
- Each milestone follows verifier-kit's guide: settle the lifecycle and its names (step 2); audit it and record
  each answer in `spec/<machine>/audit/decisions.md` (step 3); write the chapter (step 3); two
  readings on different models (step 4); answer their questions into the chapter (step 5);
  ratify the spine with its drawing, `drawing.yaml` and the `drawing.svg` that `spine draw` makes
  from it without warnings (step 6); write a Python adapter (step 8); run `spine conform` against today's
  binary at margin 2 or more and classify each divergence as the change its ADR makes or a
  defect.
- A gap for each defect conform finds that its ADR does not fix, and a gap for each answer that
  makes docs wrong, listing those docs.

## Out of scope

- Building ADR-0056, ADR-0057 or ADR-0058: each gets its own epic, which starts from its
  ratified spine.
- Correcting the docs an answer makes wrong, and fixing the defects conform finds: each goes in a
  change of its own, so a spine's review stays about the behavior.
- Machines off the path: AC and TDD phases, branch rules, id allocation, and the pilot's
  milestone machine.
- The rest of verifier-kit's guide: behaviors, content checks, mutants, the gates and the
  builder (steps 7 and 9–12).
- Races between clients: verifier-kit plays one machine one sequence at a time.

## Constraints

- verifier-kit is pinned at v0.3.3. It builds with Go 1.27, which `GOTOOLCHAIN=auto` fetches;
  aiwf keeps its own Go version.
- Files live under `spec/<machine>/` and `conformance/` at the repository root, in verifier-kit's
  layout.
- An adapter uses only Python's standard library, runs through verifier-kit's adapter shim, plays
  each event with the `aiwf` on `PATH` in disposable repositories, and only translates (guide,
  step 8).
- Each reader sees only its own prompt: the two audit readers don't see each other's output, and
  the two spine readers see neither the code nor each other.

## Success criteria

- [ ] Each machine listed under *Milestones* has a ratified `spec/<machine>/spine.yaml` that
      `spine check` passes, and a `drawing.svg` that is what `spine draw` makes from it.
- [ ] Every conflict and uncovered case an audit raised has an answer in that machine's
      `audit/decisions.md`.
- [ ] Each machine has a recorded `spine conform` run against the binary at margin 2 or more,
      with every divergence classified and every defect filed as a gap.
- [ ] Every answer that makes docs wrong has a gap listing them.
- [ ] ADR-0057 and ADR-0058 are accepted, each revised first where an answer changes it.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does each audit add the history reader to the docs reader? | yes, for that milestone's audit | Decided at each milestone's audit step |
| Is the toolkit installed into the repository (`verify` and `.verifier-kit/`), or is `spine` run from a pinned clone outside it? | yes, for the first milestone | Decided when the first milestone starts |
| Which two models do the readings? | yes, for that milestone's readings | Decided at each milestone's reading step |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| ADR-0058's amend may not fit in one spine; nothing has measured it | med | The verb commit output milestone names what doesn't fit, which stays checked only by the transactions model (`docs/research/transactions/model/RESULTS-amend.md`) |
| The audits raise many questions | med | The pilot's two readings of its milestone machine disagreed on 32 cases and left others undecided, and these came down to four questions; questions are asked one at a time, grouped by what decides them |
| Where the ADR isn't built, most divergences are its change, and a defect can hide among them | med | Classify each divergence by its shortest witness before filing |
| A conform run on a larger machine may take an hour or more; unmeasured | low | Record each run's time; the pilot's scope spine took about two minutes at margin 2 |

## Milestones

- `M-0364` — The scope machine (ADR-0056) · depends on: —
- `M-0365` — Operation identity (ADR-0057) · depends on: —
- `M-0366` — Verb commit output (ADR-0058) · depends on: `M-0364`, `M-0365`, whose ids an amended
  commit carries

## References

- ADR-0056, ADR-0057 and ADR-0058: the behavior each machine models.
- `docs/research/fewer-planning-commits/verifier-kit-pilot/`: the pilot's spines, adapters and
  runs.
- `docs/research/fewer-planning-commits.md`: the research the three ADRs draw on.
- `docs/research/collaborative-state-research-specification-2026-10-24.md`, §6 "Model before
  implementing: verifier-kit oracles".
- `docs/research/transactions/model/RESULTS-amend.md`: the existing model of the amend designs.
- verifier-kit's guide at v0.3.3: <https://github.com/23min/verifier-kit/blob/v0.3.3/docs/guide.md>.
