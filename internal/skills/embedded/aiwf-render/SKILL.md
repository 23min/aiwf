---
name: aiwf-render
description: Use when the user asks to render the planning state as a static HTML site, publish governance views, or generate the project status page.
---

# aiwf-render

`aiwf render --format=html` produces a self-contained directory of HTML files: `index.html` (epics table), one page per epic and milestone, plus `status.html` (the same project snapshot `aiwf status` carries, browser-formatted). A single embedded stylesheet ships alongside; no JS, no runtime, no external assets.

## What it does

Walks the planning tree, then writes:

- `index.html` — every epic with the `met / (total - cancelled)` AC rollup and a findings rollup.
- One `E-NN.html` per epic — milestones table, dependency edges, linked entities, recent activity.
- One `M-NNN.html` per milestone — six tabs (Overview, Manifest, Build, Tests, Commits, Provenance). Tab show/hide is `:target`-driven so per-tab URLs (`M-007.html#tab-build`) are bookmarkable.
- `status.html` — the in-flight epics + open decisions + open gaps + recent activity view (same `buildStatus` helper as the markdown `aiwf status`).
- `assets/style.css` — one stylesheet shared across every page.

Read-only — no commit. Re-running into the same out_dir overwrites the files; rendering twice produces byte-identical output.

## When to use

| User says | Run |
|---|---|
| "render the governance HTML" | `aiwf render --format=html` |
| "publish the status page" | `aiwf render --format=html` (status.html is part of the standard render) |
| "build the static site" | `aiwf render --format=html --out <dir>` |
| "show me the rendered tree" | `aiwf render --format=html` then open `site/index.html` in the browser |

For the markdown roadmap (epics + milestones table), use `aiwf render roadmap` instead — different surface, also a `render` subcommand.

## Configuration

Lives in `aiwf.yaml`:

```yaml
html:
  out_dir: site            # default; relative to the repo root
  commit_output: false     # default; framework-managed gitignore covers out_dir/
```

`out_dir` is the directory the renderer writes into; absolute paths are honored, relative paths resolve against the repo root. `commit_output: false` (default) means the framework adds `<out_dir>/` to `.gitignore` on the next `aiwf init`/`aiwf update`. Set `commit_output: true` and re-run `aiwf update` to remove the gitignore line and commit the rendered HTML alongside source.

Most projects publish via CI rather than committing the output. Four deployment patterns (local, GitHub Pages artifact, `gh-pages` branch, committed-to-source) are documented in [`docs/pocv3/plans/governance-html-plan.md`](../../docs/pocv3/plans/governance-html-plan.md) §2.

## Flags

| Flag | Effect |
|---|---|
| `--format=html` | required; selects the static-site surface |
| `--out <dir>` | override `aiwf.yaml.html.out_dir` for this invocation |
| `--scope <id>` | reserved (incremental render; not yet implemented) |
| `--no-history` | reserved (skip git-log walks per page; not yet implemented) |
| `--pretty` | indent the JSON envelope on stdout |

The verb always emits a JSON envelope on stdout: `{ "result": { "out_dir": "<abs>", "files_written": N, "elapsed_ms": M } }`. Useful for CI scripts.

## After running

Open `index.html` in a browser to confirm the render is what the user expected. Common follow-ups:

- If the user wants the page on the web, point them at the four deployment patterns.
- If the rendered tree looks wrong (missing page, missing tab content), `aiwf check` is the first stop — the renderer is a pure projection of the tree, so render-side issues are usually validation findings the user hasn't seen yet.
- The default `out_dir` is gitignored; if the user expected files to commit, check `aiwf.yaml.html.commit_output`.
