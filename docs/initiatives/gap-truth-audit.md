---
title: Gap truth audit — every gap TODO.md orders, measured against the kernel
status: captured
date: 2026-08-23
---

# Gap truth audit

A dated snapshot of measured divergence between what the gaps `TODO.md` orders
*claim*, and what the kernel at `da34c1009` actually *does*.

Like [`entity-truth-audit.md`](entity-truth-audit.md) and
[`normative-docs-drift-audit.md`](normative-docs-drift-audit.md), this is an
**inventory rather than a proposal**, and it ages by construction: every entry is
either fixed (and deleted) or still true. The `date:` is what makes it honest.

Its subject is narrower than the 2026-08-19 entity audit and its bar is higher.
That audit covered every open gap, live ADR, live decision and Normative doc at a
sampling of one reviewer per clustered batch. This one covers only the gaps
`TODO.md` actually orders — but audits each against the code, drives the binary
wherever a claim is behavioural, and additionally judges `TODO.md`'s own line for
every one of them.

## Scope and method

144 subjects, complete — no sampling:

| Subject class | n |
|---|---|
| Open gaps listed as `TODO.md` entries | 142 |
| Gaps `TODO.md` names that are not on `main` | 1 (G-0622) |
| `TODO.md` itself — preamble, in-flight header, cluster theses | 1 |

Eight further gaps that `TODO.md` cites only in prose as evidence (G-0462,
G-0468, G-0494, G-0542, G-0547, G-0556, G-0558, G-0559) were checked as *claims*
rather than as subjects: what `TODO.md` asserts about each was measured, but their
archived bodies were not re-audited, per the forget-by-default archival convention.

47 independent auditors over 47 batches of two to four gaps, grouped by cluster so
each read one subject area; the orchestrating session audited the 48th batch —
`TODO.md`'s own prose, the in-flight header, and G-0622 — and is not independent of
`TODO.md`, which it wrote. Every auditor worked read-only against a binary built
from the tree under audit, never the `aiwf` on `PATH` — which is, as G-0471 says
and this audit measured, built from a commit that is not an ancestor of `HEAD`.
The repository was not modified; `HEAD` did not move.

Each auditor was required to:

1. **Enumerate every claim before checking any**, then settle each one.
2. **Resolve every reference** — paths, `file:line`, Go identifiers, entity ids
   *and their statuses*, finding codes, `aiwf <verb>` mentions, links, config keys.
3. **For every claim about an ADR or decision, locate the exact sentence that
   carries it** and check its *scope*, not merely its direction. Where no sentence
   carries the claim, the claim is wrong rather than unsupported.
4. **Judge `TODO.md`'s own line** for the gap, independently of the body.
5. **Decide whether the gap is still a gap at all**, naming the closing commit
   where it is not.

**The evidence bar was the point.** A claim counted as settled only where four
things sat together: the command, the expected result, the observed output, and
the environment. Reading settled nothing. Claims that could not be settled are
recorded as **unverifiable** with the command that would settle them, rather than
asserted.

## Headline

| | |
|---|---|
| Findings | **605** (88 high) |
| Subjects with ≥1 finding | **134 / 143** |
| Subjects with no finding at all | **9 / 143** |
| Verdict `sound` (the thesis holds) | **40 / 143** |
| Gaps carrying ≥1 high-severity finding | **57** |

`sound` and *clean* are not the same verdict, and the gap between them is the
useful number: of the 40 gaps whose thesis survives intact, **32 still carry at
least one finding** against a supporting claim. Only nine subjects came back with
nothing at all — G-0459, G-0460, G-0461, G-0471, G-0513, G-0518, G-0544, G-0568
and G-0622.

**Almost every gap is still a real gap.** 110 of 143 survive intact, 31 partly,
and only 2 are wholly gone. What has decayed is the scaffolding: counts that no
longer re-derive, symbols that were renamed, premises a later decision overtook,
and — the class this audit found most of — *prescriptions that are now wrong*.

## Cross-cutting patterns

### 1. The prescription is wrong even where the subject is right (the most consequential class)

A reader who trusts the gap and follows its Resolution does the wrong work. This
is distinct from a stale count, and more expensive: the gap's premise reads sound,
so nothing prompts a re-check before the work starts. Eight measured instances:

- **G-0417** step 2 would write a *new* false spec cell. A non-resolving but
  ritual-shaped `--branch` is accepted (`status: ok`, scope opened), so the cells'
  precondition is wrong in **outcome**, not just in error code.
- **G-0448**'s *Where to fix* names two files and omits `internal/cli/check/provenance.go`,
  which individually invokes **seven** rules and is their only non-test caller —
  leaving exactly the half-wired state the gap warns about.
- **G-0604**'s load-bearing description of the shared walk holds for **one** of five
  sites; building the extraction to it would silently widen resolution at four
  rules, one of which blocks pushes.
- **G-0444**'s fix would write a false statement into a Normative doc:
  `readHistoryChain` is live and exported as `entityview.ReadHistoryChain`,
  extracted three days *before* the gap was filed.
- **G-0524**'s fix fails a CI policy. `PolicyM0132DevcontainerShape` *requires* the
  `${localWorkspaceFolder}/..` the gap proposes removing — applied verbatim in a
  clone, the policy goes red. The gap's "it lands with" list omits it.
