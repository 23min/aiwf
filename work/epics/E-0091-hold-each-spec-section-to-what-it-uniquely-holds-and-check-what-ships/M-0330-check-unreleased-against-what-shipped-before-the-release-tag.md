---
id: M-0330
title: Check Unreleased against what shipped before the release tag
status: in_progress
parent: E-0091
tdd: required
acs:
    - id: AC-1
      title: The audit reports a shipped-surface delta nothing under Unreleased cites
      status: met
      tdd_phase: done
    - id: AC-2
      title: A shipped-surface commit with no entity trailer is reported, not skipped
      status: met
      tdd_phase: done
    - id: AC-3
      title: The base release is a tag reachable from the commit under test
      status: met
      tdd_phase: done
    - id: AC-4
      title: The release tag workflow invokes the audit and the target it names exists
      status: met
      tdd_phase: done
    - id: AC-5
      title: The audit reports an uncited delta in the shape a release commit presents
      status: met
      tdd_phase: done
---
## Goal

Make a release name everything it ships. An audit compares the commits that
changed embedded content since the last release against what the release notes
cite, and the release tag's CI job fails when something is uncited. Which notes
it reads depends on where it runs: `[Unreleased]` before the release commit,
and the version's own section once that commit has moved the entries down.

## Closes

- G-0529 — CHANGELOG completeness rests on recall at epic wrap and is never
  checked.

The gap's direction names two properties and this milestone delivers one of
them over one surface class: a shipped delta named under `[Unreleased]` before
a release, enumerated over the embedded trees. That is the class the v0.34.0
omissions came from, and the gap records that its own named-surface list —
finding codes, verbs, config keys, exit codes — would not have reached them.
The named-surface half stays undone, and it is the half that would have caught
the earlier thin entry, so it is a residual rather than a subsumed alternative.
It takes its own gap at wrap.

The per-epic citation property falls out of the same comparison wherever an
epic's work touched a shipped surface. An epic that shipped only compiled
behaviour is outside what this milestone sees, and rides the same residual.

## Context

The `[Unreleased]` section is written at two moments: a patch's own wrap, and an
epic's wrap. Nothing verifies the result. The tag workflow fires on a pushed
`v*` tag and confirms a `## [X.Y.Z]` heading exists, which a stub satisfies.

Measured 2026-09-09 on this branch: 18 non-merge commits have touched
`internal/skills/` since v0.34.0, and every one carries an `aiwf-entity` trailer.
Rolling each milestone up to its parent epic leaves five entities owing an entry,
of which `[Unreleased]` names three. The two it does not name are E-0091, whose
milestone deltas await its wrap entry, and G-0659, a patch that changed a shipped
ritual and wrote no line although the patch ritual mandates one with no skip.

That trailer is what makes the property computable: a shipped-surface change
already names the entity that owns it. It is guaranteed only for the ritual
skill files under `internal/skills/embedded-rituals`, though — not for the verb
skills, the templates, the agent cards or the guidance fragment. Elsewhere it is
habit, and the habit is one release cycle old: across the three ranges that
already shipped, the untrailered share of shipped-surface commits ran 20 of 20
into v0.32.0, then 38 of 52, then 2 of 11, against 0 of 18 on the current range.
An audit that trusts the trailer has to say when it is absent.

Backtested over those same three ranges, the comparison reports two uncited
entities into v0.33.0 and none into the other two. One of the two is documented:
`[0.33.0]` carries an entry for G-0635's change that names no id. So what the
audit proves is that nothing under the section cites the entity, which is a rule
it imposes rather than an omission it observes — an entry describing a change
while naming no id is indistinguishable from no entry at all. D-0087 settles
what each finding then does: the uncited entity fails the release, the
untrailered commit is reported and does not.

The backlog is two entities and both clear before the audit can block: E-0091's
at its own wrap, G-0659's as a changelog line this milestone writes.

## Acceptance criteria

### AC-1 — The audit reports a shipped-surface delta nothing under Unreleased cites

The report names the entity nothing cites and at least one commit behind it, and
it fails the release. Naming the entity is what the test asserts, not the exit
code alone: an unreadable changelog and an unresolvable base ref both fail on
this path already, so an exit-code assertion passes with the comparison absent.

A milestone rolls up to its parent epic, and the epic is what must be cited — a
milestone's user-visible delta lands in its epic's entry, never its own. The
rollup resolves through the tree loader, which reaches archived entities. A path
scan does not, and the failure is silent rather than loud: an archived
milestone's id survives the rollup and is reported as an uncited entity in its
own right. A composite trailer value resolves to its milestone before the
rollup, and ids compare canonicalized, since a narrower legacy width names the
same entity.

Merge commits are excluded. Their file lists carry the merged branch's changes,
already attributed to the commits that made them, so counting them attributes
one delta twice.

