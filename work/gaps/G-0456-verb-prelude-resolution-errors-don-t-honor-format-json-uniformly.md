---
id: G-0456
title: Verb prelude resolution errors don't honor --format=json uniformly
status: open
discovered_in: M-0279
---
## What's missing

The per-verb `ResolveRoot → ResolveActor` prelude reports resolution failures two different ways depending on the verb. The ~21 text-arm verbs (promote, move, cancel, add, rename, retitle, reallocate, authorize, edit-body, set-area, set-priority, rename-area, contract bind / unbind / recipe-install / recipe-remove, milestone tdd / depends-on, acknowledge illegal / mistag) print a plain stderr line (`aiwf <verb>: <err>`) and exit, ignoring `--format=json`. The two envelope-arm verbs (archive, rewidth) emit a structured error envelope that honors `--format=json`. A `--format=json` consumer therefore receives a machine-readable envelope from a prelude failure on archive / rewidth but an unstructured stderr line from the same failure class on every other verb.

## Why it matters

`--format=json` is the documented machine-consumable output contract; a scripted caller parsing the stdout envelope cannot rely on it for prelude (root / actor) resolution errors on the majority of verbs. The inconsistency is latent and pre-existing — E-0072's `ResolvePrelude` / `ResolvePreludeEnvelope` split faithfully preserved it rather than introducing it (the convergence work was behavior-preserving by constraint). Resolving it means deciding whether every verb should route its prelude errors through the envelope path (a behavior change to ~21 sites), which was deliberately out of scope for the convergence milestone. Surfaced by the design-lens review during the M-0279 wrap.
