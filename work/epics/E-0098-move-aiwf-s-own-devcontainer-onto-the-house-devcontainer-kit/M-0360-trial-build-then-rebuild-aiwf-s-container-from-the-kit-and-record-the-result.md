---
id: M-0360
title: Trial-build, then rebuild aiwf's container from the kit and record the result
status: draft
parent: E-0098
depends_on:
    - M-0359
tdd: none
acs:
    - id: AC-1
      title: A trial container built alongside passes every check
      status: open
    - id: AC-2
      title: The rebuilt aiwf container passes every check and resumes sessions
      status: open
---

## Goal

Bring the kit-generated container up twice, first as a trial beside the current
container and then in place of it, and record what each showed.

## Closes

- (none)

## Context

M-0359 lands the kit-generated `.devcontainer/`. VS Code builds a container from
the files in the checkout it opens, and a rebuild removes the running container
before creating its replacement; the session doing this work runs inside that
container. The trial proves the new container before the current one is replaced.

## Acceptance criteria

### AC-1 — A trial container built alongside passes every check

**Pass criterion**, met by a record of command, expectation, observation and
environment: in a container built from a separate clone of the milestone's branch,
`/workspaces` holds only the repository and no `CLAUDE.md` outside it; `aiwf
version` reports a build of that clone's commit; `go version` reports the Go named by
`GO_VERSION` in `.github/workflows/go.yml`; golangci-lint, govulncheck and gitleaks report
the versions pinned in `.devcontainer/project/post-create.sh`; `make ci` passes; `aiwf
doctor` reports the git hooks installed; a commit made in the trial clone carries the
same `aiwf-actor` as the commits made before the move, now that the container's git identity
comes from the host's global git config; Claude Code and Codex start logged in, with
their existing sessions listed. **Edge cases**: the trial clone's container name, which
must differ from the running container's, changed only in the trial clone and
never committed. **Code references**: none; the record lives in this milestone's
Validation section.

### AC-2 — The rebuilt aiwf container passes every check and resumes sessions

**Pass criterion**, met by a record: after the current container is rebuilt from
the merged `.devcontainer/`, the AC-1 checks hold, and `claude --resume` lists
sessions from before the rebuild. The record covers what G-0699 left unobserved:
Codex configuration, sessions and login surviving a rebuild. **Edge cases**: a
failed rebuild, which triggers the rollback in Design notes and is itself
recorded. **Code references**: none.

## Constraints

- The running container is not replaced until AC-1 is met.
- Credential contents are never read or recorded; only whether a login holds.

## Design notes

- Before the rebuild: `tmux-up save` in the running container.
- Trial: clone the milestone branch on the host into its own folder, change the
  container name in its `runArgs` locally, open it in VS Code.
- Rollback: on the host, `git revert` the merge that brought the new
  `.devcontainer/` in, rebuild, and resume sessions with `claude --resume`.

## Out of scope

- Changes to the generated files; a defect found here is filed as a gap, or fixed in this
  milestone when it is small.

## Dependencies

- M-0359.

## References

- E-0098, G-0524, G-0699.

---

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
