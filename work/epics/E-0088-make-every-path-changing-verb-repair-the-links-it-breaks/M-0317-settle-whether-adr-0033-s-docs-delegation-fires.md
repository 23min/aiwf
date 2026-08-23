---
id: M-0317
title: Settle whether ADR-0033's docs delegation fires
status: in_progress
parent: E-0088
tdd: none
acs:
    - id: AC-1
      title: A command shows whether doc-lint reports a docs-to-work link break
      status: open
    - id: AC-2
      title: The measured answer is routed to the gaps owning the docs half
      status: open
---

## Goal

Establish by measurement whether the advisory check ADR-0033 delegates `docs/`
link integrity to actually reports a break, and route the answer to the two gaps
that own that half.

## Context

ADR-0033's second bullet scopes non-entity narrative out of the movers on the
principle that a verb commit must not reach outside the entity set it owns, and
delegates it to an advisory doc-lint markdown-link-integrity check instead.

Two open gaps measure that narrative rotting anyway. G-0478 counts 59 relative
links from `docs/` into `work/` and finds four broken across two move events,
with the `link-check` workflow red for six consecutive runs before anyone
noticed. G-0439 logs the same shape at a release cut, red for nine runs, and
adds a second instance where a relocation sweep skipped `CHANGELOG.md`. Both
repairs were hand-edits found while looking at something else.

So either the delegation fires and those gaps are mis-framed as defects when
they are an accepted consequence, or ADR-0033 delegates to something that does
not report — and which of those is true is unknown. This milestone produces a
finding rather than a change, which is why it carries no dependency and is
sequenced last.

## Acceptance criteria

### AC-1 — A command shows whether doc-lint reports a docs-to-work link break

In a disposable tree: break a link from a `docs/` file into `work/` by moving
its target, then run the check ADR-0033 names. The command, the result expected
if the delegation works, the output observed, and the environment are recorded
together. Reading the check's source does not settle this.

### AC-2 — The measured answer is routed to the gaps owning the docs half

The result reaches G-0478 and G-0439 as a disposition, not as a note here. If
the delegation fires, both are re-framed against what it actually covers. If it
does not, ADR-0033 carries a delegation to a check that does not report, and
that is recorded against the ADR.

## Constraints

- **Measure; do not conclude from source.** The milestone exists because the
  question has been answered by reading before and the reading disagreed with
  the observed rot.
- **Do not widen the movers.** Whatever the answer, changing what a verb rewrites
  is out of scope here — ADR-0033's second bullet is not being revisited by this
  milestone.
- **Do not absorb the gaps' work.** This milestone disposes of a question; it
  does not implement either gap's resolution shape.

## Design notes

The two gaps disagree slightly about scope — G-0478 is entity-moves-break-docs-
links, G-0439 adds sweeps skipping `CHANGELOG.md` and other non-entity roots.
The measurement should cover both shapes, since a check that catches one and not
the other is a third possible answer.

## Out of scope

- Implementing link rewriting for `docs/`.
- Changing `link-check` or the doc-lint ritual.
- Resolving G-0478 or G-0439 beyond dispositioning the delegation question.

## Dependencies

None. Independent of the other milestones and runnable at any point.

## References

- ADR-0033 — the second bullet, which names the delegation
- G-0478, G-0439 — the gaps that own the `docs/` half
- E-0088 — the parent epic

## Work log

### AC-1 — A command shows whether doc-lint reports a docs-to-work link break

**Both answers, at different addresses.** The check ADR-0033 names has no
mechanical trigger; a check it never mentions covers the class · commit
2a65981da · internal/policies green, 18 M-0317 cases

**Environment.** Disposable clone of this repo at `origin/main` da34c1009,
devcontainer Linux x86_64. lychee 0.24.2 — the version
`lycheeverse/lychee-action@v2` installs by default, so these runs are what CI
executes rather than a stand-in for it. `.lychee.toml` as committed.

**Does the named delegate fire?** ADR-0033 routes non-entity narrative to "the
advisory `wf-doc-lint` markdown-link-integrity check (G-0390)". Searching every
automation surface — `.github/workflows/`, `Makefile`, all six materialized git
hooks, `.claude/settings.json` — for `doc-lint` expected at least one invocation
if that delegation is mechanical. None exists outside the embedded ritual
snapshot. G-0390 is `addressed`, so the check was built; it was built as a skill,
and its only callers are `aiwfx-wrap-milestone`, `aiwfx-wrap-epic`, and the
reviewer agent card. All three are LLM-invoked, which the kernel's own principle
declines to count as a guarantee.

**Does anything else?** `lychee --config .lychee.toml './**/*.md'` expected exit
0 for a clean delegated class. It exits 2 on three errors, each a `docs/` file
naming a `work/` path that no longer exists — the links to G-0584 and G-0317,
both since swept to `archive/`. The workflow passes `fail: true`, so link-check
is red on `origin/main` as this is written.

**Does a mover's break get reported?** `aiwf retitle G-0311` vacated its path
while the docs links kept naming the old slug — G-0478's shape, produced by a
verb rather than waited for. The same command was expected to report the vacated
path if the coverage is real: three errors became nine, the six new ones all
naming it.

**Is a non-entity root covered?** G-0439's second instance is a `CHANGELOG.md`
link into a relocated doc. `exclude_path` lists `work` and `docs/archive`, which
would settle the question against coverage if it filtered link *targets*. Moving
`docs/archive/pocv3/gaps-pre-migration.md` expected silence on that reading, and
produced eleven errors, one of them attributed to `CHANGELOG.md`. So
`exclude_path` filters the files lychee reads and never the destinations it
resolves — which is equally why `work` sitting in that list does not blind it to
docs-to-work links.

**How much of the class is covered?** Bucketing every `docs/` file linking into
`work/` by whether an `exclude_path` prefix shadows it: 79 links, 74 in files
lychee reads. The five it does not read are all under `docs/archive/pocv3/`,
exempt by the Archival tier's forget-by-default convention rather than by
oversight.

That coverage figure is the half that can rot without a signal, so it is what the
committed test pins. Adding a docs subtree to `exclude_path` takes its links out
of the checked set while link-check keeps passing, because the links stop being
read rather than starting to resolve.