- **G-0478** and **G-0439** both prescribe what accepted **ADR-0033** explicitly
  declines ("No pre-push check rule is added for this concern" / "a verb commit
  must not reach outside the entity set it owns"). Neither cites it; G-0439 was
  re-bodied *after* the ADR was accepted.
- **G-0539**'s resolution assumes verb commits stage files and fire hooks. Measured,
  five verbs produced five commits and fired **zero** hooks — they commit via
  plumbing. The proposed marker would do nothing.
- **G-0445**'s option 2 would break every consumer: `docs/adr/` *is* aiwf-managed
  (`aiwf add adr` writes there), so scoping the exclusion to `work/` would make
  every mid-edit ADR false-refuse a legitimate red promote.

### 2. Attribution overclaims its record (45 findings)

The failure **G-0594** exists to record, measured 45 times across 143 gaps — four
days after the rule against it shipped, and in gaps written before and after it.

**G-0594 is itself one of them.** It writes that a specification's *"only gloss"* of
the reading state *ambiguous* is one phrase; the same document defines the term
explicitly 16 lines later, in a commit that is an ancestor of the edit making the
claim. Nine of its ten attributions hold exactly, quotes verbatim and scopes
correct — the tenth fails in precisely the class the gap exists to record.

**`CLAUDE.md` carries one too.** Its stress-harness section says the
workflow-legality group "is the composition coverage G-0121 asks for". Measured, it
reaches none of G-0121's sub-gap 1, none of sub-gap 3's named AC invariant (the
walker seeds no ACs), none of the three chains sub-gap 2 names, and none of the
two-branch merge class. The sentence is inherited verbatim from **D-0063's
*Question***, which then spends the whole decision arguing the coverage is
inadequate. The premise travelled; the qualifier did not.

The remedy this invites is sharpening the wording again, which is what already
failed. This audit records the measurement rather than proposing that remedy.

### 3. Records that a later commit overtook, unnoticed (35 dead premises)

Repeat offenders, each cited in the present tense by multiple gaps:

- **`aiwf rewidth`**, retired by ADR-0039 on 2026-08-03 — still cited live by
  **G-0169, G-0282, G-0400, G-0412, G-0456**. It was the previous audit's headline
  offender and remains one.
- **`internal/policies/sovereign.go`**, deleted 2026-08-06 under D-0061 — cited as a
  live precedent by **G-0535** and **G-0540**.
- **`skill_edit_structural_test_backstop.go`**, deleted by `667c8e0fc` under D-0071 —
  the mechanism **G-0580**'s whole body describes, and the fix **G-0370** prescribes.
- **Cancelled epics still read as live work**: E-0080 (**G-0121, G-0564, G-0568**),
  E-0076 (**G-0471, G-0504**), E-0085 (**G-0583**).
- **`cmd/aiwf/`** as a sweep scope — now one 21-line `main.go` — in **G-0234, G-0412, G-0535**.

### 4. Counts that no longer re-derive (35 findings)

The failure the shipped guidance names. A sample, measured:

| gap | claims | measures |
|---|---|---|
| G-0618 | 19 of 50 shipped files | 19 of **51** |
| G-0607 | twelve regexp sites | **32** |
| G-0400 | twelve scenarios, 10 of 38 verbs | **16** scenarios, **16 of 39** |
| G-0508 | three (title) / four (body) copies | **eight** |
| G-0375 | 221 + 62 failures | **431 + 109** |
| G-0497 | 93 bare writes / 13 packages | **90** / 13 |
| G-0571 | 35 gaps, 60 entities, 119 findings | **31, 56, 111** |
| G-0553 | 102 rows | **107** |
| G-0439 | red for nine runs | **39** |
| G-0396 | ~a quarter non-derivable; ~fifty warn | **10.7%**; **324** |

Several re-derive *exactly at the commit that wrote them* — the method was sound
and the number was a copy. `docs/design/growth.md` remains the counter-example
worth imitating: re-run this audit, and its baseline table still reproduces to the
digit, because every figure names the script behind it.

### 5. A test is holding a wrong prescription in place

`internal/policies/m0289_residue_gap_test.go:60` requires the literals `"Widen"` and
`"canonical id"` to appear in **G-0517**'s `## Resolution` section, keyed by
`const m0289ResidueGap = "G-0517"`, pinning an AC of an archived `done` milestone.

Measured, G-0517's premise is false: of the 41 lines carrying a real-shaped narrow
id across the three surfaces, **zero are citations** — all are worked-example
fiction. Widening them would do what sweep commit `ece70f010` warned against:
*"widening would turn an invented id into a citation of whatever entity now holds
it."* The correct fix is the opposite, and the honest edit turns CI red.

This is a prose-presence assertion of the class **D-0070** retired for shipped
surfaces. Entity bodies fall outside D-0070's scope, so it survived — and has now
graduated from vacuous to obstructive.

## New defects this audit found

Not stale claims in existing gaps — live defects with no gap of their own.

### `body-prose-id` cannot see a non-ASCII fake id

`idTokenPattern` (`internal/check/body_prose_id.go:68`) is
`\b(?:E|M|G|D|C|ADR)-[A-Za-z0-9_]+…`. `[A-Za-z0-9_]` is ASCII-only, so `G-α` never
matches the token pattern and is never considered. Measured: ASCII `G-alpha`
refuses the add; Greek `G-α` passes clean.

**Four live instances sit on `main` right now** — `G-0235`'s body line 33 carries
`G-α`, `G-ζ`, `G-δ`, `G-η`, and `aiwf check` reports the tree clean. The real ids
are named in the gap's own cited source: G-0227, G-0232, G-0230, G-0233. This is
precisely the class `CLAUDE.md` bans ("no letter/placeholder suffixes"), and the
check written to enforce it has an encoding-shaped hole. Belongs in cluster 1.

### `link-check` has been red on `main` for 10 consecutive runs

Commit `d82259f5a` — *"aiwf archive: sweep 8 entities into archive/"*, 2026-08-22
**14:35** — moved G-0584 and G-0317 into `work/gaps/archive/`. The run at 14:26 was
green; the first red is at 14:**37**. Three links broke:

- `docs/initiatives/milestone-preflight-as-independent-review.md:744` → G-0584
- `docs/initiatives/quality-signal-and-cadence.md:226` and `:455` → G-0317

`.lychee.toml` excludes `docs/explorations`, `docs/research`, `docs/archive`,
`work/` and `ROADMAP.md` — **`docs/initiatives` is not on that list**. The
Forward-looking tier is under link discipline and `aiwf archive` does not rewrite
links out of it. This is G-0439's and G-0478's subject with live evidence.

### Two shipped surfaces offer an override the kernel refuses

`aiwf-promote/SKILL.md:47` says `--force` "lets the milestone reach `done`". Measured:
exit 1, HEAD unchanged. The `milestone-done-incomplete-acs` hint recommends
`--force`, then M-0293's caveat sentence contradicts it — the operator reads a
remedy and its retraction in one line.

### `PolicyLayeringDirection` mis-describes its own layering

Its doc comment says `internal/policies` "imports only entity, a leaf-ward edge".
Its non-test files import `entity`, `gitops`, **and** `tree`.

### The always-on guidance channel is serving a stale fragment

`.claude/aiwf-guidance.md` is stamped `v0.32.0-928` at 131 lines against an
embedded source of 146. Three rules are missing from the copy an assistant reads
every turn: the whole *"A gap names what is wrong and where"* bullet, D-0075's
AC-evidence sentence, and G-0610's retraction paragraph — the latter two accepted
and pushed **the same day**. `aiwf doctor` reports `guidance: ok`. That is G-0504's
defect and G-0600's, measured live in the session that audited them.

### 24 GB is reclaimable in `/tmp` while the disk is at 97%

1367 `aiwf-int-build-*` and `stresstest-shared-bin-*` directories older than 24
hours, totalling 24.4 GB, against 8.6 GB free on the overlay. **G-0555** argues
an age-based sweep cannot safely reclaim this; measured, the last 24 hours account
for only 6.5 GB. The gap — and `TODO.md`'s line for it — currently tell a reader
not to reclaim it.

## What causes this — measured, not assumed

Every finding was re-read by four independent classifiers, one per quarter of the
ledgers, and assigned a cause. They counted **616** findings against the collector's
605; the difference is confirmed-sound entries the collector skipped and the
classifiers scored. Percentages below are of 616.

| cause | n | share | what it means |
|---|---:|---:|---|
| **wrong at birth** | 252 | 41% | false or unsupported the day it was committed |
| **drift** | 123 | 20% | true when written, and the world moved |
| **scope-overclaim** | 116 | 19% | a record cited for more than its sentence carries |
| **unsearched absence** | 48 | 8% | "nothing does X" — and X existed, findable |
| **overtaken by decision** | 33 | 5% | an ADR, decision or epic inverted the ground |
| **arithmetic** | 24 | 4% | count sound, method sound, tree moved |
| convention / house style | 14 | 2% | drafting-history narration and the like |
| unknowable or unclear | 6 | 1% | external, or the ledger did not say |

**Drift is a fifth.** The four samples put it at 23.5%, 19.3%, 20.1% and 15.7% —
independent reads, one conclusion. **Roughly two thirds (416 of 616) was false or
unsupported at the moment it was written**, and no freshness mechanism reaches any
of it.

Three findings sharpen this beyond the table.

**Drift is often too fast to re-audit against.** G-0073's "nothing reconciles them"
was reconciled **17 seconds** after the section was committed. G-0553's Fix-column
claim died 9.6 hours later, same day. G-0514's three went stale at 27 minutes, 2
hours and 1 day. No review cadence beats that; the answer is not to write the copy.

**Arithmetic rots only because it is undated.** Six of the eleven counts in one
sample re-derived *exactly* at their authoring commit. A dated measurement is a
historical observation and cannot rot — which is why `docs/design/growth.md`'s
baseline table still reproduces to the digit under re-measurement.

**Absence claims carry severity.** Bodies making negative-existence claims
("nothing does X", "no verb", "no chokepoint", "the only") carry more findings and,
more sharply, are likelier to carry a *high* one:

| negative-existence claims in the body | mean findings | carries ≥1 high |
|---|---:|---:|
| none | 3.92 | 32% |
| one or two | 3.95 | **48%** |
| three or more | **5.47** | 46% |

Correlation with finding count is +0.18 — weak but real. The related intuition that
*absolutes in general* rot is **not** supported: general absolute density (`every`,
`all`, `never`, `any`, `exactly`) correlates slightly **negatively** with findings
(3.89 against 4.30). Body length is the stronger predictor at **+0.44** — the
longest third of bodies average 5.19 findings and 50% carry a high, against 3.53
and 36% for the shortest. A reviewer told to check every absolute would spend the
budget on noise; told to check negative-existence claims in long bodies, they are
aimed at the class where one grep settles it.

### Two amplifiers that are structural, not authorial

**Contagion.** A false claim in one record is re-asserted in the next without
re-derivation. G-0333's "every mutating verb ends with `projectionFindings`" was
copied from `internal/verb/verb.go`'s own package doc, which G-0422 later corrected
as false for 15 entry points. G-0551's overstatement came from D-0059's Reasoning.
G-0439's wrong run-count has already propagated into M-0317's Context; G-0552's
unverifiable cache figure into `TODO.md`. Fixing a source stops downstream copies;
fixing copies does not.

**Closure throughput.** 46 gaps closed in the 20 days since v0.32.0. 35 of them
appear somewhere in this audit's findings, across 206 mentions. Every closure can
invalidate a claim in the ~140 gaps still open, and nothing tells them: G-0599
closing is why G-0548 went stale, G-0617 closing is why G-0073's own closing
condition is already met, G-0559 closing is why G-0517 and G-0560 both carry a dead
sequencing premise. The backlog's own throughput is a drift source.

### The toolchain that produced any of this is not recorded

Whether a given fix was made with a current binary, current skills and current
guidance is **not answerable from the record, for any commit**.

- The materialized guidance is stamped from `ad5bc521d` (2026-08-22 14:20). Eleven
  of the 46 closures postdate it, so those sessions read a fragment already missing
  three shipped rules. For the other 35 nothing can be said: `.claude/` is
  gitignored and overwritten in place, so no past session's guidance is
  reconstructible.
- Of the eighteen trailer keys the kernel writes, **none records a binary version**,
  and nothing in the verb or commit path captures one.

This is the blind spot behind G-0471 and G-0600, stated as a provenance property
rather than a verb defect.

The gap is one of **coverage, not investment**. The provenance model is among the
most heavily built parts of the kernel: roughly 2,300 lines across the five files
whose subject it is, thirteen finding codes, an authorize FSM, force sovereignty,
and coherence checks on both the verb side and the history-walking side. **37 of the
143 audited gaps — a quarter of the backlog — are substantially about it.** All of
that answers *who is accountable* and *who ran the verb*, to a precision little else
in the tree matches. None of it answers *what produced this*.

The vocabulary is also closed and expensive to extend. A single existing trailer key
(`aiwf-prior-entity`) is referenced by every layer that writes, validates, renders or
documents a commit — the trailer writer, the coherence rules, the hint text,
`history`, `render`, two shipped skills, and the tests pinning each. A new key is a
mandate against a policed set, not a field.

Provenance's own defects feed the backlog directly. G-0603 — no chokepoint catches a
missing `aiwf-entity:` trailer while the edit is still cheap — has a measured
consequence in this audit: commit `793b1ad97` fixed **G-0562 and G-0578** on
2026-08-19 carrying no entity trailer, so neither gap closed. Both were still open
four days later, audited as live subjects, and listed in `TODO.md` as two entries
with word-for-word identical descriptions. Two of the 143 subjects existed because a
trailer was absent.

## What would prevent it

Ordered by yield per unit of machinery, not by importance. Each item names the class
it attacks and what it would have caught in this corpus. Where the honest answer is
that no mechanism exists, it says so rather than proposing one that would not work.

The ceiling for everything mechanical here is **12–18% of findings**. Four
classifiers reached that independently. The two thirds that was wrong at birth has
no chokepoint, and the items in Tier 3 are the ones to stop looking for.

### Tier 1 — ship these

**T1.1 — One citation-resolver rule over entity bodies.** Resolvers: filesystem
paths, `path:line`, Go symbols, entity ids **and their current status**, backticked
`aiwf <verb>` / `--flag` / finding codes. Fired at the body-write seam ADR-0043
already defines, plus pre-push. Most machinery exists and is proven — `body_prose_id.go`
already tokenises and masks body prose, `skill_coverage.go` already resolves
backticked verbs against the live Cobra tree, and the planning tree is simply a
corpus nobody pointed them at. *Catches 10–14% of findings across all four samples,
spanning drift, overtaken-by-decision and part of wrong-at-birth. The status half
alone accounts for 7 of 15 high-severity findings in one sample* — "addressed by
E-0076" (cancelled), "until G-0557 lands" (addressed), "ADR-0003 is accepted"
(rejected). *Ban-shaped. Will fire on the existing bodies: ship at warning, sweep,
then promote to error.*

The status resolver is the narrowest slice and is filed as **G-0626**, which
measures the shape and locates the fix: `BodyProseIDIndex` already resolves each
body citation to the cited `*entity.Entity`, and the status that entity carries is
never read. Mint the finding code subcoded from the start, so the remaining
resolvers widen it by subcode rather than by minting further codes.

**T1.2 — Three one-liners.** Best ratio in the list.
- Add `TODO.md` to `aiwf.yaml`'s `docs.paths` (currently `[README.md,
  docs/workflows.md]`, `strict: true`). 175 ids in that file are checked by nothing.
- Widen `body-prose-id`'s token regex from `[A-Za-z0-9_]` to Unicode. Four fabricated
  ids sit on `main` today and `aiwf check` reports clean.
- Point `comment-history-attrition`'s phrase list at entity bodies. The scanner, the
  phrases and the `//history:ok` escape all already ship; only the corpus changes.
  *Catches the ~2% convention class.*

