---
name: aiwfx-release
description: Tags a release after an aiwf epic is closed and merged. Determines the semver bump, updates CHANGELOG.md, creates an annotated git tag, runs health checks. The aiwf epic is already `done` at this point — this skill captures the release act in CHANGELOG and git, not in aiwf state. Never tags or pushes without explicit human approval.
---

# aiwfx-release

Tags and publishes a release. aiwf has no `released` status — `done` is the terminal state for an epic. This skill records the release act in the artifacts that downstream consumers see (git tag, CHANGELOG), not in aiwf state.

## When to use

An epic has been wrapped (`aiwfx-wrap-epic` ran, status is `done`, integration branch merged to mainline). The user says: *"release v1.2"*, *"tag a release"*, *"publish"*.

If the epic isn't wrapped yet, run `aiwfx-wrap-epic` first.

## Workflow

### 1. Pre-release checks

- On `main` (or the project's mainline name).
- Working tree clean.
- All tests pass.
- Build is green.
- The epic that justifies this release has `status: done`.

If anything is red, stop. Releases ride on green commits.

### 2. Determine the version

- Check the current version (latest git tag): `git describe --tags --abbrev=0`.
- Apply semantic versioning per the project's conventions:
  - `MAJOR` — breaking changes.
  - `MINOR` — new features, backward compatible.
  - `PATCH` — bug fixes only.
- Walk the commits since the last tag and classify them. If any commit suggests a breaking change (banner in the message, breaking-change footer, or just on inspection), `MAJOR` is the right bump.
- **Confirm the bump with the user.** If they want a different version, use that.

### 3. Update CHANGELOG.md

Add a new release section. Group entries by Added / Changed / Fixed / Removed. Reference the epic and major milestones; keep entries user-observable, not diff-shaped.

```markdown
## [vX.Y.Z] — YYYY-MM-DD

### Added
- <user-observable feature> (E-NN, M-NNN)

### Changed
- <observable change>

### Fixed
- <bug fix>

### Removed
- <retired feature>
```

If the project keeps an `[Unreleased]` section at the top of CHANGELOG, move its contents into the new release section.

Stage the CHANGELOG. Show the diff.

### 4. 🛑 Commit gate (CHANGELOG)

Show the user the CHANGELOG diff. Propose: `docs(changelog): vX.Y.Z`.

**Stop and wait for "commit" approval.**

```bash
git commit -m "docs(changelog): vX.Y.Z"
```

### 5. 🛑 Tag gate

Confirm with the user: *"Tag as vX.Y.Z and push?"*

```bash
git tag -a vX.Y.Z -m "Release vX.Y.Z: <one-line summary>"
git push origin main
git push origin vX.Y.Z
```

### 6. Post-release verification

- If CI/CD auto-publishes on tag push (npm, PyPI, container registry, GitHub Release), watch the pipeline. On success, confirm the artifact is consumable.
- Run any project-specific health check (smoke test, canary, rollback drill).
- If a deployment failed: assess whether to rollback the tag (rare — usually fix-forward is safer). Don't reuse the version number.

### 7. Optional: link the release to the epic

If the project records release ↔ epic linkage somewhere (a release notes doc, an external tracker), update it now.

The aiwf epic stays `done`. There's no separate "released" status — the git tag and CHANGELOG entry are the durable record of the release.

### 8. Capture any release-time decision

If a notable release-time decision was made (rolled back, hotfixed, deferred a feature out of the cut), capture it via `aiwfx-record-decision`.

## Constraints

- 🛑 **Never tag or push without explicit human approval** (steps 4, 5).
- Releases run on green commits only. No "release this with the failing test, we'll fix in a patch."
- Versions are immutable. If `vX.Y.Z` has a problem, the next release is `vX.Y.(Z+1)` — don't move the tag.
- Don't skip CHANGELOG. Future-you and downstream consumers depend on it.

## Anti-patterns

- *Tagging without checking the diff since the last tag.* The bump might be wrong.
- *Releasing from a feature branch.* Tags are on mainline.
- *Auto-publishing on every tag without a confirmation step.* The tag-push is the gate; if CI/CD watches it, that's fine, but the tag itself is a deliberate human act.
- *Updating the aiwf epic to a "released" status.* aiwf doesn't have one. `done` is terminal; the release record lives in git history.

## Out of scope

Wrapping the epic itself (that's `aiwfx-wrap-epic`). Authoring release notes for marketing — CHANGELOG is the truthful record; marketing copy is downstream.
