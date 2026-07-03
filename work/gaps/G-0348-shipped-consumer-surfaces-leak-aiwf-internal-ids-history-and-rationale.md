---
id: G-0348
title: Shipped consumer surfaces leak aiwf-internal ids, history, and rationale
status: open
---
## What's missing

Consumer-facing surfaces that `aiwf init` / `aiwf update` materialize into a
downstream `.claude/` — verb skills, ritual skills, role-agent cards, entity
templates, the always-on guidance fragment, and the statusline script — cite
aiwf's **own internal entity ids** and carry aiwf's **development history and
rationale**. Both are meaningless (and rot) in a consumer tree. A downstream AI
was told epic activation is "human-sovereign per `M-0095`" — an id that names
nothing in the consumer's repo.

The `skill-body-id` check (`internal/check/skill_body_id.go`, from `G-0299`) is
meant to be the chokepoint for the id half, but it scans a narrow slice and the
leaks route around it. Four blind spots:

1. **Frontmatter is skipped.** It splits YAML off and scans only the body, so
   the consumer-visible `description:` field is unchecked — `aiwfx-start-epic`'s
   description cites `ADR-0023` and an `E-03` example.
2. **`SKILL.md`-only.** The walk skips entity **templates** (`epic-spec.md` has
   `# e.g. [E-0002]`) and **role-agent cards** (`builder.md`, `planner.md`, …),
   which also materialize into `.claude/`.
3. **Two dirs only** (`internal/skills/embedded`, `.../embedded-rituals`). It
   never scans `embedded-guidance/` (the always-on fragment) or
   `embedded-statusline/`.
4. Code spans and link destinations are masked — correct, and **kept**: code
   examples stay exempt (a runnable `aiwf show G-NNNN`-style example and an
   ADR doc-link are legitimate).

The largest single offender is the statusline (`embedded-statusline/
statusline.sh`), which the check never sees: its comments cite `G-0304` (×5),
`G-0188`, `G-0189` (×2), `G-0303`, `G-0310`, `ADR-0026`, and read as aiwf's
changelog — e.g. `# ... (G-0304, superseding the earlier in-flight-list
behavior)`.

Beyond ids, the same surfaces carry **history and rationale** a consumer should
not receive: "the v1 separate tracking doc is gone" asides
(`aiwfx-start-milestone`, `aiwfx-wrap-milestone`, `templates/milestone-spec.md`),
"Why date and decided_by are in the body ..." argumentation blocks
(`templates/adr.md`, `templates/decision.md`), and dense *because*-rationale in
`aiwfx-start-epic`. Downstream instructions should be imperative, short, and
free of historical references, rationale, argumentation, and war-stories.

## Why it matters

A shipped artifact that cites `M-0095`, or narrates why a v1 convention was
retired, is the silent-correctness class aiwf exists to close — pointed the
wrong way. The id references rot as entities change status / archive / rewidth;
the history is noise a consumer cannot act on; the rationale invites a
downstream reader to re-litigate a decision that is not theirs. It also
contradicts the framework's own consumer-vs-development guidance split:
development provenance belongs in this repo's `CLAUDE.md`, the design docs, and
commit trailers — never in a materialized consumer artifact. The existing
chokepoint gives false confidence: it looks like the leak is policed, but four
common id-leak paths (and all of the history/rationale axis) are unchecked.

## Direction

One change, three coordinated parts:

- **Extend `skill-body-id` to close the four holes.** Scan the `description:`
  frontmatter field; scan every materialized `*.md` under the ritual tree
  (templates and agent cards, not just `SKILL.md`); add `embedded-guidance/`;
  and add a raw comment-scoped scan of `embedded-statusline/*.sh` (the markdown
  `proseMask` does not apply to shell). **Keep code spans and link destinations
  exempt.** Extending the check forces cleaning every existing id leak in the
  same change, or it fails its own pre-push.
- **Clean the existing leaks and strip the history/rationale axis.** Rewrite the
  statusline comments, the `aiwfx-start-epic` description, the `E-0002` template
  example, the "v1 is gone" asides, and the rationale blocks into imperative,
  consumer-scoped prose with no aiwf-internal ids, provenance tags, history, or
  argumentation. This half is largely a manual prose rewrite — no regex judges
  "clear and short".
- **Add an authoring principle to this repo's `CLAUDE.md`** (the
  consumer-vs-development guidance section): shipped skill / template / agent /
  guidance / statusline prose is imperative and consumer-scoped — no
  aiwf-internal ids, no provenance tags, no history, no rationale or
  war-stories. This is the human-review backstop for the part the extended check
  cannot mechanize.

Scope note: status-value vocabulary (`superseded`, `deprecated`, `retired`,
`rejected`) in FSM tables and enums is correct domain language, not history —
leave it. Each edited `SKILL.md` under `embedded-rituals/**` needs a referencing
structural test (`skill-edit-structural-test-backstop`); the check extension
carries firing-fixture tests over each newly-scanned surface (frontmatter,
template, agent card, guidance, statusline comment).
