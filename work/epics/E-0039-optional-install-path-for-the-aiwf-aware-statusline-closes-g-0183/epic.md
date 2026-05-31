---
id: E-0039
title: Optional install path for the aiwf-aware statusline (closes G-0183)
status: active
---
# E-0039 — Optional install path for the aiwf-aware statusline (closes G-0183)

## Goal

Let a downstream aiwf consumer opt into the aiwf-aware Claude Code statusline
via `aiwf init/update --statusline` — portable across Linux and macOS, with
activation gated by explicit per-invocation consent — without aiwf ever quietly
editing a settings file.

## Context

aiwf materializes every framework artifact it ships (verb skills, ritual
skills, role agents, entity templates, git hooks) into a consumer repo via
`aiwf init` / `aiwf update`. The one artifact it does not ship is
`.claude/statusline.sh` — the only aiwf-*aware* surface among them: it derives
the in-flight epic/milestone/gap from the ritual branch shapes
(`milestone/M-…`, `epic/E-…`, `patch/[Gg]-…`), resolves them against the
`work/epics` / `work/gaps` layout, colors each id by its frontmatter `status:`,
counts sibling ritual worktrees (`+N⎇`), and renders the context-window token
ball. A consumer's only path to it today is copying the file out of this repo
by hand — which orphans maintenance (portability fixes never reach the copy)
and gives no drift signal.

This epic closes [G-0183](../../gaps/G-0183-aiwf-has-no-install-path-for-its-aiwf-aware-claude-code-statusline.md).
It builds on the embedding mechanism established by ADR-0014 / E-0038 (rituals
shipped embedded in the binary, materialized by `init`/`update`) — but applies
it with one deliberate difference: the statusline is embedded yet **excluded
from the unconditional refresh set**, so a consumer's customizations survive
`aiwf update`.

## Scope

### In scope

- Embed `statusline.sh` in the binary (`go:embed`), excluded from the
  unconditional refresh set — scaffold-once, not byte-refreshed.
- A `--statusline` flag on `aiwf init` and `aiwf update` (a flag, not a new
  verb), with `--scope project|user` (default `project`). Writes the script
  only if absent; never clobbers; bare `aiwf update` never touches it.
- Consent-gated activation: interactive `[y/N]` confirm when a TTY is present,
  explicit `--wire-settings` flag otherwise; wiring into `settings.local.json`
  (project scope) or `~/.claude/settings.json` (user scope); never clobbering a
  pre-existing `statusLine` key.
- Portability fixes to the shipped script: `tac` → `tail -r` fallback;
  literal-tab sync parse → `read -r ahead behind`.
- An `aiwf doctor` block (advisory) covering missing `jq`/`gh` with
  platform-aware install hints, installed-but-not-wired state, embedded-vs-
  on-disk drift, and a container user-scope nudge.
- An ADR amending the "aiwf never edits settings.json" stance to "not without
  explicit per-invocation consent," with CLAUDE.md and the `doctor.go` comment
  updated to match.

### Out of scope

- Auto-detecting a devcontainer and silently choosing `--scope user`. Detection
  drives a doctor *recommendation* only; the choice stays explicit.
- Managing a shared/committed statusline for teams (wiring into the tracked
  `settings.json` + tracking the script). Documented as a manual path, not
  automated.
- Changing how the other materialized artifacts (skills, hooks, agents,
  templates) refresh. This epic adds one exception to the refresh set; it does
  not restructure the set.

## Constraints

- **Correctness must not depend on the LLM.** Every AC pins a mechanical
  assertion per CLAUDE.md's AC-promotion rule — including the doc-shaped ACs
  (ADR content, CLAUDE.md amendment), which assert on named sections.
- **aiwf writes only inside the repo it is invoked in.** User-scope
  (`~/.claude/`) is an explicit operator choice gated by `--scope user`, never
  auto-selected from environment sniffing.
- **The script stays fail-soft on every segment** — any erroring segment
  collapses to `?` or is dropped; a missing `jq`/`gh` degrades, never breaks.
- **Embedded but scaffold-once.** The script source travels with the binary
  (so fixes ship and `doctor` can detect drift), but the consumer's on-disk
  copy is written once and never overwritten by a routine `aiwf update`.

## Success criteria

<!-- Observable at epic close, not tests. -->

