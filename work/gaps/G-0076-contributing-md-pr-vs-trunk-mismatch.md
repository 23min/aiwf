---
id: G-0076
title: CONTRIBUTING.md describes PR-based workflow at odds with trunk-based model on main
status: open
---

`CONTRIBUTING.md` describes a PR-based contribution flow (Issues + Discussions + PRs + templates + a `pr-conventions` CI check). The supporting machinery sits in `.github/ISSUE_TEMPLATE/`, `.github/pull_request_template.md`, and `.github/workflows/pr-conventions.yml` — all carried into the trunk by the pre-promotion merge.

`CLAUDE.md` (per the prep-PoC framing edits in `987273d`) states: *"Trunk-based development on `main`: commit directly, no PR ceremony."*

The two stances contradict. The gap is the open question:

**Should the kernel repo run a PR-based flow, or commit-and-push directly to `main`?**

The choice is binary. Picking either eliminates the contradiction; trying to keep both ("PR for outsiders, direct for maintainers") leaves new contributors guessing which set of rules is operative when, and the `pr-conventions` CI check still gates everyone.

Once decided:

- **PR-based flow:** drop the trunk-based statement from `CLAUDE.md`; route maintainer changes through PRs too; keep `.github/` machinery as-is.
- **Local commit and push:** rewrite `CONTRIBUTING.md` to match; remove or repurpose `.github/ISSUE_TEMPLATE/`, `pull_request_template.md`, `pr-conventions.yml`; tighten `CLAUDE.md`'s wording so the rule is explicit and unmissable.

`CONTRIBUTING.md` body prose (e.g. *"The architecture is settled; the implementation is being built in the open"*) also pre-dates the trunk promotion and reads oddly post-merge — a rewrite or substantial trim is part of the fix either way.

Surfaced during Step 6.B doc-lint pass of the trunk promotion.