**T1.3 — Duplicate check at `aiwf add gap`.** Print the nearest open-gap titles
before allocating an id. G-0562 and G-0578 name one call site, filed five days apart,
neither referencing the other; G-0580 and G-0618 are the same shape. *The only item
in this list that reduces the number of gaps filed.*

**T1.4 — Cite a record for its holding, not its content.** When another record's
claim is load-bearing on what you are writing, name the record and state what fails
here if that claim is wrong; do not reproduce the claim, or the reasoning behind it.
The test is whether a correction to the cited record would leave the sentence
needing an edit — if it would, it is a copy. Folds into the embedded guidance's
existing *Keep the reasoning; derive the facts* bullet rather than adding one, and
that bullet already carries an `m0211-guidance-operating-anchors` entry, so the rule
is protected without a new anchor. *Attacks the contagion amplifier — which is why
the copies are unfindable today, per Tier 3. Ban-shaped; costs once.*

### Tier 2 — real, but costlier

**T2.1 — Aim the independent reviewer at absence claims in long bodies.** Filing has
no independent-review step; `wf-patch` step 6 already has exactly that shape for
code, and this audit is the evidence it works — fresh-context reviewers found 605
findings across 143 gaps that the authors had not, most settled by one command in
minutes. Two constraints
the measurement imposes on the instruction: **do not** ask for absolutes in general
(no signal, and slightly negative), and **do** ask for every negative-existence claim
to be settled by the search that would falsify it, prioritising long bodies. The
mechanical companion is a warning-severity scan for absence phrasing unaccompanied by
a command block. *Targets the 8% unsearched-absence class, which is where the
severity concentrates. Mandate-shaped — trigger by body shape, not universally.*

**T2.2 — Date-or-derive for counts.** Any cardinal in a body is either dated
("Measured 2026-08-05: 93 sites") or replaced by a reference phrase naming the
detector. Grep-shaped, same masking machinery as `body-prose-id`. *Catches the whole
4% arithmetic class — six of eleven in one sample re-derived exactly at their
authoring commit and rot only from the undated present tense.*

**T2.3 — Quoted-span attribution.** An attribution naming a record carries a verbatim
span, and a rule checks that span appears literally in the named source. *Catches
~11 of the scope-overclaim half, including all three fabricated quotations — one of
which occurs exactly once in repository history, inside the body that presents it as
a quote.* Costs an authoring-convention change before any rule helps. Worth it
because this is the class where review is demonstrably failing: G-0593 shipped the
prose rule, G-0594 measured the next specification breaching it, and this audit found
G-0594 breaching it inside its own highest-severity finding.

**T2.4 — Reverse sweep at decision time.** A ritual step on `aiwf promote <ADR|D>
accepted` and on epic cancellation: name the open gaps this now overtakes. *Reaches
most of the 5% overtaken-by-decision class, which no reference-based rule can touch —
those gaps do not cite the record that killed them.* Also the only item that attacks
the closure-throughput amplifier directly.

**T2.5 — Record the toolchain that produced a mutation.** Makes an entire class
*investigable*: "was this done with current tooling?" is unanswerable for every
commit ever made, and each day without it adds more. It would also give G-0471 the
detector it lacks — a commit whose binary predates its own tree becomes visible.
*Priced against the reach of an existing trailer key, this is a mandate against a
closed, policed vocabulary rather than a cheap addition.* A design question
sits underneath it and should be settled first: **does the version belong on each
commit at all, or on the materialized artifact set?** The statusline already stamps
itself, reads the stamp back, and refuses to downgrade — a working precedent that
points away from a nineteenth trailer key and toward a stamp the artifacts carry and
`doctor` compares.

### Tier 3 — no good solution exists

Recorded so the search stops here rather than recurring.

**Wrong-at-birth behavioural claims (~41%, the largest class).** "Verbs stage entity
files" (they use plumbing and fire no hooks). "CI's `aiwf check` always runs"
(`git log -S` returns zero commits, ever). No check reads a gap body and drives the
CLI. The only control is the filing-time discipline the shipped guidance already
states — the command, the expected result, the observed output, the environment
together — and by this repo's own kernel principle, a guarantee that depends on the
LLM remembering is not a guarantee. A mandatory `## Measured` section could make the
*absence* of evidence visible without judging content, at a per-gap tax H3 warns
against.

**Paraphrase attribution (~13 of scope-overclaim).** "The ADR names the same shape",
"which is the failure D-0070 measured" — a record cited, unquoted, for what it does
not say. T2.3 converts the quoted subset; the paraphrased remainder is unreachable.

**Uniqueness and universals.** "The only", "every other verb", "never a false-refuse".
A grep can flag them for a human; nothing can judge them.

**Contagion.** Nothing can tell that a sentence was lifted from a record rather than
re-derived. The control is fixing sources, not copies — which requires that the
copies be findable, and they are not. G-0333's copy cites
`internal/verb/promote.go:171`, not the package doc it was lifted from, and never
names G-0422, which corrected the claim on 2026-07-18.

Verifying a cited record before citing it was weighed and rejected. Measured
2026-08-24: open gap bodies carry 779 citations across 134 of 167 gaps, a mean of
5.9 per citing gap — roughly six verifications per gap filed, and 779 to retrofit,
which is the mandate shape H3 warns against. It also misses the carriers: of the
four instances above, one has a Go package doc as its source and two have non-gap
targets, so a rule scoped to gap citations fires on at most one. And a verification
is a claim about a moment — one instance measured here expired 2 hours 34 minutes
after it was written.

The write-side control that *is* reachable is T1.4.

The entity citation graph offers no leverage of its own: measured the same day, 287
distinct targets for those 779 citations, mean 2.7, most-cited target 11. There is
no hub set among entities to fix. The one instance traceable end to end had a Go
package doc as its hub, which points at package docs and design-doc prose as where
a false universal reaches the most readers.

**Wrong causal stories.** "An accident of function signature" for what is a declared,
AC-pinned layering boundary. Nothing greps for a wrong explanation.

## Ready to act on

### Promote

| gap | commit | note |
|---|---|---|
| G-0562 | `793b1ad97` | fixed 2026-08-19; duplicate of G-0578 |
| G-0578 | `793b1ad97` | same site, same commit, same fix |
| G-0573 | `29eb2a94c` | ADR-0042 rules the residue moot rather than fixed |
| G-0622 | `0e7500a8e` | already `addressed` on the M-0315 branch; arrives terminal on merge |

`793b1ad97` and `0e7500a8e` carry **no `aiwf-entity:` trailer**, which is why nothing
closed G-0562 or G-0578 — G-0603's defect with a measured consequence.

### Close as superseded

**G-0580** — its mechanism was deleted under D-0071, and **G-0618** names the same
scope hole against the *current* predicate, plus three surfaces G-0580 misses.

### Rewrite rather than close

**G-0212** — M-0243 (done, all 5 ACs met) converted four of its six items into named
stress scenarios and M-0241 covered a fifth. Two returned answers *contradicting*
the gap: merging always produces a genuine git conflict rather than a silent
overwrite, and repolock is a `syscall.Flock`, inter-process by construction.

**G-0111** — concern 4 shipped as `9cad2af3e` under G-0431; only the declarative
`closes:` field remains. Its concern 3 prescribes `aiwf authorize --end`, a flag
that has never existed.

**G-0060** — `aiwf promote G-NNNN addressed --by-commit <sha>` works, verified
end-to-end; a branch model exists in `branchparse.go`; `aiwf check` has patch-shaped
invariants. What survives is narrower: no patch-side queryable record, and `check`
does not validate the SHA's value.

### Per-subject verdicts

