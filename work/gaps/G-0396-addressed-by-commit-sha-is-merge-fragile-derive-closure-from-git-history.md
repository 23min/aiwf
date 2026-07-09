---
id: G-0396
title: addressed_by_commit SHA is merge-fragile; derive closure from git history
status: open
discovered_in: E-0063
---
## What's missing

`addressed_by_commit` stores a raw git SHA in a gap's frontmatter, captured at
`aiwf promote <id> addressed --by-commit <sha>` time. It is a denormalized copy
of a fact git already owns — "this gap's closure landed in this commit" — and it
is not stable across the operation that makes the closure real. When the closing
work rides a patch/epic branch that is reconciled (a `git merge main` into the
branch, a rebase) before the `--no-ff` merge to trunk, the commit that lands on
trunk carries a **different** SHA than the one recorded. The recorded SHA is
orphaned: it resolves to a real commit reachable from no branch and not from
trunk, while the fix shipped under a new hash. Measured here: 5 gaps (G-0020,
G-0069, G-0076, G-0119, G-0226) carry orphaned SHAs whose fixes are demonstrably
on trunk under a different commit.

This is a single-source-of-truth violation (C1): the closing commit's identity
lives in git, and frontmatter caches a fragile version of it with no
invalidation rule.

**It is currently symptomless.** No read surface displays `addressed_by_commit`
— not `aiwf show`, not `--format json`, not the roadmap or list renderers. Its
value is read only as "is it non-empty?" by the `gap-addressed-has-resolver`
check. So whether a stored SHA is on trunk or orphaned is invisible in current
behavior; the rot has no user-facing effect today.

## Why it matters

A check policing "is the recorded SHA on trunk" (the direction G-0357 proposed)
is **unsound**: SHA-reachability cannot distinguish a genuinely-lost fix
(abandoned branch, never merged — the real problem) from a benign rewritten SHA
(fix merged under a new hash — a bookkeeping artifact); both present
identically, and here the benign case is 100% of what such a check finds. It
would ship persistent false-positive warnings and erode trust in `aiwf check`.
G-0357 is retired `wontfix` on this basis. The verb-time guard (G-0355) is
unaffected and remains the load-bearing guarantee (it anchors on HEAD at promote
time, correct at that moment); the fragility is only in the stored value
outliving its SHA.

## Decision — additive derivation, keep the field

The field is **not** dropped and **not** migrated. Deriving closure from git is
added *on top of* the stored SHA, not as a replacement, because the two sources
each cover the other's blind spot:

- **Stored SHA:** always present (works offline, on a shallow clone, the instant
  a gap closes) but can go stale.
- **Derived-from-trunk** — the trunk-reachable commit carrying `aiwf-entity:
  <id>` + `aiwf-to: addressed`, which `aiwf history` already parses: never
  stale, but blank for roughly a quarter of this repo's closed gaps. That is the
  legacy cohort (`G-0001` through `G-0055` and peers) bulk-imported already
  `addressed`, which never went through a trailered `promote`, so no closure
  commit exists to find. It is also blank offline, on a shallow clone, or before
  the close reaches trunk.

The design is reconciliation, not replacement:

- **`aiwf show` derives the trunk-reachable closure when one exists, and falls
  back to the stored SHA otherwise.** When neither yields an answer (shallow
  clone, stale trunk ref, pre-convention closure), it reports the closing commit
  as **unknown** — never "not closed." A missing derivation means "cannot see it
  from here," not "the gap is open."
- **Keep the `addressed_by_commit` arm of `gap-addressed-has-resolver`.** For the
  legacy cohort the stored SHA is the only resolver record; dropping the field
  would make roughly fifty legitimately-closed gaps fire the resolver warning —
  the exact noise this avoids.
- **No `aiwf-closes` trailer in v1.** A trailer naming a distinct fix commit has
  no verb to stamp it (aiwf trailers its own entity mutations, not arbitrary code
  commits), so it would be hand-authored and forgettable — the same fragility
  this replaces. Derive the promote commit; treat a distinct or multi-commit fix
  reference as out of scope until a real consumer needs it.
- **Implementation must** canonicalize ids on both sides (historical trailers
  carry narrow widths, e.g. `G-062` for `G-0062`), follow the reallocate /
  `prior_ids` chain (a renumbered gap's closure was recorded under its old id),
  build the trunk index in a single pass for any bulk read (never one git walk
  per gap), and degrade to "unknown" when trunk history is unavailable — never
  error.

Dropping or migrating the field is **out of scope**: the strict frontmatter
parser would eject every file still carrying the key, and the legacy cohort has
no derivable replacement. `addressed_by` (the entity-id resolver) is untouched —
ids survive rebase, so only the raw-SHA resolver was ever fragile.

## Deferred until a read surface needs it

Because the rot is symptomless today, this is **recorded but not scheduled.**
Implement the derivation only when `aiwf show` (or another surface) actually
needs to display a closing commit: there is no bug to fix in current behavior,
only a provenance nicety to add later. Retiring G-0357 returns the tree to fully
symptomless. When built, it warrants an ADR (it changes what a read derives
versus what frontmatter is trusted for) and is one milestone-sized change — not
a `wf-patch`, not an epic.

## Provenance

Surfaced while auditing G-0357's proposed check against this repo (6 off-trunk
SHAs across 5 gaps, all false positives), then stress-tested from four
independent angles that established the roughly-25% non-derivable legacy cohort,
the strict-frontmatter and resolver-precondition constraints on dropping the
field, and the shallow-clone / trunk-staleness limits that make derivation
additive rather than authoritative. Descends from the G-0355 verb-time guard;
supersedes G-0357.
