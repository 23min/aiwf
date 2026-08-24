---
id: M-0317
title: Settle whether ADR-0033's docs delegation fires
status: in_progress
parent: E-0088
tdd: none
acs:
    - id: AC-1
      title: A command shows whether doc-lint reports a docs-to-work link break
      status: met
    - id: AC-2
      title: The measured answer is routed to the gaps owning the docs half
      status: met
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
mechanical trigger; a check it never mentions covers the class · measured at
`origin/main` da34c1009

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
`work/` by whether lychee reads it, with the read-set taken from
`--dump-inputs`: 78 links, 73 in files lychee reads. The five it does not read are all under `docs/archive/pocv3/`,
exempt by the Archival tier's forget-by-default convention rather than by
oversight. The count excludes link shapes sitting inside code spans, which
lychee does not resolve either — counting them reports 79 links and still 73
read, the extra being one illustrative link in ADR-0008, a file lychee does not
read.

The read-set here is lychee's own, taken from `--dump-inputs`, not inferred from
the config: `exclude_path` entries are regular expressions matched against the
whole path, so `work` reaches `docs/workflows.md` and `.git` reaches
`ADR-0008-…-4-digits.md` through the `igit` in "digits". Nine documents are
dropped that way; eight are Normative, five of those accepted ADRs, and the
ninth is `docs/working-paper.md`, which the Exploratory tier exempts anyway.
None of the eight carries a docs-to-work link today, so the figures above are
unaffected — but the exclusion list is not doing what its shape suggests, and
G-0625 carries whether to anchor it.

That coverage figure is the half that can rot without a signal: excluding a docs
subtree takes its links out of the checked set while link-check keeps passing,
because the links stop being read rather than starting to resolve. It is a dated
observation rather than an invariant, and it is left as one — see `## Reviewer
notes` for why the guard that pinned it was removed instead of repaired.

### AC-2 — The measured answer is routed to the gaps owning the docs half

**Routed to three records, and the ADR was the one carrying a false claim** ·
commit c30b92051 · internal/policies green

Both branches this AC anticipated turned out to apply, because the answer sits
between them. The delegate ADR-0033 names does not fire, so the ADR is where that
is recorded; the class is covered anyway, so both gaps are re-framed against what
the covering check actually does.

**ADR-0033.** The Decision's second bullet asserted that non-entity narrative "is
covered by the advisory `wf-doc-lint` markdown-link-integrity check". It now names
`link-check`, and says plainly that `wf-doc-lint` is ritual-invoked and so carries
no mechanical trigger. Consequences claimed the residual was "covered by advisory
detection only"; it is covered by a CI gate that fails the run, which is stronger
than advisory and still lands after the push. The decision itself is untouched —
declining to auto-rewrite, on the ground that a verb commit must not reach outside
its entity set, is unaffected by which detector sits behind it.

**G-0478** opened by asserting that nothing reports the breakage, which is the
claim the measurement contradicts, and its title said the same. Both now say what
is so. Its Resolution shape keeps the detection half but for a different reason:
lychee already computes that answer, so a rule inside `aiwf check` buys no new
detection — it buys a pre-push refusal attributable to the commit that caused it,
rather than a workflow result someone has to notice.

**G-0439** needed the least. Its third sketch — detect at the sweep rather than at
the release cut — was already the right shape, and the measurement narrows why:
`CHANGELOG.md` was never outside the detector's reach, so what a sweep skips is
the repair, not the report.

The routing is pinned by the ADR's citation of both gaps rather than by any
wording in the three bodies. `body-prose-id` already refuses a dangling id, so the
uncovered failure was the citation going missing, which that rule cannot see.

## Validation

`make check-fast` exit 0 (race suite plus the full `golangci-lint` set), `aiwf
check` 0 errors against a worktree-built binary, `AIWF_COVERAGE_BASE=1b73a9480
make coverage-gate` exit 0. The measurement runs are recorded per-question under
AC-1 rather than repeated here.

## Deferrals

Three gaps opened from what the measurement turned up. None is this milestone's
to resolve — its Out of scope rules out changing `link-check`, the doc-lint
ritual, or either owning gap's resolution shape.