| gap | verdict | still a gap | high | med | low |
|---|---|---|---|---|---|
| G-0060 | `stale-claims` | partly | 4 | 5 | 3 |
| G-0068 | `stale-claims` | yes | 2 | 2 | 2 |
| G-0070 | `stale-claims` | partly | 2 | 3 | 0 |
| G-0073 | `stale-claims` | yes | 3 | 5 | 2 |
| G-0110 | `contradicted-by-code` | partly | 2 | 1 | 2 |
| G-0111 | `stale-claims` | partly | 2 | 4 | 4 |
| G-0121 | `stale-claims` | yes | 1 | 4 | 2 |
| G-0161 | `stale-claims` | yes | 3 | 3 | 2 |
| G-0168 | `stale-claims` | partly | 4 | 4 | 3 |
| G-0169 | `stale-claims` | partly | 2 | 2 | 1 |
| G-0212 | `already-addressed` | partly | 3 | 1 | 2 |
| G-0217 | `contradicted-by-code` | partly | 1 | 1 | 2 |
| G-0233 | `stale-claims` | yes | 0 | 2 | 3 |
| G-0234 | `stale-claims` | yes | 0 | 3 | 1 |
| G-0235 | `stale-claims` | partly | 0 | 8 | 4 |
| G-0246 | `stale-claims` | yes | 0 | 1 | 2 |
| G-0249 | `stale-claims` | yes | 0 | 1 | 3 |
| G-0253 | `sound` | yes | 0 | 2 | 1 |
| G-0254 | `stale-claims` | yes | 1 | 1 | 2 |
| G-0282 | `stale-claims` | yes | 3 | 1 | 4 |
| G-0302 | `stale-claims` | yes | 0 | 2 | 1 |
| G-0307 | `stale-claims` | yes | 0 | 4 | 2 |
| G-0311 | `stale-claims` | yes | 1 | 3 | 1 |
| G-0328 | `stale-claims` | partly | 1 | 1 | 1 |
| G-0333 | `stale-claims` | partly | 1 | 6 | 2 |
| G-0366 | `stale-claims` | yes | 0 | 2 | 1 |
| G-0369 | `sound` | yes | 0 | 0 | 3 |
| G-0370 | `stale-claims` | yes | 1 | 3 | 1 |
| G-0372 | `stale-claims` | yes | 0 | 2 | 1 |
| G-0375 | `stale-claims` | yes | 0 | 3 | 3 |
| G-0385 | `sound` | yes | 0 | 0 | 3 |
| G-0396 | `stale-claims` | yes | 0 | 5 | 1 |
| G-0398 | `stale-claims` | yes | 1 | 2 | 2 |
| G-0400 | `stale-claims` | yes | 0 | 3 | 3 |
| G-0412 | `stale-claims` | yes | 1 | 5 | 1 |
| G-0414 | `sound` | yes | 0 | 0 | 1 |
| G-0417 | `stale-claims` | yes | 0 | 3 | 3 |
| G-0434 | `stale-claims` | yes | 0 | 3 | 2 |
| G-0436 | `stale-claims` | partly | 0 | 3 | 3 |
| G-0439 | `stale-claims` | yes | 0 | 2 | 1 |
| G-0442 | `stale-claims` | partly | 1 | 4 | 1 |
| G-0444 | `stale-claims` | yes | 1 | 0 | 2 |
| G-0445 | `stale-claims` | yes | 1 | 1 | 4 |
| G-0448 | `stale-claims` | partly | 2 | 1 | 3 |
| G-0453 | `stale-claims` | yes | 0 | 3 | 3 |
| G-0454 | `stale-claims` | yes | 0 | 3 | 2 |
| G-0455 | `stale-claims` | yes | 0 | 1 | 3 |
| G-0456 | `stale-claims` | yes | 0 | 1 | 2 |
| G-0458 | `stale-claims` | yes | 0 | 1 | 0 |
| G-0459 | `sound` | yes | 0 | 0 | 0 |
| G-0460 | `sound` | yes | 0 | 0 | 0 |
| G-0461 | `sound` | yes | 0 | 0 | 0 |
| G-0464 | `sound` | yes | 0 | 0 | 1 |
| G-0471 | `sound` | yes | 0 | 0 | 0 |
| G-0472 | `stale-claims` | partly | 1 | 3 | 3 |
| G-0473 | `stale-claims` | yes | 0 | 2 | 4 |
| G-0477 | `sound` | yes | 0 | 1 | 1 |
| G-0478 | `stale-claims` | yes | 2 | 6 | 0 |
| G-0483 | `stale-claims` | yes | 0 | 4 | 2 |
| G-0486 | `stale-claims` | partly | 0 | 3 | 1 |
| G-0493 | `sound` | yes | 0 | 1 | 0 |
| G-0497 | `stale-claims` | yes | 0 | 3 | 1 |
| G-0498 | `sound` | yes | 0 | 0 | 2 |
| G-0500 | `sound` | yes | 0 | 0 | 1 |
| G-0501 | `sound` | yes | 0 | 0 | 2 |
| G-0502 | `sound` | yes | 0 | 0 | 3 |
| G-0504 | `stale-claims` | yes | 1 | 2 | 1 |
| G-0506 | `stale-claims` | yes | 0 | 1 | 2 |
| G-0508 | `stale-claims` | yes | 1 | 3 | 1 |
| G-0510 | `sound` | yes | 0 | 1 | 2 |
| G-0512 | `sound` | yes | 0 | 0 | 1 |
| G-0513 | `sound` | yes | 0 | 0 | 0 |
| G-0514 | `stale-claims` | yes | 2 | 4 | 0 |
| G-0516 | `sound` | yes | 0 | 0 | 2 |
| G-0517 | `stale-claims` | yes | 1 | 3 | 1 |
| G-0518 | `sound` | yes | 0 | 0 | 0 |
| G-0519 | `sound` | yes | 0 | 1 | 0 |
| G-0523 | `stale-claims` | yes | 1 | 1 | 1 |
| G-0524 | `stale-claims` | partly | 0 | 2 | 3 |
| G-0526 | `stale-claims` | partly | 0 | 3 | 2 |
| G-0527 | `stale-claims` | partly | 2 | 1 | 1 |
| G-0529 | `stale-claims` | yes | 1 | 3 | 1 |
| G-0530 | `stale-claims` | partly | 3 | 3 | 1 |
| G-0533 | `sound` | yes | 0 | 0 | 3 |
| G-0535 | `stale-claims` | yes | 2 | 2 | 1 |
| G-0536 | `stale-claims` | yes | 1 | 2 | 1 |
| G-0537 | `sound` | yes | 0 | 0 | 2 |
| G-0538 | `sound` | yes | 0 | 0 | 2 |
| G-0539 | `stale-claims` | yes | 1 | 1 | 0 |
| G-0540 | `stale-claims` | partly | 0 | 2 | 3 |
| G-0543 | `sound` | yes | 0 | 0 | 1 |
| G-0544 | `sound` | yes | 0 | 0 | 0 |
| G-0545 | `stale-claims` | yes | 0 | 2 | 2 |
| G-0546 | `stale-claims` | partly | 0 | 2 | 2 |
| G-0548 | `stale-claims` | yes | 0 | 5 | 1 |
| G-0549 | `sound` | yes | 0 | 0 | 1 |
| G-0550 | `sound` | yes | 0 | 0 | 3 |
| G-0551 | `stale-claims` | yes | 0 | 1 | 2 |
| G-0552 | `sound` | yes | 0 | 0 | 1 |
| G-0553 | `stale-claims` | yes | 0 | 3 | 0 |
| G-0554 | `sound` | yes | 0 | 1 | 2 |
| G-0555 | `stale-claims` | yes | 1 | 4 | 2 |
| G-0560 | `stale-claims` | yes | 2 | 3 | 1 |
| G-0561 | `sound` | yes | 0 | 1 | 0 |
| G-0562 | `already-addressed` | partly | 2 | 0 | 0 |
| G-0563 | `stale-claims` | yes | 0 | 4 | 3 |
| G-0564 | `stale-claims` | yes | 1 | 2 | 1 |
| G-0565 | `sound` | yes | 0 | 0 | 2 |
| G-0568 | `sound` | yes | 0 | 0 | 0 |
| G-0569 | `sound` | yes | 0 | 0 | 2 |
| G-0571 | `stale-claims` | yes | 1 | 3 | 2 |
| G-0572 | `stale-claims` | yes | 0 | 2 | 2 |
| G-0573 | `already-addressed` | partly | 3 | 3 | 1 |
| G-0574 | `stale-claims` | yes | 1 | 1 | 2 |
| G-0575 | `sound` | yes | 0 | 0 | 3 |
| G-0576 | `sound` | yes | 0 | 1 | 2 |
| G-0577 | `stale-claims` | partly | 1 | 2 | 1 |
| G-0578 | `already-addressed` | no | 2 | 0 | 0 |
| G-0579 | `stale-claims` | yes | 0 | 2 | 3 |
| G-0580 | `dead-premise` | partly | 1 | 3 | 0 |
| G-0581 | `stale-claims` | yes | 0 | 2 | 0 |
| G-0583 | `sound` | yes | 0 | 2 | 2 |
| G-0585 | `stale-claims` | yes | 0 | 1 | 3 |
| G-0586 | `stale-claims` | partly | 1 | 3 | 1 |
| G-0587 | `contradicted-by-code` | partly | 1 | 3 | 1 |
| G-0588 | `sound` | yes | 0 | 1 | 2 |
| G-0589 | `stale-claims` | partly | 1 | 4 | 1 |
| G-0590 | `stale-claims` | yes | 1 | 5 | 0 |
| G-0591 | `stale-claims` | yes | 0 | 2 | 2 |
| G-0594 | `stale-claims` | partly | 1 | 3 | 2 |
| G-0595 | `stale-claims` | yes | 0 | 3 | 2 |
| G-0600 | `stale-claims` | yes | 1 | 0 | 1 |
| G-0601 | `stale-claims` | yes | 1 | 2 | 1 |
| G-0602 | `sound` | yes | 0 | 0 | 3 |
| G-0603 | `sound` | yes | 0 | 0 | 1 |
| G-0604 | `stale-claims` | partly | 1 | 2 | 0 |
| G-0605 | `stale-claims` | yes | 0 | 2 | 1 |
| G-0606 | `stale-claims` | yes | 1 | 1 | 1 |
| G-0607 | `stale-claims` | yes | 0 | 2 | 1 |
| G-0608 | `stale-claims` | yes | 0 | 2 | 4 |
| G-0613 | `stale-claims` | yes | 1 | 1 | 2 |
| G-0618 | `stale-claims` | yes | 0 | 2 | 0 |
| G-0622 | `already-addressed` | no | 0 | 0 | 0 |

### High-severity findings

#### G-0060 · `dead-premise`

