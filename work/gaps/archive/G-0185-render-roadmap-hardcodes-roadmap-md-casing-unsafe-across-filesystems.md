---
id: G-0185
title: render roadmap hardcodes ROADMAP.md casing, unsafe across filesystems
status: addressed
addressed_by_commit:
    - f7fd1f99
---
## Problem

`aiwf render roadmap --write` writes to a hardcoded `filepath.Join(rootDir, "ROADMAP.md")` (uppercase) at the repo root (`internal/cli/render/render.go`). There is no `--out` flag and no `aiwf.yaml` knob — unlike `render --format=html`, which has both `--out` and `html.out_dir`. Three code paths in the verb match the filename case-sensitively: the existing-content read used for `## Candidates` preservation + idempotency (`os.ReadFile(dest)`), the staged-edit guard (`if p == "ROADMAP.md"`), and `gitops.Add(ctx, rootDir, "ROADMAP.md")`.

The verb is invoked by five shipped ritual surfaces (`aiwfx-plan-epic`, `aiwfx-plan-milestones`, `aiwfx-wrap-epic`, `aiwfx-wrap-milestone` ×2) plus the `planner` agent, which reports "Updated ROADMAP.md" as a done-state.

## Impact (observed downstream)

A consumer repo tracks a lowercase `roadmap.md` (a legitimate convention). The same command behaves differently by filesystem:

- **macOS (case-insensitive APFS):** `ROADMAP.md` resolves to the existing `roadmap.md`; the verb updates the right file. Works by accident.
- **case-sensitive Linux/CI:** `os.ReadFile("ROADMAP.md")` returns `ErrNotExist`; the verb **creates and commits a second, divergent `ROADMAP.md`**, leaving the human-facing `roadmap.md` stale. The staged-edit guard is bypassed (case-sensitive compare), and the hand-curated `## Candidates` block from `roadmap.md` is lost.

Net: same repo + same command + different filesystem → opposite file targeted; a ritual/agent reports success having updated a file the human never reads. This is the G-0010 case-path footgun, but for a generated root artifact that the `casePaths` check does not cover.

The parallel `STATUS.md` surface shares the root cause (hardcoded uppercase literal; case-sensitive `.gitignore` match `statusMdGitignoreLine = "STATUS.md"` in `internal/initrepo/initrepo.go`) but is benign in practice (STATUS.md is gitignored and machine-read). Out of scope here; file a follow-up if a consumer hits the `.gitignore` lowercase miss.

## Resolution (this gap)

Make the roadmap renderer **case-reconciling**, plus a chokepoint finding. No new `aiwf.yaml` knob (honors YAGNI and stays clear of E-0039's config work):

1. **Verb (`internal/cli/render/render.go`):** before choosing the write destination, scan the repo root for an existing case-insensitive match of `roadmap.md`. If exactly one exists, write to *that* path (preserve the consumer's casing); otherwise write canonical `ROADMAP.md`. Apply the same case-insensitive resolution to the staged-edit guard and the `gitops.Add` path so all three agree on the resolved filename.
2. **Check (`internal/check/`):** add an advisory finding (`roadmap-case-collision`) that fires when more than one case-variant of the roadmap file exists at the repo root (the genuinely-broken state reconciliation cannot silently resolve; only physically possible on a case-sensitive filesystem). Pre-push chokepoint, consistent with the existing `casePaths` rule.
3. **Docs / discoverability:** state the canonical name (`ROADMAP.md`, uppercase) in the `aiwf-render` skill; add the new finding code to the `aiwf-check` skill's finding table; add a README link to `ROADMAP.md` so consumers default to the canonical casing.

## Done when

- A case-sensitive-filesystem test proves `render roadmap --write` against a repo containing only `roadmap.md` writes/commits *that* file (no new `ROADMAP.md`), preserves its `## Candidates`, and the idempotency no-op path keys off the resolved name.
- A test proves the staged-edit guard trips for a staged `roadmap.md` (any case).
- A `check` fixture with both `ROADMAP.md` and `roadmap.md` present at root yields the new finding; a clean tree yields none.
- `aiwf-render` SKILL.md states the canonical name; `aiwf-check` SKILL.md lists the new finding code; README links `ROADMAP.md`.
- `go test -race ./...`, `golangci-lint run`, and `go build` are all green.

## Decision: accommodating, not opinionated

Chosen behavior is **reconcile to whatever casing exists** (accommodating) rather than **force `ROADMAP.md` and error on a lowercase variant** (opinionated). Reconcile is the strict superset: it never breaks an existing consumer convention and removes the cross-filesystem divergence with zero migration. The advisory finding nudges toward the canonical name without blocking. Reversible to opinionated later with a small delta.

## Refs

- Surfaced via a downstream consumer (lowercase `roadmap.md`; macOS→Linux transition).
- Related: G-0010 (casePaths for entities), G-0169 (render roadmap lacks `--format=json`), and the `render --format=html` `--out` / `html.out_dir` precedent.
