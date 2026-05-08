# Contributing to ai-workflow

This project is in **preview**. The architecture is settled; the implementation is being built in the open. The ways to engage are described below.

If you're new here, please read [`docs/archive/architecture.md`](docs/archive/architecture.md) first. Most "should we add X?" questions are answered there.

---

## Engagement model

The project uses both Issues and Discussions, with a clear split.

### Discussions

Use **[Discussions](https://github.com/23min/ai-workflow-v2/discussions)** for:

- "Should we add X?" — feature ideas, scope debates.
- "How does Y work?" — usage Q&A.
- "What about Z?" — open-ended architecture commentary.
- RFC-style design conversations before any work is committed.

Discussions are the right place to start when you're not sure whether something is a bug, a feature, or a misunderstanding. They have no close conditions; they end when the conversation does.

### Issues

Use **[Issues](https://github.com/23min/ai-workflow-v2/issues)** for **trackable work** with a clear close condition. Two templates ship:

- **Bug** — file when shipped behavior diverges from documented behavior. Requires version, reproduction, expected vs. actual.
- **Design question** — file when the architecture doc is unclear or seems internally inconsistent. Reference the section in question.

There is no "feature request" template by design. Feature ideas start as Discussions; once a discussion converges into "yes, let's do this," it graduates to an Issue (often by the maintainer or whoever's about to do the work).

Blank issues are disabled. Pick a template or open a Discussion.

### Pull requests

Pull requests are gated by **prior conversation**. A PR description references the Issue or Discussion that established the work is wanted. Drive-by PRs without prior context will be asked to start one.

This isn't gatekeeping — it's signal. The architecture doc has strong opinions; PRs that don't engage with them tend to bounce. Fifteen minutes in a Discussion saves a day of revisions.

---

## Pull request expectations

Once a PR is open:

- **Reference the prior conversation** in the description (Issue # or Discussion link).
- **Keep changes small.** One logical change per PR. Refactoring + a feature in the same PR doubles the review cost.
- **Update `CHANGELOG.md`** under `[Unreleased]` for any user-visible change. Lead with the user-observable effect, not the diff.
- **Run the pre-PR audit** described in [`CLAUDE.md`](CLAUDE.md). Walk the diff against the rules; report conformance in the description.
- **Tests stay green.** Race-detector clean. 100% coverage on internal Go packages (per [`CLAUDE.md` § Go conventions](CLAUDE.md#go-conventions)).

---

## Commit message style

Conventional Commits, in the form:

```text
<type>(<scope>): <short subject>

<body — what changed and why, not how>
```

Types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`. Scopes follow the directory/module touched (e.g., `feat(eventlog)`, `fix(verify)`, `docs(architecture)`).

Sign-offs are not required. The Apache-2.0 inbound license terms apply to all contributions; by opening a PR you agree your contribution is licensed under [LICENSE](LICENSE).

---

## What gets accepted readily

- Bug fixes for shipped behavior, with a test that reproduces the bug.
- Documentation clarifications grounded in a question someone actually asked.
- Adapter additions (a new host's skill format, e.g., a different IDE) that don't change the framework's contracts.
- New modules that fill a clear gap and follow the module-structure conventions in [`docs/archive/architecture.md`](docs/archive/architecture.md) §10.

## What needs more discussion first

- Changes to the boundary contract format.
- Changes to the event envelope schema.
- New verbs that don't fit cleanly into the verb set.
- Anything that affects the LLM/engine boundary.

These are framework-shaping. Open a Discussion before opening a PR; the conversation will save everyone time.

## What gets pushed back

- Changes that move structural truth toward the AI assistant. (Architecture §11, item 1.)
- Changes that have the engine generating prose. (Architecture §11, item 4.)
- Auto-cascading transitions ("when X, automatically do Y to all dependents"). The framework is visibility-not-automation by design.
- Plugin / third-party-extension scaffolding without a current consumer.

---

## Code of conduct

Be technically honest, professionally kind, and willing to be wrong. The architecture has strong opinions; people don't have to.

Issues, Discussions, and PRs are public artifacts that future contributors will read. Keep them readable.

---

## License

By contributing, you agree your contributions are licensed under the [Apache License, Version 2.0](LICENSE) — the same license as the project. No CLA, no separate paperwork.
