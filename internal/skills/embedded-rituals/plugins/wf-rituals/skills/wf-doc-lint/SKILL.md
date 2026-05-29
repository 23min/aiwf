---
name: wf-doc-lint
description: Mechanical checks over the project's narrative documentation tree (`docs/`). Reports broken code references, removed-feature docs, orphan files, and documentation TODOs. Reports only — never rewrites prose. Use before opening a PR that touches code with documentation impact, or as a periodic doc-hygiene pass.
---

# wf-doc-lint

A non-interactive lint over a project's narrative docs. Surfaces drift between docs and code without rewriting anything. The output is a report; what to do about each finding is the human's call.

This skill is intentionally narrow: structural correctness of narrative documentation, mechanical checks a pipeline can run. It does not judge content quality, alignment of prose with code semantics, or doc tier. Those need human or AI prose work, not a lint.

## When to use

- Before opening a PR that touches code referenced from docs.
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

### 5. Contributor-state and path leaks

Absolute filesystem paths in committed text leak contributor-local state — usernames, project layouts, machine-specific install locations. None of it informs a downstream reader; all of it inflates the surface where personal identifiers and unrelated-project names creep into public-facing repos.

Patterns that should trip the rule:

- `/Users/<name>/...` (macOS home)
- `/home/<name>/...` (Linux home; allowlist well-known devcontainer users like `/home/vscode/`)
- `C:\Users\<name>\...` (Windows variants)
- `~/<...>`, `$HOME/<...>` (shell home expansion in prose; idiomatic in shell scripts — allowlist by file path)
- `/tmp/`, `/private/`, `/var/`, `/opt/` (machine-state paths)

Unlike the other four checks above (which are skill-internal heuristics the LLM applies), this one's right home is a **standalone tool the consumer's repo owns and configures**, because:

- The rule is deterministic and mechanical (regex → match), so it deserves a real chokepoint, not LLM-judged advisory.
- The exact patterns and allowlist policy are repo-specific (each project has its own contributor identities to exclude, its own legitimate paths to allow, its own placeholder conventions for test fixtures).
- The same tool needs to be invokable from multiple chokepoints: pre-commit hook, CI workflow, manual run — that distribution problem is already solved by mature tools.

**Recommended tool family:** [gitleaks](https://github.com/gitleaks/gitleaks) (also viable: `detect-secrets`, `ggshield`). All three are CLI-based, support custom regex rules, and have first-class integrations for pre-commit hooks, GitHub Actions, and ad-hoc invocation. The plugin does not ship rules; the consumer's repo owns its `.gitleaks.toml` (or equivalent), tuned to its own contributors and allowlist.

**What the consumer adds:**

1. A `.gitleaks.toml` at the repo root with the path-leak rules and the project-specific allowlist (archive dirs, testdata fixtures, well-known devcontainer users, codified test placeholders).
2. A pre-commit hook that runs `gitleaks detect --config=.gitleaks.toml --no-banner --no-git` and blocks on non-zero exit.
3. Optionally, a CI workflow that runs the same command. The pre-commit hook gives fast feedback; CI is the authoritative chokepoint.

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

When this skill runs as advisory ritual, point the operator at their `.gitleaks.toml` and recommend `gitleaks detect`. Don't try to re-implement the rule in prose — the operator's existing tool is the source of truth.

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
   - **Scoped** — docs that intersect with a given change-set (a list of changed source files). Faster, useful inside a PR.
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
- *Treating doc-lint findings as a CI gate.* Drift is expected; surface it, don't fence on it. Block-on-zero is too strict for any real codebase.
- *Hand-editing the report.* It's a snapshot. Re-run the lint to update.
- *Confusing this with content review.* This skill catches symbols that no longer exist. It does not catch "this paragraph is misleading" or "this example is wrong even though the symbol is real."

## Constraints

- 🛑 Never modifies any file under `docs/` — including for "obvious" mechanical fixes.
- Findings include `path:line` references. A finding without a location is unactionable.
- Report is the entire output. No side-effect writes.
