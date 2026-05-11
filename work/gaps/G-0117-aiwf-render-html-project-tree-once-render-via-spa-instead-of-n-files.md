---
id: G-0117
title: 'aiwf render html: project tree once, render via SPA instead of N files'
status: open
---
## What's missing

`aiwf render html` produces one HTML file per entity (`E-NNNN.html`, `M-NNNN.html`, `G-NNNN.html`, …) plus two top-level pages and ten per-kind index pairs. The `PageDataResolver` in [cmd/aiwf/render_cmd.go](../../../cmd/aiwf/render_cmd.go) is called once per entity and reaches into git for history, scopes, and trailers each time — so wall-clock cost scales as *N entities × per-entity git scans*, not as template execution. On the kernel's own tree the render is already noticeably slow, and the curve gets worse with every entity added.

The render output is also overbuilt for the consumer model: a static site backed by N HTML files only pays off if those pages are crawled, deep-linked from outside, or served from a remote host. In practice the rendered site is opened locally (or in a single shared location) and navigated by humans clicking around the index/sidebar. The "one file per entity" shape is paying the cost of a permalink surface that nobody is using.

## Why it matters

Render time is the friction point that determines whether `aiwf render html` is run *every* time the planning tree changes (cheap → routine) or *occasionally* when someone needs the site (expensive → stale). Today it tilts toward the second; a faster render would close the loop between editing the tree and seeing the result.

The render verb is also one of the few aiwf surfaces a non-developer reaches for — making it slow undermines the "narrative shape of the tree" affordance the HTML view is meant to provide.

## Proposed shape — recommended: SPA with embedded JSON (Option A)

Emit a single `index.html` shell with the entire projected tree inlined as `<script id="data" type="application/json">…</script>`. A small vanilla-JS file (no framework) does hash routing (`#/E-0030`, `#/M-0090/AC-2`) and renders views from the JSON in the DOM. Total output: ~4 files (`index.html`, `assets/style.css`, `assets/app.js`, optional sibling `data.json` for tooling).

Render cost collapses to *load tree once → project once → write 4 files*. The seam already exists: [internal/htmlrender/pagedata.go](../../../internal/htmlrender/pagedata.go) is essentially the view-model layer to serialize, and `PageDataResolver` is the chokepoint where the per-entity git scans live today.

Staging suggestion: land a `aiwf render json` verb first (project the tree once into a typed JSON envelope, no HTML behavior change). That JSON becomes a contract — useful as a machine-readable surface in its own right (CI scripts, downstream tooling), and the SPA renderer becomes a second milestone that reads from it. The existing HTML renderer keeps working in parallel during transition.

Two real costs to weigh:

- **Permalink shape changes**: `E-0030.html` → `index.html#/E-0030`. Cross-references in `docs/`, skills, and CHANGELOG entries need a sweep before flipping.
- **JS as a dependency**: the kernel is currently zero-JS by design (CSS `:target` + `:has()` does tab switching). Introducing a JS file is a real category shift — small, but worth an ADR-shaped note in the rationale.

Embedding the JSON (rather than fetching a sibling `data.json` at runtime) sidesteps the `file://` CORS block — browsers refuse `fetch()` over `file://`, but inline `<script type="application/json">` reads from the DOM with no fetch.

## Alternatives considered

**Option B — Hybrid: static shell + dynamic detail.** Keep `index.html` and `status.html` as static HTML (landing page renders with JS disabled). Detail pages collapse into one client-rendered view over embedded JSON. Marginally better than A for the JS-off case, but doubles the rendering code paths and the maintenance surface. Probably not worth it unless graceful degradation under "no JS" is an explicit requirement.

**Option C — Keep static, just cache the resolver.** Memoize git scans inside `PageDataResolver`, batch the `git log` reads, or pre-build a per-entity history index in one pass. Smallest possible change; still emits N files; doesn't address the "one-page" shape goal. Useful as a stopgap if the SPA work can't land soon — and worth doing anyway as the first measurement of where the time actually goes.

The right next step is probably to profile first (confirm the resolver is the dominant cost), then commit to A in two milestones (JSON projection verb, then SPA renderer). Option C may land first as a quick measurement-then-cache pass if the profile points there.