Evidence: a fixture repo whose range carries one shipped-surface commit under an
entity the section does not cite, asserted to report that entity; the same range
with the entity cited, asserted to report nothing. The rollup runs against an
archived milestone, which is the reading a path scan gets wrong; the
narrow-width reading gets a range that turns on nothing else.

What turns that report into a failed release is the shared policy harness, which
fails one test per violation, and the wiring that reaches the harness from the
release tag. Neither is asserted here: the harness is pinned by every policy in
the suite, and the wiring is AC-4's.

### AC-2 — A shipped-surface commit with no entity trailer is reported, not skipped

An untrailered shipped-surface commit is reported by subject, at a severity that
does not fail the release. Skipping it is the failure mode the audit is most
likely to have: the comparison is driven by trailers, so a commit carrying none
contributes to neither side and passes silently.

The trailer is guaranteed on one tree only. The provenance backstop covers the
ritual skill files; the verb skills, the templates, the agent cards and the
guidance fragment carry a trailer by habit. An audit that trusts the trailer
everywhere under-reports exactly where the guarantee stops, and says nothing
while it does.

It reports rather than blocks because it establishes nothing about the
changelog: the audit cannot attribute the commit, so it cannot say whether an
entry covers it, and the repair a failing release would imply — rewriting a
landed commit's trailers — is not available. D-0087 carries the argument.

Evidence: a range carrying one shipped-surface commit with no entity trailer and
nothing else uncited, asserted to report that commit and to produce no
release-failing finding. Both halves are asserted, because each fails a
different wrong implementation: one that treats an absent trailer as nothing to
attribute drops the report, and one that routes it to the blocking half instead.
The second assertion runs through the release-gating entry point rather than the
renderer beneath it, since a leak into the blocking half leaves the renderer
untouched. Measured on the current range the tree offers no such commit, so the
fixture builds one rather than reading history.

### AC-3 — The base release is a tag reachable from the commit under test

The range the audit reads starts at a tag reachable from the commit under
test. A tag on a branch that commit cannot reach is not its base, and choosing
it compares the change against a release that never contained it.

Two things narrow that further, both stated with the base resolution in AC-5.
A tag pointing at the commit under test is excluded, since it would otherwise
be its own base and the range would be empty. And `git describe` answers with
the *nearest* reachable tag by history rather than the newest by date — the
same answer on a linear trunk, and the safer one where the two differ.

This is where a branch and trunk part. The audit runs on a branch in practice —
the local gate runs before the merge — while on trunk the newest tag and the
newest reachable tag are usually the same commit. A trunk-only test therefore
passes against a resolution that reads the tag list and sorts it.

Evidence: one fixture repo carrying a tag on a branch the commit under test
cannot reach, asserted to resolve to the reachable tag instead. A repo with no
tag at all resolves to the root commit rather than failing, since a first
release has no predecessor.

### AC-4 — The release tag workflow invokes the audit and the target it names exists

The workflow that fires on a release tag runs the audit, and the target it names
exists. Both halves, because either alone is satisfied by a broken wiring: a
step naming a target the Makefile does not define fails only when someone cuts a
release, and a defined target nothing invokes is a command that never runs.

The claim is scoped to the wiring, not to the outcome. Whether the job fails a
release whose notes are incomplete cannot be asserted without running the
workflow, which the test suite does not reach. What it can assert is that the
invocation resolves, so renaming either side reports.

Four links, not two. The workflow step names a target, the target runs a test by
name, that test is the audit's release-gate entry point, and the job it runs in
checks out the history the base resolution needs. A chain asserted only at its
ends stays green while its middle names a test that no longer exists — a break
that surfaces when someone cuts a release and nowhere earlier.

The fourth link is invisible to the other three and the reason they are not
enough. The runner's checkout is shallow by default and fetches no tags, so a
job taking that default resolves its base against history it does not have. The
audit then compares the release against the wrong range and reports whatever
that range happens to contain, while every other assertion here stays green.

Evidence: the recipe resolved by running make rather than by reading the
Makefile, so renaming an intermediate target keeps it green and only the audit
dropping out turns it red; the workflow's step asserted to invoke that target;
the resolved recipe asserted to name the audit's entry-point test and to pass a
base; and the job running that step asserted to check out full history. Each
link cut in turn, and the chain reports every time.

### AC-5 — The audit reports an uncited delta in the shape a release commit presents

At the pushed tag the audit reads a different pair of inputs than it does
before the release commit, and reaches the same verdict. The base excludes any
tag pointing at the commit under test, and the section read becomes the version
being released rather than `[Unreleased]`.

Which notes those are comes from the changelog rather than from the tag: they
are every section stacked above the one naming the base release. The range is
`base..HEAD`, so the notes describing it are everything written since that
release, however many sections that spans.

