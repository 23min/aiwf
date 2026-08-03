# Growth — how aiwf's own apparatus scales with the work it validates

**Status:** living document. This is the *ground doc* for growth in aiwf — where
the growth model, the measured baselines, the levers, and the iteration log live.
Update it when you measure, ship a lever, or change the model.

Companion reading: [`design-decisions.md`](design-decisions.md) (what the kernel
commits to), [`../initiatives/quality-signal-and-cadence.md`](../initiatives/quality-signal-and-cadence.md)
(whether a red gate *means* anything — the adjacent question, deliberately
framed the other way round: that initiative takes the apparatus as given and asks
about its signal, this doc asks about its rate of increase).

---

## TL;DR

1. **The apparatus grows faster than the code it validates, and the gap widens.**
   Production code grew 3.8× over the first twelve weeks; test code grew 6.3×;
   the policy corpus grew 7.6× and now stands at 57% of the size of all
   production code. Nothing here is individually unreasonable; the compound
   effect is.
2. **The mechanism is an asymmetry, not a lapse in discipline.** A majority of
   the chokepoints classified below are satisfied only by *adding* an artifact,
   and none can retire a test. There is exactly one attrition mechanism in the
   repo, and it regulates an exemption list rather than an artifact population.
   A rule set that can only mandate has one stable direction.
3. **Prose creep is count-driven, not length-driven, at roughly 4:1.** Acceptance
   criteria, milestone specs and gap bodies show *no* length trend across the
   whole history. Entity *count* rose 4.4× while words-per-entity rose 31%. A
   word cap would address the smaller term and add a gate to the larger problem.

---

## Measured baseline

Recorded on the kernel repo itself, 2026-08-02, via
[`scripts/growth-report.py`](../../scripts/growth-report.py). The 2026-05-10
column is the same script run with `--at`, not a hand-kept figure.

| dimension | 2026-05-10 | 2026-08-02 | × |
|---|---|---|---|
| production lines (`internal/`) | 20,460 | 78,031 | 3.8× |
| test lines (`internal/`) | 28,095 | 176,490 | **6.3×** |
| test : production ratio | 1.37 | 2.26 | |
| policy files | 39 | 245 | 6.3× |
| policy chokepoints (distinct ids) | 27 | 73 | 2.7× |
| policy lines | 5,877 | 44,743 | **7.6×** |
| policy lines as % of production | 28.7% | **57.3%** | |
| entity files | 216 | 959 | 4.4× |
| entity body words | 131,465 | 763,878 | 5.8× |
| words per entity | 609 | 797 | 1.3× |
| shipped skill + guidance words | 14,474 | 72,744 | 5.0× |

Derived costs at the 2026-08-02 point: cold `go test ./...` is **4m47s** (449s of
package time, `internal/cli/integration` alone 244s); duplicate blocks of ≥100
tokens in the test corpus, measured by running `dupl` with its `_test.go`
exclusions removed, number **211 — 207 of them in test files across 91 files**.

Gap flow over the same window: **426 opened, 324 closed, 110 open**, median age of
an open gap 22.6 days. Flow figures count distinct ids carrying an `add` trailer
and are a lower bound — the earliest gaps predate the trailer convention.

**Half the ledger is same-day.** 160 of 324 closed gaps — **49%** — closed within
24 hours of being opened, at a median of 6 commits and 450 words of body prose
each: 76,851 words of gap prose describing work that was already in flight.

### A worked example

M-0286 added **120 production lines** to the archive sweep and **1,081 test lines
across 17 tests** — 9.0:1. A by-hand marginal-detection audit (revert each fix,
record which tests go red, drop any test that is never the unique detector) cut 5
tests and one test dimension with **no loss of defect detection**: all six
single-fix reverts stayed caught, and package test time fell 13.8s → 10.7s. The
same milestone's epic minted 4 new policies, 2,073 lines.

---

## The growth model

