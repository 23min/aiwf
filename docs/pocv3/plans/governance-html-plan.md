# Governance HTML render plan

**Status:** proposal · **Audience:** PoC iteration I3 (continuation of [`poc-plan.md`](poc-plan.md) sessions 1–5, [`contracts-plan.md`](contracts-plan.md) I1, [`acs-and-tdd-plan.md`](acs-and-tdd-plan.md) I2, and [`provenance-model-plan.md`](provenance-model-plan.md) I2.5).

**Dependencies.** I3 depends on two I2/I2.5 deliverables: (a) the reverse-reference index (`acs-and-tdd-plan.md` step 11) — used by the render's `referenced_by` field and by the scope reachability check; (b) the full provenance model (I2.5) — the Provenance tab renders scopes-as-section and the timeline carries scope chips. Without I2.5, the Provenance tab is meaningfully degraded (single-actor history with no scope grouping). Land both before starting I3 step 5 (templates).

This document plans an HTML render surface for aiwf — a static-site generator that produces a per-repo governance page from canonical planning state, derived from data the kernel already holds.

The motivation is concrete: when planning state is forced through a visual representation, drift and inconsistency become obvious in ways they aren't in raw markdown. The render is the legibility layer; the kernel remains the truth layer.

This factoring is the load-bearing call: aiwf generates a directory of static HTML files with no runtime, no server, no auth, no database. Read-only by design. The HTML render is *another* renderer alongside `aiwf status --format=md` and `aiwf show --format=json`, not a separate product.

For the design context that justifies this shape, see [`design-decisions.md`](../design/design-decisions.md) §"Governance HTML render (added in I3)". For the kernel's AC and TDD model that this surface renders, see [`acs-and-tdd-plan.md`](acs-and-tdd-plan.md).

---

## 0. Preconditions

Land before starting any I3 step:

| Prerequisite | Where defined | Why I3 needs it |
|---|---|---|
| **I2 step 11 — reverse-reference index on `aiwf show`** | [`acs-and-tdd-plan.md`](acs-and-tdd-plan.md) §11 step 11 | Drives the `referenced_by` field consumed by the epic page's "Linked entities" section and the milestone Manifest tab's cross-references. Also a transitive dependency via I2.5 (scope reachability uses the same index). |
| **I2 steps 1–10 (acceptance criteria + TDD)** | [`acs-and-tdd-plan.md`](acs-and-tdd-plan.md) §11 steps 1–10 | The Manifest, Build, and Tests tabs all render the AC + TDD-phase model. The `aiwf-to:` trailer feeds the Build tab's phase chips. |
| **I2.5 (provenance model in full)** | [`provenance-model-plan.md`](provenance-model-plan.md) | The Provenance tab renders scope-as-section + chronological timeline with scope chips. Without I2.5, the tab degrades to single-actor history with no scope grouping — and the `--audit-only` events from I2.5 step 5b have nowhere to render their distinct chip. Land I3 step 5 (templates) only after I2.5 closes. |

If a template is started before its precondition lands, the test fixture will fail to produce the expected HTML (missing fields render as empty strings; missing scope events render as bare actor rows). The failure mode is "test mismatch," not "silent wrong output," but sequencing keeps the test surface honest.

### Within-iteration build order

The numbered steps in §9 are not strictly sequential. The actual DAG:

```
1 (JSON completeness on aiwf show)         ← depends on I2 step 11 (referenced_by)
   │
   ├── 2 (aiwf-tests: trailer + warning)   ← independent of 1; can land in parallel
   │
   └── 3 (render package skeleton)         ← depends on 1 (consumes the extended ShowView)
            │
            └── 4 (aiwf render verb)        ← depends on 3 (uses the page generator interface)
                     │
                     └── 5 (templates + CSS)  ← depends on 4 (verb writes the files); also depends on I2.5 in full for the Provenance tab
                              │
                              └── 6 (cross-cutting render details)  ← polish on top of 5
                                       │
                                       └── 7 (documentation)         ← reflects the now-stable surface
```

**Suggested commit cadence:** 1 → 2 (in parallel with 1, can land first) → 3 → 4 → 5 → 6 → 7. Each step is one commit; the templates step (5) is the largest and can be split per template (`index.tmpl`, `epic.tmpl`, `milestone.tmpl`) if it grows beyond a single reviewable diff.

