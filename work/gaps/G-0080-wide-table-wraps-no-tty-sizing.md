---
id: G-0080
title: Wide-table verbs wrap mid-row; no TTY-aware sizing or truncation
status: open
discovered_in: M-0076
---
## Problem

aiwf's wide-table verbs (`aiwf status` today; `aiwf list` once E-0020 lands) render via `text/tabwriter`, which has no notion of terminal width. When the title or description column overflows, tabwriter wraps the second column into the first column's gutter, breaking the visual scan of the id column — the very anchor users rely on for navigation.

Concrete symptom: `aiwf status` output where a milestone title like "Writer surface for milestone depends_on (closes G-0072)" wraps onto a second visual line that starts in column 0, making the next row's `M-NNN` indistinguishable from the previous row's wrapped tail.

Adjacent issues that surfaced in the same discussion:

- **No TTY-aware output mode.** aiwf has `--format=json` for piping but no equivalent split for *human* output between "in a terminal" (formatted) and "redirected" (plain). Output formatting decisions (color, bold headers, glyphs) need an `IsTerminal` check, separate from the data-shape decision.
- **Ad-hoc glyph use.** `aiwf status` already uses `✓` and `→` in-band; there's no documented palette and no consistent application across verbs. The natural set is `✓` met/done, `→` in-progress, `○` open/draft, `✗` cancelled/wontfix/rejected — all BMP, 1-cell wide, render in any modern terminal.
- **No truncation surface.** Verbs have no `--no-trunc` / `--wide` escape hatch for users who want full text when the terminal is narrow.
- **Inconsistent header treatment.** No bold or underline anchor; the eye has to find columns by counting whitespace.

## Resolution sketch

The design space was discussed and the agreed shape is:

1. **TTY-aware sizing using stdlib only.** `golang.org/x/term` for `IsTerminal` + `GetSize`; keep `text/tabwriter` for the rendering. No `lipgloss`, no `go-pretty`, no new deps.
2. **One-flex-column-per-verb truncation.** Designate one column (title in `aiwf status`, description in `aiwf list`) as the flex column. When natural widths overflow available terminal width, the flex column shrinks and truncates with `…`; fixed columns (id, status, dates) never shrink.
3. **Bold headers** via raw ANSI `\033[1m…\033[0m`, gated on TTY + `NO_COLOR` env var (per https://no-color.org).
4. **Glyph palette** for status, applied consistently: `✓` (met/done), `→` (in-progress/active), `○` (open/draft/proposed), `✗` (cancelled/wontfix/rejected).
5. **No gridlines.** Whitespace separation; reflow on terminal resize is a non-issue when nothing is being drawn that resize could break.
6. **`--no-trunc` flag** on wide-table verbs for the escape hatch.

## Out of scope (deferred)

- **Progress bars** for long-running verbs (`aiwf render` is the only candidate today); revisit when there's a second consumer or when render's runtime warrants it. A stdlib spinner is the likely first move, not a `go-pretty` dep.
- **Wide-character support** (emoji, CJK). `text/tabwriter` counts runes, not display cells; current glyph palette is all 1-cell BMP so this works. If output ever needs to include emoji or CJK, `github.com/mattn/go-runewidth` is the de facto fix — note it as a known limitation, don't pull the dep speculatively.
- **Auto-pager** for long output. `git log` does this; tasteful default but a separate decision from column UX. Defer until there's a verb whose output regularly exceeds terminal height.

## Why this is a gap, not a milestone

The fix is a self-contained UX layer over existing verbs; it doesn't touch the kernel's correctness commitments (entity vocabulary, FSM, provenance, validation) and has no `aiwf check` finding to gate on. Once a consumer is willing to absorb the change, this can be picked up as a small milestone under a future UX-focused epic, or as a one-shot patch via `wf-patch` if the implementation lands in a single commit.

## Discovered in

E-0022 / M-0076 review session 2026-05-08, while discussing how column-based aiwf output handles long titles and terminal resize.
