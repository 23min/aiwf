---
id: G-094
title: GitHub repo name 'ai-workflow-v2' is post-promotion historical; mismatches kernel identity 'aiwf' across go install path, internal/version proxy queries, and 155 in-repo references
status: open
---
The GitHub repo at `github.com/23min/ai-workflow-v2` carries a historical name. The "v2" suffix dates to the framework's pre-promotion history (PoC v1 → v2 → v3, with v3 becoming the canonical aiwf kernel as documented in CLAUDE.md and `docs/pocv3/archive/poc-plan-pre-migration.md`). Post-promotion, the repo's content *is* aiwf — the kernel binary, its tests, its planning tree, its documentation. The `-v2` suffix is now misleading: it suggests a current-state versioning concept that doesn't exist.

This mismatches kernel identity in three operator-facing surfaces:

- **Go module path.** `go.mod` declares `module github.com/23min/ai-workflow-v2`. The canonical install command is `go install github.com/23min/ai-workflow-v2/cmd/aiwf@latest`. The binary is `aiwf`; the install path mentions `ai-workflow-v2`. New users see two names for the same thing.
- **`internal/version/version.go`** hardcodes the module path for `aiwf upgrade`'s proxy queries. The version verb queries `github.com/23min/ai-workflow-v2`; the binary it produces is `aiwf`.
- **Documentation references**: `CHANGELOG.md`, `README.md`, `CONTRIBUTING.md`, `CLAUDE.md`, and `.golangci.yml` carry literal `github.com/23min/ai-workflow-v2` URLs. 155 files in the repo reference the path.

## What's missing

A coherent rename to `github.com/23min/aiwf`, with the in-repo Go module path updated atomically and a v1.0.0 release cut at the new path to mark the post-promotion canonical home.

## Why it matters

Three failure modes the mismatch invites:

1. **Discoverability cost.** Anyone discovering the project through a blog post, a search, or word-of-mouth searches GitHub for "aiwf" and finds nothing. The repo's actual name is the project's history (v2 of an earlier framework), not its current identity. Documentation-hierarchy gap (per G-092) compounds: an LLM agent reading the tree builds a mental model where the repo's name suggests still-iterating PoC, when the kernel is stable enough to graduate.

2. **Install-command churn deferred is install-command churn doubled.** `go install github.com/23min/ai-workflow-v2/cmd/aiwf@latest` is the canonical install today. If the rename happens later (after blog posts, README links, and operator muscle memory have crystallized around the old command), every reference breaks at once. Doing the rename near the graduation moment — when the user base is small and the disruption window is minimal — costs less than deferring.

3. **Squatter risk on the post-rename old name.** Whatever rename mechanism is used, after the rename `ai-workflow-v2` either redirects (Option A, GitHub's auto-redirect) or stays as an archived repo under owner control (Option B, this gap's recommended path). Option A's redirect breaks if anyone creates a new repo at `23min/ai-workflow-v2` after the rename. Option B preserves the old URL under owner control, archived. For a public Go module being installed via `go install ...`, Option B is the supply-chain-safer move.

## Fix shape

**Two-phase migration via a blank-target repo (Option B):**

### Phase 1 — In-repo coherent rewrite (single wf-patch on current repo)

1. Update `go.mod` module path: `github.com/23min/ai-workflow-v2` → `github.com/23min/aiwf`.
2. Sed-pass all `.go` imports across 155 reference sites (`internal/`, `cmd/aiwf/`, test files, fixtures).
3. Update `internal/version/version.go` proxy-path constant (and its test assertion in `version_test.go`).
4. Update documentation URL references: `CHANGELOG.md`, `README.md`, `CONTRIBUTING.md`, `CLAUDE.md`, `.golangci.yml`.
5. Update `aiwf init`'s install-command nudge text (per `internal/skills/embedded/aiwf-init/` if it exists, or wherever the printRitualsSuggestion equivalent lives for the install path).
6. Verify: `go build ./...`, `go test -race ./...`, `golangci-lint run`, `aiwf check`, `aiwf doctor --self-check`.
7. Single commit; conventional subject `chore(aiwf): rename module path to github.com/23min/aiwf`. Don't push yet.