- **Claim:** *"'Patch' appears in the consumer-facing rituals (the optional `wf-rituals` plugin's TDD cycle / code-review / doc-lint surface) … The kernel says nothing about it."*
- **Measured:** `wf-rituals` is no longer an optional marketplace plugin, and patch is no longer a mention inside three other skills — it is its own 16-step ritual. ADR-0014 (`accepted`, added + promoted 2026-05-29) §1 rules the rituals are embedded in the binary and materialized by `aiwf init`/`update`. `<aiwf> update --help` offers no plugin-selection flag. `wf-patch/SKILL.md` was vendored into the embedded snapshot on 2026-05-29 (`8a2c8acdf`), and `.claude/skills/wf-patch` exists as a materialized artifact in this repo alongside eight other `wf-*` skills. G-0060's "What's missing" text predates all of it (last touched at the 2026-05-11 rename `9df927060`; the 2026-07-22 edit `765e8da78` appended the Investigation section without revisiting these bullets).
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0060)

#### G-0060 · `false-claim`

- **Claim:** *"No branch model. A patch is whatever-shaped work on whatever-shaped branch, with no kernel-level guidance."*
- **Measured:** The kernel carries a patch branch grammar in Go and a closed rung ladder that includes `patch` as a rung. `internal/branchparse/branchparse.go:30` is `^(?:epic/([Ee]-\d+)|milestone/([Mm]-\d+)|patch/([Gg]-\d+))(?:-|$)`, and `branchparse.go:127-132` declares `legalRungPairs` = `{trunk→epic, epic→milestone, milestone→patch, epic→patch}` with the comments *"milestone → patch — wf-patch under a milestone"* and *"epic → patch — wf-patch directly under an epic"*. Driven live: in a fixture, `<aiwf> authorize E-0001 --to ai/claude` succeeds from branch `patch/G-0001-fix-widget` and is refused from branch `randomthing` with *"current checkout \"randomthing\" does not match a ritual shape … (epic/E-NNNN-&lt;slug&gt; / milestone/M-NNNN-&lt;slug&gt; / patch/g-NNNN-&lt;slug&gt;)"*. ADR-0010 (`accepted`, 2026-05-11/12) line 54 carries the branch-namespace table row for `wf-patch`, and `wf-patch/SKILL.md` Constraints pins `patch/G-NNNN-<short-slug>`. `internal/cli/worktree/worktree.go:51` uses `aiwf worktree add patch/G-0100-fix` as the verb's own help example.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0060)

#### G-0060 · `false-claim`

- **Claim:** *"**`aiwf check` has no patch-shaped invariants to enforce.** Without a defined shape, `check` cannot say anything is wrong."*
- **Measured:** `aiwf check` treats `patch/` as one of exactly three ritual namespaces it polices. `internal/check/reflog_walk.go:168-175` (`ritualShape`) matches `epic/`, `milestone/`, `patch/`, and `listRitualHeads` (`reflog_walk.go:137`) filters the branch set it walks for force-pushed-away AI commits through it; `internal/cli/check/isolation_escape_oracle.go:319` names the same set. The consuming finding is `isolation-escape` (`internal/check/isolation_escape.go:34`, `codes.ClassBranchChoreography`), which fires when an AI actor's commit lands off its scope's recorded branch. So a patch branch has a defined shape *and* a check rule that judges commits on it. `internal/check/id_rename_untrailered.go:36` calls this "the branch-policing finding set alongside isolation-escape". Reading of source only — I did not stage a force-pushed orphan to drive the rule.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0060)

#### G-0060 · `false-claim`

- **Claim:** *"No relationship to gaps, ADRs, or contracts. A patch that closes a gap has no formal way to record `closes G-NNN` in a way `aiwf check` can validate."*
- **Measured:** The closure route exists, is verb-guarded, and *is* partly check-validated. Driven end-to-end in a fixture reproducing the `wf-patch` shape (branch `patch/G-0001-fix-thing` → commit → `git merge --no-ff` → `aiwf promote G-0001 addressed --by-commit <merge-sha>`): the promote succeeded and wrote `addressed_by_commit: [d61c164…]` into the gap's frontmatter. The verb refuses every degenerate form — no resolver → *"promoting a gap to \"addressed\" requires --by &lt;entity-id&gt; or --by-commit &lt;sha&gt; so the gap-addressed-has-resolver rule is satisfied"*; a fabricated SHA → *"does not resolve to a commit in this repo"*; a real-but-unreachable SHA → *"resolves to a commit not reachable from HEAD"*. On the check side, `gapAddressedHasResolver` (`internal/check/check.go:980-995`) fires at **warning** when an addressed gap carries neither resolver. **What is genuinely missing is narrower than the claim:** `aiwf check` validates only *presence*, not the value — a hand-edited `addressed_by_commit: [not-a-sha-at-all]` produced no finding at all in the fixture.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0060)

#### G-0068 · `contradicted-by-code`

- **Claim:** "Only `/ac` is a literal in source. The other six are derived via
- **Measured:** It is not. In a clone of HEAD I replaced the only haystack
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0068)

#### G-0068 · `stale-claims`

- **Claim:** "Discovered during M-0066/AC-6's RED-first sanity check: deleting the
- **Measured:** True when written, false now. At `56ad4b841^` the site read
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0068)

#### G-0070 · `dead-premise`

- **Claim:** "The `recommended-plugin-not-installed` finding-code string appears verbatim in
- **Measured:** The check does not exist, so nothing appears in the output. `$A doctor`
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0070)

#### G-0070 · `dead-premise`

- **Claim:** "When implemented, M-0070's AC-3 contract is automatically satisfied without
- **Measured:** It cannot be. `$A show M-0070` → `done`, archived, AC-3 "Each missing plugin
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0070)

#### G-0073 · `dead-premise`

- **Claim:** two of the five "concrete cross-kind cases the kernel can't represent today", plus one fix-shape arm, rest on ADR-0003 and the `finding` kind.
- **Measured:** ADR-0003 is **`rejected`** and archived (`docs/adr/archive/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md`), reversed by ADR-0045 (`accepted`, 2026-08-19). E-0019's *Dependencies* section — the very prose G-0073 cites — now reads *"**The F-NNNN storage model** ([ADR-0003]) — **withdrawn.** aiwf carries six entity kinds and `finding` is not one … The two implementation epics this list anticipated … neither was filed"*, and lists ADR-0004 as *"accepted, and the convention has shipped. **Met.**"*. So the bullet *"E-0019's Dependencies prose lists ADR-0003 and ADR-0004 as required"* is false on both halves — one is withdrawn, the other is met. The bullet *"**Implementation-epic chains.** Once ADR-0003 is ratified, an implementation epic for `finding` is filed"* names a future that has been ruled out. Fix-shape item 1's *"plus future finding"* and the predicate table's row *"| Finding (future) | `resolved` |"* are dead for the same reason.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0073)

#### G-0073 · `false-claim`

- **Measured:** **the template moved.** `db843cead` (2026-08-22, one day before HEAD, `aiwf-entity: G-0617`) deleted the line `depends_on: []           # optional: prior epic ids; e.g. [E-NNNN]` from `internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/epic-spec.md`. `grep -n depends_on` on that file returns nothing; the frontmatter block is now `id / title / status` plus a pointer comment to `aiwf schema epic`. G-0617 ("Template frontmatter restates each kind's vocabulary, unchecked") is `addressed` and archived. The gap's own stated settlement condition is met.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0073)

#### G-0073 · `false-claim`

- **Claim:** *"milestone→milestone edges are first-class (writer verb pending per G-0072)"*, and *"G-0072 stays open as the discovery record"*.
- **Measured:** the two specs **agree today**, and have since 17 seconds after this section was committed. G-0073's edit is `f202250c1` at `2026-08-12 13:13:41 +0000`; `9e9835ae1` (`aiwf edit-body E-0083`) landed at `2026-08-12 13:13:58 +0000` and replaced the quoted resolution path. E-0083's open-questions table row 120 now reads *"Shared with E-0084, which adds the membership rule ADR-0043 decides. Settled jointly before either epic's first milestone lands, not inside whichever starts first — ADR-0043 names the question and neither epic owns it alone."* E-0084's constraint (line 74) and table row 104 match. E-0083 additionally added a risk row (line 128) that **cites this gap by id** for the general point. So the *instance* is settled; the *argument it supports* — no `depends_on` edge expresses "neither of us lands before we settle this together" — survives, and is now sourced from E-0083 rather than from a disagreement that no longer exists.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0073)

#### G-0110 · `contradicted-by-code`

- **Measured:** false as stated. In a disposable single-package module at module
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0110)

#### G-0110 · `dead-premise`

- **Claim:** "With `--diff` broken for new files, the operator either runs the
- **Measured:** a working diff-scoped mutation run shipped. `make mutate-diff`
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0110)

#### G-0111 · `already-addressed`

- **Claim:** Concern 4 — "Wrap doesn't close the epic's named resolution gaps … The mechanism is missing on the epic surface." Restated in *Why it matters* as "**no mechanism to land the gap-status flips the epic claims to deliver**".
- **Measured:** The mechanism shipped on 2026-07-21 in `9cad2af3e` ("fix(ritual): close gaps a milestone's own prose claims to fix at wrap (G-0431)"). `aiwfx-wrap-epic/SKILL.md:24` now carries Precondition 6 — "Neither the epic's own spec nor any milestone's left a gap open that it explicitly claims to fix" — with the disposition rules at line 26; `aiwfx-wrap-milestone/SKILL.md:26` identifies the gaps at step 1 and line 241 runs `aiwf promote G-NNNN addressed --by-commit <sha>` at step 13, ahead of the milestone's own promote-done. The owning gap **G-0431** ("Milestone/epic wrap never closes gaps their own prose claims to fix") is `addressed`. This is precisely the "Skill-driven" option G-0111 itself enumerates; only the "Declarative" option (a `closes:`/`addressed_by:` field on the resolver's frontmatter) is unbuilt — `$A schema` shows milestone references are still only `parent` and `depends_on`.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0111)

#### G-0111 · `dead-path`