- [ ] A consumer can run `aiwf update --statusline` in an existing repo and
      `aiwf init --statusline` in a fresh one, obtaining a working
      `.claude/statusline.sh`, the matching gitignore entry, and a printed
      activation snippet — with a subsequent bare `aiwf update` leaving their
      edited copy untouched.
- [ ] `--scope user` installs a single `~/.claude/statusline.sh` that renders
      correctly from any worktree in the same (dev)container without re-install.
- [ ] No activation path mutates a settings file without explicit
      per-invocation consent (a TTY confirm or `--wire-settings`); a
      pre-existing `statusLine` key is never overwritten.
- [ ] The shipped script renders its token and ahead/behind sync segments
      correctly on both Linux and macOS.
- [ ] `aiwf doctor`, when the statusline is installed, reports its dependency,
      wiring, and drift state.
- [ ] The "never edits settings.json" stance is superseded by the ADR listed in
      the *ADRs produced* table below, and CLAUDE.md plus the `doctor.go`
      comment reflect the amended stance.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does Claude Code honor `statusLine` in `settings.local.json`? | yes (for the wiring milestone) | Verify at the consent-wiring milestone; if not, fall back to `settings.json` with a louder consent prompt. |
| Full ADR vs lighter `decision` entity for the stance change? | no | Settle at the ADR milestone; leaning ADR since it revises a documented invariant. |
| Shared/committed statusline for teams? | no | Out of scope this epic; ship a documented manual path, revisit if friction appears. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Editing a settings file reformats the operator's JSON (key order/whitespace lost on re-marshal). | med | Add-key-if-absent only; write a `.bak`; refuse to touch a file that already carries a `statusLine`. |
| aiwf's first interactive prompt sets a precedent that leaks into other verbs. | med | Gate the prompt strictly to this opt-in flag; the non-TTY path uses the explicit `--wire-settings` flag, never a hidden prompt. |
| Hand-copied statusline snapshots already in the wild go stale. | low | Embed the source + a `doctor` drift nudge so an installed copy can detect it lags the binary. |

## Milestones

<!-- Execution order. Status lives in each milestone's frontmatter, not here. -->

- [M-0153](work/epics/E-0039-optional-install-path-for-the-aiwf-aware-statusline-closes-g-0183/M-0153-statusline-script-portability-fixes-tac-and-tab-parse.md) — Script portability fixes (`tac`→`tail -r`, literal-tab parse→`read -r`) · depends on: —
- [M-0154](work/epics/E-0039-optional-install-path-for-the-aiwf-aware-statusline-closes-g-0183/M-0154-adr-amend-settings-json-stance-to-consent-gated.md) — ADR amending the settings.json stance to consent-gated · depends on: —
- [M-0155](work/epics/E-0039-optional-install-path-for-the-aiwf-aware-statusline-closes-g-0183/M-0155-embed-statusline-and-add-statusline-scaffold-with-scope.md) — Embed + `--statusline` flag with `--scope`, gitignore, printed snippet (no settings write) · depends on: M-0153
- [M-0156](work/epics/E-0039-optional-install-path-for-the-aiwf-aware-statusline-closes-g-0183/M-0156-consent-gated-statusline-settings-wiring.md) — Consent-gated settings wiring (TTY prompt + `--wire-settings`, no-clobber) · depends on: M-0154, M-0155
- [M-0157](work/epics/E-0039-optional-install-path-for-the-aiwf-aware-statusline-closes-g-0183/M-0157-aiwf-doctor-statusline-block.md) — `aiwf doctor` statusline block (deps + platform hints, wiring state, drift, container nudge) · depends on: M-0155

## ADRs produced

- ADR-NNNN — Amend the "aiwf never edits settings.json" stance to "not without explicit per-invocation consent" (allocated when the ADR milestone runs).

## References

- [G-0183](../../gaps/G-0183-aiwf-has-no-install-path-for-its-aiwf-aware-claude-code-statusline.md) — the gap this epic closes (carries the full design direction).
- [G-0184](../../gaps/G-0184-aiwf-check-misses-invented-id-shaped-tokens-no-rule-against-fabricating-ids.md) — related finding surfaced while planning this epic; the planning prose here is the surface that bug lives on.
- ADR-0014 / E-0038 — the embedded-rituals precedent this epic's shipping mechanism builds on.
- `internal/cli/doctor/doctor.go` (the "never edits settings.json" comment) and CLAUDE.md — the stance surfaces this epic amends.
- `.claude/statusline.sh` — the canonical script being shipped.
