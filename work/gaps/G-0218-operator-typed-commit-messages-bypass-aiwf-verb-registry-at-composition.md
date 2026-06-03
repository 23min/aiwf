---
id: G-0218
title: Operator-typed commit messages bypass aiwf-verb registry at composition
status: open
discovered_in: M-0160
---

## What's missing

Three gating layers, all of which let a fabricated `aiwf-verb:` trailer pass into committed history:

1. **Composition-time** — the kernel's verbs emit aiwf-verb trailers themselves with closed-set values; the operator never types these in normal use. But for plain git operations (`git commit`, `git merge`, `git rebase`, etc.) the operator composes the commit message by hand, and **nothing in the kernel watches the in-progress message**. Any string can be typed in the `aiwf-verb:` slot.

2. **Pre-commit hook** runs `aiwf check --shape-only`. The shape mode does NOT include the `trailer-verb-unknown` rule, which walks commit history (full-check territory). So a fabricated trailer in a freshly-composed commit passes pre-commit silently.

3. **Pre-push hook** DOES run the full `aiwf check` including `trailer-verb-unknown`. But the rule fires at warning severity (per G-0150's design — "introduce as warning so the rule can introduce without retroactive breakage of historical fabricated trailers"). Push proceeds; the rule informs but does not block.

Net effect: an operator can write a fabricated trailer, commit it, push it, and the kernel's only response is a post-hoc advisory. The chokepoint is **descriptive, not prescriptive**.

## Why it matters

Per CLAUDE.md "Framework correctness must not depend on the LLM's behavior" — the kernel's role is to refuse bad shapes mechanically, not to inform after the fact. The `aiwf-verb:` namespace is a closed set sourced from the running binary's Cobra command tree. Any value outside that set IS by construction a fabrication (operator typo, LLM hallucination, or category confusion such as treating `git merge` as if it were an aiwf verb). The kernel knows the closed set at composition time; failing to refuse at composition time is a kernel-discipline gap.

Concrete instance surfaced during M-0160 epic-merge prep: an operator (me, in this case) drafted `git merge --no-ff` commit messages for both M-0159 (yesterday, commit `e1dc6dc6`) and M-0160 (today, commit `734dca4b`) that carried `aiwf-verb: merge`. `merge` is a git concept, not an aiwf verb — there is no `aiwf merge` command in the Cobra tree. The trailer was a category confusion: the operator was expressing the kernel-relevant intent of the merge ("this commit transitions M-0NNN's work into the epic") in the trailer namespace, which is reserved for verb-emitted structural shape.

The error happened twice — the M-0160 merge repeated the M-0159 mistake. **Repeated operator error is the canonical signal that a chokepoint is missing**, per CLAUDE.md's standard hint ("if you see this happen more than once").

Worse: the rule's post-hoc detection at push time doesn't help the operator catch the error before it lands. Once committed and pushed, the history carries the fabricated trailer permanently; the only remediation paths are `aiwf acknowledge-illegal <sha>` (silences the rule without rewriting history) or history rewrite (blocked by the trunk-aware push model).

## Proposed fix shape

**Primary: a `commit-msg` git hook** managed by aiwf, materialized into `.git/hooks/commit-msg` with the same `# aiwf:commit-msg` marker pattern as the existing pre-commit and pre-push hooks. The hook receives the in-progress commit-message file path as `$1`; parses any `aiwf-verb:` trailer values from it; refuses (exits non-zero) when any value is outside the registered Cobra command tree AND not in the ritualVerbs allowlist (the same closed sets `trailer-verb-unknown` reads). Composition gate: the operator's commit attempt is refused with a clear message naming the offending value and the closed set.

Composition-time check is fast: regex-match the in-progress message against the closed set; no history walk required. Same closed-set sources as the existing rule, just consulted earlier.

**Secondary: rule-side severity tightening.** With the commit-msg hook in place, the `trailer-verb-unknown` rule's post-hoc warning becomes structurally redundant for new commits (they can't land with bad trailers). Promoting the rule from warning to error severity for new commits (perhaps via a `--since` window or a "post-hook-landed-at" marker) closes the loop. Historical fabricated trailers remain at warning to avoid retroactive breakage; the rule fires error-severity only on commits authored after the hook landed.

