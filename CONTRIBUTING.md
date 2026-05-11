# Contributing to aiwf

`aiwf` is an experimental framework for keeping humans and AI assistants in sync about what's planned, decided, and done. Forward design and rationale live at [`docs/pocv3/design/design-decisions.md`](docs/pocv3/design/design-decisions.md); engineering rules live in [`CLAUDE.md`](CLAUDE.md). Read those before sending substantive changes.

---

## Two workflows, one trunk

The project develops trunk-based on `main`. The contribution path depends on whether you have write access:

- **Maintainers** commit directly to `main`. No PR ceremony, no review queue. The pre-commit and pre-push hooks (`aiwf check`) plus CI are the chokepoints. See [`CLAUDE.md` § Working in this repo](CLAUDE.md#working-in-this-repo).
- **Outside contributors** propose changes through GitHub. Start with an Issue or Discussion (templates ship under `.github/ISSUE_TEMPLATE/`), then open a PR (`pull_request_template.md` walks the expected body). The `pr-conventions` workflow checks PR title, issue citation, and CHANGELOG touch.

Both paths land on the same trunk; the PR machinery is the protocol for changes coming in from outside the maintainer set, not a parallel internal process.

---

## Engaging before code

Most useful work starts with a conversation, not a diff.

### Discussions

Use **[Discussions](https://github.com/23min/aiwf/discussions)** for:

- "Should we add X?" — feature ideas, scope debates.
- "How does Y work?" — usage Q&A.
- "What about Z?" — open-ended commentary on design choices.

Discussions have no close condition; they end when the conversation does.

### Issues

Use **[Issues](https://github.com/23min/aiwf/issues)** for **trackable work** with a clear close condition. Templates:

- **Bug** — file when shipped behavior diverges from documented behavior.
- **Design question** — file when something in the design docs is unclear or seems internally inconsistent.
- **Task** — file a unit of planned work.

There is no "feature request" template by design. Feature ideas start as Discussions; once a discussion converges, file an Issue (or, if you're a maintainer, file a gap directly via `aiwf add gap`).

Blank issues are disabled. Pick a template or open a Discussion.

---

## Pull requests (outside contributors)

PRs are welcome from outside contributors. The expectations:

- **Cite the prior conversation** in the description (Issue # or Discussion link). Drive-by PRs without prior context will be asked to start one.
- **Keep changes small.** One logical change per PR.
- **Update [`CHANGELOG.md`](CHANGELOG.md)** under `[Unreleased]` for any user-visible change, or apply the `internal-only` label.
- **Run the validation set** from [`CLAUDE.md` § How to validate changes](CLAUDE.md#how-to-validate-changes): `go test -race ./...`, `golangci-lint run`, `go build`.

`pr-conventions.yml` mechanically checks the first three; the rest is on you.

---

## Commit message style

Conventional Commits, in the form:

```text
<type>(<scope>): <short subject>

<body — what changed and why, not how>
```

Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`. Scopes follow the directory/module touched (e.g., `feat(check)`, `fix(verb)`, `docs(adr)`).

Mutating `aiwf` verbs additionally write structured trailers (`aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`) so `aiwf history <id>` can reconstruct per-entity timelines. See [`CLAUDE.md` § Commit conventions](CLAUDE.md#commit-conventions).

The Apache-2.0 inbound license terms apply to all contributions; by sending a patch you agree your contribution is licensed under [LICENSE](LICENSE). No CLA, no separate paperwork.

---

## What gets accepted readily

- Bug fixes for shipped behavior, with a test that reproduces the bug.
- Documentation clarifications grounded in a question someone actually asked.
- Coverage additions on existing `internal/...` packages.

## What needs more discussion first

- Changes to the six entity kinds, their status sets, or the FSM.
- Changes to the commit-trailer keys, JSON envelope, or check-rule codes.
- New verbs (read [`CLAUDE.md` § Designing a new verb](CLAUDE.md#designing-a-new-verb) first).
- Anything that touches the *what aiwf commits to* list in [`docs/pocv3/design/design-decisions.md`](docs/pocv3/design/design-decisions.md).

Open a Discussion before opening a PR; the conversation will save everyone time.

## What gets pushed back

- Changes that move framework correctness onto the LLM's behavior. (`CLAUDE.md` § Engineering principles.)
- Speculative interfaces or "we might need this later" config knobs (YAGNI).
- Auto-cascading transitions — the framework is visibility-not-automation by design.

---

## Code of conduct

Be technically honest, professionally kind, and willing to be wrong. The design has strong opinions; people don't have to.

Issues, Discussions, and PRs are public artifacts that future contributors will read. Keep them readable.

---

## License

By contributing, you agree your contributions are licensed under the [Apache License, Version 2.0](LICENSE) — the same license as the project.
