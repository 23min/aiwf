---
id: G-0736
title: Contribution templates ask for checks, documents and commands that do not exist
status: open
discovered_in: M-0335
---
## What's missing

The pull-request and task templates, and the workflow that checks pull requests, ask a contributor for checks, documents and commands the repository does not hold where they point.

- `.github/pull_request_template.md:5` asks for the "principles checklist (CLAUDE.md §"The principles checklist")", and `:9` for the "Pre-PR audit" in `CLAUDE.md`; `CLAUDE.md` has neither section.
- Its test plan, at the end of the file, runs `go test -race ./tools/...` and `bash tests/test-install.sh`; neither `tools/` nor `tests/test-install.sh` exists.
- `.github/ISSUE_TEMPLATE/task.yml:75` sends the contributor to "the principles checklist in `CLAUDE.md` (or §12 of `docs/architecture.md`)"; `docs/architecture.md` has six numbered sections. Its examples cite `events.jsonl`, an event log aiwf deliberately does not keep. A principles checklist exists only as §12 of the archived `docs/archive/architecture.md`, a frozen snapshot.
- `.github/workflows/pr-conventions.yml:3,8` describes itself as enforcing PR rules from `CLAUDE.md`'s "Issue and PR conventions" and "Pre-PR audit", and leaves the rest to "a human review against the principles checklist".

```
$ ls tools tests/test-install.sh
ls: cannot access 'tools': No such file or directory
ls: cannot access 'tests/test-install.sh': No such file or directory
$ grep -cE 'principles checklist|Pre-PR audit|Issue and PR conventions' CLAUDE.md
0
$ grep -c '^## [0-9]*\.' docs/architecture.md
6
$ grep -n '^## 12\.' docs/archive/architecture.md
688:## 12. The principles checklist
```

Expected: every check, document and command the templates and the workflow header name exists where they point. Observed 2026-10-11 in the aiwf devcontainer, as above.

Where a fix would land: `.github/pull_request_template.md`, `.github/ISSUE_TEMPLATE/task.yml` and the header of `.github/workflows/pr-conventions.yml`.

## Why it matters

An outside contributor following the pull-request template is asked to walk a checklist and run commands that do not exist, so either they stop to ask or they skip the steps, and the template's request carries no weight.

A reviewer comparing a pull request against the template cannot tell what was expected of it, since the expectations it names are not defined anywhere.