- **Claim:** Resolution path — "Cross-repo coupling pattern from M-0090 / M-0096 applies — author the skill body in `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md` and copy at wrap."
- **Measured:** `internal/policies/testdata` does not exist. The canonical authoring location is pinned as a Go constant: `internal/policies/aiwfx_wrap_epic_test.go:19` — `const aiwfxWrapEpicFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md"`, whose own doc comment (line 11-12) calls it "the canonical authoring location for the `aiwfx-wrap-epic` skill body — the embedded ritual snapshot". The "copy at wrap" half is dead too: ADR-0016 (`accepted`) archives the upstream rituals repo and ADR-0014 (`accepted`) makes the embedded snapshot the single source. A reader following this path writes into a directory that does not exist and looks for a repo that takes no changes.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0111)

#### G-0121 · `stale-claim`

- **Claim:** "G-0567 is one such disagreement, live and unfixed."
- **Measured:** `$A show G-0567` → "status: addressed · priority: high · archived", with
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0121)

#### G-0161 · `contradicted-by-code`

- **Claim:** "**ANTI-0007** (no kernel branch-of-verb rule): invoke a mutating
- **Measured:** the verb refuses. In the fixture, on `main`, with milestone
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0161)

#### G-0161 · `contradicted-by-code`

- **Claim:** "**ANTI-0012** (epic → active with zero milestones legal): promote an
- **Measured:** the finding **does** fire. After `$A add epic` + `$A promote
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0161)

#### G-0161 · `contradicted-by-code`

- **Claim:** "**ANTI-0001** (no `≥1 AC` requirement): create a milestone with 0
- **Measured:** two problems.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0161)

#### G-0168 · `contradicted-by-code`

- **Claim:** The table row `| milestone | `tdd:` | `aiwf add milestone --tdd …` | **none** |`,
- **Measured:** `aiwf milestone tdd <M-id> --policy none|advisory|required [--reason …]`
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0168)

#### G-0168 · `dead-advice`

- **Claim:** `## Workaround (current)` — "the operator hand-edits the YAML and commits
- **Measured:** That commit cannot be made in any repo with aiwf's hooks installed. The
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0168)

#### G-0168 · `dead-premise`

- **Claim:** `### Prerequisites for closing` — "**G-0285** (root `--help` banner drift) — the
- **Measured:** Both premises are false now, and both gaps are `addressed`. The root banner
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0168)

#### G-0168 · `already-addressed`

- **Claim:** `### The check-rule half has landed; a residual remains for the verb` — "The
- **Measured:** Shipped, exactly as specified. In the fixture, M-0002 (`tdd: none`) with
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0168)

#### G-0169 · `retired-verb-cited-present-tense`

- **Claim:** "**Mutating, bespoke output path:** `aiwf import` (multi-entity import) and
- **Measured:** Both halves are false at HEAD.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0169)

#### G-0169 · `contradicted-by-code`

- **Claim:** "A CI consumer scripting `aiwf import --format=json` or `aiwf render roadmap
- **Measured:** False for `import`, true for `render roadmap`.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0169)

#### G-0212 · `already-shipped`

- **Claim:** the six classes are catalogued "from history evidence +
- **Measured:** M-0243 "Named scenarios from G-0212 and G-0269" is `done` under
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0212)

#### G-0212 · `contradicted-by-code`

- **Measured:** the scenario built to test exactly this states the opposite as
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0212)

#### G-0212 · `contradicted-by-code`

- **Measured:** both halves are false. (a) repolock is an **inter-process**
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0212)

#### G-0217 · `false-claim`

- **Claim:** "The current message fires in BOTH cases with identical wording" — case 1
- **Measured:** An `in_progress` milestone whose branch is ahead of trunk emits **no
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0217)

#### G-0254 · `dead-premise`

- **Claim:** "**`aiwf check` rule (pre-push + CI)** — the authoritative
- **Measured:** **no workflow invokes `aiwf check`, and none ever has.**
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0254)

#### G-0282 · `dead-premise`

- **Claim:** the gated-annotation extension "is not independently buildable yet
- **Measured:** the verb exists. `aiwf milestone tdd --help` →
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0282)

#### G-0282 · `contradicted-by-record`

- **Claim:** the `required → advisory` downgrade is "exactly the
- **Measured:** G-0168 — the sibling gap this section itself names — settled the
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0282)

#### G-0282 · `retired-verb`

- **Measured:** `rewidth` no longer exists. `<audit>/aiwf rewidth --help` →
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0282)

#### G-0311 · `false-claim`

- **Claim:** *"the kernel forces it into three separate epics wired by `depends_on`, with no entity that names 'the subtitle feature.'"*
- **Measured:** epics cannot be wired by `depends_on` at all. `<aiwf> schema epic` prints `optional fields: (none)` / `reference fields: (none)`. `entity.ForwardRefs`' default arm (`internal/entity/refs.go:63-66`) states *"KindEpic and any future kind without outbound refs falls through to an empty list"*. In a fixture, an epic hand-carrying `depends_on: [ADR-0001]` drew **zero** findings and `<aiwf> show E-0001 --format=json` carried no `depends_on` key — accepted, stored, read by nothing. `<aiwf> epic depends-on E-0001 --on E-0002` → `aiwf: unknown command "epic" for "aiwf"`. The real fallback is unwired sibling epics plus prose, which is worse than the sentence claims — the gap's case survives and strengthens, but the sentence tells a reader a mechanism exists that does not.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0311)

#### G-0328 · `contradicted-by-code`

- **Claim:** "There is no **standing** test that reproduces the byte-identity
- **Measured:** three standing, untagged tests do exactly this, in
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0328)

#### G-0333 · `false-absence-claim`

- **Claim:** The Tier-1/Tier-2 boundary "is stated in no AI-discoverable channel
- **Measured:** False, and false at the moment of filing. Six surfaces carry it;
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0333)

#### G-0370 · `dead-premise`

- **Claim:** "Needs a hand-written pinning test under `internal/policies/` (the
- **Measured:** The named gate no longer exists.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0370)

#### G-0398 · `dead-premise`

- **Claim:** "Discovered indirectly: `internal/stresstest/verb_sequence.go`'s
- **Measured:** The ordering was inverted four days after this gap was written. `Run` in
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0398)

#### G-0412 · `stale-claim`

- **Claim:** the wording "has since been copied verbatim into `internal/cli/renamearea`,
- **Measured:** 11 of the 12 named files carry neither the string nor a `ResolveRoot(` call
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0412)

#### G-0442 · `contradicted-by-code`

- **Claim:** "There is no verb to **amend, add to, or clear** either field afterward
- **Measured:** three of the four sub-claims are false.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0442)

#### G-0444 · `contradicted-by-code`

- **Claim:** "`readHistoryChain` survives only in test comments under `internal/cli/integration/` (e.g. `canonicalize_history_test.go`, `show_cmd_test.go`), **not in production code**; the chain logic now lives inline in `history.go`." Restated in "Why it matters" ("the chain logic is inline") and made operative in "Suggested approach" ("describe the inline `PriorIDs`-chain handling rather than the **retired** `readHistoryChain`").
- **Measured:** `internal/entityview/historyevent.go:117` declares `func ReadHistoryChain(ctx context.Context, root string, chain []string) ([]HistoryEvent, error)` — production code, not a test file — and `internal/cli/history/history.go:100` calls it (`events, err := entityview.ReadHistoryChain(ctx, rootDir, chain)`). It landed in `5d331e61d` (2026-07-20, "feat(entityview): extract read-side helpers into a neutral package (M-0272)"), three days before G-0444 was filed (`beb1bc399`, 2026-07-23), so the claim was already false at filing. What *is* inline in `Run` is the chain **assembly** (lines 85–98: seed with the queried id, resolve via `ResolveByCurrentOrPriorID`, append `PriorIDs` and the canonical id) — not the "greps the union in one pass" step that `id-allocation.md:164` attributes to `readHistoryChain`, which is precisely what `ReadHistoryChain` still does. The doc line's *description* is therefore still accurate; only the spelling and the owning package moved.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0444)

#### G-0445 · `contradicted-by-code`

- **Claim:** "`docs/` is not an aiwf-managed path: in a consumer repo `docs/` may hold real shippable implementation."
- **Measured:** `docs/adr/` is an aiwf-managed path in *every* consumer. `$A --root <fx> add adr --title "Some architectural choice" --body …` created `docs/adr/ADR-0001-some-architectural-choice.md` and committed it (`git show --stat` names that single file). The loader walks `docs/adr` (`internal/tree/tree.go:288`), `aiwf add` routes ADRs there (`internal/verb/add.go:490`), `aiwf init` creates it (`internal/initrepo/initrepo.go:686`), `aiwf archive` sweeps to `docs/adr/archive/` (`internal/verb/archive.go:428`), and `internal/trunk/trunk.go:88,239` reads trunk state from `"work/"` **and** `"docs/adr/"`. The consequence is load-bearing for the gap's own remedy list: I measured that a dirty `docs/adr/ADR-0001-….md` alongside a dirty test path passes `--phase red` today (exit 0), and that any *non-excluded* dirty non-test path refuses (control: `src/component.jsx` → "refusing — red-first requires the test to change before the implementation … : src/component.jsx"). `isPlanningPath` reaches `docs/adr/` only via the `docs/` prefix (4-line function, read at `promote_phase_gate.go:78-85`), so the gap's option 2 — exclude `work/` only — would make every mid-edit ADR false-refuse a legitimate red promote.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0445)

#### G-0448 · `contradicted-by-code`

- **Claim:** "split by an accident of function signature rather than a declared boundary" / "Whether a rule lands in surface one or surface two is decided purely by 'does it need git/ctx/config' — not by any intentional layering."
- **Measured:** the boundary is declared in three independent places and is load-bearing.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0448)

#### G-0448 · `stale-count`

- **Claim:** "Rules are dispatched from two parallel surfaces" (also in the title), with the git/history/area rules "individually invoked and appended in `internal/cli/check/check.go` (~lines 127-305)".
- **Measured:** four rule-invoking sites, and the largest git/history group is in a file the gap never names.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0448)

#### G-0472 · `dead-premise`

- **Claim:** "The hook installers carry a cost of a different kind, and it is the
- **Measured:** All four installers now refuse. `ensurePreHook:1350` has
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0472)

