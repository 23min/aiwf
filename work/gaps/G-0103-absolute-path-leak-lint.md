---
id: G-0103
title: absolute-path leak lint
status: open
discovered_in: M-0089
---

## What's missing

No preventive check flags absolute filesystem paths in committed text. Patterns that should trip the rule:

- `/Users/<name>/...` (macOS home)
- `/home/<name>/...` (Linux home)
- `/tmp/<...>`, `/private/<...>`, `/var/<...>`, `/opt/<...>`
- `C:\Users\<name>\...`, `C:\\Users\\<name>\\...` (Windows variants)
- `~/<...>`, `$HOME/<...>` (shell home expansion)

The rule belongs in the rituals plugin's `wf-doc-lint` ritual (already the home for content-shape rules — broken refs, removed-feature docs, orphan files, TODOs). Mechanically it should be a single shell script (awk/grep) that scans tracked text formats (markdown, source code, shell scripts) under a uniform regex pattern, with file-glob exemptions for `testdata/` and `archive/` subdirectories and intra-file exemption for triple-backtick code blocks. The same script is invoked from:

- The `wf-doc-lint` skill (advisory ritual the human/LLM runs).
- A CI workflow template (`/.github/workflows/wf-doc-lint.yml`) the rituals plugin ships; consumers copy it for hard-gate enforcement.
- Optionally a contributor pre-commit hook.

**Single source of truth for the pattern.** Don't duplicate the regex in `.golangci.yml`'s `forbidigo` config — the shell script's glob already covers `*.go`. If a future rule needs language-aware scanning (string literals vs. comments distinction), that rule lives in a language-aware tool; this rule does not need that distinction because both contexts are leaks.

## Why it matters

Absolute filesystem paths in committed text leak contributor-local state: usernames, project layouts, machine-specific install locations. None of it informs a downstream reader; all of it inflates the surface where personal identifiers and unrelated-project names creep into repos that are otherwise public-facing.

Discovered in this repo's M-0089 session: 4 active-prose leaks in `CLAUDE.md` and the E-0027 epic spec, ~10 in archive subdirs (forget-by-default per ADR-0004, skip), and ~33 in `docs/explorations/surveys/liminara/*.md` referencing another local project's filesystem layout. The active leaks landed quietly because no chokepoint flagged them at commit time.

## Out of scope for aiwf

This is not an aiwf-entity concern. aiwf-check's surface is *structure* of aiwf entities (sections, AC discipline, FSM, ids, archive shape, trailers, provenance) — not arbitrary prose-content hygiene. Adding a generic-content rule to aiwf-check would conflate two concerns and blur the framework's surface. Every future content-shape rule would have the same argument and aiwf-check accretes scope.

The rituals plugin is the right home: opt-in by consumers, already mandated for narrative-prose hygiene, distributable via the marketplace.

## Kindred concerns

- **G-0091** — "No preventive check for body-prose path-form refs to entity files." Same class (content-shape lint over prose), different concern (internal repo paths that break on archive moves vs. external absolute paths that leak local state). Distinct rule, same lives-in-wf-doc-lint disposition.

If a third wf-doc-lint enrichment surfaces, the natural shape is a single epic for *"wf-doc-lint enrichment with content-shape rules"* with one milestone per rule. With two open (G-0091 + G-0103), single-milestone-per-gap is still right.

## Distribution shape

Rituals plugin (`ai-workflow-rituals` upstream) ships:
1. Updated `wf-doc-lint` SKILL.md naming the rule.
2. The shell script (single source of truth for the pattern).
3. The CI workflow template.

Consumers using `wf-rituals@ai-workflow-rituals` install pick up the rule via the plugin's update path. Hard-gate enforcement is consumer-opt-in (copy the workflow template); ritual-style advisory invocation is automatic.

## Worked example (this repo, at gap-filing time)

- **Active prose (4 paths, all `/Users/<name>/Projects/ai-workflow-rituals/`)**: `CLAUDE.md:135`, `work/epics/E-0027-.../epic.md:19`, `work/epics/E-0027-.../M-0090-...md:31, :53`. These are the immediate Group-A leaks for any pre-rule cleanup pass.
- **Archive subdirs (~10 paths)**: per ADR-0004 forget-by-default discipline, the rule should skip `archive/`. The historical leaks stay as artifacts.
- **Survey docs (~33 paths in `docs/explorations/surveys/liminara/*.md`)**: cite a separate public repo via absolute local paths. Sanitization would rewrite them to public URL refs.

Pre-rule cleanup of Group-A is necessary before the rule can land without immediately blocking on its own author's existing leaks.
