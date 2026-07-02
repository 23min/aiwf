---
name: wf-doc-lint
description: Mechanical checks over the project's narrative documentation tree (`docs/`). Reports broken code references, removed-feature docs, orphan files, and documentation TODOs. Reports only — never rewrites prose. Use before proposing a change for merge that touches code with documentation impact, or as a periodic doc-hygiene pass.
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

## Related: repo-wide secret / path-leak scanning (standalone tool)

Distinct from the four doc heuristics above: absolute filesystem paths and secrets in committed text leak contributor-local state — usernames, machine layouts, install locations — and none of it informs a downstream reader. This is **not** one of the four doc heuristics, for three reasons:

- **Scope is repo-wide, not docs-scoped.** A leaked path is a defect anywhere in the tree (source, tests, config, git history), not just under `docs/`. The four heuristics above are docs-scoped; this scan is not.
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

1. Identify the docs root. Default: `docs/`. Some projects use `documentation/`, `book/`, or a flat top-level. Read `README.md` or look for the obvious folder.
2. Decide on scope:
   - **Full** — every doc under the docs root.
   - **Scoped** — docs that intersect with a given change-set (a list of changed source files). Faster, useful inside a focused review.
3. Run each of the four checks above.
4. Emit the report (see Output below).
5. **Stop.** Don't fix anything. Don't append to a log file. Don't update an index.

## Output format

```markdown
# Doc lint report — <YYYY-MM-DD>

**Scope:** full | scoped (<N> changed files)
**Docs root:** docs/

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

## Summary
- <N> findings: <breakdown>
- <one-line takeaway>
```

If nothing is found, the report has empty sections and the summary is a single line: *"No findings — `<N>` docs checked."*

## Anti-patterns

- *"Auto-fix the dead links"* — never. Even mechanical-looking fixes (deleting a link, renaming a symbol reference) are prose changes that need human approval.
- *Treating the four doc heuristics as a CI gate.* Drift in the doc heuristics is expected; surface it, don't fence on it — block-on-zero is too strict for those. (The repo-wide secret / path-leak scan is the deliberate exception: it is deterministic and *does* gate at pre-push + CI — see "Related: repo-wide secret / path-leak scanning" above. Don't conflate the two.)
- *Hand-editing the report.* It's a snapshot. Re-run the lint to update.
- *Confusing this with content review.* This skill catches symbols that no longer exist. It does not catch "this paragraph is misleading" or "this example is wrong even though the symbol is real."

## Constraints

- 🛑 Never modifies any file under `docs/` — including for "obvious" mechanical fixes.
- Findings include `path:line` references. A finding without a location is unactionable.
- Report is the entire output. No side-effect writes.
