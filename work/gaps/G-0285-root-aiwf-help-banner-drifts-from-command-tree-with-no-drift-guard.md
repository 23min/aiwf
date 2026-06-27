---
id: G-0285
title: Root aiwf --help banner drifts from command tree with no drift guard
status: open
prior_ids:
    - G-0282
---
## What's missing

The root `aiwf --help` banner — produced by `printHelp()` in `internal/cli/root.go` — is a hand-maintained string literal that no test diffs against the registered Cobra command tree. It has drifted:

- **Six registered top-level verbs are absent from the `Verbs:` block:** `acknowledge-illegal`, `archive`, `list`, `milestone`, `retitle`, `rewidth`. They are registered via `cmd.AddCommand(...)` (around `internal/cli/root.go:161-190`) but missing from the literal (around `root.go:233-265`). `list` and `archive` are among the most-used verbs and each has its own skill.
- **The `Flags for 'add'` section omits real flags:** `--tdd` (required at creation for milestones, `internal/cli/add/add.go:95`), `--area` (`add.go:98`), and `--body-file` (`add.go:104`).
- **The `Flags for 'promote' and 'cancel'` section omits** `--by`, `--by-commit`, `--superseded-by`. (These are documented in the aiwf-promote skill, so they remain AI-discoverable there — banner-only drift.)
- **The `update` verb one-liner understates current scope:** it now also materializes ritual skills, role agents, the `aiwf-guidance.md` fragment, and git hooks (commitment #5), not just the `aiwf-*` skills.

No mechanical chokepoint binds the banner to the command set. `internal/cli/integration/completion_drift_test.go` and `internal/policies/skill_coverage.go` both derive their verb set from the live Cobra tree and check orthogonal properties (completion wiring; skill coverage) — neither reads `printHelp()`. `root_test.go`'s `TestNewRootCmd_HasExpectedVerbs` checks the tree against a hardcoded want-list, not the banner; `TestExecute_Help` asserts only a single title substring.

## Why it matters

`aiwf <verb> --help` is the first channel the kernel's own "kernel functionality must be AI-discoverable" principle names. When the banner omits a verb, an AI assistant consulting it concludes the capability does not exist — which is exactly what happened: a session believed no post-creation `depends_on` editor existed and proposed building one, when `aiwf milestone depends-on` already shipped. The omission also violates the "auto-completion is the human-facing peer of AI-discoverability — both must traverse the same canonical surface, not two parallel ones" rule: tab-completion lists `milestone` (and the other five); the help banner does not. The two surfaces have measurably diverged.

## Proposed fix

1. **Sync the banner:** add the six verbs, the three `add` flags, the three promote flags, and correct the `update` description.
2. **Add the missing chokepoint:** a policy test (mirroring the `completion_drift_test.go` tree-walk pattern) that enumerates `NewRootCmd().Commands()` and asserts every registered top-level command name appears in `printHelp()` output — and every registered value-taking flag appears in the relevant flags section — with an explicit opt-out list for deliberate omissions. Without this, the banner re-drifts the next time a verb or flag is added.