#### G-0478 · `stale-claim`

- **Claim:** Two links in `docs/initiatives/quality-signal-and-cadence.md` *still* name
- **Measured:** False now, and false when the sentence was last revised. Both links point at
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0478)

#### G-0478 · `dead-premise`

- **Claim:** The recommended first move is "**Detection** … a check rule that resolves every
- **Measured:** An accepted ADR declines exactly that, and a prior gap proposing it was
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0478)

#### G-0504 · `dead-premise`

- **Claim:** "G-0471 — the binary-versus-source staleness axis, addressed by
- **Measured:** E-0076 is `status: cancelled`, archived — `aiwf cancel E-0076 ->
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0504)

#### G-0508 · `undercount`

- **Claim:** "Four policies scan `internal/verb` for package-level function
- **Measured:** **Eight** files in `internal/policies/` carry that walk. The four
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0508)

#### G-0514 · `dead-premise`

- **Claim:** "Measured on the shipped tree, these currently fire under that message" — followed
- **Measured:** `skill-body-id` fires on **nothing** in the tree today — `aiwf check` reports
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0514)

#### G-0514 · `dead-premise`

- **Claim:** "This lands on the sweep as concrete friction. A sweep driven by the rule's output
- **Measured:** The sweep is M-0288 (`Sweep shipped surfaces to canonical placeholders and
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0514)

#### G-0517 · `false-premise`

- **Claim:** "These are mostly citations of entities that were genuinely real
- **Measured:** Of the 113 raw narrow-id occurrences in the three surfaces, 71
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0517)

#### G-0523 · `overclaimed-attribution`

- **Claim:** "The events are already wired, so this needs no consent surface
- **Measured:** Both halves are false.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0523)

#### G-0527 · `contradicted-by-code`

- **Claim:** "Asking for the verb that does not exist reports success: `aiwf worktree remove
- **Measured:** False for all five cases. `aiwf worktree remove /tmp/nonexistent-abc-zzz` printed
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0527)

#### G-0527 · `claim-about-what-does-not-exist`

- **Claim:** "Nothing removes one: teardown is `git worktree remove` plus `git branch -d`,
- **Measured:** False, and false at authoring time. Four surfaces name it, three of them shipped:
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0527)

#### G-0529 · `contradicted-by-record`

- **Claim:** "The failure is not hypothetical. E-0075 wrapped with an entry that
- **Measured:** Wrong on all three particulars, and the timeline runs the other
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0529)

#### G-0530 · `already-addressed`

- **Claim:** "The classification of `entity-body-empty` in the shipped-surface table of
- **Measured:** The correction landed on 2026-08-04, 14 hours after the gap was filed, and it
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0530)

#### G-0530 · `count-does-not-rederive`

- **Claim:** "`## Work log` is the sharpest case — the wrap ritual mandates one entry per
- **Measured:** The wrap-ritual half re-derives exactly (see below). The emptiness half does not,
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0530)

#### G-0530 · `dead-premise`

- **Claim:** "Gap is the largest population and has no template at all, so its structure is
- **Measured:** First half true — 620 gaps vs 317 milestones, 88 epics, 75 decisions, 46 ADRs,
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0530)

#### G-0535 · `dead-premise`

- **Claim:** Option 3 — "**Give each an anti-orphan assertion**, as the sovereign
- **Measured:** The sovereign policy no longer exists. `internal/policies/sovereign.go`,
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0535)

#### G-0535 · `dead-premise`

- **Claim:** "That is what separates these from G-0534, where `internal/verb`
- **Measured:** G-0534 is `addressed` and archived; its redundancy question was
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0535)

#### G-0536 · `dead-premise`

- **Claim:** "On the tree as it stands the step reports errors on day one. Which
- **Measured:** false in both the strictest and the realistic reproduction.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0536)

#### G-0539 · `contradicted-by-code`

- **Claim:** "The pre-commit half prevents rather than detects, and needs one distinction to be safe: a verb's own commit also stages entity files. The verbs drive git themselves, so they can mark their own commits — an environment variable set across the commit call is the cheap version, with the hook skipping when it is present." And: "the second and third are verbs, covered by the same marker" (of `aiwf archive`'s sweep and `aiwf reallocate`'s rewrite).
- **Measured:** aiwf verb commits fire **no git hooks whatsoever** and stage nothing in the index, so a pre-commit rule reading `git diff --cached` would never see a verb's commit and needs no marker or exemption. In the fixture, with a logging `pre-commit.local` and `commit-msg.local` installed, five verb invocations produced five commits and zero hook firings; the plain `git commit` control fired both hooks.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0539)

#### G-0555 · `contradicted-by-measurement`

- **Claim:** an age-based sweep cannot reclaim this, therefore only a code fix can — "a sweep
- **Measured:** a 24-hour-margin sweep run right now would delete **1367 directories holding
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0555)

#### G-0560 · `dead-premise`

- **Claim:** "G-0559 gates it — the strings originate in `internal/entity/entity.go` and
- **Measured:** G-0559 is `status: addressed`, `addressed_by_commit: b778bc9f32ad…`
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0560)

#### G-0560 · `dead-premise`

- **Claim:** "**ADR-0003 is `accepted` and unimplemented** — an accepted decision to add a
- **Measured:** `ADR-0003 | rejected | Add finding (F-NNN) as a seventh entity kind |
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0560)

#### G-0562 · `already-fixed`

- **Claim:** "`writeWorktreeHookScript` in `internal/policies/worktree_rituals_check_hook_test.go`
- **Measured:** false at HEAD. `internal/policies/worktree_rituals_check_hook_test.go:28-34`
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0562)

#### G-0562 · `duplicate`

- **Claim:** (implicit) G-0562 and G-0578 are separate gaps.
- **Measured:** **They are duplicates — same file, same helper, same single call site.**
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0562)

#### G-0564 · `dead-premise`

- **Claim:** "Two of them — composition tests across verb chains, and tree-level
- **Measured:** `E-0080` status is **`cancelled`**, path
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0564)

#### G-0571 · `superseded-premise`

- **Claim:** "The narrower option is a create-time refusal on `aiwf add --body-file` and `aiwf edit-body --body-file`, which fires only on new content and would raise none." — presented as an open option.
- **Measured:** `ADR-0043` (status `accepted`, dated 2026-08-11, i.e. after this body's last edit on 2026-08-08) has already settled the question, and settled it differently and more broadly. Its Decision section reads: *"Body-section membership is enforced at the seams where body bytes are written, and nowhere else."* — **Seam one** is *"A scan over the bytes a verb is about to write, called by every body-supplying verb, refusing the write at error severity for every kind"* (every body-supplying verb, not the two `--body-file` paths), and **Seam two** is *"A gate riding the commit range the provenance audit already resolves, scoped to entities whose body content differs between the range base and HEAD."* `E-0084` ("Enforce body-section membership at the write seams", `proposed`) implements it and names `- Closing G-0571.` The body mentions neither record. A reader acting on the gap's sentence would design a narrower gate than the one already ratified.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0571)

#### G-0573 · `contradicted-by-code`

- **Claim:** title, plus "It calls `check.Run` directly, and applies none of the four
- **Measured:** the guard applies all four. `internal/verb/common.go:143-148`:
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0573)

#### G-0573 · `contradicted-by-code`

- **Claim:** "So a knob that escalates a finding to error severity is invisible to every
- **Measured:** false — falsified against the binary. In a fixture with
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0573)

#### G-0573 · `dead-premise`

- **Claim:** implied throughout — that the surviving symptom is a live, fixable defect
- **Measured:** ADR-0042 (`Retire tdd.strict; require a complete body at the readiness
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0573)

#### G-0574 · `overclaimed-absence`

- **Claim:** "What is missing is a decided answer to what makes two findings the same finding for the purpose of the guard … That decision is the prerequisite for both repairs above, and it is currently unwritten."
- **Measured:** `D-0046` — *Diff the shared contract gate by finding identity, not the full struct* — exists, status `accepted`, decided 2026-07-21 (i.e. 18 days before G-0574 was filed on 2026-08-08). It decides exactly this question for `internal/verb/contractgate.go`, names the identity subset (`Code`, `Severity`, `EntityID`, `Subcode`, `Path`), and excludes `Message` for precisely the reason G-0574 describes (a message that interpolates positional/derived state makes an unchanged finding read as introduced). Its **Consequences** section is not scoped to the contract gate: *"Any future consumer that diffs two `contractcheck.Run` (or similarly shaped) result sets for equality should treat `Message` (and any other `Run()`-computed prose field) as unsafe for an equality/identity key across two separate invocations whose input differs by more than the field under test."* `projectionFindings` is a similarly-shaped consumer of `check.Run`. The claim is not merely unsupported — a written decision points the opposite way and the body does not mention it.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0574)

#### G-0577 · `contradicted-by-code`

- **Claim:** "…`IsTerminalACStatus` drives the AC cancel path's convergence guard, so flipping a terminal silently changes what `aiwf cancel` does with no test to catch it."
- **Measured:** Two tests catch it, in two packages. Patching `acTransitions["deferred"]` from `{}` to `{"open"}` in a clone: `TestCancelAC_TerminalStatus_ReturnsNoOp/deferred` fails at `ac_same_state_noop_test.go:48` with *"fixture assumes \"deferred\" is terminal in the AC FSM; it is not"*, and `TestIsLegalACTransition_AllPairs/deferred->open` fails at `transition_test.go:196` with *"IsLegalACTransition(\"deferred\", \"open\") = true, want false"*. Nothing about it is silent. (Nuance worth keeping: the first is a fixture-precondition `t.Fatalf`, so it fails *before* exercising `aiwf cancel`'s behaviour — but it fails CI, which is what "no test to catch it" denies.) The half of the sentence that is true: `IsTerminalACStatus` does drive `cancelAC`'s convergence guard — `internal/verb/ac.go:279`.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0577)

#### G-0578 · `already-addressed`

- **Claim:** "`internal/policies/worktree_rituals_check_hook_test.go` writes the hook
- **Measured:** False at HEAD. The file's only executable write is line 31,
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0578)