**Hard gate before step 5:** I2.5 must be closed. The Provenance tab cannot be built against a partial I2.5 surface — the scope-as-section render assumes every authorize commit, every `aiwf-scope:` event, and every `aiwf-scope-ends:` trailer is in place.

---

## 1. Site shape

| Path | Contents |
|---|---|
| `index.html` | Top-level — epics list, status pills, AC met-rollup per epic, recent activity |
| `<epic-id>.html` | Epic overview — milestones table, dependency DAG, linked entities, history |
| `<milestone-id>.html` | Per-milestone — six tabs (Overview, Manifest, Build, Tests, Commits, Provenance). ACs are rendered inline in the Manifest tab and addressable via `#ac-N` anchors. |
| `assets/style.css` | Single hand-written stylesheet, embedded in binary, no JS |

All pages are rendered from `aiwf show <id> --format=json` data plus `git log`. No runtime; no JS; no client-side rendering.

---

## 2. Where the output goes

### Default: `site/` at repo root, gitignored

Standard static-site-generator convention (Hugo, Jekyll, Astro, mdBook, Docusaurus all use the same pattern). Recognizable to anyone who has worked with one. Clean separation of source (`work/`, `docs/`) from generated output.

### Why not `work/html/`

- aiwf's `work/` directory is canonical planning state; mixing source and generated artifacts is the same anti-pattern as committing `node_modules/` or `dist/`.
- Every regeneration produces diff churn that obscures real changes in PRs ("M-007.html: 3,012 changes" tells a reviewer nothing).
- Merge conflicts in generated HTML are pure noise — they always exist when two branches both regenerate, and resolving them is busywork.
- Forces a render step on every commit to keep them current, or you accept staleness.

### Configuration

| Mechanism | Behavior |
|---|---|
| `aiwf.yaml.html.out_dir` | Path relative to repo root. Default `site`. |
| `aiwf.yaml.html.commit_output` | Bool, default `false`. Expresses user intent to commit the render output. The gitignore block is a *derived* artifact controlled by this field — see below. |
| `--out <path>` flag on `aiwf render` | Overrides the YAML's `out_dir`. |
| `aiwf init` and `aiwf update` | When `commit_output: false` (default), both verbs add `out_dir` to the framework-managed `.gitignore` block alongside `.claude/skills/aiwf-*`, etc. When `commit_output: true`, both verbs *remove* `out_dir` from the managed block (and never re-add it on subsequent runs). The block is idempotent — re-running either verb converges to the same state for a given `aiwf.yaml`. |

User intent lives in `aiwf.yaml`; the gitignore block is derived. This avoids the failure mode where a user manually unignores `out_dir`, then loses that change the next time `aiwf update` runs.

### Deployment patterns

The render output is portable; aiwf does not own the deployment pipeline. Four patterns cover the realistic cases.

**1. Local only.** `aiwf render && open site/index.html`. Personal viewing for solo work or quick "does this render look right?" checks. No CI, no commits, no sharing beyond screenshots or screen-share. The fastest feedback loop and the right default while iterating on what the page should look like.

```bash
aiwf render --format=html --out site/
open site/index.html        # macOS
xdg-open site/index.html    # Linux
```

**2. GitHub Pages artifact (recommended for shared visibility).** A CI action runs `aiwf render` on push to `main` and uploads the directory directly to Pages. No publishing branch in the repo; Pages serves the artifact. Source `main` stays clean — no render churn anywhere.

```yaml
# .github/workflows/render.yml
on: { push: { branches: [main] } }
jobs:
  render:
    runs-on: ubuntu-latest
    permissions: { contents: read, pages: write, id-token: write }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go install github.com/<org>/aiwf/cmd/aiwf@latest
      - run: aiwf render --format=html --out site/
      - uses: actions/upload-pages-artifact@v3
        with: { path: site }
      - uses: actions/deploy-pages@v4
```

**3. `gh-pages` branch (traditional Pages pattern).** A CI action publishes the rendered output to a separate `gh-pages` branch. The branch accumulates render churn, but nobody opens PRs against it, so the churn is invisible to code review. This is the classic Hugo/Jekyll/Docusaurus pattern; use it when you can't or don't want to use the artifact API (e.g., older GitHub Enterprise, mirroring to other static hosts).

