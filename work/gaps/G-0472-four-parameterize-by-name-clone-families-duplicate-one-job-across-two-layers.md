---
id: G-0472
title: Four parameterize-by-name clone families duplicate one job across two layers
status: open
---
## What's missing

Four families of near-identical functions each implement one job, differing only
by a name or a key. A whole-tree `dupl` run at threshold 100, with the config's
path exclusions lifted and its output limits removed, surfaces every pair;
reading them as families rather than pairs shows the shape.

**Hook installers** — `internal/initrepo/initrepo.go`: `ensurePreHook` (1332),
`ensurePreCommitHook` (1445), `ensureCommitMsgHook` (1520) and
`ensurePostCommitHook` (1601). All four share the signature
`(ctx, root, dryRun) (StepResult, bool, error)`, resolve the hooks dir, join a
filename, compare a marker, and branch on dry-run. They differ by the hook's name
and its script body. `dupl` reports only the 1445/1520 pair (60 lines) because it
compares pairs rather than clustering, so the linter understates this one by half.

**Legacy-key strippers, twice over** — `ensureLegacyActorClean` (969) and
`ensureLegacyAiwfVersionClean` (1012) in the same file differ only by which
legacy key they remove (36 lines). `config.StripLegacyActor` (897) and
`config.StripLegacyAiwfVersion` (954) are the same job again at the config layer
(27 lines). One concept, four implementations, spread across two packages.

**YAML block replacers** — `aiwfyaml.(*Doc).replaceContracts` (726) and
`(*Doc).replaceHooks` (105) differ by which block they target (28 lines).

## Why it matters

Every family is the cheap kind of duplication: a shared unit is not merely
possible, it is what the code is already shaped like. Nothing needs designing —
each family collapses by taking the differing name or key as a parameter.

The cost of leaving them is drift one plausible line at a time. Four hook
installers mean a fix to marker handling, dry-run reporting, or conflict
detection has four homes, and the fourth is the one a reviewer misses.

The legacy-key strippers are the sharper case, because the duplication crosses a
package boundary. `initrepo` and `config` each carry their own idea of what
stripping a superseded key means, so a change to that behaviour has to be made
twice by someone who has noticed both. That is a single-source-of-truth problem,
not only a reuse one.

## Options

1. **Collapse each family in place** — parameterize by the differing name or key,
   one commit per family, tests following each. Smallest steps, each
   independently reviewable, and it retires most of the `dupl` exclusion
   catalogue as a side effect.
2. **Collapse the hook installers only** and leave the rest. The hook family is
   the largest and most active, so it carries most of the risk; the other three
   are stable code that is unlikely to be edited soon.
3. **Reshape the legacy-key strippers as one cross-layer helper** and treat the
   other families separately. Addresses the single-source-of-truth half first, on
   the argument that a cross-package duplicate is worse than an intra-file one
   even when it is smaller.

Option 1 is the lean, taken family by family rather than as one sweep, with the
hook installers first because they are the largest and the most edited. Option 3's
reasoning is right about severity and composes with option 1 — it is an ordering
preference, not an alternative.

## Scope

Discovered by a `wf-structural-sweep` pass after E-0073 wrapped. The clones
predate that epic; none was introduced by it.