Every chokepoint under `internal/policies/` falls into one of four shapes,
classified by what *satisfying* it does to the artifact population. The script
counts the corpus; this split is a judgment about each rule and is re-derived by
hand, so it carries its own date rather than regenerating — **classified
2026-08-02, over the 73 chokepoints of that day**:

| shape | n | satisfied by | effect on population |
|---|---|---|---|
| **mandate** | 39 (53%) | adding an artifact, once per subject | grows with the codebase |
| **ban** | 24 (33%) | not writing a construct | neutral |
| **uniqueness** | 8 (11%) | collapsing to one site | shrinks |
| **exactness** | 2 (3%) | set-equality | bidirectional |

Three properties of that distribution drive everything above:

- **The chokepoints that can force a deletion — the uniqueness and exactness
  rows — are all single-seam pins on production code.** Not one concerns tests,
  which are the larger part of the corpus.
- **Exactly one chokepoint can say a test does not earn its place**
  (`stress-lane-census`), and its remedy is to *move* the test to another lane,
  not to delete it.
- **The repo's only true attrition mechanism** — the stale-allowlist test that
  forces `firing-fixture-presence`'s grandfather ledger to shrink — regulates an
  *exemption list*, not an artifact population. It worked — that ledger has all
  but burned down. Nothing plays the same role for tests, prose, or gaps.

Mandates compose, which is where the multiplier comes from. One new finding code
obligates four artifacts (a test, a hint, a skill entry, a discoverability
entry). One new policy obligates a firing fixture. One edit to a shipped
`SKILL.md` obligates a policy test. A milestone that touches all three surfaces
pays all three chains.

The one duplication detector the repo already owns, `dupl`, is **switched off
across the test corpus** by three exclusions in `.golangci.yml` (G-0473 tracks
that catalogue being unowned). That is the single largest disabled lever.

### The shipped surfaces

The split above covers `internal/policies/` only — the apparatus this repo runs
on itself. The rules that *ship* were classified separately, by the same by-hand
method — **classified 2026-08-03**, and so carrying its own date for the same
reason. Two of the four shapes differ, which is itself a finding and is taken up
below: `attrition` replaces `uniqueness` (the kernel rules that shrink a
population do it by removing or archiving an entry, not only by collapsing to
one site), and `neutral` replaces `exactness` (a shape absent here, displaced by
a class the internal corpus does not produce).

The finding-code population is the distinct values of the `Code*` constants
declared in `internal/check/*.go`, which is narrower than the set `aiwf check`
surfaces (subcodes are folded into their parent, and codes emitted from other
layers are out of scope):

```bash
grep -rhoP '\bCode[A-Za-z0-9]+\s*=\s*"\K[^"]+' internal/check/*.go | sort -u | wc -l
```

| surface | mandate | ban | attrition | neutral |
|---|---|---|---|---|
| `internal/check` finding codes (57 per the command above) | 16 (28%) | 27 (47%) | 6 (11%) | 8 (14%) |
| always-on guidance fragment (top-level rules) | 2 | 8 | 0 | 1 |

**The shipped kernel does not carry the internal corpus's mandate bias** — 28%
against the 53% above, with a ban majority its finding codes and the always-on
fragment both share. Five of the sixteen kernel mandates are gated by three
`aiwf.yaml` knobs (the `areas` block, `areas.required`, `coverage_roots`) and are
inert until a consumer opts in. Six codes force removal or collapse outright,
and they shrink an *artifact* population — where the policy corpus' one true
attrition mechanism regulates an exemption list, as noted above.

Two findings the internal classification could not have surfaced:

- **`entity-body-empty` is the highest-cost mandate in the shipped surface.** It
  hardcodes required prose sections per kind, always on, no knob. It is what
  sets the per-unit price of every entity in every consumer repo, and it pairs
  with the ritual-side filing mandate: one decides *that* an entity is opened,
  this decides what it costs once opened. Narrowing the first leaves the second
  intact.
