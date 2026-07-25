---
name: wf-structural-sweep
description: On-demand whole-codebase discovery pass for dead paths, convergent duplication, and unconsumed data flow — the findings a per-diff review and an exact-clone linter both structurally miss. Runs four lenses (whole-program reachability, the clone-detector's known-duplication catalogue, a convergence-scoring pass over the code-health rubric, and a producer→consumer data-flow trace for dropped values and orphan stages), triages each finding against tracked ownership, and emits a scorecard plus gap candidates. Use when auditing an inherited or fast-changing codebase, before a large refactor, or when a module feels heavy. Discovery only — it proposes, it does not auto-fix.
---

# wf-structural-sweep

A repeatable **discovery pass** over a whole codebase. It surfaces two classes of structural rot that slip past the everyday gates: **dead paths** (code that nothing reaches) and **convergent duplication** (the same job implemented more than once in textually-different code). It produces a scorecard and a list of tracked-issue candidates — it never edits code.

This is the discovery front-end for `wf-codebase-health`. That skill is the *rubric* — the forces and smells a reasoner scores against. This skill is the *mechanism* — the concrete sequence that turns the rubric loose on the whole tree and triages what comes back.

## Why a ritual, and not just a linter

The highest-value findings here are exactly the ones a diff review and an exact-clone linter cannot reach:

- A **diff review** sees one change at a time. A dead path or a duplicated implementation is a whole-*graph* property — a one-line change can close a cycle or add the second copy of a helper, and the defect lives in the interaction with the rest of the tree, which the diff does not show.
- An **exact-clone detector** matches text. Two implementations that do the same job in different code — the "we would expect one shared helper, but there are two" smell — are invisible to it. Only a reasoner reading structure surfaces them.

So the pass is worth running deliberately, on demand, as a named sequence.

## When to use

Reach for this pass in *situations*, not on a fixed schedule — when to run it is your judgment, not a rule this skill imposes:

- Auditing an inherited or "vibe-coded" codebase, to decide where to start.
- After a stretch of fast, machine-authored change, to catch accumulated drift before it calcifies.
- A module or package that feels heavy, or that you keep editing in several places for one conceptual change.
- Before a large refactor, to size the duplication it will need to absorb.

## The four lenses

Run all four; they find different things. Each lens is described by its *method* — reach for your stack's tool (see "Per-stack tools" below), not a single hardcoded command.

### Lens 1 — Dead paths (reachability)

Run a **whole-program reachability analysis** — one that treats tests as roots so test-only helpers are not mistaken for rot. It finds functions, types, and branches that nothing reachable ever calls. This is stronger than a package-scoped "unused" check: it catches code reachable only from *other* dead code, and code kept alive solely by a linter-silencing guard (a reference that exists only to suppress the unused-symbol warning).

Read every hit, then **triage before proposing removal** (see below).

### Lens 2 — Textual clones (the known-duplication catalogue)

Run the **clone detector** your project already gates on — and read its *exclusion / known-duplication list as a catalogue*, not merely as "is the build green." Every grandfathered entry is a real clone someone chose to defer; the list is a standing inventory of duplication the team has already acknowledged. Ask which entries are now cheap to collapse.

### Lens 3 — Convergent duplication (the reasoner's lens)

This is the lens no tool reaches. Score the codebase against the `wf-codebase-health` rubric — optionally fanning out one reviewer per package for a large tree — hunting two shapes:

- **Same job, different code** — two or more units that compute the same thing in textually-divergent code, where one shared unit is expected.
- **Helper exists but is bypassed** — a correct shared seam already exists, but several call sites re-implement it inline instead of routing through it. These are the cheapest wins: route, do not design.

Emit a scorecard (Strong / Weak / Missing per principle) with location evidence.

### Lens 4 — Data flow (producer→consumer)

Trace each value the system produces — a field, a computed or derived result, a stage's output — to where it is consumed. This is the lens no reachability or clone tool reaches: a value can be "used" in the call-graph sense (assigned, passed as an argument) yet flow nowhere. Flag:

- **Produced-but-unconsumed** — a value set, passed, or returned but never read for a decision. Reachability calls it live; it isn't.
- **Orphan stages** — a step whose output nothing downstream consumes, or an emitted item a later stage never handles.
- **Duplicate derivations** — the same value computed in two places where one should be the source the other reads. (Convergent duplication seen through the data graph.)
- **Data-dependency cycles** — a producer that depends on a later consumer's output; a dependency graph you expected to be acyclic that turns out not to be.

Reach for your stack's data-flow / def-use tooling where it exists; where it doesn't, trace by reading — the method holds, only the automation changes.

## Triage before you delete

A reachability tool flags **deliberately-retained** code exactly as it flags genuine orphan — the two look identical. Before proposing any deletion, check whether the finding is *owned*:

- Is there an open issue, an accepted decision, or a coupled specification entry that keeps this code on purpose? Retained-but-deprecated code often waits on a coupled change (a downstream reference to repoint, a migration to sequence) — deleting it ad-hoc preempts and half-does that tracked work.
- Only code with **no owner and no coupling** is a clean removal candidate.

Skipping this step is the pass's sharpest failure mode: a confident deletion that contradicts a recorded decision.

## Output

- A **scorecard** for the reviewed scope, and a ranked list of the highest-leverage findings.
- **Tracked-issue candidates**, not edits — file each finding worth keeping as its own issue, scoped and sized, so the pass leaves something durable behind. A convergence finding and a dead-path finding are usually separate issues.
- Nothing is fixed in place. This skill proposes; a separate change-and-review flow disposes.

## Per-stack tools

The lenses are stack-agnostic; the tools are not. Name your stack's equivalent:

- **Go** — whole-program reachability via `deadcode` (run with tests as roots); clone detection via the `dupl` linter; the rubric pass by hand.
- **Other stacks** — substitute the equivalent whole-program dead-code analyzer and clone detector your ecosystem provides; the reasoner's convergence lens is the same everywhere.

If your stack lacks one of the mechanical tools, run the lens by inspection — the method still holds; only the automation changes.

## Anti-patterns

- **Deleting on the tool's say-so.** A reachability hit is a *candidate*, not a verdict — triage ownership first.
- **Reading the clone list as pass/fail.** Its value is the catalogue of deferred duplication, not the green checkmark.
- **Auto-fixing during the sweep.** Discovery and disposal are separate steps; mixing them buries findings inside diffs no one reviews as a set.
- **One lens only.** The four find different rot; running just the mechanical ones misses the convergent duplication and dropped-value findings that motivate the pass.

## Constraints

- 🛑 This skill never edits code. It emits a scorecard and issue candidates; a separate flow makes the changes.
- Findings carry location evidence. A finding without a location is unactionable.
- Triage every reachability hit against tracked ownership before proposing removal.
