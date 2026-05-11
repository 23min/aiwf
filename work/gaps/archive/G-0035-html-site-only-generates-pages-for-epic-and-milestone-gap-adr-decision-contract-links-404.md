---
id: G-0035
title: HTML site only generates pages for epic and milestone — gap/ADR/decision/contract links 404
status: addressed
addressed_by_commit:
  - d1bf1e1
---

Resolved in commit `(this commit)` (fix(aiwf): G35/G36 — render gap/ADR/decision/contract pages with HTML markdown bodies). New shared `entity.tmpl` covers the four kinds without specialized rendering; `htmlrender.Render` iterates over `KindADR`/`KindGap`/`KindDecision`/`KindContract` after the existing epic/milestone loops, calling a new `renderEntity` that pulls per-page data from a new `PageDataResolver.EntityData(id)` method. Default resolver returns frontmatter + sidebar (no body — that's a cmd-side concern); cmd-side `renderResolver.EntityData` reads the body from disk, parses sections via the new `entity.ParseBodySectionsOrdered`, and surfaces forward+reverse linked entities and recent history. Tests: per-kind structural assertions on `G-001.html`, `ADR-0001.html`, `D-001.html`, `C-001.html` (kicker carries the kind label, `<h1>` carries the title, sidebar link back to index present). Smoke: rendered a synthesized kernel-style tree and walked every kind's page in a browser. Pairs with G36 — fixing both in the same commit means the new gap/ADR/decision/contract pages don't ship with the same rendered-as-raw-text defect.

---

<a id="g36"></a>
