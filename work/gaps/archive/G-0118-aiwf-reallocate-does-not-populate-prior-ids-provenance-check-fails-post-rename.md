---
id: G-0118
title: aiwf reallocate does not populate prior_ids; provenance check fails post-rename
status: addressed
discovered_in: E-0029
addressed_by_commit:
    - 6e1a0c0a803e4c9aa66dcc45590c467b4a8b05f6
---
## What's missing

`aiwf reallocate` and the kernel's `provenance-authorization-out-of-scope` rule don't compose when the renumbered entity carried historical commits made under an authorized scope. Concrete state where this surfaces: an entity is allocated on a feature branch, work proceeds against it under an `aiwf authorize` scope that records `aiwf-authorized-by: <SHA>` on each commit, an id-collision is detected later (typically at merge time, because a parallel session allocated the same id on trunk), `aiwf reallocate` renumbers the local entity (file slug, frontmatter `id:`, forward references in entity bodies — all updated in one commit). **But commit trailers are immutable.** Every historical commit on the feature branch still names the *old* id in its `aiwf-entity:` trailer. Post-reallocate the old id now resolves to the *other* entity (the parallel allocation that took it), which doesn't reach the original scope-entity, so `provenance-authorization-out-of-scope` fires on push against every commit that ran under the scope.

The renumbered entity's frontmatter has a `prior_ids:` field — but `aiwf reallocate` doesn't populate it, and the reachability walk in the provenance audit doesn't consult it. So the rename history isn't recoverable from kernel state and the only way to push the branch is `--no-verify` (bypassing the chokepoint that the framework relies on for correctness) or rewriting git history (operationally expensive, breaks `aiwf history`).

Reproduction (from E-0029 wrap):

1. Session A allocates M-0102 on epic branch `epic/E-0029-glanceable-render` (under `aiwf authorize E-0029 --to ai/claude`).
2. Session B allocates M-0102 on trunk (under a different epic, E-0030) — both sessions pass the allocator's "scan trunk for collisions" because A's M-0102 never reached trunk yet.
3. A wraps E-0029, attempts merge into trunk: `aiwf check` fires `ids-unique/trunk-collision`.
4. A runs `aiwf reallocate work/epics/.../M-0102-...md` → renumbers to M-0107 in one commit.
5. A retries the merge: succeeds (file-level collision resolved).
6. A runs `git push origin main`: pre-push hook fires `provenance-authorization-out-of-scope` 9 times — once per historical commit on the epic branch whose `aiwf-entity` trailer named M-0102 under the open scope.

## Why it matters

The kernel's design pillars name `aiwf check` as the chokepoint that makes the framework's guarantees real (rather than depending on LLM behavior). When the documented path to resolve a collision (`aiwf reallocate`) leaves the chokepoint in a state where push cannot proceed cleanly, operators are pushed to `--no-verify` — bypassing the very guarantee the kernel commits to. The cycle is self-defeating: the rule catches a real problem, the documented remediation can't fully resolve it, so the operator's only options are (a) bypass the rule, (b) rewrite history (which breaks the `aiwf history` trail the kernel also commits to), or (c) leave the work unpushed. None of those are clean. And this isn't a rare race — it's the expected outcome when two sessions work in parallel on independent feature branches under authorized scopes and one of them lands first.

The fix is mechanical and bounded: populate `prior_ids:` (the field already exists in the renumbered entity's frontmatter) in `aiwf reallocate`'s commit, and have the `provenance-authorization-out-of-scope` reachability walk follow `prior_ids` chains. The reachability walk on `aiwf-entity: M-0102` would resolve to the renumbered M-0107 via `prior_ids: [M-0102]`, and M-0107 → E-0029 reaches the scope-entity. Both sides of the composition are then consistent: the reallocate verb maintains the kernel state the provenance audit relies on.

A complementary improvement: `aiwf reallocate` could warn when the renumbered entity has commits under an active authorize scope, so the operator sees the trailer-rot fallout at reallocate time rather than at push time. That's a UX surface, not a correctness fix.

Discovered during E-0029 wrap when a cross-session race put M-0102 on both the epic branch (Repair Playwright e2e suite) and trunk (E-0030's authorize --branch flag work). Wrap landed locally but push blocked; tracked here so the fix lands in a kernel-side milestone or epic.
