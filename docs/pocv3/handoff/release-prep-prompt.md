# Release-prep handoff prompt

**Goal:** prepare the `poc/aiwf-v3` branch for a `v0.2.0` cut. The HTML render iteration (I3) shipped end-to-end across ~15 commits ending at `44ea40b`; what remains is small but load-bearing pre-release cleanup. Read this whole document before starting — context matters more than speed here.

## What's already shipped

I3 is closed. Recent commits on `poc/aiwf-v3` (newest first):

- `44ea40b feat(aiwf): integrate aiwf status as a rendered page` — `status.html` reuses the existing `buildStatus` + `readRecentActivity` helpers; sidebar gains a "Project status" link.
- `fefb354 feat(aiwf): logo + wordmark in sidebar; export docs/logo.svg` — three-bar SVG mark, currentColor (adapts to dark mode), wordmark `aiwf`. Standalone copy at `docs/logo.svg` (iris baked in, `#5e6ad2`).
- `9b88108 feat(aiwf): left-side navigation panel for HTML render` — two-column layout, `<details>`/`<summary>` per epic, current-page ancestors pre-expanded, `aria-current="page"` on the active link, mobile stacks below 768px.
- `302de69 fix(aiwf): cache-bust the rendered stylesheet by content hash` — `assets/style.css?v=<8-hex-sha256>`; the user couldn't see CSS changes across reloads on `file://`.
- `119879c refactor(aiwf): Linear-leaning palette` — iris `#5e6ad2`, muted greens/oranges, hairline borders, `--pill-bg-mix` / `--pill-border-mix` exposed as variables.
- `cce0c21 feat(aiwf): polish HTML render` — modern `color-mix()` pills, accent stripe, kicker, tabular nums, `prefers-color-scheme` dark mode.
- Earlier: I3 steps 1–7 (`7fd6524` through `606bfab`).

State of the world:
- 38 Playwright specs all green; Go race suite + lints all green.
- Smoke fixture at `/tmp/aiwf-smoke/site/` (may be stale).
- One Playwright spec note: "current page link carries aria-current=page" explicitly asserts the index page has zero `aria-current` links — see "Menu reordering" below for what changes there.

## The four fixes to land (one commit, in this order)

### 1. `aiwf update` strips the legacy `actor:` field from `aiwf.yaml`

This is the user's load-bearing complaint: after `go install …@latest` + `aiwf update`, the legacy `actor:` field is left in `aiwf.yaml`. `aiwf doctor` notes it but doesn't clean it, and the residual makes it hard for the user to tell at a glance whether the config is in sync with the binary.

Implementation:
- `tools/internal/config/config.go`: new `StripLegacyActor(root string) (changed bool, err error)`. Reads `aiwf.yaml`, strips any `^actor:.*$` line (textual, not a YAML round-trip — KISS for a known-dead key), writes back if changed. Returns `false, nil` when the field is absent. Pure function side-effect; idempotent.
- `tools/internal/initrepo/initrepo.go`: new `ensureLegacyActorClean(root, dryRun)` step called from `RefreshArtifacts` between `ensureSkills` and `ensureGitignore`. Step ledger entry shape:
  - `removed deprecated 'actor:' field` when changed.
  - `ActionPreserved` (no message) when not present.
  - In dry-run mode, reports the would-be action without writing.
- Test: in `tools/internal/initrepo/initrepo_test.go`, add `TestInit_StripsLegacyActor` — write `aiwf.yaml` with `aiwf_version: 0.1.0\nactor: human/peter\n`, run `Init`, assert the resulting file has no `actor:` line and `aiwf_version` is preserved. Companion negative test: `aiwf.yaml` without the field stays byte-identical (preserves comments).

### 2. `aiwf render --help` returns proper usage

Currently `aiwf render --help` falls into `runRender`'s dispatcher, which prints `aiwf render: unknown subcommand "--help"`. Fix:

- In `tools/cmd/aiwf/render_cmd.go` `runRender`, before the subcommand switch:
  ```go
  if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
      printRenderHelp()
      return exitOK
  }
  ```
- New `printRenderHelp()` listing both surfaces: `aiwf render roadmap [--write]` and `aiwf render --format=html [--out <dir>] [--scope <id>] [--no-history] [--pretty]`. One short paragraph per, then a hint that `aiwf help` is the master surface.
- Test: in `tools/cmd/aiwf/render_site_cmd_test.go` (or a new `render_help_test.go`), assert `run([]string{"render", "--help"})` returns `exitOK` and stdout contains both `roadmap` and `--format=html`.

