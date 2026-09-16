---
id: G-0683
title: No automated gate catches a deleted CHANGELOG entry before the release tag
status: open
---
## What's missing

`TestPolicy_ChangelogCompleteness`, in
`internal/policies/changelog_completeness_test.go`, is the only check that
compares `CHANGELOG.md`'s `[Unreleased]` against what shipped, and the only
automated run of it is at the release tag. The reasoning for that placement, in
the comment above the `changelog-audit` target in `Makefile`, is about a
*missing* entry: "a milestone's delta is legitimately absent from [Unreleased]
until its epic wraps, so asking earlier would need an in-flight-epic exemption."
The same placement governs an entry that existed and was *deleted*, which that
reasoning does not cover. Committed to trunk, such a deletion passes every
automated check that runs before the tag. Before the tag it is caught only by
running `make changelog-audit` by hand, which `CLAUDE.md` § "Release process"
asks for and the `aiwfx-release` ritual does not name:

```
$ grep -rn 'changelog-audit' internal/skills/; echo "exit=$?"
exit=1
```

Measured 2026-09-16 on a clone of `main` at `0b4af6141` (nearest tag `v0.35.0`),
with G-0677's entry — `CHANGELOG.md` lines 31–42 — deleted and committed:

```
$ sed -i '31,42d' CHANGELOG.md
$ git -c user.name=scratch -c user.email=scratch@example.invalid commit -q --no-verify -am "scratch: delete the G-0677 changelog entry"
$ go build -o ../aiwf ./cmd/aiwf && ../aiwf check; echo "exit=$?"
...
14 findings (0 errors, 14 warnings)
run `aiwf check --verbose` for each warning's location and remediation hint
exit=0
$ go test -run '^TestPolicy_ChangelogCompleteness$' -count=1 -v ./internal/policies/ | grep '^---'
--- SKIP: TestPolicy_ChangelogCompleteness (0.00s)
$ make changelog-audit; echo "exit=$?"
...
    changelog_completeness_test.go:568: [changelog-completeness] CHANGELOG.md: G-0677 changed a shipped surface but nothing under [Unreleased] cites it; add an entry naming G-0677 (f952d05 docs(ritual): route misfiled gap content to its existing homes).
--- FAIL: TestPolicy_ChangelogCompleteness (0.14s)
FAIL
FAIL	github.com/23min/aiwf/internal/policies	0.160s
FAIL
make: *** [Makefile:243: changelog-audit] Error 1
exit=2
```

None of the fourteen `aiwf check` findings concerns the changelog. No rule under
`internal/check` reads the file:

```
$ grep -rln --include='*.go' --exclude='*_test.go' '"CHANGELOG' internal/
internal/policies/changelog_completeness.go
```

Every automated gate before the tag runs the test suite without
`AIWF_CHANGELOG_BASE`, so the audit skips as above:

```
$ grep -n '^ci:' Makefile
291:ci: vet lint test-cov coverage-gate-only selfcheck
$ grep -rln 'AIWF_CHANGELOG_BASE' .github/workflows/; echo "exit=$?"
exit=1
$ grep -rln 'make changelog-audit' .github/workflows/
.github/workflows/changelog-check.yml
$ tail -1 .git/hooks/pre-push
exec "$AIWF" check
```

`.github/workflows/changelog-check.yml` triggers on `push: tags: - "v*"`. The
pre-push hook first runs `pre-push.local`, a link to `scripts/git-hooks/pre-push`
(lint, gitleaks, the comment scan), and no file under `scripts/git-hooks/`
mentions the changelog.

The one other gate before the tag that checks changelog entries, the `changelog`
job in `.github/workflows/pr-conventions.yml`, runs only on a pull request —
maintainers commit to trunk — and is skipped when the pull request carries the
`internal-only` label. It passes whenever the diff adds at least one line to
`CHANGELOG.md` and no release heading other than `[Unreleased]`
(`in_hunk && /^\+/ && !/^\+\+\+/ { added=1 }`), so an edit that adds one entry
and overwrites a neighbour's heading satisfies it.

## Why it matters

Unless someone remembers the manual run, the deletion is first reported after
the release exists. A pushed `v*` tag is the release — `CHANGELOG.md` states that
releases ship as git tags on `main`, which the Go module proxy resolves when a
consumer runs `aiwf upgrade` — so consumers can install a release whose notes
omit a change they receive, and a failing job cannot withdraw it.

## Related

- G-0529 — the gap the audit closed.
- M-0330 — where the audit's release-tag placement was chosen.
- G-0671 — the audit's other open blind spot: a delta in a named kernel surface.