### Phase 2 — GitHub-side migration

1. Create blank `github.com/23min/aiwf` repo (empty; no README, no `.gitignore`).
2. Local `git remote set-url origin git@github.com:23min/aiwf.git`.
3. `git push --mirror` — tags, branches, full history travel to the new repo.
4. Verify the new repo's tree, tags, and CI configs render correctly.
5. Cut `v1.0.0` tag at the new repo to mark the post-promotion canonical home (CHANGELOG entry per the release process in CLAUDE.md).
6. Archive `github.com/23min/ai-workflow-v2` on GitHub. Optionally edit its README to one line: *"Renamed to [github.com/23min/aiwf](https://github.com/23min/aiwf). This archive preserves pre-rename history."*
7. Reinstall the local binary: `go install github.com/23min/aiwf/cmd/aiwf@v1.0.0`.

## Why Option B over Option A (GitHub rename)

This gap recommends Option B (new blank repo + mirror push) over Option A (GitHub's built-in rename) for four reasons specific to this repo's profile:

1. **No significant GitHub-side metadata to lose.** Solo dev; trunk-based on main per CLAUDE.md (no open PRs); issues tracked as aiwf gap entities rather than GitHub Issues; minimal stars/watchers/forks. The cost of B (lose GitHub-side metadata) is near-zero here.

2. **Squatter-proof under owner control.** Option A vacates the old name; Option B keeps it as an archived repo under the same owner. For a public Go module, the supply-chain-safer move.

3. **Clean v1.0.0 cut at the new path.** Bundling rename + v1.0.0 release at the new repo makes the post-promotion graduation legible: pre-promotion history at the old (archived) repo, post-graduation development at the new. Operator-facing release process gains a clean major-version demarcation aligned with the kernel-name unification.

4. **Old URL stays durably resolvable.** GitHub's auto-redirect under Option A is HTTP-only and brittle (breaks on squat-and-replace). Option B's archived repo remains permanently accessible under owner control with an explicit README pointer; cached Go-proxy installs at the old path keep working until the proxy expires them naturally.

## Out of scope

- **Renaming the companion `23min/ai-workflow-rituals` repo.** Separate concern; the rituals plugin's marketplace path (per CLAUDE.md *Operator setup*) doesn't depend on this repo's name. If the rituals repo also wants a name change for symmetry, file a sibling gap.

- **Vanity import path** (e.g., `aiwf.dev/cli`). Long-term solution that decouples the import path from any GitHub repo name. Requires DNS + a redirect server. Premature for current scale; revisit if the repo ever moves orgs.

- **`/v2` major-version path discipline.** Go's import-path rules require a `/v2` suffix at v2+ (per `go.mod` semver-import compatibility rules). At v1, the path is bare; at an eventual v2.0.0 the module path becomes `github.com/23min/aiwf/v2`. Out of scope for this rename; flagged as a future concern.

- **Coordinating with downstream consumers.** No public consumers exist yet beyond the kernel author's own dogfooding. If consumers appear before the rename ships, treat as a supersession event: announce the new path; cut the v1.0.0 tag at the new repo; let the proxy serve old-path tags from cache while consumers migrate at their pace.

## References

- **CLAUDE.md** "What aiwf commits to" — kernel identity is `aiwf`; the binary is `aiwf`; the documentation calls the framework `aiwf`. Current repo name is the only operator-facing surface that doesn't.
- **CLAUDE.md** *Release process* — versioning, tagging, CHANGELOG. Phase 2 step 5 follows this process at the new repo.
- **G-092** — doc-authority hierarchy across `docs/`. Naming clarity at the repo level compounds doc-authority clarity inside the tree.
- **`go.mod`** — current module path declaration.
- **`internal/version/version.go`** — proxy-query path constant; updated atomically in Phase 1.
- **`docs/pocv3/archive/poc-plan-pre-migration.md`** — pre-promotion history that motivates the v2 historical name and supports archiving the old repo as historical record.