```yaml
- run: aiwf render --format=html --out site/
- uses: peaceiris/actions-gh-pages@v3
  with:
    github_token: ${{ secrets.GITHUB_TOKEN }}
    publish_dir: ./site
```

**4. Committed to source tree.** If GitHub Pages is disabled in your org, or you want HTML accessible via `raw.githubusercontent.com` without Pages, commit the output to `main:docs/site/`. Steps: set `aiwf.yaml.html.out_dir: docs/site` and `aiwf.yaml.html.commit_output: true`; run `aiwf update` (the managed gitignore block updates automatically — no hand-edit required); add a pre-commit hook that runs `aiwf render` so the committed copy stays fresh. Determinism (§8) bounds the diff churn to actual content changes. Use this only when the other patterns don't apply — it's the most maintenance for the least cleanliness.

---

The key insight: "gitignored from source" and "available on the web" are not in tension — that's exactly what CI was built for. The trap is conflating *committed* with *published*. Once you commit the render, every regeneration bloats your reviewable diff; once you publish via CI, the source repo stays clean and the published version stays fresh.

Beyond GitHub Pages, the same `site/` directory drops cleanly into S3 (`aws s3 sync site/ s3://bucket/`), Cloudflare Pages, Netlify, GitLab Pages, an internal nginx, or any other static host. aiwf's job stops at writing the directory.

---

## 3. Page specs

### 3.1 `index.html`

| Section | Source |
|---|---|
| Header | repo name, last-updated timestamp (most recent commit on `aiwf-*` trailer set) |
| Epics table | id, title, status pill, milestone count, AC met-rollup `met / (total - cancelled)` (e.g., `12/18`), last activity date |
| Findings rollup | total `aiwf check` findings by severity, with link to a separate `findings.html` if non-trivial |

### 3.2 `<epic-id>.html` — epic overview

| Section | Source |
|---|---|
| Header | epic frontmatter (id, title, status), AC met-rollup across all milestones, recent activity timestamp |
| Goal & Scope | epic body (`## Goal`, `## Scope`, `## Out of scope`) |
| Milestones | rows: id, title, status pill, AC progress (`3/5`), TDD policy, last activity |
| Dependency DAG | indented list with Unicode arrows representing `depends_on` edges among this epic's milestones |
| Linked entities | ADRs, decisions, gaps, contracts that reference the epic or any of its milestones (forward + reverse via the existing reference graph) |
| Recent activity | last N events from `aiwf history E-NN`, actor-attributed |

### 3.3 `<milestone-id>.html` — six tabs

Each tab is emitted as a `<section id="tab-...">`. Tab nav is a sticky in-page strip of `<a href="#tab-...">` links. CSS shows the section that matches `:target`; the bare URL (no fragment) shows Overview by default via the standard CSS sibling-selector trick:

```css
section[data-tab] { display: none; }
section[data-tab]:target { display: block; }
/* Default: Overview shows when nothing is targeted. */
section[data-tab="overview"] { display: block; }
section[data-tab]:target ~ section[data-tab="overview"] { display: none; }
```

Result: per-tab URLs are bookmarkable and shareable; browser back/forward switches tabs naturally; no JS; screen readers see all six sections in DOM order.

| Tab | Source |
|---|---|
| **Overview** (default; bare-URL section) | milestone frontmatter + `## Goal` body + linked decisions + status pill + AC progress |
| **Manifest** | ACs (frontmatter + matching `### AC-N — <title>` body sections) + progress gauge from met-ratio. Each AC is wrapped in `<section id="ac-N">` for cross-page anchor links (e.g., `M-007.html#ac-1`). |
| **Build** | per-AC TDD timeline from phase trailers; AC steps as vertical stack with phase chips (red/green/refactor/done) |
| **Tests** | per-AC test metrics from `aiwf-tests:` trailer (pass/fail/skip counts), latest per AC (where "latest" = first hit when walking `aiwf history M-NNN/AC-N`); rollup at milestone level. Header carries a **policy badge** showing `strict` (when `aiwf.yaml.tdd.require_test_metrics: true`) or `advisory` (default). In strict mode, ACs missing the trailer render as a red row matching the `acs-tdd-tests-missing` finding; in advisory mode, missing-trailer ACs render as a neutral "no metrics recorded" row with no warning. |
| **Commits** | `git log` filtered by `aiwf-entity: M-NNN` and composite-id trailers |
| **Provenance** | scope-as-section. Top: scopes table for this entity (auth SHA short form, agent, principal, opened, current state `active`/`paused`/`ended`, end date, event count). Below: chronological event timeline; each row carries a `[scope-id]` chip when `aiwf-authorized-by:` is present, no chip when direct. Pause/resume/end events render as scope-state changes (`[E-03 paused]`, `[E-03 resumed]`, `[E-03 ended]`). `--force` overrides highlighted with reason. Actor column uses `principal via agent` syntax when they differ. Per [`provenance-model.md`](../design/provenance-model.md). |

