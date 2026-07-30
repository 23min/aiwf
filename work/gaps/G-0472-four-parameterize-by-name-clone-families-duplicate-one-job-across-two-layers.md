---
id: G-0472
title: Four parameterize-by-name clone families duplicate one job across two layers
status: open
---
## What's missing

Several families of near-identical functions each implement one job, differing
only by a name or a key. A whole-tree `dupl` run at threshold 100 — with the
config's path exclusions lifted and `--max-issues-per-linter 0
--max-same-issues 0`, without which golangci-lint truncates at 50 findings —
surfaces the pairs. Reading them as families rather than pairs shows the shape.

**Hook installers** — `internal/initrepo/initrepo.go`: `ensurePreHook` (1332),
`ensurePreCommitHook` (1445), `ensureCommitMsgHook` (1520) and
`ensurePostCommitHook` (1601). All four return `(StepResult, bool, error)`,
resolve the hooks dir, join a filename, compare a marker and branch on dry-run.
Three take `(ctx, root, dryRun)`; `ensurePostCommitHook` takes a fourth
`regenStatus` and carries a `if !regenStatus` opt-out block (1613-1648) that no
sibling has. So three collapse on the hook name alone, and the fourth has a
second behavioral axis to absorb or to leave documented as a sibling.

`dupl` reports only the 1445/1520 pair at threshold 100. That is the token
threshold, not a limitation of pairwise output: dupl does cluster, emitting
pairwise edges that form a cycle. At threshold 60 the three later installers
appear as one — `1459-1475` → `1534-1550` → `1650-1666` → back to `1459-1475`.
Their common fragment is about seventeen lines, which clears 60 and not 100.
`ensurePreHook` shares no clone fragment with any sibling even at 60; it belongs
to the family by shape, not by dupl's measure.

**Legacy-key strippers** — `config.StripLegacyActor` (897) and
`config.StripLegacyAiwfVersion` (954) differ only by which key they remove (27
lines). `initrepo.ensureLegacyActorClean` (969) and
`ensureLegacyAiwfVersionClean` (1012) differ the same way (36 lines) and
*delegate* to the config pair (initrepo.go:996, 1039) — they are dry-run and
`StepResult` reporting wrappers, not a second implementation of stripping.

The cross-layer defect is narrower than duplicated behaviour, and is a bug rather
than a smell. `initrepo` re-implements the *detection* predicate as
`strings.HasPrefix(line, "actor:")` (981) instead of asking `config`, whose
predicate is `isTopLevelActorLine` (config.go:930) and is deliberately
top-level-only. It then discards config's `changed` return (996). If the two
predicates disagree — a nested `actor:` key under some future block matches
initrepo's prefix test and not config's — `initrepo` reports "removed deprecated
'actor:' field" when nothing was removed.

**YAML block replacers** — `aiwfyaml.(*Doc).replaceContracts` (726) and
`(*Doc).replaceHooks` (105) differ by which block they target (28 lines), and
`replaceHooks`'s own doc comment says it "Mirrors replaceContracts".
`appendContracts` and `appendHooks` (hooks.go:117) are a second mirror pair in the
same file that dupl does not flag.

**Verb scaffolds** — `internal/cli/contract/recipes.go` (247-279) and
`internal/cli/contract/unbind.go` (41-73) differ by a verb-name string and the
verb function they call (33 lines). This family is the residue of the shared
prelude extraction that landed under E-0072, and its structure is already pinned
by M-0280's verb-scaffold test, so collapsing it further needs that test's shape
considered rather than only the clone.

## Why it matters

The name-differing families are the cheap kind of duplication: a shared unit is
not merely possible, it is what the code is already shaped like. Each collapses by
taking the differing name or key as a parameter — route, do not design.

The cost of leaving them is drift one plausible line at a time. Four hook
installers mean a fix to marker handling, dry-run reporting or conflict detection
has four homes, and the fourth is the one a reviewer misses. The `appendHooks` /
`appendContracts` pair is evidence the drift is already latent: it is the same
mirror relationship as `replaceHooks` / `replaceContracts`, sitting below dupl's
threshold and therefore invisible to the gate.

The detection-predicate divergence is the one with a wrong-output failure mode
rather than a maintenance cost, which is why it belongs beside this analysis even
though it is small: it was found by asking why one job had two shapes.

## Options

1. **Collapse each family in place** — parameterize by the differing name or key,
   one commit per family, tests following each. Smallest steps, each
   independently reviewable, and it clears four of the eight `dupl` file
   exclusions as a side effect (G-0473 covers the rest of that list).
2. **Collapse the hook installers only** and leave the others. That family is the
   largest and the most edited, so it carries most of the risk; the remainder is
   stable code unlikely to be touched soon.
3. **Fix the detection-predicate defect first**, separately from any collapse.
   It is the only item here that can produce a wrong message rather than extra
   maintenance, and it is a few lines: route initrepo's detection through
   `config`, and consume the `changed` return it already gets back.

Option 3 first, then option 1 family by family with the hook installers leading —
the defect is small, independently testable, and does not want to wait behind a
refactor. Option 2 is a reasonable stopping point if the appetite runs out after
the hook family.

Note for whoever takes option 1: cross-package sharing is not automatically the
right answer here. D-0045 (accepted) deliberately duplicated a small git guard in
`entityview` rather than importing `cliutil`, on layering grounds. The families
above are within-package or within-layer, which is why parameterizing is
straightforward; a fix that reaches across a package boundary needs that decision
distinguished rather than assumed.

## Related

- G-0473 — the `dupl` exclusion list these files sit on, unowned and partly stale
- G-0447, G-0449, G-0450, G-0451 — the duplication-and-dead-code lineage this
  continues
- D-0045 — accepted decision to duplicate rather than import across a boundary
- M-0280 — the verb-scaffold structural test governing the contract family

## Scope

Discovered by a `wf-structural-sweep` pass after E-0073 wrapped. Every function
named above predates that epic, which touched none of the clone-bearing files.
The contract family was last reshaped under E-0072.
