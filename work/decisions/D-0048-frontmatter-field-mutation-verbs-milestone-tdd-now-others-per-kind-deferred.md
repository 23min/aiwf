---
id: D-0048
title: 'Frontmatter-field mutation verbs: milestone tdd now, others per-kind, deferred'
status: proposed
relates_to:
    - G-0168
    - G-0442
---
## Question

G-0168 identifies four frontmatter fields set only at `aiwf add` time with no
post-creation mutation verb: milestone `tdd:`, gap `discovered_in:`, decision
`relates_to:`, contract `linked_adrs:`. Hand-editing any of them bypasses the
kernel's verb-chokepoint convention (a fictional `aiwf-verb:` trailer, a path no
`--help` reveals, and `aiwf history` naming a verb that resolves to nothing).
What verbs close this, in what shape, and on what schedule?

## Decision

**1. `tdd:` — build now, as a uniform ordinary mutator.**
`aiwf milestone tdd <M-id> --policy none|advisory|required [--reason "..."]`
(exact spelling settled in the implementing milestone). It is a plain frontmatter
data mutator: any actor (human, or an authorized `ai/` with a principal), an
optional `--reason`, standard trailers, and **no directional or sovereign
gating** in either direction — weakening (`required → advisory|none`) is treated
identically to strengthening.

**2. The three id-reference fields — per-kind subverbs, deferred until demanded.**
When friction actually appears, `discovered_in:` / `relates_to:` / `linked_adrs:`
each get a **per-kind subverb** — `aiwf gap discovered-in`,
`aiwf decision relates-to`, `aiwf contract linked-adrs` — mirroring the existing
`aiwf milestone depends-on` idiom (`--on <ids>` / `--clear`). **Not** a generic
`aiwf relate --field <name>` multiplexer. No code is written for these now; only
the shape is fixed.

**3. The set-at-transition fields are out of scope → G-0442.**
`gap.addressed_by` / `adr.superseded_by` are written once as a side effect of
their FSM transition; their amend/clear editor is a distinct problem (it must
refuse an independent set that bypasses the transition) tracked in G-0442.

## Reasoning

**Uniform-ordinary gating for `tdd:` (decision 1).** A rule whose ceremony flips
on direction is an exception by construction; symmetry wants none. Gating a data
field as *sovereign* would also make `tdd:` the first non-status entry in the
sovereign-act tier (`internal/entity/sovereign.go`), which is keyed on FSM status
edges and pinned closed by `TestSovereignActShapes_AllFSMLegal` — forcing a
carve-out into that invariant. And aiwf enforces guarantees at the check layer
(`aiwf check` + pre-push + CI), not via verb-refusals ("errors are findings, not
parse failures"): a mutation verb records the change with provenance, it is not a
policy chokepoint. A weakening stays fully auditable in `aiwf history` regardless.
If scrutiny of a weaken-after-met is ever wanted it arrives as a *symmetric
advisory finding*, never a directional verb gate — and not until real friction
demands it.

**Per-kind over generic (decision 2).** The three fields each live on exactly one
kind, so a `--field` multiplexer earns no cross-kind reuse (unlike `set-priority`
/ `set-area`, whose single field spans several kinds). A generic `relate` verb
would be a novel shape with no precedent — aiwf's field mutators are per-kind
subverbs or fields-named-as-verbs, never `--field`-multiplexed — and it mis-names
`discovered_in`, a single-valued provenance pointer rather than a relation.
Per-kind matches the one real precedent (`aiwf milestone depends-on`), names each
field precisely, and is FSM-safe trivially.

**Deferral (decision 2).** Only `tdd:` has shown friction (twice — the M-0120
downgrade and the later upgrade re-discovery). The other three are
consistency-gap observations, not friction reports; `discovered_in` in particular
is provenance that rarely legitimately changes. YAGNI: fix the proven need, fix
the *shape* of the rest so the guarantee stays honest, and defer the code until
demand is real.

## Follow-ups

- The `tdd:` verb must **refuse with an actionable hint** naming the offending
  ACs — never auto-seed — when a milestone flip would leave an already-`met` AC
  phaseless. A milestone-spec detail, not settled here (see G-0168).
- If the uniform-ordinary / check-layer-governance principle proves load-bearing
  across more verbs, graduate it from this project-scoped decision to an ADR.