#### G-0578 · `duplicate`

- **Claim:** implicit — the gap is filed as a fresh finding and references no sibling.
- **Measured:** G-0578 and G-0562 name **the same single call site**, not two.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0578)

#### G-0580 · `dead-premise`

- **Claim:** "The skill-edit **structural-test** backstop fails the profile-driven
- **Measured:** that predicate no longer exists. Commit `667c8e0fc`
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0580)

#### G-0586 · `contradicted-by-code`

- **Claim:** "It directs the reader to `aiwf show`, and no entity template carries a section in which an unsettled claim would appear."
- **Measured:** False, twice over. (a) The shipped epic template `internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/epic-spec.md` carries `## Open questions` — a table with columns `Question | Blocking? | Resolution path`. It has been there since `8a2c8acdf` (2026-05-29), i.e. before this gap was filed on 2026-08-15, so this is not drift. 54 entity files in `work/` currently carry a `## Open questions` section. (b) `aiwf show` does surface it: `aiwf show E-0086 --format=json` returns body key `open_questions`. The milestone template additionally carries `## Deferrals` ("Work this milestone deliberately punted") and `## Reviewer notes`, both surfaced by `aiwf show M-0312 --format=json` as `deferrals` / `reviewer_notes`. This is the load-bearing sentence for "the pointer does not close the hole either", and it does not hold.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0586)

#### G-0587 · `contradicted-by-code`

- **Claim:** the title and the second paragraph — a shipped skill instructing a
- **Measured:** shipped surfaces already name exactly that corpus by path, ship, and pass.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0587)

#### G-0589 · `dead-premise`

- **Claim:** "It will fork a third time at implementation. A shipped review pass has to
- **Measured:** No review pass will ship. `D-0069` — *Reject the dispatched reading
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0589)

#### G-0590 · `false-claim`

- **Claim:** The HTML render path carries the cancel reason too, so `show` is the only surface that omits it — the field is "already rendered by two other surfaces".
- **Measured:** The HTML render omits it entirely. `htmlrender.HistoryRow` has no `Body` field at all; `internal/cli/render/resolver.go:666` populates `Reason: e.Reason` (the `aiwf-reason:` trailer, which no cancel commit carries — see the next finding), and no template renders even that: `epic.tmpl:80-93` renders a four-column Date/Verb/Detail/Actor table, and `milestone.tmpl:174-184` renders Date/Verb/Actor plus force/audit/scope chips. Rendering the site and grepping it confirms: the E-0058 cancel reason appears in no rendered page except `G-0590.html` (this gap's own body) and `ADR-0031.html`. Two surfaces *do* carry it — `aiwf history` text and the JSON envelopes — but not the one the body names.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0590)

#### G-0594 · `overclaimed-attribution`

- **Claim:** "A shipped skill defined the reading state *ambiguous* as the
- **Measured:** "the specification" resolves to
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0594)

#### G-0600 · `contradicted-by-code`

- **Claim:** "Stamping is uneven. **Only the guidance fragment carries a version today**, so a
- **Measured:** Both paragraphs are answered by shipped code the gap does not mention.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0600)

#### G-0601 · `contradicted-by-code`

- **Claim:** "`aiwf history <id>` renders a row only for a commit carrying `aiwf-verb`." (opening sentence) and "Every row that projection does render carries a verb." (end of §What's missing)
- **Measured:** The projection's drop rule is `verb == "" && actor == ""` — a commit carrying `aiwf-entity` + `aiwf-actor` and **no** `aiwf-verb` renders, with a blank verb column. 19 such commits exist on `main`. `aiwf history M-0158` renders one of them (`1951f02`, subject `feat(spec): M-0158/AC-1+AC-7+AC-8 — scaffold layer-4 branch package`) with an empty verb field. Both claims are false, and were false when the gap was filed (`internal/entityview/` has no commit in `667c8e0f..main`).
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0601)

#### G-0604 · `contradicted-by-code`

- **Claim:** "Each spells the same walk — resolve by current id, fall back to prior ids, then scan the stub list."
- **Measured:** false for every site except the backstop. `internal/check` consults `prior_ids` nowhere in these walks — `grep` for `PriorIDs`/`ByPriorID`/`ResolveByCurrentOrPriorID` in non-test Go finds the prior-id arm only in `internal/tree/tree.go`, `internal/cli/history/history.go:87`, `internal/policies/skill_edit_provenance_backstop.go:169`, and (through `resolveViaPriorIDs`) `internal/check/provenance.go` + `promote_on_wrong_branch.go` — none of which is one of the cited copies. Confirmed behaviourally in a fixture: a `relates_to: D-0009` where `D-0009` lives only in another entity's `prior_ids:` yields `refs-resolve` / subcode `unresolved`, severity **error** — i.e. `refsResolve` does not fall back to prior ids. The same fixture with `D-0009` in body prose yields `body-prose-id` / `unresolved`, severity **error** — `BodyProseIDIndex` does not either. Expected, had the body been true: both silent.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0604)

#### G-0606 · `false-count`

- **Claim:** "The live instance is `m0211-guidance-operating-anchors` …" and "The cost of
- **Measured:** a second live instance exists and predates the first.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0606)

#### G-0613 · `contradicted-by-code`

- **Claim:** "The wrap ritual is now the only surface naming a category set, so whichever
- **Measured:** Two shipped surfaces name a closed category set, not one.
- **Evidence:** [full finding](gap-truth-audit-evidence.md#g-0613)


## `TODO.md`'s own accuracy

Every gap's `TODO.md` line was judged independently of its body. **82 measured
accurate, 58 drifted.**

Two drifted in the *useful* direction — the line is right where the body is wrong,
so the body is what needs the edit: **G-0536** (the line already says "Unblocked —
G-0556 landed"; the body still claims the step reports errors on day one) and
**G-0568** (the line is stale in promising a severity field the body explicitly
rejects — the inverse case) and **G-0444** and **G-0586**, whose lines are more
accurate than their own bodies.

The file's own structural claims:

| claim | measured |
|---|---|
| cluster 8: "Six gaps on one surface" | **seven** entries — and it was seven before the last refresh too, so it never re-derived |
| cluster 6: "Done-condition is mechanical: zero OPEN entries in the NoOp allowlist" | the allowlist has 12 entries, 6 OPEN, naming G-0458/59/60 — **G-0461 appears nowhere in `internal/`**, and no test reads the `Reason` strings, so the condition is neither complete nor mechanical |
| cluster 12: "Neither member is a kernel concern" | the cluster has **five** members, and G-0554's body says "This one is a defect in shipped content" |
| preamble: "Clusters 1–9 cover the defect surface. Cluster 10 covers absences" | there are **twelve** clusters; 11 and 12 are named by neither arm |
| cluster 1: "The largest cluster" | re-derives — 24 against 19 and 18 |
| cluster 9: "about one finding per 200 commits" | re-derives — it is G-0533's own honest-yield sentence, not its raw block count |
| in-flight header (six claims about E-0088) | all six re-derive |

Two lines contradict each other inside one cluster: cluster 3's **G-0564** entry
treats E-0080 as live scoping while its **G-0121** entry four lines later says it
is cancelled.

Cluster placement drifted for three members: **G-0439** (a behavioural mover gap
with an ADR collision and an in-flight milestone, filed under "citations that no
longer resolve"), **G-0519** and **G-0548** (missing chokepoints, not stale
citations — G-0519's body has a section headed *"Why this is larger than it
looks"*), and **G-0572** (no laundered bytes, no hand-edit, no guard — it belongs
with G-0121's composition axis).

Two members of cluster 4 — "Cheap fixes, batchable now — Small, no design content
between them" — are neither: **G-0493** carries three mutually-exclusive resolution
routes, and **G-0622** is already fixed on a branch.

The unfiled entry on spec adequacy is stale in a way worth naming: it says "three
captures already circle this … before opening anything". Something *was* opened —
`wf-measure-spec` shipped 2026-08-14 and was withdrawn the next day ("it
contradicts the specification it was built from"); E-0085 and M-0309 are cancelled.
`TODO.md` names none of E-0085, M-0308, M-0309, `wf-measure-spec`, D-0066, or the
initiative doc that replaced them.

## What no measurement here settles

- Whether these rules would hold for an author who did not write them. Every
  breach measured in §2 is by the same small set of authors, as G-0594 already
  says of its own evidence.
- Whether an independent pass is the only thing that finds this class. It is what
  found it three times now — but no run has tried and failed by another method.
- Whether the backlog's growth is a problem or a designed consequence of "a
  confirmed defect leaves behind a check … never a silent correction". This audit
  measures decay, not volume.

- Whether the cause classification is right at its boundaries. The four classifiers
  read ledgers rather than re-measuring the tree, and drew the wrong-at-birth /
  scope-overclaim / convention lines differently — which is why their
  "false when written" totals span 60% to 78% while their drift figures agree within
  eight points. The drift share is robust; the split *inside* the remainder is not.
- Whether any of this generalises past the authors who wrote it. Every breach
  measured here is by the same small set, as G-0594 already says of its own evidence.

Claims individual auditors could not settle are recorded as **unverifiable** in
their batch ledgers, each with the command that would settle it, rather than
asserted here.

## Provenance

Audited 2026-08-23 against the working tree at `da34c1009`, with `TODO.md` read in
its uncommitted working-tree state. 48 independent auditors over 48 batches, each
working read-only against a binary built from that tree. No fixes were applied in
the same pass, so every citation reflects the tree as read; `HEAD` did not move and
the working tree ended byte-identical to how it started.

A second pass over the completed ledgers assigned every finding a cause, four
independent classifiers over one quarter each, settling authoring-order questions
with read-only `git log` against the same tree. §"What causes this" and §"What would
prevent it" are its output; the prevention items are scored against this corpus and
carry no claim about any other.

Per-batch ledgers carrying the full evidence — every finding's command, expected
result, observed output, and quoted fragment, plus the unverifiable claims — are
21,926 lines across 48 files and are not reproduced here.
