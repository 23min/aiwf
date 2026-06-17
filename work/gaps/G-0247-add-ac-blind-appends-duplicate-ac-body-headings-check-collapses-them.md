---
id: G-0247
title: add ac blind-appends duplicate AC body headings; check collapses them
status: addressed
addressed_by_commit:
    - fec1db93
---
## The defect

`aiwf add ac` appends an AC body heading unconditionally, and `aiwf
check` collapses duplicate headings — so a duplicated `### AC-N` block
is created by normal use and is then invisible to validation. The hole
in the check is the exact shape of the verb's blind spot.

### Verb side — blind append

`addAC` allocates the next id from the frontmatter count
(`base := len(parent.ACs)`, `internal/verb/ac.go:103`) and calls
`appendACHeading` (`ac.go:131`), which **unconditionally** appends
`\n### AC-N — <title>\n\n` at the end of the body (`ac.go:419-423`)
with no check for a pre-existing `### AC-N` line.

### Check side — set-collapse hides it

`scanACHeadings` returns a **set** of ids
(`internal/check/acs.go:440-453`); two `### AC-2` headings collapse to
one entry. `acsBodyCoherence` (`acs.go:356-426`) then sees the id
present in both body and frontmatter → no `missing-heading`, no
`orphan-heading`. The rationale comment at `acs.go:435-439` explicitly
claims duplicates can't slip through — but its reasoning only covers a
duplicate that is *also* a frontmatter dup or an orphan. A duplicate of
an id that **is** in frontmatter is neither, so it passes clean.

## Reproduction (normal flow, from the shipped template)

1. `aiwf add milestone …` from the template
   (`internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/milestone-spec.md`):
   frontmatter `acs: []` (line 8) plus body placeholder headings
   `### AC-1`, `### AC-2` (lines 38, 42). `aiwf add milestone` does not
   parse body `### AC` into frontmatter.
2. `aiwf add ac M-NNNN --title "real AC"` → `base = 0` → allocates
   `AC-1` → appends a second `### AC-1 — real AC`.
3. Body now has two `### AC-1` headings; frontmatter has `AC-1`.
4. `aiwf check` is green on the duplicate (the leftover `### AC-2`
   placeholder shows up as an orphan-heading warning, but the AC-1
   *duplicate* does not). Add a second real AC and the AC-2 orphan
   clears too — N duplicated headings, fully green.

## Fix shape (one wf-patch, both sides)

1. **Verb:** make `appendACHeading` idempotent — if a `### AC-N` line
   already exists, rewrite it in place (the `rewriteACHeading` helper at
   `ac.go:446` already does in-place rewrite) instead of appending a
   second one.
2. **Check:** have `scanACHeadings` count headings per id and add a
   `duplicate-heading` subcode to `acsBodyCoherence` that flags
   `count > 1`. Deterministic, zero false positives. Update the stale
   rationale comment at `acs.go:435-439`.
3. Both sides get a fixture/seam test; the template-derived repro above
   is the canonical fixture.

## Explicitly out of scope (recorded so it is not lost)

The downstream report bundled two further "check is blind to drift"
sub-claims that are **not** part of this fix:

- **`### AC-N — <title>` vs frontmatter title mismatch.** This is a
  *deliberate* design choice, pinned by
  `TestAcsBodyCoherence_TitleTextNotChecked`
  (`internal/check/acs_test.go:614-650`): coherence pairs by id only,
  "prose is not parsed." Reversing it is a philosophy shift, not a free
  lint — out of scope here.
- **One-directional prose edges.** Vague as stated; overlaps the
  standing "prose is not parsed" stance and G-0073 (cross-kind blocking
  via body prose). Not actionable in this gap.

Source: downstream consumer feedback, 2026-06-12.
