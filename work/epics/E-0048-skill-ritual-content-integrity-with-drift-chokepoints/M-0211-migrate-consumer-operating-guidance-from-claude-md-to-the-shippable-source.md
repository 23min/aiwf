---
id: M-0211
title: Migrate consumer operating guidance from CLAUDE.md to the shippable source
status: in_progress
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
      status: open
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
`CLAUDE.md`'s id-collision section is split in place: the two consumer-operating
blocks reduce to a pointer at the shipped homes; the merge-time `git mv`
mechanics, the allocator-scan detail, the reallocate discipline, and the E-0033
history stay.

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

## Decisions made during implementation

## Validation

## Deferrals

## Reviewer notes