Reading a single section cannot serve this. A range crossing a release has its
entries split — the released ones under their version heading, the rest still
accumulating under `[Unreleased]` — and reading either alone reports the
other's entities as uncited. Measured on this repo, a rule that read one
section reported four entities whose entries sit in `[Unreleased]`, one of them
the entry this milestone itself wrote.

Taking the notes above the base also removes the tag from the question, which
is what makes the answer the same either side of the tag push. The release
commit moves entries under a new version heading and the tag arrives after;
both states put those entries above the base's heading. Where no section names
the base — an explicit SHA, or the root-commit fallback in a repo with no tags
— the whole file is read: the audit cannot bound the notes, and reading
everything can only make it miss a real omission, where reading too little
invents ones.

Both halves are needed, and each hides the other. Resolved to the tag at HEAD
the range is empty, so the audit reports nothing whatever the notes say.
Resolved to the previous tag while still reading `[Unreleased]`, the section is
the empty one the release commit just opened, so every entity reports uncited.
Measured on this repo at v0.34.0: `git describe --tags --abbrev=0` returns
v0.34.0, and `[Unreleased]` at that commit holds nothing between its heading
and the release heading below it.

The pre-release reading is unchanged and has to stay so. An operator running
the target before cutting the release compares the newest reachable tag against
`[Unreleased]`, which is what makes the target useful at the moment the notes
are still being written.

AC-4 scopes itself to the wiring and is true as written: the workflow does
invoke the target and the target does exist. The outcome it declines to judge
is this criterion's, and it is reachable by a fixture — the workflow need not
run for the audit's verdict in the release shape to be asserted.

Evidence: one fixture carrying the release shape this project documents —
entries under `[Unreleased]`, a shipped delta under an entity nothing cites,
then a commit renaming that heading to the version and opening a fresh empty
one, then the tag — asserted to report the uncited entity with HEAD at the tag.
The same fixture is asserted before the release commit as well, so the fix
cannot be a swap that buys the tagged shape by losing the pre-release one.

## Constraints

- Scope is this repo. The embedded trees exist only in aiwf's own source, so a
  consumer running this audit would check nothing. It is a repo invariant, not a
  shipped verb, and no consumer-facing surface changes.
- No new top-level verb, no new skill, no completion wiring.
- The audit runs at the release boundary, not at push. A milestone delta awaiting
  its epic's wrap entry is correctly absent until a release intervenes, which is
  why asking at push time would need an in-flight-epic exemption and asking at the
  tag needs none.
- Attribution follows the rule already documented: a milestone's delta belongs to
  its parent epic's entry, not its own.

## Design notes

- The tag workflow already fires on `v*` tags. The audit joins that workflow
  rather than adding one.
- `make comment-history-audit` is the shape precedent: a policy that also carries
  a focused target.
- D-0031 fixed the changelog category set and G-0613 questions it. Not settled
  here — that is a decision amendment, not a check.
- The two findings carry different severities, settled in D-0087. An uncited
  entity is a proven violation of a stated rule; an untrailered commit is a hole
  in the audit's own evidence.
- The rollup from milestone to parent epic goes through `tree.Load`, never a
  path scan. The loader resolves across active and archive; a glob over
  `work/epics/*/` does not, and an archived milestone then survives the rollup
  and reports as an uncited entity of its own.

## Surfaces touched

- `internal/policies/`
- `Makefile`
- `.github/workflows/changelog-check.yml`
- `CLAUDE.md` (release process)

## Out of scope

- The named-surface half of G-0529's direction — finding codes, verb names,
  config keys, exit codes. Those need list comparison rather than trailer
  reading, and the case that failed twice is the embedded trees.
- A changelog check for consumer repos. Different surfaces, no evidence yet.
- G-0613's category set.

## Dependencies

- None.

## Coverage notes

- No uncovered statement remains in `internal/policies/changelog_completeness.go`.
- One `//coverage:ignore`, in `resolveChangelogBase`: the branch taken when
  `git rev-list --max-parents=0 HEAD` exits zero having printed nothing.
  Measured, an unborn HEAD exits 128 and a HEAD that resolves always reaches a
  root, so the branch cannot be taken. The guard stays because the alternative
  is indexing an empty slice.
- Every criterion was mutation-probed, and every probe run after a fix was
  re-run to confirm the fix killed what it was written for. One equivalent
  mutant is recorded rather than chased: `--no-merges`, whose exclusion git's
  own default already achieves for a merge with no `--diff-merges`.

## References

- G-0529 — CHANGELOG completeness rests on recall at epic wrap and is never checked
- G-0613 — the wrap changelog category set omits Removed, which practice uses
- D-0031 — changelog entries are copied from wrap.md, not independently authored

## Release note

