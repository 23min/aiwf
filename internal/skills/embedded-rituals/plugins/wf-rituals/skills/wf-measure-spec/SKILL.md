---
name: wf-measure-spec
description: Measure a specification's factual claims by running commands before any code is written, challenge each acceptance criterion for the failure it prevents, and sweep the prose around it when it asserts a count or an enumeration. Use at the start of a milestone or of a change that closes a tracked item, or when the user invokes wf-measure-spec.
---

# wf-measure-spec

A review checks work against a specification. Nothing checks the specification against reality, so a spec that is confidently wrong survives every downstream gate and the work built on it inherits the error.

This ritual is that missing check. It runs before implementation, on the entity that specifies the work — a milestone, or the tracked item a change closes. It measures what the spec asserts, attacks what it asks for, and leaves a record of what it changed.

The parts are not equal, and they are ordered by what they return. Measuring claims is cheap and catches the most. Challenging criteria is cheap and catches the class no measurement reaches. Sweeping what is already written about the code is the expensive one, and it runs when the work touches code that already exists.

## When to use

- Starting a milestone, before the first test is written.
- Opening a change that closes a tracked item, where the item's own body is the specification.
- The work will touch code that already exists, so there is already prose written about it that can disagree with reality.
- A spec asserts a count, a list, or "the only place that does X" — the shapes that are cheap to state and easy to get wrong.
- The user invokes `wf-measure-spec`, or asks whether a spec is right.

It needs a specification to measure. Run it on an entity that carries claims, not on a change with no tracked item behind it.

## The three parts

### 1 — Measure every factual claim by running a command

Read the spec and list what it asserts about the world as it is now — how many call sites, which kinds resolve, what a file contains, where a behavior lives. Each of those is a claim, and each has a command that settles it. Run the command. Record what it returned next to what the spec said. Do not reason about whether a claim is plausible; a plausible claim is exactly the kind that survives review while being wrong.

### 2 — Challenge each acceptance criterion

For every criterion, ask two questions. *What failure does this prevent?* — a criterion that names no failure is a preference, not a criterion. *Can its letter be satisfied while its intent is not?* — `wf-vacuity`'s probes apply to a criterion as they do to an assertion: one satisfied by a tautology, by an over-narrowed case, or by prose containing a word rather than a property holding, cannot be written non-vacuously and should be cut or rewritten before it becomes a test nothing can fail.

### 3 — Sweep everything written about the code the work will touch

This part runs when the work touches code that already exists, and it is the expensive one. Start from what the work touches — the files, and the names in them: functions, types, config keys, commands, error strings. Search the project for each. Read what comes back: reference docs, other specs, instructions that ship to users, comments in the code around it.

Search trunk as it is now, and download it first. A local copy of a remote branch moves only when it is fetched, so searching it without fetching returns whatever was current the last time anyone pulled — which is the stale read this step exists to avoid. No checkout is needed; a ref is searchable in place:

```bash
git fetch --quiet <remote>
git grep -n "<name>" <remote>/<trunk-branch>
```

Search once, on the names. Do not widen to the concept those names belong to: the concept is a word like "history" or "archive", it appears in a large fraction of everything written, and the round it produces is too big to read and so does not get read.

**Follow a reference only where the text leans on it, and only one hop.** What you read will cite other records — a decision, a design doc, a tracked item. Most of those citations are history ("added for X", "regression pin for X") or example data, and neither goes stale. The ones worth following say *this is correct because that record says so*. Check those, then stop: you settle whether a borrowed claim still holds by measuring it, not by reading what that record cites in turn. Measurement is what ends the walk. Where the cited record carries a status, a terminal one settles it without reading the record at all.

**Compare what you find against your measurements, never against each other.** Two documents disagreeing tells you nothing about which one is wrong. Run the command that settles it, and let the command decide which text is stale.

**Two things this will not find, so do not read a clean sweep as an all-clear.** Prose that drifted before the change you are making names things that no longer exist, while the names you are searching are the ones that do — they cannot meet, and the longer a text has been wrong the less likely this is to reach it. And a surface whose defect is its shape rather than its wording — a heading at the wrong level, a section in the wrong place — names no symbol, so no search for a symbol arrives at it. Both of those want a check that runs on every change rather than a reader who runs once; finding one is a reason to write the check, not to sweep harder.

**A sweep finding is a hypothesis until a command settles it.** Do not act on one, do not write it into the spec, and do not report it, before running the command that confirms it. The sweep reads prose, and prose is the evidence most likely to be wrong — it is what the measurement step exists to check.

**Where the sweep finds one fact stated across several surfaces, some of them wrong, the fix is an owner and its derivations — not a correction per copy.** Corrections drift apart again; each copy is a fact nothing re-derives. Give the fact one definition and make every other surface read from it, so the next drift is a failure instead of a difference.

## The record

A completed pass writes a `## Spec measurement` section into the body of the entity whose claims it measured. The section's presence is the record. Its contents are for a human; its presence is what a later reader — a person or a rule — can find without interpreting prose.

Land it the way the project lands entity body edits — through the verb that carries provenance, not a plain commit. Where the project uses aiwf, run:

```bash
aiwf edit-body <id>
```

Write the section in both cases below. A pass that recorded nothing is indistinguishable from a pass that never ran, and the whole point of the record is to tell those apart.

### The pass changed the entity

State what was claimed, what the command returned, and what changed as a result. Name the commands, so the next reader re-runs them instead of re-deriving them. If a criterion was cut or rewritten, say which and why — that is the judgment no command recovers.

### The pass changed nothing

State that the claims were measured and held, and name the commands that measured them. This is the outcome that most needs writing down: it is the one that looks identical to a skipped pass from the outside, and it is the one a reader is most tempted to leave out.

## Anti-patterns

- *Reading the spec instead of measuring it.* Reasoning about whether a claim is plausible is what review already does, and it is what let the claim through.
- *Sweeping first.* It is the expensive part and the one that produces false findings. The two cheap parts run first and often make the sweep unnecessary.
- *Acting on a sweep finding without a command behind it.* The sweep's own headline claim is the one most likely to be wrong.
- *Searching for the spec's phrasings.* That finds the places that agree with a spec you have not yet checked. Search for the code's names instead; they are the same whether the spec is right or wrong.
- *Following every record the text cites.* Each one is cited from a dozen more, so the second hop is hundreds of files and the third is the whole project. Follow the claims the text leans on, one hop.
- *Reading a clean sweep as an all-clear.* It found nothing where it can see. Two known classes sit outside that, and both are listed above.
- *Correcting every copy of a drifted fact.* That is the same defect re-created with a longer fuse. One owner, derivations everywhere else.
- *Skipping the record because nothing changed.* A no-change pass and a skipped pass are the same absence unless one of them writes something.
- *Rewriting the implementation.* This ritual measures and records; it does not build. What it changes is the specification.

## Constraints

- 🛑 Every factual claim is settled by a command, not by judgment. A claim you did not measure is a claim you did not check.
- 🛑 The pass ends with a `## Spec measurement` section in the measured entity, in both outcomes. No exceptions for a pass that found nothing.
- The sweep searches from the code outward, never from the spec's wording. Starting from the spec means searching for confirmation of claims that may themselves be wrong.
- The sweep searches names, not concepts, and follows a borrowed claim one hop and no further. Both bounds exist so the pass terminates in an afternoon; a sweep with no bound is one nobody finishes and therefore one nobody runs.
- A criterion with no mechanical form is not a criterion. Cut it or rewrite it; do not carry it forward as one.
- This ritual runs before implementation. Once code exists, the spec has already shaped it and the measurement is worth less.
