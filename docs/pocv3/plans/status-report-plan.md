## Status report plan

**Status:** proposal · **Audience:** PoC continuation, builds on the existing `aiwf status` verb (`tools/cmd/aiwf/status_cmd.go`).

A nicely rendered, host-agnostic status view of the project so users without VSCode and without GitHub Issues still get a one-glance picture: what's in flight, what's ahead, what's open, what's broken. Extends `aiwf status` with a markdown output format that embeds mermaid diagrams. No server, no HTML, no GitHub Pages — markdown renders in every target the framework already cares about (GitHub web, VSCode, Obsidian, `glow`, `mdcat`).

This is a renderer change, not new state. The same `tree.Load` + `check.Run` pass that `aiwf status` already does feeds a richer report struct and a third renderer.

---

### 1. Why extend `aiwf status` rather than add a new verb

Same data, different output. A new verb (`aiwf report`, `aiwf snapshot`) would carry the same `tree.Load` + `check.Run` body and would have to be kept in lockstep with `status` for the rest of time. KISS says: third renderer next to text and JSON. The `--format` flag already exists; markdown is one more value.

### 2. New report sections

Two additions to `statusReport`:

- **`PlannedEpics []statusEpic`** — epics with `status == "proposed"`, sorted by id, each with its planned milestones (`status` ∈ {`draft`, `proposed`} per the milestone state set). Renders as the "Roadmap" section. Tiny but answers "what's on deck?" without leaving the terminal.
- **`Warnings []statusFinding`** — the actual warning findings from `check.Run`, not just the count. Code, message, entity id, path. Errors stay summarized as a count + the existing `run aiwf check for details` hint, because errors mean validation failed and the user should run `aiwf check` directly. Warnings are advisory and worth surfacing inline.

The status `cancelled` epics and `done` epics are not surfaced — they're history, not roadmap. `aiwf history` covers that axis.

### 3. The markdown renderer

`renderStatusMarkdown(w, r)` writes a single self-contained markdown document with these sections, in order:

1. **Header** — `# aiwf status — <date>` and a one-line health summary.
2. **In flight** — H2, table per epic (id, title, status, milestone count). Followed by one mermaid `flowchart LR` per active epic with each milestone as a node, statuses as colors via `classDef`.
3. **Roadmap** — H2, lists `PlannedEpics`. Same shape as in flight, just for proposed epics. If empty, an italic `_(nothing planned)_`.
4. **Open decisions** — H2, table.
5. **Open gaps** — H2, table with discovered-in column.
6. **Warnings** — H2, table (code, entity, message). If zero, `_(none)_`. Errors stay compressed to a count line above the table.
7. **Recent activity** — H2, table (date, actor, verb, detail).

Mermaid is used only for the in-flight + roadmap flowcharts. State-machine diagrams per kind are tempting but redundant with `aiwf schema`. Gantt is tempting but the model has no dates. YAGNI both.

The output is plain markdown — no HTML, no JS. A consumer who wants HTML pipes through `pandoc` or any markdown→HTML tool. That's not aiwf's job.

### 4. CLI surface

```
aiwf status --format=md > STATUS.md
```

No new flags. `--pretty` is documented as JSON-only already, so it's a no-op for `md` (warn on stderr if combined? No — match the existing tolerant behavior of `--format=text --pretty` which silently ignores).

### 5. What's deliberately out of scope

- **HTML output.** Add when (if) someone reports markdown's mermaid coverage isn't enough. The generator and template are easy to add as a sibling renderer.
- **`aiwf serve` / live updates.** Not until someone demands it.
- **GitHub Pages publishing.** Reintroduces the GitHub dependency the user is trying to avoid.
- **Per-kind state-machine diagrams.** Already covered by `aiwf schema`; not snapshot-worthy.
- **Gantt / timeline diagrams.** No dates in the model.
- **Filtering flags** (`--epic`, `--since`). The existing report is already small; if it ever gets too long, paginate later.

### 6. Tests

- Table-driven `renderStatusMarkdown` test with a representative tree fixture and a golden file under `tools/cmd/aiwf/testdata/status_md/golden.md`. Mermaid blocks pinned by the golden so accidental shape drift is caught.
- Empty-tree case: every section renders an empty-state line; no orphan headers without bodies.
- `PlannedEpics` populated from a fixture with mixed `proposed` / `active` / `done` epics, asserts only the proposed ones land in the roadmap section.
- `Warnings` populated from a fixture that intentionally trips a known warning (e.g. a slug-dropped-chars case from G8 or an open gap warning), asserts the row renders with code + message + entity id.

### 7. Reversal

`aiwf status --format=md` is read-only and produces no commit. The "what reverses this?" question doesn't apply (no mutation). The output file's lifecycle is the user's: regenerate at will; if committed, `git rm` if no longer wanted.

### 8. Sequencing

1. Extend `statusReport` with `PlannedEpics` + `Warnings []statusFinding`. Update `buildStatus`. Update text renderer to include them (Roadmap section + Warnings list). Update existing golden text test.
2. Add `--format=md` plus `renderStatusMarkdown`. Add golden markdown test fixture.
3. Update `docs/pocv3/architecture.md` verb table if it lists formats.
4. Tick this plan in `docs/pocv3/gaps.md` if any gap turns out to be a precondition (none expected).

Each step is one commit.
