---
id: M-0153
title: Statusline script portability and robustness fixes
status: draft
parent: E-0039
tdd: advisory
---
# M-0153 — Statusline script portability and robustness fixes

## Goal

Make the shipped statusline render correctly on macOS as well as Linux, and
survive editor/tooling reflow, by removing two fragile environmental
assumptions from the script.

## Context

`.claude/statusline.sh` today uses bare `tac` to walk the transcript (absent on
stock macOS → the token segment silently reads zero) and parses git ahead/behind
by splitting on a literal tab embedded in the source (`${counts%%<TAB>*}`),
which breaks the instant an editor, copy-paste, or patch tool reflows the tab to
spaces. Both fail soft to *wrong* output rather than crashing. This milestone
fixes the tracked script in place; embedding it ships at M-0155.

## Acceptance criteria

<!-- Formal ACs are added at aiwfx-start-milestone via `aiwf add ac M-0153`.
     Intended shape, to be made testable then: -->

The fixed script reads the transcript via a `tail -r … || tac` fallback (BSD/
macOS first, GNU second) and parses ahead/behind via `read -r ahead behind`
(default-IFS split on space *or* tab), with a mechanical assertion that the
robust forms are present and the fragile forms (bare `tac`, literal-tab
parameter expansion) are absent. The fix is behavior-preserving on Linux.

## Constraints

- Stays fail-soft on every segment.
- Mechanical evidence is required for AC promotion even though `tdd: advisory`
  (per CLAUDE.md's AC-promotion rule) — a content assertion over the script.
- The content assertion is anchored to the specific constructs — it asserts the
  robust forms are present (the `tail -r … || tac` fallback, the
  `read -r ahead behind` parse) **and** that the fragile forms (bare `tac`, the
  literal-tab parameter expansion) are absent. Not a loose whole-file grep, per
  CLAUDE.md's "substring assertions are not structural assertions" rule, so a
  later reflow cannot reintroduce a fragile form undetected.

## Design notes

- `tail -r "$f" 2>/dev/null || tac "$f"` — BSD/macOS `tail -r` first, GNU `tac`
  fallback.
- `read -r ahead behind <<<"$counts"` — default IFS splits on space or tab;
  the existing `${ahead:-0}` / `${behind:-0}` guards already cover the empty case.

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
