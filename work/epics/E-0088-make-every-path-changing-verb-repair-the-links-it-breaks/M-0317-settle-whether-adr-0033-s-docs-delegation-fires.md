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
`work/` by whether an `exclude_path` prefix shadows it: 78 links, 73 in files
lychee reads. The five it does not read are all under `docs/archive/pocv3/`,
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

That coverage figure is the half that can rot without a signal, so it is what the
committed test pins. Adding a docs subtree to `exclude_path` takes its links out
of the checked set while link-check keeps passing, because the links stop being
read rather than starting to resolve.

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

## Reviewer notes

The measurement survived independent re-derivation: an outside reviewer rebuilt
the `exclude_path` semantics from its own lychee fixture, reproduced the
three-error red run at `da34c1009`, confirmed the exempt five under
`docs/archive/pocv3/`, and wrote a separate link scanner that agreed with
`linksIntoWork` across all of `docs/`. It also confirmed the G-0478 retitle broke
no inbound path-link — the failure mode this epic exists to close did not fire on
the change that closes it.

Three blocking defects came back, all in the pinning and the prose rather than in
the finding.

**The guard was disarmed by reformatting the file it reads.** Rewriting
`.lychee.toml`'s array onto one line is a semantic no-op that lychee honours; the
hand-rolled parser read it as a single bogus entry, and the check passed green
while 68 links went unread. Single-quoted literals and a trailing comment
defeated it per-entry the same way. The parser now reads exactly one spelling and
refuses every other, routing them to the emptiness check that fails. The
asymmetry is deliberate: a parser that recovers a partial list from an unfamiliar
shape hands back something indistinguishable from a genuinely short list.

**The guard's firing path was dead in the committed suite.** Every prefix
`exclude_path` shadows today is either work-link-free or the exempt archival one,
so the arm that reports a violation was reached only under a hand-applied
mutation. Correctness had been demonstrated and then reverted, which pins
nothing. `TestM0317_DocsFilesLinkingIntoWork` drives the walk over a fixture
tree; a panic injected into each of the four firing arms is now reached by the
committed suite. Worth stating plainly because no gate here could have caught
it: the whole addition is a `_test.go` file, which Go does not instrument, so the
diff-scoped coverage gate is green and silent on all of it.

**A sentence written to correct the ADR was itself false**, in the milestone whose
deliverable is factual accuracy. It claimed lychee runs over "every tracked
markdown file on markdown-touching pushes and PRs": the config's ten
`exclude_path` entries remove a great deal including the whole of `work/`, and
the push trigger is `branches: [main]` only. Both corrected, in the ADR and in
G-0478, which carried the same sentence.

Two judgment findings accepted rather than declined. The ADR's "a CI gate rather
than an advisory one" over-claimed on trunk-based flow where no PR is required
and the gate blocks no merge — softened, and G-0478 now says the first run that
can see a break is the one after the merge to `main`. And the coverage figure was
79/74, counting a link inside a code span that lychee does not resolve; the
lychee-true figure is 78/73, corrected in AC-1 with the discrepancy named.

Declined: renaming the two `LinksIntoWork` subtests that say a suffix "is split"
after the split was deleted. The assertions still pin real behaviour — a mutant
narrowing the capture kills exactly those two — so the names are inaccurate about
mechanism, not about outcome, and D1 pins behaviour rather than implementation.

Not adopted here, recorded instead: `markdownLinkRegex` does not match
CommonMark's titled `[text](path "title")` form, contrary to what its comment
claimed. No tracked document uses that form, so nothing is missed today; widening
it would change what `auditDanglingEntityRefs` scans, which is a decision this
milestone should not make. The comment now states what is so and G-0624 carries
the decision.

The milestone's `## Context` still frames the question as open. That is the
plan-time premise explaining why the work was scoped, and the answer lives in the
Work log; restating it in both places would fork the finding into two copies.

A second independent pass over the full change-set found the model underneath the
guard wrong, and with it two derived numbers.

**`exclude_path` entries are regular expressions, not directory prefixes.**
`lychee --help` says so and `--dump-inputs` confirms it: `work` reaches
`docs/workflows.md`, and `.git` reaches `ADR-0008-…-4-digits.md` through the
`igit` inside "digits". The guard bucketed entries by a `docs/` prefix and walked
only what it judged shadowed, so a work-link added to any of the nine affected
files left it green — the precise failure it exists to catch, demonstrated by
adding one to `docs/workflows.md`. It now compiles each entry and matches it
against every candidate path, keying the archival exemption on the linking file
instead of the entry; the same probe fails, and the same link under
`docs/design/` still passes. `TestM0317_FirstMatch` pins the semantics, and a
mutant swapping the match back to a prefix test dies.