### 3. New `aiwf-render` skill

Mirror the existing `tools/internal/skills/embedded/aiwf-status/` shape. New directory `tools/internal/skills/embedded/aiwf-render/SKILL.md`. Content (~40 lines):
- Frontmatter: `name`, `description: Use when the user asks to render the planning state as a static HTML site, publish governance views, or generate the project status page.`
- Body: short overview of `aiwf render --format=html`, the `aiwf.yaml.html` block (`out_dir`, `commit_output`), the four deployment patterns (link to `docs/pocv3/plans/governance-html-plan.md` §2), the ergonomic note that the site is gitignored by default.
- Don't reproduce the deployment YAML — link instead.

The G21 discoverability policy will pick it up automatically via the existing `loadCheckCodeLiterals` walker, but no new finding code is added so no policy work needed here.

### 4. README — short HTML page mention

**Per the user's explicit constraint: do NOT mention sidebar, status page, dark mode, or logo.** What to add instead: clarify that the HTML page provides both *governance* (entity tree + per-page tabs) and *status* (in-flight work, decisions, gaps, recent activity) views. One paragraph in the existing "HTML render" section:

> The render covers two surfaces: the **governance** view (epics + milestones + ACs + provenance, one page per entity) and the **project status** view (in-flight work, open decisions, open gaps, recent activity). The status view replaces the markdown output of `aiwf status` for browser consumption — same data, same `buildStatus` helper.

That's it for README. No mention of nav, dark mode, logo, palette.

### 5. Menu reordering (sidebar template + tests + index page title)

Three coupled changes:

**a. Reorder sidebar top items.** Currently:
```
GOVERNANCE       (sidebar-title label)
- Overview
- Project status
- E-01 ...
- E-02 ...
```

Target:
```
- Project status
- Overview
- E-01 ...
- E-02 ...
```

Specifically:
- Remove the `GOVERNANCE` `<p class="sidebar-title">` label entirely (the user wants it gone).
- Swap order of `Overview` and `Project status` — Project status comes first.

**b. Index page title is now "Overview", not "Governance".**
- `IndexData.Title` defaults to "Governance" in both resolvers. Change the default to `"Overview"`. The page's `<h1>` and `<title>` then read "Overview".
- Kicker line stays: `aiwf · governance` → keep, since the kicker labels the *site* not the page.

**c. Update Playwright + Go tests.**
- `tools/e2e/playwright/tests/render.spec.ts`:
  - Test `lists every epic with AC met-rollup` asserts `h1` is "Governance" — change to "Overview".
  - Test `current page link carries aria-current=page` asserts the index page has zero `aria-current` links. After the change, the Overview link IS the current page on `index.html`, so it WILL have `aria-current="page"`. Update assertion: `await expect(page.locator(\`aside.sidebar a[aria-current="page"]\`)).toHaveCount(1)` and the href is `index.html`.
  - Sidebar tests asserting the order of links — search for `sidebar-top` or `Overview` and `Project status`; tighten to assert order.
- `tools/cmd/aiwf/render_site_cmd_test.go` and `render_templates_test.go`: scan for any "Governance" assertion on the index page, retarget to "Overview".

**d. Sidebar template change (`tools/internal/htmlrender/embedded/_sidebar.tmpl`):**
- Drop the `<p class="sidebar-title">Governance</p>` line.
- Reorder the two `<li>` entries inside `<ul class="sidebar-top">` so Project status is first.
- Mark Overview link with `aria-current="page"` on the index page. Sidebar template doesn't currently know "this is the index page" — add `IsCurrentIndex bool` to `SidebarData` and set it true in both resolvers' `IndexData()` paths. The Overview link consults this flag.

**e. CSS:** `.sidebar-title` rule can stay (defensive) but is now unused. Optional: remove it. The brand mark + wordmark stay.

## Verification checklist

After the commit, run all of these:

