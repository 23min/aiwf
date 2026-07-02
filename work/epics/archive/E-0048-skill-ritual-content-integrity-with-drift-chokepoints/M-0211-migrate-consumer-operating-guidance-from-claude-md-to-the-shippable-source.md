---
id: M-0211
title: Migrate consumer operating guidance from CLAUDE.md to the shippable source
status: done
parent: E-0048
depends_on:
    - M-0196
tdd: advisory
acs:
    - id: AC-1
      title: Cross-branch id-allocation rule migrated to guidance and aiwf-add skill
      status: met
    - id: AC-2
      title: Drift chokepoint asserts operating anchors present in shipped guidance
      status: met
    - id: AC-3
      title: Authoring rule records the audience dividing principle in CLAUDE.md
      status: met
---
## Goal

Consumer-facing *operating* guidance — how to drive aiwf the tool in any repo —
must ship to consumers via the embedded guidance source
(`internal/skills/embedded-guidance/aiwf-guidance.md`), which `aiwf init` /
`aiwf update` materialize into a consumer's `.claude/aiwf-guidance.md` and wire
into their root `CLAUDE.md`. A rule that lands in this repo's `CLAUDE.md`
instead is invisible to every consumer and forks from the single source of
truth (G-0313).

The one concrete instance today is the **cross-branch id-allocation workflow** —
allocate on your working branch; in a multi-clone setup `aiwf add --fetch` and
push promptly; an unpushed peer id is invisible; collisions resolve with `aiwf
reallocate`. It lives only in this repo's `CLAUDE.md`; the always-on guidance
and the `aiwf-add` / `aiwf-reallocate` verb skills never carried it, so it ships
nowhere — the exact failure mode G-0313 names. This milestone migrates it into
its shippable homes, records the audience-based dividing principle as an
authoring rule, and adds a mechanical drift chokepoint that reddens if a curated
consumer-operating rule regresses out of the shipped fragment. Final milestone of
E-0048.

The dividing principle is **audience, not importance**: "how to OPERATE aiwf in
any repo" ships (guidance always-on; verb / ritual skills on-demand); "how to
DEVELOP aiwf itself" stays in `CLAUDE.md` and correctly does not ship. Hybrid
sections are split, not moved wholesale — the `CLAUDE.md` id-collision section
keeps its merge-time repo-development specialization behind a pointer (Option B).

## Acceptance criteria

### AC-1 — Cross-branch id-allocation rule migrated to guidance and aiwf-add skill

The cross-branch allocation workflow moves from `CLAUDE.md`-only into its two
shippable homes, following G-0313's four-tier layering (always-on guidance →
on-demand skills). The guidance source gains a **tight** operating rule: allocate
ids on your working branch; in a multi-clone setup `aiwf add --fetch` and push
promptly so a peer's pushed id is seen and yours reaches them, else risk a
collision `aiwf reallocate` resolves. The `aiwf-add` verb skill gains the **full**
mechanics on-demand — one-machine-multiple-worktrees (nothing to do) vs
separate-clones (`--fetch`, push promptly), allocate-on-any-branch, the
unpushed-peer-invisible expectation, `--fetch`-failure-warns-not-blocks, and the
unmerged-branch-can't-be-referenced-by-id-in-prose / backtick-until-trunk caveat.
`CLAUDE.md`'s id-collision section is split in place (Option B): the two
consumer-operating blocks reduce to a pointer at the shipped homes; the merge-time
`git mv` mechanics, the allocator-scan detail, the reallocate discipline, and the
E-0033 history stay.

**Pass criterion**: a structural test scoped to the guidance's allocation bullet
asserts the tight rule is present (`--fetch`, allocate-on-branch, push-promptly);
a second scoped to the `aiwf-add` skill's cross-branch section asserts the full
mechanics are present. **Edge cases**: the assertions are section-scoped, not
flat greps (per CLAUDE.md *Substring assertions are not structural assertions*).
**Code references**: `internal/skills/embedded-guidance/aiwf-guidance.md`,
`internal/skills/embedded/aiwf-add/SKILL.md`; tests under `internal/policies/`.

### AC-2 — Drift chokepoint asserts operating anchors present in shipped guidance

A new `internal/policies/` policy asserts that a curated set of consumer-operating
anchors — gate-per-mutation, reallocate-not-`git mv`, AC-mechanical-evidence,
one-decision-at-a-time, never-suggest-pause, body-prose-id, and the cross-branch
allocation rule — is present in the embedded guidance source. Trimming an
operating rule out of the shipped fragment (drift back toward `CLAUDE.md`-only)
reddens CI. The policy is a CI-tier Go test, not an `aiwf check` finding: the
guidance source is an aiwf-repo authoring artifact absent in a consumer tree,
where such a check would be inert — the same class as the M-0209 / M-0210
policies.

**Pass criterion**: the policy passes green on the live tree (every curated
anchor present today) and reddens on a firing fixture — a synthetic guidance file
with one anchor stripped returns a violation. **Edge cases**: a missing guidance
file and an unreadable one each surface a violation rather than passing
vacuously. **Code references**: the policy in `internal/policies/`; its firing
rows route through the shared `TestFiringFixtures_MultiSite` harness, lighting
the construction line so the G-0259 meta-gate is satisfied for a policy added
after that gate.

### AC-3 — Authoring rule records the audience dividing principle in CLAUDE.md

