---
id: G-0119
title: aiwfx-wrap-epic promotes target before wrap-bundle commits complete
status: addressed
discovered_in: E-0029
addressed_by_commit:
    - "4763355"
---
## What's missing

The `aiwfx-wrap-epic` ritual (in the `ai-workflow-rituals` plugin) promotes the wrap target to `done` *before* creating subsequent wrap-related commits (artefact, reallocate, CHANGELOG, etc.). The promote terminates the active authorize scope. Subsequent commits then carry `aiwf-authorized-by:` referencing the just-ended scope plus `aiwf-actor: ai/claude` with no live authorization — both surface as `aiwf check` errors on push (`provenance-authorization-ended` and, after surgical trailer removal, `provenance-no-active-scope`), blocking the wrap.

Concrete reproduction (E-0029 wrap, 2026-05-12):

```
c030cb9  01:00:55  aiwf promote E-0029 active -> done   ← scope bd7e49b ends here
25c11e1  01:03:03  chore(E-0029): wrap artefact         ← post-promote, still carries authorize trailers
312f378  01:04:58  aiwf reallocate M-0102 -> M-0107
48014b7  01:05:28  chore(epic): wrap E-0029
32a406c  01:06:18  chore(E-0029): CHANGELOG entry for wrap
```

`aiwf check` on push fires `provenance-authorization-ended` against 25c11e1 (and would fire against the other post-promote commits if they carried the same trailer set — varies by ritual step).

## Why it matters

The wrap ritual itself produces commits the kernel's provenance audit cannot validate. Operators are pushed to `--no-verify` to land their wraps, or to history rewrite (which breaks `aiwf history`). This composes with G-0118 as the *second* wrap-time chokepoint trap discovered during E-0029. Both surface as the same symptom (push blocked, no clean remediation); both have the same architectural shape (a documented ritual leaves the chokepoint in a state where the documented remediation isn't enough).

## Likely fix

Reorder the ritual so the terminal `aiwf promote E-XXXX active -> done` is the **last** commit in the wrap bundle, after the artefact, reallocate, CHANGELOG, and any other wrap-related work. This keeps every wrap-bundle commit under the live scope, and the scope-ending promote is itself the natural last act.

Alternative (if reordering is impractical for some skill-internal reason): stamp post-promote commits as `aiwf-actor: human/peter` and drop the authorize trailers — treating the LLM as a tool of conversation for wrap cleanup, per the provenance model's "no co-actor inflation" rule when work isn't autonomous.

Companion: G-0120 captures the reader-side accommodation in the kernel rules, for the case where the historical bad commits already exist and forward-only fixes won't reach them. Both gaps should land together — G-0119 prevents the trap for future wraps; G-0120 lets existing bad commits validate without bypass.

Discovered during E-0029 push (2026-05-13), after G-0118's fix dropped the `out-of-scope` errors and revealed this previously-masked finding underneath.