```bash
# Go suite
go test -race ./tools/...
golangci-lint run ./tools/...

# Playwright suite (assumes make e2e-install was done previously)
make e2e

# Real upgrade simulation: build a binary, scaffold a fixture with a
# legacy actor: field, run aiwf update, confirm the field is gone and
# the gitignore got the site/ entry, then aiwf doctor reports clean.
go build -o /tmp/aiwf-prerel ./tools/cmd/aiwf
mkdir -p /tmp/aiwf-prerel-fixture && cd /tmp/aiwf-prerel-fixture
git init -q && git config user.email t@e.x && git config user.name t
cat > aiwf.yaml <<EOF
aiwf_version: 0.1.0
actor: human/peter
EOF
/tmp/aiwf-prerel update
grep -c "^actor:" aiwf.yaml  # must be 0
grep -c "^site/" .gitignore   # must be 1
/tmp/aiwf-prerel doctor       # must NOT carry the "deprecated actor" note

# Smoke render of the fixture site (produced by Playwright's
# fixture script earlier) and visually confirm:
#   - sidebar top order: "Project status" then "Overview"
#   - no "GOVERNANCE" label above the brand mark
#   - index page <h1> reads "Overview"
#   - all other pages unchanged
```

## Suggested commit shape

One commit, conventional-commits subject:

```
fix(aiwf): release-prep cleanup — strip legacy actor on update; render --help; aiwf-render skill; menu reordering; README clarification

- aiwf update strips deprecated `actor:` field from aiwf.yaml
  (textual line-strip; idempotent; reported in step ledger).
  Closes the upgrade-flow gap where doctor only emitted a soft
  "note:" the user reported missing among other output.

- aiwf render --help / -h / help now prints verb usage instead
  of "unknown subcommand --help".

- New tools/internal/skills/embedded/aiwf-render/SKILL.md so AI
  assistants can discover the verb through the standard channel
  (G21 discoverability).

- Sidebar reordered: "Project status" precedes "Overview"; the
  GOVERNANCE label is removed; SidebarData.IsCurrentIndex marks
  the Overview link aria-current on index.html.

- Index page <h1> + <title> read "Overview" instead of
  "Governance" (the kicker still says "aiwf · governance" since
  it labels the site, not the page).

- README's HTML-render section gains one paragraph clarifying
  that the page covers both governance (entity tree) and status
  (in-flight work, decisions, gaps, recent activity) surfaces.
  Sidebar / status page / dark mode / logo are deliberately NOT
  mentioned in the README per scope.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
```

## After the commit

1. Update `docs/pocv3/plans/governance-html-plan.md` §11 status table footnote to mention the post-step-7 additions (logo, sidebar, status page, palette, cache-busting). One short bullet block.
2. Update `docs/pocv3/plans/poc-plan.md` if the I3 row needs touchup.
3. Tag `v0.2.0` (manual: `git tag v0.2.0 && git push origin v0.2.0`). Confirm `go install github.com/23min/ai-workflow-v2/tools/cmd/aiwf@v0.2.0` resolves.
4. Verify `aiwf doctor --check-latest` reports the new tag (proxy may take a few minutes to cache).

## Things to NOT do

- Don't refactor `tools/cmd/` → `cmd/` or extract test helpers — those are the deferred refactors from the earlier discussion (separate v0.3.0 candidate).
- Don't touch the existing `:has()` CSS rules; they're load-bearing.
- Don't add raster image assets — SVG is the kernel's image story.
- Don't bump go directive past 1.22 (tools/CLAUDE.md rule).

## Files most likely to touch

- `tools/internal/config/config.go` (new helper)
- `tools/internal/config/config_test.go` (helper test)
- `tools/internal/initrepo/initrepo.go` (new step + wire into RefreshArtifacts)
- `tools/internal/initrepo/initrepo_test.go` (new tests)
- `tools/cmd/aiwf/render_cmd.go` (--help handling)
- `tools/cmd/aiwf/render_site_cmd_test.go` (count + filename assertions, "Overview" not "Governance")
- `tools/cmd/aiwf/render_templates_test.go` (any "Governance" assertions)
- `tools/internal/htmlrender/pagedata.go` (`SidebarData.IsCurrentIndex`)
- `tools/internal/htmlrender/default_resolver.go` + `tools/cmd/aiwf/render_resolver.go` (set IsCurrentIndex; also change Title default to "Overview")
- `tools/internal/htmlrender/embedded/_sidebar.tmpl` (reorder + drop label + Overview aria-current)
- `tools/internal/skills/embedded/aiwf-render/SKILL.md` (new)
- `tools/e2e/playwright/tests/render.spec.ts` ("Overview" assertion + aria-current expected on index)
- `README.md` (one paragraph)

Total: small, contained, all reversible. Should land in one commit, ~15 files touched, ~150 lines net change.
