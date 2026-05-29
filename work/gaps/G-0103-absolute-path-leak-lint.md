---
id: G-0103
title: absolute-path leak lint
status: addressed
discovered_in: M-0089
addressed_by_commit:
    - 3aed5b36
---

## What's missing

No preventive check flags absolute filesystem paths in committed text. Patterns that should trip the rule:

- `/Users/<name>/...` (macOS home)
- `/home/<name>/...` (Linux home)
- `/tmp/<...>`, `/private/<...>`, `/var/<...>`, `/opt/<...>`
- `C:\Users\<name>\...`, `C:\\Users\\<name>\\...` (Windows variants)
- `~/<...>`, `$HOME/<...>` (shell home expansion)

## Why it matters

Absolute filesystem paths in committed text leak contributor-local state: usernames, project layouts, machine-specific install locations. None of it informs a downstream reader; all of it inflates the surface where personal identifiers and unrelated-project names creep into repos that are otherwise public-facing.

Discovered in this repo's M-0089 session: 4 active-prose leaks in `CLAUDE.md` and the E-0027 epic spec, ~10 in archive subdirs (forget-by-default per ADR-0004, skip), and ~33 in `docs/explorations/surveys/liminara/*.md` referencing another local project's filesystem layout. The active leaks landed quietly because no chokepoint flagged them at commit time.

## Resolution shape (2026-05-29)

After design discussion the resolution diverged from this gap's original proposal in two important ways:

1. **External tool, not a custom shell script.** The original proposal was to ship a single shell script (awk/grep) from the rituals plugin. Investigation showed this would invent a new architectural pattern (non-`.md` content in skills, with an unsolved distribution story for CI workflows that can't reach the plugin cache). The cleaner shape uses [gitleaks](https://github.com/gitleaks/gitleaks) — a mature CLI tool designed for exactly this class (contributor-state and secret detection), with first-class integrations for pre-commit hooks, GitHub Actions, and ad-hoc invocation. The "single source of truth" objection still holds: one tool config, all callers.

2. **Rules live in the consumer repo, not the plugin.** Each project has different contributor identities to exclude, different legitimate paths to allowlist, different placeholder conventions for test fixtures. The plugin's role is to surface the practice (when to run, why, what the rule class looks like); the consumer repo owns its `.gitleaks.toml`.

### What landed

**aiwf repo:**

- `.gitleaks.toml` with the path-leak rule set + project-specific allowlist (archive subdirs per ADR-0004, `testdata/` fixtures, the well-known devcontainer home, a codified test-placeholder username scoped to `_test.go` files).
- Pre-commit hook (`scripts/git-hooks/pre-commit`) extended to run `gitleaks git --staged --config=.gitleaks.toml --no-banner` (scans only the staged content, so unstaged WIP doesn't block clean commits). Tolerant: missing `gitleaks` on PATH warns and continues, matching the existing `go`-missing pattern.
- Active-prose sweep: `CLAUDE.md` rituals-repo reference rewritten to its GitHub URL; `internal/cli/doctor/env_pathhint_internal_test.go` contributor identifiers replaced with the codified test-placeholder username; survey docs under `docs/explorations/surveys/liminara/` sanitized via path-only rewrite — the `/Users/<contributor>/Projects/` prefix dropped, leaving bare `liminara/...` repo-relative citations (preserves citation, removes operator marker).
- CLAUDE.md "What's enforced and where" table gains a new chokepoint row.

**Rituals plugin (`https://github.com/23min/ai-workflow-rituals`):**

- `wf-doc-lint/SKILL.md` gains a section 5 describing the path-leak class, recommending `gitleaks` (and family) as the canonical tool, naming the consumer-side ownership pattern, and showing an example rule shape inline.
- No new files in the plugin — matches the established pure-`SKILL.md` convention.

### Out of scope for this resolution

- **CI workflow.** The pre-commit hook is operator-local; CI is the authoritative chokepoint per the kernel's "framework correctness must not depend on operator behavior" principle. Adding `.github/workflows/gitleaks.yml` is a natural follow-up but not done in this patch — pre-commit gives the immediate value, and CI is a one-file addition for a later session.
- **Other content-shape rules.** The wf-doc-lint SKILL.md's first four checks remain LLM-judged; only the path-leak rule is delegated to a deterministic external tool. Future content-shape rules can follow the same "delegate to a real tool, configure in consumer repo" pattern.

## Kindred concerns

- **G-0091** — "No preventive check for body-prose path-form refs to entity files." Already addressed (2026-05-11) via a different mechanism. This gap's original cross-reference claimed "same disposition" but the resolutions diverged: G-0091 → kernel-side check; G-0103 → external tool in consumer repo.

## Worked example (this repo, at gap-filing time)

- **Active prose (4 paths, all `/Users/<name>/Projects/ai-workflow-rituals/`)**: `CLAUDE.md:135` and three under `work/epics/E-0027-.../`. E-0027 has since been wrapped + archived; only the CLAUDE.md leak remained active by 2026-05-29.
- **Archive subdirs (~10 paths)**: per ADR-0004 forget-by-default discipline, the rule skips `archive/`. Historical leaks stay as artifacts.
- **Survey docs (~33 paths in `docs/explorations/surveys/liminara/*.md`)**: cited a local layout for a separate (private) repo. Sanitized via path-only rewrite that dropped the `/Users/<contributor>/Projects/` prefix; preserves all citation information, removes the operator identifier.

At resolution time the actual leak count via gitleaks was 49 (38 survey + 7 doctor test + 4 pluginstate test + 1 CLAUDE.md). After sanitization + allowlist tuning the count went to 0.
