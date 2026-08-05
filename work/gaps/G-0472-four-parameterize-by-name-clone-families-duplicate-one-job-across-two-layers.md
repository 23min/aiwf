---
id: G-0472
title: Four parameterize-by-name clone families duplicate one job across two layers
status: open
priority: medium
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

The cross-layer redundancy is narrower than duplicated behaviour. `initrepo`
re-implements the *detection* predicate as `strings.HasPrefix(line, "actor:")`
(980) instead of asking `config`, whose predicate is `isTopLevelActorLine`, and
it discards config's `changed` return (995).

The two predicates are equivalent for every input, so this costs maintenance
rather than correctness. `isTopLevelActorLine`'s reject-`actorxxx:` guard is
unreachable — the `strings.HasPrefix(trimmed, "actor:")` test above it already
rejects that shape — and its `TrimRight(line, "\r\n")` cannot affect a
left-prefix match, since none of the six characters in `actor:` is a carriage
return or newline. Both predicates are therefore column-0-only by the same
mechanism, and a nested key, being indented, fails both. `isTopLevelAiwfVersionLine`
is the same shape. With the predicates equivalent, the discarded `changed` can
diverge only under a race between initrepo's read and config's re-read inside a
single verb invocation.

The redundancy is still worth removing, and the fix is not parameterization.
`config.Strip*` already computes the fact initrepo's loop re-derives and hands it
back, so consuming that return removes the second predicate from the write path.
The dry-run path needs one more thing: `ensureLegacyActorClean` has to report
what it *would* remove without writing, and `StripLegacyActor` writes
unconditionally, so `config` must also export a detection-only call. Neither its
predicate nor the `LegacyActor` field `Load` populates is usable as-is — the
first is unexported, the second is a YAML parse rather than the textual test.

The dead guard G-0477 tracks lives in `config`'s predicate, the one that
survives this fix; it is a separate one-line cleanup for whoever is already in
these two files.

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

One of these families is the cheap kind of duplication, where a shared unit is
not merely possible but is what the code is already shaped like: the YAML block
replacers collapse by taking the target byte range as a parameter — route, do not
design. The other three look like that and are not, for reasons the Options
section gives per family.

The cost of leaving them is drift one plausible line at a time. Four hook
installers mean a fix to marker handling, dry-run reporting or conflict detection
has four homes, and the fourth is the one a reviewer misses. The `appendHooks` /
`appendContracts` pair is evidence the drift is already latent: it is the same
mirror relationship as `replaceHooks` / `replaceContracts`, sitting below dupl's
threshold and therefore invisible to the gate. That cost is knowingly accepted
for the three families whose collapsed form would be worse than the duplication
— accepting a maintenance cost is the trade, not an oversight.

The hook installers carry a cost of a different kind, and it is the largest thing
on this page: their four members disagree about what a read fault means, three of
them destroying a user's unreadable hook and reporting success. That is a
wrong-output failure mode rather than a maintenance one, it is tracked as G-0557,
and it is the work this family actually wants. It is not why the collapse is
wrong — the Options section gives that reason, on the shared unit's shape — but
it is why a collapse attempted before G-0557 lands would change behaviour
silently, picking one of three semantics with no test to red-light the choice.

## Options

1. **Collapse each family in place** — parameterize by the differing name or key,
   one commit per family, tests following each. Smallest steps, each
   independently reviewable, and it clears six of the eight `dupl` file
   exclusions as a side effect — the two it leaves are the stale entries with no
   clone to clear (G-0473 covers that list).
2. **Collapse only the families whose collapsed form is simpler**, and record a
   reason for each one left alone. Keeps the shared units that pay for
   themselves without treating "these look alike" as sufficient grounds.
3. **Route initrepo's detection through `config`** — consume the `changed` return
   on the write path, and add an exported detection-only call for the dry-run
   path — separately from any collapse. It single-sources a duplicated predicate
   rather than parameterizing one.

Read against the code, the four families do not share a verdict. **Take option 2,
resolved per family as below, with option 3 as the legacy-key family's share of
it.** Option 1 is what the families look like from outside and what reading them
rules out: it clears the most exclusions, and it is wrong for three of the four.

**The YAML block replacers collapse.** A splice helper taking the raw bytes, the
byte range and the block, and an append helper taking the raw bytes and the
block — two free functions, no options struct and no behaviour flag, with each
method reduced to a call plus a field assignment. This is the only family whose
collapsed form is strictly simpler than what it replaces.

**The legacy-key family takes the smaller fix above**, not a collapse. The
config-side pair would parameterize cleanly on the key name, but it is migration
code with a closed set of two keys that by design never grows, so the collapse is
churn worth doing only by someone already in the file for the predicate fix.

**The hook installers should not be collapsed**, and the reason is the shape of
the shared unit rather than anything a later fix changes. A spec needs the hook
name, the marker, the script function, and *two* operator-facing detail strings
that no single parameter generates: the commit-msg installer's plain detail
carries a `"$1"` its own migrated detail drops, where the other three repeat the
same fragment in both. `ensurePostCommitHook` then adds a behaviour flag plus an
uninstall arm carrying the tree's only use of its removal outcome. An options
struct plus a flag plus a second exit path is worse than the duplication it
replaces, and that stays true however the family's other problems resolve.

Separately, and until G-0557 lands, a collapse would also have to silently pick
one of three error semantics with no test to red-light the choice. Landing G-0557
does not make the collapse advisable afterwards.

If this verdict holds it is owed as a recorded decision rather than left as one
gap's lean. G-0473's option 4 needs a `//nolint:dupl` rationale to cite for the
exemption this family keeps, and an open gap's lean is not something a permanent
exemption can rest on. D-0045 is the precedent for the shape. The decision records
the judgment on the shared unit's shape, which is the criterion G-0557 names as
the one that should settle this — not the error semantics.

**The contract verb scaffolds should not be collapsed either.** Every line they
share is already a call into a single-sourced helper — the residue after E-0072
extracted what was extractable, kept routed that way by the verb-scaffold policy.
A shared function would need six parameters plus a callback to replace that
residue, and what remains per-verb is the verb's own identity: its label, its
diagnostic name, the verb function it calls. Three facts that ought to differ are
not duplication worth abstracting over.

A caution for any collapse here: cross-package sharing is not automatically the
right answer. D-0045 (accepted) deliberately duplicated a small git guard in
`entityview` rather than importing `cliutil`, on layering grounds. The YAML
replacers — the one family that does collapse — are within-package, which is why
parameterizing them is straightforward. A fix that reaches across a package
boundary needs that decision distinguished rather than assumed.

## Related

- G-0557 — the hook installers' read-fault divergence: a live data-loss path, and
  the work that family wants instead of a collapse
- G-0477 — the dead guard in `config`'s surviving legacy-key predicate
- G-0473 — the `dupl` exclusion list these files sit on, unowned and partly stale;
  its option 4 consumes the hook-installer verdict above as a recorded decision
- G-0447, G-0449, G-0450, G-0451 — the duplication-and-dead-code lineage this
  continues
- D-0045 — accepted decision to duplicate rather than import across a boundary
- M-0280 — the verb-scaffold structural test governing the contract family

## Scope

Discovered by a `wf-structural-sweep` pass after E-0073 wrapped. Every function
named above predates that epic, which touched none of the clone-bearing files.
The contract family was last reshaped under E-0072.