A release tag now fails when its notes do not say what it ships. The
`changelog-check.yml` workflow gained a second job, running `make
changelog-audit`: it compares every commit since the last release that changed
aiwf's embedded skill, ritual, template, agent-card, hook and guidance content
against what the release notes cite, and fails the tag on a shipped change no
entry names. A milestone's delta is cited by its parent epic, never by its own
id. Go source under those trees is the code that materializes them rather than
content a consumer receives, so it owes no entry.

The audit reads whichever section is current, working that out from the
changelog rather than from a tag: while the topmost version heading is the
release the base names, notes are still accumulating under `[Unreleased]`; once
a newer one appears, the release commit has moved them there and that heading
is read instead. Tag or no tag, so the answer is the same in the window between
the release commit and the tag push. Any tag on the commit under test is
excluded from the base, since otherwise it would be its own base and the range
would be empty.

The audit reports a second finding without failing on it — a shipped-surface
commit carrying no `aiwf-entity` trailer. With no entity named it cannot tell
whether an entry covers that commit, so it says so rather than stopping a
release it cannot show is incomplete.

Run `make changelog-audit` before tagging rather than meeting it at the push.
`AIWF_CHANGELOG_BASE=<ref> make changelog-audit` audits a past range; unset, the
audit does not run at all, which is what keeps it off every push.

## Decisions made during implementation

- D-0087 — the changelog audit blocks an uncited entity and reports an
  untrailered commit

## Validation

Measured 2026-09-10 on `milestone/M-0330-check-unreleased-against-what-shipped-before-the-release-tag`
at aba8ed5ca:

- `make ci` — green. Build, vet, the full golangci-lint set, `go test -race`,
  the diff-scoped coverage gate, the firing-fixture meta-gate, and the 29-step
  `aiwf doctor --self-check`. Total statement coverage 91.0%.
- `aiwf check` — 0 errors, 8 warnings, none of them on this milestone. They are
  the pre-existing archive-sweep backlog plus two the branch's own shape
  produces: `epic-active-no-drafted-milestones`, since this was the last drafted
  milestone, and `provenance-untrailered-scope-undefined`, since the branch has
  no upstream.
- `make changelog-audit` — reports E-0091 and exits non-zero, which is correct:
  that entry is the epic wrap's to write. It reported G-0659 too until this
  milestone wrote the line G-0659 was owed.
- `AIWF_CHANGELOG_BASE=v0.33.0 make changelog-audit`, the past-range form the
  release process documents — reports E-0091 only, the same single true
  finding, over a range that spans a release.
- The audit against this repo's real v0.34.0 tag — base resolves to `v0.33.0`
  and no entity is reported uncited. Five of the six entities in that range are
  cited by id and the sixth, M-0325, clears through its parent epic E-0090, so
  the rollup runs on real history rather than on a fixture. Two commits there
  carry no entity trailer and are reported as unattributed, which fails nothing
  (D-0087).

## Deferrals

- G-0671 — the audit sees no delta in a surface the kernel enumerates (a
  finding code, a verb, an `aiwf.yaml` key, an exit code), and no epic that
  shipped only compiled behaviour. This is the residual `## Closes` names: the
  half of G-0529's direction this milestone does not deliver, and the half that
  would have caught the thin entry G-0509 records.
- G-0672 — the git range scan is duplicated between this audit and the
  shipped-ritual provenance backstop. Recorded rather than extracted here: the
  duplication is worth removing on its own terms, and measurably did not cause
  either defect this milestone's review found.

## Reviewer notes

- Four independent review rounds ran over this milestone: two in parallel on
  the first pass, then a deciding pass, then a fourth after its findings were
  fixed. Each of the first three found a defect introduced by the previous
  round's fix. A fifth was declined by the human once severity had fallen from
  "the gate can never fire" to prose, and after the one part no reviewer had
  seen — the whole-file fallback for a base no heading names — was
  mutation-probed directly and turned up the version-prefix collision now
  pinned in `TestChangelogRangeNotes`.

- An AC that scopes itself away from an outcome states why the outcome cannot
  be reached, not that the surface carrying it cannot be run. AC-4 scoped to
  the wiring on the grounds that the workflow is unrunnable here; the outcome
  behind it needed no workflow, and went unasserted for four review rounds
  while every wiring assertion stayed green. AC-5 carries it now.
- The audit's watched tree holds both embedded content and the Go that
  materializes it. Only the first ships; the second is a kernel-surface change,
  which this audit does not cover and G-0671 tracks.
- Content introduced only by a merge resolution is invisible to this audit.
  Measured: `git log --name-only` emits no file list for a merge without an
  explicit `--diff-merges`, and no `log.diffMerges` setting changes that, so
  such content belongs to no non-merge commit and is attributed to nothing. It
  is the same blind spot G-0602 tracks for the shipped-ritual provenance gate,
  and closing it there closes it here.
