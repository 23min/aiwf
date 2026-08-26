---
id: D-0079
title: The remaining citation resolvers do not ship either
status: accepted
relates_to:
    - D-0078
    - G-0595
---
## Question

The gap-truth audit's largest Tier 1 item proposed one citation-resolver rule over
entity bodies: resolve filesystem paths, `path:line` references, Go symbols, and
backticked `aiwf <verb>` / `--flag` / finding codes, fired at the body-write seam
and again pre-push, shipped at warning and promoted to error after a sweep.

D-0078 removed one resolver from that set — entity ids and their status. The audit
held the rest unaffected, on the ground that "each asks whether a referent exists,
not what a sentence claims about it."

Do any of the four remaining resolvers ship?

## Decision

No. None of the four ships, and the item closes.

The retirement-surface scan (`internal/policies/m0290_retirement_surface_test.go`)
also stays scoped to `CLAUDE.md` and the doc tiers it already walks. `work/` is not
added to it.

## Reasoning

Measured 2026-08-26 over the 275 non-archived entity files, counting each backticked
citation once per occurrence:

| resolver | resolve | would fire | genuine |
|---|---|---|---|
| repo-relative path | 371 | 36 | 9 |
| `path:line` | 44 | 2 | 2 |
| `aiwf <verb>` | 552 | 19 | 3 to 6 |
| Go symbol | 488 | 7 | 1 to 3 |

Roughly a quarter of what fires is the defect. The rest is not noise to be tuned
away — it is how a body legitimately writes:

- **A proposal names what it proposes.** G-0235 cites
  `internal/policies/cited_entity_ids_resolve.go` and
  `internal/policies/cache_invalidation_documented.go`, and states in the same body
  that neither exists.
- **A negative test names what must not exist.** G-0161 specifies "ANTI-0005 (no
  `reactivate` verb): invoke `aiwf reactivate`, assert 'unknown command' exit". A
  resolver firing there is wrong in the strongest available sense.
- **A different root resolves.** Eight path fires name `agents/`, `skills/` and
  `templates/` paths that resolve under the materialized `.claude/` tree.
- **Prose shortens a path.** Seven name a suffix of a live path — `check/provenance.go`
  for `internal/check/provenance.go`.

`path:line` fails for its own reason: a line-existence test detects only a number
past end of file, which is 2 of 44. A line that moved — the failure the item exists
to catch — leaves the citation resolvable and wrong.

Widening the retirement scan to `work/` was measured as the cheap alternative and
fails identically: 9 live entities carry the retired verb's name, 2 cite it as a
live capability, and one of the 7 remaining is D-0042, whose title is *"rewidth
reference sweep stays active-tree-only by design"*. A decision about a removed thing
must name it. That scan's own header already excludes ADRs as dated records never
rewritten; `work/decisions/` is the same genre.

**Why this is D-0078's holding rather than a new one.** D-0078 names the condition
that would reopen it: a citation carried in a field, or in a body section whose
meaning is fixed, so that a claim of dependence can be told from a mention. These
four resolvers do not meet it. Each reads structure about the *referent* — a path, a
verb name, a symbol — while the citation itself stays a token in free prose, and a
reader must still decide whether the sentence asserts the referent exists. The
audit's distinction between asking whether a referent exists and asking what a
sentence claims does not survive contact: whether a referent *should* exist is
exactly what the sentence claims.

## Consequences

- The audit's Tier 1 retains one live item, tracked as G-0640.
- G-0168 and G-0229, the two bodies that offered the retired verb as a live
  capability, are repaired by hand. The other seven citations stand.
- What would reopen this is unchanged from D-0078: a citation carried in a field or
  a fixed-meaning body section. A structured `cites:` list would qualify. A token in
  prose does not, whatever kind of thing it points at.