Per-AC content is rendered inline in the Manifest tab and addressable via `#ac-N` anchors (e.g., `M-007.html#ac-1`). A separate per-AC page is out of scope until real navigation friction shows up.

---

## 4. The `aiwf-tests:` trailer

A new optional commit trailer carrying per-cycle test metrics:

```
aiwf-verb: promote
aiwf-entity: M-007/AC-1
aiwf-to: green
aiwf-actor: ai/claude
aiwf-tests: pass=12 fail=0 skip=0
```

### Format

Loose `key=value` pairs separated by whitespace. Recognized keys:

| Key | Meaning |
|---|---|
| `pass` | passing test count |
| `fail` | failing test count |
| `skip` | skipped test count |
| `total` | total run count (optional; derivable) |

Unknown keys are tolerated and ignored — keeps the format extensible without a kernel change.

### Who writes it

The kernel owns the write path. Phase-promoting verbs accept a typed `--tests` flag:

```
aiwf promote M-007/AC-1 --phase done --tests "pass=12 fail=0 skip=0"
```

The kernel parses the flag, validates each `key=value` pair against the recognized set above (write-strict: unknown keys are rejected at the verb boundary; non-negative integers required), and writes the trailer in the same commit as the phase promotion. Read-side parsing remains tolerant — unknown keys in already-committed trailers are ignored, so future trailer extensions don't break old readers.

The rituals plugin's `wf-tdd-cycle` skill invokes `aiwf promote --tests …` rather than constructing the trailer itself; this keeps a single write path. Format-agnostic at the *plugin* layer — the skill captures whatever its language's test runner reports (`go test -json`, JUnit XML, pytest, Jest) and reduces to the three integers before calling the kernel.

Solo users without the rituals plugin and CI scripts that want to record metrics from a `go test -json` parse can use the kernel flag directly.

### Aggregation

