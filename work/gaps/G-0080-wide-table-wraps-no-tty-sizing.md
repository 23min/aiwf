---
id: G-0080
title: Wide-table verbs wrap mid-row; no TTY-aware sizing or truncation
status: addressed
discovered_in: M-0076
addressed_by_commit:
    - "774090e1"
    - 02c349f6
---
## Problem (historical, 2026-05-08)

aiwf's wide-table verbs (`aiwf status` today; `aiwf list` once E-0020 lands) render via `text/tabwriter`, which has no notion of terminal width. When the title or description column overflows, tabwriter wraps the second column into the first column's gutter, breaking the visual scan of the id column — the very anchor users rely on for navigation.

Concrete symptom: `aiwf status` output where a milestone title like "Writer surface for milestone depends_on (closes G-0072)" wraps onto a second visual line that starts in column 0, making the next row's `M-NNN` indistinguishable from the previous row's wrapped tail.

Adjacent issues that surfaced in the same discussion:

- **No TTY-aware output mode.** aiwf has `--format=json` for piping but no equivalent split for *human* output between "in a terminal" (formatted) and "redirected" (plain). Output formatting decisions (color, bold headers, glyphs) need an `IsTerminal` check, separate from the data-shape decision.
- **Ad-hoc glyph use.** `aiwf status` already uses `✓` and `→` in-band; there's no documented palette and no consistent application across verbs. The natural set is `✓` met/done, `→` in-progress, `○` open/draft, `✗` cancelled/wontfix/rejected — all BMP, 1-cell wide, render in any modern terminal.
- **No truncation surface.** Verbs have no `--no-trunc` / `--wide` escape hatch for users who want full text when the terminal is narrow.
- **Inconsistent header treatment.** No bold or underline anchor; the eye has to find columns by counting whitespace.

## Resolution sketch (the original design that landed)

1. **TTY-aware sizing using stdlib only.** `golang.org/x/term` for `IsTerminal` + `GetSize`; keep `text/tabwriter` for the rendering. No `lipgloss`, no `go-pretty`, no new deps.
2. **One-flex-column-per-verb truncation.** Designate one column (title in `aiwf status`, description in `aiwf list`) as the flex column. When natural widths overflow available terminal width, the flex column shrinks and truncates with `…`; fixed columns (id, status, dates) never shrink.
3. **Bold headers** via raw ANSI `\033[1m…\033[0m`, gated on TTY + `NO_COLOR` env var (per https://no-color.org).
4. **Glyph palette** for status, applied consistently: `✓` (met/done), `→` (in-progress/active), `○` (open/draft/proposed), `✗` (cancelled/wontfix/rejected).
5. **No gridlines.** Whitespace separation; reflow on terminal resize is a non-issue when nothing is being drawn that resize could break.
6. **`--no-trunc` flag** on wide-table verbs for the escape hatch.

## Disposition (2026-05-20, organic close)

Every item in the Resolution sketch landed across two "partial G-0080" commits, plus subsequent housekeeping. As of HEAD:

- **(1) TTY-aware sizing**: `internal/render/term.go` wraps `golang.org/x/term`'s `IsTerminal` + `GetSize`; consumers call `render.TermWidth(os.Stdout)` and get 0 when stdout is piped.
- **(2) One-flex-column truncation**: `aiwf list` calls `ComputeTitleBudget(rows, statuses, termWidth)` to derive a per-row rune cap with a `MinTitleColumnRunes` floor; `aiwf status` calls `TruncStatusTitle(title, termWidth, prefix, tail)` reusing the same floor. Both gate on `termWidth > 0`, so piped output is full-text.
- **(3) Bold headers**: `render.Bold(s, colorEnabled)` emits ANSI `\033[1m…\033[0m`; `colorEnabled` is computed once per verb call from TTY + `NO_COLOR` env. `aiwf list` bolds its header row; `aiwf status` bolds section labels. Test coverage in `internal/render/color_test.go` pins NO_COLOR behavior across set/unset/empty cases.
- **(4) Glyph palette**: `render.StatusGlyph(status)` maps the closed kernel-status vocabulary to the four-glyph palette as documented. Glyphs are content (visible in piped output too), not style.
- **(5) No gridlines**: tabwriter is configured with `' '` padding and 0 flags — whitespace-only separation.
- **(6) `--no-trunc` flag**: wired on both `aiwf list` and `aiwf status`; sets `termWidth = 0` for the render path so truncation is skipped regardless of TTY.

Load-bearing commits: `774090e1` (truncation), `02c349f6` (bold section labels + glyph palette extension). Followed by `61672339` which moved `aiwf list` into its own subpackage with the helpers exported for cross-verb use.

## What's NOT in the original Resolution sketch

The original gap explicitly deferred:

- Progress bars for long-running verbs (still deferred — no second consumer)
- Wide-character support (still deferred — emoji/CJK not needed; `go-runewidth` noted but not pulled)
- Auto-pager (still deferred — no verb output regularly exceeds terminal height)

These remain valid defer-until-forcing-function candidates. None opens as a new gap at close time per the closure-without-successor pattern; if any becomes attractive, file fresh then.

## References

- `internal/render/term.go` — TTY sizing
- `internal/render/color.go` + `internal/render/color_test.go` — NO_COLOR contract
- `internal/cli/list/list.go` `RenderListRowsText` + `ComputeTitleBudget` + `MinTitleColumnRunes`
- `internal/cli/status/status.go` `TruncStatusTitle`
- E-0022 / M-0076 — review session 2026-05-08 where the gap was filed.

## Discovered in

E-0022 / M-0076 review session 2026-05-08, while discussing how column-based aiwf output handles long titles and terminal resize.
