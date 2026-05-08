---
id: G-076
title: CONTRIBUTING.md describes PR-based workflow at odds with trunk-based model on main
status: open
---
`CONTRIBUTING.md` describes a formal contribution flow (Issues + Discussions + PRs + templates + a `pr-conventions` CI check) that was added on `main` pre-promotion to support a PR-based workflow. The supporting machinery is in `.github/ISSUE_TEMPLATE/`, `.github/pull_request_template.md`, and `.github/workflows/pr-conventions.yml` — all carried into the trunk by the merge.

PoC's `CLAUDE.md` (per the prep-PoC framing edits in commit `987273d`) states: *"Trunk-based development on `main`: commit directly, no PR ceremony."* The two stances need reconciliation.

Three options:

- **Drop PR ceremony** — rewrite `CONTRIBUTING.md` to match trunk-based reality; remove or repurpose the `.github/` machinery.
- **Keep for external contributions only** — clarify in `CONTRIBUTING.md` that maintainers commit directly, while contributors from outside the project use the PR flow; tighten CLAUDE.md to match.
- **Adopt PR ceremony formally** — update `CLAUDE.md` to drop the trunk-based statement; route maintainer changes through PRs too.

`CONTRIBUTING.md` body prose (e.g. *"The architecture is settled; the implementation is being built in the open"*) also pre-dates the trunk promotion and reads oddly post-merge — a rewrite or substantial trim is part of any of the three options.

Surfaced during Step 6.B doc-lint pass of the trunk promotion.
