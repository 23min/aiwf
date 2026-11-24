---
id: G-0785
title: Docs say resume picks the latest-paused scope; the verb picks the latest-opened
status: addressed
discovered_in: M-0364
addressed_by_commit:
    - 66fbbc69fdba9a207aa151c3104269aaac54a618
---
## What's missing

The docs and the verb's own help say `--resume` picks the most recently paused scope; the verb
picks the most recently opened paused scope. `authorizeTransition`
(`internal/verb/authorize.go:461`) selects with `mostRecentScopeInState` (`authorize.go:663`),
"the most-recently-opened scope whose current state matches". "Most-recently-paused" is written
in:

- `docs/design/provenance-model.md` §"Verb behavior": "resumes the *most-recently-paused scope*
  for `<id>`";
- `docs/design/legal-workflows-first-principles.md` R-FP-0136;
- `docs/design/legal-workflows-audit.md` R-AUDIT-0219 and R-RULE-072;
- `internal/skills/embedded/aiwf-authorize/SKILL.md`, its command list and §Lifecycle;
- the help text, `internal/cli/root.go:340` and `internal/cli/authorize/authorize.go:65` and
  `:100`, and the doc comments at `internal/verb/authorize.go:166` and `:185`.

`legal-workflows-audit.md` R-AUDIT-0123 says both pause and resume act on "the
most-recently-opened active scope", which names the right order and the wrong state for resume.

Measured 2026-11-21 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a tree whose code is `main` at 0c00b37c4 (`aiwf version`: `v0.42.1-0.20261121183124-0c00b37c47b3+dirty`), in a disposable repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs, and `milestone/M-0001-first-milestone` checked out. Ids in the block name entities in that repository. Two scopes were opened on `M-0001` with two `aiwf authorize M-0001 --to ai/claude` runs.

```
opened 856a2c3 then b33fca3
$ aiwf authorize M-0001 --pause p1        # pauses b33fca3
$ aiwf authorize M-0001 --pause p2        # pauses 856a2c3, now the most recently paused
$ aiwf authorize M-0001 --resume r1
$ aiwf show M-0001
    856a2c3  human/t → ai/claude  state: paused   opened 2026-11-21  events: 2
    b33fca3  human/t → ai/claude  state: active   opened 2026-11-21  events: 3
```

## Why it matters

With more than one paused scope on an entity, an operator who follows the help text resumes a
scope other than the one they expect, and the records that state the rule disagree with the verb
that applies it.