That the primary figures were right anyway is luck rather than method: none of
the nine carries a docs-to-work link. The naive count was reported as 79 and 74
and is 79 and 73, because the code-span link it counts sits in ADR-0008, which
lychee does not read. The read-set is now taken from `--dump-inputs` rather than
inferred. G-0625 carries the exclusion list itself, which is the larger finding
underneath: five accepted ADRs and `docs/workflows.md` are outside link discipline
by accident of substring.

**The parser had a second escape.** It terminated the array at the first `]`
byte, which a bracket inside an in-array comment satisfies — truncating the list
and leaving a fragment that still parses, so the caller received a short list
rather than the refusal it can detect. It now requires a line that is exactly
`]`. The refusal semantics were also unpinned: flipping `return nil` to
`continue` passed the whole suite, because every strictness case held only
unreadable lines. A mixed case closes that, and the mutant now dies.

Two dispositions changed on reflection. The subtests naming a suffix "split"
were renamed after all — the prior round declined on the ground that they pin
behaviour, which stands, but they contradicted `linksIntoWork`'s own doc comment
eight lines above, and two comments in one file disagreeing is its own defect.
The vacuous "empty destination" case was dropped rather than kept as
documentation: `[nothing]()` never matches the pattern, so it exercised no arm
and widening the capture to `*` left the suite green. The anchor and query cases
collapsed to one, since no mutant killed either alone.

G-0624 also now records the pointy-bracket destination `[x](<path>)`, which fails
in the opposite direction from the titled form: it matches, but the capture keeps
the delimiters, so a valid link reads as unresolvable. Both measured.

A third pass found the same class a third time, and the trigger was a change this
milestone itself proposes. TOML decodes escapes inside a basic string, so the
bytes on an `exclude_path` line are not the value lychee compiles: `"^\\.git/"`
reaches lychee as `^\.git/`, and the parser returned the raw bytes, yielding a
regex that matches nothing the intended one matches — non-empty, plausible, and
wrong, which is the single answer the caller cannot detect. Demonstrated as a
live false green. It matters immediately rather than theoretically because
anchoring is what G-0625 proposes, so the next edit to this config would have
tripped it.

The parser now splits the two TOML string forms on the property that actually
distinguishes them: a literal string processes no escapes, so its bytes are its
value and it is read as-is; a basic string carrying a backslash is refused. That
also corrects a decision from the first round, where single-quoted entries were
refused as merely unfamiliar — they are the one form that is always safe to read
raw, and they are the spelling an anchored entry has to use, since `"^\.git/"` is
not loadable TOML at all.

Three smaller corrections. The count of accidentally-excluded documents was
nine in the milestone and the gap while the test comment beside it said eight;
eight is right for Normative tier, the ninth being `docs/working-paper.md`, which
the Exploratory tier already exempts — the change-set was contradicting itself.
G-0625's proposed spelling `"^\.git/"` does not load, corrected to the literal
form, and its scope now names the parser as well as the matcher. And a
half-applied edit of mine had shipped two overlapping comment sentences.

Two findings taken from the round's track-for-later, both the same shape as
things earlier rounds treated as blocking. Two `FirstMatch` cases claimed more
than they tested — the anchored case passed with anchors ignored, because a
trailing slash was doing the work, and the first-match case passed a
return-the-last-match mutant because only one entry matched. Both now
discriminate. And `markdownLinkRegex`, whose corrected comment is the only
production change in this milestone, was unpinned across the whole package: a
mutant admitting the titled form — directly falsifying that comment — left the
suite green. `TestM0317_MarkdownLinkRegexShapes` pins all four shapes, which also
makes G-0624's premise re-checkable rather than remembered.

Carried into G-0478 rather than fixed: the sweep for existing detectors missed
one. `auditDanglingEntityRefs` already resolves path-form entity references at
policy-suite tier over a two-entry list, and its own comment invites extension —
a materially cheaper option than the gap's argument had weighed, and the only
detector covering `ROADMAP.md`, which lychee excludes outright.
