---
id: M-0153
title: Statusline script portability and robustness fixes
status: in_progress
parent: E-0039
tdd: advisory
acs:
    - id: AC-1
      title: Transcript walk works on macOS and Linux (tail -r with tac fallback)
      status: met
    - id: AC-2
      title: Ahead/behind parse survives editor tab→space reflow (default-IFS split)
      status: met
    - id: AC-3
      title: Read-only git calls skip the optional index lock (GIT_OPTIONAL_LOCKS=0)
      status: open
---
# M-0153 — Statusline script portability and robustness fixes

## Goal

Make the shipped statusline render correctly on macOS as well as Linux, survive
editor/tooling reflow, and stop contending for the Git index lock on every
render — by removing three fragile environmental assumptions from the script.

## Context

`.claude/statusline.sh` today uses bare `tac` to walk the transcript (absent on
stock macOS → the token segment silently reads zero) and parses git ahead/behind
by splitting on a literal tab embedded in the source (`${counts%%<TAB>*}`),
which breaks the instant an editor, copy-paste, or patch tool reflows the tab to
spaces. Both fail soft to *wrong* output rather than crashing.

Third, the script runs `git status` (and other read-only git calls) on every
render, and `git status` opportunistically refreshes the on-disk index — which
takes `.git/index.lock`. When a render is SIGKILLed mid-refresh (a newer render
supersedes it) the lock is stranded; and a concurrent `git commit` in the same
repo can fail with `Unable to create '.git/index.lock': File exists`. Exporting
`GIT_OPTIONAL_LOCKS=0` makes every git subprocess skip that optional index write
— correct output, no lock taken — eliminating both the contention and the
stale-lock risk, since the script only ever reads.

This milestone fixes the tracked script in place; embedding it ships at M-0155.

## Acceptance criteria

<!-- Formal ACs are added at aiwfx-start-milestone via `aiwf add ac M-0153`.
     Intended shape, to be made testable then: -->

The fixed script reads the transcript via a `tail -r … || tac` fallback (BSD/
macOS first, GNU second), parses ahead/behind via `read -r ahead behind`
(default-IFS split on space *or* tab), and exports `GIT_OPTIONAL_LOCKS=0` before
its first git call so no read-only git invocation takes the index lock — with a
mechanical assertion that all three robust forms are present and the two fragile
forms (bare `tac`, literal-tab parameter expansion) are absent. The fix is
behavior-preserving on Linux.

## Constraints

- Stays fail-soft on every segment.
- Mechanical evidence is required for AC promotion even though `tdd: advisory`
  (per CLAUDE.md's AC-promotion rule) — a content assertion over the script.
- The content assertion is anchored to the specific constructs — it asserts the
  robust forms are present (the `tail -r … || tac` fallback, the
  `read -r ahead behind` parse, the `GIT_OPTIONAL_LOCKS=0` export) **and** that
  the fragile forms (bare `tac`, the literal-tab parameter expansion) are absent.
  Not a loose whole-file grep, per CLAUDE.md's "substring assertions are not
  structural assertions" rule, so a later reflow cannot reintroduce a fragile
  form undetected.

## Design notes

- `tail -r "$f" 2>/dev/null || tac "$f"` — BSD/macOS `tail -r` first, GNU `tac`
  fallback.
- `read -r ahead behind <<<"$counts"` — default IFS splits on space or tab;
  the existing `${ahead:-0}` / `${behind:-0}` guards already cover the empty case.
- `export GIT_OPTIONAL_LOCKS=0` near the top, before the first git call — git
  then skips the opportunistic index refresh that takes `.git/index.lock`.
  Equivalent to prefixing each call with `git --no-optional-locks …`; the env
  export covers them all at once. The script only reads, so there is no downside
  (a later real git command just re-does the cheap stat-refresh itself). Added
  in Git 2.15 for exactly this background-consumer case.

## Surfaces touched

- `.claude/statusline.sh`
- a content-assertion test (exact location chosen at start-milestone)

## Out of scope

- Embedding, the `--statusline` flag, settings wiring, doctor — later milestones.

## Dependencies

- None.

## References

- [E-0039](epic.md) · [G-0183](../../gaps/G-0183-aiwf-has-no-install-path-for-its-aiwf-aware-claude-code-statusline.md)

---

## Work log

- (pending)

## Decisions made during implementation

- (none)

## Validation

- (pending)

## Deferrals

- (none)

## Reviewer notes

- (none)

### AC-1 — Transcript walk works on macOS and Linux (tail -r with tac fallback)

### AC-2 — Ahead/behind parse survives editor tab→space reflow (default-IFS split)

### AC-3 — Read-only git calls skip the optional index lock (GIT_OPTIONAL_LOCKS=0)

