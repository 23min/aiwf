# `docs/explorations/surveys/` — corpus mining outputs

Each subdirectory here is the output of mining a real, working repo for **policy candidates** — the rules / guards / decisions a project encodes about how its codebase or its work must operate. The corpora exist to give the *policies-as-primitive* exploration (`docs/explorations/01–05*.md` and `docs/research/13-policies-as-primitive.md`) something concrete to argue against, not synthetic toy examples.

Distinct from `docs/research/surveys/` — those are *landscape* surveys (literature/operating-model reviews); these are *corpus* surveys (mined material from specific repos).

## Subdirectories

- [`flowtime/`](flowtime/) — mined from `flowtime-vnext` (.NET / TS / Rust / Python polyglot, uses an earlier `aiwf` for work tracking).
- [`liminara/`](liminara/) — mined from `liminara` (CUE-heavy schemas, multi-pack contracts, framework-style docs governance).

## Provenance note

These corpora were extracted as scratch material before the decision to graduate them into tracked content. The source projects (FlowTime, Liminara) are referenced by name and with internal entity ids verbatim throughout, because the original mining was for private use. Both projects are public, so this is acceptable, but the *shape* of the corpora is what generalizes — the named entities are illustrative of one specific repo's invariants and won't make sense outside it.

If material from here ever graduates further (into framework docs, an ADR, or external publication), neutralize the project-specific names per each subdirectory's own sanitization note.