`CLAUDE.md` gains a short authoring section naming the dividing principle —
audience, not importance — that routes consumer-operating rules to the embedded
guidance source (always-on) plus the verb / ritual skills (on-demand) and keeps
repo-development guidance in `CLAUDE.md`, with hybrid rules split, not moved
wholesale. The section points at the AC-2 chokepoint as the mechanical backstop
and states that discipline is the interim catch for the forward judgment-call the
mechanical test cannot classify (a brand-new rule's audience).

**Pass criterion**: a structural test scoped to the new authoring section asserts
it names the dividing principle, points at the embedded guidance source as the
shippable home, and references the chokepoint. **Edge cases**: the assertion is
scoped to the section heading, not a file-wide grep. **Code references**:
`CLAUDE.md`; the test under `internal/policies/`.

## Work log

### AC-1 — cross-branch allocation migrated to the shippable homes
Guidance gained the folded id-collision bullet carrying the tight cross-branch
rule (`--fetch` / allocate-on-branch / push-promptly); `aiwf-add` gained the
`## Allocating ids across branches and clones` section with the full mechanics;
`CLAUDE.md`'s id-collision section was split in place (Option B) to a pointer.
Pinned by `TestM0211_AC1_GuidanceCarriesCrossBranchRule`,
`TestM0211_AC1_AddSkillCarriesCrossBranchMechanics`, and
`TestM0211_AC1_ClaudeMdIdCollisionSplitInPlace`. Commit `d52f037f`.

### AC-2 — drift chokepoint
`PolicyM0211GuidanceOperatingAnchors` asserts the curated operating-anchor set is
present in the shipped guidance. Pinned by `TestPolicy_M0211GuidanceOperatingAnchors`
(live tree) plus the `m0211/missing-file` and `m0211/missing-anchor` rows in
`TestFiringFixtures_MultiSite` (non-vacuity + G-0259 meta-gate). Commit `d52f037f`.

### AC-3 — authoring rule
`CLAUDE.md` gained the `## Consumer-operating guidance vs repo-development
guidance` section naming the audience-not-importance principle, pointing at the
guidance source, and referencing the chokepoint. Pinned by
`TestM0211_AC3_AuthoringRuleNamesDividingPrinciple`. Commit `d52f037f`.

## Decisions made during implementation

- **Option B (split-in-place) for the `CLAUDE.md` id-collision section.** Of the
  three trim options (A pointer-only / B split-in-place / C add-only), the
  operator chose B: reduce the consumer-operating avoidance blocks to a pointer,
  fold the full mechanics into the `aiwf-add` skill (nothing dropped), and keep
  the merge-time `git mv` resolution mechanics + the E-0033 history. A
  planning-conversation decision recorded here; no new ADR.
- **Chokepoint as a Go policy, not an `aiwf check` finding.** The guidance source
  is an aiwf-repo authoring artifact (absent in consumer trees), so a check rule
  would be inert there — matching the M-0209 / M-0210 policy class. The policy
  guards the regression direction (a shipped rule drifting *out*); the forward
  judgment-call (a brand-new rule's audience) is the authoring rule's job.
- **Folded the cross-branch rule into the existing reallocate bullet.** The
  always-on guidance is hard-capped at 50 lines (M-0163/AC-4) and was already at
  49. A separate bullet blew the budget; folding all id-collision operating
  guidance into one bullet kept it at exactly 50 while shipping the rule — and is
  arguably better organization (all collision ops in one place). The detail lives
  on-demand in the `aiwf-add` skill, per G-0313's tight-fragment principle.

## Validation

- `make check-fast` (go vet + golangci-lint + full `go test`): green — exit 0.
- `make coverage-gate`: green — diff-scoped statement coverage complete;
  firing-fixture meta-gate satisfied (`m0211-guidance-operating-anchors` lit by
  its two firing rows, not in `grandfatherDark`); no stale allowlist; the
  skill-edit backstop does not apply (`aiwf-add` is under `embedded/`, not
  `embedded-rituals/`).
- Guidance at the 50-line budget (`TestGuidance_WithinLineBudget`); the seven
  M-0163 rules intact (`TestRenderGuidance_ContainsAllRules`, reallocate literal
  contiguous).
- No real entity id-shapes in the shipped `aiwf-add` / guidance prose (G-0299).

## Deferrals

None filed as gaps. The reviewer noted (non-blocking) that AC-2's spec wording
implies both a missing-file *and* an unreadable-file case, but both route through
the single `if err != nil` construction line, which the `m0211/missing-file`
firing fixture already lights — coverage and the G-0259 meta-gate are satisfied
by the one fixture, so no separate unreadable-file fixture is warranted.

## Reviewer notes

Independent fresh-context reviewer returned **APPROVE**, verifying every
load-bearing claim by measurement. Non-vacuity of the chokepoint was proven by
revert-and-test — stripping `--fetch` and, separately, `body-prose-id` from the
live guidance each reddened `TestPolicy_M0211GuidanceOperatingAnchors` naming the
dropped anchor, restored clean. The reviewer confirmed: the three AC tests are
section-scoped structural assertions (not flat greps); the guidance stays at the
50-line budget with all seven M-0163 rules intact and the reallocate literal
contiguous; no id-shape leakage in the shipped `aiwf-add` prose; the Option-B
CLAUDE.md split is minimal with no unrelated churn; and the tree was restored
clean after the mutation probes. Two non-blocking observations, both accepted
without code churn: the empty wrap-side sections + open ACs (expected pre-wrap,
now resolved) and the AC-2 unreadable-file edge (shares the missing-file err
line). The reviewer also flagged a tooling artifact — the `Read` tool served
stale pre-commit content for the guidance source and spec, so it used `git show`
/ working-tree `grep` as authoritative; not a code issue.