**Tertiary: discipline pin** in `aiwfx-wrap-milestone` SKILL.md (and CLAUDE.md in this repo) — "plain git operations DO NOT get aiwf-* trailers." The discipline is documented for human + LLM operators until the structural fix lands; after the fix, the documentation can simplify to "the kernel will refuse." Operators reaching for `aiwf-verb: <something>` in a hand-typed commit message should stop — that's a fabrication.

## Test surface

Once the commit-msg hook lands:

- Refusal test: a synthetic commit message carrying `aiwf-verb: <unregistered>` (e.g. `aiwf-verb: merge`, `aiwf-verb: implement`, `aiwf-verb: fakeverb`) feeds to the hook → exit non-zero with informative stderr.
- Pass-through test: a commit message carrying `aiwf-verb: promote` (a registered verb) feeds to the hook → exit zero.
- Pass-through test: a commit message carrying NO `aiwf-verb:` trailer feeds to the hook → exit zero (operator chose plain conventional-commits form, that's fine).
- Pass-through test: a commit message carrying a recognized ritual-verb (per the ritualVerbs allowlist at `internal/check/trailer_verb_unknown.go`) feeds to the hook → exit zero.
- Hook materialization test: `aiwf init` / `aiwf update` write `.git/hooks/commit-msg` with the `# aiwf:commit-msg` marker; the marker-based pattern preserves user-written hooks.
- Sabotage-verifiable: revert the hook's value-check → fabricated trailers pass; the discriminating test fires.

## Workaround

Until the structural fix lands, the discipline is operator awareness:

- **Plain git operations DO NOT get aiwf-* trailers.** No `aiwf-verb:`, no `aiwf-entity:`, no `aiwf-actor:` in hand-typed `git commit`, `git merge`, `git rebase` messages. The git committer identity already records who did the operation; the merge commit's tree already records what got merged; nothing the aiwf trailer namespace adds is missing.
- **The aiwf-verb trailer is reserved for aiwf verb invocations.** If your fingers reach for `aiwf-verb:` outside of an `aiwf <verb>` invocation, stop — it's a fabrication.
- Pre-push surfaces the fabrication post-hoc via `trailer-verb-unknown`; treat the warning as a real signal to amend before next push (where possible) or acknowledge-illegal (where amend is blocked by the trunk-aware push model).

The discipline is documented in `aiwfx-wrap-milestone` SKILL.md and CLAUDE.md until the commit-msg hook lands.

## Closing this gap

When the impl lands:
- `commit-msg` git hook in `internal/initrepo/` (or wherever pre-push hook materialization lives), marker-managed.
- Hook check sources from the same closed sets as `trailer-verb-unknown` (Cobra tree + ritualVerbs allowlist).
- Tests above land alongside the implementation.
- `aiwfx-wrap-milestone` skill text simplified once the discipline becomes mechanical (the explicit "no aiwf-* trailers on plain git" note can drop to "the kernel will refuse"; the broader trailer-ontology note stays as background).
- CLAUDE.md note simplified or removed similarly.
- Optional: rule severity tightening per the secondary fix above.
- Promote G-0218 to `addressed` with `--by M-NNNN`.

## Discovered in

M-0160 — observed at epic-merge push time when the pre-push `aiwf check` flagged two `trailer-verb-unknown` warnings: the M-0159 merge commit (`e1dc6dc6`, authored yesterday at M-0159 wrap-merge) and the M-0160 merge commit (`734dca4b`, authored today at M-0160 wrap-merge) both carried `aiwf-verb: merge`. The operator made the same category-confusion error twice. The kernel's post-hoc detection caught the second instance but couldn't have prevented the first — and didn't block the push of either.