- **The ritual corpus is where the mandates concentrate.** Every review finding
  produces an artifact on every disposition — pinned defect, unpinned defect,
  uncovered requirement, accepted judgment, declined judgment — and the review's
  three attrition questions are the only counterweight anywhere in the ritual
  surface. That asymmetry is the subject of D-0054, whose "record obligations,
  not events" rule names the exit that needs no *further* record: a defect
  already fixed and pinned by the check landing with it, where the check the
  disposition already required is the record.

The shape misfit noted above is itself informative. Cost-per-subject and
grows-the-population are two axes that coincide in the policy corpus — a policy
needs a test file — and come apart here: `acs-tdd-audit` costs four promotes per
acceptance criterion and adds nothing to the tree. The eight "neutral" codes are
that class, and it is the class `exactness` had no room for.

## What is *not* the mechanism

Measured across all 289 milestones and 491 gaps, per-unit prose length shows **no
trend**:

| surface | trend |
|---|---|
| AC body words | median 42–70 throughout — flat |
| milestone spec words | 673 → 2,031 (early) → 800–1,400 since — flat to declining |
| gap body words | 276 → 650 → 310 → 590 — noisy, no trend |

Growth is in the *number* of entities, not their size, at roughly 4:1. A length
cap on acceptance criteria or milestone specs would therefore address the minor
term while adding a gate to a repo whose measured problem is gate count. Don't.

The two surfaces that did grow per-unit are the shipped skills (5.0×) and the
repo's own `CLAUDE.md` (6,040 → 11,367 → cut to 5,517 → 8,033 words). `CLAUDE.md`
is the existence proof that a prose surface can be cut back hard without a gate,
and also that it re-grows afterwards.

## Levers

Cost measured, not estimated. None of these is currently shipped.

| lever | what it moves | measured cost | obligation |
|---|---|---|---|
| a cheap-fix escape in the wrap ritual's deferral rule | gap rate, same-day share | text edit to three shipped surfaces | **removes** |
| `dupl` reporting over the test corpus | test duplication | free as a metric; as a gate, the whole backlog measured above to triage | adds, as a gate |
| on-demand marginal-detection audit | test count | ~9 min per file; 100% recall, **11% precision** | adds |
| a retirement mechanism for mandates | policy count | unbuilt | adds |

On the third: a mutation-based audit that flags every test which is never the
unique detector of a mutant reproduces a by-hand audit's verdict completely
(5 of 5) — while also flagging 39 tests that the by-hand audit kept. Tightening
the criterion to strict subsumption (kills ⊆ *and* coverage ⊆) yields a set
**disjoint** from human judgment (0 of 5). No threshold reconciles the two, so
this belongs on demand and never in a gate. Its by-product is worth more than its
verdict: the same run surfaced 33 undetected mutants in one file, including 13
loop-filter branches no test distinguishes.

## Re-measuring

```bash
make growth-report                              # snapshot HEAD
make growth-report GROWTH_BASELINE=<rev>        # HEAD against an earlier point
scripts/growth-report.py --at <rev> --tsv       # reconstruct any past baseline
```

Every metric derives from git history and a tree snapshot, so the *before*
picture is reconstructible at any commit. A lever can therefore land first and be
measured afterwards; nothing has to be measured in advance to stay comparable.

Advisory throughout — this measures, it never gates. A growth budget enforced by
a chokepoint would be the same mistake one level up.

## Iteration log

| date | what changed | headline metric after |
|---|---|---|
| 2026-08-02 | baseline recorded; no lever shipped | test:prod 2.26, policy share 57.3%, same-day gap share 49% |
| 2026-08-03 | shipped surfaces classified; first lever shipped — the review disposition that needs no further record when a defect is already fixed and pinned (D-0054) | kernel finding codes 28% mandate, against the policy corpus' 53% |
