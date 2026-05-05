# aiwf status — 2026-05-05

_48 entities · 0 errors · 43 warnings · run `aiwf check` for details_

## In flight

_(no active epics)_

## Roadmap

_(nothing planned)_

## Open decisions

_(none)_

## Open gaps

| ID | Title | Discovered in |
|----|-------|---------------|
| G-022 | Provenance model extension surface |  |
| G-023 | Delegated \`--force\` via \`aiwf authorize --allow-force\` |  |
| G-038 | The kernel repo does not dogfood aiwf — feasibility and fit need investigation |  |
| G-048 | \`aiwf init\` doesn't honor \`core.hooksPath\` — installs hooks into \`.git/hooks/\` regardless |  |
| G-049 | gap-resolved-has-resolver fires chronically on legacy-imported gaps |  |

## Warnings

| Code | Entity | Path | Message |
|------|--------|------|---------|
| gap-resolved-has-resolver | G-001 | work/gaps/G-001-contract-paths-can-escape-the-repo-via-or-symlinks.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-002 | work/gaps/G-002-apply-is-not-atomic-on-partial-failure.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-003 | work/gaps/G-003-pre-push-hook-fails-opaquely-when-validators-are-missing.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-004 | work/gaps/G-004-no-concurrent-invocation-guard.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-005 | work/gaps/G-005-reallocate-s-prose-references-are-warnings-not-errors.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-006 | work/gaps/G-006-design-docs-are-stale-relative-to-i1-contracts.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-007 | work/gaps/G-007-skill-namespace-is-a-convention-not-a-guard.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-008 | work/gaps/G-008-slugify-silently-drops-non-ascii.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-009 | work/gaps/G-009-aiwf-doctor-self-check-is-not-run-in-ci.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-010 | work/gaps/G-010-macos-case-insensitive-filesystem-assumption.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-011 | work/gaps/G-011-context-context-not-threaded-through-mutation-verbs.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-012 | work/gaps/G-012-pre-push-hook-hard-codes-binary-path-at-install-time.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-013 | work/gaps/G-013-no-windows-guard.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-014 | work/gaps/G-014-parse-failure-cascades-into-refs-resolve-findings.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-015 | work/gaps/G-015-no-published-per-kind-schema-for-skill-authors.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-017 | work/gaps/G-017-no-published-per-kind-body-template-for-skill-authors.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-018 | work/gaps/G-018-contract-config-validation-is-hook-only-on-contract-bind-and-add-contract-validator.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-019 | work/gaps/G-019-aiwf-init-writes-per-skill-gitignore-entries-new-skills-aren-t-covered.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-020 | work/gaps/G-020-aiwf-add-ac-accepts-prose-titles-renders-one-giant-ac-n-title-heading.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-021 | work/gaps/G-021-kernel-surface-is-partially-undocumented-for-ai-assistants.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-024 | work/gaps/G-024-manual-commits-bypass-aiwf-verb-trailers-no-first-class-repair-path.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-025 | work/gaps/G-025-pre-commit-policy-hook-is-per-clone-install-by-copy-drifts-silently.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-026 | work/gaps/G-026-findings-have-tests-policy-mirrors-g21-s-old-shape-only-sees-named-constant-codes.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-027 | work/gaps/G-027-test-the-seam-policy-missing-verb-level-integration-tests-skipped-the-cmd-helper-integration.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-028 | work/gaps/G-028-version-latest-test-was-implementation-driven-not-contract-driven-stale-latest-cache-went-unnoticed.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-029 | work/gaps/G-029-pseudo-version-regex-was-example-driven-not-spec-driven-initial-test-set-missed-two-of-three-forms-plus-dirty.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-030 | work/gaps/G-030-git-log-grep-false-positives-leak-prose-mention-commits-into-recent-activity-aiwf-history.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-031 | work/gaps/G-031-squash-merge-from-the-github-ui-defeats-the-trailer-survival-contract.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-032 | work/gaps/G-032-merge-commits-silently-bypass-the-untrailered-entity-audit.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-033 | work/gaps/G-033-aiwf-doctor-self-check-doesn-t-exercise-the-audit-only-recovery-path.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-034 | work/gaps/G-034-mutating-verbs-sweep-pre-staged-unrelated-changes-into-their-commit.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-035 | work/gaps/G-035-html-site-only-generates-pages-for-epic-and-milestone-gap-adr-decision-contract-links-404.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-036 | work/gaps/G-036-entity-body-markdown-rendered-as-escaped-raw-text-in-html.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-037 | work/gaps/G-037-cross-branch-id-collisions-split-the-audit-trail-allocator-is-local-tree-only.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-039 | work/gaps/G-039-aiwf-upgrade-mis-parses-go-env-output-when-gobin-is-unset.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-040 | work/gaps/G-040-work-is-mechanically-unprotected-aiwf-check-silently-ignores-stray-files.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-041 | work/gaps/G-041-tree-discipline-ran-only-at-pre-push-llm-loop-signal-lands-too-late.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-042 | work/gaps/G-042-pre-commit-hook-coupled-enforcement-and-convenience-status-md-auto-update-false-removed-the-tree-discipline-gate-too.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-043 | work/gaps/G-043-go-toolchain-and-lint-surface-trail-current-best-practice-llm-generated-go-drifts-toward-stale-idioms.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-044 | work/gaps/G-044-test-surface-is-example-driven-only-no-fuzz-property-or-mutation-coverage-of-high-value-parsers-and-fsms.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-045 | work/gaps/G-045-aiwf-managed-git-hooks-don-t-compose-with-consumer-written-hooks.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-046 | work/gaps/G-046-aiwf-upgrade-fails-opaquely-when-the-install-package-path-changes-between-releases.md | gap is marked addressed but addressed_by is empty |
| gap-resolved-has-resolver | G-047 | work/gaps/G-047-aiwf-version-pin-is-required-set-once-and-never-auto-maintained-chronic-doctor-noise.md | gap is marked addressed but addressed_by is empty |

## Recent activity

| Date | Actor | Verb | Detail |
|------|-------|------|--------|
| 2026-05-05 | human/peter | add | aiwf add gap G-049 'gap-resolved-has-resolver fires chronically on legacy-imported gaps' |
| 2026-05-05 | human/peter | import | feat(aiwf): G38 — bulk-import legacy gaps from gaps.md into the entity tree |

