---
name: wf-doc-lint
description: Mechanical checks over the project's narrative documentation tree (`docs/` plus hand-authored root files like `README.md`). Reports broken code references, removed-feature docs, orphan files, documentation TODOs, broken markdown links, stale CLI invocations, and structural drift (table of contents, heading hierarchy). Reports only — never rewrites prose. Use before proposing a change for merge that touches code with documentation impact, or as a periodic doc-hygiene pass.
---

# wf-doc-lint

A non-interactive lint over a project's narrative docs. Surfaces drift between docs and code without rewriting anything. The output is a report; what to do about each finding is the human's call.

This skill is intentionally narrow: structural correctness of narrative documentation, mechanical checks a pipeline can run. It does not judge content quality, alignment of prose with code semantics, or doc tier. Those need human or AI prose work, not a lint.

## When to use

- Before proposing a change for merge that touches code referenced from docs.
- After a refactor that renamed or deleted symbols.
- As a periodic doc-hygiene pass on a slow week.
- Inside a heavier wrap ritual that wants a doc check before declaring done.

## What it checks

### 1. Code-reference drift

Walk every doc in `docs/` (or whatever the project's docs root is). Find references to code symbols (function names, class names, file paths, route paths, config keys). For each reference:

- Does the symbol exist in the current source tree?
- If the doc was edited recently and the symbol it references was deleted in the same window, that's likely intentional (a documentation sweep). Otherwise it's drift.

Report broken references with the doc's `path:line` and the missing symbol.

### 2. Removed-feature docs

If a doc describes a feature whose code has been removed, flag it. Heuristic: docs that are the sole remaining reference to a symbol-like token, where the token doesn't appear anywhere in source, are candidates. False positives are tolerable; the reviewer decides whether the doc should be archived or rewritten.

### 3. Orphan documents

Docs with no inbound links from any other doc, README, or code file. They might be alive (a top-level overview that everything links to externally) or dead (a stale draft someone forgot to archive). Flag for human inspection — don't delete.

### 4. Documentation TODOs

`TODO`, `FIXME`, `XXX`, or similar markers anywhere under `docs/`. List them with `path:line`. Most projects accumulate these; the lint just makes them visible so they can be triaged or grandfathered explicitly.

### 5. Markdown link integrity

Nothing in checks 1–4 verifies that a markdown link itself resolves — check 1 verifies referenced *code symbols*, not link targets. Walk every markdown link and bare anchor in a doc's scope and check three link shapes:

```
link text (relative/doc/path.md)        intra-repo link — does the target file exist relative to the linking doc?
link text (other-doc.md#a-heading)      anchor into another doc, or a bare same-file anchor — does a heading slug to it (lowercase, spaces to hyphens, punctuation stripped)?
link text (../path/to/source-file.ext)  source-file link — does the referenced path exist in the source tree (whatever the project's language)?
```

Report each broken link with the linking doc's `path:line` and the unresolved target. Two false-positive traps: a destination that resolves to a directory rather than a file is a valid link, not a broken one; and illustrative link syntax sitting inside inline code or a fenced block is prose about links, not a live link — skip both kinds of span, not just fenced blocks.

### 6. CLI-invocation resolution

For a project that ships a CLI, a backticked invocation of the project's own tool (e.g. `` `aiwf <verb> --flag` ``) must resolve against what the tool actually accepts today — the same principle a repo may already enforce for skill bodies, generalized to `docs/`. Verify against the tool's real command surface, not a hand-written summary: run the tool's own help (recursing into subcommand help where the CLI framework nests one) rather than trusting a top-level banner, which can omit real verbs — the check is the same regardless of what language or CLI framework the tool is built with. Flag a doc invocation naming a verb or flag the current tool no longer recognizes, with the doc's `path:line`. Heuristic, like check 2: a doc explicitly discussing a deferred, proposed, or hypothetical verb (hedged language — "would", "deferred", "not yet filed" — common in a research or design-exploration doc) is not drift. False positives are tolerable; the reviewer decides whether an unresolved invocation is stale or intentionally aspirational.

### 7. Structural checks

- **Table-of-contents drift** — a hand-maintained `## Contents` (or similar) list whose entries link to `#heading` anchors *within the same doc* that no longer match the headings below it: a renamed, removed, or reordered heading the list never followed. A `## Contents` that instead indexes sibling files (a directory README linking to other files in the same folder) is check 5's job, not this one — it carries no internal headings to drift against.
- **Heading-hierarchy sanity** — a skipped heading level (`##` straight to `####`), two headings in the same doc that collide on the same anchor slug, or a heading with no content before the next heading of equal-or-shallower level.

## Related: repo-wide secret / path-leak scanning (standalone tool)

Distinct from the seven doc heuristics above: absolute filesystem paths and secrets in committed text leak contributor-local state — usernames, machine layouts, install locations — and none of it informs a downstream reader. This is **not** one of the seven doc heuristics, for three reasons:

- **Scope is repo-wide, not docs-scoped.** A leaked path is a defect anywhere in the tree (source, tests, config, git history), not just under `docs/`. The seven heuristics above are docs-scoped; this scan is not.
- **It is deterministic and mechanical** (regex → match), so it deserves a **real chokepoint**, not an LLM-judged advisory heuristic.
- **Its patterns and allowlist are repo-specific** — each project has its own contributor identities to exclude, its own legitimate paths to allow, its own test-fixture placeholder conventions.

So this belongs in a **standalone tool the consumer's repo owns and configures**, invoked from its own chokepoints — not folded into this advisory doc-lint.

Patterns worth catching: `/Users/<name>/…` (macOS home), `/home/<name>/…` (Linux home; allowlist well-known devcontainer users like `/home/vscode/`), `C:\Users\<name>\…` (Windows), `~/…` / `$HOME/…` in prose (idiomatic in shell scripts — allowlist by file path), and machine-state paths (`/tmp/`, `/private/`, `/var/`, `/opt/`).

**Recommended tool family:** [gitleaks](https://github.com/gitleaks/gitleaks) (also viable: `detect-secrets`, `ggshield`). aiwf ships no rules; the consumer's repo owns its `.gitleaks.toml`, tuned to its own contributors and allowlist.

**Where to wire it — the push is the trust boundary.** A secret is not exposed until it is **pushed**, so the scan belongs at **pre-push** (the real boundary) plus an operator-independent **CI** job (the authoritative chokepoint a skipped local hook can't bypass). A *pre-commit* hook only taxes every commit's latency without being the boundary — recommend against it.

Use the current gitleaks subcommands (v8.x deprecated the old `detect`):

```bash
# History scan — every committed blob; for the CI job and the pre-push hook.
gitleaks git --config=.gitleaks.toml --no-banner

# Filesystem scan — the working tree only; for a fast ad-hoc check.
gitleaks dir --config=.gitleaks.toml --no-banner
```

**Example rule shape (gitleaks TOML):**

```toml
[[rules]]
id = "path-leak-darwin-home"
description = "Absolute path leak: macOS home directory"
regex = '''/Users/[A-Za-z][A-Za-z0-9_.-]*/'''
tags = ["path-leak", "contributor-state"]

[[allowlists]]
description = "archive/ subdirs are forget-by-default"
paths = ['''/archive/''']
```

When this skill runs as an advisory ritual, point the operator at their `.gitleaks.toml` and the pre-push + CI wiring above; don't re-implement the rule in prose — the operator's existing tool is the source of truth.

## What it deliberately does NOT do

- **Does not rewrite prose.** Every finding is for human attention.
- **Does not delete files.** Even orphans get reported, not removed.
- **Does not judge content quality.** "This sentence is unclear" is not a doc-lint finding.
- **Does not enforce a freshness threshold.** "Last edited 6 months ago" is not drift on its own.
- **Does not index docs or maintain a catalog.** That's a separate machinery you can add later if needed; this skill is a one-shot check.

## Workflow

1. Identify the docs root. Default: `docs/` **plus** the repo's hand-authored top-level narrative files — `README.md`, `CONTRIBUTING.md`. Some projects use `documentation/`, `book/`, or a flat top-level instead of `docs/`; read `README.md` or look for the obvious folder.
   - **Generated or gitignored root files stay out of scope** — a roadmap/status/whiteboard/todo file regenerated by a project tool (e.g. `ROADMAP.md`, `STATUS.md`, `WHITEBOARD.md`, `TODO.md`) is never linted: if it drifts, the fix is to regenerate the source, not to file a lint finding.
   - **Append-only history stays out of scope** — a `CHANGELOG.md` correctly names removed features and stale flags; that is the record it keeps. Checks 5 and 6 (link integrity, CLI-invocation resolution) would false-positive on nearly every old entry, so it is excluded rather than carrying a per-check exemption.
   - **Orphan documents (check 3) does not apply to root narrative files** — `README.md` is the root everything else links to, so "no inbound links" is expected, not drift. Checks 1, 2, 4–7 apply to root files the same as any other doc in scope.
2. Decide on scope:
   - **Full** — every doc under the docs root (as widened above).
   - **Scoped** — docs that intersect with a given change-set (a list of changed source files). Faster, useful inside a focused review.
3. Run each of the seven checks above.
4. Emit the report (see Output below).
5. **Stop.** Don't fix anything. Don't append to a log file. Don't update an index.

## Output format

```markdown
# Doc lint report — <YYYY-MM-DD>

**Scope:** full | scoped (<N> changed files)
**Docs root:** docs/ + README.md, CONTRIBUTING.md

## Broken code references
- `docs/architecture/api.md:42` — references `OldHandler.serve()`; symbol not found in source.
- `docs/guides/cli.md:7` — references `--legacy-flag`; flag removed in commit <SHA>.

## Removed-feature docs
- `docs/features/x-mode.md` — describes "X mode"; no source reference found.

## Orphan files
- `docs/scratch/notes-2024.md` — no inbound links; last modified <date>.

## Documentation TODOs
- `docs/getting-started.md:18` — `TODO: rewrite once auth lands`
- `docs/api.md:104` — `FIXME: example may be stale`

## Broken links
- `docs/guides/setup.md:12` — links to `docs/guides/old-name.md`; target does not exist.
- `README.md:30` — links to `docs/api.md#old-heading`; no heading slugs to `old-heading`.

## Stale CLI invocations
- `docs/guides/cli.md:22` — `` `aiwf frobnicate --legacy` ``; verb not found in `--help`.

## Structural issues
- `docs/architecture/api.md` — `## Contents` lists "Overview" but no such heading exists below it.
- `docs/guides/cli.md` — heading level skips from `##` to `####` at line 40.

## Summary
- <N> findings: <breakdown>
- <one-line takeaway>
```

If nothing is found, the report has empty sections and the summary is a single line: *"No findings — `<N>` docs checked."*

## Anti-patterns

- *"Auto-fix the dead links"* — never. Even mechanical-looking fixes (deleting a link, renaming a symbol reference) are prose changes that need human approval.
- *Treating the seven doc heuristics as a CI gate.* Drift in the doc heuristics is expected; surface it, don't fence on it — block-on-zero is too strict for those. (The repo-wide secret / path-leak scan is the deliberate exception: it is deterministic and *does* gate at pre-push + CI — see "Related: repo-wide secret / path-leak scanning" above. Don't conflate the two.)
- *Hand-editing the report.* It's a snapshot. Re-run the lint to update.
- *Confusing this with content review.* This skill catches symbols that no longer exist. It does not catch "this paragraph is misleading" or "this example is wrong even though the symbol is real."

## Constraints

- 🛑 Never modifies any file in scope (`docs/`, `README.md`, `CONTRIBUTING.md`) — including for "obvious" mechanical fixes.
- Findings include `path:line` references. A finding without a location is unactionable.
- Report is the entire output. No side-effect writes.