- **Per AC:** the **first commit returned by `aiwf history M-NNN/AC-N`** that carries an `aiwf-tests:` trailer is authoritative. Defining "latest" via the `aiwf history` iterator (rather than wall-clock or committer-date) makes aggregation rebase- and amend-stable: any rewrite that preserves the topology preserves the metrics.
- **Per milestone:** sum across ACs (each AC's first-with-trailer in history order).
- **Per epic:** sum across milestones.

The Build tab and the Tests tab share this iterator — they cannot disagree on which commit is "latest" because they read from the same source.

### Validation (opt-in)

New finding: `acs-tdd-tests-missing` (warning), gated on a new YAML field.

| Severity | Trigger |
|---|---|
| warning | `aiwf.yaml.tdd.require_test_metrics: true` AND milestone `tdd: required` AND AC `tdd_phase: done` AND no `aiwf-tests:` trailer on the first commit returned by `aiwf history` for that AC |

The YAML field is `false` by default. Without it, the trailer is purely informational metadata and `aiwf check` emits no finding for its absence — this decouples kernel correctness from the rituals-plugin install state and keeps the warning meaningful for users who explicitly opted into stricter governance. The Tests tab's policy badge (§3.3) renders the strict-vs-advisory mode visibly so the user can always tell whether missing metrics are a finding or just absent.

The fix path when the warning fires in strict mode: re-run the cycle through the kernel verb (`aiwf promote --tests …`), or, if the milestone reached `done` via a verb that didn't carry the trailer, set `require_test_metrics: false` on the milestone-by-milestone basis (out of scope for I3) or accept the warning as a known incomplete record.

### Why a trailer, not a file

| Property | File per milestone | Trailer per commit |
|---|---|---|
| Inherently fresh at commit time | snapshot ages | commit-atomic |
| Per-AC granularity | needs schema | via composite id |
| Format-agnostic | needs picking | free-form string |
| New file shape kernel must know | yes | no |
| KISS | medium | tiny |

If per-test detail is later needed (which test ran, duration, etc.), the upgrade path is an `aiwf-tests-detail:` trailer pointing to a per-AC sidecar file, or a normalized JSON. The metrics-tab use case does not need that yet.

---

## 5. The `aiwf render` verb

```
aiwf render --format=html [--out <dir>] [--scope <id>] [--no-history] [--pretty]
```

| Flag | Behavior |
|---|---|
| `--format=html` | Required. HTML is one render format; `--format=md` produces a single-doc markdown rollup; `--format=json` already exists via `aiwf show`. |
| `--out <dir>` | Output directory. Overrides `aiwf.yaml.html.out_dir`. |
| `--scope <id>` | Render only this entity and its referenced children. Useful for fast iteration. |
| `--no-history` | Skip git-log walks per page; faster but Build / Commits / Provenance tabs are degraded. |
| `--pretty` | Standard JSON envelope formatting on stdout. |

Read-only: no commit produced. Not in the "every mutating verb produces exactly one commit" set.

Standard JSON envelope on stdout:

```json
{
  "tool": "aiwf",
  "version": "0.x.0",
  "status": "ok",
  "findings": [],
  "result": {
    "out_dir": "site",
    "files_written": 47,
    "elapsed_ms": 234
  },
  "metadata": { "correlation_id": "..." }
}
```

---

## 6. Templates and assets

| Asset | Location |
|---|---|
| HTML templates | `internal/render/embedded/*.tmpl` (Go `html/template`) |
| CSS | `internal/render/embedded/style.css` (single hand-written stylesheet) |
| Optional favicon | `internal/render/embedded/favicon.svg` |

All embedded in the binary via `embed.FS`. No runtime asset paths to resolve. No external CDN dependency.

CSS guidelines:

- Hand-written; no Tailwind, no Bootstrap, no framework.
- CSS custom properties for color tokens so `prefers-color-scheme: dark` is a 10-line addition (optional in I3).
- Tables, lists, status pills, tab nav. That's it. Total stylesheet target: under 5 KB.

Templates produce semantic HTML — `<nav>`, `<main>`, `<section>`, `<article>`. Skinnable later if needed; not in scope now.

---

## 7. JSON-completeness preconditions on `aiwf show`

The HTML render is a thin templating layer over `aiwf show <id> --format=json`. An audit of the current envelope (post-I2 step 7c, commit `3f743a8`) shows three capabilities the render needs:

| Capability | Status | Lands in |
|---|---|---|
| Entity frontmatter (full, including `acs[]`) | exists | — |
| `aiwf check` findings scoped to the entity (and sub-entities for milestones) | exists (`show_cmd.go` filters by entity id, including composite ids) | — |
| History from `aiwf history <id>` with parsed trailers | exists for the I2 trailer set; extend trailer parser for `aiwf-tests:` | I3 step 1 |
| **Reverse references** (`referenced_by`): which other entities reference this one | missing | **I2** (lands ahead of I3 — `aiwf check` audits and `aiwf show` benefit independently of the render) |
| **Body sections** parsed into named blocks (`goal`, `acs.AC-N.description`, `work_log`, `decisions`, `validation`, `deferrals`, `reviewer_notes`) | partial (heading walker exists in `roadmap/`; `ShowView` carries no body content) | **I3 step 1** |
| Forward references (`references`) | already structured in `Entity.References[]` and emitted by `aiwf check`'s `collectRefs` | trivial to surface on `ShowView` if not already; covered by step 1 |

The reverse-ref index is added to [`acs-and-tdd-plan.md`](acs-and-tdd-plan.md) §11 as a new I2 step (invert the existing forward-ref collection in `check/check.go`, expose as `referenced_by` on `ShowView`). Body section parsing lands as I3 step 1 alongside the `aiwf-tests:` trailer parser.

### AI-discoverability rule

Every new envelope field, body-section name, finding code, trailer key, flag, and YAML field must be reachable through the channels an AI assistant routinely consults: `aiwf <verb> --help`, the embedded skills under `.claude/skills/aiwf-*`, the kernel's CLAUDE.md, or the design docs cross-referenced from it. If an AI assistant has to grep source code to learn a kernel capability, the capability is undocumented. This is a general rule (now in CLAUDE.md and design-decisions.md) and applies to every step in §9 below.

---

## 8. Cross-cutting render conventions

Design constraints derived from the kernel's first principles (one truth source, mechanical validation, queryable provenance):

- **One progress metric per surface.** AC met-ratio (`met / (total - cancelled)`) is the only milestone-level progress number. No competing "steps complete" or "jobs done."
- **Every date labeled with source.** "First commit", "Last activity", "AC met", "Decision approved" — never bare timestamps without role.
- **Status pill colors driven by `aiwf check` findings.** Green when checks pass; yellow when warnings; red when errors. No second pill that contradicts (e.g., red status with green test count).
- **Forced transitions visually distinct.** Provenance tab marks `aiwf-force` events with a distinct chip and surfaces the reason inline.
- **Linked entities resolved once.** Don't show the same review/decision in two tabs. Provenance is the home for ratification events; Overview links to the latest accepted decision but doesn't repeat its body.
- **Test-metrics policy is rendered explicitly.** The Tests tab carries a `strict` / `advisory` badge so a reader can tell whether missing metrics are a finding or just absent.

### Determinism

The render is a pure function of the input tree. The same tree must produce byte-identical output on every run — a load-bearing property for deployment pattern 4 (committed to source) and for golden-HTML tests in step 5. Three rules:

- **No wall-clock timestamps.** Every header/footer date comes from commit metadata (the latest `aiwf-*` trailer commit for the entity, or the most recent commit on the repo for the index page). Never `time.Now()`.
- **Sorted map iteration.** Go's `range` over maps is intentionally randomized. Templates iterate via a sorted-keys helper; any new map iteration site uses the same helper.
- **Sorted directory enumeration.** `os.ReadDir` (sorted since Go 1.16) and `filepath.WalkDir` are used everywhere; no `filepath.Walk` (older API, fewer guarantees).

A "render twice, byte-compare" test in step 4 pins the property.

---

## 9. Build plan

### Step 1 — JSON completeness on `aiwf show`

- [ ] Body section parser — named blocks per kind (no full markdown parser; section-heading walker is enough). Reuses the heading-walker shape already in `roadmap/extractSection`. Output is a `Body` struct on `ShowView` carrying named blocks: `goal`, `scope`, `out_of_scope`, `acs[N].description`, `work_log`, `decisions`, `validation`, `deferrals`, `reviewer_notes` (per kind).
- [ ] `ShowView` extended: `Body` struct as above; `referenced_by []string` populated from the I2 reverse-ref index.
- [ ] Trailer parser extended for `aiwf-tests:` (key=value tokens; tolerant on read).
- [ ] `aiwf show --help` enumerates the body-section names and the new envelope fields. Embedded skill `aiwf-show` (or equivalent) updated to match.
- [ ] Tests: golden JSON files per entity kind covering Body, `referenced_by`, and `aiwf-tests:` parsing.

### Step 2 — `aiwf-tests:` trailer (kernel write path + opt-in warning)

- [ ] Trailer key registration in `internal/gitops/` for read and write.
- [ ] `--tests "key=value …"` flag on every phase-promoting verb (`aiwf promote --phase`, `aiwf add ac` when seeding `red`). Write-strict validation: keys ∈ {`pass`, `fail`, `skip`, `total`}; values are non-negative integers; unknown keys rejected at the verb boundary.
- [ ] Verb writes the trailer in the same commit as the phase promotion (one commit, no separate write).
- [ ] Aggregation helpers: `LatestTestsForAC(history, compositeID)` returning the first hit when walking `aiwf history`; `RollupTestsForMilestone(...)`, `RollupTestsForEpic(...)`.
- [ ] New `aiwf.yaml.tdd.require_test_metrics` field (bool, default `false`); round-trip in `aiwfyaml/`.
- [ ] `acs-tdd-tests-missing` warning in `aiwf check`, gated on `require_test_metrics: true`.
- [ ] Rituals plugin update (in `ai-workflow-rituals`): `wf-tdd-cycle` calls `aiwf promote --tests …` rather than constructing the trailer itself.
- [ ] `aiwf promote --help` documents the `--tests` flag including the recognized keys; `aiwf check --help` mentions `acs-tdd-tests-missing` and the gating field.
- [ ] Tests: trailer parsing (read-loose), kernel-write validation (write-strict), aggregation rebase-stability (rebase a fixture branch; assert metrics survive), opt-in warning fires only when YAML field is `true`, kernel verb works without rituals plugin installed.

### Step 3 — Render package skeleton

- [ ] `internal/render/` package with template loading from `embed.FS`.
- [ ] Page generator interface: `RenderIndex`, `RenderEpic`, `RenderMilestone`. (No `RenderAC` — per-AC content is inline in the milestone Manifest tab; addressable via `#ac-N` anchor.)
- [ ] Path resolver: id → output filename. No subdirectory scheme; no composite-id pages. Composite-id links are `M-NNN.html#ac-N`.
- [ ] Sorted-keys helper for all map iteration in templates.
- [ ] History-walk strategy: documented as one walk per page in I3; the optimization to "single log walk per render, in-memory `entityID → []commit`" is a known escape hatch for when render time becomes a real cost. Not built now.
- [ ] Tests: round-trip a fixture tree to a directory; assert file presence and link integrity (every internal link resolves to a written file or anchor).

### Step 4 — `aiwf render --format=html` verb

- [ ] Flag parsing (`--format`, `--out`, `--scope`, `--no-history`, `--pretty`).
- [ ] `aiwf.yaml.html.out_dir` field (default `site`) and `aiwf.yaml.html.commit_output` (bool, default `false`); both round-tripped in `aiwfyaml/`.
- [ ] `aiwf init` and `aiwf update` consult `commit_output`: when `false` the framework-managed gitignore block contains `out_dir`; when `true` the block does not contain `out_dir` (and previous-run state is reconciled, not duplicated). Idempotent; converges to the same state for a given `aiwf.yaml`.
- [ ] JSON envelope on stdout per §5: `out_dir`, `files_written` (count), `elapsed_ms`. No path list (deferred until a real consumer asks).
- [ ] **Determinism test:** render a fixture tree twice into separate dirs; assert byte-identical output across both runs.
- [ ] Gitignore tests: (a) default-false adds and keeps the entry across re-runs, (b) flipping to `true` removes it on the next `aiwf update`, (c) flipping back to `false` re-adds it, (d) idempotence on repeat invocations of either verb.
- [ ] `aiwf render --help` documents both YAML fields, the flag set, and the envelope shape; `aiwf init --help` and `aiwf update --help` mention the gitignore behavior.
- [ ] Tests: end-to-end render against fixture tree; HTML well-formedness; link integrity from §9 step 3 carried through.

### Step 5 — Templates and CSS

- [ ] `index.tmpl` — epics list with `met / (total - cancelled)` rollup column.
- [ ] `epic.tmpl` — overview, milestones table, DAG, linked entities (forward + reverse via `referenced_by`), history.
- [ ] `milestone.tmpl` — six tabs as `<section>` elements with `:target`-driven show/hide and the CSS-default-tab sibling trick (Overview shows on bare URL). Each AC inside the Manifest tab is a `<section id="ac-N">` for cross-page anchor links.
- [ ] `style.css` — handcrafted, under 5 KB. CSS custom properties for color tokens. The `:target` show/hide rules are the only visual-state machine; no other CSS-driven state.
- [ ] All template `range` over maps goes through the sorted-keys helper from step 3 (determinism rule).
- [ ] Tests: golden HTML for each template against fixture data; visiting a tab URL (`<milestone>.html#tab-build`) produces the same HTML byte-for-byte as the bare URL plus a fragment (rendered HTML doesn't depend on the fragment — the CSS does the work at view time).

### Step 6 — Cross-cutting render details

- [ ] Status pills with consistent color tokens, driven by `aiwf check` findings (§8).
- [ ] Date labels with explicit role (§8).
- [ ] Forced transitions visually distinct in Provenance tab; reason rendered inline.
- [ ] Tests-tab policy badge (`strict` / `advisory`) per §3.3.
- [ ] `:target` tab show/hide CSS verified against the determinism test (no JS, no mutation, deterministic across re-renders).
- [ ] Tests: visual-regression style golden HTML for each cross-cutting concern.

### Step 7 — Documentation

- [ ] README section: how to render, where output lands, deployment patterns 1–4, the `commit_output` flag.
- [ ] Example GitHub Action snippet for Pages deployment (in README, not bundled).
- [ ] Update `aiwf doctor` to mention the render output dir if present and surface a hint when `commit_output: true` but `out_dir` is still ignored (a misconfiguration recoverable by `aiwf update`).

---

## 10. What is NOT in scope

| Feature | Why not |
|---|---|
| `aiwf serve` (HTTP server) | Static files don't need a server. Adds runtime, ports, lifecycle, auth — none of which earn their cost. |
| Interactivity / forms / write paths | The moment you add edits, you've shipped a product with sessions, auth, state. The kernel is the source of truth; edits go through verbs. |
| Client-side JS framework | No React, no Vue, no Svelte. Pure HTML + CSS. |
| Build pipeline (npm, esbuild, etc.) | Single Go binary writes a single static directory. Zero JS toolchain. |
| Mermaid.js / mmdc dependency | Indented Unicode-arrow list is enough. `aiwf status --format=md` already provides the mermaid view. |
| CI test execution / live test runs | Tests tab renders test counts *committed during the TDD cycle*, not test runs in CI. aiwf does not run tests. |
| Live data / WebSockets / auto-refresh | Static. Re-render to update. |
| Multi-tenant / multi-repo dashboards | One repo per render. |
| Authentication / roles | Read-only static files. Filesystem permissions are the access control. |
| Per-page configuration / theming UI | One global stylesheet. Customize via fork or PR if a real need shows up. |
| Diagrams, image embedding, charts | Numeric tables and progress bars only. |
| GitHub Issues / Linear sync | aiwf doesn't sync with external trackers (per existing scope decisions). |
| Approval workflow as state | We don't model "requested-reviewers" or "blocking reviewer." Provenance is descriptive (who did what), not prescriptive. |

If real friction shows up later, revisit. YAGNI.

---

## 11. Status

| Step | State | Owner |
|---|---|---|
| 1 — JSON completeness on `aiwf show` (body parser; trailer parser ext.; depends on I2 reverse-ref index) | shipped (`7fd6524`) | core |
| 2 — `aiwf-tests:` trailer (kernel write path + opt-in warning) | shipped (`d7fd072` + `77ccfb1`) | core (+ rituals repo for `wf-tdd-cycle` rewire — pending) |
| 3 — render package skeleton (`internal/htmlrender/`) | shipped (`aeda5b5`) | core |
| 4 — `aiwf render --format=html` verb + gitignore reconciliation | shipped (`6730c1a` + `056139d`) | core |
| 5 — templates and CSS | shipped (`e3977ad`) | core |
| 6 — cross-cutting render details | shipped within step 5 (status pills, force/audit chips, policy badge, `:target` show/hide, determinism) | core |
| 7 — documentation | shipped (this commit; README HTML render section + `aiwf doctor` render report) | core |

**Post-step-7 polish landed for the v0.2.0 cut:**

- Linear-leaning palette with `color-mix()` pills, accent stripe, kicker, tabular nums, `prefers-color-scheme` dark mode (`cce0c21`, `119879c`).
- Stylesheet cache-busting via content-hash query string for `file://` reloads (`302de69`).
- Two-column layout with a left navigation panel (`<details>`/`<summary>` per epic, current-page ancestors pre-expanded, `aria-current="page"` on the active link, mobile stacks below 768px) (`9b88108`).
- Three-bar SVG brand mark + `aiwf` wordmark in the sidebar; standalone export at `docs/logo.svg` (`fefb354`).
- `status.html` integrated as a rendered page reusing `buildStatus` + `readRecentActivity` (`44ea40b`).
- v0.2.0 release-prep cleanup — sidebar reordered (`Project status` precedes `Overview`), legacy `actor:` field stripped on `aiwf update`, `aiwf-render` skill, `aiwf render --help` (`406ac48`).

**Out of scope for this iteration (deferred to real-use friction):**

- The `--scope <id>` and `--no-history` flags on `aiwf render --format=html` are accepted but not yet implemented — both are reserved seams that surface in the help text. Step 4's `runRenderSite` walks the full tree on every invocation; partial-tree rendering is a follow-on if iteration speed becomes painful.
- Per-AC tests aggregation helpers (`LatestTestsForAC`, `RollupTestsForMilestone`, `RollupTestsForEpic`) listed in step 2 — the milestone Tests tab consumes the same first-with-trailer scan inline (`firstTestsTrailer` in `render_resolver.go`); the named helpers can be extracted when a second consumer needs them.
- Rituals plugin rewire of `wf-tdd-cycle` to `aiwf promote --tests …` — landing in `ai-workflow-rituals` (separate repo). Kernel work is complete and the verb works without the plugin update.