- **G-0625** — `.lychee.toml`'s `exclude_path` entries are unanchored regular
  expressions, so `work` and `.git` between them drop eight Normative documents,
  five of them accepted ADRs, from link-check. Nothing breaks today; the entries
  do not do what their shape suggests.
- **G-0624** — `markdownLinkRegex` does not match CommonMark's titled link form
  and mis-captures the pointy-bracket one, and a second pattern in the same
  package resolves the titled form differently. No tracked document uses either
  shape today.
- **G-0627** — the AC mechanical-evidence rule has no shape for an observational
  claim, which is what produced this milestone's discarded machinery. See the
  Reviewer notes below.

## Reviewer notes

The measurement survived four independent re-derivations. Reviewers rebuilt the
`exclude_path` semantics from their own lychee fixtures, reproduced the
three-error red run at `da34c1009`, the 3-to-9 retitle experiment, the
`CHANGELOG.md` attribution and the 78/73/5 counts with independently written
scanners, and confirmed the G-0478 retitle broke no inbound path-link — the
failure mode this epic exists to close did not fire on the change that closes
it. No figure in the finding survived unchecked, and none was found wrong that
is not corrected above.

**The AC-1 guard was removed rather than repaired, and that is the milestone's
most useful lesson.** Its claim — every docs file linking into `work/` is a file
lychee reads — is not one this repo can check, because lychee is not available
to the test suite. Pinning it meant re-implementing lychee's file selection in
Go: a TOML parser and a regex matcher, 573 lines. Four review rounds found
four defects and every one was the same shape, the copy disagreeing with the
original — a config reformat the parser mis-read, entries modelled as directory
prefixes where lychee uses unanchored regexes, TOML escapes returned raw as a
different regex, and finally a second array, `exclude`, that silences the whole
class in one line and that the model did not cover at all. Each fix was correct
and each left the next one waiting.

The measured cost decided it: 682 lines of test against five lines of production
change, four of them a corrected comment. A second implementation of another
tool's behaviour is a second source of truth, and this one was kept honest only
by repeated adversarial review. The coverage figure is a dated observation, so it
is recorded as one.

The claim that guard pinned was never AC-1's. AC-1 asks for a measurement, and a
measurement cannot break — the command either was run and reported what it
reported, or it was not. The invariant was invented to satisfy the
mechanical-evidence rule, and then machinery was built to pin the invention. That
is a defect in the rule rather than in this milestone alone: it presumes every
criterion asserts a standing property, and its escape hatch — restate the AC so
something mechanical can carry it — does not distinguish narrowing a claim from
substituting a proxy for it. G-0627 carries that, including the discriminator
that seems to hold: a claim re-derivable from artefacts the test can reach takes
a test, and one that records an observation of something outside the repository
takes the four-part record instead. AC-1 stays `met` on that record; the honest
evidence for a measurement is its reproducibility, not a tripwire.

What survives is what checks our own artefacts rather than modelling someone
else's. `TestM0317_AC2_…` compares ADR-0033's citations against the tree and has
been defect-free throughout. `TestM0317_MarkdownLinkRegexShapes` pins the
behaviour of a production pattern whose comment this milestone corrected, and
makes G-0624's premise re-checkable.

Findings taken along the way, and worth keeping: ADR-0033's description of
`link-check` was wrong twice before it was right — the glob does not reach
dot-directories, `exclude_path` is regex-matched, and there are three triggers,
not two. The accidentally-excluded document count is eight Normative, the ninth
being `docs/working-paper.md`, which the Exploratory tier exempts anyway. G-0625
holds the exclusion-list decision, G-0624 the link-pattern one, and G-0478 gained
a third resolution option nobody had weighed: `auditDanglingEntityRefs` already
resolves this class at policy-suite tier over a two-entry list, and its own
comment invites extension.

Declined, so the next reviewer meets a decision rather than a blank. The
milestone's `## Context` still frames the question as open: that is the plan-time
premise explaining why the work was scoped, and the answer lives in the Work log,
so restating it in both places would fork the finding. And the `exclude` array's
ability to silence the class in one line is left unguarded rather than modelled —
it is real, it is measured above, and guarding it would rebuild exactly what was
just removed. It belongs with G-0625, which owns the config.
